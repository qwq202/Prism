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
