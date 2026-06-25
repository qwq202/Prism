package palm2

import (
	"bytes"
	"encoding/json"
	"testing"

	adaptercommon "chat/adapter/common"
	"chat/globals"
)

func TestGeminiChunkParsesOfficialCacheUsageMetadata(t *testing.T) {
	instance := NewChatInstance("https://generativelanguage.googleapis.com", "test-key")

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(`{
		"candidates": [
			{
				"content": {
					"parts": [{"text": "cached"}]
				}
			}
		],
		"usageMetadata": {
			"promptTokenCount": 100,
			"cachedContentTokenCount": 40,
			"candidatesTokenCount": 8,
			"totalTokenCount": 108
		}
	}`), &response); err != nil {
		t.Fatalf("unmarshal gemini response: %v", err)
	}

	chunk, err := instance.GetGeminiChunk("", response)
	if err != nil {
		t.Fatalf("parse gemini response: %v", err)
	}
	if chunk.Usage == nil {
		t.Fatalf("expected usage metadata")
	}
	if chunk.Usage.PromptTokens != 100 ||
		chunk.Usage.PromptCacheHitTokens != 40 ||
		chunk.Usage.PromptCacheMissTokens != 60 ||
		chunk.Usage.CompletionTokens != 8 ||
		chunk.Usage.TotalTokens != 108 {
		t.Fatalf("unexpected usage metadata: %#v", chunk.Usage)
	}
}

func TestGeminiChunkParsesSnakeCaseCacheUsageMetadata(t *testing.T) {
	instance := NewChatInstance("https://generativelanguage.googleapis.com", "test-key")

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(`{
		"usage_metadata": {
			"prompt_token_count": 75,
			"cached_content_token_count": 25,
			"candidates_token_count": 5,
			"total_token_count": 80
		}
	}`), &response); err != nil {
		t.Fatalf("unmarshal gemini response: %v", err)
	}

	chunk, err := instance.GetGeminiChunk("", response)
	if err != nil {
		t.Fatalf("parse gemini response: %v", err)
	}
	if chunk.Usage == nil {
		t.Fatalf("expected usage metadata")
	}
	if chunk.Usage.PromptTokens != 75 ||
		chunk.Usage.PromptCacheHitTokens != 25 ||
		chunk.Usage.PromptCacheMissTokens != 50 ||
		chunk.Usage.CompletionTokens != 5 ||
		chunk.Usage.TotalTokens != 80 {
		t.Fatalf("unexpected usage metadata: %#v", chunk.Usage)
	}
}

func TestGeminiChatBodyIncludesCachedContent(t *testing.T) {
	instance := NewChatInstance("https://generativelanguage.googleapis.com", "test-key")
	cacheName := " cachedContents/test-cache "

	body := instance.GetGeminiChatBody(&adaptercommon.ChatProps{
		Model:         "gemini-2.5-flash",
		Message:       []globals.Message{{Role: globals.User, Content: "summarize"}},
		CachedContent: &cacheName,
	})

	if body.CachedContent != "cachedContents/test-cache" {
		t.Fatalf("expected cachedContent to be normalized into request body, got %q", body.CachedContent)
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal gemini body: %v", err)
	}
	if !json.Valid(data) || !bytes.Contains(data, []byte(`"cachedContent":"cachedContents/test-cache"`)) {
		t.Fatalf("expected marshaled body to include cachedContent, got %s", string(data))
	}
}

func TestGeminiChatBodyAcceptsSnakeCaseCachedContent(t *testing.T) {
	instance := NewChatInstance("https://generativelanguage.googleapis.com", "test-key")
	cacheName := "projects/123/locations/us-central1/cachedContents/456"

	body := instance.GetGeminiChatBody(&adaptercommon.ChatProps{
		Model:              "gemini-2.5-flash",
		Message:            []globals.Message{{Role: globals.User, Content: "summarize"}},
		CachedContentSnake: &cacheName,
	})

	if body.CachedContent != cacheName {
		t.Fatalf("expected snake_case cached_content to populate cachedContent, got %q", body.CachedContent)
	}
}

func TestGeminiCacheCreationUsageKeepsTotalOnly(t *testing.T) {
	usage := (&GeminiUsageMetadata{TotalTokenCount: 43125}).TokenUsage()

	if usage == nil {
		t.Fatalf("expected usage metadata")
	}
	if usage.TotalTokens != 43125 ||
		usage.PromptTokens != 0 ||
		usage.PromptCacheHitTokens != 0 ||
		usage.PromptCacheMissTokens != 0 {
		t.Fatalf("unexpected cache creation usage metadata: %#v", usage)
	}
}
