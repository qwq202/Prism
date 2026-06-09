package manager

import (
	"chat/globals"
	"testing"
)

func TestFormatBuiltinToolNames(t *testing.T) {
	if got := formatBuiltinToolNames(nil); got != "[]" {
		t.Fatalf("expected empty builtin tool list, got %q", got)
	}

	got := formatBuiltinToolNames([]string{"google_search", "url_context"})
	if got != "[google_search,url_context]" {
		t.Fatalf("unexpected builtin tool list: %q", got)
	}
}

func TestBuildToolCallEvent(t *testing.T) {
	call := globals.ToolCall{
		Id:   "call_1",
		Type: "function",
		Function: globals.ToolCallFunction{
			Name:      "memory_tool",
			Arguments: `{"action":"create"}`,
		},
	}

	event := buildToolCallEvent(call, "start")
	if event == nil {
		t.Fatalf("expected tool call event to be created")
	}

	if event.Id != "call_1" || event.Name != "memory_tool" || event.Status != "start" {
		t.Fatalf("unexpected tool call event payload: %#v", event)
	}

	if event.Arguments != `{"action":"create"}` {
		t.Fatalf("unexpected tool call arguments: %#v", event)
	}
}

func TestBuildToolResultEventMarksErrors(t *testing.T) {
	call := globals.ToolCall{
		Id:   "call_1",
		Type: "function",
		Function: globals.ToolCallFunction{
			Name:      "memory_tool",
			Arguments: `{"action":"create"}`,
		},
	}

	toolMessage := globals.Message{
		Role:    globals.Tool,
		Content: `{"status":"error","error":"reason is required"}`,
		ToolCallId: func() *string {
			value := "call_1"
			return &value
		}(),
	}

	event := buildToolResultEvent(call, toolMessage)
	if event == nil {
		t.Fatalf("expected tool result event to be created")
	}

	if event.Status != "error" {
		t.Fatalf("expected error status, got %#v", event)
	}

	if event.Error != "reason is required" {
		t.Fatalf("expected parsed error message, got %#v", event)
	}
}
