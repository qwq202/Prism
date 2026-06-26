package conversation

import (
	"chat/auth"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"errors"
	"strings"
)

const defaultConversationName = "new chat"
const minConversationContext = 5
const maxConversationContext = 25
const defaultConversationContext = minConversationContext

type Conversation struct {
	Auth                     bool                   `json:"auth"`
	UserID                   int64                  `json:"user_id"`
	Id                       int64                  `json:"id"`
	Name                     string                 `json:"name"`
	Message                  []globals.Message      `json:"message"`
	Model                    string                 `json:"model"`
	TaskID                   string                 `json:"task_id,omitempty"`
	UpdatedAt                string                 `json:"updated_at,omitempty"`
	Favorite                 bool                   `json:"favorite"`
	Persisted                bool                   `json:"-"`
	EnableWeb                bool                   `json:"enable_web"`
	WebSearch                bool                   `json:"web_search"`
	URLContext               bool                   `json:"url_context"`
	XSearch                  bool                   `json:"x_search"`
	Fetch                    bool                   `json:"fetch"`
	GeminiThinkingBudget     int                    `json:"gemini_thinking_budget"`
	DeepseekThinkingDisabled bool                   `json:"deepseek_thinking_disabled"`
	DeepseekReasoningEffort  string                 `json:"deepseek_reasoning_effort"`
	OpenAIReasoningEffort    string                 `json:"openai_reasoning_effort"`
	OpenAIReasoningSummary   string                 `json:"openai_reasoning_summary"`
	Shared                   bool                   `json:"shared"`
	Context                  int                    `json:"context"`
	CustomInstruction        string                 `json:"custom_instruction,omitempty"`
	LearningMode             bool                   `json:"learning_mode"`
	MemoryEnabled            bool                   `json:"memory_enabled"`
	MemoryHistoryEnabled     bool                   `json:"memory_history_enabled"`
	ResponseFormat           interface{}            `json:"-"`
	CacheControl             map[string]interface{} `json:"cache_control,omitempty"`
	PromptCacheKey           string                 `json:"prompt_cache_key,omitempty"`
	PromptCacheRetention     string                 `json:"prompt_cache_retention,omitempty"`
	CachedContent            string                 `json:"cachedContent,omitempty"`
	CachedContentSnake       string                 `json:"cached_content,omitempty"`
	Thinking                 interface{}            `json:"-"`

	MaxTokens         *int     `json:"max_tokens,omitempty"`
	Temperature       *float32 `json:"temperature,omitempty"`
	TopP              *float32 `json:"top_p,omitempty"`
	TopK              *int     `json:"top_k,omitempty"`
	PresencePenalty   *float32 `json:"presence_penalty,omitempty"`
	FrequencyPenalty  *float32 `json:"frequency_penalty,omitempty"`
	RepetitionPenalty *float32 `json:"repetition_penalty,omitempty"`
}

type FormMessage struct {
	Type                    string                 `json:"type"`
	Message                 string                 `json:"message"`
	Transient               bool                   `json:"transient,omitempty"`
	Web                     bool                   `json:"web"`
	WebSearch               bool                   `json:"web_search"`
	URLContext              bool                   `json:"url_context"`
	XSearch                 bool                   `json:"x_search"`
	Fetch                   bool                   `json:"fetch"`
	GeminiThinkingBudget    int                    `json:"gemini_thinking_budget"`
	DeepseekThinkingEnabled *bool                  `json:"deepseek_thinking_enabled,omitempty"`
	DeepseekReasoningEffort string                 `json:"deepseek_reasoning_effort"`
	OpenAIReasoningEffort   string                 `json:"openai_reasoning_effort"`
	OpenAIReasoningSummary  string                 `json:"openai_reasoning_summary"`
	Model                   string                 `json:"model"`
	IgnoreContext           bool                   `json:"ignore_context"`
	Context                 int                    `json:"context"`
	CustomInstruction       string                 `json:"custom_instruction,omitempty"`
	LearningMode            bool                   `json:"learning_mode"`
	MemoryEnabled           bool                   `json:"memory_enabled"`
	MemoryHistoryEnabled    bool                   `json:"memory_history_enabled"`
	ResponseFormat          interface{}            `json:"response_format,omitempty"`
	CacheControl            map[string]interface{} `json:"cache_control,omitempty"`
	PromptCacheKey          string                 `json:"prompt_cache_key,omitempty"`
	PromptCacheRetention    string                 `json:"prompt_cache_retention,omitempty"`
	CachedContent           string                 `json:"cachedContent,omitempty"`
	CachedContentSnake      string                 `json:"cached_content,omitempty"`
	Thinking                interface{}            `json:"thinking,omitempty"`

	// request params
	MaxTokens         *int     `json:"max_tokens,omitempty"`
	Temperature       *float32 `json:"temperature,omitempty"`
	TopP              *float32 `json:"top_p,omitempty"`
	TopK              *int     `json:"top_k,omitempty"`
	PresencePenalty   *float32 `json:"presence_penalty,omitempty"`
	FrequencyPenalty  *float32 `json:"frequency_penalty,omitempty"`
	RepetitionPenalty *float32 `json:"repetition_penalty,omitempty"`
}

