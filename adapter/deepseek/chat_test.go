package deepseek

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"reflect"
	"testing"
)

func TestGetChatEndpointNormalizesCustomRelay(t *testing.T) {
	instance := NewChatInstance("https://api.example.com/", "secret")
	if got := instance.GetChatEndpoint(); got != "https://api.example.com/v1/chat/completions" {
		t.Fatalf("unexpected custom relay endpoint: %s", got)
	}

	instance = NewChatInstance("https://api.example.com/v1", "secret")
	if got := instance.GetChatEndpoint(); got != "https://api.example.com/v1/chat/completions" {
		t.Fatalf("unexpected custom v1 endpoint: %s", got)
	}
}

func TestGetChatEndpointPreservesOfficialEndpoint(t *testing.T) {
	instance := NewChatInstance("https://api.deepseek.com", "secret")
	if got := instance.GetChatEndpoint(); got != "https://api.deepseek.com/chat/completions" {
		t.Fatalf("unexpected official endpoint: %s", got)
	}

	instance = NewChatInstance("https://api.deepseek.com/v1", "secret")
	if got := instance.GetChatEndpoint(); got != "https://api.deepseek.com/v1/chat/completions" {
		t.Fatalf("unexpected official v1 endpoint: %s", got)
	}
}

func TestGetChatBodyStripsUnsupportedParamsForDeepseekV4ThinkingByDefault(t *testing.T) {
	instance := NewChatInstance("https://api.deepseek.com", "secret")
	effort := "high"

	body, ok := instance.GetChatBody(&adaptercommon.ChatProps{
		Model:            globals.DeepseekV4Pro,
		Message:          []globals.Message{{Role: globals.User, Content: "hello"}},
		Temperature:      utils.ToPtr(float32(0.7)),
		TopP:             utils.ToPtr(float32(0.9)),
		PresencePenalty:  utils.ToPtr(float32(0.1)),
		FrequencyPenalty: utils.ToPtr(float32(0.2)),
		Logprobs:         utils.ToPtr(true),
		TopLogprobs:      utils.ToPtr(2),
		ReasoningEffort:  &effort,
	}, false).(ChatRequest)
	if !ok {
		t.Fatalf("expected ChatRequest body")
	}

	if body.Temperature != nil || body.TopP != nil || body.PresencePenalty != nil || body.FrequencyPenalty != nil {
		t.Fatalf("expected sampling parameters to be stripped in deepseek v4 thinking mode, got %#v", body)
	}
	if body.Logprobs != nil || body.TopLogprobs != nil {
		t.Fatalf("expected logprobs parameters to be stripped in deepseek v4 thinking mode, got %#v", body)
	}
	if body.ReasoningEffort == nil || *body.ReasoningEffort != "high" {
		t.Fatalf("expected reasoning_effort to be preserved in thinking mode, got %#v", body.ReasoningEffort)
	}
}

func TestGetChatBodyKeepsSamplingForDeepseekV4NonThinking(t *testing.T) {
	instance := NewChatInstance("https://api.deepseek.com", "secret")
	effort := "high"

	body, ok := instance.GetChatBody(&adaptercommon.ChatProps{
		Model:            globals.DeepseekV4Flash,
		Message:          []globals.Message{{Role: globals.User, Content: "hello"}},
		Temperature:      utils.ToPtr(float32(0.7)),
		TopP:             utils.ToPtr(float32(0.9)),
		PresencePenalty:  utils.ToPtr(float32(0.1)),
		FrequencyPenalty: utils.ToPtr(float32(0.2)),
		Logprobs:         utils.ToPtr(true),
		TopLogprobs:      utils.ToPtr(2),
		ReasoningEffort:  &effort,
		Thinking:         map[string]interface{}{"type": "disabled"},
	}, false).(ChatRequest)
	if !ok {
		t.Fatalf("expected ChatRequest body")
	}

	if body.Temperature == nil || body.TopP == nil || body.PresencePenalty == nil || body.FrequencyPenalty == nil {
		t.Fatalf("expected sampling parameters to remain in deepseek v4 non-thinking mode, got %#v", body)
	}
	if body.Logprobs == nil || body.TopLogprobs == nil {
		t.Fatalf("expected logprobs parameters to remain in deepseek v4 non-thinking mode, got %#v", body)
	}
	if body.ReasoningEffort != nil {
		t.Fatalf("expected reasoning_effort to be omitted outside thinking mode, got %#v", body.ReasoningEffort)
	}
}

