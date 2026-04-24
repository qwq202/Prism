package responsescompat

import (
	"chat/globals"
	"chat/utils"
	"strings"
)

func NormalizeRole(role string) string {
	switch role {
	case globals.System, globals.User, globals.Assistant:
		return role
	default:
		return globals.User
	}
}

func MessageText(message globals.Message) string {
	if message.Content != "" {
		return message.Content
	}

	if message.FunctionCall != nil {
		return utils.Marshal(*message.FunctionCall)
	}

	if message.ToolCalls != nil {
		return utils.Marshal(*message.ToolCalls)
	}

	return ""
}

func ReplayFunctionCalls(message globals.Message) []interface{} {
	if message.ToolCalls == nil || len(*message.ToolCalls) == 0 {
		return nil
	}

	items := make([]interface{}, 0, len(*message.ToolCalls))
	for _, toolCall := range *message.ToolCalls {
		items = append(items, OutputItem{
			Type:      "function_call",
			Name:      toolCall.Function.Name,
			Arguments: toolCall.Function.Arguments,
			CallID:    toolCall.Id,
		})
	}

	return items
}

func FunctionCallOutput(message globals.Message) *FunctionCallOutputInput {
	if message.ToolCallId == nil || strings.TrimSpace(*message.ToolCallId) == "" {
		return nil
	}

	return &FunctionCallOutputInput{
		Type:   "function_call_output",
		CallID: strings.TrimSpace(*message.ToolCallId),
		Output: message.Content,
	}
}

func ExtractOutputText(items []OutputItem) string {
	chunks := make([]string, 0)
	for _, item := range items {
		if item.Type != "message" || item.Role != globals.Assistant {
			continue
		}

		for _, content := range item.Content {
			if content.Type == "output_text" && content.Text != "" {
				chunks = append(chunks, content.Text)
			}
		}
	}

	return strings.Join(chunks, "")
}

func ExtractToolCalls(items []OutputItem) *globals.ToolCalls {
	toolCalls := make(globals.ToolCalls, 0)
	for idx, item := range items {
		if item.Type != "function_call" || strings.TrimSpace(item.Name) == "" {
			continue
		}

		toolCalls = append(toolCalls, globals.ToolCall{
			Index: utils.ToPtr(idx),
			Type:  "function",
			Id:    item.CallID,
			Function: globals.ToolCallFunction{
				Name:      item.Name,
				Arguments: item.Arguments,
			},
		})
	}

	if len(toolCalls) == 0 {
		return nil
	}

	return &toolCalls
}

func BuildResponseChunk(output []OutputItem) *globals.Chunk {
	content := ExtractOutputText(output)
	toolCalls := ExtractToolCalls(output)
	if content == "" && toolCalls == nil {
		return &globals.Chunk{}
	}

	return &globals.Chunk{
		Content:  content,
		ToolCall: toolCalls,
	}
}

func EmitOutputText(delta string) *globals.Chunk {
	if delta == "" {
		return nil
	}

	return &globals.Chunk{
		Content: delta,
	}
}

func EmitFunctionCallEvent(item *OutputItem) *globals.Chunk {
	if item == nil || item.Type != "function_call" || strings.TrimSpace(item.Name) == "" {
		return nil
	}

	toolCalls := globals.ToolCalls{
		{
			Index: utils.ToPtr(0),
			Type:  "function",
			Id:    item.CallID,
			Function: globals.ToolCallFunction{
				Name:      item.Name,
				Arguments: item.Arguments,
			},
		},
	}

	return &globals.Chunk{
		ToolCall: &toolCalls,
	}
}
