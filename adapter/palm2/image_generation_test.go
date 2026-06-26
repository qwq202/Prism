package palm2

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiInlineDataUsesOfficialJSONFields(t *testing.T) {
	body := GeminiChatBody{
		Contents: []GeminiContent{
			{
				Role: GeminiUserType,
				Parts: []GeminiChatPart{
					{
						InlineData: &GeminiInlineData{
							MimeType: "image/png",
							Data:     palm2InlineBase64Png,
						},
					},
				},
			},
		},
	}

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal gemini body: %v", err)
	}

	payload := string(raw)
	if !strings.Contains(payload, `"inlineData"`) || !strings.Contains(payload, `"mimeType"`) {
		t.Fatalf("expected official inlineData/mimeType fields, got %s", payload)
	}
	if strings.Contains(payload, "inline_data") || strings.Contains(payload, "mime_type") {
		t.Fatalf("did not expect legacy inline_data/mime_type fields, got %s", payload)
	}
}

func TestGeminiImageGenerationUsesInteractionsOutputImage(t *testing.T) {
	previousImageStore := globals.AcceptImageStore
	globals.AcceptImageStore = false
	defer func() {
		globals.AcceptImageStore = previousImageStore
	}()

	instance := NewChatInstance("https://generativelanguage.googleapis.com", "test-key")
	endpoint := instance.GetChatEndpoint(globals.Gemini31FlashImage, false)
	if endpoint != "https://generativelanguage.googleapis.com/v1beta/interactions" {
		t.Fatalf("expected interactions endpoint, got %s", endpoint)
	}
	if strings.Contains(endpoint, "generateContent") || strings.Contains(endpoint, "streamGenerateContent") {
		t.Fatalf("gemini image generation endpoint must use official interactions API: %s", endpoint)
	}

	body := instance.GetGeminiInteractionBody(&adaptercommon.ChatProps{
		Model: globals.Gemini31FlashImage,
		Message: []globals.Message{
			{Role: globals.User, Content: "draw a nano banana"},
		},
		ResponseFormat: map[string]interface{}{
			"type":         "image",
			"mime_type":    "image/jpeg",
			"aspect_ratio": "1:8",
			"image_size":   "512px",
		},
		Thinking: map[string]interface{}{
			"thinking_level": "high",
		},
	})
	if body.Model != globals.Gemini31FlashImage || body.Input != "draw a nano banana" {
		t.Fatalf("unexpected interactions body: %#v", body)
	}
	if body.ResponseFormat == nil ||
		body.ResponseFormat.MimeType != "image/jpeg" ||
		body.ResponseFormat.AspectRatio != "1:8" ||
		body.ResponseFormat.ImageSize != "512px" {
		t.Fatalf("expected official response_format, got %#v", body.ResponseFormat)
	}
	if body.GenerationConfig == nil || body.GenerationConfig.ThinkingLevel != "high" {
		t.Fatalf("expected thinking_level=high, got %#v", body.GenerationConfig)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(`{
		"output_text": "done",
		"output_image": {
			"mime_type": "image/png",
			"data": "`+palm2InlineBase64Png+`"
		}
	}`), &response); err != nil {
		t.Fatalf("unmarshal gemini response: %v", err)
	}

	chunk, err := instance.GetGeminiInteractionChunk(response)
	if err != nil {
		t.Fatalf("parse gemini image response: %v", err)
	}
	content := chunk.Content
	if !strings.Contains(content, "done") {
		t.Fatalf("expected text output, got %q", content)
	}
	expectedImage := "![image](data:image/png;base64," + palm2InlineBase64Png + ")"
	if !strings.Contains(content, expectedImage) {
		t.Fatalf("expected inline image markdown %q, got %q", expectedImage, content)
	}
}

func TestGemini3ProImageInteractionOptionsAreModelScoped(t *testing.T) {
	instance := NewChatInstance("https://generativelanguage.googleapis.com", "test-key")

	body := instance.GetGeminiInteractionBody(&adaptercommon.ChatProps{
		Model: globals.Gemini3ProImage,
		ResponseFormat: map[string]interface{}{
			"type":         "image",
			"mime_type":    "image/png",
			"aspect_ratio": "1:8",
			"image_size":   "512px",
		},
		Thinking: map[string]interface{}{
			"thinking_level": "high",
		},
	})
	if body.ResponseFormat == nil ||
		body.ResponseFormat.AspectRatio != "1:1" ||
		body.ResponseFormat.ImageSize != "1K" {
		t.Fatalf("expected gemini 3 pro defaults for unsupported options, got %#v", body.ResponseFormat)
	}
	if body.GenerationConfig != nil {
		t.Fatalf("gemini 3 pro image should not receive image thinking config, got %#v", body.GenerationConfig)
	}

	body = instance.GetGeminiInteractionBody(&adaptercommon.ChatProps{
		Model: globals.Gemini3ProImage,
		ResponseFormat: map[string]interface{}{
			"type":         "image",
			"aspect_ratio": "21:9",
			"image_size":   "4K",
		},
	})
	if body.ResponseFormat == nil ||
		body.ResponseFormat.AspectRatio != "21:9" ||
		body.ResponseFormat.ImageSize != "4K" {
		t.Fatalf("expected gemini 3 pro supported options, got %#v", body.ResponseFormat)
	}
}

