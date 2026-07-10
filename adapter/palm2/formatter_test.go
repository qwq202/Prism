package palm2

import (
	"chat/globals"
	"strings"
	"testing"
)

const palm2InlineBase64Png = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGP4z8DwHwAFAAH/iZk9HQAAAABJRU5ErkJggg=="

func TestGetGeminiContentUsesInlineBase64ImageCapability(t *testing.T) {
	imageURL := "data:image/png;base64," + palm2InlineBase64Png
	content := "show " + imageURL

	parts := getGeminiContent(nil, content, "gemini-2.5-flash")
	if len(parts) != 2 {
		t.Fatalf("expected text + inline image parts, got %#v", parts)
	}

	if parts[0].Text == nil || strings.TrimSpace(*parts[0].Text) != "show" {
		t.Fatalf("expected visible text part, got %#v", parts[0])
	}

	if parts[1].InlineData == nil {
		t.Fatalf("expected inline image part, got %#v", parts[1])
	}
	if parts[1].InlineData.MimeType != "image/png" {
		t.Fatalf("expected image mime type image/png, got %q", parts[1].InlineData.MimeType)
	}
	if parts[1].InlineData.Data != palm2InlineBase64Png {
		t.Fatalf("expected inline image raw base64 %q, got %q", palm2InlineBase64Png, parts[1].InlineData.Data)
	}
}

func TestGetGeminiChatTextPrefersFinalImage(t *testing.T) {
	previousImageStore := globals.AcceptImageStore
	globals.AcceptImageStore = false
	defer func() {
		globals.AcceptImageStore = previousImageStore
	}()

	const thoughtImageData = "dGhvdWdodA=="
	const finalImageData = "ZmluYWw="
	instance := &ChatInstance{}
	parts := []GeminiChatPart{
		{Text: ptrString("planning"), Thought: true},
		{
			Thought: true,
			InlineData: &GeminiInlineData{
				MimeType: "image/png",
				Data:     thoughtImageData,
			},
		},
		{
			InlineData: &GeminiInlineData{
				MimeType: "image/png",
				Data:     finalImageData,
			},
			ThoughtSignature: ptrString("sig"),
		},
	}

	content := instance.GetGeminiChatText(globals.Gemini3ProImage, parts)
	if !strings.Contains(content, "<think>") || !strings.Contains(content, "planning") {
		t.Fatalf("expected reasoning text to remain, got %q", content)
	}
	expectedFinalImage := "![image](data:image/png;base64," + finalImageData + ")"
	if !strings.Contains(content, expectedFinalImage) {
		t.Fatalf("expected final image in answer channel, got %q", content)
	}
	if strings.Contains(content, thoughtImageData) {
		t.Fatalf("did not expect intermediate thought image, got %q", content)
	}
}

func TestGetGeminiChatTextFallsBackToLastThoughtImage(t *testing.T) {
	previousImageStore := globals.AcceptImageStore
	globals.AcceptImageStore = false
	defer func() {
		globals.AcceptImageStore = previousImageStore
	}()

	const firstThoughtImageData = "Zmlyc3Q="
	const lastThoughtImageData = "bGFzdA=="
	instance := &ChatInstance{}
	parts := []GeminiChatPart{
		{Text: ptrString("planning"), Thought: true},
		{
			Thought: true,
			InlineData: &GeminiInlineData{
				MimeType: "image/png",
				Data:     firstThoughtImageData,
			},
		},
		{
			Thought: true,
			InlineData: &GeminiInlineData{
				MimeType: "image/png",
				Data:     lastThoughtImageData,
			},
		},
	}

	content := instance.GetGeminiChatText(globals.Gemini3ProImage, parts)
	if !strings.Contains(content, lastThoughtImageData) {
		t.Fatalf("expected last thought image as fallback, got %q", content)
	}
	if strings.Contains(content, firstThoughtImageData) {
		t.Fatalf("did not expect earlier thought image, got %q", content)
	}
	if strings.Count(content, "![image](") != 1 {
		t.Fatalf("expected exactly one fallback image, got %q", content)
	}
}

func TestGetGeminiStreamTextDefersThoughtImageFallback(t *testing.T) {
	previousImageStore := globals.AcceptImageStore
	globals.AcceptImageStore = false
	defer func() {
		globals.AcceptImageStore = previousImageStore
	}()

	instance := &ChatInstance{isFirstReasoning: true}
	parts := []GeminiChatPart{
		{Text: ptrString("planning"), Thought: true},
		{
			Thought: true,
			InlineData: &GeminiInlineData{
				MimeType: "image/png",
				Data:     palm2InlineBase64Png,
			},
		},
	}

	content := instance.GetGeminiStreamText(globals.Gemini3ProImage, parts)
	expectedImage := "![image](data:image/png;base64," + palm2InlineBase64Png + ")"
	if !strings.Contains(content, "planning") || strings.Contains(content, expectedImage) {
		t.Fatalf("expected stream to defer thought image, got %q", content)
	}
	if fallback := instance.takeGeminiStreamFallbackImage(); fallback != expectedImage {
		t.Fatalf("expected deferred thought image fallback, got %q", fallback)
	}
	if fallback := instance.takeGeminiStreamFallbackImage(); fallback != "" {
		t.Fatalf("expected fallback to be consumed once, got %q", fallback)
	}
}

func TestGetGeminiStreamTextDiscardsThoughtFallbackAfterFinalImage(t *testing.T) {
	previousImageStore := globals.AcceptImageStore
	globals.AcceptImageStore = false
	defer func() {
		globals.AcceptImageStore = previousImageStore
	}()

	instance := &ChatInstance{isFirstReasoning: true}
	instance.GetGeminiStreamText(globals.Gemini3ProImage, []GeminiChatPart{
		{
			Thought: true,
			InlineData: &GeminiInlineData{
				MimeType: "image/png",
				Data:     "dGhvdWdodA==",
			},
		},
	})
	content := instance.GetGeminiStreamText(globals.Gemini3ProImage, []GeminiChatPart{
		{
			InlineData: &GeminiInlineData{
				MimeType: "image/png",
				Data:     "ZmluYWw=",
			},
		},
	})

	if !strings.Contains(content, "ZmluYWw=") || strings.Contains(content, "dGhvdWdodA==") {
		t.Fatalf("expected only final image in stream, got %q", content)
	}
	if fallback := instance.takeGeminiStreamFallbackImage(); fallback != "" {
		t.Fatalf("did not expect thought fallback after final image, got %q", fallback)
	}
}
