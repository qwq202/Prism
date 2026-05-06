package manager

import (
	"chat/globals"
	"testing"
)

func TestLatestMessageContentHandlesEmptySegment(t *testing.T) {
	if content, ok := latestMessageContent(nil); ok || content != "" {
		t.Fatalf("expected empty segment to be rejected, got content=%q ok=%v", content, ok)
	}

	content, ok := latestMessageContent([]globals.Message{
		{Role: globals.User, Content: "first"},
		{Role: globals.User, Content: "latest"},
	})
	if !ok || content != "latest" {
		t.Fatalf("expected latest message content, got content=%q ok=%v", content, ok)
	}
}
