package xai

import (
	adaptercommon "chat/adapter/common"
	compat "chat/adapter/responsescompat"
	"chat/globals"
	"chat/utils"
	"errors"
	"fmt"
	"strings"
)

func (c *ChatInstance) GetChatEndpoint() string {
	return fmt.Sprintf("%s/v1/responses", c.GetEndpoint())
}

func formatInputMessage(props *adaptercommon.ChatProps, message globals.Message) *InputMessage {
	text := compat.MessageText(message)

	if compat.NormalizeRole(message.Role) == globals.User {
		content, urls := utils.ExtractImages(text, true)
		items := []InputMessageContent{
			{
				Type: "input_text",
				Text: &content,
			},
		}

		for _, rawURL := range urls {
			url, err := utils.NormalizeInternalAttachmentImageURL(rawURL)
			if err != nil {
				globals.Warn(fmt.Sprintf("[xai] cannot normalize attachment image: %s", err.Error()))
				url = rawURL
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
		Role: compat.NormalizeRole(message.Role),
		Content: []InputMessageContent{
			{
				Type: "input_text",
				Text: &text,
			},
		},
	}
}

func formatMessages(props *adaptercommon.ChatProps) ([]interface{}, bool) {
	input := make([]interface{}, 0, len(props.Message))
	hasImages := false

	for _, message := range props.Message {
		if message.Role == globals.Tool {
			if output := compat.FunctionCallOutput(message); output != nil {
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

			input = append(input, compat.ReplayFunctionCalls(message)...)
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

func (c *ChatInstance) GetChatBody(props *adaptercommon.ChatProps, stream bool) ResponseRequest {
	input, hasImages := formatMessages(props)
	var store *bool
	if hasImages {
		store = utils.ToPtr(false)
	}
	tools := getResponseTools(props)

	return ResponseRequest{
		Model:           props.Model,
		Input:           input,
		MaxOutputTokens: props.MaxTokens,
		Temperature:     props.Temperature,
		TopP:            props.TopP,
		Tools:           tools,
		ToolChoice:      props.ToolChoice,
		ResponseFormat:  props.ResponseFormat,
		Store:           store,
		Stream:          stream,
	}
}

func buildResponseChunk(form *ResponseResponse) *globals.Chunk {
	if form == nil {
		return &globals.Chunk{}
	}

	return compat.BuildResponseChunk(form.Output)
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
	return compat.EmitFunctionCallEvent(item)
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
