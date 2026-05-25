package palm2

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"encoding/json"
	"fmt"
	"strings"
)

func getGeminiRole(role string) string {
	switch role {
	case globals.User:
		return GeminiUserType
	case globals.Tool:
		return GeminiUserType
	case globals.Assistant, globals.System:
		return GeminiModelType
	default:
		return GeminiUserType
	}
}

func getMimeType(content string) string {
	segment := strings.Split(content, ".")
	if len(segment) == 0 || len(segment) == 1 {
		return "image/png"
	}

	suffix := strings.TrimSpace(strings.ToLower(segment[len(segment)-1]))

	switch suffix {
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "heif":
		return "image/heif"
	case "heic":
		return "image/heic"
	default:
		return "image/png"
	}
}

func getGeminiContent(parts []GeminiChatPart, content string, model string) []GeminiChatPart {
	if model == globals.GeminiPro {
		if strings.TrimSpace(content) == "" {
			return parts
		}
		return append(parts, GeminiChatPart{
			Text: &content,
		})
	}

	raw, urls := utils.ExtractImages(content, true)
	if len(urls) > geminiMaxImages {
		urls = urls[:geminiMaxImages]
	}

	if strings.TrimSpace(raw) != "" {
		parts = append(parts, GeminiChatPart{
			Text: &raw,
		})
	}

	for _, url := range urls {
		data, err := utils.ConvertToBase64(url)
		if err != nil {
			continue
		}

		parts = append(parts, GeminiChatPart{
			InlineData: &GeminiInlineData{
				MimeType: getMimeType(url),
				Data:     data,
			},
		})
	}

	return parts
}

func toJSONObject(data string) interface{} {
	if len(strings.TrimSpace(data)) == 0 {
		return map[string]interface{}{}
	}

	var result interface{}
	if err := json.Unmarshal([]byte(data), &result); err == nil {
		return result
	}

	return map[string]interface{}{
		"content": data,
	}
}

func toGeminiFunctionArgs(data string) interface{} {
	value := toJSONObject(data)

	if obj, ok := value.(map[string]interface{}); ok {
		return obj
	}

	return map[string]interface{}{
		"value": value,
	}
}

func getGeminiFunctionCallPart(name string, arguments string) GeminiChatPart {
	return GeminiChatPart{
		FunctionCall: &GeminiFunctionCall{
			Name: name,
			Args: toGeminiFunctionArgs(arguments),
		},
	}
}

func getToolNameByID(history []globals.Message, id string) string {
	for _, message := range history {
		if message.ToolCalls == nil {
			continue
		}

		for _, call := range *message.ToolCalls {
			if call.Id == id {
				return call.Function.Name
			}
		}
	}

	return ""
}

func getGeminiToolResponsePart(history []globals.Message, message globals.Message) *GeminiChatPart {
	name := ""
	if message.Name != nil {
		name = *message.Name
	} else if message.ToolCallId != nil {
		name = getToolNameByID(history, *message.ToolCallId)
	}

	if len(name) == 0 {
		return nil
	}

	return &GeminiChatPart{
		FunctionResponse: &GeminiFunctionResponse{
			Name: name,
			Response: map[string]interface{}{
				"output": toJSONObject(message.Content),
			},
		},
	}
}

func replayGeminiHiddenMetadata(parts []GeminiChatPart, metadata *globals.GeminiHiddenMetadata) []GeminiChatPart {
	normalized := metadata.Normalized(globals.GeminiThoughtSignatureLimit)
	if normalized == nil || len(normalized.ThoughtSignatures) == 0 {
		return parts
	}

	result := make([]GeminiChatPart, len(parts))
	copy(result, parts)

	index := 0
	// We only persist a flat signature list, so original per-part placement is unavailable.
	// For deterministic replay and Gemini function-calling continuity, map signatures to
	// functionCall parts first. Extra signatures cannot be sent as metadata-only request
	// parts because Vertex AI Express rejects parts with no initialized data oneof.
	for i := range result {
		if index >= len(normalized.ThoughtSignatures) {
			return result
		}

		if result[i].FunctionCall == nil || result[i].ThoughtSignature != nil {
			continue
		}

		result[i].ThoughtSignature = utils.ToPtr(normalized.ThoughtSignatures[index])
		index++
	}

	return result
}

