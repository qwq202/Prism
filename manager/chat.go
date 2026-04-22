package manager

import (
	"chat/adapter"
	adaptercommon "chat/adapter/common"
	"chat/addition/web"
	"chat/admin"
	"chat/auth"
	"chat/billing"
	"chat/channel"
	"chat/globals"
	"chat/manager/conversation"
	"chat/manager/memory"
	"chat/utils"
	"time"

	"database/sql"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

const defaultMessage = "empty response"
const interruptMessage = "interrupted"

func CollectQuota(c *gin.Context, user *auth.User, buffer *utils.Buffer, uncountable bool, err error) {
	db := utils.GetDBFromContext(c)
	quota := buffer.GetQuota()

	if user == nil || quota <= 0 {
		return
	}

	if buffer.IsEmpty() || err != nil {
		return
	}

	if !uncountable {
		user.UseQuota(db, quota)
	}
}

type partialChunk struct {
	Chunk *globals.Chunk
	End   bool
	Hit   bool
	Error error
}

func createStopSignal(conn *Connection) chan bool {
	stopSignal := make(chan bool, 1)
	go func(conn *Connection, stopSignal chan bool) {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer func() {
			ticker.Stop()
			if r := recover(); r != nil && !strings.Contains(fmt.Sprintf("%s", r), "closed channel") {
				stack := debug.Stack()
				globals.Warn(fmt.Sprintf("caught panic from stop signal: %s\n%s", r, stack))
			}
		}()

		for {
			select {
			case <-ticker.C:
				state := conn.PeekStop() != nil // check the stop state
				stopSignal <- state

				if state {
					break
				}
			}
		}
	}(conn, stopSignal)

	return stopSignal
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
	return adaptercommon.CreateChatProps(&adaptercommon.ChatProps{
		Model:                model,
		OriginalModel:        model,
		Message:              segment,
		CustomInstruction:    instance.GetCustomInstruction(),
		MemoryPrompt:         memoryPrompt,
		RecentChatsPrompt:    recentChatsPrompt,
		Tools:                tools,
		ToolChoice:           toolChoice,
		EnableWeb:            instance.IsEnableWeb(),
		EnableWebSearch:      instance.IsEnableWebSearch(),
		EnableURLContext:     instance.IsEnableURLContext(),
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
) (hit bool, err error, interrupted bool) {
	chunkChan := make(chan partialChunk, 24)
	interruptSignal := make(chan error, 1)
	stopSignal := createStopSignal(conn)

	defer func() {
		close(interruptSignal)
		close(stopSignal)
		close(chunkChan)
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
			hit, err = channel.NewChatRequestWithCache(cache, captureBuffer, group, props, func(data *globals.Chunk) error {
				if len(interruptSignal) > 0 {
					return errors.New(interruptMessage)
				}

				chunkChan <- partialChunk{Chunk: data, End: false, Hit: false, Error: nil}
				return nil
			})
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

				if err := conn.SendClient(globals.ChatSegmentResponse{
					Message: content,
					Quota:   streamBuffer.GetQuota(),
					End:     false,
					Plan:    plan,
				}); err != nil {
					globals.Warn(fmt.Sprintf("failed to send message to client: %s", err.Error()))
					interruptSignal <- err
					return hit, nil, true
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
				interruptSignal <- errors.New("signal")
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

func buildMemoryContext(db *sql.DB, user *auth.User, instance *conversation.Conversation, model string, group string) memoryContext {
	ctx := memoryContext{}
	if user == nil {
		return ctx
	}

	userID := user.GetID(db)
	if instance.IsMemoryEnabled() {
		memories, err := memory.ListByUser(db, userID, "", memory.DefaultMemoryLimit)
		if err != nil {
			globals.Warn(fmt.Sprintf("[memory] failed to load memories: %s", err.Error()))
		} else {
			ctx.MemoryPrompt = memory.BuildMemoryPrompt(memories)
			ids := make([]int64, 0, len(memories))
			for _, item := range memories {
				ids = append(ids, item.ID)
			}
			if err := memory.Touch(db, userID, ids); err != nil {
				globals.Warn(fmt.Sprintf("[memory] failed to touch memories: %s", err.Error()))
			}
		}

		ctx.Writable = memory.CanUseWritableTools(model, group)
	}

	if instance.IsMemoryHistoryEnabled() {
		chats, err := memory.ListRecentConversations(db, userID, instance.GetId(), memory.DefaultRecentChatNum)
		if err != nil {
			globals.Warn(fmt.Sprintf("[memory] failed to load recent chats: %s", err.Error()))
		} else {
			ctx.RecentChatsPrompt = memory.BuildRecentChatsPrompt(chats)
		}
	}

	return ctx
}

func createMemoryToolChatTask(
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
) (hit bool, err error, interrupted bool) {
	workingSegment := utils.DeepCopy(segment)
	memoryPrompt := ctx.MemoryPrompt
	recentChatsPrompt := ctx.RecentChatsPrompt
	tools := memory.BuildToolDefinition()
	toolChoice := memory.BuildAutoToolChoice()

	for round := 0; round < memory.MaxToolRounds; round++ {
		roundBuffer := utils.NewBuffer(model, workingSegment, liveBuffer.GetCharge())
		if round > 0 {
			liveBuffer.InputTokens += roundBuffer.CountInputToken()
			liveBuffer.Quota += utils.CountInputQuota(liveBuffer.GetCharge(), roundBuffer.CountInputToken())
		}

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

		hit, err, interrupted = createRoundTask(conn, user, roundBuffer, nil, db, cache, group, props, plan)
		if err != nil || interrupted {
			return hit, err, interrupted
		}

		assistant := extractAssistantMessageFromBuffer(roundBuffer, false)
		if assistant.ToolCalls == nil || len(*assistant.ToolCalls) == 0 {
			if content := roundBuffer.Read(); content != "" {
				liveBuffer.Write(content)
				if err := conn.SendClient(globals.ChatSegmentResponse{
					Message: content,
					Quota:   liveBuffer.GetQuota(),
					End:     false,
					Plan:    plan,
				}); err != nil {
					return hit, err, true
				}
			}
			liveBuffer.SetGeminiHiddenMetadata(roundBuffer.GetGeminiHiddenMetadata())
			return hit, nil, false
		}

		toolMessages := memory.ExecuteToolCalls(db, user, assistant.ToolCalls)
		workingSegment = append(workingSegment, assistant)
		workingSegment = append(workingSegment, toolMessages...)

		if instance.IsMemoryEnabled() {
			memories, listErr := memory.ListByUser(db, user.GetID(db), "", memory.DefaultMemoryLimit)
			if listErr != nil {
				globals.Warn(fmt.Sprintf("[memory] failed to refresh memories: %s", listErr.Error()))
			} else {
				memoryPrompt = memory.BuildMemoryPrompt(memories)
			}
		}
	}

	return hit, nil, false
}

func createChatTask(
	conn *Connection, user *auth.User, buffer *utils.Buffer, db *sql.DB, cache *redis.Client,
	model string, instance *conversation.Conversation, segment []globals.Message, plan bool,
) (hit bool, err error, interrupted bool) {
	chunkChan := make(chan partialChunk, 24) // the channel to send the chunk data
	interruptSignal := make(chan error, 1)   // the signal to interrupt the chat task routine
	stopSignal := createStopSignal(conn)     // the signal to stop from the client

	defer func() {
		// close all channels
		close(interruptSignal)
		close(stopSignal)
		close(chunkChan)
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
			props := adaptercommon.CreateVideoProps(&adaptercommon.VideoProps{
				Model:  model,
				Prompt: segment[len(segment)-1].Content,
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

							job, err := utils.UnmarshalString[RelayVideoJob](data.Content)
							if err == nil && job.Id != "" && job.Status == "completed" {
								backendUrl := channel.SystemInstance.GetBackend()
								videoUrl := fmt.Sprintf("%s/v1/videos/%s/content", backendUrl, job.Id)
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
				job, err := utils.UnmarshalString[RelayVideoJob](finalJobJson)
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

			if err := conn.SendClient(globals.ChatSegmentResponse{
				Message: buffer.WriteChunk(data.Chunk),
				Quota:   buffer.GetQuota(),
				End:     false,
				Plan:    plan,
			}); err != nil {
				globals.Warn(fmt.Sprintf("failed to send message to client: %s", err.Error()))
				interruptSignal <- err
				return hit, nil, true
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
				interruptSignal <- errors.New("signal")

				return hit, nil, true
			}
		}
	}
}

func extractAssistantMessageFromBuffer(buffer *utils.Buffer, interrupted bool) globals.Message {
	if buffer.IsEmpty() {
		hiddenMetadata := buffer.GetGeminiHiddenMetadata()
		if buffer.HasHiddenMetadataOnly() {
			return globals.Message{
				Role:                 globals.Assistant,
				Content:              "",
				GeminiHiddenMetadata: hiddenMetadata,
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
	segment := adapter.ClearMessages(model, web.ToChatSearched(instance, restart, group, cache))

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
	var hit bool
	var err error
	var interrupted bool
	if globals.IsVideoModel(model) {
		hit, err, interrupted = createChatTask(conn, user, buffer, db, cache, model, instance, segment, plan)
	} else {
		memCtx := buildMemoryContext(db, user, instance, model, group)
		if memCtx.Writable {
			hit, err, interrupted = createMemoryToolChatTask(conn, user, buffer, db, cache, model, group, instance, segment, plan, memCtx)
		} else {
			props := buildChatProps(conn, instance, model, segment, buffer, memCtx.MemoryPrompt, memCtx.RecentChatsPrompt, nil, nil, false)
			hit, err, interrupted = createRoundTask(conn, user, buffer, buffer, db, cache, group, props, plan)
		}
	}

	admin.AnalyseRequest(model, buffer, err)
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
			"", buffer.GetRecordPrompts(), buffer.GetRecordResponsePrompts(),
			buffer.GetChannelId(), buffer.GetChannelName(),
		)
	}

	if interrupted {
		return extractAssistantMessageFromBuffer(buffer, true)
	}

	if buffer.IsEmpty() {
		if buffer.HasHiddenMetadataOnly() {
			conn.Send(globals.ChatSegmentResponse{
				End: true,
			})
			return extractAssistantMessageFromBuffer(buffer, interrupted)
		}

		conn.Send(globals.ChatSegmentResponse{
			Message: defaultMessage,
			End:     true,
		})
		return extractAssistantMessageFromBuffer(buffer, interrupted)
	}

	conn.Send(globals.ChatSegmentResponse{
		End:   true,
		Quota: buffer.GetQuota(),
		Plan:  plan,
	})

	return extractAssistantMessageFromBuffer(buffer, interrupted)
}
