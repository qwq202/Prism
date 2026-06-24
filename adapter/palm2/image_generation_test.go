package palm2

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"encoding/json"
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
			"aspect_ratio": "16:9",
			"image_size":   "2K",
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
		body.ResponseFormat.AspectRatio != "16:9" ||
		body.ResponseFormat.ImageSize != "2K" {
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
