package azure

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"testing"
)

func TestGetChatResponseContentRejectsEmptyChoices(t *testing.T) {
	if _, err := getChatResponseContent(&ChatResponse{}); err == nil {
		t.Fatalf("expected empty choices to return an error")
	}
}

func TestGetChatBodyIncludesPromptCacheParams(t *testing.T) {
	instance := NewChatInstance("2025-01-01-preview", "sk-test", "https://azure.example.com")
	cacheKey := "stable-prefix"
	cacheRetention := "24h"

	body, ok := instance.GetChatBody(&adaptercommon.ChatProps{
		Model:                "gpt-5.1",
		Message:              []globals.Message{{Role: globals.User, Content: "hello"}},
		PromptCacheKey:       &cacheKey,
		PromptCacheRetention: &cacheRetention,
	}, false).(ChatRequest)
	if !ok {
		t.Fatalf("expected ChatRequest body")
	}
	if body.PromptCacheKey == nil || *body.PromptCacheKey != cacheKey {
		t.Fatalf("expected prompt_cache_key to be included, got %#v", body.PromptCacheKey)
	}
	if body.PromptCacheRetention == nil || *body.PromptCacheRetention != cacheRetention {
		t.Fatalf("expected prompt_cache_retention to be included, got %#v", body.PromptCacheRetention)
	}
}

func TestProcessLineNormalizesPromptCacheUsage(t *testing.T) {
	chunk, err := (&ChatInstance{}).ProcessLine(`{"choices":[],"usage":{"prompt_tokens":80,"completion_tokens":8,"total_tokens":88,"prompt_tokens_details":{"cached_tokens":48}}}`, false)
	if err != nil {
		t.Fatalf("expected usage-only stream chunk to parse, got %v", err)
	}
	if chunk.Usage == nil {
		t.Fatalf("expected usage chunk")
	}
	if chunk.Usage.PromptCacheHitTokens != 48 || chunk.Usage.PromptCacheMissTokens != 32 {
		t.Fatalf("expected cached prompt usage to be normalized, got %#v", chunk.Usage)
	}
	if chunk.Usage.PromptTokensDetails != nil {
		t.Fatalf("expected provider-specific prompt details to be normalized away, got %#v", chunk.Usage.PromptTokensDetails)
	}
}
