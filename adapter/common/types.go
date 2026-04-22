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
	MemoryPrompt         string                 `json:"memory_prompt,omitempty"`
	RecentChatsPrompt    string                 `json:"recent_chats_prompt,omitempty"`
	MemoryEnabled        bool                   `json:"memory_enabled,omitempty"`
	MemoryHistoryEnabled bool                   `json:"memory_history_enabled,omitempty"`
	MaxTokens            *int                   `json:"max_tokens,omitempty"`
	PresencePenalty      *float32               `json:"presence_penalty,omitempty"`
	FrequencyPenalty     *float32               `json:"frequency_penalty,omitempty"`
	RepetitionPenalty    *float32               `json:"repetition_penalty,omitempty"`
	Temperature          *float32               `json:"temperature,omitempty"`
	TopP                 *float32               `json:"top_p,omitempty"`
	TopK                 *int                   `json:"top_k,omitempty"`
	Stop                 interface{}            `json:"stop,omitempty"`
	ResponseFormat       interface{}            `json:"response_format,omitempty"`
	StreamOptions        interface{}            `json:"stream_options,omitempty"`
	Thinking             interface{}            `json:"thinking,omitempty"`
	Logprobs             *bool                  `json:"logprobs,omitempty"`
	TopLogprobs          *int                   `json:"top_logprobs,omitempty"`
	Tools                *globals.FunctionTools `json:"tools,omitempty"`
	ToolChoice           *interface{}           `json:"tool_choice,omitempty"`
	EnableWeb            bool                   `json:"-"`
	EnableWebSearch      bool                   `json:"-"`
	EnableURLContext     bool                   `json:"-"`
	EnableXSearch        bool                   `json:"-"`
	GeminiThinkingBudget *int                   `json:"-"`
	ChannelType          string                 `json:"-"`
	ClientContext        string                 `json:"-"`
	DisableCache         bool                   `json:"-"`
	Buffer               *utils.Buffer          `json:"-"`
}

const currentDateTimePromptPrefix = "Current date and time reference:"
const clientContextPromptPrefix = "Current client device reference:"
const personalizationPromptPrefix = "User personalization preferences:"
const memoryCapabilityPromptPrefix = "Memory capability state:"
const memoryPromptPrefix = "Saved user memories:"
const recentChatsPromptPrefix = "Recent conversation references:"

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

func buildMemoryCapabilityPrompt(memoryEnabled bool, memoryHistoryEnabled bool) string {
	savedMemoryState := "disabled"
	if memoryEnabled {
		savedMemoryState = "enabled"
	}

	recentChatsState := "disabled"
	if memoryHistoryEnabled {
		recentChatsState = "enabled"
	}

	return fmt.Sprintf(
		"%s\n- Saved user memories: %s.\n- Cross-conversation recent chat references: %s.\n- The messages already present in this chat are always available as the current conversation context.\n- If saved memories are enabled and relevant memories are provided in the saved-memories section, you may use them to answer questions about the user's long-term preferences, profile, or standing instructions.\n- Do not claim that you cannot access saved memories when they are enabled and relevant saved memories are present in the prompt.\n- Do not claim that you can browse or enumerate a separate memory database or hidden backend list unless the user is explicitly shown such a list in the current chat.\n- If saved memories or recent chat references are disabled, clearly say you only see the messages already included in the current chat.",
		memoryCapabilityPromptPrefix,
		savedMemoryState,
		recentChatsState,
	)
}

func injectMemoryCapabilities(messages []globals.Message, memoryEnabled bool, memoryHistoryEnabled bool) []globals.Message {
	cloned := utils.DeepCopy[[]globals.Message](messages)
	prompt := buildMemoryCapabilityPrompt(memoryEnabled, memoryHistoryEnabled)

	for i := range cloned {
		if cloned[i].Role != globals.System {
			continue
		}

		content := strings.TrimSpace(cloned[i].Content)
		if strings.Contains(content, memoryCapabilityPromptPrefix) {
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

func appendPromptSection(content string, prefix string, prompt string) string {
	content = strings.TrimSpace(content)
	prompt = strings.TrimSpace(prompt)

	if prompt == "" {
		return content
	}

	if hasPromptSection(content, prefix) {
		return content
	}

	section := fmt.Sprintf("%s\n%s", prefix, prompt)
	if content == "" {
		return section
	}

	return fmt.Sprintf("%s\n\n%s", content, section)
}

func hasPromptSection(content string, prefix string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == prefix {
			return true
		}
	}

	return false
}

func injectReferencePrompts(messages []globals.Message, memoryPrompt string, recentChatsPrompt string) []globals.Message {
	memoryPrompt = strings.TrimSpace(memoryPrompt)
	recentChatsPrompt = strings.TrimSpace(recentChatsPrompt)
	if memoryPrompt == "" && recentChatsPrompt == "" {
		return messages
	}

	cloned := utils.DeepCopy[[]globals.Message](messages)
	for i := range cloned {
		if cloned[i].Role != globals.System {
			continue
		}

		cloned[i].Content = appendPromptSection(cloned[i].Content, memoryPromptPrefix, memoryPrompt)
		cloned[i].Content = appendPromptSection(cloned[i].Content, recentChatsPromptPrefix, recentChatsPrompt)
		return cloned
	}

	content := appendPromptSection("", memoryPromptPrefix, memoryPrompt)
	content = appendPromptSection(content, recentChatsPromptPrefix, recentChatsPrompt)
	return append([]globals.Message{
		{
			Role:    globals.System,
			Content: content,
		},
	}, cloned...)
}

func (c *ChatProps) SetupBuffer(buf *utils.Buffer) {
	c.Message = injectCurrentDateTime(c.Message, c.ClientContext)
	c.Message = injectPersonalization(c.Message, c.CustomInstruction)
	c.Message = injectMemoryCapabilities(c.Message, c.MemoryEnabled, c.MemoryHistoryEnabled)
	c.Message = injectReferencePrompts(c.Message, c.MemoryPrompt, c.RecentChatsPrompt)
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
