package claude

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"testing"
)

func TestGetMessagesReplaysThinkingToolsAndToolResults(t *testing.T) {
	instance := NewChatInstance("https://api.anthropic.com", "test")
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
		Model: "claude-sonnet-4-20250514",
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
		t.Fatalf("expected 2 anthropic messages, got %d", len(messages))
	}

	assistant := messages[0]
	if assistant.Role != globals.Assistant {
		t.Fatalf("expected assistant role, got %q", assistant.Role)
	}
	if len(assistant.Content) != 3 {
		t.Fatalf("expected thinking + text + tool_use blocks, got %#v", assistant.Content)
	}
	if assistant.Content[0].Type != "thinking" || assistant.Content[0].Thinking == nil || *assistant.Content[0].Thinking != "plan" {
		t.Fatalf("expected thinking block replay, got %#v", assistant.Content[0])
	}
	if assistant.Content[0].Signature == nil || *assistant.Content[0].Signature != "sig-1" {
		t.Fatalf("expected thinking signature replay, got %#v", assistant.Content[0].Signature)
	}
	if assistant.Content[1].Type != "text" || assistant.Content[1].Text == nil || *assistant.Content[1].Text != "Done" {
		t.Fatalf("expected visible text block, got %#v", assistant.Content[1])
	}
	if assistant.Content[2].Type != "tool_use" || assistant.Content[2].ID == nil || *assistant.Content[2].ID != "toolu_1" {
		t.Fatalf("expected tool_use block, got %#v", assistant.Content[2])
	}

	toolResult := messages[1]
	if toolResult.Role != globals.User {
		t.Fatalf("expected tool result to be replayed as user role, got %q", toolResult.Role)
	}
	if len(toolResult.Content) != 1 || toolResult.Content[0].Type != "tool_result" {
		t.Fatalf("expected tool_result block, got %#v", toolResult.Content)
	}
	if toolResult.Content[0].ToolUseID == nil || *toolResult.Content[0].ToolUseID != "toolu_1" {
		t.Fatalf("expected tool result id to be preserved, got %#v", toolResult.Content[0].ToolUseID)
	}
}

func TestProcessLineParsesThinkingAndToolUseSSE(t *testing.T) {
	instance := NewChatInstance("https://api.anthropic.com", "test")
	instance.resetStreamState()

	thinkingStart, err := instance.ProcessLine(`event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking"}}`)
	if err != nil {
		t.Fatalf("unexpected thinking start error: %v", err)
	}
	if thinkingStart.Content != "" {
		t.Fatalf("expected no visible content on thinking start, got %q", thinkingStart.Content)
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
	if thinkingStop.Content != "\n</think>\n\n" {
		t.Fatalf("expected think closing wrapper, got %q", thinkingStop.Content)
	}
	if thinkingStop.ReasoningContent == nil || *thinkingStop.ReasoningContent != "plan" {
		t.Fatalf("expected reasoning content, got %#v", thinkingStop.ReasoningContent)
	}
	if thinkingStop.ClaudeHiddenMetadata == nil || len(thinkingStop.ClaudeHiddenMetadata.ThinkingBlocks) != 1 {
		t.Fatalf("expected claude metadata, got %#v", thinkingStop.ClaudeHiddenMetadata)
	}
	if thinkingStop.ClaudeHiddenMetadata.ThinkingBlocks[0].Signature != "sig-1" {
		t.Fatalf("expected signature replay metadata, got %#v", thinkingStop.ClaudeHiddenMetadata.ThinkingBlocks[0])
	}

	toolStart, err := instance.ProcessLine(`event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"lookup_weather"}}`)
	if err != nil {
		t.Fatalf("unexpected tool start error: %v", err)
	}
	if toolStart.ToolCall == nil || len(*toolStart.ToolCall) != 1 {
		t.Fatalf("expected tool metadata on start, got %#v", toolStart.ToolCall)
	}

	toolDeltaA, err := instance.ProcessLine(`event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"city\":\"Sha"}}`)
	if err != nil {
		t.Fatalf("unexpected tool delta error: %v", err)
	}
	if toolDeltaA.ToolCall == nil || len(*toolDeltaA.ToolCall) != 1 || (*toolDeltaA.ToolCall)[0].Function.Arguments != `{"city":"Sha` {
		t.Fatalf("expected partial tool input, got %#v", toolDeltaA.ToolCall)
	}

	toolDeltaB, err := instance.ProcessLine(`event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"nghai\"}"}}`)
	if err != nil {
		t.Fatalf("unexpected tool delta error: %v", err)
	}
	if toolDeltaB.ToolCall == nil || len(*toolDeltaB.ToolCall) != 1 || (*toolDeltaB.ToolCall)[0].Function.Arguments != `nghai"}` {
		t.Fatalf("expected second partial tool input, got %#v", toolDeltaB.ToolCall)
	}

	toolStop, err := instance.ProcessLine(`event: content_block_stop
data: {"type":"content_block_stop","index":1}`)
	if err != nil {
		t.Fatalf("unexpected tool stop error: %v", err)
	}
	if toolStop.ToolCall == nil || len(*toolStop.ToolCall) != 1 {
		t.Fatalf("expected final tool call on stop, got %#v", toolStop.ToolCall)
	}
	if (*toolStop.ToolCall)[0].Function.Arguments != `{"city":"Shanghai"}` {
		t.Fatalf("expected accumulated tool arguments, got %q", (*toolStop.ToolCall)[0].Function.Arguments)
	}
}