func NewAnonymousConversation() *Conversation {
	return &Conversation{
		Auth:    false,
		UserID:  -1,
		Id:      -1,
		Name:    defaultConversationName,
		Message: []globals.Message{},
		Model:   globals.GPT3Turbo,
		Context: defaultConversationContext,
	}
}

func NewConversation(db *sql.DB, id int64) *Conversation {
	return &Conversation{
		Auth:    true,
		UserID:  id,
		Id:      GetConversationLengthByUserID(db, id) + 1,
		Name:    defaultConversationName,
		Message: []globals.Message{},
		Model:   globals.GPT3Turbo,
	}
}

func ExtractConversation(db *sql.DB, user *auth.User, id int64, ref string) *Conversation {
	if ref != "" {
		if instance := UseSharedConversation(db, user, ref); instance != nil {
			return instance
		}
	}

	if user == nil {
		return NewAnonymousConversation()
	}

	if id == -1 {
		// create new conversation
		return NewConversation(db, user.GetID(db))
	}

	// load conversation
	if instance := LoadConversation(db, user.GetID(db), id); instance != nil {
		return instance
	} else {
		return NewConversation(db, user.GetID(db))
	}
}

func (c *Conversation) GetModel() string {
	if len(c.Model) == 0 {
		return globals.GPT3Turbo
	}
	return c.Model
}

func (c *Conversation) IsEnableWeb() bool {
	return c.EnableWeb
}

func (c *Conversation) IsEnableWebSearch() bool {
	return c.WebSearch
}

func (c *Conversation) IsEnableURLContext() bool {
	return c.URLContext
}

func (c *Conversation) IsEnableXSearch() bool {
	return c.XSearch
}

func (c *Conversation) IsEnableFetch() bool {
	return c.Fetch
}

func (c *Conversation) GetGeminiThinkingBudget() *int {
	return &c.GeminiThinkingBudget
}

func (c *Conversation) IsDeepseekThinkingEnabled() bool {
	return !c.DeepseekThinkingDisabled
}

func (c *Conversation) GetDeepseekReasoningEffort() string {
	return globals.NormalizeDeepseekReasoningEffort(c.DeepseekReasoningEffort)
}

func (c *Conversation) GetOpenAIReasoningEffort() string {
	return strings.TrimSpace(strings.ToLower(c.OpenAIReasoningEffort))
}

func (c *Conversation) GetOpenAIReasoningSummary() string {
	return strings.TrimSpace(strings.ToLower(c.OpenAIReasoningSummary))
}

func (c *Conversation) GetResponseFormat() interface{} {
	return c.ResponseFormat
}

func (c *Conversation) GetCacheControl() map[string]interface{} {
	if len(c.CacheControl) == 0 {
		return nil
	}
	return c.CacheControl
}

func optionalStringPtr(value string) *string {
	text := strings.TrimSpace(value)
	if text == "" {
		return nil
	}
	return &text
}

func (c *Conversation) GetPromptCacheKey() *string {
	return optionalStringPtr(c.PromptCacheKey)
}

