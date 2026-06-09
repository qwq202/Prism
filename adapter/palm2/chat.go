package palm2

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var geminiMaxImages = 16

const defaultVertexAIExpressEndpoint = "https://aiplatform.googleapis.com"

func getGeminiAPIVersion(model string) string {
	if strings.Contains(model, "preview") || strings.Contains(model, "exp") || strings.Contains(model, "latest") {
		return "v1beta"
	}

	return "v1"
}

func getVertexAIExpressAPIVersion(model string) string {
	if getGeminiAPIVersion(model) == "v1beta" {
		return "v1beta1"
	}

	return "v1"
}

func getVertexAIExpressModelPath(model string) string {
	model = strings.Trim(strings.TrimSpace(model), "/")
	if strings.HasPrefix(model, "publishers/") || strings.HasPrefix(model, "projects/") {
		return model
	}

	model = strings.TrimPrefix(model, "models/")
	return fmt.Sprintf("publishers/google/models/%s", model)
}

func (c *ChatInstance) GetVertexAIExpressChatEndpoint(model string, stream bool) string {
	endpoint := strings.TrimRight(strings.TrimSpace(c.Endpoint), "/")
	if endpoint == "" {
		endpoint = defaultVertexAIExpressEndpoint
	}

	action := "generateContent"
	if stream {
		action = "streamGenerateContent"
	}

	values := url.Values{}
	if stream {
		values.Set("alt", "sse")
	}
	values.Set("key", c.ApiKey)

	return fmt.Sprintf(
		"%s/%s/%s:%s?%s",
		endpoint,
		getVertexAIExpressAPIVersion(model),
		getVertexAIExpressModelPath(model),
		action,
		values.Encode(),
	)
}

func (c *ChatInstance) GetChatEndpoint(model string, stream bool) string {
	if c.VertexAIExpress {
		return c.GetVertexAIExpressChatEndpoint(model, stream)
	}

	if model == globals.ChatBison001 {
		return fmt.Sprintf("%s/v1beta2/models/%s:generateMessage?key=%s", c.Endpoint, model, c.ApiKey)
	}

	version := getGeminiAPIVersion(model)

	if stream {
		return fmt.Sprintf("%s/%s/models/%s:streamGenerateContent?alt=sse&key=%s", c.Endpoint, version, model, c.ApiKey)
	}

	return fmt.Sprintf("%s/%s/models/%s:generateContent?key=%s", c.Endpoint, version, model, c.ApiKey)
}

func (c *ChatInstance) ConvertMessage(message []globals.Message) []PalmMessage {
	var result []PalmMessage
	for i, item := range message {
		if len(item.Content) == 0 {
			// palm model: message must include non empty content
			continue
		}

		if item.Role == globals.Tool {
			continue
		}

		if i > 0 && item.Role == result[len(result)-1].Author {
			// palm model: messages must alternate between authors
			result[len(result)-1].Content += " " + item.Content
			continue
		}

		result = append(result, PalmMessage{
			Author:  item.Role,
			Content: item.Content,
		})
	}
	return result
}

func (c *ChatInstance) GetPalm2ChatBody(props *adaptercommon.ChatProps) *PalmChatBody {
	return &PalmChatBody{
		Prompt: PalmPrompt{
			Messages: c.ConvertMessage(props.Message),
		},
	}
}

func (c *ChatInstance) GetGeminiChatBody(props *adaptercommon.ChatProps) *GeminiChatBody {
	enableCodeExecution := props.EnableCodeExecution && globals.SupportGeminiCodeExecution(props.Model)

	return &GeminiChatBody{
		SystemInstruction: c.GetGeminiSystemInstruction(props.Model, props.Message),
		Contents:          c.GetGeminiContents(props.Model, props.Message),
		Tools:             mergeGeminiTools(getGeminiBuiltinTools(props.EnableWebSearch, props.EnableURLContext, enableCodeExecution), getGeminiTools(props.Tools)),
		ToolConfig:        getGeminiToolConfig(props.ToolChoice),
		GenerationConfig: GeminiConfig{
			Temperature:     props.Temperature,
			MaxOutputTokens: props.MaxTokens,
			TopP:            props.TopP,
			TopK:            props.TopK,
			ThinkingConfig:  getGeminiThinkingConfig(props),
		},
	}
}

func (c *ChatInstance) GetPalm2ChatResponse(data interface{}) (string, error) {
	if form := utils.MapToStruct[PalmChatResponse](data); form != nil {
		if len(form.Candidates) == 0 {
			return "", fmt.Errorf("palm2 error: the content violates content policy")
		}
		return form.Candidates[0].Content, nil
	}
	return "", fmt.Errorf("palm2 error: cannot parse response")
}

func (c *ChatInstance) buildGeminiChunk(model string, parts []GeminiChatPart, stream bool) *globals.Chunk {
	content := c.GetGeminiChatText(model, parts)
	if stream {
		content = c.GetGeminiStreamText(model, parts)
	}

	return &globals.Chunk{
		Content:              content,
		ToolCall:             getGeminiToolCalls(parts),
		GeminiHiddenMetadata: getGeminiHiddenMetadataFromParts(parts),
	}
}

