package openairesponses

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"testing"
)

func requireInputItems(t *testing.T, input interface{}) []interface{} {
	t.Helper()

	items, ok := input.([]interface{})
	if ok {
		return items
	}

	t.Fatalf("expected []interface{} input payload, got %#v", input)
	return nil
}

func TestGetChatBodyMapsResponseFormatToTextFormat(t *testing.T) {
	instance := &ChatInstance{}
	props := &adaptercommon.ChatProps{
		Model:           "gpt-5.4",
		ResponseFormat:  map[string]interface{}{"type": "json_schema", "name": "answer"},
		Thinking:        map[string]interface{}{"effort": "medium"},
		EnableWebSearch: true,
		ResponseInclude: []string{
			"reasoning.encrypted_content",
		},
		Message: []globals.Message{
			{Role: globals.System, Content: "你是一个有帮助的助手"},
			{Role: globals.User, Content: "你好"},
		},
	}

	body := instance.GetChatBody(props, true)
	if body.Text == nil {
		t.Fatalf("expected structured outputs config to map into text.format")
	}
	textMap, ok := body.Text.(map[string]interface{})
	if !ok {
		t.Fatalf("expected text config map, got %#v", body.Text)
	}
	if textMap["format"] == nil {
		t.Fatalf("expected text.format to be populated, got %#v", textMap)
	}
	if body.Reasoning == nil {
		t.Fatalf("expected reasoning config to pass through")
	}
	if len(body.Tools) != 1 || body.Tools[0].Type != "web_search" {
		t.Fatalf("expected builtin web_search tool, got %#v", body.Tools)
	}
	if len(body.Include) != 1 || body.Include[0] != "reasoning.encrypted_content" {
		t.Fatalf("expected include passthrough, got %#v", body.Include)
	}
	if body.Instructions == nil || *body.Instructions != "你是一个有帮助的助手" {
		t.Fatalf("expected system message to map into instructions, got %#v", body.Instructions)
	}
}

func TestGetChatBodyIncludesPromptCacheParams(t *testing.T) {
	instance := &ChatInstance{}
	cacheKey := "stable-prefix"
	cacheRetention := "24h"
	props := &adaptercommon.ChatProps{
		Model:                "gpt-5.1",
		PromptCacheKey:       &cacheKey,
		PromptCacheRetention: &cacheRetention,
		Message: []globals.Message{
			{Role: globals.User, Content: "hello"},
		},
	}

	body := instance.GetChatBody(props, false)
	if body.PromptCacheKey == nil || *body.PromptCacheKey != cacheKey {
		t.Fatalf("expected prompt_cache_key to be included, got %#v", body.PromptCacheKey)
	}
	if body.PromptCacheRetention == nil || *body.PromptCacheRetention != cacheRetention {
		t.Fatalf("expected prompt_cache_retention to be included, got %#v", body.PromptCacheRetention)
	}
}

func TestGetChatBodyUsesGPT56PromptCacheOptions(t *testing.T) {
	instance := &ChatInstance{}
	cacheKey := "stable-prefix"
	cacheRetention := "24h"
	props := &adaptercommon.ChatProps{
		Model:                "gpt-5.6",
		PromptCacheKey:       &cacheKey,
		PromptCacheRetention: &cacheRetention,
		Message: []globals.Message{
			{Role: globals.User, Content: "hello"},
		},
	}

	body := instance.GetChatBody(props, false)
	if body.PromptCacheRetention != nil {
		t.Fatalf("expected gpt-5.6 to omit deprecated prompt_cache_retention, got %#v", body.PromptCacheRetention)
	}
	options, ok := body.PromptCacheOptions.(map[string]string)
	if !ok || options["ttl"] != "30m" {
		t.Fatalf("expected gpt-5.6 prompt cache options, got %#v", body.PromptCacheOptions)
	}
}

func TestGetChatBodySkipsNativeWebToolForUnsupportedModel(t *testing.T) {
	instance := &ChatInstance{}
	props := &adaptercommon.ChatProps{
		Model:           "o1",
		EnableWebSearch: true,
		Message: []globals.Message{
			{Role: globals.User, Content: "你好"},
		},
	}

	body := instance.GetChatBody(props, false)
	if len(body.Tools) != 0 {
		t.Fatalf("expected no builtin web_search tool for unsupported model, got %#v", body.Tools)
	}
}