func (c *Conversation) GetPromptCacheRetention() *string {
	return optionalStringPtr(c.PromptCacheRetention)
}

func (c *Conversation) GetCachedContent() *string {
	return optionalStringPtr(c.CachedContent)
}

func (c *Conversation) GetCachedContentSnake() *string {
	return optionalStringPtr(c.CachedContentSnake)
}

func (c *Conversation) GetThinking() interface{} {
	return c.Thinking
}

func (c *Conversation) GetContextLength() int {
	if c.Context <= 0 {
		return defaultConversationContext
	}

	return c.Context
}

func (c *Conversation) SetModel(model string) {
	if len(model) == 0 {
		model = globals.GPT3Turbo
	}
	c.Model = model
}

func (c *Conversation) SetEnableWeb(enable bool) {
	c.EnableWeb = enable
}

func (c *Conversation) SetEnableWebSearch(enable bool) {
	c.WebSearch = enable
}

func (c *Conversation) SetEnableURLContext(enable bool) {
	c.URLContext = enable
}

func (c *Conversation) SetEnableXSearch(enable bool) {
	c.XSearch = enable
}

func (c *Conversation) SetEnableFetch(enable bool) {
	c.Fetch = enable
}

func (c *Conversation) SetGeminiThinkingBudget(budget int) {
	c.GeminiThinkingBudget = budget
}

func (c *Conversation) SetDeepseekThinkingEnabled(enabled bool) {
	c.DeepseekThinkingDisabled = !enabled
}

func (c *Conversation) SetDeepseekReasoningEffort(effort string) {
	c.DeepseekReasoningEffort = globals.NormalizeDeepseekReasoningEffort(effort)
}

func (c *Conversation) SetOpenAIReasoningEffort(effort string) {
	c.OpenAIReasoningEffort = strings.TrimSpace(strings.ToLower(effort))
}

func (c *Conversation) SetOpenAIReasoningSummary(summary string) {
	c.OpenAIReasoningSummary = globals.NormalizeOpenAIResponsesReasoningSummary(summary)
}

func (c *Conversation) SetResponseFormat(format interface{}) {
	c.ResponseFormat = format
}

func (c *Conversation) SetCacheControl(cacheControl map[string]interface{}) {
	if len(cacheControl) == 0 {
		c.CacheControl = nil
		return
	}
	c.CacheControl = cacheControl
}

func (c *Conversation) SetPromptCacheKey(key string) {
	c.PromptCacheKey = strings.TrimSpace(key)
}

func (c *Conversation) SetPromptCacheRetention(retention string) {
	c.PromptCacheRetention = strings.TrimSpace(retention)
}

func (c *Conversation) SetCachedContent(cachedContent string) {
	c.CachedContent = strings.TrimSpace(cachedContent)
}

func (c *Conversation) SetCachedContentSnake(cachedContent string) {
	c.CachedContentSnake = strings.TrimSpace(cachedContent)
}

func (c *Conversation) SetThinking(thinking interface{}) {
	c.Thinking = thinking
}

func (c *Conversation) GetTemperature() *float32 {
	return c.Temperature
}

func (c *Conversation) SetTemperature(temperature *float32) {
	c.Temperature = temperature
}

func (c *Conversation) GetTopP() *float32 {
	return c.TopP
}

func (c *Conversation) SetTopP(topP *float32) {
	c.TopP = topP
}

func (c *Conversation) GetTopK() *int {
	return c.TopK
}

func (c *Conversation) SetTopK(topK *int) {
	c.TopK = topK
}

func (c *Conversation) GetPresencePenalty() *float32 {
	return c.PresencePenalty
}

func (c *Conversation) SetPresencePenalty(presencePenalty *float32) {
	c.PresencePenalty = presencePenalty
}

func (c *Conversation) GetFrequencyPenalty() *float32 {
	return c.FrequencyPenalty
}

func (c *Conversation) SetFrequencyPenalty(frequencyPenalty *float32) {
	c.FrequencyPenalty = frequencyPenalty
}

func (c *Conversation) GetRepetitionPenalty() *float32 {
	return c.RepetitionPenalty
}

