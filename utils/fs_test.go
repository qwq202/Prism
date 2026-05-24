package utils

import (
	"strings"
	"testing"
)

func TestSafeJoinRejectsEscapingSegments(t *testing.T) {
	if _, err := SafeJoin("storage/generation/data/hash", "../config.yaml"); err == nil {
		t.Fatalf("expected parent-directory segment to be rejected")
	}
	if _, err := SafeJoin("storage/generation/data/hash", "/tmp/evil.txt"); err == nil {
		t.Fatalf("expected absolute segment to be rejected")
	}
	if _, err := SafeJoin("storage/generation/data/hash", `..\evil.txt`); err == nil {
		t.Fatalf("expected windows parent-directory segment to be rejected")
	}
}

func TestSafeJoinAllowsNestedRelativePaths(t *testing.T) {
	got, err := SafeJoin("storage/generation/data/hash", "src/main.go")
	if err != nil {
		t.Fatalf("expected nested path to be accepted: %v", err)
	}
	if got != "storage/generation/data/hash/src/main.go" {
		t.Fatalf("unexpected joined path %q", got)
	}
}

func TestSafeFileNameRemovesPathSeparators(t *testing.T) {
	got := SafeFileName("../Draft/One:Final", "article")
	if strings.Contains(got, "/") || strings.Contains(got, "\\") || strings.Contains(got, "..") {
		t.Fatalf("expected filename to be sanitized, got %q", got)
	}
	if got == "" {
		t.Fatalf("expected filename fallback")
	}
}
