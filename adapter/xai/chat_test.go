package xai

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

func TestEmitReasoningSummaryStartsThinkBlock(t *testing.T) {
	started := false

	chunk := emitReasoningSummary("step 1", &started)
	if chunk == nil {
		t.Fatalf("expected reasoning chunk")
	}
	if chunk.Content != "<think>\nstep 1" {
		t.Fatalf("unexpected reasoning chunk content: %q", chunk.Content)
	}
	if !started {
		t.Fatalf("expected reasoning block to be marked as started")
	}
}

func TestEmitOutputTextClosesThinkBlockBeforeAnswer(t *testing.T) {
	reasoningStarted := true
	reasoningClosed := false

	chunk := emitOutputText("final answer", &reasoningStarted, &reasoningClosed)
	if chunk == nil {
		t.Fatalf("expected output chunk")
	}
	if chunk.Content != "\n</think>\n\nfinal answer" {
		t.Fatalf("unexpected output chunk content: %q", chunk.Content)
	}
	if !reasoningClosed {
		t.Fatalf("expected reasoning block to be closed")
	}
}

func TestEmitOutputTextWithoutReasoningLeavesAnswerUntouched(t *testing.T) {
	reasoningStarted := false
	reasoningClosed := false

	chunk := emitOutputText("final answer", &reasoningStarted, &reasoningClosed)
	if chunk == nil {
		t.Fatalf("expected output chunk")
	}
	if chunk.Content != "final answer" {
		t.Fatalf("unexpected output chunk content: %q", chunk.Content)
	}
}

func TestGetChatBodyDisablesStoreForImageRequests(t *testing.T) {
	instance := &ChatInstance{}
	props := &adaptercommon.ChatProps{
		Model: "grok-4-1-fast-reasoning",
		Message: []globals.Message{
			{
				Role:    globals.User,
				Content: "![image](https://example.com/test.png)\n这是什么",
			},
		},
	}

	body := instance.GetChatBody(props, false)
	if body.Store == nil || *body.Store {
		t.Fatalf("expected xai image requests to set store=false")
	}
	items := requireInputItems(t, body.Input)
	if len(items) != 1 {
		t.Fatalf("unexpected input payload shape: %#v", body.Input)
	}
	message, ok := items[0].(InputMessage)
	if !ok {
		t.Fatalf("expected input message item, got %#v", items[0])
	}
	if len(message.Content) != 2 {
		t.Fatalf("unexpected input payload shape: %#v", message)
	}
}

func TestGetChatBodyLeavesStoreUnsetForTextOnlyRequests(t *testing.T) {
	instance := &ChatInstance{}
	props := &adaptercommon.ChatProps{
		Model: "grok-4-1-fast-reasoning",
		Message: []globals.Message{
			{
				Role:    globals.User,
				Content: "你好",
			},
		},
	}

	body := instance.GetChatBody(props, false)
	_ = requireInputItems(t, body.Input)
	if body.Store != nil {
		t.Fatalf("expected text-only xai requests to leave store unset")
	}
}

func TestGetChatBodyKeepsSystemMessagesInInput(t *testing.T) {
	instance := &ChatInstance{}
	props := &adaptercommon.ChatProps{
		Model: "grok-4-1-fast-reasoning",
		Message: []globals.Message{
			{
				Role:    globals.System,
				Content: "你是一个有帮助的助手",
			},
			{
				Role:    globals.User,
				Content: "你好",
			},
		},
	}

	body := instance.GetChatBody(props, false)
	items := requireInputItems(t, body.Input)
	if len(items) != 2 {
		t.Fatalf("expected system+user input messages, got %#v", body.Input)
	}
	first, ok := items[0].(InputMessage)
	if !ok {
		t.Fatalf("expected first xai item to be input message, got %#v", items[0])
	}
	if first.Role != globals.System {
		t.Fatalf("expected first xai input role to stay system, got %q", first.Role)
	}
	if first.Content[0].Text == nil || *first.Content[0].Text != "你是一个有帮助的助手" {
		t.Fatalf("expected system prompt to stay in input, got %#v", first.Content)
	}
}

func TestGetChatBodyAddsImageAndVideoUnderstandingTools(t *testing.T) {
	instance := &ChatInstance{}
	props := &adaptercommon.ChatProps{
		Model:           "grok-4-1-fast-reasoning",
		EnableWebSearch: true,
		EnableXSearch:   true,
		ResponseFormat:  map[string]interface{}{"type": "json_object"},
		Message: []globals.Message{
			{
				Role:    globals.User,
				Content: "帮我搜一下",
			},
		},
	}

	body := instance.GetChatBody(props, true)
	if len(body.Tools) != 2 {
		t.Fatalf("expected 2 builtin xai tools, got %#v", body.Tools)
	}
	if body.Tools[0].Type != "web_search" || body.Tools[0].EnableImageUnderstanding == nil || !*body.Tools[0].EnableImageUnderstanding {
		t.Fatalf("expected web_search to enable image understanding, got %#v", body.Tools[0])
	}
	if body.Tools[1].Type != "x_search" || body.Tools[1].EnableVideoUnderstanding == nil || !*body.Tools[1].EnableVideoUnderstanding {
		t.Fatalf("expected x_search to enable video understanding, got %#v", body.Tools[1])
	}
	if len(body.Include) != 1 || body.Include[0] != "verbose_streaming" {
		t.Fatalf("expected xai stream body to request verbose streaming, got %#v", body.Include)
	}
	if body.ResponseFormat == nil {
		t.Fatalf("expected response format to pass through")
	}
}

func TestGetChatBodyOmitsVerboseStreamingForNonReasoningModel(t *testing.T) {
	instance := &ChatInstance{}
	props := &adaptercommon.ChatProps{
		Model:           "grok-4-1-fast-non-reasoning",
		EnableWebSearch: true,
		Message: []globals.Message{
			{
				Role:    globals.User,
				Content: "帮我搜一下",
			},
		},
	}

	body := instance.GetChatBody(props, true)
	if len(body.Tools) != 1 || body.Tools[0].Type != "web_search" {
		t.Fatalf("expected builtin web_search tool, got %#v", body.Tools)
	}
	if body.Include != nil {
		t.Fatalf("expected non-reasoning xai model to omit include, got %#v", body.Include)
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
		Model: "grok-4-1-fast-reasoning",
		Message: []globals.Message{
			{Role: globals.User, Content: "记住我的偏好"},
			{Role: globals.Assistant, ToolCalls: &toolCalls},
			{Role: globals.Tool, ToolCallId: globalsToPtr("call_1"), Content: `{"status":"success"}`},
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

func globalsToPtr(v string) *string {
	return &v
}
