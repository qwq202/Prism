package openairesponses

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
	imageDetail := "high"
	role := normalizeRole(message.Role)

	if role == globals.User {
		content, urls := utils.ExtractImages(text, true)
		items := []InputMessageContent{
			{
				Type: "input_text",
				Text: &content,
			},
		}

		for _, rawURL := range urls {
			url := rawURL
			if props.Buffer != nil {
				if obj, err := utils.NewImage(url); err == nil {
					props.Buffer.AddImage(obj)
				}
			}

			items = append(items, InputMessageContent{
				Type:     "input_image",
				ImageURL: &url,
				Detail:   &imageDetail,
			})
		}

		return &InputMessage{
			Role:    globals.User,
			Content: items,
		}
	}

	contentType := "input_text"
	if role == globals.Assistant {
		contentType = "output_text"
	}

	return &InputMessage{
		Role: role,
		Content: []InputMessageContent{
			{
				Type: contentType,
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

func formatMessages(props *adaptercommon.ChatProps) ([]interface{}, *string) {
	input := make([]interface{}, 0, len(props.Message))
	instructions := make([]string, 0)

	for _, message := range props.Message {
		if message.Role == globals.System {
			text := strings.TrimSpace(getMessageText(message))
			if text != "" {
				instructions = append(instructions, text)
			}
			continue
		}

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

		input = append(input, *formatted)
	}

	var instructionText *string
	if len(instructions) > 0 {
		joined := strings.Join(instructions, "\n\n")
		instructionText = &joined
	}

	return input, instructionText
}

func getResponseTools(props *adaptercommon.ChatProps) []ResponseTool {
	tools := make([]ResponseTool, 0)
	if props == nil {
		return tools
	}

	if props.EnableWebSearch && globals.IsOpenAIResponsesNativeWebModel(props.Model) {
		tools = append(tools, ResponseTool{
			Type: "web_search",
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

func getResponseTextConfig(props *adaptercommon.ChatProps) interface{} {
	if props == nil || props.ResponseFormat == nil {
		return nil
	}

	return map[string]interface{}{
		"format": props.ResponseFormat,
	}
}

func getReasoningEffort(thinking interface{}) string {
	if thinking == nil {
		return ""
	}

	switch value := thinking.(type) {
	case map[string]interface{}:
		if effort, ok := value["effort"].(string); ok {
			return strings.TrimSpace(strings.ToLower(effort))
		}
	case map[string]string:
		return strings.TrimSpace(strings.ToLower(value["effort"]))
	}

	return ""
}

func getResponseSamplingConfig(
	model string,
	thinking interface{},
	temperature *float32,
	topP *float32,
) (*float32, *float32) {
	effort := getReasoningEffort(thinking)
	normalized := strings.TrimSpace(strings.ToLower(model))

	switch {
	case strings.HasPrefix(normalized, "gpt-5.4-pro"):
		return nil, nil
	case normalized == "gpt-5.2-pro" || strings.HasPrefix(normalized, "gpt-5.2-pro-"):
		return nil, nil
	case globals.IsOpenAIGPT54Model(model), strings.HasPrefix(normalized, "gpt-5.2"):
		if effort != "" && effort != "none" {
			return nil, nil
		}
	case normalized == "gpt-5" || normalized == "gpt-5-pro" || strings.HasPrefix(normalized, "gpt-5-pro-"):
		return nil, nil
	}

	return temperature, topP
}

func (c *ChatInstance) GetChatBody(props *adaptercommon.ChatProps, stream bool) ResponseRequest {
	input, instructions := formatMessages(props)
	tools := getResponseTools(props)
	temperature, topP := getResponseSamplingConfig(
		props.Model,
		props.Thinking,
		props.Temperature,
		props.TopP,
	)

	return ResponseRequest{
		Model:              props.Model,
		Instructions:       instructions,
		Input:              input,
		MaxOutputTokens:    props.MaxTokens,
		Temperature:        temperature,
		TopP:               topP,
		Tools:              tools,
		ToolChoice:         props.ToolChoice,
		ParallelToolCalls:  props.ParallelToolCalls,
		Text:               getResponseTextConfig(props),
		Reasoning:          props.Thinking,
		Include:            props.ResponseInclude,
		PreviousResponseID: props.PreviousResponseID,
		Store:              props.ResponseStore,
		Stream:             stream,
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

func emitOutputText(delta string) *globals.Chunk {
	if delta == "" {
		return nil
	}

	return &globals.Chunk{
		Content: delta,
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
			case "response.output_text.delta":
				chunk = emitOutputText(event.Delta)
			case "response.output_item.done", "response.function_call_arguments.done":
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
				return fmt.Errorf("openai responses error: %s", form.Error.Message)
			}

			return fmt.Errorf("openai responses error: %s", strings.TrimSpace(err.Body))
		}

		return fmt.Errorf("openai responses error: %s", err.Error)
	}

	if ticks == 0 {
		return errors.New("openai responses error: empty response")
	}

	return nil
}
