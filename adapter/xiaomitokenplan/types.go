package xiaomitokenplan

import "chat/globals"

type ImageURL struct {
	URL    string  `json:"url"`
	Detail *string `json:"detail,omitempty"`
}

type MessageContent struct {
	Type     string    `json:"type"`
	Text     *string   `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type MessageContents []MessageContent

type Message struct {
	Role             string                `json:"role"`
	Content          interface{}           `json:"content"`
	Name             *string               `json:"name,omitempty"`
	FunctionCall     *globals.FunctionCall `json:"function_call,omitempty"`
	ToolCallID       *string               `json:"tool_call_id,omitempty"`
	ToolCalls        *globals.ToolCalls    `json:"tool_calls,omitempty"`
	ReasoningContent *string               `json:"reasoning_content,omitempty"`
}

type ChatRequest struct {
	Model               string                 `json:"model"`
	Messages            interface{}            `json:"messages"`
	MaxCompletionTokens *int                   `json:"max_completion_tokens,omitempty"`
	Stream              bool                   `json:"stream"`
	PresencePenalty     *float32               `json:"presence_penalty,omitempty"`
	FrequencyPenalty    *float32               `json:"frequency_penalty,omitempty"`
	Temperature         *float32               `json:"temperature,omitempty"`
	TopP                *float32               `json:"top_p,omitempty"`
	Stop                interface{}            `json:"stop,omitempty"`
	ResponseFormat      interface{}            `json:"response_format,omitempty"`
	Thinking            interface{}            `json:"thinking,omitempty"`
	Tools               *globals.FunctionTools `json:"tools,omitempty"`
	ToolChoice          *interface{}           `json:"tool_choice,omitempty"`
}

type ResponseMessage struct {
	Role             string                `json:"role"`
	Content          string                `json:"content"`
	Name             *string               `json:"name,omitempty"`
	FunctionCall     *globals.FunctionCall `json:"function_call,omitempty"`
	ToolCallID       *string               `json:"tool_call_id,omitempty"`
	ToolCalls        *globals.ToolCalls    `json:"tool_calls,omitempty"`
	ReasoningContent *string               `json:"reasoning_content,omitempty"`
}

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
}

type ChatStreamErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}
