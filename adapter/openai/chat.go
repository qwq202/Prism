package openai

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

func (c *ChatInstance) GetChatEndpoint(props *adaptercommon.ChatProps) string {
	if props.Model == globals.GPT3TurboInstruct {
		return c.GetAPIEndpoint("completions")
	}
	return c.GetAPIEndpoint("chat/completions")
}

func (c *ChatInstance) GetCompletionPrompt(messages []globals.Message) string {
	result := ""
	for _, message := range messages {
		result += fmt.Sprintf("%s: %s\n", message.Role, message.Content)
	}
	return result
}

func (c *ChatInstance) GetLatestPrompt(props *adaptercommon.ChatProps) string {
	if len(props.Message) == 0 {
		return ""
	}

	return props.Message[len(props.Message)-1].Content
}

func (c *ChatInstance) GetChatBody(props *adaptercommon.ChatProps, stream bool) interface{} {
	if props.Model == globals.GPT3TurboInstruct {
		// for completions
		return CompletionRequest{
			Model:    props.Model,
			Prompt:   c.GetCompletionPrompt(props.Message),
			MaxToken: props.MaxTokens,
			Stream:   stream,
		}
	}

	messages := formatMessages(props)

	// o1, o3, gpt-5 compatibility
	isNewModel := len(props.Model) >= 2 && (props.Model[:2] == "o1" || props.Model[:2] == "o3") || strings.HasPrefix(props.Model, "gpt-5")

	var temperature *float32
	if isNewModel {
		temp := float32(1.0)
		temperature = &temp
	} else {
		temperature = props.Temperature
	}

	request := ChatRequest{
		Model:                props.Model,
		Messages:             messages,
		Stream:               stream,
		StreamOptions:        getStreamOptions(props, stream),
		PresencePenalty:      props.PresencePenalty,
		FrequencyPenalty:     props.FrequencyPenalty,
		Temperature:          temperature,
		TopP:                 props.TopP,
		PromptCacheKey:       normalizedStringPtr(props.PromptCacheKey),
		PromptCacheRetention: normalizedStringPtr(props.PromptCacheRetention),
		SessionID:            getOpenRouterSessionID(c, props),
		Tools:                props.Tools,
		ToolChoice:           props.ToolChoice,
	}

	if isNewModel {
		request.MaxCompletionTokens = props.MaxTokens
	} else {
		request.MaxToken = props.MaxTokens
	}
	return request
}

func normalizedStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	text := strings.TrimSpace(*value)
	if text == "" {
		return nil
	}
	return &text
}

func getOpenRouterSessionID(c *ChatInstance, props *adaptercommon.ChatProps) *string {
	if c == nil || props == nil || c.GetErrorPrefix() != "openrouter" {
		return nil
	}
	return normalizedStringPtr(props.SessionID)
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

// CreateChatRequest is the native http request body for openai
func (c *ChatInstance) CreateChatRequest(props *adaptercommon.ChatProps) (string, error) {
	if globals.IsOpenAIDalleModel(props.Model) {
		return c.CreateImage(props)
	}

	res, err := utils.Post(
		c.GetChatEndpoint(props),
		c.GetHeader(),
		c.GetChatBody(props, false),
		props.Proxy,
	)

	if err != nil || res == nil {
		return "", fmt.Errorf("%s error: %s", c.GetErrorPrefix(), err.Error())
	}

	data := utils.MapToStruct[ChatResponse](res)
	if data == nil {
		return "", fmt.Errorf("%s error: cannot parse response", c.GetErrorPrefix())
	} else if hasResponseError(data.Error) {
		return "", errors.New(formatResponseError(c.GetErrorPrefix(), data.Error))
	}
	if len(data.Choices) == 0 {
		return "", fmt.Errorf("%s error: no choices", c.GetErrorPrefix())
	}
	return formatReasoningContent(
		getReasoningText(data.Choices[0].Message),
		data.Choices[0].Message.Content,
	), nil
}

func hideRequestId(message string) string {
	// xxx (request id: 2024020311120561344953f0xfh0TX)

	exp := regexp.MustCompile(`\(request id: [a-zA-Z0-9]+\)`)
	return exp.ReplaceAllString(message, "")
}

// CreateStreamChatRequest is the stream response body for openai
func (c *ChatInstance) CreateStreamChatRequest(props *adaptercommon.ChatProps, callback globals.Hook) error {
	if globals.IsOpenAIDalleModel(props.Model) {
		if url, err := c.CreateImage(props); err != nil {
			return err
		} else {
			return callback(&globals.Chunk{
				Content: url,
			})
		}
	}

	isCompletionType := props.Model == globals.GPT3TurboInstruct
	c.isFirstReasoning = true
	c.isReasonOver = false

	ticks := 0
	err := utils.EventScanner(&utils.EventScannerProps{
		Method:  "POST",
		Uri:     c.GetChatEndpoint(props),
		Headers: c.GetHeader(),
		Body:    c.GetChatBody(props, true),
		Callback: func(data string) error {
			ticks += 1

			partial, err := c.ProcessLine(data, isCompletionType)
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

			msg := fmt.Sprintf("%s (type: %s)", form.Error.Message, form.Error.Type)
			return errors.New(hideRequestId(msg))
		}
		return err.Error
	}

	if ticks == 0 {
		return errors.New("no response")
	}

	return nil
}
