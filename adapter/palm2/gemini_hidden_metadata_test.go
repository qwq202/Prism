package palm2

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"testing"
)

func ptrString(value string) *string {
	return &value
}

func TestGetGeminiContentsReplaySignaturesOnFunctionCalls(t *testing.T) {
	instance := &ChatInstance{}
	toolCalls := globals.ToolCalls{
		{
			Type: "function",
			Id:   "call-1",
			Function: globals.ToolCallFunction{
				Name:      "lookup_weather",
				Arguments: `{"city":"shanghai"}`,
			},
		},
		{
			Type: "function",
			Id:   "call-2",
			Function: globals.ToolCallFunction{
				Name:      "lookup_air_quality",
				Arguments: `{"city":"shanghai"}`,
			},
		},
	}

	messages := []globals.Message{
		{
			Role:    globals.User,
			Content: "Need weather and AQI",
		},
		{
			Role:      globals.Assistant,
			ToolCalls: &toolCalls,
			GeminiHiddenMetadata: &globals.GeminiHiddenMetadata{
				ThoughtSignatures: []string{" sig-first ", "sig-second", "sig-overflow"},
			},
		},
	}

	contents := instance.GetGeminiContents("gemini-3.0-flash", messages)
	if len(contents) != 2 {
		t.Fatalf("expected 2 gemini contents, got %d", len(contents))
	}

	parts := contents[1].Parts
	if len(parts) != 2 {
		t.Fatalf("expected 2 function call parts, got %d", len(parts))
	}

	if parts[0].FunctionCall == nil || parts[0].FunctionCall.Name != "lookup_weather" {
		t.Fatalf("expected first part to keep first function call, got %#v", parts[0].FunctionCall)
	}
	if parts[0].ThoughtSignature == nil || *parts[0].ThoughtSignature != "sig-first" {
		t.Fatalf("expected first function call to carry first signature, got %#v", parts[0].ThoughtSignature)
	}

	if parts[1].FunctionCall == nil || parts[1].FunctionCall.Name != "lookup_air_quality" {
		t.Fatalf("expected second part to keep second function call, got %#v", parts[1].FunctionCall)
	}
	if parts[1].ThoughtSignature == nil || *parts[1].ThoughtSignature != "sig-second" {
		t.Fatalf("expected second function call to carry second signature, got %#v", parts[1].ThoughtSignature)
	}

	if len(contents[1].Parts) != 2 {
		t.Fatalf("expected overflow signature to be dropped instead of emitted as metadata-only request part, got %#v", contents[1].Parts)
	}
}

func TestGetGeminiContentsWithoutMetadataKeepsLegacySameRoleMerge(t *testing.T) {
	instance := &ChatInstance{}
	toolCalls1 := globals.ToolCalls{
		{
			Type: "function",
			Id:   "call-1",
			Function: globals.ToolCallFunction{
				Name:      "lookup_weather",
				Arguments: `{"city":"beijing"}`,
			},
		},
	}
	toolCalls2 := globals.ToolCalls{
		{
			Type: "function",
			Id:   "call-2",
			Function: globals.ToolCallFunction{
				Name:      "lookup_air_quality",
				Arguments: `{"city":"beijing"}`,
			},
		},
	}

	messages := []globals.Message{
		{
			Role:    globals.User,
			Content: "start tool chain",
		},
		{
			Role:      globals.Assistant,
			ToolCalls: &toolCalls1,
		},
		{
			Role:      globals.Assistant,
			ToolCalls: &toolCalls2,
		},
	}

	contents := instance.GetGeminiContents("gemini-3.0-flash", messages)
	if len(contents) != 2 {
		t.Fatalf("expected legacy same-role merge when no metadata exists, got %d contents", len(contents))
	}

	parts := contents[1].Parts
	if len(parts) != 2 {
		t.Fatalf("expected merged assistant function-call parts, got %d parts", len(parts))
	}

	if parts[0].FunctionCall == nil || parts[1].FunctionCall == nil {
		t.Fatalf("expected both merged parts to remain function-call parts, got %#v", parts)
	}
	if parts[0].ThoughtSignature != nil || parts[1].ThoughtSignature != nil {
		t.Fatalf("expected no signature injection without metadata, got %#v", parts)
	}
}

