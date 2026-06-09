package manager

import (
	"chat/adapter"
	adaptercommon "chat/adapter/common"
	"chat/addition/fetch"
	"chat/addition/web"
	"chat/admin"
	"chat/auth"
	"chat/billing"
	"chat/channel"
	"chat/globals"
	"chat/manager/conversation"
	"chat/manager/memory"
	"chat/utils"
	"context"
	"time"

	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

const defaultMessage = "empty response"
const interruptMessage = "interrupted"
const maxFetchToolRounds = 20

func summarizeToolCallArguments(arguments string) string {
	arguments = strings.TrimSpace(arguments)
	if len(arguments) <= 240 {
		return arguments
	}

	return arguments[:240] + "..."
}

func summarizeToolCalls(calls *globals.ToolCalls) string {
	if calls == nil || len(*calls) == 0 {
		return "[]"
	}

	items := make([]string, 0, len(*calls))
	for _, call := range *calls {
		items = append(items, fmt.Sprintf(
			"{id:%s name:%s args:%s}",
			call.Id,
			call.Function.Name,
			summarizeToolCallArguments(call.Function.Arguments),
		))
	}

	return "[" + strings.Join(items, ", ") + "]"
}

func formatBuiltinToolNames(names []string) string {
	if len(names) == 0 {
		return "[]"
	}

	return "[" + strings.Join(names, ",") + "]"
}

func buildToolCallEvent(call globals.ToolCall, status string) *globals.ChatSegmentToolCall {
	name := strings.TrimSpace(call.Function.Name)
	if name == "" {
		return nil
	}

	return &globals.ChatSegmentToolCall{
		Id:        strings.TrimSpace(call.Id),
		Name:      name,
		Arguments: strings.TrimSpace(call.Function.Arguments),
		Status:    status,
	}
}

func buildToolResultEvent(call globals.ToolCall, toolMessage globals.Message) *globals.ChatSegmentToolCall {
	event := buildToolCallEvent(call, "success")
	if event == nil {
		return nil
	}

	raw := strings.TrimSpace(toolMessage.Content)
	event.Result = raw

	var result memory.ToolResult
	if err := json.Unmarshal([]byte(raw), &result); err == nil {
		if strings.TrimSpace(result.Error) != "" || strings.EqualFold(strings.TrimSpace(result.Status), "error") {
			event.Status = "error"
			event.Error = strings.TrimSpace(result.Error)
			event.Result = ""
		}
	}

	return event
}

func buildThinkingConfig(instance *conversation.Conversation, model string) interface{} {
	if instance == nil {
		return nil
	}

	if globals.SupportXiaomiTokenPlanThinkingControl(model) {
		effort := globals.NormalizeXiaomiTokenPlanThinkingEffort(model, instance.GetOpenAIReasoningEffort())
		if effort == "" {
			return nil
		}
		if effort == "none" {
			return map[string]interface{}{"type": "disabled"}
		}

		return map[string]interface{}{"type": "enabled"}
	}

	if !globals.SupportOpenAIResponsesReasoningControl(model) {
		return nil
	}

	effort := globals.NormalizeOpenAIResponsesReasoningEffort(
		model,
		instance.GetOpenAIReasoningEffort(),
		instance.IsEnableWebSearch(),
	)
	if effort == "" {
		return nil
	}

	config := map[string]interface{}{
		"effort": effort,
	}
	if effort != "none" {
		summary := globals.NormalizeOpenAIResponsesReasoningSummary(instance.GetOpenAIReasoningSummary())
		if summary != "none" {
			config["summary"] = summary
		}
	}

	return config
}

func buildDeepseekThinkingConfig(instance *conversation.Conversation, model string) (interface{}, *string) {
	if instance == nil || !globals.IsDeepseekV4Model(model) {
		return nil, nil
	}

	if !instance.IsDeepseekThinkingEnabled() {
		return map[string]interface{}{"type": "disabled"}, nil
	}

	effort := instance.GetDeepseekReasoningEffort()
	return map[string]interface{}{"type": "enabled"}, &effort
}

func sendToolCallEvents(conn *Connection, calls *globals.ToolCalls, status string, quota float32, plan bool) error {
	if calls == nil || len(*calls) == 0 {
		return nil
	}

	for _, call := range *calls {
		event := buildToolCallEvent(call, status)
		if event == nil {
			continue
		}

		conn.TrySendClient(globals.ChatSegmentResponse{
			Quota:    quota,
			ToolCall: event,
			End:      false,
			Plan:     plan,
		})
	}

	return nil
}

func sendToolResultEvents(conn *Connection, calls *globals.ToolCalls, toolMessages []globals.Message, quota float32, plan bool) error {
	if calls == nil || len(*calls) == 0 || len(toolMessages) == 0 {
		return nil
	}

	callIndex := make(map[string]globals.ToolCall, len(*calls))
	for _, call := range *calls {
		callID := strings.TrimSpace(call.Id)
		if callID == "" {
			continue
		}
		callIndex[callID] = call
	}

	for _, toolMessage := range toolMessages {
		callID := strings.TrimSpace(utils.ToString(toolMessage.ToolCallId))
		if callID == "" {
			continue
		}

		call, ok := callIndex[callID]
		if !ok {
			continue
		}

		event := buildToolResultEvent(call, toolMessage)
		if event == nil {
			continue
		}

		conn.TrySendClient(globals.ChatSegmentResponse{
			Quota:    quota,
			ToolCall: event,
			End:      false,
			Plan:     plan,
		})
	}

	return nil
}

func CollectQuota(c *gin.Context, user *auth.User, buffer *utils.Buffer, uncountable bool, err error) {
	db := utils.GetDBFromContext(c)
	quota := buffer.GetQuota()

	if user == nil || quota <= 0 {
		return
	}

	if buffer.IsEmpty() || err != nil {
		return
	}

	if uncountable {
		if !auth.FinalizeSubscriptionUsage(db, utils.GetCacheFromContext(c), user, buffer.GetModel(), quota) {
			globals.Warn(fmt.Sprintf("failed to finalize subscription usage (model: %s, quota: %.4f)", buffer.GetModel(), quota))
		}
		return
	}

	collectUserQuota(db, user, quota)
}

func collectUserQuota(db *sql.DB, user *auth.User, quota float32) {
	if !user.UseQuota(db, quota) {
		user.ForceUseQuota(db, quota)
	}
}

type partialChunk struct {
	Chunk *globals.Chunk
	End   bool
	Hit   bool
	Error error
}

type removeSignalHandler func(index int)

func createStopSignal(conn *Connection, onRemove removeSignalHandler) (<-chan bool, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	stopSignal := make(chan bool, 1)
	go func(conn *Connection, stopSignal chan bool) {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer func() {
			ticker.Stop()
			if r := recover(); r != nil {
				stack := debug.Stack()
				globals.Warn(fmt.Sprintf("caught panic from stop signal: %s\n%s", r, stack))
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				state := false
				for {
					form := conn.PeekStop()
					if form == nil {
						break
					}
					if form.Type == StopType {
						state = true
						break
					}
					if form.Type == RemoveType && onRemove != nil {
						// Remove can race with restart; persist it without interrupting the new response.
						id, err := getId(form.Message)
						if err != nil {
							globals.Warn(fmt.Sprintf("failed to parse remove event while polling chat controls: %s", err.Error()))
							continue
						}
						onRemove(id)
					}
				}

				if state {
					select {
					case <-stopSignal:
					default:
					}
					select {
					case stopSignal <- true:
					case <-ctx.Done():
					}
					return
				}

				select {
				case stopSignal <- false:
				default:
				}
			}
		}
	}(conn, stopSignal)

	return stopSignal, cancel
}

func createRemoveSignalHandler(db *sql.DB, instance *conversation.Conversation) removeSignalHandler {
	return func(index int) {
		if instance == nil || db == nil {
			return
		}
		removed := instance.RemoveMessage(index)
		if removed.Role == "" {
			return
		}
		instance.SaveConversation(db)
	}
}

func cloneChatPropsWithBuffer(props *adaptercommon.ChatProps, buffer *utils.Buffer) *adaptercommon.ChatProps {
	if props == nil {
		return nil
	}

	cloned := *props
	cloned.Buffer = buffer
	return &cloned
}

func newRequestBufferFromCaptureBuffer(props *adaptercommon.ChatProps, captureBuffer *utils.Buffer) *utils.Buffer {
	if props == nil || captureBuffer == nil {
		return captureBuffer
	}

	requestBuffer := utils.NewBuffer(props.Model, props.Message, captureBuffer.GetCharge())
	requestBuffer.SetPrompts(props)
	return requestBuffer
}

func syncBufferChannel(target *utils.Buffer, source *utils.Buffer) {
	if target == nil || source == nil || target == source {
		return
	}

	target.SetChannel(source.GetChannelId(), source.GetChannelName())
}

func signalInterrupt(interruptSignal chan error, err error) {
	select {
	case interruptSignal <- err:
	default:
	}
}

func waitForRoundTaskEnd(chunkChan <-chan partialChunk) {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	for {
		select {
		case data := <-chunkChan:
			if data.End {
				return
			}
		case <-timer.C:
			return
		}
	}
}

func recordBuiltinToolRequest(buffer *utils.Buffer, instance *conversation.Conversation, model string) {
	if buffer == nil || instance == nil {
		return
	}

	requestedTools := make([]string, 0, 3)
	if instance.IsEnableWebSearch() {
		requestedTools = append(requestedTools, "google_search")
	}
	if instance.IsEnableURLContext() {
		requestedTools = append(requestedTools, "url_context")
	}

	codeExecutionEnabled := instance.IsEnableCodeExecution()
	codeExecutionSupported := globals.SupportGeminiCodeExecution(model)
	if codeExecutionEnabled && codeExecutionSupported {
		requestedTools = append(requestedTools, "code_execution")
	}

	globals.Debug(fmt.Sprintf(
		"[builtin-tools] request conversation_id=%d model=%s tools_requested=%s web_search_enabled=%v url_context_enabled=%v code_execution_enabled=%v code_execution_supported=%v",
		instance.GetId(),
		model,
		formatBuiltinToolNames(requestedTools),
		instance.IsEnableWebSearch(),
		instance.IsEnableURLContext(),
		codeExecutionEnabled,
		codeExecutionSupported,
	))

	if codeExecutionEnabled {
		buffer.SetCodeExecutionToolUsage(codeExecutionEnabled, codeExecutionSupported)
	}
}

func buildChatProps(
	conn *Connection,
	instance *conversation.Conversation,
	model string,
	segment []globals.Message,
	buffer *utils.Buffer,
	memoryPrompt string,
	recentChatsPrompt string,
	tools *globals.FunctionTools,
	toolChoice *interface{},
	disableCache bool,
) *adaptercommon.ChatProps {
	thinking := buildThinkingConfig(instance, model)
	var reasoningEffort *string
	if thinking == nil {
		thinking, reasoningEffort = buildDeepseekThinkingConfig(instance, model)
	}

	return adaptercommon.CreateChatProps(&adaptercommon.ChatProps{
		Model:                model,
		OriginalModel:        model,
		Message:              segment,
		CustomInstruction:    instance.GetCustomInstruction(),
		LearningMode:         instance.IsLearningModeEnabled(),
		MemoryPrompt:         memoryPrompt,
		RecentChatsPrompt:    recentChatsPrompt,
		MemoryEnabled:        instance.IsMemoryEnabled(),
		MemoryHistoryEnabled: instance.IsMemoryHistoryEnabled(),
		Tools:                tools,
		ToolChoice:           toolChoice,
		EnableWeb:            instance.IsEnableWeb(),
		EnableWebSearch:      instance.IsEnableWebSearch(),
		EnableURLContext:     instance.IsEnableURLContext(),
		EnableCodeExecution:  instance.IsEnableCodeExecution() && globals.SupportGeminiCodeExecution(model),
		EnableXSearch:        instance.IsEnableXSearch(),
		GeminiThinkingBudget: instance.GetGeminiThinkingBudget(),
		Thinking:             thinking,
		ReasoningEffort:      reasoningEffort,
		MaxTokens:            instance.GetMaxTokens(),
		Temperature:          instance.GetTemperature(),
		TopP:                 instance.GetTopP(),
		TopK:                 instance.GetTopK(),
		PresencePenalty:      instance.GetPresencePenalty(),
		FrequencyPenalty:     instance.GetFrequencyPenalty(),
		RepetitionPenalty:    instance.GetRepetitionPenalty(),
		ClientContext:        extractClientContext(conn.GetCtx()),
		DisableCache:         disableCache,
	}, buffer)
}

func createRoundTask(
	conn *Connection,
	user *auth.User,
	captureBuffer *utils.Buffer,
	streamBuffer *utils.Buffer,
	db *sql.DB,
	cache *redis.Client,
	group string,
	props *adaptercommon.ChatProps,
	plan bool,
	onRemove removeSignalHandler,
) (hit bool, err error, interrupted bool) {
	chunkChan := make(chan partialChunk, 24)
	interruptSignal := make(chan error, 1)
	stopSignal, stopPolling := createStopSignal(conn, onRemove)

	defer func() {
		stopPolling()
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil && !strings.Contains(fmt.Sprintf("%s", r), "closed channel") {
				stack := debug.Stack()
				globals.Warn(fmt.Sprintf("caught panic from round chat request: %s\n%s", r, stack))
			}
		}()

		if props.DisableCache {
			err = channel.NewChatRequest(group, props, func(data *globals.Chunk) error {
				if len(interruptSignal) > 0 {
					return errors.New(interruptMessage)
				}

				chunkChan <- partialChunk{Chunk: data, End: false, Hit: false, Error: nil}
				return nil
			})
		} else {
			requestBuffer := newRequestBufferFromCaptureBuffer(props, captureBuffer)
			requestProps := cloneChatPropsWithBuffer(props, requestBuffer)
			hit, err = channel.NewChatRequestWithCache(cache, requestBuffer, group, requestProps, func(data *globals.Chunk) error {
				if len(interruptSignal) > 0 {
					return errors.New(interruptMessage)
				}

				if requestBuffer != nil && requestBuffer != captureBuffer && data != nil {
					requestBuffer.WriteChunk(data)
				}

				chunkChan <- partialChunk{Chunk: data, End: false, Hit: false, Error: nil}
				return nil
			})
			syncBufferChannel(captureBuffer, requestBuffer)
		}

		chunkChan <- partialChunk{Chunk: nil, End: true, Hit: hit, Error: err}
	}()

	for {
		select {
		case data := <-chunkChan:
			if data.Error != nil && data.Error.Error() == interruptMessage {
				interrupted = true
				if data.End {
					return hit, nil, true
				}
				continue
			}

			hit = data.Hit
			err = data.Error

			if data.End {
				return
			}

			chunk := data.Chunk
			if captureBuffer != nil && chunk != nil {
				captureBuffer.WriteChunk(chunk)
			}

			if streamBuffer != nil {
				content := ""
				if chunk != nil {
					content = chunk.Content
					if captureBuffer == streamBuffer {
						content = streamBuffer.GetChunk()
					} else if content != "" {
						streamBuffer.Write(content)
					}
				}

				if !conn.TrySendClient(globals.ChatSegmentResponse{
					Message: content,
					Quota:   streamBuffer.GetQuota(),
					End:     false,
					Plan:    plan,
				}) {
					continue
				}
			}
		case signal := <-stopSignal:
			if signal {
				quota := float32(0)
				if streamBuffer != nil {
					quota = streamBuffer.GetQuota()
				} else if captureBuffer != nil {
					quota = captureBuffer.GetQuota()
				}

				globals.Info(fmt.Sprintf("client stopped the chat request (model: %s, client: %s)", props.Model, conn.GetCtx().ClientIP()))
				_ = conn.SendClient(globals.ChatSegmentResponse{
					Quota: quota,
					End:   true,
					Plan:  plan,
				})
				signalInterrupt(interruptSignal, errors.New("signal"))
				waitForRoundTaskEnd(chunkChan)
				return hit, nil, true
			}
		}
	}
}

