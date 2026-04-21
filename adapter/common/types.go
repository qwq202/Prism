package adaptercommon

import (
	"chat/globals"
	"chat/utils"
	"fmt"
	"strings"
	"time"
)

type RequestProps struct {
	MaxRetries *int                `json:"-"`
	Current    int                 `json:"-"`
	Group      string              `json:"-"`
	Proxy      globals.ProxyConfig `json:"-"`
}

type VideoProps struct {
	RequestProps

	Model         string `json:"model,omitempty"`
	OriginalModel string `json:"-"`

	Prompt         string  `json:"prompt"`
	Seconds        *string `json:"seconds,omitempty"`
	Size           *string `json:"size,omitempty"`
	InputReference *string `json:"input_reference,omitempty"`

	User string `json:"-"`
}

type ChatProps struct {
	RequestProps

	Model         string `json:"model,omitempty"`
	OriginalModel string `json:"-"`

	Message              []globals.Message      `json:"messages,omitempty"`
	CustomInstruction    string                 `json:"custom_instruction,omitempty"`
	MaxTokens            *int                   `json:"max_tokens,omitempty"`
	PresencePenalty      *float32               `json:"presence_penalty,omitempty"`
	FrequencyPenalty     *float32               `json:"frequency_penalty,omitempty"`
	RepetitionPenalty    *float32               `json:"repetition_penalty,omitempty"`
	Temperature          *float32               `json:"temperature,omitempty"`
	TopP                 *float32               `json:"top_p,omitempty"`
	TopK                 *int                   `json:"top_k,omitempty"`
	Tools                *globals.FunctionTools `json:"tools,omitempty"`
	ToolChoice           *interface{}           `json:"tool_choice,omitempty"`
	EnableWeb            bool                   `json:"-"`
	EnableWebSearch      bool                   `json:"-"`
	EnableURLContext     bool                   `json:"-"`
	EnableXSearch        bool                   `json:"-"`
	GeminiThinkingBudget *int                   `json:"-"`
	ChannelType          string                 `json:"-"`
	ClientContext        string                 `json:"-"`
	Buffer               *utils.Buffer          `json:"-"`
}

const currentDateTimePromptPrefix = "Current date and time reference:"
const clientContextPromptPrefix = "Current client device reference:"
const personalizationPromptPrefix = "User personalization preferences:"

func buildCurrentDateTimePrompt(clientContext string) string {
	now := time.Now()
	prompt := fmt.Sprintf(
		"%s %s (%s). Treat this as the current local time unless the user specifies a different timezone.",
		currentDateTimePromptPrefix,
		now.Format("2006-01-02 15:04:05"),
		now.Location().String(),
	)

	if strings.TrimSpace(clientContext) == "" {
		return prompt
	}

	return fmt.Sprintf("%s\n%s %s", prompt, clientContextPromptPrefix, strings.TrimSpace(clientContext))
}

func injectCurrentDateTime(messages []globals.Message, clientContext string) []globals.Message {
	if len(messages) == 0 {
		return []globals.Message{
			{
				Role:    globals.System,
				Content: buildCurrentDateTimePrompt(clientContext),
			},
		}
	}

	cloned := utils.DeepCopy[[]globals.Message](messages)
	prompt := buildCurrentDateTimePrompt(clientContext)

	for i := range cloned {
		if cloned[i].Role != globals.System {
			continue
		}

		content := strings.TrimSpace(cloned[i].Content)
		if strings.HasPrefix(content, currentDateTimePromptPrefix) {
			return cloned
		}

		if content == "" {
			cloned[i].Content = prompt
		} else {
			cloned[i].Content = fmt.Sprintf("%s\n\n%s", prompt, cloned[i].Content)
		}
		return cloned
	}

	return append([]globals.Message{
		{
			Role:    globals.System,
			Content: prompt,
		},
	}, cloned...)
}

func injectPersonalization(messages []globals.Message, customInstruction string) []globals.Message {
	customInstruction = strings.TrimSpace(customInstruction)
	if customInstruction == "" {
		return messages
	}

	cloned := utils.DeepCopy[[]globals.Message](messages)
	prompt := fmt.Sprintf("%s\n%s", personalizationPromptPrefix, customInstruction)

	for i := range cloned {
		if cloned[i].Role != globals.System {
			continue
		}

		content := strings.TrimSpace(cloned[i].Content)
		if strings.Contains(content, personalizationPromptPrefix) {
			return cloned
		}

		if content == "" {
			cloned[i].Content = prompt
		} else {
			cloned[i].Content = fmt.Sprintf("%s\n\n%s", content, prompt)
		}
		return cloned
	}

	return append([]globals.Message{
		{
			Role:    globals.System,
			Content: prompt,
		},
	}, cloned...)
}

func (c *ChatProps) SetupBuffer(buf *utils.Buffer) {
	c.Message = injectCurrentDateTime(c.Message, c.ClientContext)
	c.Message = injectPersonalization(c.Message, c.CustomInstruction)
	if buf == nil {
		return
	}
	buf.SetPrompts(c)
	c.Buffer = buf
}

func CreateChatProps(props *ChatProps, buffer *utils.Buffer) *ChatProps {
	props.SetupBuffer(buffer)
	return props
}

func CreateVideoProps(props *VideoProps) *VideoProps {
	return props
}
