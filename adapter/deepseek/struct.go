package deepseek

import (
	"chat/globals"
)

// DeepSeek API is similar to OpenAI API with additional reasoning content

type ChatRequest struct {
	Model            string                 `json:"model"`
	Messages         []globals.Message      `json:"messages"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	Stream           bool                   `json:"stream"`
	Temperature      *float32               `json:"temperature,omitempty"`
	TopP             *float32               `json:"top_p,omitempty"`
	PresencePenalty  *float32               `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float32               `json:"frequency_penalty,omitempty"`
	Stop             interface{}            `json:"stop,omitempty"`
	ResponseFormat   interface{}            `json:"response_format,omitempty"`
	Thinking         interface{}            `json:"thinking,omitempty"`
	StreamOptions    interface{}            `json:"stream_options,omitempty"`
	Logprobs         *bool                  `json:"logprobs,omitempty"`
	TopLogprobs      *int                   `json:"top_logprobs,omitempty"`
	Tools            *globals.FunctionTools `json:"tools,omitempty"`
	ToolChoice       *interface{}           `json:"tool_choice,omitempty"`
}

// ChatResponse is the native http request body for deepseek
type ChatResponse struct {
	ID                string `json:"id"`
	Object            string `json:"object"`
	Created           int64  `json:"created"`
	Model             string `json:"model"`
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
	Choices           []struct {
		Index        int             `json:"index"`
		Message      globals.Message `json:"message"`
		FinishReason string          `json:"finish_reason"`
		Logprobs     interface{}     `json:"logprobs,omitempty"`
	} `json:"choices"`
	Usage struct {
		PromptTokens            int `json:"prompt_tokens"`
		CompletionTokens        int `json:"completion_tokens"`
		TotalTokens             int `json:"total_tokens"`
		PromptCacheHitTokens    int `json:"prompt_cache_hit_tokens,omitempty"`
		PromptCacheMissTokens   int `json:"prompt_cache_miss_tokens,omitempty"`
		CompletionTokensDetails struct {
			ReasoningTokens int `json:"reasoning_tokens,omitempty"`
		} `json:"completion_tokens_details,omitempty"`
	} `json:"usage"`
}

// ChatStreamResponse is the stream response body for deepseek
type ChatStreamResponse struct {
	ID                string `json:"id"`
	Object            string `json:"object"`
	Created           int64  `json:"created"`
	Model             string `json:"model"`
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
	Choices           []struct {
		Delta        globals.Message `json:"delta"`
		Index        int             `json:"index"`
		FinishReason string          `json:"finish_reason"`
		Logprobs     interface{}     `json:"logprobs,omitempty"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens            int `json:"prompt_tokens"`
		CompletionTokens        int `json:"completion_tokens"`
		TotalTokens             int `json:"total_tokens"`
		PromptCacheHitTokens    int `json:"prompt_cache_hit_tokens,omitempty"`
		PromptCacheMissTokens   int `json:"prompt_cache_miss_tokens,omitempty"`
		CompletionTokensDetails struct {
			ReasoningTokens int `json:"reasoning_tokens,omitempty"`
		} `json:"completion_tokens_details,omitempty"`
	} `json:"usage,omitempty"`
}

type ChatStreamErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}
