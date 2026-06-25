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

func TestNormalizeTokenUsageUsesPromptTokensDetailsCachedTokens(t *testing.T) {
	usage := NormalizeTokenUsage(&TokenUsage{
		PromptTokens: 120,
		PromptTokensDetails: &PromptTokensDetails{
			CachedTokens: 96,
		},
	})

	if usage.PromptCacheHitTokens != 96 || usage.PromptCacheMissTokens != 24 {
		t.Fatalf("expected cached prompt details to become hit=96 miss=24, got %#v", usage)
	}
	if usage.PromptTokensDetails != nil {
		t.Fatalf("expected provider-specific prompt details to be normalized away, got %#v", usage.PromptTokensDetails)
	}
}