type memoryContext struct {
	MemoryPrompt      string
	RecentChatsPrompt string
	Writable          bool
}

func appendFunctionTools(target globals.FunctionTools, source *globals.FunctionTools) globals.FunctionTools {
	if source == nil {
		return target
	}

	return append(target, (*source)...)
}

func buildAvailableToolDefinitions(fetchEnabled bool, memoryWritable bool) *globals.FunctionTools {
	tools := make(globals.FunctionTools, 0)
	if memoryWritable {
		tools = appendFunctionTools(tools, memory.BuildToolDefinition())
	}
	if fetchEnabled {
		tools = appendFunctionTools(tools, fetch.BuildToolDefinition())
	}

	if len(tools) == 0 {
		return nil
	}
	return &tools
}

func buildAutoToolChoice() *interface{} {
	choice := interface{}("auto")
	return &choice
}

func unsupportedToolResult(call globals.ToolCall) globals.Message {
	return globals.Message{
		Role: globals.Tool,
		Content: utils.Marshal(map[string]string{
			"status": "error",
			"action": call.Function.Name,
			"error":  "unsupported tool",
		}),
		ToolCallId: utils.ToPtr(call.Id),
	}
}

func executeAvailableToolCall(db *sql.DB, user *auth.User, call globals.ToolCall) globals.Message {
	switch call.Function.Name {
	case memory.MemoryToolName:
		messages := memory.ExecuteToolCalls(db, user, &globals.ToolCalls{call})
		if len(messages) > 0 {
			return messages[0]
		}
	case fetch.ToolName:
		return fetch.ExecuteToolCall(call)
	}

	return unsupportedToolResult(call)
}

