package xiaomitokenplan

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"encoding/json"
	"strings"
	"testing"
)

func TestGetChatEndpointUsesTokenPlanBaseURL(t *testing.T) {
	instance := NewChatInstance("", "tp-test")
	if got := instance.GetChatEndpoint(); got != "https://token-plan-cn.xiaomimimo.com/v1/chat/completions" {
		t.Fatalf("unexpected default endpoint: %s", got)
	}

	instance = NewChatInstance("https://token-plan-cn.xiaomimimo.com/v1", "tp-test")
	if got := instance.GetChatEndpoint(); got != "https://token-plan-cn.xiaomimimo.com/v1/chat/completions" {
		t.Fatalf("unexpected official endpoint: %s", got)
	}

	instance = NewChatInstance("https://proxy.example.com", "tp-test")
	if got := instance.GetChatEndpoint(); got != "https://proxy.example.com/v1/chat/completions" {
		t.Fatalf("unexpected proxy endpoint: %s", got)
	}
}

func TestGetHeaderUsesTokenPlanAPIKey(t *testing.T) {
	headers := NewChatInstance("", "tp-test").GetHeader()
	if got := headers["api-key"]; got != "tp-test" {
		t.Fatalf("expected api-key header, got %q", got)
	}
	if _, ok := headers["Authorization"]; ok {
		t.Fatalf("did not expect Authorization header: %#v", headers)
	}
}

func TestGetHeaderAddsBearerForCustomEndpoint(t *testing.T) {
	headers := NewChatInstance("https://api.example.com", "sk-test").GetHeader()
	if got := headers["api-key"]; got != "sk-test" {
		t.Fatalf("expected api-key header, got %q", got)
	}
	if got := headers["Authorization"]; got != "Bearer sk-test" {
		t.Fatalf("expected bearer authorization for custom endpoint, got %q", got)
	}
}

func TestGetChatBodyPreservesXiaomiReasoningContent(t *testing.T) {
	maxTokens := 1024
	thinking := map[string]string{"type": "disabled"}
	props := &adaptercommon.ChatProps{
		Model:     "mimo-v2.5-pro",
		MaxTokens: &maxTokens,
		Thinking:  thinking,
		Message: []globals.Message{
			{
				Role:             globals.Assistant,
				Content:          "Answer",
				ReasoningContent: utils.ToPtr("plan"),
			},
			{Role: globals.User, Content: "Continue"},
		},
	}

	body := NewChatInstance("", "tp-test").GetChatBody(props, true)
	if body.MaxCompletionTokens == nil || *body.MaxCompletionTokens != maxTokens {
		t.Fatalf("expected max_completion_tokens %d, got %#v", maxTokens, body.MaxCompletionTokens)
	}
	if body.Thinking == nil {
		t.Fatalf("expected thinking config to be preserved")
	}

	messages, ok := body.Messages.([]Message)
	if !ok || len(messages) != 2 {
		t.Fatalf("unexpected message payload: %#v", body.Messages)
	}
	if messages[0].ReasoningContent == nil || *messages[0].ReasoningContent != "plan" {
		t.Fatalf("expected reasoning_content replay, got %#v", messages[0].ReasoningContent)
	}

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	if !strings.Contains(string(raw), `"max_completion_tokens":1024`) {
		t.Fatalf("expected max_completion_tokens in body: %s", raw)
	}
	if strings.Contains(string(raw), `"max_tokens"`) {
		t.Fatalf("did not expect max_tokens in body: %s", raw)
	}
}

func TestProcessLineStreamsReasoningContent(t *testing.T) {
	instance := NewChatInstance("", "tp-test")

	chunk, err := instance.ProcessLine(`{"choices":[{"delta":{"reasoning_content":"plan"},"index":0}]}`)
	if err != nil {
		t.Fatalf("unexpected reasoning chunk error: %v", err)
	}
	if chunk.Content != "<think>\nplan" {
		t.Fatalf("unexpected reasoning content chunk: %q", chunk.Content)
	}
	if chunk.ReasoningContent == nil || *chunk.ReasoningContent != "plan" {
		t.Fatalf("expected reasoning content, got %#v", chunk.ReasoningContent)
	}

	chunk, err = instance.ProcessLine(`{"choices":[{"delta":{"content":"Answer"},"index":0}]}`)
	if err != nil {
		t.Fatalf("unexpected content chunk error: %v", err)
	}
	if chunk.Content != "\n</think>\n\nAnswer" {
		t.Fatalf("expected reasoning close before content, got %q", chunk.Content)
	}
}

