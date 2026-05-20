package manager

import (
	"chat/globals"
	"chat/utils"
	"testing"
)

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

func TestExecuteSearchToolCallRequiresQuery(t *testing.T) {
	call := globals.ToolCall{
		Id:   "call_search",
		Type: "function",
		Function: globals.ToolCallFunction{
			Name:      "search",
			Arguments: `{"type":"web"}`,
		},
	}

	message := executeSearchToolCall(call)
	if message.Role != globals.Tool || message.ToolCallId == nil || *message.ToolCallId != "call_search" {
		t.Fatalf("unexpected search tool message metadata: %#v", message)
	}

	payload, err := utils.UnmarshalString[map[string]string](message.Content)
	if err != nil || payload == nil {
		t.Fatalf("expected JSON search tool payload, got %q: %v", message.Content, err)
	}
	if payload["status"] != "error" || payload["error"] != "query is required" {
		t.Fatalf("unexpected search tool error payload: %#v", payload)
	}
}