func hasGeminiRequestPartData(part GeminiChatPart) bool {
	if part.Text != nil && strings.TrimSpace(*part.Text) != "" {
		return true
	}
	if part.InlineData != nil && strings.TrimSpace(part.InlineData.Data) != "" {
		return true
	}
	return part.FunctionCall != nil || part.FunctionResponse != nil
}

func filterGeminiRequestParts(parts []GeminiChatPart) []GeminiChatPart {
	result := make([]GeminiChatPart, 0, len(parts))
	for _, part := range parts {
		if !hasGeminiRequestPartData(part) {
			continue
		}
		result = append(result, part)
	}
	return result
}

func canStartGeminiContents(parts []GeminiChatPart) bool {
	for _, part := range parts {
		if part.Text != nil && strings.TrimSpace(*part.Text) != "" {
			return true
		}
		if part.InlineData != nil && strings.TrimSpace(part.InlineData.Data) != "" {
			return true
		}
	}
	return false
}

func getGeminiParts(model string, history []globals.Message, message globals.Message) []GeminiChatPart {
	var parts []GeminiChatPart

	if message.ToolCalls != nil && len(*message.ToolCalls) > 0 {
		parts = make([]GeminiChatPart, 0, len(*message.ToolCalls))
		for _, call := range *message.ToolCalls {
			parts = append(parts, getGeminiFunctionCallPart(call.Function.Name, call.Function.Arguments))
		}
	} else if message.FunctionCall != nil {
		parts = []GeminiChatPart{
			getGeminiFunctionCallPart(message.FunctionCall.Name, message.FunctionCall.Arguments),
		}
	} else if message.Role == globals.Tool {
		part := getGeminiToolResponsePart(history, message)
		if part != nil {
			parts = []GeminiChatPart{*part}
		}
	} else {
		parts = getGeminiContent(make([]GeminiChatPart, 0), message.Content, model)
	}

	if message.Role == globals.Assistant {
		parts = replayGeminiHiddenMetadata(parts, message.GeminiHiddenMetadata)
	}

	return filterGeminiRequestParts(parts)
}

func appendGeminiContent(result []GeminiContent, role string, parts []GeminiChatPart) []GeminiContent {
	if len(parts) == 0 {
		return result
	}

	if len(result) > 0 && role == result[len(result)-1].Role {
		last := result[len(result)-1].Parts
		if !hasGeminiThoughtSignatureParts(last) && !hasGeminiThoughtSignatureParts(parts) {
			result[len(result)-1].Parts = append(last, parts...)
			return result
		}
	}

	return append(result, GeminiContent{
		Role:  role,
		Parts: parts,
	})
}

func hasGeminiThoughtSignatureParts(parts []GeminiChatPart) bool {
	for _, part := range parts {
		if part.ThoughtSignature != nil {
			return true
		}
	}

	return false
}

func (c *ChatInstance) GetGeminiSystemInstruction(model string, messages []globals.Message) *GeminiContent {
	parts := make([]GeminiChatPart, 0)
	for _, item := range messages {
		if item.Role != globals.System || len(item.Content) == 0 {
			continue
		}

		raw, _ := utils.ExtractImages(item.Content, true)
		raw = strings.TrimSpace(raw)
		if len(raw) == 0 {
			continue
		}

		parts = append(parts, GeminiChatPart{
			Text: &raw,
		})
	}

	if len(parts) == 0 {
		return nil
	}

	return &GeminiContent{Parts: parts}
}

