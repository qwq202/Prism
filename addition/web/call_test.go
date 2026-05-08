package web

import (
	"chat/globals"
	"strings"
	"testing"
)

func TestRecentUserSearchContextUsesLatestThreeUserMessages(t *testing.T) {
	messages := []globals.Message{
		{Role: globals.User, Content: "第一条用户消息"},
		{Role: globals.Assistant, Content: "助手回复"},
		{Role: globals.User, Content: "第二条用户消息"},
		{Role: globals.System, Content: "系统消息"},
		{Role: globals.User, Content: "第三条用户消息"},
		{Role: globals.User, Content: "第四条用户消息"},
	}

	got := recentUserSearchContext(messages, 3)
	want := strings.Join([]string{
		"Recent user messages:",
		"1. 第二条用户消息",
		"2. 第三条用户消息",
		"3. 第四条用户消息",
	}, "\n")

	if got != want {
		t.Fatalf("expected recent user context %q, got %q", want, got)
	}
}

func TestRecentUserSearchContextFallsBackToLastMessage(t *testing.T) {
	got := recentUserSearchContext([]globals.Message{
		{Role: globals.System, Content: "系统消息"},
		{Role: globals.Assistant, Content: "助手回复"},
	}, 3)

	if got != "助手回复" {
		t.Fatalf("expected fallback to last message, got %q", got)
	}
}
