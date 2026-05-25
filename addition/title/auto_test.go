package title

import (
	"chat/globals"
	"strings"
	"testing"
)

func TestBuildTitleContextStripsFileImageBlocks(t *testing.T) {
	context := buildTitleContext([]globals.Message{
		{
			Role: globals.User,
			Content: "```file\n[[image.png]]\n" +
				"data:image/png;base64," + strings.Repeat("A", 512) +
				"\n```\n\n这是好事啊",
		},
		{
			Role:    globals.Assistant,
			Content: "我看到的是一张空白图片。",
		},
	})

	if strings.Contains(context, "data:image") || strings.Contains(context, "base64") || strings.Contains(context, "image.png") {
		t.Fatalf("expected title context to strip image attachment payload, got %q", context)
	}
	if !strings.Contains(context, "这是好事啊") {
		t.Fatalf("expected visible user text to remain, got %q", context)
	}
	if !strings.Contains(context, "我看到的是一张空白图片") {
		t.Fatalf("expected assistant text to remain, got %q", context)
	}
}

func TestBuildTitleContextStripsStandaloneImageDataBeforeTruncation(t *testing.T) {
	context := buildTitleContext([]globals.Message{
		{
			Role: globals.User,
			Content: "data:image/png;base64," + strings.Repeat("B", 512) +
				" 请总结这张图",
		},
	})

	if strings.Contains(context, "data:image") || strings.Contains(context, "base64") {
		t.Fatalf("expected standalone image data to be stripped before title truncation, got %q", context)
	}
	if !strings.Contains(context, "请总结这张图") {
		t.Fatalf("expected trailing text to survive stripping, got %q", context)
	}
}