func getGeminiThinkingConfig(props *adaptercommon.ChatProps) *GeminiThinkingConfig {
	if props == nil || props.GeminiThinkingBudget == nil {
		return nil
	}

	if globals.IsGeminiNoThinkingModel(props.Model) {
		return nil
	}

	if globals.SupportGeminiThinkingLevel(props.Model) {
		level := getGeminiThinkingLevel(props.Model, *props.GeminiThinkingBudget)
		if level == "" {
			return nil
		}

		config := &GeminiThinkingConfig{
			ThinkingLevel: utils.ToPtr(level),
		}
		if level != "minimal" {
			config.IncludeThoughts = utils.ToPtr(true)
		}
		return config
	}

	if !globals.SupportGeminiThinkingBudget(props.Model) {
		return nil
	}

	thinkingBudget := normalizeGeminiThinkingBudget(props.Model, *props.GeminiThinkingBudget)
	config := &GeminiThinkingConfig{
		ThinkingBudget: &thinkingBudget,
	}

	if thinkingBudget > 0 {
		config.IncludeThoughts = utils.ToPtr(true)
	}

	return config
}

func getGeminiThinkingLevel(model string, value int) string {
	switch {
	case value <= 0 && strings.HasPrefix(model, "gemini-3.1-pro"):
		return "low"
	case value <= 0:
		return "minimal"
	case value <= 2048:
		return "low"
	case value <= 6144:
		return "medium"
	default:
		return "high"
	}
}

func normalizeGeminiThinkingBudget(model string, value int) int {
	switch {
	case strings.HasPrefix(model, "gemini-2.5-pro"):
		if value <= 0 {
			return 128
		}
		if value < 128 {
			return 128
		}
		if value > 32768 {
			return 32768
		}
		return value
	case strings.HasPrefix(model, "gemini-2.5-flash-lite"):
		if value == -1 {
			return -1
		}
		if value > 0 && value < 512 {
			return 512
		}
		if value > 24576 {
			return 24576
		}
		return value
	default:
		if value == -1 {
			return -1
		}
		if value > 24576 {
			return 24576
		}
		return value
	}
}

func (c *ChatInstance) GetGeminiContents(model string, message []globals.Message) []GeminiContent {
	// gemini role should be user-model

	result := make([]GeminiContent, 0)
	for i, item := range message {
		if item.Role == globals.System {
			continue
		}

		parts := getGeminiParts(model, message[:i], item)
		if len(parts) == 0 {
			// gemini model: message must include non empty content or valid tool payload
			continue
		}

		role := getGeminiRole(item.Role)
		if len(result) == 0 {
			if role != GeminiUserType || !canStartGeminiContents(parts) {
				// Context windows can start in the middle of a conversation after the user
				// changes the retained context count. Gemini requires the first content
				// turn to be a real user prompt, so drop orphaned leading model/tool turns.
				continue
			}
		}

		result = appendGeminiContent(result, role, parts)
	}

	return result
}

func getGeminiTools(tools *globals.FunctionTools) []GeminiTool {
	if tools == nil || len(*tools) == 0 {
		return nil
	}

	declarations := make([]GeminiFunctionDeclaration, 0, len(*tools))
	for _, tool := range *tools {
		if tool.Type != "function" {
			continue
		}

		declarations = append(declarations, GeminiFunctionDeclaration{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  tool.Function.Parameters,
		})
	}

	if len(declarations) == 0 {
		return nil
	}

	return []GeminiTool{{FunctionDeclarations: declarations}}
}

func getGeminiBuiltinWebTools(enableSearch bool, enableURLContext bool) []GeminiTool {
	tools := make([]GeminiTool, 0, 2)

	if enableURLContext {
		tools = append(tools, GeminiTool{URLContext: &GeminiURLContext{}})
	}

	if enableSearch {
		tools = append(tools, GeminiTool{GoogleSearch: &GeminiGoogleSearch{}})
	}

	if len(tools) == 0 {
		return nil
	}

	return tools
}

func mergeGeminiTools(tools ...[]GeminiTool) []GeminiTool {
	merged := make([]GeminiTool, 0)
	for _, group := range tools {
		if len(group) == 0 {
			continue
		}
		merged = append(merged, group...)
	}

	if len(merged) == 0 {
		return nil
	}

	return merged
}

