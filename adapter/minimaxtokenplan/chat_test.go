package minimaxtokenplan

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"testing"
)

const minimaxInlineBase64Png = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGP4z8DwHwAFAAH/iZk9HQAAAABJRU5ErkJggg=="

func TestAnthropicCompatibleUsageCountsPromptCacheTokens(t *testing.T) {
	usage := (&AnthropicUsage{
		InputTokens:              50,
		OutputTokens:             7,
		CacheCreationInputTokens: 100,
		CacheReadInputTokens:     200,
	}).TokenUsage()

	if usage.PromptTokens != 350 ||
		usage.CompletionTokens != 7 ||
		usage.TotalTokens != 357 ||
		usage.PromptCacheWriteTokens != 100 ||
		usage.PromptCacheMissTokens != 0 ||
		usage.PromptCacheHitTokens != 200 {
		t.Fatalf("unexpected anthropic-compatible cache usage mapping: %#v", usage)
	}
}

func TestGetChatBodyPreservesAnthropicCompatiblePromptCacheControls(t *testing.T) {
	instance := NewChatInstance("https://api.minimaxi.com/anthropic", "test")
	topLevelCacheControl := map[string]interface{}{"type": "ephemeral"}
	systemCacheControl := map[string]interface{}{"type": "ephemeral", "ttl": "1h"}
	messageCacheControl := map[string]interface{}{"type": "ephemeral", "ttl": "5m"}

	body := instance.GetChatBody(&adaptercommon.ChatProps{
		Model:        "MiniMax-M2.1",
		CacheControl: topLevelCacheControl,
		Message: []globals.Message{
			{Role: globals.System, Content: "Long-lived instructions", CacheControl: systemCacheControl},
			{Role: globals.User, Content: "Reusable context", CacheControl: messageCacheControl},
		},
	}, false)

	if body.CacheCtrl["type"] != "ephemeral" {
		t.Fatalf("expected top-level cache_control to be preserved, got %#v", body.CacheCtrl)
	}

	systemBlocks, ok := body.System.([]ContentBlock)
	if !ok || len(systemBlocks) != 1 {
		t.Fatalf("expected cache-controlled system content blocks, got %#v", body.System)
	}
	if systemBlocks[0].CacheCtrl["ttl"] != "1h" {
		t.Fatalf("expected system cache_control to be preserved, got %#v", systemBlocks[0].CacheCtrl)
	}

	if len(body.Messages) != 1 || len(body.Messages[0].Content) != 1 {
		t.Fatalf("expected one user text block, got %#v", body.Messages)
	}
	if body.Messages[0].Content[0].CacheCtrl["ttl"] != "5m" {
		t.Fatalf("expected message block cache_control to be preserved, got %#v", body.Messages[0].Content[0].CacheCtrl)
	}
}

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
	if toolStop.ToolCall != nil {
		t.Fatalf("expected no duplicate tool snapshot on stop, got %#v", toolStop.ToolCall)
	}
}

func TestProcessLineToolUseDoesNotDuplicateArgumentsInBuffer(t *testing.T) {
	instance := NewChatInstance("https://api.minimaxi.com/anthropic", "test")
	instance.resetStreamState()
	buffer := &utils.Buffer{}

	lines := []string{
		`event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"lookup_weather"}}`,
		`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"city\":\"Sha"}}`,
		`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"nghai\"}"}}`,
		`event: content_block_stop
data: {"type":"content_block_stop","index":0}`,
	}

	for _, line := range lines {
		chunk, err := instance.ProcessLine(line)
		if err != nil {
			t.Fatalf("unexpected process error: %v", err)
		}
		buffer.WriteChunk(chunk)
	}

	if buffer.ToolCalls == nil || len(*buffer.ToolCalls) != 1 {
		t.Fatalf("expected one accumulated tool call, got %#v", buffer.ToolCalls)
	}
	got := (*buffer.ToolCalls)[0].Function.Arguments
	if got != `{"city":"Shanghai"}` {
		t.Fatalf("expected non-duplicated tool arguments, got %q", got)
	}
}

func TestParseEventJoinsMultilineData(t *testing.T) {
	eventType, payload := parseEvent("event: message\ndata: {\"a\":\ndata: 1}")

	if eventType != "message" {
		t.Fatalf("expected message event, got %q", eventType)
	}
	if payload != "{\"a\":\n1}" {
		t.Fatalf("expected joined payload, got %q", payload)
	}
}

func TestGetTextBlocksUsesInlineBase64ImageCapability(t *testing.T) {
	instance := NewChatInstance("https://api.minimaxi.com/anthropic", "test")
	url := "data:image/png;base64," + minimaxInlineBase64Png

	props := &adaptercommon.ChatProps{
		Model: globals.Claude3,
		Message: []globals.Message{
			{
				Role:    globals.User,
				Content: "look this " + url,
			},
		},
	}

	messages := instance.GetMessages(props)
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	blocks := messages[0].Content
	if len(blocks) != 2 {
		t.Fatalf("expected text + image blocks, got %#v", blocks)
	}

	if blocks[0].Type != "text" || blocks[0].Text == nil || *blocks[0].Text != "look this" {
		t.Fatalf("unexpected text block: %#v", blocks[0])
	}

	if blocks[1].Type != "image" || blocks[1].Source == nil {
		t.Fatalf("expected image block, got %#v", blocks[1])
	}
	if blocks[1].Source.Type != "base64" {
		t.Fatalf("expected image source type to remain base64, got %q", blocks[1].Source.Type)
	}

	mediaType, ok := blocks[1].Source.MediaType.(string)
	if !ok || mediaType != "image/png" {
		t.Fatalf("expected inline-base64 image media type image/png, got %#v", blocks[1].Source.MediaType)
	}

	data, ok := blocks[1].Source.Data.(string)
	if !ok || data != minimaxInlineBase64Png {
		t.Fatalf("expected inline-base64 image data %q, got %#v", minimaxInlineBase64Png, data)
	}
}
