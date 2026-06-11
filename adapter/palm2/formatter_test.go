package palm2

import (
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
