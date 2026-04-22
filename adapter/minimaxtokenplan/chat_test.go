package minimaxtokenplan

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"testing"
)

func TestGetMessagesReplaysThinkingToolsAndToolResults(t *testing.T) {
	instance := NewChatInstance("https://api.minimaxi.com/anthropic", "test")
	toolCalls := globals.ToolCalls{
		{
			Type: "function",
			Id:   "toolu_1",
			Function: globals.ToolCallFunction{
				Name:      "lookup_weather",
				Arguments: `{"city":"Shanghai"}`,
			},
		},
	}

	props := &adaptercommon.ChatProps{
		Model: "MiniMax-M2.1",
		Message: []globals.Message{
			{
				Role:             globals.Assistant,
				Content:          "<think>\nplan\n</think>\n\nDone",
				ReasoningContent: utils.ToPtr("plan"),
				ToolCalls:        &toolCalls,
				ClaudeHiddenMetadata: &globals.ClaudeHiddenMetadata{
					ThinkingBlocks: []globals.ClaudeThinkingBlock{
						{Thinking: "plan", Signature: "sig-1"},
					},
				},
			},
			{
				Role:       globals.Tool,
				Content:    `{"status":"success"}`,
				ToolCallId: utils.ToPtr("toolu_1"),
			},
		},
	}

	messages := instance.GetMessages(props)
	if len(messages) != 2 {
		t.Fatalf("expected 2 anthropic-compatible messages, got %d", len(messages))
	}

	assistant := messages[0]
	if len(assistant.Content) != 3 {
		t.Fatalf("expected thinking + text + tool_use blocks, got %#v", assistant.Content)
	}
	if assistant.Content[0].Type != "thinking" || assistant.Content[0].Thinking == nil || *assistant.Content[0].Thinking != "plan" {
		t.Fatalf("expected thinking block replay, got %#v", assistant.Content[0])
	}
	if assistant.Content[2].Type != "tool_use" || assistant.Content[2].ID == nil || *assistant.Content[2].ID != "toolu_1" {
		t.Fatalf("expected tool_use block, got %#v", assistant.Content[2])
	}

	toolResult := messages[1]
	if toolResult.Role != globals.User || len(toolResult.Content) != 1 || toolResult.Content[0].Type != "tool_result" {
		t.Fatalf("expected tool_result replay, got %#v", toolResult)
	}
}

func TestProcessLineParsesThinkingAndToolUseSSE(t *testing.T) {
	instance := NewChatInstance("https://api.minimaxi.com/anthropic", "test")
	instance.resetStreamState()

	_, err := instance.ProcessLine(`event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking"}}`)
	if err != nil {
		t.Fatalf("unexpected thinking start error: %v", err)
	}

	thinkingDelta, err := instance.ProcessLine(`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"plan"}}`)
	if err != nil {
		t.Fatalf("unexpected thinking delta error: %v", err)
	}
	if thinkingDelta.Content != "<think>\nplan" {
		t.Fatalf("expected think opening wrapper, got %q", thinkingDelta.Content)
	}

	_, err = instance.ProcessLine(`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"sig-1"}}`)
	if err != nil {
		t.Fatalf("unexpected signature delta error: %v", err)
	}

	thinkingStop, err := instance.ProcessLine(`event: content_block_stop
data: {"type":"content_block_stop","index":0}`)
	if err != nil {
		t.Fatalf("unexpected thinking stop error: %v", err)
	}
	if thinkingStop.ClaudeHiddenMetadata == nil || len(thinkingStop.ClaudeHiddenMetadata.ThinkingBlocks) != 1 {
		t.Fatalf("expected minimax thinking metadata, got %#v", thinkingStop.ClaudeHiddenMetadata)
	}

	toolStart, err := instance.ProcessLine(`event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"lookup_weather"}}`)
	if err != nil {
		t.Fatalf("unexpected tool start error: %v", err)
	}
	if toolStart.ToolCall == nil || len(*toolStart.ToolCall) != 1 {
		t.Fatalf("expected tool metadata on start, got %#v", toolStart.ToolCall)
	}

	_, err = instance.ProcessLine(`event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"city\":\"Sha"}}`)
	if err != nil {
		t.Fatalf("unexpected tool delta error: %v", err)
	}

	_, err = instance.ProcessLine(`event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"nghai\"}"}}`)
	if err != nil {
		t.Fatalf("unexpected tool delta error: %v", err)
	}

	toolStop, err := instance.ProcessLine(`event: content_block_stop
data: {"type":"content_block_stop","index":1}`)
	if err != nil {
		t.Fatalf("unexpected tool stop error: %v", err)
	}
	if toolStop.ToolCall == nil || len(*toolStop.ToolCall) != 1 || (*toolStop.ToolCall)[0].Function.Arguments != `{"city":"Shanghai"}` {
		t.Fatalf("expected accumulated tool arguments, got %#v", toolStop.ToolCall)
	}
}
