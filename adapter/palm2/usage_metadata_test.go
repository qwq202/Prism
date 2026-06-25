package palm2

import (
	"encoding/json"
	"testing"
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
