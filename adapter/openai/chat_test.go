package openai

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"strings"
	"testing"
)

func TestGetChatEndpointAvoidsDuplicatingV1(t *testing.T) {
	props := &adaptercommon.ChatProps{Model: "gpt-4o-mini"}
	instance := NewChatInstance("https://api.openai.com", "sk-test")

	if got := instance.GetChatEndpoint(props); got != "https://api.openai.com/v1/chat/completions" {
		t.Fatalf("expected OpenAI chat endpoint, got %q", got)
	}

	instance = NewChatInstance("https://proxy.example.com/v1", "sk-test")
	if got := instance.GetChatEndpoint(props); got != "https://proxy.example.com/v1/chat/completions" {
		t.Fatalf("expected existing v1 endpoint to be reused, got %q", got)
	}
}

func TestOpenRouterUsesDocumentedBaseURL(t *testing.T) {
	props := &adaptercommon.ChatProps{Model: "openai/gpt-4o-mini"}
	instance := NewOpenRouterChatInstance("https://openrouter.ai/api/v1", "sk-or-test")

	if got := instance.GetChatEndpoint(props); got != "https://openrouter.ai/api/v1/chat/completions" {
		t.Fatalf("expected OpenRouter chat endpoint, got %q", got)
	}

	instance = NewOpenRouterChatInstance("https://openrouter.ai", "sk-or-test")
	if got := instance.GetChatEndpoint(props); got != "https://openrouter.ai/api/v1/chat/completions" {
		t.Fatalf("expected bare OpenRouter host to normalize, got %q", got)
	}
}

func TestOpenRouterHeaders(t *testing.T) {
	headers := NewOpenRouterChatInstance("", "sk-or-test").GetHeader()

	if got := headers["Authorization"]; got != "Bearer sk-or-test" {
		t.Fatalf("expected bearer authorization, got %q", got)
	}
	if got := headers["X-OpenRouter-Title"]; got != "Prism" {
		t.Fatalf("expected OpenRouter title header, got %q", got)
	}
}

func TestOpenRouterStreamErrorUsesProviderPrefix(t *testing.T) {
	instance := NewOpenRouterChatInstance("", "sk-or-test")

	_, err := instance.ProcessLine(`{"error":{"message":"No auth credentials found","type":"invalid_request_error"}}`, false)
	if err == nil {
		t.Fatal("expected stream error")
	}
	if !strings.Contains(err.Error(), "openrouter error: No auth credentials found") {
		t.Fatalf("expected OpenRouter error prefix, got %q", err.Error())
	}
}

func TestProcessLineNormalizesPromptCacheUsage(t *testing.T) {
	instance := NewChatInstance("https://api.openai.com", "sk-test")

	chunk, err := instance.ProcessLine(`{"choices":[],"usage":{"prompt_tokens":100,"completion_tokens":10,"total_tokens":110,"prompt_tokens_details":{"cached_tokens":64}}}`, false)
	if err != nil {
		t.Fatalf("expected usage-only stream chunk to parse, got %v", err)
	}
	if chunk.Usage == nil {
		t.Fatalf("expected usage chunk")
	}
	if chunk.Usage.PromptCacheHitTokens != 64 || chunk.Usage.PromptCacheMissTokens != 36 {
		t.Fatalf("expected cached prompt usage to be normalized, got %#v", chunk.Usage)
	}
	if chunk.Usage.PromptTokensDetails != nil {
		t.Fatalf("expected provider-specific prompt details to be normalized away, got %#v", chunk.Usage.PromptTokensDetails)
	}
}

func TestSiliconFlowUsesDocumentedBaseURL(t *testing.T) {
	props := &adaptercommon.ChatProps{Model: "Qwen/Qwen3-Coder-480B-A35B-Instruct"}
	instance := NewSiliconFlowChatInstance("", "sk-sf-test")

	if got := instance.GetChatEndpoint(props); got != "https://api.siliconflow.cn/v1/chat/completions" {
		t.Fatalf("expected SiliconFlow default chat endpoint, got %q", got)
	}

	instance = NewSiliconFlowChatInstance("https://api.siliconflow.cn", "sk-sf-test")
	if got := instance.GetChatEndpoint(props); got != "https://api.siliconflow.cn/v1/chat/completions" {
		t.Fatalf("expected bare SiliconFlow host to use v1 endpoint, got %q", got)
	}

	instance = NewSiliconFlowChatInstance("https://api.siliconflow.cn/v1", "sk-sf-test")
	if got := instance.GetChatEndpoint(props); got != "https://api.siliconflow.cn/v1/chat/completions" {
		t.Fatalf("expected existing SiliconFlow v1 endpoint to be reused, got %q", got)
	}
}

func TestSiliconFlowHeaders(t *testing.T) {
	headers := NewSiliconFlowChatInstance("", "sk-sf-test").GetHeader()

	if got := headers["Authorization"]; got != "Bearer sk-sf-test" {
		t.Fatalf("expected bearer authorization, got %q", got)
	}
}

func TestSiliconFlowStreamErrorUsesProviderPrefix(t *testing.T) {
	instance := NewSiliconFlowChatInstance("", "sk-sf-test")

	_, err := instance.ProcessLine(`{"error":{"message":"invalid api key","type":"invalid_request_error"}}`, false)
	if err == nil {
		t.Fatal("expected stream error")
	}
	if !strings.Contains(err.Error(), "siliconflow error: invalid api key") {
		t.Fatalf("expected SiliconFlow error prefix, got %q", err.Error())
	}
}

func TestSiliconFlowRequestsStreamUsageByDefault(t *testing.T) {
	instance := NewSiliconFlowChatInstance("", "sk-sf-test")

	body, ok := instance.GetChatBody(&adaptercommon.ChatProps{
		Model:   "Pro/zai-org/GLM-4.7",
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
