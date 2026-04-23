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
