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
	case globals.User, globals.Assistant:
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
			if props != nil && props.ChannelType == globals.XAIChannelType && utils.IsInternalAttachmentURL(url) {
				if normalized, err := utils.NormalizeImageToVisionDataURL(url); err == nil {
					url = normalized
				} else {
					globals.Warn(fmt.Sprintf("[openai responses] cannot normalize xai attachment image: %s", err.Error()))
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

func formatMessages(props *adaptercommon.ChatProps) ([]InputMessage, *string, bool) {
	input := make([]InputMessage, 0, len(props.Message))
	instructions := make([]string, 0)
	hasImages := false

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

		for _, item := range formatted.Content {
			if item.Type == "input_image" && item.ImageURL != nil && strings.TrimSpace(*item.ImageURL) != "" {
				hasImages = true
				break
			}
		}

		input = append(input, *formatted)
	}

	var instructionText *string
	if len(instructions) > 0 {
		joined := strings.Join(instructions, "\n\n")
		instructionText = &joined
	}

	return input, instructionText, hasImages
}

func getResponseTools(props *adaptercommon.ChatProps) []ResponseTool {
	tools := make([]ResponseTool, 0)

	if props != nil && props.ChannelType == globals.XAIChannelType {
		if props.EnableWebSearch {
			tools = append(tools, ResponseTool{Type: "web_search"})
		}
		if props.EnableXSearch {
			tools = append(tools, ResponseTool{Type: "x_search"})
		}
	}

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

func (c *ChatInstance) GetChatBody(props *adaptercommon.ChatProps) ResponseRequest {
	input, instructions, hasImages := formatMessages(props)
	var store *bool
	if props != nil && props.ChannelType == globals.XAIChannelType && hasImages {
		store = utils.ToPtr(false)
	}

	return ResponseRequest{
		Model:           props.Model,
		Instructions:    instructions,
		Input:           input,
		MaxOutputTokens: props.MaxTokens,
		Temperature:     props.Temperature,
		TopP:            props.TopP,
		Tools:           getResponseTools(props),
		ToolChoice:      props.ToolChoice,
		Store:           store,
		Stream:          false,
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

func (c *ChatInstance) CreateXAIStreamChatRequest(props *adaptercommon.ChatProps, callback globals.Hook) error {
	reasoningStarted := false
	reasoningClosed := false
	ticks := 0
	body := c.GetChatBody(props)

	err := utils.EventScanner(&utils.EventScannerProps{
		Method:  "POST",
		Uri:     c.GetChatEndpoint(),
		Headers: c.GetHeader(),
		Body: ResponseRequest{
			Model:           body.Model,
			Instructions:    body.Instructions,
			Input:           body.Input,
			MaxOutputTokens: body.MaxOutputTokens,
			Temperature:     body.Temperature,
			TopP:            body.TopP,
			Tools:           body.Tools,
			ToolChoice:      body.ToolChoice,
			Stream:          true,
		},
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

	if reasoningStarted && !reasoningClosed {
		if closeErr := callback(&globals.Chunk{Content: "\n</think>\n\n"}); closeErr != nil {
			return closeErr
		}
		reasoningClosed = true
		ticks += 1
	}

	if ticks == 0 {
		return errors.New("openai responses error: empty response")
	}

	return nil
}

func (c *ChatInstance) CreateStreamChatRequest(props *adaptercommon.ChatProps, callback globals.Hook) error {
	if props != nil && props.ChannelType == globals.XAIChannelType {
		return c.CreateXAIStreamChatRequest(props, callback)
	}

	body := c.GetChatBody(props)
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

	content := extractOutputText(form)
	if content == "" {
		return errors.New("openai responses error: empty response")
	}

	return callback(&globals.Chunk{
		Content: content,
	})
}
