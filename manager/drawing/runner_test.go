package drawing

import (
	"chat/globals"
	"chat/utils"
	"testing"
)

func TestNewDrawingChatPropsDisablesCache(t *testing.T) {
	messages := []globals.Message{{Role: globals.User, Content: "draw a pig"}}
	buffer := &utils.Buffer{}
	responseFormat := map[string]interface{}{"type": "image", "aspect_ratio": "1:1"}
	thinking := map[string]interface{}{"thinking_level": "minimal"}

	props := newDrawingChatProps(
		"gemini-3-pro-image",
		messages,
		responseFormat,
		thinking,
		buffer,
	)

	if !props.DisableCache {
		t.Fatalf("expected drawing requests to disable response caching")
	}
	if props.Model != "gemini-3-pro-image" || props.OriginalModel != "gemini-3-pro-image" {
		t.Fatalf("unexpected drawing model props: %#v", props)
	}
	if props.Buffer != buffer {
		t.Fatalf("expected drawing props to retain the request buffer")
	}
	if props.ResponseFormat == nil || props.Thinking == nil {
		t.Fatalf("expected drawing request options to be preserved")
	}
}
