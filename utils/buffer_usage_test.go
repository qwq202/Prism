package utils

import (
	"chat/globals"
	"math"
	"strings"
	"testing"
)

type usageTestCharge struct{}

func (usageTestCharge) GetType() string             { return globals.TokenBilling }
func (usageTestCharge) GetModels() []string         { return nil }
func (usageTestCharge) GetInput() float32           { return 1 }
func (usageTestCharge) GetOutput() float32          { return 2 }
func (usageTestCharge) SupportAnonymous() bool      { return true }
func (usageTestCharge) IsBilling() bool             { return true }
func (usageTestCharge) IsBillingType(t string) bool { return t == globals.TokenBilling }
func (usageTestCharge) GetLimit() float32           { return 0 }

type usageCacheTestCharge struct {
	hit  float32
	miss float32
}

func (usageCacheTestCharge) GetType() string             { return globals.TokenBilling }
func (usageCacheTestCharge) GetModels() []string         { return nil }
func (usageCacheTestCharge) GetInput() float32           { return 1 }
func (usageCacheTestCharge) GetOutput() float32          { return 2 }
func (usageCacheTestCharge) SupportAnonymous() bool      { return true }
func (usageCacheTestCharge) IsBilling() bool             { return true }
func (usageCacheTestCharge) IsBillingType(t string) bool { return t == globals.TokenBilling }
func (usageCacheTestCharge) GetLimit() float32           { return 0 }
func (c usageCacheTestCharge) GetCacheHit() (float32, bool) {
	return c.hit, true
}
func (c usageCacheTestCharge) GetCacheMiss() (float32, bool) {
	return c.miss, true
}

func TestBufferRecordsOfficialUsage(t *testing.T) {
	buffer := &Buffer{}
	buffer.WriteChunk(&globals.Chunk{
		Content: "hello",
		Usage: &globals.TokenUsage{
			PromptTokens:          30,
			CompletionTokens:      7,
			TotalTokens:           37,
			PromptCacheHitTokens:  20,
			PromptCacheMissTokens: 10,
			CompletionTokensDetails: globals.CompletionTokensDetails{
				ReasoningTokens: 3,
			},
		},
	})

	usage := buffer.GetUsage()
	if usage == nil {
		t.Fatalf("expected usage to be recorded")
	}
	if usage.PromptCacheHitTokens != 20 || usage.PromptCacheMissTokens != 10 {
		t.Fatalf("unexpected prompt cache usage: %#v", usage)
	}
	if usage.CompletionTokensDetails.ReasoningTokens != 3 {
		t.Fatalf("expected reasoning tokens to be recorded, got %#v", usage.CompletionTokensDetails)
	}

	detail := buffer.GetBillingDetail()
	if !strings.Contains(detail, "prompt_cache_hit_tokens") || !strings.Contains(detail, "prompt_cache_miss_tokens") {
		t.Fatalf("expected billing detail to include cache usage, got %q", detail)
	}
}

func TestBufferNormalizesOfficialPromptTokenDetails(t *testing.T) {
	buffer := &Buffer{}
	buffer.WriteChunk(&globals.Chunk{
		Usage: &globals.TokenUsage{
			PromptTokens:     30,
			CompletionTokens: 7,
			PromptTokensDetails: &globals.PromptTokensDetails{
				CachedTokens: 12,
			},
		},
	})

	usage := buffer.GetUsage()
	if usage == nil {
		t.Fatalf("expected usage to be recorded")
	}
	if usage.PromptCacheHitTokens != 12 {
		t.Fatalf("expected cached prompt tokens to normalize as cache-hit tokens, got %#v", usage)
	}
	if usage.PromptTokensDetails != nil {
		t.Fatalf("expected provider-specific prompt token details to be cleared, got %#v", usage.PromptTokensDetails)
	}
	if usage.TotalTokens != 37 {
		t.Fatalf("expected total tokens to be derived, got %#v", usage)
	}

	detail := buffer.GetBillingDetail()
	if !strings.Contains(detail, `"prompt_cache_hit_tokens":12`) {
		t.Fatalf("expected billing detail to include normalized cache-hit tokens, got %q", detail)
	}
	if strings.Contains(detail, "prompt_tokens_details") {
		t.Fatalf("expected billing detail to omit provider-specific token details, got %q", detail)
	}
}

