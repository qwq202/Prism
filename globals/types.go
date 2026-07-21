package globals

import "encoding/json"

type Hook func(data *Chunk) error

const (
	GeminiThoughtSignatureLimit     = 32
	GeminiThoughtSignatureMaxBytes  = 4096
	ClaudeThinkingBlockLimit        = 32
	ClaudeThinkingTextMaxBytes      = 32768
	ClaudeThinkingSignatureMaxBytes = 4096
)

type GeminiHiddenMetadata struct {
	ThoughtSignatures []string `json:"thought_signatures,omitempty"`
}

type ClaudeThinkingBlock struct {
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
}

type ClaudeHiddenMetadata struct {
	ThinkingBlocks []ClaudeThinkingBlock `json:"thinking_blocks,omitempty"`
}

func (m *GeminiHiddenMetadata) UnmarshalJSON(data []byte) error {
	type rawMetadata struct {
		ThoughtSignatures []string `json:"thought_signatures,omitempty"`
		ThoughtSignature  *string  `json:"thought_signature,omitempty"`
	}

	var raw rawMetadata
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	signatures := make([]string, 0, len(raw.ThoughtSignatures)+1)
	for _, signature := range raw.ThoughtSignatures {
		signatures = append(signatures, signature)
	}

	if raw.ThoughtSignature != nil && len(*raw.ThoughtSignature) > 0 {
		signatures = append(signatures, *raw.ThoughtSignature)
	}

	m.ThoughtSignatures = NormalizeGeminiThoughtSignatures(signatures, GeminiThoughtSignatureLimit)
	return nil
}

type Message struct {
	MessageID            string                 `json:"id,omitempty"`
	Role                 string                 `json:"role"`
	Content              string                 `json:"content"`
	RequestID            string                 `json:"request_id,omitempty"` // client idempotency key; stripped before model requests
	Status               string                 `json:"status,omitempty"`
	CacheControl         map[string]interface{} `json:"cache_control,omitempty"`
	Model                string                 `json:"model,omitempty"`
	Name                 *string                `json:"name,omitempty"`
	FunctionCall         *FunctionCall          `json:"function_call,omitempty"`          // only `function` role
	ToolCallId           *string                `json:"tool_call_id,omitempty"`           // only `tool` role
	ToolCalls            *ToolCalls             `json:"tool_calls,omitempty"`             // only `assistant` role
	ReasoningContent     *string                `json:"reasoning_content,omitempty"`      // only for deepseek reasoner models
	GeminiHiddenMetadata *GeminiHiddenMetadata  `json:"gemini_hidden_metadata,omitempty"` // hidden gemini metadata for replay
	ClaudeHiddenMetadata *ClaudeHiddenMetadata  `json:"claude_hidden_metadata,omitempty"` // hidden claude thinking metadata for replay
	Quota                float32                `json:"quota,omitempty"`                  // persisted display cost
	Plan                 bool                   `json:"plan,omitempty"`                   // whether the cost used subscription quota
	ContextCleared       bool                   `json:"context_cleared,omitempty"`        // internal marker for context window resets
}

type Chunk struct {
	Content              string                `json:"content"`
	ToolCall             *ToolCalls            `json:"tool_call,omitempty"`
	FunctionCall         *FunctionCall         `json:"function_call,omitempty"`
	ReasoningContent     *string               `json:"reasoning_content,omitempty"`
	GeminiHiddenMetadata *GeminiHiddenMetadata `json:"gemini_hidden_metadata,omitempty"` // hidden gemini metadata for replay
	ClaudeHiddenMetadata *ClaudeHiddenMetadata `json:"claude_hidden_metadata,omitempty"` // hidden claude thinking metadata for replay
	Usage                *TokenUsage           `json:"usage,omitempty"`
}

type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

type PromptTokensDetails struct {
	CachedTokens     int `json:"cached_tokens,omitempty"`
	CacheWriteTokens int `json:"cache_write_tokens,omitempty"`
	ImageTokens      int `json:"image_tokens,omitempty"`
}