func TestGetChatBodyDropsSamplingForGPT5Reasoning(t *testing.T) {
	instance := &ChatInstance{}
	temperature := float32(0.2)
	topP := float32(0.8)
	props := &adaptercommon.ChatProps{
		Model:       "gpt-5",
		Temperature: &temperature,
		TopP:        &topP,
		Thinking:    map[string]interface{}{"effort": "low"},
		Message: []globals.Message{
			{Role: globals.User, Content: "你好"},
		},
	}

	body := instance.GetChatBody(props, false)
	if body.Temperature != nil || body.TopP != nil {
		t.Fatalf("expected sampling params to be stripped for gpt-5 reasoning, got temp=%#v topP=%#v", body.Temperature, body.TopP)
	}
}

func TestGetChatBodySupportsGPT55ReasoningAndWebSearch(t *testing.T) {
	instance := &ChatInstance{}
	temperature := float32(0.2)
	topP := float32(0.8)
	props := &adaptercommon.ChatProps{
		Model:           "gpt-5.5",
		Temperature:     &temperature,
		TopP:            &topP,
		Thinking:        map[string]interface{}{"effort": "xhigh"},
		EnableWebSearch: true,
		Message: []globals.Message{
			{Role: globals.User, Content: "联网查一下今天的 OpenAI 文档"},
		},
	}

	body := instance.GetChatBody(props, false)
	if body.Temperature != nil || body.TopP != nil {
		t.Fatalf("expected sampling params to be stripped for gpt-5.5 reasoning, got temp=%#v topP=%#v", body.Temperature, body.TopP)
	}
	if body.Reasoning == nil {
		t.Fatalf("expected gpt-5.5 reasoning config to pass through")
	}
	if len(body.Tools) != 1 || body.Tools[0].Type != "web_search" {
		t.Fatalf("expected gpt-5.5 to use hosted web_search tool, got %#v", body.Tools)
	}
}

func TestGetChatBodySupportsGPT54MiniReasoningAndWebSearch(t *testing.T) {
	instance := &ChatInstance{}
	temperature := float32(0.2)
	topP := float32(0.8)
	props := &adaptercommon.ChatProps{
		Model:           "gpt-5.4-mini",
		Temperature:     &temperature,
		TopP:            &topP,
		Thinking:        map[string]interface{}{"effort": "xhigh"},
		EnableWebSearch: true,
		Message: []globals.Message{
			{Role: globals.User, Content: "联网查一下 OpenAI 文档"},
		},
	}

	body := instance.GetChatBody(props, false)
	if body.Temperature != nil || body.TopP != nil {
		t.Fatalf("expected sampling params to be stripped for gpt-5.4-mini reasoning, got temp=%#v topP=%#v", body.Temperature, body.TopP)
	}
	if body.Reasoning == nil {
		t.Fatalf("expected gpt-5.4-mini reasoning config to pass through")
	}
	if len(body.Tools) != 1 || body.Tools[0].Type != "web_search" {
		t.Fatalf("expected gpt-5.4-mini to use hosted web_search tool, got %#v", body.Tools)
	}
}

func TestGetChatBodyDropsSamplingForGPT54NoReasoning(t *testing.T) {
	instance := &ChatInstance{}
	temperature := float32(0.2)
	topP := float32(0.8)
	props := &adaptercommon.ChatProps{
		Model:       "gpt-5.4",
		Temperature: &temperature,
		TopP:        &topP,
		Thinking:    map[string]interface{}{"effort": "none"},
		Message: []globals.Message{
			{Role: globals.User, Content: "你好"},
		},
	}

	body := instance.GetChatBody(props, false)
	if body.Temperature != nil || body.TopP != nil {
		t.Fatalf("expected sampling params to be stripped when gpt-5.4 reasoning is none, got temp=%#v topP=%#v", body.Temperature, body.TopP)
	}
}