func TestGeminiImageGenerationDowngradesUnsupportedInteractions(t *testing.T) {
	paths := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/v1beta/interactions":
			_, _ = w.Write([]byte(`{"error":{"message":"Invalid URL (POST /v1beta/interactions)","type":"invalid_request_error","param":"","code":""}}`))
		case "/v1beta/models/gemini-3.1-flash-image:generateContent":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read generateContent body: %v", err)
			}
			payload := string(body)
			if strings.Contains(payload, "systemInstruction") {
				t.Fatalf("image generateContent fallback should not include chat systemInstruction, got %s", payload)
			}
			if !strings.Contains(payload, `"responseModalities":["TEXT","IMAGE"]`) {
				t.Fatalf("expected image response modalities, got %s", payload)
			}
			_, _ = w.Write([]byte(`{
				"candidates": [
					{
						"content": {
							"parts": [
								{
									"inlineData": {
										"mimeType": "image/png",
										"data": "` + palm2InlineBase64Png + `"
									}
								}
							]
						}
					}
				]
			}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	instance := NewChatInstance(server.URL, "test-key")
	props := &adaptercommon.ChatProps{
		Model: globals.Gemini31FlashImage,
		Message: []globals.Message{
			{Role: globals.System, Content: "chat-only system prompt"},
			{Role: globals.User, Content: "draw a pig"},
		},
		ResponseFormat: map[string]interface{}{
			"type":         "image",
			"mime_type":    "image/png",
			"aspect_ratio": "1:1",
			"image_size":   "1K",
		},
	}

	chunk, err := instance.CreateGeminiChatRequest(props)
	if err != nil {
		t.Fatalf("expected generateContent downgrade, got error: %v", err)
	}
	if !strings.Contains(chunk.Content, "![image](data:image/png;base64,"+palm2InlineBase64Png+")") {
		t.Fatalf("expected downgraded image markdown, got %q", chunk.Content)
	}
	if len(paths) != 2 ||
		paths[0] != "/v1beta/interactions" ||
		paths[1] != "/v1beta/models/gemini-3.1-flash-image:generateContent" {
		t.Fatalf("expected interactions then generateContent, got %#v", paths)
	}

	paths = paths[:0]
	if _, err := instance.CreateGeminiChatRequest(props); err != nil {
		t.Fatalf("expected cached generateContent downgrade, got error: %v", err)
	}
	if len(paths) != 1 || paths[0] != "/v1beta/models/gemini-3.1-flash-image:generateContent" {
		t.Fatalf("expected remembered downgrade to skip interactions, got %#v", paths)
	}
}

func TestGemini25FlashImageInteractionOptionsOmitUnsupportedFields(t *testing.T) {
	instance := NewChatInstance("https://generativelanguage.googleapis.com", "test-key")

	body := instance.GetGeminiInteractionBody(&adaptercommon.ChatProps{
		Model: globals.Gemini25FlashImage,
		ResponseFormat: map[string]interface{}{
			"type":         "image",
			"aspect_ratio": "21:9",
			"image_size":   "4K",
		},
		Thinking: map[string]interface{}{
			"thinking_level": "high",
		},
	})
	if body.ResponseFormat == nil ||
		body.ResponseFormat.AspectRatio != "21:9" ||
		body.ResponseFormat.ImageSize != "" {
		t.Fatalf("expected gemini 2.5 flash image to omit image_size, got %#v", body.ResponseFormat)
	}
	if body.GenerationConfig != nil {
		t.Fatalf("gemini 2.5 flash image should not receive thinking config, got %#v", body.GenerationConfig)
	}

	body = instance.GetGeminiInteractionBody(&adaptercommon.ChatProps{
		Model: globals.Gemini25FlashImage,
		ResponseFormat: map[string]interface{}{
			"type":         "image",
			"aspect_ratio": "1:8",
		},
	})
	if body.ResponseFormat == nil || body.ResponseFormat.AspectRatio != "1:1" {
		t.Fatalf("expected unsupported gemini 2.5 ratio to fall back to 1:1, got %#v", body.ResponseFormat)
	}
}

func TestGeminiInteractionChunkParsesStepContent(t *testing.T) {
	instance := NewChatInstance("https://generativelanguage.googleapis.com", "test-key")

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(`{
		"steps": [
			{
				"type": "model_output",
				"content": [
					{"type": "output_text", "text": "ready"},
					{
						"type": "output_image",
						"mime_type": "image/png",
						"data": "`+palm2InlineBase64Png+`"
					}
				]
			}
		]
	}`), &response); err != nil {
		t.Fatalf("unmarshal gemini response: %v", err)
	}

	chunk, err := instance.GetGeminiInteractionChunk(response)
	if err != nil {
		t.Fatalf("parse gemini interaction response: %v", err)
	}

	if !strings.Contains(chunk.Content, "ready") {
		t.Fatalf("expected text output, got %q", chunk.Content)
	}
	if !strings.Contains(chunk.Content, "![image](data:image/png;base64,"+palm2InlineBase64Png+")") {
		t.Fatalf("expected image markdown, got %q", chunk.Content)
	}
}

func TestGeminiInteractionChunkParsesGenerateContentInlineData(t *testing.T) {
	instance := NewChatInstance("https://generativelanguage.googleapis.com", "test-key")

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(`{
		"candidates": [
			{
				"content": {
					"parts": [
						{"text": "ready"},
						{
							"inlineData": {
								"mimeType": "image/png",
								"data": "`+palm2InlineBase64Png+`"
							}
						}
					]
				}
			}
		],
		"usageMetadata": {
			"promptTokenCount": 12,
			"candidatesTokenCount": 8,
			"totalTokenCount": 20
		}
	}`), &response); err != nil {
		t.Fatalf("unmarshal gemini response: %v", err)
	}

	chunk, err := instance.GetGeminiInteractionChunk(response)
	if err != nil {
		t.Fatalf("parse gemini generateContent-shaped response: %v", err)
	}

	if !strings.Contains(chunk.Content, "ready") {
		t.Fatalf("expected text output, got %q", chunk.Content)
	}
	if !strings.Contains(chunk.Content, "![image](data:image/png;base64,"+palm2InlineBase64Png+")") {
		t.Fatalf("expected image markdown, got %q", chunk.Content)
	}
}

func TestGeminiInteractionChunkParsesNestedImageURL(t *testing.T) {
	instance := NewChatInstance("https://generativelanguage.googleapis.com", "test-key")

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(`{
		"output": [
			{
				"type": "message",
				"content": [
					{"type": "output_text", "text": "done"},
					{
						"type": "image_generation_call",
						"result": {
							"type": "output_image",
							"image_url": "https://example.com/result.webp"
						}
					}
				]
			}
		]
	}`), &response); err != nil {
		t.Fatalf("unmarshal gemini response: %v", err)
	}

	chunk, err := instance.GetGeminiInteractionChunk(response)
	if err != nil {
		t.Fatalf("parse gemini nested image response: %v", err)
	}

	if !strings.Contains(chunk.Content, "done") {
		t.Fatalf("expected text output, got %q", chunk.Content)
	}
	if !strings.Contains(chunk.Content, "![image](https://example.com/result.webp)") {
		t.Fatalf("expected image url markdown, got %q", chunk.Content)
	}
}

func TestGeminiInteractionChunkParseErrorIncludesResponseShape(t *testing.T) {
	instance := NewChatInstance("https://generativelanguage.googleapis.com", "test-key")

	_, err := instance.GetGeminiInteractionChunk(map[string]interface{}{
		"name": "operations/test",
		"metadata": map[string]interface{}{
			"state": "running",
		},
	})
	if err == nil {
		t.Fatalf("expected parse error")
	}
	if !strings.Contains(err.Error(), "keys=metadata:{state} name") {
		t.Fatalf("expected response shape in error, got %q", err.Error())
	}
}

func TestGeminiInteractionChunkParsesCompatibleError(t *testing.T) {
	instance := NewChatInstance("https://generativelanguage.googleapis.com", "test-key")

	_, err := instance.GetGeminiInteractionChunk(map[string]interface{}{
		"error": map[string]interface{}{
			"message": "Invalid URL (POST /v1beta/interactions)",
			"type":    "invalid_request_error",
			"param":   "",
			"code":    "",
		},
	})
	if err == nil {
		t.Fatalf("expected compatible error")
	}
	if !strings.Contains(err.Error(), "Invalid URL (POST /v1beta/interactions)") ||
		!strings.Contains(err.Error(), "invalid_request_error") {
		t.Fatalf("expected compatible error details, got %q", err.Error())
	}
}
