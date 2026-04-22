package deepseek

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"errors"
	"fmt"
	"strings"
)

type ChatInstance struct {
	Endpoint         string
	ApiKey           string
	isFirstReasoning bool
	isReasonOver     bool
}

func (c *ChatInstance) GetEndpoint() string {
	return c.Endpoint
}

func (c *ChatInstance) GetApiKey() string {
	return c.ApiKey
}

func (c *ChatInstance) GetHeader() map[string]string {
	return map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", c.GetApiKey()),
	}
}

func NewChatInstance(endpoint, apiKey string) *ChatInstance {
	return &ChatInstance{
		Endpoint:         endpoint,
		ApiKey:           apiKey,
		isFirstReasoning: true,
	}
}

func NewChatInstanceFromConfig(conf globals.ChannelConfig) adaptercommon.Factory {
	return NewChatInstance(
		conf.GetEndpoint(),
		conf.GetRandomSecret(),
	)
}

func (c *ChatInstance) GetChatEndpoint() string {
	return fmt.Sprintf("%s/chat/completions", c.GetEndpoint())
}

func (c *ChatInstance) GetChatBody(props *adaptercommon.ChatProps, stream bool) interface{} {
	messages := props.Message
	// Keep legacy compatibility for older deployments that reject an initial assistant role.
	if len(messages) > 0 && messages[0].Role == globals.Assistant {
		messages = make([]globals.Message, len(props.Message))
		copy(messages, props.Message)
		messages[0].Role = globals.User
	}

	temperature := props.Temperature
	topP := props.TopP
	presencePenalty := props.PresencePenalty
	frequencyPenalty := props.FrequencyPenalty
	logprobs := props.Logprobs
	topLogprobs := props.TopLogprobs

	if isReasoningModel(props.Model) {
		temperature = nil
		topP = nil
		presencePenalty = nil
		frequencyPenalty = nil
		logprobs = nil
		topLogprobs = nil
	}

	var streamOptions interface{}
	if stream {
		streamOptions = props.StreamOptions
	}

	return ChatRequest{
		Model:            props.Model,
		Messages:         messages,
		MaxTokens:        props.MaxTokens,
		Stream:           stream,
		Temperature:      temperature,
		TopP:             topP,
		PresencePenalty:  presencePenalty,
		FrequencyPenalty: frequencyPenalty,
		Stop:             props.Stop,
		ResponseFormat:   props.ResponseFormat,
		Thinking:         props.Thinking,
		StreamOptions:    streamOptions,
		Logprobs:         logprobs,
		TopLogprobs:      topLogprobs,
		Tools:            props.Tools,
		ToolChoice:       props.ToolChoice,
	}
}

func isReasoningModel(model string) bool {
	return strings.TrimSpace(model) == globals.DeepseekR1
}

func processChatResponse(data string) *ChatResponse {
	if form := utils.UnmarshalForm[ChatResponse](data); form != nil {
		return form
	}
	return nil
}

func processChatStreamResponse(data string) *ChatStreamResponse {
	if form := utils.UnmarshalForm[ChatStreamResponse](data); form != nil {
		return form
	}
	return nil
}

func processChatErrorResponse(data string) *ChatStreamErrorResponse {
	if form := utils.UnmarshalForm[ChatStreamErrorResponse](data); form != nil {
		return form
	}
	return nil
}

func formatReasoning(reasoning *string, content string) string {
	if reasoning == nil || *reasoning == "" {
		return content
	}

	if content == "" {
		return fmt.Sprintf("<think>\n%s\n</think>", *reasoning)
	}

	return fmt.Sprintf("<think>\n%s\n</think>\n\n%s", *reasoning, content)
}

func (c *ChatInstance) getChoices(form *ChatStreamResponse) *globals.Chunk {
	if len(form.Choices) == 0 {
		return &globals.Chunk{Content: ""}
	}

	choice := form.Choices[0].Delta
	reasoning := choice.ReasoningContent

	if c.isFirstReasoning == false && !c.isReasonOver && reasoning == nil {
		c.isReasonOver = true
		if choice.Content != "" {
			return &globals.Chunk{
				Content:          fmt.Sprintf("\n</think>\n\n%s", choice.Content),
				ToolCall:         choice.ToolCalls,
				FunctionCall:     choice.FunctionCall,
				ReasoningContent: nil,
			}
		}

		return &globals.Chunk{
			Content:          "\n</think>\n\n",
			ToolCall:         choice.ToolCalls,
			FunctionCall:     choice.FunctionCall,
			ReasoningContent: nil,
		}
	}

	content := choice.Content
	if reasoning != nil {
		if c.isFirstReasoning {
			c.isFirstReasoning = false
			content = fmt.Sprintf("<think>\n%s", *reasoning)
		} else {
			content = *reasoning
		}
	}

	return &globals.Chunk{
		Content:          content,
		ToolCall:         choice.ToolCalls,
		FunctionCall:     choice.FunctionCall,
		ReasoningContent: reasoning,
	}
}

func (c *ChatInstance) ProcessLine(data string) (*globals.Chunk, error) {
	if form := processChatStreamResponse(data); form != nil {
		return c.getChoices(form), nil
	}

	if form := processChatErrorResponse(data); form != nil {
		if form.Error.Message != "" {
			return &globals.Chunk{Content: ""}, errors.New(fmt.Sprintf("deepseek error: %s", form.Error.Message))
		}
	}

	return &globals.Chunk{Content: ""}, nil
}

func (c *ChatInstance) CreateChatRequest(props *adaptercommon.ChatProps) (string, error) {
	res, err := utils.Post(
		c.GetChatEndpoint(),
		c.GetHeader(),
		c.GetChatBody(props, false),
		props.Proxy,
	)

	if err != nil || res == nil {
		return "", fmt.Errorf("deepseek error: %s", err.Error())
	}

	data := utils.MapToStruct[ChatResponse](res)
	if data == nil {
		return "", fmt.Errorf("deepseek error: cannot parse response")
	}

	if len(data.Choices) == 0 {
		return "", fmt.Errorf("deepseek error: no choices")
	}

	message := data.Choices[0].Message
	return formatReasoning(message.ReasoningContent, message.Content), nil
}

func (c *ChatInstance) CreateStreamChatRequest(props *adaptercommon.ChatProps, callback globals.Hook) error {
	c.isFirstReasoning = true
	c.isReasonOver = false
	err := utils.EventScanner(&utils.EventScannerProps{
		Method:  "POST",
		Uri:     c.GetChatEndpoint(),
		Headers: c.GetHeader(),
		Body:    c.GetChatBody(props, true),
		Callback: func(data string) error {
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
			return errors.New(fmt.Sprintf("deepseek error: %s (type: %s)", form.Error.Message, form.Error.Type))
		}
		return err.Error
	}

	return nil
}
