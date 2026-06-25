package palm2

import "chat/globals"

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
	CachedContent     string            `json:"cachedContent,omitempty"`
	Tools             []GeminiTool      `json:"tools,omitempty"`
	ToolConfig        *GeminiToolConfig `json:"toolConfig,omitempty"`
	GenerationConfig  GeminiConfig      `json:"generationConfig"`
}

type GeminiInteractionBody struct {
	Model            string                             `json:"model"`
	Input            interface{}                        `json:"input"`
	ResponseFormat   *GeminiInteractionResponseFormat   `json:"response_format,omitempty"`
	GenerationConfig *GeminiInteractionGenerationConfig `json:"generation_config,omitempty"`
}

type GeminiInteractionResponseFormat struct {
	Type        string `json:"type,omitempty"`
	MimeType    string `json:"mime_type,omitempty"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
	ImageSize   string `json:"image_size,omitempty"`
}

type GeminiInteractionGenerationConfig struct {
	ThinkingLevel string `json:"thinking_level,omitempty"`
}

type GeminiConfig struct {
	Temperature        *float32              `json:"temperature,omitempty"`
	MaxOutputTokens    *int                  `json:"maxOutputTokens,omitempty"`
	TopP               *float32              `json:"topP,omitempty"`
	TopK               *int                  `json:"topK,omitempty"`
	ThinkingConfig     *GeminiThinkingConfig `json:"thinkingConfig,omitempty"`
	ResponseModalities []string              `json:"responseModalities,omitempty"`
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
	Text             *string                 `json:"text,omitempty"`
	InlineData       *GeminiInlineData       `json:"inlineData,omitempty"`
	FunctionCall     *GeminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"`
	Thought          bool                    `json:"thought,omitempty"`
	ThoughtSignature *string                 `json:"thoughtSignature,omitempty"`
}

type GeminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
	URLContext           *GeminiURLContext           `json:"url_context,omitempty"`
	GoogleSearch         *GeminiGoogleSearch         `json:"google_search,omitempty"`
}

type GeminiURLContext struct{}

type GeminiGoogleSearch struct{}

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
	Candidates         []GeminiCandidate    `json:"candidates"`
	UsageMetadata      *GeminiUsageMetadata `json:"usageMetadata,omitempty"`
	UsageMetadataSnake *GeminiUsageMetadata `json:"usage_metadata,omitempty"`
}

type GeminiInteractionImage struct {
	Data          string `json:"data,omitempty"`
	MimeType      string `json:"mime_type,omitempty"`
	MimeTypeCamel string `json:"mimeType,omitempty"`
	URL           string `json:"url,omitempty"`
}

type GeminiInteractionContent struct {
	Type          string `json:"type,omitempty"`
	Text          string `json:"text,omitempty"`
	Data          string `json:"data,omitempty"`
	MimeType      string `json:"mime_type,omitempty"`
	MimeTypeCamel string `json:"mimeType,omitempty"`
	URL           string `json:"url,omitempty"`
}

type GeminiInteractionStep struct {
	Type    string                     `json:"type,omitempty"`
	Content []GeminiInteractionContent `json:"content,omitempty"`
}

type GeminiInteractionResponse struct {
	OutputText         string                     `json:"output_text,omitempty"`
	Text               string                     `json:"text,omitempty"`
	OutputImage        *GeminiInteractionImage    `json:"output_image,omitempty"`
	GeneratedImages    []GeminiInteractionImage   `json:"generated_images,omitempty"`
	Output             []GeminiInteractionContent `json:"output,omitempty"`
	Steps              []GeminiInteractionStep    `json:"steps,omitempty"`
	UsageMetadata      *GeminiUsageMetadata       `json:"usageMetadata,omitempty"`
	UsageMetadataSnake *GeminiUsageMetadata       `json:"usage_metadata,omitempty"`
}

type GeminiChatErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

type GeminiStreamResponse struct {
	Candidates         []GeminiCandidate    `json:"candidates"`
	UsageMetadata      *GeminiUsageMetadata `json:"usageMetadata,omitempty"`
	UsageMetadataSnake *GeminiUsageMetadata `json:"usage_metadata,omitempty"`
}

type GeminiUsageMetadata struct {
	PromptTokenCount             int `json:"promptTokenCount,omitempty"`
	PromptTokenCountSnake        int `json:"prompt_token_count,omitempty"`
	CandidatesTokenCount         int `json:"candidatesTokenCount,omitempty"`
	CandidatesTokenCountSnake    int `json:"candidates_token_count,omitempty"`
	TotalTokenCount              int `json:"totalTokenCount,omitempty"`
	TotalTokenCountSnake         int `json:"total_token_count,omitempty"`
	CachedContentTokenCount      int `json:"cachedContentTokenCount,omitempty"`
	CachedContentTokenCountSnake int `json:"cached_content_token_count,omitempty"`
	ThoughtsTokenCount           int `json:"thoughtsTokenCount,omitempty"`
	ThoughtsTokenCountSnake      int `json:"thoughts_token_count,omitempty"`
}

func (m *GeminiUsageMetadata) TokenUsage() *globals.TokenUsage {
	if m == nil {
		return nil
	}

	promptTokens := firstGeminiUsageValue(m.PromptTokenCount, m.PromptTokenCountSnake)
	cachedContentTokens := firstGeminiUsageValue(m.CachedContentTokenCount, m.CachedContentTokenCountSnake)
	if promptTokens == 0 && cachedContentTokens > 0 {
		promptTokens = cachedContentTokens
	}

	return globals.NormalizeTokenUsage(&globals.TokenUsage{
		PromptTokens:         promptTokens,
		CompletionTokens:     firstGeminiUsageValue(m.CandidatesTokenCount, m.CandidatesTokenCountSnake),
		TotalTokens:          firstGeminiUsageValue(m.TotalTokenCount, m.TotalTokenCountSnake),
		PromptCacheHitTokens: cachedContentTokens,
		CompletionTokensDetails: globals.CompletionTokensDetails{
			ReasoningTokens: firstGeminiUsageValue(m.ThoughtsTokenCount, m.ThoughtsTokenCountSnake),
		},
	})
}

func firstGeminiUsageValue(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
