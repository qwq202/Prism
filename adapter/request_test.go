package adapter

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
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

func TestSanitizeChatMessagesForRequestKeepsClaudeMetadataOnAnthropic(t *testing.T) {
	props := &adaptercommon.ChatProps{
		OriginalModel: "claude-sonnet-4-20250514",
		Message: []globals.Message{
			{
				Role:    globals.Assistant,
				Content: "<think>\nplan\n</think>\n\nAnswer",
				ClaudeHiddenMetadata: &globals.ClaudeHiddenMetadata{
					ThinkingBlocks: []globals.ClaudeThinkingBlock{
						{Thinking: "plan", Signature: "sig-a"},
					},
				},
			},
		},
	}

	restore := sanitizeChatMessagesForRequest(requestTestChannelConfig{
		channelType:    globals.ClaudeChannelType,
		reflectedModel: "claude-sonnet-4-20250514",
	}, props)

	if props.Message[0].ClaudeHiddenMetadata == nil {
		t.Fatalf("expected claude metadata to be preserved for anthropic requests")
	}

	restore()
	if props.Message[0].ClaudeHiddenMetadata == nil {
		t.Fatalf("expected claude metadata to remain after no-op restore")
	}
}

func TestSanitizeChatMessagesForRequestKeepsClaudeMetadataOnMiniMax(t *testing.T) {
	props := &adaptercommon.ChatProps{
		OriginalModel: "MiniMax-M2.1",
		Message: []globals.Message{
			{
				Role:    globals.Assistant,
				Content: "<think>\nplan\n</think>\n\nAnswer",
				ClaudeHiddenMetadata: &globals.ClaudeHiddenMetadata{
					ThinkingBlocks: []globals.ClaudeThinkingBlock{
						{Thinking: "plan", Signature: "sig-mini"},
					},
				},
			},
		},
	}

	restore := sanitizeChatMessagesForRequest(requestTestChannelConfig{
		channelType:    globals.MiniMaxTokenPlanCNChannelType,
		reflectedModel: "MiniMax-M2.1",
	}, props)

	if props.Message[0].ClaudeHiddenMetadata == nil {
		t.Fatalf("expected claude-style metadata to be preserved for minimax requests")
	}

	restore()
	if props.Message[0].ClaudeHiddenMetadata == nil {
		t.Fatalf("expected minimax metadata to remain after no-op restore")
	}
}

func TestSanitizeChatMessagesForRequestKeepsReasoningReplayForDeepseekV4(t *testing.T) {
	props := &adaptercommon.ChatProps{
		OriginalModel: globals.DeepseekV4Pro,
		Message: []globals.Message{
			{
				Role:             globals.Assistant,
				Model:            globals.DeepseekV4Pro,
				Content:          "<think>\nplan\n</think>\n\nAnswer",
				ReasoningContent: utils.ToPtr("plan"),
			},
			{Role: globals.User, Content: "继续"},
		},
	}

	restore := sanitizeChatMessagesForRequest(requestTestChannelConfig{
		channelType:    globals.DeepseekChannelType,
		reflectedModel: globals.DeepseekV4Pro,
	}, props)

	if props.Message[0].Content != "<think>\nplan\n</think>\n\nAnswer" {
		t.Fatalf("expected deepseek v4 thinking replay to remain, got %q", props.Message[0].Content)
	}

	if props.Message[0].ReasoningContent == nil || *props.Message[0].ReasoningContent != "plan" {
		t.Fatalf("expected deepseek v4 reasoning replay to remain, got %#v", props.Message[0].ReasoningContent)
	}

	restore()
	if props.Message[0].ReasoningContent == nil || *props.Message[0].ReasoningContent != "plan" {
		t.Fatalf("expected reasoning replay to remain after restore, got %#v", props.Message[0].ReasoningContent)
	}
}