func TestGetGeminiContentsDoesNotForceBoundaryForDroppedTextSignatures(t *testing.T) {
	instance := &ChatInstance{}
	withSignature := []globals.Message{
		{
			Role:    globals.User,
			Content: "start",
		},
		{
			Role:    globals.Assistant,
			Content: "first",
			GeminiHiddenMetadata: &globals.GeminiHiddenMetadata{
				ThoughtSignatures: []string{"sig-first"},
			},
		},
		{
			Role:    globals.Assistant,
			Content: "second",
		},
	}

	withoutSignature := []globals.Message{
		{
			Role:    globals.User,
			Content: "start",
		},
		{
			Role:    globals.Assistant,
			Content: "first",
		},
		{
			Role:    globals.Assistant,
			Content: "second",
		},
	}

	withSignatureContents := instance.GetGeminiContents("gemini-3.0-flash", withSignature)
	withoutSignatureContents := instance.GetGeminiContents("gemini-3.0-flash", withoutSignature)

	if len(withSignatureContents) != 2 {
		t.Fatalf("expected dropped text-only signatures to keep legacy same-role merge, got %d contents", len(withSignatureContents))
	}
	if len(withoutSignatureContents) != 2 {
		t.Fatalf("expected legacy merge without signature-bearing parts, got %d contents", len(withoutSignatureContents))
	}
}

func TestGetGeminiPartsReplayNotInjectedOutsideAssistantTurns(t *testing.T) {
	userMessage := globals.Message{
		Role:    globals.User,
		Content: "hello",
		GeminiHiddenMetadata: &globals.GeminiHiddenMetadata{
			ThoughtSignatures: []string{"sig-user"},
		},
	}
	userParts := getGeminiParts("gemini-3.0-flash", nil, userMessage)
	for _, part := range userParts {
		if part.ThoughtSignature != nil {
			t.Fatalf("expected no signature replay on user turn, got part %#v", part)
		}
	}

	toolName := "lookup_weather"
	toolMessage := globals.Message{
		Role:    globals.Tool,
		Content: `{"temp":"27"}`,
		Name:    &toolName,
		GeminiHiddenMetadata: &globals.GeminiHiddenMetadata{
			ThoughtSignatures: []string{"sig-tool"},
		},
	}
	toolParts := getGeminiParts("gemini-3.0-flash", nil, toolMessage)
	for _, part := range toolParts {
		if part.ThoughtSignature != nil {
			t.Fatalf("expected no signature replay on tool turn, got part %#v", part)
		}
	}
}

func TestGetGeminiContentsSkipsLeadingModelInsteadOfEmptyUser(t *testing.T) {
	instance := &ChatInstance{}
	messages := []globals.Message{
		{
			Role:    globals.Assistant,
			Content: "orphaned answer from a clipped context window",
			GeminiHiddenMetadata: &globals.GeminiHiddenMetadata{
				ThoughtSignatures: []string{"sig-orphan"},
			},
		},
		{
			Role:    globals.User,
			Content: "fresh question",
		},
	}

	contents := instance.GetGeminiContents("gemini-3.1-flash-lite-preview", messages)
	if len(contents) != 1 {
		t.Fatalf("expected leading model turn to be skipped, got %#v", contents)
	}
	if contents[0].Role != GeminiUserType {
		t.Fatalf("expected first content to be user, got %q", contents[0].Role)
	}
	if len(contents[0].Parts) != 1 || contents[0].Parts[0].Text == nil || *contents[0].Parts[0].Text != "fresh question" {
		t.Fatalf("expected fresh user prompt without an empty filler part, got %#v", contents[0].Parts)
	}
}

func TestGetGeminiContentsDropsMetadataOnlyAssistantParts(t *testing.T) {
	instance := &ChatInstance{}
	messages := []globals.Message{
		{
			Role:    globals.User,
			Content: "first question",
		},
		{
			Role: globals.Assistant,
			GeminiHiddenMetadata: &globals.GeminiHiddenMetadata{
				ThoughtSignatures: []string{"sig-only"},
			},
		},
		{
			Role:    globals.User,
			Content: "next question",
		},
	}

	contents := instance.GetGeminiContents("gemini-3.1-flash-lite-preview", messages)
	if len(contents) != 1 {
		t.Fatalf("expected metadata-only assistant turn to be dropped and user turns to merge, got %#v", contents)
	}
	for _, content := range contents {
		for _, part := range content.Parts {
			if !hasGeminiRequestPartData(part) {
				t.Fatalf("expected every request part to carry data, got %#v", part)
			}
			if part.ThoughtSignature != nil && part.FunctionCall == nil {
				t.Fatalf("expected signatures only on function call request parts, got %#v", part)
			}
		}
	}
}

func TestGetGeminiChunkCapturesThoughtSignatures(t *testing.T) {
	instance := &ChatInstance{}
	response := GeminiChatResponse{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Parts: []GeminiChatPart{
						{Text: ptrString("hello ")},
						{Text: ptrString("world")},
						{ThoughtSignature: ptrString(" sig-a ")},
						{ThoughtSignature: ptrString("sig-a")},
						{ThoughtSignature: ptrString("sig-b")},
					},
				},
			},
		},
	}

	chunk, err := instance.GetGeminiChunk("", response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chunk.Content != "hello world" {
		t.Fatalf("expected visible content to stay unchanged, got %q", chunk.Content)
	}

	if chunk.GeminiHiddenMetadata == nil {
		t.Fatalf("expected hidden metadata to be captured")
	}
	if len(chunk.GeminiHiddenMetadata.ThoughtSignatures) != 2 {
		t.Fatalf("expected deduped signatures, got %#v", chunk.GeminiHiddenMetadata.ThoughtSignatures)
	}
	if chunk.GeminiHiddenMetadata.ThoughtSignatures[0] != "sig-a" || chunk.GeminiHiddenMetadata.ThoughtSignatures[1] != "sig-b" {
		t.Fatalf("unexpected signature order/content: %#v", chunk.GeminiHiddenMetadata.ThoughtSignatures)
	}
}