type TokenUsage struct {
	PromptTokens            int                     `json:"prompt_tokens,omitempty"`
	CompletionTokens        int                     `json:"completion_tokens,omitempty"`
	TotalTokens             int                     `json:"total_tokens,omitempty"`
	ImageTokens             int                     `json:"image_tokens,omitempty"`
	PromptTokensDetails     *PromptTokensDetails    `json:"prompt_tokens_details,omitempty"`
	PromptCacheHitTokens    int                     `json:"prompt_cache_hit_tokens,omitempty"`
	PromptCacheMissTokens   int                     `json:"prompt_cache_miss_tokens,omitempty"`
	PromptCacheWriteTokens  int                     `json:"prompt_cache_write_tokens,omitempty"`
	CompletionTokensDetails CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

type PromptCacheDetail struct {
	Provider             string `json:"provider,omitempty"`
	Mode                 string `json:"mode,omitempty"`
	ChannelType          string `json:"channel_type,omitempty"`
	Status               string `json:"status,omitempty"`
	Reason               string `json:"reason,omitempty"`
	PromptTokens         int    `json:"prompt_tokens,omitempty"`
	ThresholdTokens      int    `json:"threshold_tokens,omitempty"`
	Eligible             bool   `json:"eligible"`
	Attempted            bool   `json:"attempted"`
	PromptCacheKey       bool   `json:"prompt_cache_key,omitempty"`
	PromptCacheRetention string `json:"prompt_cache_retention,omitempty"`
	CacheControl         bool   `json:"cache_control,omitempty"`
	Manual               bool   `json:"manual,omitempty"`
}

func (u *TokenUsage) IsEmpty() bool {
	return u == nil ||
		(u.PromptTokens == 0 &&
			u.CompletionTokens == 0 &&
			u.TotalTokens == 0 &&
			u.ImageTokens == 0 &&
			(u.PromptTokensDetails == nil ||
				(u.PromptTokensDetails.CachedTokens == 0 &&
					u.PromptTokensDetails.CacheWriteTokens == 0 &&
					u.PromptTokensDetails.ImageTokens == 0)) &&
			u.PromptCacheHitTokens == 0 &&
			u.PromptCacheMissTokens == 0 &&
			u.PromptCacheWriteTokens == 0 &&
			u.CompletionTokensDetails.ReasoningTokens == 0)
}

func NormalizeTokenUsage(usage *TokenUsage) *TokenUsage {
	if usage == nil {
		return nil
	}

	normalized := *usage
	if normalized.PromptTokensDetails != nil {
		if normalized.PromptCacheHitTokens == 0 && normalized.PromptTokensDetails.CachedTokens > 0 {
			normalized.PromptCacheHitTokens = normalized.PromptTokensDetails.CachedTokens
		}
		if normalized.PromptCacheWriteTokens == 0 && normalized.PromptTokensDetails.CacheWriteTokens > 0 {
			normalized.PromptCacheWriteTokens = normalized.PromptTokensDetails.CacheWriteTokens
		}
		if normalized.ImageTokens == 0 && normalized.PromptTokensDetails.ImageTokens > 0 {
			normalized.ImageTokens = normalized.PromptTokensDetails.ImageTokens
		}
		normalized.PromptTokensDetails = nil
	}
	if normalized.TotalTokens == 0 && (normalized.PromptTokens > 0 || normalized.CompletionTokens > 0) {
		normalized.TotalTokens = normalized.PromptTokens + normalized.CompletionTokens
	}
	if normalized.PromptCacheHitTokens > 0 &&
		normalized.PromptCacheWriteTokens == 0 &&
		normalized.PromptCacheMissTokens == 0 &&
		normalized.PromptTokens > normalized.PromptCacheHitTokens {
		normalized.PromptCacheMissTokens = normalized.PromptTokens - normalized.PromptCacheHitTokens
	}
	return &normalized
}

type ChatSegmentToolCall struct {
	Id        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
	Result    string `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
	Status    string `json:"status"`
}

type ChatSegmentResponse struct {
	Conversation  int64                `json:"conversation"`
	RequestID     string               `json:"request_id,omitempty"`
	RequestStatus string               `json:"request_status,omitempty"`
	Accepted      bool                 `json:"accepted,omitempty"`
	Retryable     bool                 `json:"retryable,omitempty"`
	Quota         float32              `json:"quota"`
	Keyword       string               `json:"keyword"`
	Message       string               `json:"message"`
	Title         string               `json:"title,omitempty"`
	ToolCall      *ChatSegmentToolCall `json:"tool_call,omitempty"`
	End           bool                 `json:"end"`
	Plan          bool                 `json:"plan"`
	ResponseType  string               `json:"response_type,omitempty"`
	Capabilities  []string             `json:"capabilities,omitempty"`
}

type ListModels struct {
	Object string           `json:"object"`
	Data   []ListModelsItem `json:"data"`
}

type ListModelsItem struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ProxyConfig struct {
	ProxyType int    `json:"proxy_type" mapstructure:"proxytype"`
	Proxy     string `json:"proxy" mapstructure:"proxy"`
	Username  string `json:"username" mapstructure:"username"`
	Password  string `json:"password" mapstructure:"password"`
}
