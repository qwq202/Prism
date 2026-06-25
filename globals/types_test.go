package globals

import "testing"

func TestNormalizeTokenUsageDerivesPromptCacheMissTokens(t *testing.T) {
	usage := NormalizeTokenUsage(&TokenUsage{
		PromptTokens:         100,
		CompletionTokens:     10,
		PromptCacheHitTokens: 40,
	})

	if usage.PromptCacheHitTokens != 40 || usage.PromptCacheMissTokens != 60 {
		t.Fatalf("expected hit=40 and derived miss=60, got %#v", usage)
	}
}

func TestNormalizeTokenUsageKeepsExplicitPromptCacheMissTokens(t *testing.T) {
	usage := NormalizeTokenUsage(&TokenUsage{
		PromptTokens:          100,
		PromptCacheHitTokens:  40,
		PromptCacheMissTokens: 5,
	})

	if usage.PromptCacheMissTokens != 5 {
		t.Fatalf("expected explicit cache miss tokens to be preserved, got %#v", usage)
	}
}
