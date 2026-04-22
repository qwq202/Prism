package xai

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"errors"
	"fmt"
	"strings"
)

func (c *ChatInstance) GetChatEndpoint() string {
	return fmt.Sprintf("%s/v1/responses", c.GetEndpoint())
}

func normalizeRole(role string) string {
	switch role {
	case globals.System, globals.User, globals.Assistant:
		return role
	default:
		return globals.User
	}
}

func getMessageText(message globals.Message) string {
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

func formatInputMessage(props *adaptercommon.ChatProps, message globals.Message) *InputMessage {
	text := getMessageText(message)

	if normalizeRole(message.Role) == globals.User {
		content, urls := utils.ExtractImages(text, true)
		items := []InputMessageContent{
			{
				Type: "input_text",
				Text: &content,
			},
		}

		for _, rawURL := range urls {
			url := rawURL
			if utils.IsInternalAttachmentURL(url) {
				if normalized, err := utils.NormalizeImageToVisionDataURL(url); err == nil {
					url = normalized
				} else {
					globals.Warn(fmt.Sprintf("[xai] cannot normalize attachment image: %s", err.Error()))
				}
			}
			if props.Buffer != nil {
				if obj, err := utils.NewImage(url); err == nil {
					props.Buffer.AddImage(obj)
				}
			}

			items = append(items, InputMessageContent{
				Type:     "input_image",
				ImageURL: &url,
			})
		}

		return &InputMessage{
			Role:    globals.User,
			Content: items,
		}
	}

	return &InputMessage{
		Role: normalizeRole(message.Role),
		Content: []InputMessageContent{
			{
				Type: "input_text",
				Text: &text,
			},
		},
	}
}

func formatReplayFunctionCalls(message globals.Message) []interface{} {
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

func formatFunctionCallOutput(message globals.Message) *FunctionCallOutputInput {
	if message.ToolCallId == nil || strings.TrimSpace(*message.ToolCallId) == "" {
		return nil
	}

	return &FunctionCallOutputInput{
		Type:   "function_call_output",
		CallID: strings.TrimSpace(*message.ToolCallId),
		Output: message.Content,
	}
}

func formatMessages(props *adaptercommon.ChatProps) ([]interface{}, bool) {
	input := make([]interface{}, 0, len(props.Message))
	hasImages := false

	for _, message := range props.Message {
		if message.Role == globals.Tool {
			if output := formatFunctionCallOutput(message); output != nil {
				input = append(input, *output)
			}
			continue
		}

		if message.Role == globals.Assistant && message.ToolCalls != nil && len(*message.ToolCalls) > 0 {
			if strings.TrimSpace(message.Content) != "" {
				formatted := formatInputMessage(props, message)
				if formatted != nil {
					input = append(input, *formatted)
				}
			}

			input = append(input, formatReplayFunctionCalls(message)...)
			continue
		}

		formatted := formatInputMessage(props, message)
		if formatted == nil {
			continue
		}

		for _, item := range formatted.Content {
			if item.Type == "input_image" && item.ImageURL != nil && strings.TrimSpace(*item.ImageURL) != "" {
				hasImages = true
				break
			}
		}

		input = append(input, *formatted)
	}

	return input, hasImages
}

func getResponseTools(props *adaptercommon.ChatProps) []ResponseTool {
	tools := make([]ResponseTool, 0)
	enabled := utils.ToPtr(true)

	if props.EnableWebSearch {
		tools = append(tools, ResponseTool{
			Type:                     "web_search",
			EnableImageUnderstanding: enabled,
		})
	}
	if props.EnableXSearch {
		tools = append(tools, ResponseTool{
			Type:                     "x_search",
			EnableImageUnderstanding: enabled,
			EnableVideoUnderstanding: enabled,
		})
	}

	if props.Tools == nil {
		return tools
	}

	for _, tool := range *props.Tools {
		if tool.Type != "" && tool.Type != "function" {
			continue
		}

		tools = append(tools, ResponseTool{
			Type:        "function",
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  tool.Function.Parameters,
		})
	}

	return tools
}

func getResponseInclude(stream bool, tools []ResponseTool) []string {
	if !stream || len(tools) == 0 {
		return nil
	}

	return []string{"verbose_streaming"}
}

func supportsVerboseStreaming(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if normalized == "" {
		return false
	}

	return !strings.Contains(normalized, "non-reasoning")
}

func (c *ChatInstance) GetChatBody(props *adaptercommon.ChatProps, stream bool) ResponseRequest {
	input, hasImages := formatMessages(props)
	var store *bool
	if hasImages {
		store = utils.ToPtr(false)
	}
	tools := getResponseTools(props)
	var include []string
	if supportsVerboseStreaming(props.Model) {
		include = getResponseInclude(stream, tools)
	}

	return ResponseRequest{
		Model:           props.Model,
		Input:           input,
		MaxOutputTokens: props.MaxTokens,
		Temperature:     props.Temperature,
		TopP:            props.TopP,
		Tools:           tools,
		ToolChoice:      props.ToolChoice,
		ResponseFormat:  props.ResponseFormat,
		Include:         include,
		Store:           store,
		Stream:          stream,
	}
}

func extractOutputText(form *ResponseResponse) string {
	if form == nil {
		return ""
	}

	chunks := make([]string, 0)
	for _, item := range form.Output {
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

func extractToolCalls(items []OutputItem) *globals.ToolCalls {
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

func buildResponseChunk(form *ResponseResponse) *globals.Chunk {
	if form == nil {
		return &globals.Chunk{}
	}

	content := extractOutputText(form)
	toolCalls := extractToolCalls(form.Output)
	if content == "" && toolCalls == nil {
		return &globals.Chunk{}
	}

	return &globals.Chunk{
		Content:  content,
		ToolCall: toolCalls,
	}
}

func parseResponse(data string) (*ResponseResponse, error) {
	form := utils.UnmarshalForm[ResponseResponse](data)
	if form == nil {
		return nil, errors.New("cannot parse response")
	}

	if form.Error.Message != "" {
		return nil, fmt.Errorf("%s", form.Error.Message)
	}

	return form, nil
}

func parseStreamEvent(data string) (*ResponseStreamEvent, error) {
	form := utils.UnmarshalForm[ResponseStreamEvent](data)
	if form == nil {
		return nil, errors.New("cannot parse stream event")
	}

	if form.Error.Message != "" {
		return nil, fmt.Errorf("%s", form.Error.Message)
	}

	return form, nil
}

func emitReasoningSummary(delta string, started *bool) *globals.Chunk {
	if strings.TrimSpace(delta) == "" {
		return nil
	}

	if !*started {
		*started = true
		return &globals.Chunk{
			Content: fmt.Sprintf("<think>\n%s", delta),
		}
	}

	return &globals.Chunk{
		Content: delta,
	}
}

func emitOutputText(delta string, reasoningStarted *bool, reasoningClosed *bool) *globals.Chunk {
	content := delta

	if *reasoningStarted && !*reasoningClosed {
		*reasoningClosed = true
		if content != "" {
			content = fmt.Sprintf("\n</think>\n\n%s", content)
		} else {
			content = "\n</think>\n\n"
		}
	}

	if content == "" {
		return nil
	}

	return &globals.Chunk{
		Content: content,
	}
}

func emitFunctionCallEvent(item *OutputItem) *globals.Chunk {
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

func (c *ChatInstance) CreateStreamChatRequest(props *adaptercommon.ChatProps, callback globals.Hook) error {
	reasoningStarted := false
	reasoningClosed := false
	ticks := 0
	body := c.GetChatBody(props, true)

	err := utils.EventScanner(&utils.EventScannerProps{
		Method:  "POST",
		Uri:     c.GetChatEndpoint(),
		Headers: c.GetHeader(),
		Body:    body,
		Callback: func(data string) error {
			event, parseErr := parseStreamEvent(data)
			if parseErr != nil {
				return parseErr
			}

			var chunk *globals.Chunk
			switch event.Type {
			case "response.reasoning_summary_text.delta":
				chunk = emitReasoningSummary(event.Delta, &reasoningStarted)
			case "response.output_text.delta":
				chunk = emitOutputText(event.Delta, &reasoningStarted, &reasoningClosed)
			case "response.output_item.done":
				chunk = emitFunctionCallEvent(event.Item)
			default:
				return nil
			}

			if chunk == nil {
				return nil
			}

			ticks += 1
			return callback(chunk)
		},
	}, props.Proxy)

	if err != nil {
		if err.Body != "" {
			if form := utils.UnmarshalForm[ResponseResponse](err.Body); form != nil && form.Error.Message != "" {
				return fmt.Errorf("xai responses error: %s", form.Error.Message)
			}

			return fmt.Errorf("xai responses error: %s", strings.TrimSpace(err.Body))
		}

		return fmt.Errorf("xai responses error: %s", err.Error)
	}

	if reasoningStarted && !reasoningClosed {
		if closeErr := callback(&globals.Chunk{Content: "\n</think>\n\n"}); closeErr != nil {
			return closeErr
		}
		ticks += 1
	}

	if ticks == 0 {
		return errors.New("xai responses error: empty response")
	}

	return nil
}
