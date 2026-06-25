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
	CacheControl         map[string]interface{} `json:"cache_control,omitempty"`
	CustomInstruction    string                 `json:"custom_instruction,omitempty"`
	LearningMode         bool                   `json:"learning_mode,omitempty"`
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
	PreviousResponseID   *string                `json:"previous_response_id,omitempty"`
	ReasoningEffort      *string                `json:"reasoning_effort,omitempty"`
	ResponseInclude      []string               `json:"include,omitempty"`
	ResponseStore        *bool                  `json:"store,omitempty"`
	ParallelToolCalls    *bool                  `json:"parallel_tool_calls,omitempty"`
	StreamOptions        interface{}            `json:"stream_options,omitempty"`
	PromptCacheKey       *string                `json:"prompt_cache_key,omitempty"`
	PromptCacheRetention *string                `json:"prompt_cache_retention,omitempty"`
	SessionID            *string                `json:"session_id,omitempty"`
	Thinking             interface{}            `json:"thinking,omitempty"`
	CachedContent        *string                `json:"cachedContent,omitempty"`
	CachedContentSnake   *string                `json:"cached_content,omitempty"`
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
const learningModePromptPrefix = "Learning mode guidance:"
const memoryCapabilityPromptPrefix = "Memory capability state:"
const currentModelPromptPrefix = "Current conversation model reference:"
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

func buildLearningModePrompt() string {
	return fmt.Sprintf(
		"%s\n- Learning mode is enabled for this turn.\n- Do not give the final answer immediately unless the user explicitly asks for it after trying, or safety/urgent correctness requires it.\n- Guide the user with short questions, hints, checkpoints, and step-by-step reasoning prompts so they can discover the answer themselves.\n- Ask what they have tried or what they think the next step is when useful.\n- If the user gets stuck, provide the smallest helpful hint first; then gradually reveal more.\n- When the user reaches the answer or asks to wrap up, summarize the method and key lesson clearly.",
		learningModePromptPrefix,
	)
}

func injectLearningMode(messages []globals.Message, enabled bool) []globals.Message {
	if !enabled {
		return messages
	}

	cloned := utils.DeepCopy[[]globals.Message](messages)
	prompt := buildLearningModePrompt()

	for i := range cloned {
		if cloned[i].Role != globals.System {
			continue
		}

		content := strings.TrimSpace(cloned[i].Content)
		if strings.Contains(content, learningModePromptPrefix) {
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

func buildCurrentModelPrompt(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}

	return fmt.Sprintf(
		"%s\n- The user is currently chatting with model: %s.\n- This value is the authoritative current-turn model identity.\n- If older assistant messages in this conversation mention a different model, provider, self-introduction, or identity, treat them as stale outputs from an earlier model selection rather than the current identity.\n- If the user asks who you are or which model/provider is being used right now, answer only with the current model above.\n- Do not explain identity conflicts by inventing hidden training identities, self-learned identities, or similar internal narratives.\n- Do not manually emit <think> or </think> tags in normal response text; those tags are reserved for system-native reasoning channels.\n- If the model changes in a later turn, use the updated model reference from that turn instead of any prior assumption.",
		currentModelPromptPrefix,
		model,
	)
}

func upsertPromptSection(content string, prefix string, prompt string) string {
	content = strings.TrimSpace(content)
	prompt = strings.TrimSpace(prompt)

	if prompt == "" {
		return content
	}

	if content == "" {
		return fmt.Sprintf("%s\n%s", prefix, prompt)
	}

	sections := strings.Split(content, "\n\n")
	replaced := false

	for i, section := range sections {
		lines := strings.Split(strings.TrimSpace(section), "\n")
		if len(lines) == 0 {
			continue
		}

		if strings.TrimSpace(lines[0]) == prefix {
			sections[i] = fmt.Sprintf("%s\n%s", prefix, prompt)
			replaced = true
			break
		}
	}

	if !replaced {
		sections = append(sections, fmt.Sprintf("%s\n%s", prefix, prompt))
	}

	return strings.Join(sections, "\n\n")
}

func injectCurrentModel(messages []globals.Message, model string) []globals.Message {
	prompt := buildCurrentModelPrompt(model)
	if prompt == "" {
		return messages
	}

	cloned := utils.DeepCopy[[]globals.Message](messages)
	body := strings.TrimPrefix(prompt, currentModelPromptPrefix+"\n")

	for i := range cloned {
		if cloned[i].Role != globals.System {
			continue
		}

		cloned[i].Content = upsertPromptSection(cloned[i].Content, currentModelPromptPrefix, body)
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
	c.Message = injectLearningMode(c.Message, c.LearningMode)
	c.Message = injectMemoryCapabilities(c.Message, c.MemoryEnabled, c.MemoryHistoryEnabled)
	currentModel := strings.TrimSpace(c.OriginalModel)
	if currentModel == "" {
		currentModel = strings.TrimSpace(c.Model)
	}
	c.Message = injectCurrentModel(c.Message, currentModel)
	c.Message = injectReferencePrompts(c.Message, c.MemoryPrompt, c.RecentChatsPrompt)
	if buf == nil {
		return
	}
	buf.SetResponseFormat(c.ResponseFormat)
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
