package globals

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGeminiHiddenMetadataUnmarshalLegacyThoughtSignature(t *testing.T) {
	raw := `{"thought_signature":" legacy-signature "}`

	var metadata GeminiHiddenMetadata
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if len(metadata.ThoughtSignatures) != 1 {
		t.Fatalf("expected 1 signature, got %d", len(metadata.ThoughtSignatures))
	}

	if metadata.ThoughtSignatures[0] != "legacy-signature" {
		t.Fatalf("unexpected signature value: %q", metadata.ThoughtSignatures[0])
	}
}

func TestNormalizeGeminiThoughtSignaturesBoundsAndDedupe(t *testing.T) {
	overLimit := strings.Repeat("x", GeminiThoughtSignatureMaxBytes+1)
	input := []string{
		" a ",
		"a",
		"",
		"b",
		overLimit,
		" c ",
	}

	result := NormalizeGeminiThoughtSignatures(input, 2)
	if len(result) != 2 {
		t.Fatalf("expected 2 signatures after limit, got %d", len(result))
	}

	if result[0] != "a" || result[1] != "b" {
		t.Fatalf("unexpected normalized signatures: %#v", result)
	}
}

func TestGeminiNoThinkingModelDisablesThinkingControls(t *testing.T) {
	model := "gemini-3-flash-preview-nothinking"

	if !IsGeminiNoThinkingModel(model) {
		t.Fatalf("expected %q to be detected as no-thinking gemini model", model)
	}

	if SupportGeminiThinkingLevel(model) {
		t.Fatalf("expected thinking-level support to be disabled for %q", model)
	}

	if SupportGeminiThinkingBudget(model) {
		t.Fatalf("expected thinking-budget support to be disabled for %q", model)
	}
}

func TestGemini35FlashCapabilities(t *testing.T) {
	if !IsVisionModel(Gemini35Flash) {
		t.Fatalf("expected %q to support vision inputs", Gemini35Flash)
	}

	if !SupportGeminiThinkingLevel(Gemini35Flash) {
		t.Fatalf("expected %q to use gemini thinkingLevel control", Gemini35Flash)
	}

	if SupportGeminiThinkingBudget(Gemini35Flash) {
		t.Fatalf("expected %q to not use gemini thinkingBudget control", Gemini35Flash)
	}
}

func TestGeminiCodeExecutionSupport(t *testing.T) {
	supported := []string{
		Gemini20Flash,
		Gemini25Flash,
		Gemini25Pro,
		Gemini35Flash,
		Gemini3Flash,
		Gemini3ProPreview,
		"gemini-3-flash-preview",
		"gemini-3.1-pro-preview-customtools",
		"gemini-3.1-flash-lite-preview",
	}

	for _, model := range supported {
		if !SupportGeminiCodeExecution(model) {
			t.Fatalf("expected %q to support code execution", model)
		}
	}

	unsupported := []string{
		GeminiPro,
		Gemini20FlashLite,
		Gemini3ProImagePreview,
		"gemini-3.1-flash-image-preview",
		"gemini-2.5-flash-preview-tts",
		"gemini-2.5-flash-native-audio-preview-12-2025",
		"gemini-2.0-flash-preview-image-generation",
	}

	for _, model := range unsupported {
		if SupportGeminiCodeExecution(model) {
			t.Fatalf("expected %q to not support code execution", model)
		}
	}
}
