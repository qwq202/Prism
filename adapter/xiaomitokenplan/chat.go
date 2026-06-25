package xiaomitokenplan

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"errors"
	"fmt"
	"strings"
)

func (c *ChatInstance) GetChatEndpoint() string {
	endpoint := c.GetEndpoint()
	if strings.HasSuffix(endpoint, "/v1") {
		return fmt.Sprintf("%s/chat/completions", endpoint)
	}

	return fmt.Sprintf("%s/v1/chat/completions", endpoint)
}

func (c *ChatInstance) GetChatBody(props *adaptercommon.ChatProps, stream bool) ChatRequest {
	return ChatRequest{
		Model:               props.Model,
		Messages:            formatMessages(props),
		MaxCompletionTokens: props.MaxTokens,
		Stream:              stream,
		StreamOptions:       getStreamOptions(props, stream),
		PresencePenalty:     props.PresencePenalty,
		FrequencyPenalty:    props.FrequencyPenalty,
		Temperature:         props.Temperature,
		TopP:                props.TopP,
		Stop:                props.Stop,
		ResponseFormat:      props.ResponseFormat,
		Thinking:            props.Thinking,
		Tools:               props.Tools,
		ToolChoice:          props.ToolChoice,
	}
}

func getStreamOptions(props *adaptercommon.ChatProps, stream bool) interface{} {
	if !stream {
		return nil
	}
	if props.StreamOptions != nil {
		return props.StreamOptions
	}
	return map[string]bool{"include_usage": true}
}

func hideRequestID(message string) string {
	message = strings.ReplaceAll(message, "request id", "request_id")
	return message
}

func (c *ChatInstance) resetStreamState() {
	c.isFirstReasoning = true
	c.isReasonOver = false
	c.toolCalls = make(map[int]globals.ToolCall)
}

func (c *ChatInstance) CreateStreamChatRequest(props *adaptercommon.ChatProps, callback globals.Hook) error {
	c.resetStreamState()

	ticks := 0
	err := utils.EventScanner(&utils.EventScannerProps{
		Method:  "POST",
		Uri:     c.GetChatEndpoint(),
		Headers: c.GetHeader(),
		Body:    c.GetChatBody(props, true),
		Callback: func(data string) error {
			ticks += 1

			partial, err := c.ProcessLine(data)
			if err != nil {
				return err
			}
			return callback(partial)
		},
	}, props.Proxy)

	if err != nil {
		if form := processChatErrorResponse(err.Body); form != nil {
			if form.Error.Type == "" && form.Error.Message == "" {
				return errors.New(utils.ToMarkdownCode("json", err.Body))
			}

			return errors.New(hideRequestID(fmt.Sprintf("%s (type: %s)", form.Error.Message, form.Error.Type)))
		}
		return err.Error
	}

	if ticks == 0 {
		return errors.New("no response")
	}

	return nil
}
