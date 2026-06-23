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

func TestGeminiImageGenerationUsesGenerateContentInlineData(t *testing.T) {
	previousImageStore := globals.AcceptImageStore
	globals.AcceptImageStore = false
	defer func() {
		globals.AcceptImageStore = previousImageStore
	}()

	instance := NewChatInstance("https://generativelanguage.googleapis.com", "test-key")
	endpoint := instance.GetChatEndpoint(globals.Gemini31FlashImage, false)
	if endpoint != "https://generativelanguage.googleapis.com/v1/models/gemini-3.1-flash-image:generateContent?key=test-key" {
		t.Fatalf("expected generateContent endpoint, got %s", endpoint)
	}
	if strings.Contains(endpoint, "predict") || strings.Contains(endpoint, "streamGenerateContent") {
		t.Fatalf("gemini image generation endpoint must not use legacy predict or streaming: %s", endpoint)
	}

	body := instance.GetGeminiChatBody(&adaptercommon.ChatProps{
		Model: globals.Gemini31FlashImage,
		Message: []globals.Message{
			{Role: globals.User, Content: "draw a nano banana"},
		},
	})
	if len(body.GenerationConfig.ResponseModalities) != 2 ||
		body.GenerationConfig.ResponseModalities[0] != "TEXT" ||
		body.GenerationConfig.ResponseModalities[1] != "IMAGE" {
		t.Fatalf("expected TEXT/IMAGE response modalities, got %#v", body.GenerationConfig.ResponseModalities)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(`{
		"candidates": [
			{
				"content": {
					"parts": [
						{"text": "done"},
						{
							"inlineData": {
								"mimeType": "image/png",
								"data": "`+palm2InlineBase64Png+`"
							}
						}
					]
				}
			}
		]
	}`), &response); err != nil {
		t.Fatalf("unmarshal gemini response: %v", err)
	}

	chunk, err := instance.GetGeminiChunk(globals.Gemini31FlashImage, response)
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
