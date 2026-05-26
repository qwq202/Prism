package manager

import (
	"chat/globals"
	"chat/utils"
	"testing"
)

func TestExtractAssistantMessageFromBufferPlainTextResponse(t *testing.T) {
	buffer := &utils.Buffer{}
	buffer.WriteChunk(&globals.Chunk{
		Content: "hello world",
	})

	message := extractAssistantMessageFromBuffer(buffer, false, false)

	if message.Role != globals.Assistant {
		t.Fatalf("expected role %q, got %q", globals.Assistant, message.Role)
	}

	if message.Content != "hello world" {
		t.Fatalf("expected plain text content to be preserved, got %q", message.Content)
	}

	if message.ToolCalls != nil || message.FunctionCall != nil {
		t.Fatalf("expected no function-calling payloads, got tool_calls=%#v function_call=%#v", message.ToolCalls, message.FunctionCall)
	}
}

func TestExtractAssistantMessageFromBufferToolCallOnlyResponse(t *testing.T) {
	buffer := &utils.Buffer{}
	toolCalls := globals.ToolCalls{
		{
			Type: "function",
			Id:   "tool-call-1",
			Function: globals.ToolCallFunction{
				Name:      "lookup_weather",
				Arguments: "{\"city\":\"Shanghai\"}",
			},
		},
	}
	buffer.WriteChunk(&globals.Chunk{
		ToolCall: &toolCalls,
	})

	message := extractAssistantMessageFromBuffer(buffer, false, false)

	if message.Content != "" {
		t.Fatalf("expected empty text content for tool-call-only response, got %q", message.Content)
	}

	if message.ToolCalls == nil || len(*message.ToolCalls) != 1 {
		t.Fatalf("expected one tool call to be preserved, got %#v", message.ToolCalls)
	}

	if message.FunctionCall != nil {
		t.Fatalf("expected no function_call payload, got %#v", message.FunctionCall)
	}
}

func TestExtractAssistantMessageFromBufferPreservesReasoningContent(t *testing.T) {
	buffer := &utils.Buffer{}
	buffer.WriteChunk(&globals.Chunk{
		Content:          "<think>\nplanning\n</think>\n\nfinal answer",
		ReasoningContent: utils.ToPtr("planning"),
	})

	message := extractAssistantMessageFromBuffer(buffer, false, false)

	if message.Content != "<think>\nplanning\n</think>\n\nfinal answer" {
		t.Fatalf("expected visible content to be preserved, got %q", message.Content)
	}

	if message.ReasoningContent == nil || *message.ReasoningContent != "planning" {
		t.Fatalf("expected reasoning content to be preserved, got %#v", message.ReasoningContent)
	}
}

func TestSyncToolFinalMetadataMergesOfficialUsage(t *testing.T) {
	liveBuffer := &utils.Buffer{}
	roundBuffer := &utils.Buffer{}
	roundBuffer.WriteChunk(&globals.Chunk{
		Usage: &globals.TokenUsage{
			PromptTokens:          30,
			CompletionTokens:      7,
			TotalTokens:           37,
			PromptCacheHitTokens:  20,
			PromptCacheMissTokens: 10,
		},
	})

	syncToolFinalMetadata(liveBuffer, roundBuffer)

	usage := liveBuffer.GetUsage()
	if usage == nil {
		t.Fatalf("expected usage to be merged")
	}
	if usage.PromptCacheHitTokens != 20 || usage.PromptCacheMissTokens != 10 {
		t.Fatalf("unexpected merged usage: %#v", usage)
	}
}

func TestExtractAssistantMessageFromBufferPreservesVisibleTextAndToolCalls(t *testing.T) {
	buffer := &utils.Buffer{}
	toolCalls := globals.ToolCalls{
		{
			Type: "function",
			Id:   "memory-tool-1",
			Function: globals.ToolCallFunction{
				Name:      "memory_tool",
				Arguments: "{\"action\":\"create\"}",
			},
		},
	}

	buffer.AddToolCalls(&toolCalls)
	buffer.WriteChunk(&globals.Chunk{
		Content: "已经帮你记住了。",
	})

	message := extractAssistantMessageFromBuffer(buffer, false, false)

	if message.Content != "已经帮你记住了。" {
		t.Fatalf("expected visible assistant content to be preserved, got %q", message.Content)
	}

	if message.ToolCalls == nil || len(*message.ToolCalls) != 1 {
		t.Fatalf("expected tool calls to be preserved alongside visible content, got %#v", message.ToolCalls)
	}

	if (*message.ToolCalls)[0].Function.Name != "memory_tool" {
		t.Fatalf("unexpected tool call payload: %#v", (*message.ToolCalls)[0])
	}
}