func (c *Conversation) SetRepetitionPenalty(repetitionPenalty *float32) {
	c.RepetitionPenalty = repetitionPenalty
}

func (c *Conversation) GetMaxTokens() *int {
	return c.MaxTokens
}

func (c *Conversation) SetMaxTokens(maxTokens *int) {
	c.MaxTokens = maxTokens
}

func (c *Conversation) SetContextLength(context int, ignore bool) {
	if ignore {
		c.Context = 1
		return
	}

	c.Context = normalizeContextLength(context)
}

func (c *Conversation) GetCustomInstruction() string {
	return strings.TrimSpace(c.CustomInstruction)
}

func (c *Conversation) SetCustomInstruction(customInstruction string) {
	c.CustomInstruction = strings.TrimSpace(customInstruction)
}

func (c *Conversation) IsLearningModeEnabled() bool {
	return c.LearningMode
}

func (c *Conversation) SetLearningMode(enabled bool) {
	c.LearningMode = enabled
}

func (c *Conversation) IsMemoryEnabled() bool {
	return c.MemoryEnabled
}

func (c *Conversation) SetMemoryEnabled(enabled bool) {
	c.MemoryEnabled = enabled
}

func (c *Conversation) IsMemoryHistoryEnabled() bool {
	return c.MemoryHistoryEnabled
}

func (c *Conversation) SetMemoryHistoryEnabled(enabled bool) {
	c.MemoryHistoryEnabled = enabled
}

func (c *Conversation) GetName() string {
	return c.Name
}

func (c *Conversation) SetName(db *sql.DB, name string) {
	c.Name = utils.Extract(name, 50, "...")
	c.SaveConversation(db)
}

func (c *Conversation) GetId() int64 {
	return c.Id
}

func (c *Conversation) GetUserID() int64 {
	return c.UserID
}

func (c *Conversation) SetId(id int64) {
	c.Id = id
}

func (c *Conversation) GetMessage() []globals.Message {
	return c.Message
}

func (c *Conversation) HasMessageId(id int) bool {
	return id >= 0 && id < len(c.Message)
}

func (c *Conversation) GetMessageById(id int) globals.Message {
	if !c.HasMessageId(id) {
		return globals.Message{}
	}
	return c.Message[id]
}

func (c *Conversation) GetMessageLength() int {
	return len(c.Message)
}

func (c *Conversation) GetMessageSegment(length int) []globals.Message {
	return selectRecentContextMessages(c.Message, length)
}

func hasMessagePayload(message globals.Message) bool {
	return strings.TrimSpace(message.Content) != "" ||
		message.FunctionCall != nil ||
		message.ToolCallId != nil ||
		(message.ToolCalls != nil && len(*message.ToolCalls) > 0) ||
		message.ReasoningContent != nil ||
		message.GeminiHiddenMetadata != nil ||
		message.ClaudeHiddenMetadata != nil
}

func cleanContextMessages(messages []globals.Message) []globals.Message {
	cleaned := make([]globals.Message, 0, len(messages))
	for _, message := range messages {
		if !hasMessagePayload(message) {
			continue
		}

		cleaned = append(cleaned, message)
	}

	lastClearIndex := -1
	for index, message := range cleaned {
		if message.ContextCleared {
			lastClearIndex = index
		}
	}
	if lastClearIndex >= 0 {
		cleaned = cleaned[lastClearIndex:]
	}

	result := make([]globals.Message, 0, len(cleaned))
	for index, message := range cleaned {
		if message.Role == globals.Assistant && index+1 < len(cleaned) && cleaned[index+1].Role == globals.Assistant {
			continue
		}

		result = append(result, message)
	}

	return result
}

func selectRecentContextMessages(messages []globals.Message, length int) []globals.Message {
	cleaned := cleanContextMessages(messages)
	if length <= 0 {
		length = defaultConversationContext
	}
	if length > len(cleaned) {
		return cleaned
	}

	return cleaned[len(cleaned)-length:]
}

func normalizeContextLength(length int) int {
	if length <= 0 {
		return defaultConversationContext
	}
	if length < minConversationContext {
		return minConversationContext
	}
	if length > maxConversationContext {
		return maxConversationContext
	}

	return length
}