func TestGetChatBodyDropsSamplingForGPT55NoReasoning(t *testing.T) {
	instance := &ChatInstance{}
	temperature := float32(0.2)
	topP := float32(0.8)
	props := &adaptercommon.ChatProps{
		Model:       "gpt-5.5",
		Temperature: &temperature,
		TopP:        &topP,
		Thinking:    map[string]interface{}{"effort": "none"},
		Message: []globals.Message{
			{Role: globals.User, Content: "你好"},
		},
	}

	body := instance.GetChatBody(props, false)
	if body.Temperature != nil || body.TopP != nil {
		t.Fatalf("expected sampling params to be stripped when gpt-5.5 reasoning is none, got temp=%#v topP=%#v", body.Temperature, body.TopP)
	}
}

func TestGetChatBodyDropsSamplingForGPT52NoReasoning(t *testing.T) {
	instance := &ChatInstance{}
	temperature := float32(0.2)
	topP := float32(0.8)
	props := &adaptercommon.ChatProps{
		Model:       "gpt-5.2",
		Temperature: &temperature,
		TopP:        &topP,
		Thinking:    map[string]interface{}{"effort": "none"},
		Message: []globals.Message{
			{Role: globals.User, Content: "你好"},
		},
	}

	body := instance.GetChatBody(props, false)
	if body.Temperature != nil || body.TopP != nil {
		t.Fatalf("expected sampling params to be stripped when gpt-5.2 reasoning is none, got temp=%#v topP=%#v", body.Temperature, body.TopP)
	}
}

func TestGetChatBodyOnlyKeepsSamplingForGPT51NoneReasoning(t *testing.T) {
	instance := &ChatInstance{}
	temperature := float32(0.2)
	topP := float32(0.8)
	props := &adaptercommon.ChatProps{
		Model:       "gpt-5.1",
		Temperature: &temperature,
		TopP:        &topP,
		Thinking:    map[string]interface{}{"effort": "none"},
		Message: []globals.Message{
			{Role: globals.User, Content: "你好"},
		},
	}

	body := instance.GetChatBody(props, false)
	if body.Temperature != &temperature || body.TopP != &topP {
		t.Fatalf("expected sampling params to be preserved for gpt-5.1 none reasoning, got temp=%#v topP=%#v", body.Temperature, body.TopP)
	}

	props.Thinking = map[string]interface{}{"effort": "high"}
	body = instance.GetChatBody(props, false)
	if body.Temperature != nil || body.TopP != nil {
		t.Fatalf("expected sampling params to be stripped for gpt-5.1 high reasoning, got temp=%#v topP=%#v", body.Temperature, body.TopP)
	}
}

func TestGetChatBodyDropsSamplingForGPT54Pro(t *testing.T) {
	instance := &ChatInstance{}
	temperature := float32(0.2)
	topP := float32(0.8)
	props := &adaptercommon.ChatProps{
		Model:       "gpt-5.4-pro",
		Temperature: &temperature,
		TopP:        &topP,
		Message: []globals.Message{
			{Role: globals.User, Content: "你好"},
		},
	}

	body := instance.GetChatBody(props, false)
	if body.Temperature != nil || body.TopP != nil {
		t.Fatalf("expected sampling params to be stripped for gpt-5.4-pro, got temp=%#v topP=%#v", body.Temperature, body.TopP)
	}
}

func TestGetChatBodyReplaysFunctionCallAndOutput(t *testing.T) {
	instance := &ChatInstance{}
	toolCalls := globals.ToolCalls{
		{
			Type: "function",
			Id:   "call_1",
			Function: globals.ToolCallFunction{
				Name:      "memory_tool",
				Arguments: `{"action":"create"}`,
			},
		},
	}
	props := &adaptercommon.ChatProps{
		Model: "gpt-5.4",
		Message: []globals.Message{
			{Role: globals.User, Content: "记住我的偏好"},
			{Role: globals.Assistant, ToolCalls: &toolCalls},
			{Role: globals.Tool, ToolCallId: toPtr("call_1"), Content: `{"status":"success"}`},
		},
	}

	body := instance.GetChatBody(props, false)
	items := requireInputItems(t, body.Input)
	if len(items) != 3 {
		t.Fatalf("expected user + function_call + function_call_output items, got %#v", items)
	}
	if _, ok := items[0].(InputMessage); !ok {
		t.Fatalf("expected first item to be user input message, got %#v", items[0])
	}
	functionCall, ok := items[1].(OutputItem)
	if !ok || functionCall.Type != "function_call" || functionCall.CallID != "call_1" {
		t.Fatalf("expected replayed function_call item, got %#v", items[1])
	}
	functionOutput, ok := items[2].(FunctionCallOutputInput)
	if !ok || functionOutput.Type != "function_call_output" || functionOutput.CallID != "call_1" {
		t.Fatalf("expected function_call_output item, got %#v", items[2])
	}
}

