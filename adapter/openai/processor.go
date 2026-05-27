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

func formatMessages(props *adaptercommon.ChatProps) interface{} {
	if globals.IsVisionModel(props.Model) {
		return utils.Each[globals.Message, Message](props.Message, func(message globals.Message) Message {
			if message.Role == globals.User {
				content, urls := utils.ExtractImages(message.Content, true)
				images := utils.EachNotNil[string, MessageContent](urls, func(url string) *MessageContent {
					obj, err := utils.NewImage(url)
					props.Buffer.AddImage(obj)
					if err != nil {
						globals.Info(fmt.Sprintf("cannot process image: %s (source: %s)", err.Error(), utils.Extract(url, 24, "...")))
					}

					return &MessageContent{
						Type: "image_url",
						ImageUrl: &ImageUrl{
							Url: url,
						},
					}
				})

				return Message{
					Role: message.Role,
					Content: utils.Prepend(images, MessageContent{
						Type: "text",
						Text: &content,
					}),
					Name:         message.Name,
					FunctionCall: message.FunctionCall,
					ToolCalls:    message.ToolCalls,
					ToolCallId:   message.ToolCallId,
				}
			}

			return Message{
				Role: message.Role,
				Content: MessageContents{
					MessageContent{
						Type: "text",
						Text: &message.Content,
					},
				},
				Name:         message.Name,
				FunctionCall: message.FunctionCall,
				ToolCalls:    message.ToolCalls,
				ToolCallId:   message.ToolCallId,
			}
		})
	}

	return props.Message
}

func processChatResponse(data string) *ChatStreamResponse {
	return utils.UnmarshalForm[ChatStreamResponse](data)
}

func processCompletionResponse(data string) *CompletionResponse {
	return utils.UnmarshalForm[CompletionResponse](data)
}

func processChatErrorResponse(data string) *ChatStreamErrorResponse {
	return utils.UnmarshalForm[ChatStreamErrorResponse](data)
}

func hasResponseError(responseError ResponseError) bool {
	return strings.TrimSpace(responseError.Message) != "" ||
		strings.TrimSpace(responseError.Type) != "" ||
		responseError.Code != nil
}

func formatResponseError(prefix string, responseError ResponseError) string {
	message := strings.TrimSpace(responseError.Message)
	if message == "" {
		message = "request failed"
	}
	if responseType := strings.TrimSpace(responseError.Type); responseType != "" {
		message = fmt.Sprintf("%s (type: %s)", message, responseType)
	}
	return fmt.Sprintf("%s error: %s", prefix, message)
}

func getReasoningText(message ResponseMessage) *string {
	if message.ReasoningContent != nil && *message.ReasoningContent != "" {
		return message.ReasoningContent
	}

	if message.Reasoning != nil && *message.Reasoning != "" {
		return message.Reasoning
	}

	return nil
}

func formatReasoningContent(reasoning *string, content string) string {
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
	reasoning := getReasoningText(choice)

	if c.isFirstReasoning == false && !c.isReasonOver && reasoning == nil {
		c.isReasonOver = true
		if choice.Content != "" {
			return &globals.Chunk{
				Content:      fmt.Sprintf("\n</think>\n\n%s", choice.Content),
				ToolCall:     choice.ToolCalls,
				FunctionCall: choice.FunctionCall,
			}
		}

		return &globals.Chunk{
			Content:      "\n</think>\n\n",
			ToolCall:     choice.ToolCalls,
			FunctionCall: choice.FunctionCall,
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
		Content:      content,
		ToolCall:     choice.ToolCalls,
		FunctionCall: choice.FunctionCall,
	}
}

func getCompletionChoices(form *CompletionResponse) string {
	if len(form.Choices) == 0 {
		return ""
	}

	return form.Choices[0].Text
}

func getRobustnessResult(chunk string) string {
	exp := `\"content\":\"(.*?)\"`
	compile, err := regexp.Compile(exp)
	if err != nil {
		return ""
	}

	matches := compile.FindStringSubmatch(chunk)
	if len(matches) > 1 {
		return utils.ProcessRobustnessChar(matches[1])
	} else {
		return ""
	}
}

func (c *ChatInstance) ProcessLine(data string, isCompletionType bool) (*globals.Chunk, error) {
	errorPrefix := c.GetErrorPrefix()

	if isCompletionType {
		// openai legacy support
		if completion := processCompletionResponse(data); completion != nil {
			return &globals.Chunk{
				Content: getCompletionChoices(completion),
			}, nil
		}

		globals.Warn(fmt.Sprintf("%s error: cannot parse completion response: %s", errorPrefix, data))
		return &globals.Chunk{Content: ""}, errors.New("parser error: cannot parse completion response")
	}

	if form := processChatResponse(data); form != nil {
		if hasResponseError(form.Error) {
			return &globals.Chunk{Content: ""}, errors.New(formatResponseError(errorPrefix, form.Error))
		}
		return c.getChoices(form), nil
	}

	if form := processChatErrorResponse(data); form != nil {
		return &globals.Chunk{Content: ""}, errors.New(formatResponseError(errorPrefix, form.Error))
	}

	globals.Warn(fmt.Sprintf("%s error: cannot parse chat completion response: %s", errorPrefix, data))
	return &globals.Chunk{Content: ""}, errors.New("parser error: cannot parse chat completion response")
}