func (c *ChatInstance) GetGeminiChunk(model string, data interface{}) (*globals.Chunk, error) {
	if form := utils.MapToStruct[GeminiChatResponse](data); form != nil {
		if len(form.Candidates) != 0 {
			parts := form.Candidates[0].Content.Parts
			return c.buildGeminiChunk(model, parts, false), nil
		}
	}

	if form := utils.MapToStruct[GeminiChatErrorResponse](data); form != nil {
		return nil, fmt.Errorf("gemini error: %s (code: %d, status: %s)", form.Error.Message, form.Error.Code, form.Error.Status)
	}

	return nil, fmt.Errorf("gemini: cannot parse response")
}

func (c *ChatInstance) GetGeminiChatResponse(data interface{}) (string, error) {
	chunk, err := c.GetGeminiChunk("", data)
	if err != nil {
		return "", err
	}

	return chunk.Content, nil
}

func (c *ChatInstance) CreateChatRequest(props *adaptercommon.ChatProps) (string, error) {
	uri := c.GetChatEndpoint(props.Model, false)

	if props.Model == globals.ChatBison001 {
		data, err := utils.Post(uri, map[string]string{
			"Content-Type": "application/json",
		}, c.GetPalm2ChatBody(props), props.Proxy)

		if err != nil {
			return "", fmt.Errorf("palm2 error: %s", err.Error())
		}
		return c.GetPalm2ChatResponse(data)
	}

	chunk, err := c.CreateGeminiChatRequest(props)
	if err != nil {
		return "", err
	}

	return chunk.Content, nil
}

// CreateStreamChatRequest is the stream request for palm2
func (c *ChatInstance) CreateStreamChatRequest(props *adaptercommon.ChatProps, callback globals.Hook) error {
	// Handle imagen models
	if globals.IsGoogleImagenModel(props.Model) {
		response, err := c.CreateImage(props)
		if err != nil {
			return err
		}
		return callback(&globals.Chunk{Content: response})
	}

	// Handle chat models
	if props.Model == globals.ChatBison001 {
		response, err := c.CreateChatRequest(props)
		if err != nil {
			return err
		}

		for _, item := range utils.SplitItem(response, " ") {
			if err := callback(&globals.Chunk{Content: item}); err != nil {
				return err
			}
		}
		return nil
	}

	ticks := 0
	c.isFirstReasoning = true
	c.isReasonOver = false
	scanErr := utils.EventScanner(&utils.EventScannerProps{
		Method: "POST",
		Uri:    c.GetChatEndpoint(props.Model, true),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: c.GetGeminiChatBody(props),
		Callback: func(data string) error {
			ticks += 1

			if form := utils.UnmarshalForm[GeminiStreamResponse](data); form != nil {
				if len(form.Candidates) != 0 && len(form.Candidates[0].Content.Parts) != 0 {
					parts := form.Candidates[0].Content.Parts
					return callback(c.buildGeminiChunk(props.Model, parts, true))
				}
				return nil
			}

			if form := utils.UnmarshalForm[GeminiChatErrorResponse](data); form != nil {
				return fmt.Errorf("gemini error: %s (code: %d, status: %s)", form.Error.Message, form.Error.Code, form.Error.Status)
			}

			return nil
		},
	}, props.Proxy)

	if scanErr != nil {
		if scanErr.Error != nil && strings.Contains(scanErr.Error.Error(), "status code: 404") {
			// downgrade to non-stream request
			chunk, err := c.CreateGeminiChatRequest(props)
			if err != nil {
				return err
			}
			return callback(chunk)
		}

		if scanErr.Body != "" {
			if form := utils.UnmarshalForm[GeminiChatErrorResponse](scanErr.Body); form != nil {
				return fmt.Errorf("gemini error: %s (code: %d, status: %s)", form.Error.Message, form.Error.Code, form.Error.Status)
			}
			return fmt.Errorf("gemini error: %s", scanErr.Body)
		}
		return fmt.Errorf("gemini error: %v", scanErr.Error)
	}

	if ticks == 0 {
		return errors.New("no response")
	}

	if !c.isFirstReasoning && !c.isReasonOver {
		if err := callback(&globals.Chunk{Content: "\n</think>\n\n"}); err != nil {
			return err
		}
	}

	return nil
}

func (c *ChatInstance) CreateGeminiChatRequest(props *adaptercommon.ChatProps) (*globals.Chunk, error) {
	data, err := utils.Post(c.GetChatEndpoint(props.Model, false), map[string]string{
		"Content-Type": "application/json",
	}, c.GetGeminiChatBody(props), props.Proxy)

	if err != nil {
		return nil, fmt.Errorf("gemini error: %s", err.Error())
	}

	return c.GetGeminiChunk(props.Model, data)
}

func (c *ChatInstance) GetLatestPrompt(props *adaptercommon.ChatProps) string {
	if len(props.Message) == 0 {
		return ""
	}
	return props.Message[len(props.Message)-1].Content
}