func executeAvailableToolCalls(db *sql.DB, user *auth.User, calls *globals.ToolCalls) []globals.Message {
	if calls == nil || len(*calls) == 0 {
		return nil
	}

	messages := make([]globals.Message, 0, len(*calls))
	for _, call := range *calls {
		messages = append(messages, executeAvailableToolCall(db, user, call))
	}
	return messages
}

func containsMemoryToolCall(calls *globals.ToolCalls) bool {
	if calls == nil {
		return false
	}

	for _, call := range *calls {
		if call.Function.Name == memory.MemoryToolName {
			return true
		}
	}
	return false
}

func syncToolFinalMetadata(liveBuffer *utils.Buffer, responseBuffer *utils.Buffer) {
	if liveBuffer == nil || responseBuffer == nil {
		return
	}

	liveBuffer.SetGeminiHiddenMetadata(responseBuffer.GetGeminiHiddenMetadata())
	liveBuffer.SetClaudeHiddenMetadata(responseBuffer.GetClaudeHiddenMetadata())
	liveBuffer.MergeUsage(responseBuffer)
	liveBuffer.MergeBuiltinToolUsage(responseBuffer.GetBuiltinToolUsage())
}

func buildToolLimitSystemMessage() globals.Message {
	return globals.Message{
		Role:    globals.System,
		Content: "The tool call limit for this response has been reached. Do not emit any additional tool calls, DSML, XML, JSON, or tool-call markup. Produce the final answer in natural language using the available tool results. If the available information is incomplete, say that clearly.",
	}
}