func TestGetChatBodyRequestsStreamUsageByDefault(t *testing.T) {
	instance := NewChatInstance("https://api.deepseek.com", "secret")

	body, ok := instance.GetChatBody(&adaptercommon.ChatProps{
		Model:   globals.DeepseekV4Flash,
		Message: []globals.Message{{Role: globals.User, Content: "hello"}},
	}, true).(ChatRequest)
	if !ok {
		t.Fatalf("expected ChatRequest body")
	}

	options, ok := body.StreamOptions.(map[string]bool)
	if !ok || !options["include_usage"] {
		t.Fatalf("expected stream_options.include_usage to be enabled, got %#v", body.StreamOptions)
	}
}

func TestGetChatBodyPreservesExplicitStreamOptions(t *testing.T) {
	instance := NewChatInstance("https://api.deepseek.com", "secret")
	streamOptions := map[string]interface{}{"include_usage": false}

	body, ok := instance.GetChatBody(&adaptercommon.ChatProps{
		Model:         globals.DeepseekV4Flash,
		Message:       []globals.Message{{Role: globals.User, Content: "hello"}},
		StreamOptions: streamOptions,
	}, true).(ChatRequest)
	if !ok {
		t.Fatalf("expected ChatRequest body")
	}

	if !reflect.DeepEqual(body.StreamOptions, streamOptions) {
		t.Fatalf("expected explicit stream options to be preserved, got %#v", body.StreamOptions)
	}
}

func TestSanitizeDSMLToolMarkup(t *testing.T) {
	input := "Let me fetch it. </ | DSML | tool_calls>\n< | DSML | invoke name=\"fetch_webpage\">"
	got := sanitizeDSMLToolMarkup(input)
	if got != "Let me fetch it. " {
		t.Fatalf("expected DSML markup to be stripped, got %q", got)
	}
}

func TestGetChoicesReturnsUsageOnlyChunk(t *testing.T) {
	instance := NewChatInstance("https://api.deepseek.com", "secret")

	chunk := instance.getChoices(&ChatStreamResponse{
		Usage: &globals.TokenUsage{
			PromptTokens:          30,
			CompletionTokens:      7,
			TotalTokens:           37,
			PromptCacheHitTokens:  20,
			PromptCacheMissTokens: 10,
		},
	})

	if chunk.Usage == nil {
		t.Fatalf("expected usage to be preserved")
	}
	if chunk.Usage.PromptCacheHitTokens != 20 || chunk.Usage.PromptCacheMissTokens != 10 {
		t.Fatalf("unexpected prompt cache usage: %#v", chunk.Usage)
	}
	if chunk.Content != "" || chunk.ToolCall != nil {
		t.Fatalf("expected usage-only chunk to have no visible payload, got %#v", chunk)
	}
}

func TestGetChoicesNormalizesPromptTokensDetailsUsage(t *testing.T) {
	instance := NewChatInstance("https://api.deepseek.com", "secret")

	chunk := instance.getChoices(&ChatStreamResponse{
		Usage: &globals.TokenUsage{
			PromptTokens:     30,
			CompletionTokens: 7,
			PromptTokensDetails: &globals.PromptTokensDetails{
				CachedTokens: 20,
			},
		},
	})

	if chunk.Usage == nil {
		t.Fatalf("expected usage to be preserved")
	}
	if chunk.Usage.PromptCacheHitTokens != 20 || chunk.Usage.PromptCacheMissTokens != 10 {
		t.Fatalf("unexpected normalized prompt cache usage: %#v", chunk.Usage)
	}
	if chunk.Usage.PromptTokensDetails != nil {
		t.Fatalf("expected prompt token details to be flattened, got %#v", chunk.Usage.PromptTokensDetails)
	}
}

