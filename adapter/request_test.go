package adapter

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"strings"
	"testing"
)

type requestTestChannelConfig struct {
	channelType    string
	reflectedModel string
}

func (c requestTestChannelConfig) GetType() string {
	return c.channelType
}

func (c requestTestChannelConfig) GetModelReflect(model string) string {
	if c.reflectedModel != "" {
		return c.reflectedModel
	}
	return model
}

func (c requestTestChannelConfig) GetRetry() int {
	return 1
}

func (c requestTestChannelConfig) GetRandomSecret() string {
	return ""
}

func (c requestTestChannelConfig) SplitRandomSecret(_ int) []string {
	return nil
}

func (c requestTestChannelConfig) GetEndpoint() string {
	return ""
}

func (c requestTestChannelConfig) ProcessError(err error) error {
	return err
}

func (c requestTestChannelConfig) GetId() int {
	return 1
}

func (c requestTestChannelConfig) GetProxy() globals.ProxyConfig {
	return globals.ProxyConfig{}
}

func TestSanitizeChatMessagesForRequestStripsNonGeminiMetadata(t *testing.T) {
	props := &adaptercommon.ChatProps{
		OriginalModel: "gpt-4o",
		Message: []globals.Message{
			{
				Role:    globals.User,
				Content: "hello",
			},
			{
				Role:    globals.Assistant,
				Content: "",
				GeminiHiddenMetadata: &globals.GeminiHiddenMetadata{
					ThoughtSignatures: []string{"sig-a"},
				},
			},
		},
	}

	original := props.Message
	restore := sanitizeChatMessagesForRequest(requestTestChannelConfig{
		channelType:    globals.OpenAIChannelType,
		reflectedModel: "gpt-4o",
	}, props)

	if props.Message[1].GeminiHiddenMetadata != nil {
		t.Fatalf("expected non-gemini request metadata to be stripped, got %#v", props.Message[1].GeminiHiddenMetadata)
	}

	restore()
	if props.Message[1].GeminiHiddenMetadata == nil {
		t.Fatalf("expected original metadata to be restored")
	}
	if props.Message[1].GeminiHiddenMetadata.ThoughtSignatures[0] != "sig-a" {
		t.Fatalf("expected restored signature, got %#v", props.Message[1].GeminiHiddenMetadata.ThoughtSignatures)
	}

	if len(props.Message) != len(original) {
		t.Fatalf("expected message length to remain unchanged")
	}
}

func TestSanitizeChatMessagesForRequestKeepsGeminiMetadataOnPalmGemini(t *testing.T) {
	props := &adaptercommon.ChatProps{
		OriginalModel: "gemini-2.5-pro",
		Message: []globals.Message{
			{
				Role:    globals.Assistant,
				Content: "",
				GeminiHiddenMetadata: &globals.GeminiHiddenMetadata{
					ThoughtSignatures: []string{"sig-a"},
				},
			},
		},
	}

	restore := sanitizeChatMessagesForRequest(requestTestChannelConfig{
		channelType:    globals.PalmChannelType,
		reflectedModel: "gemini-2.5-pro",
	}, props)

	if props.Message[0].GeminiHiddenMetadata == nil {
		t.Fatalf("expected gemini request metadata to be preserved")
	}

	restore()
	if props.Message[0].GeminiHiddenMetadata == nil {
		t.Fatalf("expected metadata to remain after no-op restore")
	}
}

func TestSanitizeChatMessagesForRequestStripsPalmNonGeminiModel(t *testing.T) {
	props := &adaptercommon.ChatProps{
		OriginalModel: "text-bison-001",
		Message: []globals.Message{
			{
				Role:    globals.Assistant,
				Content: "",
				GeminiHiddenMetadata: &globals.GeminiHiddenMetadata{
					ThoughtSignatures: []string{"sig-a"},
				},
			},
		},
	}

	restore := sanitizeChatMessagesForRequest(requestTestChannelConfig{
		channelType:    globals.PalmChannelType,
		reflectedModel: "text-bison-001",
	}, props)

	if props.Message[0].GeminiHiddenMetadata != nil {
		t.Fatalf("expected non-gemini model metadata to be stripped on palm channel")
	}

	restore()
	if props.Message[0].GeminiHiddenMetadata == nil {
		t.Fatalf("expected metadata to be restored after request")
	}
}

func TestClearMessagesKeepsBase64ForConfiguredVisionModel(t *testing.T) {
	originalResolver := globals.VisionModelResolver
	globals.VisionModelResolver = func(model string) bool {
		return model == "custom-vision-model"
	}
	defer func() {
		globals.VisionModelResolver = originalResolver
	}()

	image := "data:image/png;base64," + strings.Repeat("A", 128)
	messages := []globals.Message{
		{
			Role:    globals.User,
			Content: "before " + image + " after",
		},
	}

	cleared := ClearMessages("custom-vision-model", messages)
	if cleared[0].Content != messages[0].Content {
		t.Fatalf("expected configured vision model to preserve base64 image content")
	}
}