func (c *Conversation) GetChatMessage(restart bool) []globals.Message {
	if restart {
		// remove all last `assistant` role message
		cp := CopyMessage(c.Message)

		var index int
		for index = len(cp) - 1; index >= 0; index-- {
			if cp[index].Role != globals.Assistant {
				break
			}
		}
		if index >= 0 {
			cp = cp[:index+1]
		}

		return selectRecentContextMessages(cp, c.GetContextLength())
	}

	return c.GetMessageSegment(c.GetContextLength())
}

func CopyMessage(message []globals.Message) []globals.Message {
	return utils.DeepCopy[[]globals.Message](message) // deep copy
}

func (c *Conversation) GetLastMessage() globals.Message {
	if len(c.Message) == 0 {
		return globals.Message{}
	}
	return c.Message[len(c.Message)-1]
}

func (c *Conversation) AddMessage(message globals.Message) {
	c.Message = append(c.Message, message)
}

func (c *Conversation) AddMessages(messages []globals.Message) {
	c.Message = append(c.Message, messages...)
}

func (c *Conversation) InsertMessage(message globals.Message, index int) {
	c.Message = append(c.Message[:index], append([]globals.Message{message}, c.Message[index:]...)...)
}

func (c *Conversation) InsertMessages(messages []globals.Message, index int) {
	c.Message = append(c.Message[:index], append(messages, c.Message[index:]...)...)
}

func (c *Conversation) AddMessageFromUser(message string) {
	c.AddMessage(globals.Message{
		Role:    globals.User,
		Content: message,
	})
}

func (c *Conversation) AddMessageFromAssistant(message string) {
	c.AddMessage(globals.Message{
		Role:    globals.Assistant,
		Content: message,
		Model:   c.GetModel(),
	})
}

func (c *Conversation) AddMessageFromSystem(message string) {
	c.AddMessage(globals.Message{
		Role:    globals.System,
		Content: message,
	})
}

func GetMessage(data []byte) (string, error) {
	form, err := utils.Unmarshal[FormMessage](data)
	form.Message = strings.TrimSpace(form.Message)
	if err != nil {
		return "", err
	}
	if len(form.Message) == 0 {
		return "", errors.New("message is empty")
	}
	return form.Message, nil
}

func (c *Conversation) ApplyParam(form *FormMessage) {
	c.SetModel(form.Model)
	c.SetEnableWeb(form.Web || form.WebSearch || form.URLContext || form.XSearch)
	c.SetEnableWebSearch(utils.Multi(form.Web && !form.WebSearch && !form.URLContext && !form.XSearch, true, form.WebSearch))
	c.SetEnableURLContext(utils.Multi(form.Web && !form.WebSearch && !form.URLContext && !form.XSearch, true, form.URLContext))
	c.SetEnableXSearch(form.XSearch)
	c.SetEnableFetch(form.Fetch)
	c.SetGeminiThinkingBudget(form.GeminiThinkingBudget)
	if form.DeepseekThinkingEnabled != nil {
		c.SetDeepseekThinkingEnabled(*form.DeepseekThinkingEnabled)
	}
	if strings.TrimSpace(form.DeepseekReasoningEffort) != "" {
		c.SetDeepseekReasoningEffort(form.DeepseekReasoningEffort)
	}
	c.SetOpenAIReasoningEffort(form.OpenAIReasoningEffort)
	c.SetOpenAIReasoningSummary(form.OpenAIReasoningSummary)
	c.SetContextLength(form.Context, form.IgnoreContext)
	c.SetCustomInstruction(form.CustomInstruction)
	c.SetLearningMode(form.LearningMode)
	c.SetMemoryEnabled(form.MemoryEnabled)
	c.SetMemoryHistoryEnabled(form.MemoryHistoryEnabled)
	c.SetResponseFormat(form.ResponseFormat)
	c.SetCacheControl(form.CacheControl)
	c.SetPromptCacheKey(form.PromptCacheKey)
	c.SetPromptCacheRetention(form.PromptCacheRetention)
	c.SetCachedContent(form.CachedContent)
	c.SetCachedContentSnake(form.CachedContentSnake)
	c.SetThinking(form.Thinking)

	c.SetMaxTokens(form.MaxTokens)
	c.SetTemperature(form.Temperature)
	c.SetTopP(form.TopP)
	c.SetTopK(form.TopK)
	c.SetPresencePenalty(form.PresencePenalty)
	c.SetFrequencyPenalty(form.FrequencyPenalty)
	c.SetRepetitionPenalty(form.RepetitionPenalty)
}