func summarizeMemoryRecords(memories []memory.Record) string {
	if len(memories) == 0 {
		return "[]"
	}

	limit := len(memories)
	if limit > 5 {
		limit = 5
	}

	items := make([]string, 0, limit)
	for _, item := range memories[:limit] {
		content := strings.TrimSpace(item.Content)
		if len(content) > 36 {
			content = content[:36] + "..."
		}

		items = append(items, fmt.Sprintf(
			"{id:%d category:%s content:%q}",
			item.ID,
			item.Category,
			content,
		))
	}

	summary := "[" + strings.Join(items, ", ") + "]"
	if len(memories) > limit {
		summary += fmt.Sprintf(" (+%d more)", len(memories)-limit)
	}
	return summary
}

func buildMemoryContext(db *sql.DB, user *auth.User, instance *conversation.Conversation, model string, group string) memoryContext {
	ctx := memoryContext{}
	if user == nil {
		return ctx
	}

	userID := user.GetID(db)
	globals.Debug(fmt.Sprintf(
		"[memory] building context user_id=%d conversation_id=%d model=%s group=%s memory_enabled=%v history_enabled=%v",
		userID,
		instance.GetId(),
		model,
		group,
		instance.IsMemoryEnabled(),
		instance.IsMemoryHistoryEnabled(),
	))

	if instance.IsMemoryEnabled() {
		memories, err := memory.ListByUser(db, userID, "", memory.DefaultMemoryLimit)
		if err != nil {
			globals.Warn(fmt.Sprintf("[memory] failed to load memories: %s", err.Error()))
		} else {
			ctx.MemoryPrompt = memory.BuildMemoryPrompt(memories)
			globals.Debug(fmt.Sprintf(
				"[memory] loaded memories user_id=%d count=%d prompt_len=%d sample=%s",
				userID,
				len(memories),
				len(ctx.MemoryPrompt),
				summarizeMemoryRecords(memories),
			))
			ids := make([]int64, 0, len(memories))
			for _, item := range memories {
				ids = append(ids, item.ID)
			}
			if err := memory.Touch(db, userID, ids); err != nil {
				globals.Warn(fmt.Sprintf("[memory] failed to touch memories: %s", err.Error()))
			}
		}

		ctx.Writable = memory.CanUseWritableTools(model, group)
		globals.Debug(fmt.Sprintf(
			"[memory] writable tools state user_id=%d model=%s group=%s writable=%v",
			userID,
			model,
			group,
			ctx.Writable,
		))
	}

	if instance.IsMemoryHistoryEnabled() {
		chats, err := memory.ListRecentConversations(db, userID, instance.GetId(), memory.DefaultRecentChatNum)
		if err != nil {
			globals.Warn(fmt.Sprintf("[memory] failed to load recent chats: %s", err.Error()))
		} else {
			ctx.RecentChatsPrompt = memory.BuildRecentChatsPrompt(chats)
			globals.Debug(fmt.Sprintf(
				"[memory] loaded recent chats user_id=%d count=%d prompt_len=%d",
				userID,
				len(chats),
				len(ctx.RecentChatsPrompt),
			))
		}
	}

	return ctx
}

