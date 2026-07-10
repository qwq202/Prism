package utils

import (
	"chat/globals"
	"strings"
	"testing"
)

func TestBufferStreamingWritePreservesContent(t *testing.T) {
	buffer := &Buffer{}
	buffer.Write("hello")
	buffer.Write(" ")
	buffer.Write("world")

	if got := buffer.Read(); got != "hello world" {
		t.Fatalf("unexpected streamed content %q", got)
	}
	if got := string(buffer.ReadBytes()); got != "hello world" {
		t.Fatalf("unexpected streamed bytes %q", got)
	}
}

func TestBufferJSONRoundTripPreservesBuilderContent(t *testing.T) {
	buffer := &Buffer{}
	buffer.Write("cached")
	buffer.Write(" response")

	raw := Marshal(buffer)
	if raw == "" || !strings.Contains(raw, `"data":"cached response"`) {
		t.Fatalf("expected serialized streaming content, got %q", raw)
	}

	restored, err := UnmarshalString[Buffer](raw)
	if err != nil {
		t.Fatalf("unmarshal buffer: %v", err)
	}
	restored.Write(" continued")
	if got := restored.Read(); got != "cached response continued" {
		t.Fatalf("unexpected restored content %q", got)
	}
}

func TestBufferRecordOutputPrefersOfficialUsageWithoutRetokenizingResponse(t *testing.T) {
	buffer := NewBuffer(globals.GPT3Turbo, nil, usageTestCharge{})
	buffer.Write(strings.Repeat("long streamed response ", 1000))
	buffer.SetUsage(&globals.TokenUsage{
		PromptTokens:     10,
		CompletionTokens: 7,
		TotalTokens:      17,
	})

	if got := buffer.CountRecordOutputToken(); got != 7 {
		t.Fatalf("expected official completion usage 7, got %d", got)
	}
}
