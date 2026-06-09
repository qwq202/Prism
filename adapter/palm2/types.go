package palm2

const (
	GeminiUserType  = "user"
	GeminiModelType = "model"
)

type PalmMessage struct {
	Author  string `json:"author"`
	Content string `json:"content"`
}

// PalmChatBody is the native http request body for palm2
type PalmChatBody struct {
	Prompt PalmPrompt `json:"prompt"`
}

type PalmPrompt struct {
	Messages []PalmMessage `json:"messages"`
}

// PalmChatResponse is the native http response body for palm2
type PalmChatResponse struct {
	Candidates []PalmMessage `json:"candidates"`
}

// GeminiChatBody is the native http request body for gemini
type GeminiChatBody struct {
	SystemInstruction *GeminiContent    `json:"systemInstruction,omitempty"`
	Contents          []GeminiContent   `json:"contents"`
	Tools             []GeminiTool      `json:"tools,omitempty"`
	ToolConfig        *GeminiToolConfig `json:"toolConfig,omitempty"`
	GenerationConfig  GeminiConfig      `json:"generationConfig"`
}

type GeminiConfig struct {
	Temperature     *float32              `json:"temperature,omitempty"`
	MaxOutputTokens *int                  `json:"maxOutputTokens,omitempty"`
	TopP            *float32              `json:"topP,omitempty"`
	TopK            *int                  `json:"topK,omitempty"`
	ThinkingConfig  *GeminiThinkingConfig `json:"thinkingConfig,omitempty"`
}

type GeminiThinkingConfig struct {
	ThinkingBudget  *int    `json:"thinkingBudget,omitempty"`
	ThinkingLevel   *string `json:"thinkingLevel,omitempty"`
	IncludeThoughts *bool   `json:"includeThoughts,omitempty"`
}

type GeminiContent struct {
	Role  string           `json:"role,omitempty"`
	Parts []GeminiChatPart `json:"parts"`
}

type GeminiChatPart struct {
	Text                *string                    `json:"text,omitempty"`
	InlineData          *GeminiInlineData          `json:"inline_data,omitempty"`
	FunctionCall        *GeminiFunctionCall        `json:"functionCall,omitempty"`
	FunctionResponse    *GeminiFunctionResponse    `json:"functionResponse,omitempty"`
	ExecutableCode      *GeminiExecutableCode      `json:"executableCode,omitempty"`
	CodeExecutionResult *GeminiCodeExecutionResult `json:"codeExecutionResult,omitempty"`
	Thought             bool                       `json:"thought,omitempty"`
	ThoughtSignature    *string                    `json:"thoughtSignature,omitempty"`
}

type GeminiInlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
	URLContext           *GeminiURLContext           `json:"url_context,omitempty"`
	GoogleSearch         *GeminiGoogleSearch         `json:"google_search,omitempty"`
	CodeExecution        *GeminiCodeExecution        `json:"code_execution,omitempty"`
}

type GeminiURLContext struct{}

type GeminiGoogleSearch struct{}

type GeminiCodeExecution struct{}

type GeminiExecutableCode struct {
	Language string `json:"language,omitempty"`
	Code     string `json:"code,omitempty"`
}

type GeminiCodeExecutionResult struct {
	Outcome string `json:"outcome,omitempty"`
	Output  string `json:"output,omitempty"`
}

type GeminiFunctionDeclaration struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type GeminiToolConfig struct {
	FunctionCallingConfig *GeminiFunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

type GeminiFunctionCallingConfig struct {
	Mode                 string   `json:"mode,omitempty"`
	AllowedFunctionNames []string `json:"allowedFunctionNames,omitempty"`
}

type GeminiFunctionCall struct {
	Name string      `json:"name"`
	Args interface{} `json:"args,omitempty"`
}

type GeminiFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

type GeminiChatResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

type GeminiChatErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

type GeminiStreamResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

// ImageRequest is the native http request body for imagen
type ImageRequest struct {
	Instances  []ImageInstance `json:"instances"`
	Parameters ImageParameters `json:"parameters"`
}

type ImageInstance struct {
	Prompt string `json:"prompt"`
}

type ImageParameters struct {
	SampleCount      int    `json:"sampleCount,omitempty"`
	AspectRatio      string `json:"aspectRatio,omitempty"`
	PersonGeneration string `json:"personGeneration,omitempty"`
}

// ImageResponse is the native http response body for imagen
type ImageResponse struct {
	Predictions []ImagePrediction `json:"predictions"`
}

type ImagePrediction struct {
	MimeType           string `json:"mimeType"`
	BytesBase64Encoded string `json:"bytesBase64Encoded"`
}