func createToolChatTask(
	conn *Connection,
	user *auth.User,
	liveBuffer *utils.Buffer,
	db *sql.DB,
	cache *redis.Client,
	model string,
	group string,
	instance *conversation.Conversation,
	segment []globals.Message,
	plan bool,
	ctx memoryContext,
	tools *globals.FunctionTools,
	maxToolRounds int,
) (hit bool, err error, interrupted bool) {
	workingSegment := utils.DeepCopy(segment)
	memoryPrompt := ctx.MemoryPrompt
	recentChatsPrompt := ctx.RecentChatsPrompt
	toolChoice := buildAutoToolChoice()

	for round := 0; round < maxToolRounds; round++ {
		roundBuffer := utils.NewBuffer(model, workingSegment, liveBuffer.GetCharge())
		if round > 0 {
			liveBuffer.InputTokens += roundBuffer.CountInputToken()
			liveBuffer.Quota += utils.CountInputQuota(liveBuffer.GetCharge(), roundBuffer.CountInputToken())
		}

		globals.Debug(fmt.Sprintf(
			"[tools] starting tool round %d model=%s memory_prompt_len=%d recent_chats_prompt_len=%d segment_messages=%d tools=%d",
			round+1,
			model,
			len(memoryPrompt),
			len(recentChatsPrompt),
			len(workingSegment),
			len(*tools),
		))

		props := buildChatProps(
			conn,
			instance,
			model,
			workingSegment,
			roundBuffer,
			memoryPrompt,
			recentChatsPrompt,
			tools,
			toolChoice,
			true,
		)

		streamBuffer := liveBuffer

		hit, err, interrupted = createRoundTask(conn, user, roundBuffer, streamBuffer, db, cache, group, props, plan, createRemoveSignalHandler(db, instance))
		if err != nil || interrupted {
			return hit, err, interrupted
		}

		assistant := extractAssistantMessageFromBuffer(roundBuffer, false, false)
		if assistant.ToolCalls == nil || len(*assistant.ToolCalls) == 0 {
			syncToolFinalMetadata(liveBuffer, roundBuffer)
			return hit, nil, false
		}

		globals.Debug(fmt.Sprintf(
			"[tools] round %d received tool calls for model %s: %s",
			round+1,
			model,
			summarizeToolCalls(assistant.ToolCalls),
		))

		if err := sendToolCallEvents(conn, assistant.ToolCalls, "executing", liveBuffer.GetQuota(), plan); err != nil {
			return hit, err, true
		}

		toolMessages := executeAvailableToolCalls(db, user, assistant.ToolCalls)
		for _, toolMessage := range toolMessages {
			globals.Debug(fmt.Sprintf(
				"[tools] round %d tool result for model %s tool_call_id=%s payload=%s",
				round+1,
				model,
				utils.ToString(toolMessage.ToolCallId),
				summarizeToolCallArguments(toolMessage.Content),
			))
		}
		if err := sendToolResultEvents(conn, assistant.ToolCalls, toolMessages, liveBuffer.GetQuota(), plan); err != nil {
			return hit, err, true
		}
		workingSegment = append(workingSegment, assistant)
		workingSegment = append(workingSegment, toolMessages...)

		if instance.IsMemoryEnabled() && containsMemoryToolCall(assistant.ToolCalls) {
			memories, listErr := memory.ListByUser(db, user.GetID(db), "", memory.DefaultMemoryLimit)
			if listErr != nil {
				globals.Warn(fmt.Sprintf("[memory] failed to refresh memories: %s", listErr.Error()))
			} else {
				memoryPrompt = memory.BuildMemoryPrompt(memories)
			}
		}
	}

	globals.Warn(fmt.Sprintf(
		"[tools] reached max tool rounds for model %s max_rounds=%d; requesting final answer without tools",
		model,
		maxToolRounds,
	))

	workingSegment = append(workingSegment, buildToolLimitSystemMessage())
	finalBuffer := utils.NewBuffer(model, workingSegment, liveBuffer.GetCharge())
	liveBuffer.InputTokens += finalBuffer.CountInputToken()
	liveBuffer.Quota += utils.CountInputQuota(liveBuffer.GetCharge(), finalBuffer.CountInputToken())

	props := buildChatProps(
		conn,
		instance,
		model,
		workingSegment,
		finalBuffer,
		memoryPrompt,
		recentChatsPrompt,
		nil,
		nil,
		true,
	)

	hit, err, interrupted = createRoundTask(conn, user, finalBuffer, liveBuffer, db, cache, group, props, plan, createRemoveSignalHandler(db, instance))
	if err != nil || interrupted {
		return hit, err, interrupted
	}

	syncToolFinalMetadata(liveBuffer, finalBuffer)

	return hit, nil, false
}

