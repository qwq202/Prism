package claude

import "chat/globals"

type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

type ContentBlock struct {
	Type      string                 `json:"type"`
	Text      *string                `json:"text,omitempty"`
	Thinking  *string                `json:"thinking,omitempty"`
	Signature *string                `json:"signature,omitempty"`
	Source    *MessageImage          `json:"source,omitempty"`
	ID        *string                `json:"id,omitempty"`
	Name      *string                `json:"name,omitempty"`
	Input     interface{}            `json:"input,omitempty"`
	ToolUseID *string                `json:"tool_use_id,omitempty"`
	Content   interface{}            `json:"content,omitempty"`
	IsError   *bool                  `json:"is_error,omitempty"`
	CacheCtrl map[string]interface{} `json:"cache_control,omitempty"`
}

type MessageImage struct {
	Type      string      `json:"type"`
	MediaType interface{} `json:"media_type"`
	Data      interface{} `json:"data"`
}

type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"input_schema"`
}

type ToolChoice struct {
	Type string  `json:"type"`
	Name *string `json:"name,omitempty"`
}

type ChatBody struct {
	Messages    []Message        `json:"messages"`
	MaxTokens   int              `json:"max_tokens"`
	Model       string           `json:"model"`
	System      string           `json:"system,omitempty"`
	Stream      bool             `json:"stream"`
	Temperature *float32         `json:"temperature,omitempty"`
	TopP        *float32         `json:"top_p,omitempty"`
	TopK        *int             `json:"top_k,omitempty"`
	StopSeqs    interface{}      `json:"stop_sequences,omitempty"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	ToolChoice  *ToolChoice      `json:"tool_choice,omitempty"`
	Thinking    interface{}      `json:"thinking,omitempty"`
}

type StreamEvent struct {
	Type         string             `json:"type"`
	Index        int                `json:"index,omitempty"`
	ContentBlock *ContentBlock      `json:"content_block,omitempty"`
	Delta        *ContentBlockDelta `json:"delta,omitempty"`
	Error        *ChatError         `json:"error,omitempty"`
	Message      *ResponseMessage   `json:"message,omitempty"`
}

type ContentBlockDelta struct {
	Type         string  `json:"type,omitempty"`
	Text         *string `json:"text,omitempty"`
	Thinking     *string `json:"thinking,omitempty"`
	PartialJSON  *string `json:"partial_json,omitempty"`
	Signature    *string `json:"signature,omitempty"`
	StopReason   *string `json:"stop_reason,omitempty"`
	StopSequence *string `json:"stop_sequence,omitempty"`
}

type ResponseMessage struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Content    []ContentBlock `json:"content"`
	StopReason *string        `json:"stop_reason,omitempty"`
}

type ChatResponse struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Content    []ContentBlock `json:"content"`
	StopReason *string        `json:"stop_reason,omitempty"`
	Error      *ChatError     `json:"error,omitempty"`
}

type ChatError struct {
	Type    string `json:"type" binding:"required"`
	Message string `json:"message"`
}

type ChatErrorResponse struct {
	Error ChatError `json:"error"`
}

func toToolCall(id string, name string, input string, index int) globals.ToolCall {
	return globals.ToolCall{
		Index: &index,
		Type:  "function",
		Id:    id,
		Function: globals.ToolCallFunction{
			Name:      name,
			Arguments: input,
		},
	}
}
