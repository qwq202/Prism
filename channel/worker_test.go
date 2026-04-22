package channel

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"testing"
)

func TestBuildCacheChunkDoesNotPrewriteLiveBufferPayload(t *testing.T) {
	cacheBuffer := &utils.Buffer{}
	cacheBuffer.Write("cached result")
	cacheBuffer.SetInputTokens(123)

	cacheTools := globals.ToolCalls{
		{
			Id:   "call_1",
			Type: "function",
			Function: globals.ToolCallFunction{
				Name:      "lookup_weather",
				Arguments: `{"city":"Shanghai"}`,
			},
		},
	}
	cacheBuffer.SetToolCalls(&cacheTools)
	cacheBuffer.SetFunctionCall(&globals.FunctionCall{
		Name:      "lookup_weather",
		Arguments: `{"city":"Shanghai"}`,
	})
	cacheBuffer.AddReasoningContent(utils.ToPtr("Need cached reasoning."))
	cacheBuffer.SetGeminiHiddenMetadata(&globals.GeminiHiddenMetadata{
		ThoughtSignatures: []string{"sig-a"},
	})

	liveBuffer := &utils.Buffer{}
	chunk := buildCacheChunk(cacheBuffer, liveBuffer)

	if liveBuffer.CountInputToken() != 123 {
		t.Fatalf("expected cached input token to be copied, got %d", liveBuffer.CountInputToken())
	}
	if liveBuffer.Read() != "" {
		t.Fatalf("expected live buffer content to stay empty before hook write, got %q", liveBuffer.Read())
	}
	if liveBuffer.IsFunctionCalling() {
		t.Fatalf("expected no function/tool payload in live buffer before hook write")
	}
	if liveBuffer.HasGeminiHiddenMetadata() {
		t.Fatalf("expected hidden metadata to be absent before hook write")
	}

	liveBuffer.WriteChunk(chunk)

	if liveBuffer.Read() != "cached result" {
		t.Fatalf("unexpected replayed content: %q", liveBuffer.Read())
	}

	tools := liveBuffer.GetToolCalls()
	if tools == nil || len(*tools) != 1 {
		t.Fatalf("expected exactly one replayed tool call, got %#v", tools)
	}
	if (*tools)[0].Function.Arguments != `{"city":"Shanghai"}` {
		t.Fatalf("expected tool arguments to be replayed once, got %q", (*tools)[0].Function.Arguments)
	}

	functionCall := liveBuffer.GetFunctionCall()
	if functionCall == nil || functionCall.Arguments != `{"city":"Shanghai"}` {
		t.Fatalf("expected function call to be replayed once, got %#v", functionCall)
	}

	reasoning := liveBuffer.GetReasoningContent()
	if reasoning == nil || *reasoning != "Need cached reasoning." {
		t.Fatalf("expected reasoning content to be replayed once, got %#v", reasoning)
	}

	metadata := liveBuffer.GetGeminiHiddenMetadata()
	if metadata == nil || len(metadata.ThoughtSignatures) != 1 || metadata.ThoughtSignatures[0] != "sig-a" {
		t.Fatalf("expected hidden metadata to be replayed once, got %#v", metadata)
	}
}

func TestCacheHashForChatPropsIgnoresHiddenMetadataOnNonGemini(t *testing.T) {
	plain := &adaptercommon.ChatProps{
		OriginalModel: "gpt-4o",
		Message: []globals.Message{
			{Role: globals.User, Content: "hello"},
			{Role: globals.Assistant, Content: "world"},
		},
	}

	withMetadata := &adaptercommon.ChatProps{
		OriginalModel: "gpt-4o",
		Message: []globals.Message{
			{Role: globals.User, Content: "hello"},
			{
				Role:    globals.Assistant,
				Content: "world",
				GeminiHiddenMetadata: &globals.GeminiHiddenMetadata{
					ThoughtSignatures: []string{"sig-a"},
				},
			},
		},
	}

	if got, want := cacheHashForChatProps(withMetadata), cacheHashForChatProps(plain); got != want {
		t.Fatalf("expected non-gemini cache hash to ignore hidden metadata, got %q want %q", got, want)
	}

	if withMetadata.Message[1].GeminiHiddenMetadata == nil {
		t.Fatalf("expected original props to remain unchanged")
	}
}

func TestCacheHashForChatPropsKeepsHiddenMetadataOnGemini(t *testing.T) {
	plain := &adaptercommon.ChatProps{
		OriginalModel: "gemini-2.5-pro",
		Message: []globals.Message{
			{Role: globals.User, Content: "hello"},
			{Role: globals.Assistant, Content: "world"},
		},
	}

	withMetadata := &adaptercommon.ChatProps{
		OriginalModel: "gemini-2.5-pro",
		Message: []globals.Message{
			{Role: globals.User, Content: "hello"},
			{
				Role:    globals.Assistant,
				Content: "world",
				GeminiHiddenMetadata: &globals.GeminiHiddenMetadata{
					ThoughtSignatures: []string{"sig-a"},
				},
			},
		},
	}

	if got, want := cacheHashForChatProps(withMetadata), cacheHashForChatProps(plain); got == want {
		t.Fatalf("expected gemini cache hash to retain hidden metadata sensitivity")
	}
}
