package utils

import "testing"

func TestGetEncodingForChatModelUsesDeepseekFallback(t *testing.T) {
	encoding, fallback, err := getEncodingForChatModel("deepseek-v4-flash")
	if err != nil {
		t.Fatalf("expected deepseek fallback encoder, got error: %v", err)
	}

	if encoding == nil {
		t.Fatalf("expected deepseek fallback encoder instance")
	}

	if fallback != "cl100k_base" {
		t.Fatalf("expected cl100k_base fallback, got %q", fallback)
	}
}