func TestBuildGeminiChunkStreamMetadataOnly(t *testing.T) {
	instance := &ChatInstance{
		isFirstReasoning: true,
	}

	chunk := instance.buildGeminiChunk("", []GeminiChatPart{
		{
			Thought:          true,
			ThoughtSignature: ptrString("sig-final"),
		},
	}, true)

	if chunk.Content != "" {
		t.Fatalf("expected metadata-only stream part to keep empty visible content, got %q", chunk.Content)
	}

	if chunk.GeminiHiddenMetadata == nil || len(chunk.GeminiHiddenMetadata.ThoughtSignatures) != 1 {
		t.Fatalf("expected metadata-only stream chunk to keep signature, got %#v", chunk.GeminiHiddenMetadata)
	}
	if chunk.GeminiHiddenMetadata.ThoughtSignatures[0] != "sig-final" {
		t.Fatalf("unexpected stream signature value: %#v", chunk.GeminiHiddenMetadata.ThoughtSignatures)
	}

	if chunk.IsEmpty() {
		t.Fatalf("metadata-only stream chunk must be non-empty for forwarding")
	}
}

func TestGetGeminiThinkingConfigSkipsNoThinkingModels(t *testing.T) {
	config := getGeminiThinkingConfig(&adaptercommon.ChatProps{
		Model:                "gemini-3-flash-preview-nothinking",
		GeminiThinkingBudget: utilsIntPtr(4096),
	})

	if config != nil {
		t.Fatalf("expected no thinking config for nothinking model, got %#v", config)
	}
}

func TestBuildGeminiChunkSuppressesExplicitThoughtsForNoThinkingModels(t *testing.T) {
	instance := &ChatInstance{
		isFirstReasoning: true,
	}

	parts := []GeminiChatPart{
		{Text: ptrString("internal "), Thought: true},
		{Text: ptrString("answer")},
	}

	nonStream := instance.buildGeminiChunk("gemini-3-flash-preview-nothinking", parts, false)
	if nonStream.Content != "answer" {
		t.Fatalf("expected non-stream nothinking content to exclude reasoning, got %q", nonStream.Content)
	}

	instance.isFirstReasoning = true
	instance.isReasonOver = false

	stream := instance.buildGeminiChunk("gemini-3-flash-preview-nothinking", parts, true)
	if stream.Content != "answer" {
		t.Fatalf("expected stream nothinking content to exclude reasoning, got %q", stream.Content)
	}
}

func TestGetVertexAIExpressChatEndpoint(t *testing.T) {
	instance := NewVertexAIExpressChatInstance("https://aiplatform.googleapis.com/", "api key/with+chars")

	endpoint := instance.GetChatEndpoint("gemini-2.5-flash", false)
	expected := "https://aiplatform.googleapis.com/v1/publishers/google/models/gemini-2.5-flash:generateContent?key=api+key%2Fwith%2Bchars"
	if endpoint != expected {
		t.Fatalf("unexpected vertex express endpoint:\nwant %s\n got %s", expected, endpoint)
	}
}

func TestGetVertexAIExpressStreamPreviewEndpointUsesV1Beta1(t *testing.T) {
	instance := NewVertexAIExpressChatInstance("", "test-key")

	endpoint := instance.GetChatEndpoint("gemini-3-pro-preview", true)
	expected := "https://aiplatform.googleapis.com/v1beta1/publishers/google/models/gemini-3-pro-preview:streamGenerateContent?alt=sse&key=test-key"
	if endpoint != expected {
		t.Fatalf("unexpected vertex express stream endpoint:\nwant %s\n got %s", expected, endpoint)
	}
}

func TestGetVertexAIExpressChatEndpointKeepsQualifiedModelPath(t *testing.T) {
	instance := NewVertexAIExpressChatInstance("https://aiplatform.googleapis.com", "test-key")

	endpoint := instance.GetChatEndpoint("publishers/google/models/gemini-2.5-flash", false)
	expected := "https://aiplatform.googleapis.com/v1/publishers/google/models/gemini-2.5-flash:generateContent?key=test-key"
	if endpoint != expected {
		t.Fatalf("unexpected qualified model path endpoint:\nwant %s\n got %s", expected, endpoint)
	}
}

func utilsIntPtr(value int) *int {
	return &value
}
