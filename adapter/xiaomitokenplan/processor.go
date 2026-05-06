package xiaomitokenplan

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"errors"
	"fmt"
)

func formatMessages(props *adaptercommon.ChatProps) interface{} {
	return utils.Each[globals.Message, Message](props.Message, func(message globals.Message) Message {
		content := interface{}(message.Content)

		if message.Role == globals.User && globals.IsVisionModel(props.Model) {
			text, urls := utils.ExtractImages(message.Content, true)
			parts := make(MessageContents, 0, len(urls)+1)
			parts = append(parts, MessageContent{
				Type: "text",
				Text: &text,
			})

			for _, url := range urls {
				obj, err := utils.NewImage(url)
				if props.Buffer != nil {
					props.Buffer.AddImage(obj)
				}
				if err != nil {
					globals.Info(fmt.Sprintf("cannot process image: %s (source: %s)", err.Error(), utils.Extract(url, 24, "...")))
				}

				parts = append(parts, MessageContent{
					Type: "image_url",
					ImageURL: &ImageURL{
						URL: url,
					},
				})
			}

			content = parts
		}

		return Message{
			Role:             message.Role,
			Content:          content,
			Name:             message.Name,
			FunctionCall:     message.FunctionCall,
			ToolCallID:       message.ToolCallId,
			ToolCalls:        message.ToolCalls,
			ReasoningContent: message.ReasoningContent,
		}
	})
}

func processChatResponse(data string) *ChatStreamResponse {
	return utils.UnmarshalForm[ChatStreamResponse](data)
}

func processChatErrorResponse(data string) *ChatStreamErrorResponse {
	return utils.UnmarshalForm[ChatStreamErrorResponse](data)
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
	reasoning := choice.ReasoningContent

	if !c.isFirstReasoning && !c.isReasonOver && reasoning == nil {
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
		Content:          content,
		ToolCall:         choice.ToolCalls,
		FunctionCall:     choice.FunctionCall,
		ReasoningContent: reasoning,
	}
}

func (c *ChatInstance) ProcessLine(data string) (*globals.Chunk, error) {
	if form := processChatResponse(data); form != nil {
		return c.getChoices(form), nil
	}

	if form := processChatErrorResponse(data); form != nil {
		return &globals.Chunk{Content: ""}, errors.New(fmt.Sprintf("xiaomi token plan error: %s (type: %s)", form.Error.Message, form.Error.Type))
	}

	globals.Warn(fmt.Sprintf("xiaomi token plan error: cannot parse chat completion response: %s", data))
	return &globals.Chunk{Content: ""}, errors.New("parser error: cannot parse chat completion response")
}

func collectResponse(form ChatStreamResponse) (*globals.Chunk, error) {
	if len(form.Choices) == 0 {
		return nil, errors.New("xiaomi token plan error: no choices")
	}

	message := form.Choices[0].Delta
	return &globals.Chunk{
		Content:          formatReasoningContent(message.ReasoningContent, message.Content),
		ToolCall:         message.ToolCalls,
		FunctionCall:     message.FunctionCall,
		ReasoningContent: message.ReasoningContent,
	}, nil
}
