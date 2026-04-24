package responsescompat

import (
	"chat/globals"
	"testing"
)

func TestReplayFunctionCallsAndFunctionCallOutput(t *testing.T) {
	toolCalls := globals.ToolCalls{
		{
			Type: "function",
			Id:   "call_1",
			Function: globals.ToolCallFunction{
				Name:      "memory_tool",
				Arguments: `{"action":"create"}`,
			},
		},
	}

	items := ReplayFunctionCalls(globals.Message{
		Role:      globals.Assistant,
		ToolCalls: &toolCalls,
	})
	if len(items) != 1 {
		t.Fatalf("expected one replayed function_call, got %#v", items)
	}

	functionCall, ok := items[0].(OutputItem)
	if !ok || functionCall.Type != "function_call" || functionCall.Name != "memory_tool" || functionCall.CallID != "call_1" {
		t.Fatalf("unexpected function_call replay item: %#v", items[0])
	}

	output := FunctionCallOutput(globals.Message{
		Role:       globals.Tool,
		ToolCallId: stringPtr(" call_1 "),
		Content:    `{"status":"success"}`,
	})
	if output == nil || output.Type != "function_call_output" || output.CallID != "call_1" || output.Output != `{"status":"success"}` {
		t.Fatalf("unexpected function_call_output item: %#v", output)
	}
}

func TestExtractOutputTextAndToolCalls(t *testing.T) {
	chunk := BuildResponseChunk([]OutputItem{
		{
			Type: "message",
			Role: globals.Assistant,
			Content: []OutputContent{
				{Type: "output_text", Text: "hello "},
				{Type: "output_text", Text: "world"},
			},
		},
		{
			Type:      "function_call",
			Name:      "memory_tool",
			Arguments: `{"action":"create"}`,
			CallID:    "call_1",
		},
	})

	if chunk.Content != "hello world" {
		t.Fatalf("expected concatenated output text, got %q", chunk.Content)
	}
	if chunk.ToolCall == nil || len(*chunk.ToolCall) != 1 {
		t.Fatalf("expected extracted tool call, got %#v", chunk.ToolCall)
	}
	if (*chunk.ToolCall)[0].Function.Name != "memory_tool" || (*chunk.ToolCall)[0].Id != "call_1" {
		t.Fatalf("unexpected extracted tool call: %#v", (*chunk.ToolCall)[0])
	}
}

func TestEmitFunctionCallEvent(t *testing.T) {
	chunk := EmitFunctionCallEvent(&OutputItem{
		Type:      "function_call",
		Name:      "memory_tool",
		Arguments: `{"action":"create"}`,
		CallID:    "call_1",
	})

	if chunk == nil || chunk.ToolCall == nil || len(*chunk.ToolCall) != 1 {
		t.Fatalf("expected function call chunk, got %#v", chunk)
	}
}

func stringPtr(v string) *string {
	return &v
}