func TestBufferMergesOfficialUsage(t *testing.T) {
	target := &Buffer{}
	source := &Buffer{}

	target.WriteChunk(&globals.Chunk{
		Usage: &globals.TokenUsage{
			PromptTokens:          10,
			CompletionTokens:      2,
			TotalTokens:           12,
			PromptCacheHitTokens:  6,
			PromptCacheMissTokens: 4,
		},
	})
	source.WriteChunk(&globals.Chunk{
		Usage: &globals.TokenUsage{
			PromptTokens:          20,
			CompletionTokens:      3,
			TotalTokens:           23,
			PromptCacheHitTokens:  15,
			PromptCacheMissTokens: 5,
		},
	})

	target.MergeUsage(source)
	usage := target.GetUsage()
	if usage == nil {
		t.Fatalf("expected merged usage")
	}
	if usage.PromptTokens != 30 || usage.CompletionTokens != 5 || usage.TotalTokens != 35 {
		t.Fatalf("unexpected merged token totals: %#v", usage)
	}
	if usage.PromptCacheHitTokens != 21 || usage.PromptCacheMissTokens != 9 {
		t.Fatalf("unexpected merged cache tokens: %#v", usage)
	}
}

func TestBufferRecordQuotaUsesOfficialUsageWhenPresent(t *testing.T) {
	buffer := NewBuffer(globals.GPT3Turbo, nil, usageTestCharge{})
	buffer.Write("visible")
	buffer.SetUsage(&globals.TokenUsage{
		PromptTokens:     1000,
		CompletionTokens: 2000,
		TotalTokens:      3000,
	})

	if got := buffer.CountRecordInputToken(); got != 1000 {
		t.Fatalf("expected official prompt tokens, got %d", got)
	}
	if got := buffer.CountRecordOutputToken(); got != 2000 {
		t.Fatalf("expected official completion tokens, got %d", got)
	}
	if got := buffer.GetRecordQuota(); math.Abs(float64(got-5)) > 0.001 {
		t.Fatalf("expected official usage quota 5, got %f", got)
	}
}

func TestBufferRecordQuotaUsesCacheTokenPrices(t *testing.T) {
	buffer := NewBuffer(globals.GPT3Turbo, nil, usageCacheTestCharge{
		hit:  0.2,
		miss: 0.6,
	})
	buffer.Write("visible")
	buffer.SetUsage(&globals.TokenUsage{
		PromptTokens:          1000,
		CompletionTokens:      1000,
		TotalTokens:           2000,
		PromptCacheHitTokens:  300,
		PromptCacheMissTokens: 200,
	})

	// input: 500 regular * 1 + 300 cache-hit * 0.2 + 200 cache-miss * 0.6 = 0.68
	// output: 1000 * 2 = 2
	if got := buffer.GetRecordQuota(); math.Abs(float64(got-2.68)) > 0.001 {
		t.Fatalf("expected cache-aware quota 2.68, got %f", got)
	}
}

func TestNewBufferDoesNotCountBase64FileContentForConfiguredVisionModel(t *testing.T) {
	originalResolver := globals.VisionModelResolver
	globals.VisionModelResolver = func(model string) bool {
		return model == "custom-vision-model"
	}
	defer func() {
		globals.VisionModelResolver = originalResolver
	}()

	image := "data:image/png;base64," + strings.Repeat("A", 20000)
	history := []globals.Message{
		{
			Role: globals.User,
			Content: "```file\n[[plot.png]]\n" +
				image +
				"\n```\n\n怎么分析这张图？",
		},
	}
	rawTokens := NumTokensFromMessages(history, globals.GPT3Turbo, false)
	buffer := NewBuffer("custom-vision-model", history, usageTestCharge{})

	if buffer.CountInputToken() >= rawTokens {
		t.Fatalf("expected configured vision buffer to strip base64 before text token count, got %d >= %d", buffer.CountInputToken(), rawTokens)
	}
	if buffer.CountInputToken() > 200 {
		t.Fatalf("expected only surrounding text to be counted, got %d tokens", buffer.CountInputToken())
	}
}
