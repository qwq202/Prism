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

func TestGetChatEndpointUsesMiMoOpenPlatformBaseURL(t *testing.T) {
	instance := NewMiMoChatInstance("", "sk-test")
	if got := instance.GetChatEndpoint(); got != "https://api.xiaomimimo.com/v1/chat/completions" {
		t.Fatalf("unexpected default MiMo endpoint: %s", got)
	}

	instance = NewMiMoChatInstance("https://api.xiaomimimo.com", "sk-test")
	if got := instance.GetChatEndpoint(); got != "https://api.xiaomimimo.com/v1/chat/completions" {
		t.Fatalf("unexpected bare MiMo endpoint: %s", got)
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

func TestGetHeaderUsesMiMoAPIKey(t *testing.T) {
	headers := NewMiMoChatInstance("", "sk-test").GetHeader()
	if got := headers["api-key"]; got != "sk-test" {
		t.Fatalf("expected api-key header, got %q", got)
	}
	if _, ok := headers["Authorization"]; ok {
		t.Fatalf("did not expect Authorization header for official MiMo endpoint: %#v", headers)
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

func TestProcessLineUsesMiMoErrorPrefix(t *testing.T) {
	instance := NewMiMoChatInstance("", "sk-test")

	_, err := instance.ProcessLine(`{"error":{"message":"invalid key","type":"auth_error"}}`)
	if err == nil {
		t.Fatal("expected stream error")
	}
	if !strings.Contains(err.Error(), "xiaomi mimo error: invalid key") {
		t.Fatalf("expected MiMo error prefix, got %q", err.Error())
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