func latestMessageContent(messages []globals.Message) (string, bool) {
	if len(messages) == 0 {
		return "", false
	}
	return messages[len(messages)-1].Content, true
}

func createChatTask(
	conn *Connection, user *auth.User, buffer *utils.Buffer, db *sql.DB, cache *redis.Client,
	model string, instance *conversation.Conversation, segment []globals.Message, plan bool,
) (hit bool, err error, interrupted bool) {
	chunkChan := make(chan partialChunk, 24) // the channel to send the chunk data
	interruptSignal := make(chan error, 1)   // the signal to interrupt the chat task routine
	stopSignal, stopPolling := createStopSignal(conn, createRemoveSignalHandler(db, instance))

	defer func() {
		stopPolling()
	}()

	// create a new chat request routine
	go func() {
		defer func() {
			if r := recover(); r != nil && !strings.Contains(fmt.Sprintf("%s", r), "closed channel") {
				stack := debug.Stack()
				globals.Warn(fmt.Sprintf("caught panic from chat request: %s\n%s", r, stack))
			}
		}()

		if globals.IsVideoModel(model) {
			prompt, ok := latestMessageContent(segment)
			if !ok {
				chunkChan <- partialChunk{Chunk: nil, End: true, Hit: false, Error: errors.New("empty message segment")}
				return
			}

			props := adaptercommon.CreateVideoProps(&adaptercommon.VideoProps{
				Model:  model,
				Prompt: prompt,
			})
			props.User = auth.GetUsernameString(db, user)

			var finalJobJson string
			hit, err := channel.NewVideoRequestWithCache(
				cache, buffer,
				auth.GetGroup(db, user),
				props,
				func(data *globals.Chunk) error {
					if data != nil && data.Content != "" {
						if strings.HasPrefix(data.Content, "{") && strings.Contains(data.Content, "\"id\"") && strings.Contains(data.Content, "\"status\"") {
							finalJobJson = data.Content

							job, err := utils.UnmarshalString[VideoJob](data.Content)
							if err == nil && job.Id != "" && job.Status == "completed" {
								backendUrl := channel.SystemInstance.GetBackend()
								videoUrl := fmt.Sprintf("%s/videos/%s/content", backendUrl, job.Id)
								videoMarkdown := utils.GetVideoMarkdown(videoUrl, "video")

								chunkChan <- partialChunk{Chunk: &globals.Chunk{Content: videoMarkdown}, End: false, Hit: false, Error: nil}
								return nil
							}
						}
					}
					// Send original content for progress updates and other messages
					chunkChan <- partialChunk{Chunk: data, End: false, Hit: false, Error: nil}
					return nil
				},
			)

			if instance != nil && finalJobJson != "" {
				job, err := utils.UnmarshalString[VideoJob](finalJobJson)
				if err != nil {
					globals.Warn(fmt.Sprintf("[video] failed to parse job JSON: %s, finalJobJson: %s", err.Error(), finalJobJson))
				} else if job.Id == "" {
					globals.Warn(fmt.Sprintf("[video] job.Id is empty after parsing, finalJobJson: %s", finalJobJson))
				} else {
					globals.Debug(fmt.Sprintf("[video] saving task_id %s to conversation %d", job.Id, instance.GetId()))
					instance.SetTaskID(job.Id)
					if !instance.SaveConversation(db) {
						globals.Warn(fmt.Sprintf("[video] failed to save conversation with task_id %s", job.Id))
					} else {
						globals.Debug(fmt.Sprintf("[video] successfully saved task_id %s to conversation %d", job.Id, instance.GetId()))
					}
				}
			} else {
				if instance == nil {
					globals.Warn("[video] instance is nil, cannot save task_id")
				} else if finalJobJson == "" {
					globals.Warn("[video] finalJobJson is empty, cannot save task_id")
				}
			}

			chunkChan <- partialChunk{Chunk: nil, End: true, Hit: hit, Error: err}
			return
		}

		hit, err := channel.NewChatRequestWithCache(
			cache, buffer,
			auth.GetGroup(db, user),
			adaptercommon.CreateChatProps(&adaptercommon.ChatProps{
				Model:                model,
				Message:              segment,
				CustomInstruction:    instance.GetCustomInstruction(),
				EnableWeb:            instance.IsEnableWeb(),
				EnableWebSearch:      instance.IsEnableWebSearch(),
				EnableURLContext:     instance.IsEnableURLContext(),
				EnableCodeExecution:  instance.IsEnableCodeExecution() && globals.SupportGeminiCodeExecution(model),
				EnableXSearch:        instance.IsEnableXSearch(),
				GeminiThinkingBudget: instance.GetGeminiThinkingBudget(),
				MaxTokens:            instance.GetMaxTokens(),
				Temperature:          instance.GetTemperature(),
				TopP:                 instance.GetTopP(),
				TopK:                 instance.GetTopK(),
				PresencePenalty:      instance.GetPresencePenalty(),
				FrequencyPenalty:     instance.GetFrequencyPenalty(),
				RepetitionPenalty:    instance.GetRepetitionPenalty(),
				ClientContext:        extractClientContext(conn.GetCtx()),
			}, buffer),

			// the function to handle the chunk data
			func(data *globals.Chunk) error {
				// if interrupt signal is received
				if len(interruptSignal) > 0 {
					return errors.New(interruptMessage)
				}

				// send the chunk data to the channel
				chunkChan <- partialChunk{
					Chunk: data,
					End:   false,
					Hit:   false,
					Error: nil,
				}
				return nil
			},
		)

		// chat request routine is done
		chunkChan <- partialChunk{
			Chunk: nil,
			End:   true,
			Hit:   hit,
			Error: err,
		}
	}()

	for {
		select {
		case data := <-chunkChan:
			if data.Error != nil && data.Error.Error() == interruptMessage {
				interrupted = true
				if data.End {
					return hit, nil, true
				}

				// skip the interrupt message
				continue
			}

			hit = data.Hit
			err = data.Error

			if data.End {
				return
			}

			if data.Chunk != nil && data.Chunk.ToolCall != nil {
				if err := sendToolCallEvents(conn, data.Chunk.ToolCall, "start", buffer.GetQuota(), plan); err != nil {
					globals.Warn(fmt.Sprintf("failed to send tool call event to client: %s", err.Error()))
					signalInterrupt(interruptSignal, err)
					waitForRoundTaskEnd(chunkChan)
					return hit, nil, true
				}
			}

			message := buffer.WriteChunk(data.Chunk)
			if !conn.TrySendClient(globals.ChatSegmentResponse{
				Message: message,
				Quota:   buffer.GetQuota(),
				End:     false,
				Plan:    plan,
			}) {
				continue
			}

		case signal := <-stopSignal:
			// if stop signal is received
			if signal {
				globals.Info(fmt.Sprintf("client stopped the chat request (model: %s, client: %s)", model, conn.GetCtx().ClientIP()))
				_ = conn.SendClient(globals.ChatSegmentResponse{
					Quota: buffer.GetQuota(),
					End:   true,
					Plan:  plan,
				})
				signalInterrupt(interruptSignal, errors.New("signal"))
				waitForRoundTaskEnd(chunkChan)

				return hit, nil, true
			}
		}
	}
}