func TestSanitizeDSMLToolMarkupPreservesWhitespaceWithoutMarker(t *testing.T) {
	input := "\n\n## 标题\n\n- 第一项\n"
	got := sanitizeDSMLToolMarkup(input)
	if got != input {
		t.Fatalf("expected whitespace to be preserved, got %q", got)
	}
}

func TestSanitizeDSMLToolMarkupPreservesTrailingHtmlTagStart(t *testing.T) {
	input := "    <"
	got := sanitizeDSMLToolMarkup(input)
	if got != input {
		t.Fatalf("expected trailing html tag start to be preserved, got %q", got)
	}
}

func TestSanitizeDeepseekStreamTextRemovesNestedThinkTags(t *testing.T) {
	input := "先想一下 < THINK >\ninner\n</ Think > 继续"
	got := sanitizeDeepseekStreamText(input)
	if got != "先想一下 \ninner\n 继续" {
		t.Fatalf("expected raw think tags to be stripped, got %q", got)
	}
}

func TestGetChoicesStripsDSMLFromContentWithToolCalls(t *testing.T) {
	instance := NewChatInstance("https://api.deepseek.com", "secret")
	instance.isFirstReasoning = false
	instance.isReasonOver = false

	calls := globals.ToolCalls{
		{
			Id:   "call_1",
			Type: "function",
			Function: globals.ToolCallFunction{
				Name:      "fetch_webpage",
				Arguments: `{"url":"https://example.com"}`,
			},
		},
	}
	content := "I found a likely source. </ | DSML | tool_calls>"
	chunk := instance.getChoices(&ChatStreamResponse{
		Choices: []struct {
			Delta        globals.Message `json:"delta"`
			Index        int             `json:"index"`
			FinishReason string          `json:"finish_reason"`
			Logprobs     interface{}     `json:"logprobs,omitempty"`
		}{
			{
				Delta: globals.Message{
					Content:   content,
					ToolCalls: &calls,
				},
			},
		},
	})

	if chunk.ToolCall == nil || len(*chunk.ToolCall) != 1 {
		t.Fatalf("expected tool call to be preserved, got %#v", chunk.ToolCall)
	}
	if chunk.Content != "\n</think>\n\nI found a likely source. " {
		t.Fatalf("expected visible DSML markup to be stripped, got %q", chunk.Content)
	}
}

func TestGetChoicesStripsNestedThinkTagsFromReasoningWithToolCalls(t *testing.T) {
	instance := NewChatInstance("https://api.deepseek.com", "secret")

	calls := globals.ToolCalls{
		{
			Id:   "call_1",
			Type: "function",
			Function: globals.ToolCallFunction{
				Name:      "fetch_webpage",
				Arguments: `{"url":"https://example.com"}`,
			},
		},
	}
	reasoning := "Need a source. <think>\nsearching\n</think>"
	chunk := instance.getChoices(&ChatStreamResponse{
		Choices: []struct {
			Delta        globals.Message `json:"delta"`
			Index        int             `json:"index"`
			FinishReason string          `json:"finish_reason"`
			Logprobs     interface{}     `json:"logprobs,omitempty"`
		}{
			{
				Delta: globals.Message{
					ReasoningContent: &reasoning,
					ToolCalls:        &calls,
				},
			},
		},
	})

	if chunk.ToolCall == nil || len(*chunk.ToolCall) != 1 {
		t.Fatalf("expected tool call to be preserved, got %#v", chunk.ToolCall)
	}
	if chunk.Content != "<think>\nNeed a source. \nsearching\n" {
		t.Fatalf("expected nested raw think tags to be stripped, got %q", chunk.Content)
	}
	if chunk.ReasoningContent == nil || *chunk.ReasoningContent != "Need a source. \nsearching\n" {
		t.Fatalf("expected clean reasoning content, got %#v", chunk.ReasoningContent)
	}
}
