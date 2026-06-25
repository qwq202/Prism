package openai

import "chat/globals"

type ImageUrl struct {
	Url    string  `json:"url"`
	Detail *string `json:"detail,omitempty"`
}

type MessageContent struct {
	Type     string    `json:"type"`
	Text     *string   `json:"text,omitempty"`
	ImageUrl *ImageUrl `json:"image_url,omitempty"`
}

type MessageContents []MessageContent

type Message struct {
	Role             string                `json:"role"`
	Content          MessageContents       `json:"content"`
	Name             *string               `json:"name,omitempty"`
	FunctionCall     *globals.FunctionCall `json:"function_call,omitempty"` // only `function` role
	ToolCallId       *string               `json:"tool_call_id,omitempty"`  // only `tool` role
	ToolCalls        *globals.ToolCalls    `json:"tool_calls,omitempty"`    // only `assistant` role
	ReasoningContent *string               `json:"reasoning,omitempty"`     // only for claude reasoning models
}

type ResponseError struct {
	Message string      `json:"message"`
	Type    string      `json:"type,omitempty"`
	Code    interface{} `json:"code,omitempty"`
}

// ChatRequest is the request body for openai
type ChatRequest struct {
	Model               string                 `json:"model"`
	Messages            interface{}            `json:"messages"`
	MaxToken            *int                   `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int                   `json:"max_completion_tokens,omitempty"`
	Stream              bool                   `json:"stream"`
	StreamOptions       interface{}            `json:"stream_options,omitempty"`
	PresencePenalty     *float32               `json:"presence_penalty,omitempty"`
	FrequencyPenalty    *float32               `json:"frequency_penalty,omitempty"`
	Temperature         *float32               `json:"temperature,omitempty"`
	TopP                *float32               `json:"top_p,omitempty"`
	Tools               *globals.FunctionTools `json:"tools,omitempty"`
	ToolChoice          *interface{}           `json:"tool_choice,omitempty"` // string or object
}

// CompletionRequest is the request body for openai completion
type CompletionRequest struct {
	Model    string `json:"model"`
	Prompt   string `json:"prompt"`
	MaxToken *int   `json:"max_tokens,omitempty"`
	Stream   bool   `json:"stream"`
}

type ResponseMessage struct {
	Role             string                `json:"role"`
	Content          string                `json:"content"`
	Name             *string               `json:"name,omitempty"`
	FunctionCall     *globals.FunctionCall `json:"function_call,omitempty"`
	ToolCallId       *string               `json:"tool_call_id,omitempty"`
	ToolCalls        *globals.ToolCalls    `json:"tool_calls,omitempty"`
	ReasoningContent *string               `json:"reasoning_content,omitempty"`
	Reasoning        *string               `json:"reasoning,omitempty"`
}

// ChatResponse is the native http request body for openai
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int             `json:"index"`
		Message      ResponseMessage `json:"message"`
		FinishReason string          `json:"finish_reason"`
	} `json:"choices"`
	Error ResponseError       `json:"error"`
	Usage *globals.TokenUsage `json:"usage,omitempty"`
}

// ChatStreamResponse is the stream response body for openai
type ChatStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Delta        ResponseMessage `json:"delta"`
		Index        int             `json:"index"`
		FinishReason string          `json:"finish_reason"`
	} `json:"choices"`
	Error ResponseError       `json:"error"`
	Usage *globals.TokenUsage `json:"usage,omitempty"`
}

// CompletionResponse is the native http request body / stream response body for openai completion
type CompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Text  string `json:"text"`
		Index int    `json:"index"`
	} `json:"choices"`
	Usage *globals.TokenUsage `json:"usage,omitempty"`
}

type ChatStreamErrorResponse struct {
	Error ResponseError `json:"error"`
}

type ImageSize string

// ImageRequest is the request body for openai dalle image generation
type ImageRequest struct {
	Model  string    `json:"model"`
	Prompt string    `json:"prompt"`
	Size   ImageSize `json:"size"`
	N      int       `json:"n"`
}

type ImageResponse struct {
	Data []struct {
		Url     string `json:"url,omitempty"`
		B64Json string `json:"b64_json,omitempty"`
	} `json:"data"`
	Error ResponseError `json:"error"`
}

var (
	ImageSize256  ImageSize = "256x256"
	ImageSize512  ImageSize = "512x512"
	ImageSize1024 ImageSize = "1024x1024"
)
