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
	Buffer               *utils.Buffer          `json:"-"`
}

const currentDateTimePromptPrefix = "Current date and time reference:"

func buildCurrentDateTimePrompt() string {
	now := time.Now()
	return fmt.Sprintf(
		"%s %s (%s). Treat this as the current local time unless the user specifies a different timezone.",
		currentDateTimePromptPrefix,
		now.Format("2006-01-02 15:04:05"),
		now.Location().String(),
	)
}

func injectCurrentDateTime(messages []globals.Message) []globals.Message {
	if len(messages) == 0 {
		return []globals.Message{
			{
				Role:    globals.System,
				Content: buildCurrentDateTimePrompt(),
			},
		}
	}

	cloned := utils.DeepCopy[[]globals.Message](messages)
	prompt := buildCurrentDateTimePrompt()

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

func (c *ChatProps) SetupBuffer(buf *utils.Buffer) {
	c.Message = injectCurrentDateTime(c.Message)
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