func TestSanitizeChatMessagesForRequestStripsReasoningReplayForDeepseekV4NonThinking(t *testing.T) {
	props := &adaptercommon.ChatProps{
		OriginalModel: globals.DeepseekV4Flash,
		Thinking:      map[string]interface{}{"type": "disabled"},
		Message: []globals.Message{
			{
				Role:             globals.Assistant,
				Model:            globals.DeepseekV4Flash,
				Content:          "<think>\nplan\n</think>\n\nAnswer",
				ReasoningContent: utils.ToPtr("plan"),
			},
			{Role: globals.User, Content: "继续"},
		},
	}

	restore := sanitizeChatMessagesForRequest(requestTestChannelConfig{
		channelType:    globals.DeepseekChannelType,
		reflectedModel: globals.DeepseekV4Flash,
	}, props)

	if props.Message[0].Content != "Answer" {
		t.Fatalf("expected deepseek v4 non-thinking replay to be stripped, got %q", props.Message[0].Content)
	}

	if props.Message[0].ReasoningContent != nil {
		t.Fatalf("expected deepseek v4 non-thinking reasoning replay to be stripped, got %#v", props.Message[0].ReasoningContent)
	}

	restore()
	if props.Message[0].ReasoningContent == nil || *props.Message[0].ReasoningContent != "plan" {
		t.Fatalf("expected reasoning replay to be restored, got %#v", props.Message[0].ReasoningContent)
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

func TestSanitizeChatMessagesForRequestStripsOrphanedToolCallsButKeepsAssistantText(t *testing.T) {
	toolCalls := globals.ToolCalls{
		{
			Type: "function",
			Id:   "call_memory_1",
			Function: globals.ToolCallFunction{
				Name:      "memory_tool",
				Arguments: "{\"action\":\"create\"}",
			},
		},
	}

	props := &adaptercommon.ChatProps{
		OriginalModel: "deepseek-v4-flash",
		Message: []globals.Message{
			{Role: globals.User, Content: "你记一下"},
			{
				Role:      globals.Assistant,
				Content:   "已经帮你记录好了。",
				ToolCalls: &toolCalls,
			},
			{Role: globals.User, Content: "删除所有的"},
		},
	}

	restore := sanitizeChatMessagesForRequest(requestTestChannelConfig{
		channelType:    globals.DeepseekChannelType,
		reflectedModel: "deepseek-v4-flash",
	}, props)

	if got := len(props.Message); got != 3 {
		t.Fatalf("expected orphaned tool call cleanup to preserve message count, got %d", got)
	}

	if props.Message[1].ToolCalls != nil {
		t.Fatalf("expected orphaned tool_calls to be stripped, got %#v", props.Message[1].ToolCalls)
	}

	if props.Message[1].Content != "已经帮你记录好了。" {
		t.Fatalf("expected visible assistant text to remain, got %q", props.Message[1].Content)
	}

	restore()
	if props.Message[1].ToolCalls == nil || len(*props.Message[1].ToolCalls) != 1 {
		t.Fatalf("expected original orphaned tool_calls to be restored after request")
	}
}

func TestSanitizeChatMessagesForRequestKeepsMatchedToolCalls(t *testing.T) {
	toolCalls := globals.ToolCalls{
		{
			Type: "function",
			Id:   "call_memory_1",
			Function: globals.ToolCallFunction{
				Name:      "memory_tool",
				Arguments: "{\"action\":\"create\"}",
			},
		},
	}

	props := &adaptercommon.ChatProps{
		OriginalModel: "deepseek-v4-flash",
		Message: []globals.Message{
			{Role: globals.User, Content: "你记一下"},
			{
				Role:      globals.Assistant,
				Content:   "",
				ToolCalls: &toolCalls,
			},
			{
				Role:       globals.Tool,
				Content:    "{\"status\":\"success\"}",
				ToolCallId: utils.ToPtr("call_memory_1"),
			},
			{Role: globals.Assistant, Content: "已经帮你记录好了。"},
		},
	}

	restore := sanitizeChatMessagesForRequest(requestTestChannelConfig{
		channelType:    globals.DeepseekChannelType,
		reflectedModel: "deepseek-v4-flash",
	}, props)

	if props.Message[1].ToolCalls == nil || len(*props.Message[1].ToolCalls) != 1 {
		t.Fatalf("expected matched tool_calls to be preserved, got %#v", props.Message[1].ToolCalls)
	}

	if props.Message[2].Role != globals.Tool || props.Message[2].ToolCallId == nil || *props.Message[2].ToolCallId != "call_memory_1" {
		t.Fatalf("expected matching tool response to be preserved, got %#v", props.Message[2])
	}

	restore()
	if props.Message[1].ToolCalls == nil || len(*props.Message[1].ToolCalls) != 1 {
		t.Fatalf("expected matched tool_calls to remain after restore")
	}
}

func TestSanitizeChatMessagesForRequestDropsToolOnlyAssistantWithoutToolReply(t *testing.T) {
	toolCalls := globals.ToolCalls{
		{
			Type: "function",
			Id:   "call_lookup_1",
			Function: globals.ToolCallFunction{
				Name:      "lookup_weather",
				Arguments: "{\"city\":\"Shanghai\"}",
			},
		},
	}

	props := &adaptercommon.ChatProps{
		OriginalModel: "gpt-4o",
		Message: []globals.Message{
			{Role: globals.User, Content: "查天气"},
			{
				Role:      globals.Assistant,
				Content:   "",
				ToolCalls: &toolCalls,
			},
			{Role: globals.User, Content: "继续"},
		},
	}

	restore := sanitizeChatMessagesForRequest(requestTestChannelConfig{
		channelType:    globals.OpenAIChannelType,
		reflectedModel: "gpt-4o",
	}, props)

	if got := len(props.Message); got != 2 {
		t.Fatalf("expected tool-only orphaned assistant message to be removed, got %d messages", got)
	}

	if props.Message[1].Role != globals.User || props.Message[1].Content != "继续" {
		t.Fatalf("expected subsequent user message to remain after stripping orphaned tool call, got %#v", props.Message[1])
	}

	restore()
	if got := len(props.Message); got != 3 {
		t.Fatalf("expected original message history to be restored, got %d", got)
	}
}
