package xai

type InputMessageContent struct {
	Type     string  `json:"type"`
	Text     *string `json:"text,omitempty"`
	ImageURL *string `json:"image_url,omitempty"`
}

type InputMessage struct {
	Role    string                `json:"role"`
	Content []InputMessageContent `json:"content"`
}

type FunctionCallOutputInput struct {
	Type   string `json:"type"`
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

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

type OutputContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type OutputItem struct {
	Type      string          `json:"type"`
	Role      string          `json:"role,omitempty"`
	Content   []OutputContent `json:"content,omitempty"`
	Name      string          `json:"name,omitempty"`
	Arguments string          `json:"arguments,omitempty"`
	CallID    string          `json:"call_id,omitempty"`
}

type ResponseResponse struct {
	ID     string       `json:"id"`
	Object string       `json:"object"`
	Model  string       `json:"model"`
	Output []OutputItem `json:"output"`
	Error  struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

type ResponseStreamEvent struct {
	Type  string      `json:"type"`
	Delta string      `json:"delta,omitempty"`
	Item  *OutputItem `json:"item,omitempty"`
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}