func requireSingleToolCall(t *testing.T, calls *globals.ToolCalls) globals.ToolCall {
	t.Helper()
	if calls == nil || len(*calls) != 1 {
		t.Fatalf("expected one tool call, got %#v", calls)
	}
	return (*calls)[0]
}

func requireToolArguments(t *testing.T, call globals.ToolCall) map[string]string {
	t.Helper()
	args := map[string]string{}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		t.Fatalf("expected JSON tool arguments, got %q: %v", call.Function.Arguments, err)
	}
	return args
}

func TestProcessLineExtractsTextToolCallFromReasoningContent(t *testing.T) {
	instance := NewChatInstance("", "tp-test")

	chunk, err := instance.ProcessLine(`{"choices":[{"delta":{"reasoning_content":"<tool_call>\n<function=webfetch>\n<parameter=url>https://www.weather.com.cn/weather/101280901.shtml</parameter>\n<parameter=format>text</parameter>\n</function>\n</tool_call>"},"index":0}]}`)
	if err != nil {
		t.Fatalf("unexpected text tool call chunk error: %v", err)
	}

	if chunk.Content != "" {
		t.Fatalf("expected text tool call markup to be hidden, got %q", chunk.Content)
	}
	if chunk.ReasoningContent != nil {
		t.Fatalf("expected text tool call markup to be stripped from reasoning, got %#v", chunk.ReasoningContent)
	}

	call := requireSingleToolCall(t, chunk.ToolCall)
	if call.Type != "function" || call.Id == "" || call.Index == nil {
		t.Fatalf("unexpected text tool call metadata: %#v", call)
	}
	if call.Function.Name != "fetch_webpage" {
		t.Fatalf("expected webfetch alias to map to fetch_webpage, got %q", call.Function.Name)
	}
	args := requireToolArguments(t, call)
	if args["url"] != "https://www.weather.com.cn/weather/101280901.shtml" {
		t.Fatalf("unexpected url argument: %#v", args)
	}
	if args["format"] != "text" {
		t.Fatalf("unexpected format argument: %#v", args)
	}
}

