package xai

import compat "chat/adapter/responsescompat"

type InputMessageContent = compat.InputMessageContent

type InputMessage = compat.InputMessage

type FunctionCallOutputInput = compat.FunctionCallOutputInput

type ResponseTool struct {
	Type                     string      `json:"type"`
	Name                     string      `json:"name,omitempty"`
	Description              string      `json:"description,omitempty"`
	Parameters               interface{} `json:"parameters,omitempty"`
	Filters                  interface{} `json:"filters,omitempty"`
	AllowedXHandles          []string    `json:"allowed_x_handles,omitempty"`
	ExcludedXHandles         []string    `json:"excluded_x_handles,omitempty"`
	FromDate                 *string     `json:"from_date,omitempty"`
	ToDate                   *string     `json:"to_date,omitempty"`
	EnableImageUnderstanding *bool       `json:"enable_image_understanding,omitempty"`
	EnableVideoUnderstanding *bool       `json:"enable_video_understanding,omitempty"`
}

type ResponseRequest struct {
	Model              string         `json:"model"`
	Input              interface{}    `json:"input"`
	MaxOutputTokens    *int           `json:"max_output_tokens,omitempty"`
	Temperature        *float32       `json:"temperature,omitempty"`
	TopP               *float32       `json:"top_p,omitempty"`
	Tools              []ResponseTool `json:"tools,omitempty"`
	ToolChoice         *interface{}   `json:"tool_choice,omitempty"`
	ParallelToolCalls  *bool          `json:"parallel_tool_calls,omitempty"`
	ResponseFormat     interface{}    `json:"response_format,omitempty"`
	Include            []string       `json:"include,omitempty"`
	PreviousResponseID *string        `json:"previous_response_id,omitempty"`
	Store              *bool          `json:"store,omitempty"`
	Stream             bool           `json:"stream,omitempty"`
}

type OutputContent = compat.OutputContent

type OutputItem = compat.OutputItem

type ResponseResponse struct {
	ID     string                `json:"id"`
	Object string                `json:"object"`
	Model  string                `json:"model"`
	Output []OutputItem          `json:"output"`
	Usage  *compat.ResponseUsage `json:"usage,omitempty"`
	Error  struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

type ResponseStreamEvent struct {
	Type     string            `json:"type"`
	Delta    string            `json:"delta,omitempty"`
	Item     *OutputItem       `json:"item,omitempty"`
	Response *ResponseResponse `json:"response,omitempty"`
	Error    struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}