func (c *Conversation) AddMessageFromByte(data []byte) (string, error) {
	form, err := utils.Unmarshal[FormMessage](data)
	if err != nil {
		return "", err
	}
	form.Message = strings.TrimSpace(form.Message)
	if len(form.Message) == 0 {
		return "", errors.New("message is empty")
	}

	if err := c.AddMessageFromForm(&form); err != nil {
		return "", err
	}

	return form.Message, nil
}

func (c *Conversation) AddMessageFromForm(form *FormMessage) error {
	form.Message = strings.TrimSpace(form.Message)
	if len(form.Message) == 0 {
		return errors.New("message is empty")
	}

	message := globals.Message{
		Role:           globals.User,
		Content:        form.Message,
		ContextCleared: form.IgnoreContext,
	}
	c.AddMessage(message)
	c.ApplyParam(form)

	return nil
}

func (c *Conversation) HandleMessage(db *sql.DB, form *FormMessage) bool {
	head := len(c.Message) == 0 || c.Name == defaultConversationName
	if err := c.AddMessageFromForm(form); err != nil {
		return false
	}
	if head {
		c.SetName(db, form.Message)
	}
	c.SaveConversation(db)
	return true
}

func (c *Conversation) HandleMessageFromByte(db *sql.DB, data []byte) bool {
	head := len(c.Message) == 0
	msg, err := c.AddMessageFromByte(data)
	if err != nil {
		return false
	}
	if head {
		c.SetName(db, msg)
	}
	c.SaveConversation(db)
	return true
}

func (c *Conversation) GetLatestMessage() string {
	return c.Message[len(c.Message)-1].Content
}

func (c *Conversation) SaveResponse(db *sql.DB, message globals.Message) bool {
	message.Role = globals.Assistant
	if message.Model == "" {
		message.Model = c.GetModel()
	}

	// Keep UI semantics stable: do not persist assistant messages with no visible payload.
	if strings.TrimSpace(message.Content) == "" && message.FunctionCall == nil && (message.ToolCalls == nil || len(*message.ToolCalls) == 0) {
		return false
	}

	c.AddMessage(message)
	c.SaveConversation(db)
	return true
}

func (c *Conversation) CountMessagesByRole(role string) int {
	count := 0
	for _, message := range c.Message {
		if message.Role == role {
			count++
		}
	}

	return count
}

func (c *Conversation) RemoveMessage(index int) globals.Message {
	if index < 0 || index >= len(c.Message) {
		return globals.Message{}
	}
	message := c.Message[index]
	c.Message = append(c.Message[:index], c.Message[index+1:]...)
	return message
}

func (c *Conversation) RemoveLatestMessage() globals.Message {
	return c.RemoveMessage(len(c.Message) - 1)
}

func (c *Conversation) RemoveLatestMessageWithRole(role string) globals.Message {
	if len(c.Message) == 0 {
		return globals.Message{}
	}

	message := c.Message[len(c.Message)-1]
	if message.Role == role {
		return c.RemoveLatestMessage()
	}

	return globals.Message{}
}

func (c *Conversation) EditMessage(index int, message string) {
	if index < 0 || index >= len(c.Message) {
		return
	}
	c.Message[index].Content = message
}

func (c *Conversation) DeleteMessage(index int) {
	if index < 0 || index >= len(c.Message) {
		return
	}
	c.Message = append(c.Message[:index], c.Message[index+1:]...)
}

func (c *Conversation) GetTaskID() string {
	return c.TaskID
}

func (c *Conversation) SetTaskID(taskID string) {
	c.TaskID = taskID
}
