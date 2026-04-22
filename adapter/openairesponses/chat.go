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
	if props == nil || props.Tools == nil {
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

func (c *ChatInstance) GetChatBody(props *adaptercommon.ChatProps, stream bool) ResponseRequest {
	input, instructions := formatMessages(props)
	tools := getResponseTools(props)

	return ResponseRequest{
		Model:           props.Model,
		Instructions:    instructions,
		Input:           input,
		MaxOutputTokens: props.MaxTokens,
		Temperature:     props.Temperature,
		TopP:            props.TopP,
		Tools:           tools,
		ToolChoice:      props.ToolChoice,
		ResponseFormat:  props.ResponseFormat,
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

func (c *ChatInstance) CreateStreamChatRequest(props *adaptercommon.ChatProps, callback globals.Hook) error {
	body := c.GetChatBody(props, false)
	raw, err := utils.PostRaw(
		c.GetChatEndpoint(),
		c.GetHeader(),
		body,
		props.Proxy,
	)
	if err != nil {
		return fmt.Errorf("openai responses error: %s", err.Error())
	}

	form, parseErr := parseResponse(raw)
	if parseErr != nil {
		return fmt.Errorf("openai responses error: %s", parseErr.Error())
	}

	chunk := buildResponseChunk(form)
	if chunk.IsEmpty() {
		return errors.New("openai responses error: empty response")
	}

	return callback(chunk)
}