func extractAssistantMessageFromBuffer(buffer *utils.Buffer, interrupted bool, plan bool) globals.Message {
	if buffer.IsEmpty() {
		geminiHiddenMetadata := buffer.GetGeminiHiddenMetadata()
		claudeHiddenMetadata := buffer.GetClaudeHiddenMetadata()
		if buffer.HasHiddenMetadataOnly() {
			return globals.Message{
				Role:                 globals.Assistant,
				Content:              "",
				GeminiHiddenMetadata: geminiHiddenMetadata,
				ClaudeHiddenMetadata: claudeHiddenMetadata,
			}
		}

		return globals.Message{
			Role:    globals.Assistant,
			Content: defaultMessage,
		}
	}

	message := globals.Message{
		Role:                 globals.Assistant,
		Content:              buffer.ReadWithDefault(defaultMessage),
		GeminiHiddenMetadata: buffer.GetGeminiHiddenMetadata(),
		ClaudeHiddenMetadata: buffer.GetClaudeHiddenMetadata(),
		Plan:                 plan,
	}
	if buffer.GetCharge() != nil {
		message.Quota = buffer.GetQuota()
	}

	// Interrupted streams may contain partial/incomplete tool payloads.
	// Keep visible text, but avoid persisting broken function-calling state
	// or incomplete hidden reasoning context.
	if interrupted {
		return message
	}

	message.ReasoningContent = buffer.GetReasoningContent()
	message.ToolCalls = buffer.GetToolCalls()
	message.FunctionCall = buffer.GetFunctionCall()
	return message
}