func TestProcessLineStripsTextToolCallFromMixedReasoningContent(t *testing.T) {
	instance := NewChatInstance("", "tp-test")

	reasoning := "先抓取网页。\n<tool_call>\n<function=fetch_webpage>\n<parameter=url>https://example.com</parameter>\n</function>\n</tool_call>\n拿到内容后再总结。"
	raw, err := json.Marshal(map[string]interface{}{
		"choices": []map[string]interface{}{
			{
				"delta": map[string]string{
					"reasoning_content": reasoning,
				},
				"index": 0,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal test payload: %v", err)
	}

	chunk, err := instance.ProcessLine(string(raw))
	if err != nil {
		t.Fatalf("unexpected mixed reasoning chunk error: %v", err)
	}

	if chunk.Content != "<think>\n先抓取网页。\n拿到内容后再总结。" {
		t.Fatalf("expected clean reasoning content chunk, got %q", chunk.Content)
	}
	if chunk.ReasoningContent == nil || *chunk.ReasoningContent != "先抓取网页。\n拿到内容后再总结。" {
		t.Fatalf("expected clean reasoning content, got %#v", chunk.ReasoningContent)
	}

	call := requireSingleToolCall(t, chunk.ToolCall)
	if call.Function.Name != "fetch_webpage" {
		t.Fatalf("expected fetch_webpage tool name, got %q", call.Function.Name)
	}
	args := requireToolArguments(t, call)
	if args["url"] != "https://example.com" {
		t.Fatalf("unexpected url argument: %#v", args)
	}
}

func TestProcessLineBuffersSplitTextToolCallFromReasoningContent(t *testing.T) {
	instance := NewChatInstance("", "tp-test")

	lines := []string{
		`{"choices":[{"delta":{"reasoning_content":"<tool_call>\n"},"index":0}]}`,
		`{"choices":[{"delta":{"reasoning_content":"<function=search>\n<parameter=query>江门 今天 天气 2026年5月20日</parameter>\n"},"index":0}]}`,
		`{"choices":[{"delta":{"reasoning_content":"<parameter=type>web</parameter>\n</function>\n</tool_call>"},"index":0}]}`,
	}

	for i, line := range lines[:2] {
		chunk, err := instance.ProcessLine(line)
		if err != nil {
			t.Fatalf("unexpected split text tool chunk %d error: %v", i, err)
		}
		if chunk.Content != "" || chunk.ReasoningContent != nil || chunk.ToolCall != nil {
			t.Fatalf("expected split text tool chunk %d to be buffered, got %#v", i, chunk)
		}
	}

	chunk, err := instance.ProcessLine(lines[2])
	if err != nil {
		t.Fatalf("unexpected final split text tool chunk error: %v", err)
	}
	if chunk.Content != "" {
		t.Fatalf("expected split text tool markup to stay hidden, got %q", chunk.Content)
	}
	if chunk.ReasoningContent != nil {
		t.Fatalf("expected split text tool markup to be stripped from reasoning, got %#v", chunk.ReasoningContent)
	}

	call := requireSingleToolCall(t, chunk.ToolCall)
	if call.Function.Name != "search" {
		t.Fatalf("expected search tool name, got %q", call.Function.Name)
	}
	args := requireToolArguments(t, call)
	if args["query"] != "江门 今天 天气 2026年5月20日" {
		t.Fatalf("unexpected query argument: %#v", args)
	}
	if args["type"] != "web" {
		t.Fatalf("unexpected type argument: %#v", args)
	}
}

func TestCollectResponseExtractsTextToolCallFromReasoningContent(t *testing.T) {
	reasoning := "<tool_call>\n<function=fetch_webpage>\n<parameter=url>https://example.com</parameter>\n</function>\n</tool_call>"
	chunk, err := collectResponse(ChatStreamResponse{
		Choices: []struct {
			Delta        ResponseMessage `json:"delta"`
			Index        int             `json:"index"`
			FinishReason string          `json:"finish_reason"`
		}{
			{
				Delta: ResponseMessage{
					ReasoningContent: &reasoning,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected collect response error: %v", err)
	}

	if chunk.Content != "" {
		t.Fatalf("expected hidden text tool call markup, got %q", chunk.Content)
	}
	if chunk.ReasoningContent != nil {
		t.Fatalf("expected stripped reasoning content, got %#v", chunk.ReasoningContent)
	}

	call := requireSingleToolCall(t, chunk.ToolCall)
	if call.Function.Name != "fetch_webpage" {
		t.Fatalf("expected fetch_webpage tool name, got %q", call.Function.Name)
	}
	args := requireToolArguments(t, call)
	if args["url"] != "https://example.com" {
		t.Fatalf("unexpected url argument: %#v", args)
	}
}

func findToolCallByID(calls globals.ToolCalls, id string) *globals.ToolCall {
	for i := range calls {
		if calls[i].Id == id {
			return &calls[i]
		}
	}
	return nil
}

func TestProcessLineNormalizesInterleavedToolCallDeltas(t *testing.T) {
	instance := NewChatInstance("", "tp-test")
	buffer := &utils.Buffer{}

	lines := []string{
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_search","type":"function","function":{"name":"search","arguments":""}}]},"index":0}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":1,"id":"call_calc","type":"function","function":{"name":"calculate","arguments":""}}]},"index":0}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"query\":\"mi"}}]},"index":0}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":1,"function":{"arguments":"{\"expr\":\"1+"}}]},"index":0}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"mo\"}"}}]},"index":0}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":1,"function":{"arguments":"1\"}"}}]},"index":0}]}`,
	}

	for _, line := range lines {
		chunk, err := instance.ProcessLine(line)
		if err != nil {
			t.Fatalf("unexpected tool call chunk error: %v", err)
		}
		buffer.WriteChunk(chunk)
	}

	calls := buffer.GetToolCalls()
	if calls == nil || len(*calls) != 2 {
		t.Fatalf("expected two tool calls, got %#v", calls)
	}

	search := findToolCallByID(*calls, "call_search")
	if search == nil {
		t.Fatalf("expected call_search in %#v", *calls)
	}
	if search.Function.Name != "search" {
		t.Fatalf("expected search tool name, got %q", search.Function.Name)
	}
	if search.Function.Arguments != `{"query":"mimo"}` {
		t.Fatalf("expected search arguments to stay on index 0, got %q", search.Function.Arguments)
	}

	calculate := findToolCallByID(*calls, "call_calc")
	if calculate == nil {
		t.Fatalf("expected call_calc in %#v", *calls)
	}
	if calculate.Function.Name != "calculate" {
		t.Fatalf("expected calculate tool name, got %q", calculate.Function.Name)
	}
	if calculate.Function.Arguments != `{"expr":"1+1"}` {
		t.Fatalf("expected calculate arguments to stay on index 1, got %q", calculate.Function.Arguments)
	}
}
