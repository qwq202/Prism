package utils

import (
	"chat/globals"
	"strings"
	"testing"
)

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

func TestBufferRecordsBuiltinToolUsage(t *testing.T) {
	buffer := &Buffer{}
	buffer.SetCodeExecutionToolUsage(true, true)
	buffer.WriteChunk(&globals.Chunk{
		BuiltinToolUsage: &globals.BuiltinToolUsage{
			CodeExecution: &globals.BuiltinToolUsageStatus{
				Used: true,
			},
		},
	})

	usage := buffer.GetBuiltinToolUsage()
	if usage == nil || usage.CodeExecution == nil {
		t.Fatalf("expected code execution usage to be recorded, got %#v", usage)
	}
	if !usage.CodeExecution.Enabled || !usage.CodeExecution.Sent || !usage.CodeExecution.Used {
		t.Fatalf("unexpected code execution usage: %#v", usage.CodeExecution)
	}

	detail := buffer.GetBillingDetail()
	for _, expected := range []string{"tools", "code_execution", "enabled", "sent", "used"} {
		if !strings.Contains(detail, expected) {
			t.Fatalf("expected billing detail to include %q, got %q", expected, detail)
		}
	}
}