func ChatHandler(conn *Connection, user *auth.User, instance *conversation.Conversation, restart bool) globals.Message {
	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			globals.Warn(fmt.Sprintf("caught panic from chat handler: %s (instance: %s, client: %s)\n%s",
				err, instance.GetModel(), conn.GetCtx().ClientIP(), stack,
			))
		}
	}()

	db := conn.GetDB()
	cache := conn.GetCache()

	model := instance.GetModel()
	group := auth.GetGroup(db, user)
	segment := web.ToChatSearched(instance, restart, group, cache)
	segment = adapter.ClearMessages(model, segment)

	check, plan := auth.CanEnableModelWithSubscription(db, cache, user, model, segment)
	conn.Send(globals.ChatSegmentResponse{
		Conversation: instance.GetId(),
	})

	if check != nil {
		message := check.Error()
		conn.Send(globals.ChatSegmentResponse{
			Message: message,
			Quota:   0,
			End:     true,
		})
		return globals.Message{
			Role:    globals.Assistant,
			Content: message,
		}
	}

	buffer := utils.NewBuffer(model, segment, channel.ChargeInstance.GetCharge(model))
	recordBuiltinToolRequest(buffer, instance, model)
	var hit bool
	var err error
	var interrupted bool
	toolEnabled := false
	if globals.IsVideoModel(model) {
		hit, err, interrupted = createChatTask(conn, user, buffer, db, cache, model, instance, segment, plan)
	} else {
		memCtx := buildMemoryContext(db, user, instance, model, group)
		fetchToolEnabled := instance.IsEnableFetch() && memory.CanUseToolCalls(model, group)
		tools := buildAvailableToolDefinitions(fetchToolEnabled, memCtx.Writable)
		toolEnabled = tools != nil
		if tools != nil {
			maxToolRounds := memory.MaxToolRounds
			if fetchToolEnabled {
				maxToolRounds = maxFetchToolRounds
			}
			hit, err, interrupted = createToolChatTask(conn, user, buffer, db, cache, model, group, instance, segment, plan, memCtx, tools, maxToolRounds)
		} else {
			props := buildChatProps(conn, instance, model, segment, buffer, memCtx.MemoryPrompt, memCtx.RecentChatsPrompt, nil, nil, false)
			hit, err, interrupted = createRoundTask(conn, user, buffer, buffer, db, cache, group, props, plan, createRemoveSignalHandler(db, instance))
		}
	}

	admin.AnalyseRequest(model, buffer, err)
	billing.RecordModelUsageMetric(db, model, buffer, err)
	if adapter.IsAvailableError(err) {
		globals.Warn(fmt.Sprintf("%s (model: %s, client: %s)", err, model, conn.GetCtx().ClientIP()))

		auth.RevertSubscriptionUsage(db, cache, user, model)
		conn.Send(globals.ChatSegmentResponse{
			Message: err.Error(),
			End:     true,
		})
		return globals.Message{
			Role:    globals.Assistant,
			Content: err.Error(),
		}
	}

	if !hit {
		CollectQuota(conn.GetCtx(), user, buffer, plan, err)
	}

	if !adapter.IsAvailableError(err) && user != nil && !buffer.IsEmpty() {
		userId := auth.GetId(db, user)
		billing.CreateRecord(
			db, userId, user.Username, "consume",
			buffer.GetTokenName(), model,
			int64(buffer.CountInputToken()), int64(buffer.CountOutputToken(false)),
			float64(buffer.GetRecordQuota()), buffer.GetDuration(),
			buffer.GetBillingDetail(), buffer.GetRecordPrompts(), buffer.GetRecordResponsePrompts(),
			buffer.GetChannelId(), buffer.GetChannelName(),
		)
	}

	if interrupted {
		return extractAssistantMessageFromBuffer(buffer, true, plan)
	}

	if buffer.IsEmpty() {
		globals.Warn(fmt.Sprintf(
			"[chat] empty response for model %s (interrupted=%v, tool_enabled=%v)",
			model,
			interrupted,
			toolEnabled,
		))
		if buffer.HasHiddenMetadataOnly() {
			conn.Send(globals.ChatSegmentResponse{
				End: true,
			})
			return extractAssistantMessageFromBuffer(buffer, interrupted, plan)
		}

		conn.Send(globals.ChatSegmentResponse{
			Message: defaultMessage,
			End:     true,
		})
		return extractAssistantMessageFromBuffer(buffer, interrupted, plan)
	}

	conn.Send(globals.ChatSegmentResponse{
		End:   true,
		Quota: buffer.GetQuota(),
		Plan:  plan,
	})

	return extractAssistantMessageFromBuffer(buffer, interrupted, plan)
}