func TestExtractAssistantMessageFromBufferMetadataOnlyResponse(t *testing.T) {
	buffer := &utils.Buffer{}
	buffer.WriteChunk(&globals.Chunk{
		GeminiHiddenMetadata: &globals.GeminiHiddenMetadata{
			ThoughtSignatures: []string{"sig-1"},
		},
	})

	message := extractAssistantMessageFromBuffer(buffer, false, false)

	if message.Content != "" {
		t.Fatalf("expected metadata-only response to keep empty content, got %q", message.Content)
	}

	if message.GeminiHiddenMetadata == nil || len(message.GeminiHiddenMetadata.ThoughtSignatures) != 1 {
		t.Fatalf("expected metadata signatures to be preserved, got %#v", message.GeminiHiddenMetadata)
	}

	if message.ToolCalls != nil || message.FunctionCall != nil {
		t.Fatalf("expected metadata-only response without function-calling payloads, got tool_calls=%#v function_call=%#v", message.ToolCalls, message.FunctionCall)
	}
}

func TestExtractAssistantMessageFromBufferPreservesClaudeMetadata(t *testing.T) {
	buffer := &utils.Buffer{}
	buffer.WriteChunk(&globals.Chunk{
		ClaudeHiddenMetadata: &globals.ClaudeHiddenMetadata{
			ThinkingBlocks: []globals.ClaudeThinkingBlock{
				{Thinking: "plan", Signature: "sig-claude"},
			},
		},
	})

	message := extractAssistantMessageFromBuffer(buffer, false, false)

	if message.Content != "" {
		t.Fatalf("expected claude metadata-only response to keep empty content, got %q", message.Content)
	}

	if message.ClaudeHiddenMetadata == nil || len(message.ClaudeHiddenMetadata.ThinkingBlocks) != 1 {
		t.Fatalf("expected claude hidden metadata to be preserved, got %#v", message.ClaudeHiddenMetadata)
	}
}

func TestExtractAssistantMessageFromBufferInterruptedDropsFunctionPayloads(t *testing.T) {
	buffer := &utils.Buffer{}
	toolCalls := globals.ToolCalls{
		{
			Type: "function",
			Id:   "partial-tool-call",
			Function: globals.ToolCallFunction{
				Name:      "lookup_weather",
				Arguments: "{\"city\":\"Sha",
			},
		},
	}

	buffer.WriteChunk(&globals.Chunk{Content: "partial visible text"})
	buffer.WriteChunk(&globals.Chunk{ToolCall: &toolCalls})
	buffer.WriteChunk(&globals.Chunk{
		FunctionCall: &globals.FunctionCall{
			Name:      "lookup_air_quality",
			Arguments: "{\"city\":\"Sha",
		},
	})
	buffer.WriteChunk(&globals.Chunk{
		GeminiHiddenMetadata: &globals.GeminiHiddenMetadata{
			ThoughtSignatures: []string{"sig-2"},
		},
	})

	message := extractAssistantMessageFromBuffer(buffer, true, false)

	if message.Content != "partial visible text" {
		t.Fatalf("expected visible text to remain on interrupted response, got %q", message.Content)
	}

	if message.ToolCalls != nil || message.FunctionCall != nil {
		t.Fatalf("expected interrupted response to drop partial function-calling payloads, got tool_calls=%#v function_call=%#v", message.ToolCalls, message.FunctionCall)
	}

	if message.GeminiHiddenMetadata == nil || len(message.GeminiHiddenMetadata.ThoughtSignatures) != 1 {
		t.Fatalf("expected metadata to remain available, got %#v", message.GeminiHiddenMetadata)
	}
}