func getGeminiToolConfig(toolChoice *interface{}) *GeminiToolConfig {
	if toolChoice == nil || *toolChoice == nil {
		return nil
	}

	config := &GeminiToolConfig{
		FunctionCallingConfig: &GeminiFunctionCallingConfig{
			Mode: "AUTO",
		},
	}

	switch value := (*toolChoice).(type) {
	case string:
		switch strings.ToLower(value) {
		case "none":
			config.FunctionCallingConfig.Mode = "NONE"
		case "required":
			config.FunctionCallingConfig.Mode = "ANY"
		default:
			config.FunctionCallingConfig.Mode = "AUTO"
		}
		return config
	case map[string]interface{}:
		if fn, ok := value["function"].(map[string]interface{}); ok {
			if name, ok := fn["name"].(string); ok && len(name) > 0 {
				config.FunctionCallingConfig.Mode = "ANY"
				config.FunctionCallingConfig.AllowedFunctionNames = []string{name}
			}
		}
		return config
	default:
		return config
	}
}

func getGeminiToolCalls(parts []GeminiChatPart) *globals.ToolCalls {
	calls := make(globals.ToolCalls, 0)

	for i, part := range parts {
		if part.FunctionCall == nil {
			continue
		}

		args := "{}"
		if part.FunctionCall.Args != nil {
			args = utils.Marshal(part.FunctionCall.Args)
		}

		calls = append(calls, globals.ToolCall{
			Type: "function",
			Id:   fmt.Sprintf("gemini_call_%d", i),
			Function: globals.ToolCallFunction{
				Name:      part.FunctionCall.Name,
				Arguments: args,
			},
		})
	}

	if len(calls) == 0 {
		return nil
	}

	return &calls
}

func getGeminiHiddenMetadataFromParts(parts []GeminiChatPart) *globals.GeminiHiddenMetadata {
	signatures := make([]string, 0, len(parts))
	for _, part := range parts {
		if part.ThoughtSignature == nil {
			continue
		}

		signatures = append(signatures, *part.ThoughtSignature)
	}

	return (&globals.GeminiHiddenMetadata{
		ThoughtSignatures: signatures,
	}).Normalized(globals.GeminiThoughtSignatureLimit)
}

func (c *ChatInstance) GetGeminiText(parts []GeminiChatPart) string {
	builder := strings.Builder{}

	for _, part := range parts {
		if part.Text == nil {
			continue
		}
		builder.WriteString(*part.Text)
	}

	return builder.String()
}

func getGeminiReasoningText(parts []GeminiChatPart) string {
	builder := strings.Builder{}

	for _, part := range parts {
		if part.Text == nil || !part.Thought {
			continue
		}
		builder.WriteString(*part.Text)
	}

	return builder.String()
}

func getGeminiAnswerText(parts []GeminiChatPart) string {
	builder := strings.Builder{}

	for _, part := range parts {
		if part.Text == nil || part.Thought {
			continue
		}
		builder.WriteString(*part.Text)
	}

	return builder.String()
}

func (c *ChatInstance) GetGeminiChatText(model string, parts []GeminiChatPart) string {
	if globals.IsGeminiNoThinkingModel(model) {
		return strings.TrimSpace(getGeminiAnswerText(parts))
	}

	reasoning := strings.TrimSpace(getGeminiReasoningText(parts))
	answer := strings.TrimSpace(getGeminiAnswerText(parts))

	if reasoning == "" {
		return answer
	}

	if answer == "" {
		return fmt.Sprintf("<think>\n%s\n</think>", reasoning)
	}

	return fmt.Sprintf("<think>\n%s\n</think>\n\n%s", reasoning, answer)
}

func (c *ChatInstance) GetGeminiStreamText(model string, parts []GeminiChatPart) string {
	builder := strings.Builder{}

	for _, part := range parts {
		if part.Text == nil || *part.Text == "" {
			continue
		}

		if part.Thought {
			if globals.IsGeminiNoThinkingModel(model) {
				continue
			}
			if c.isFirstReasoning {
				c.isFirstReasoning = false
				builder.WriteString("<think>\n")
			}
			builder.WriteString(*part.Text)
			continue
		}

		if !c.isFirstReasoning && !c.isReasonOver {
			c.isReasonOver = true
			builder.WriteString("\n</think>\n\n")
		}

		builder.WriteString(*part.Text)
	}

	return builder.String()
}