func TestGetChatBodyEncodesAssistantHistoryAsOutputText(t *testing.T) {
	instance := &ChatInstance{}
	props := &adaptercommon.ChatProps{
		Model: "gpt-5.4",
		Message: []globals.Message{
			{Role: globals.User, Content: "你好"},
			{Role: globals.Assistant, Content: "你好，我是 GPT-5.4"},
		},
	}

	body := instance.GetChatBody(props, false)
	items := requireInputItems(t, body.Input)
	if len(items) != 2 {
		t.Fatalf("expected user + assistant history items, got %#v", items)
	}

	assistantMessage, ok := items[1].(InputMessage)
	if !ok {
		t.Fatalf("expected second item to be assistant input message, got %#v", items[1])
	}
	if assistantMessage.Role != globals.Assistant {
		t.Fatalf("expected assistant role, got %#v", assistantMessage.Role)
	}
	if len(assistantMessage.Content) != 1 || assistantMessage.Content[0].Type != "output_text" {
		t.Fatalf("expected assistant history to use output_text, got %#v", assistantMessage.Content)
	}
}

func TestBuildResponseChunkExtractsFunctionCalls(t *testing.T) {
	form := &ResponseResponse{
		Output: []OutputItem{
			{
				Type:      "function_call",
				Name:      "memory_tool",
				Arguments: `{"action":"create"}`,
				CallID:    "call_1",
			},
		},
	}

	chunk := buildResponseChunk(form)
	if chunk.ToolCall == nil || len(*chunk.ToolCall) != 1 {
		t.Fatalf("expected function calls to be extracted, got %#v", chunk.ToolCall)
	}
	if (*chunk.ToolCall)[0].Function.Name != "memory_tool" {
		t.Fatalf("unexpected function call payload: %#v", (*chunk.ToolCall)[0])
	}
}

func TestBuildResponseChunkIncludesReasoningSummary(t *testing.T) {
	form := &ResponseResponse{
		Output: []OutputItem{
			{
				Type: "reasoning",
				Summary: []ReasoningSummaryContent{
					{Type: "summary_text", Text: "核对输入。"},
				},
			},
			{
				Type: "message",
				Role: globals.Assistant,
				Content: []OutputContent{
					{Type: "output_text", Text: "完成"},
				},
			},
		},
	}

	chunk := buildResponseChunk(form)
	expected := "<think>\n核对输入。\n</think>\n\n完成"
	if chunk.Content != expected {
		t.Fatalf("expected reasoning summary content, got %q", chunk.Content)
	}
}

func TestEmitReasoningSummaryAndOutputText(t *testing.T) {
	started := false
	closed := false

	summary := emitReasoningSummary("先想一步", &started)
	if summary == nil || summary.Content != "<think>\n先想一步" || !started {
		t.Fatalf("unexpected reasoning summary chunk: %#v started=%v", summary, started)
	}

	answer := emitOutputText("答案", &started, &closed)
	if answer == nil || answer.Content != "\n</think>\n\n答案" || !closed {
		t.Fatalf("unexpected output text chunk: %#v closed=%v", answer, closed)
	}
}

func TestEmitFunctionCallEvent(t *testing.T) {
	chunk := emitFunctionCallEvent(&OutputItem{
		Type:      "function_call",
		Name:      "memory_tool",
		Arguments: `{"action":"create"}`,
		CallID:    "call_1",
	})
	if chunk == nil || chunk.ToolCall == nil || len(*chunk.ToolCall) != 1 {
		t.Fatalf("expected tool call chunk, got %#v", chunk)
	}
}

func toPtr(v string) *string {
	return &v
}
