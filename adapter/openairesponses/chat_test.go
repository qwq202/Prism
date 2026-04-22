package openairesponses

import "testing"

func TestBuildResponseChunkExtractsFunctionCalls(t *testing.T) {
	form := &ResponseResponse{
		Output: []OutputItem{
			{
				Type:      "function_call",
				Name:      "memory_tool",
				Arguments: `{"action":"create"}`,
				CallID:    "call_1",
			},
		},
	}

	chunk := buildResponseChunk(form)
	if chunk.ToolCall == nil || len(*chunk.ToolCall) != 1 {
		t.Fatalf("expected function calls to be extracted, got %#v", chunk.ToolCall)
	}
	if (*chunk.ToolCall)[0].Function.Name != "memory_tool" {
		t.Fatalf("unexpected function call payload: %#v", (*chunk.ToolCall)[0])
	}
}
