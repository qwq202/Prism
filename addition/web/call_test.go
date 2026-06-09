package web

import (
	"chat/globals"
	"strings"
	"testing"
)

func withSearchAPIKey(t *testing.T, value string) {
	t.Helper()
	previous := globals.SearchApiKey
	globals.SearchApiKey = value
	t.Cleanup(func() {
		globals.SearchApiKey = previous
	})
}

func TestShouldUseFallbackSearchWhenToolCallsUnsupported(t *testing.T) {
	withSearchAPIKey(t, "tvly-test")

	if !ShouldUseFallbackSearch(true, "gpt-4o", false) {
		t.Fatalf("expected fallback search when web is enabled and tool calls are unsupported")
	}
	if ShouldUseFallbackSearch(true, "gpt-4o", true) {
		t.Fatalf("expected fallback search to stay off when tool calls are supported")
	}
	if ShouldUseFallbackSearch(false, "gpt-4o", false) {
		t.Fatalf("expected fallback search to stay off when web is disabled")
	}
}

func TestShouldUseFallbackSearchRequiresAPIKey(t *testing.T) {
	withSearchAPIKey(t, " ")

	if ShouldUseFallbackSearch(true, "gpt-4o", false) {
		t.Fatalf("expected fallback search to stay off without Tavily API key")
	}
}

func TestShouldUseFallbackSearchSkipsNativeWebProviders(t *testing.T) {
	withSearchAPIKey(t, "tvly-test")

	for _, model := range []string{"gemini-3.5-flash", "grok-4-1-fast-reasoning", "gpt-5.5"} {
		if ShouldUseFallbackSearch(true, model, false) {
			t.Fatalf("expected fallback search to stay off for native web provider model %q", model)
		}
	}
}

func TestCanUseSearchToolRequiresToolCalls(t *testing.T) {
	withSearchAPIKey(t, "tvly-test")

	if !CanUseSearchTool(true, "gpt-4o", true) {
		t.Fatalf("expected search tool when web is enabled and tool calls are supported")
	}
	if CanUseSearchTool(true, "gpt-4o", false) {
		t.Fatalf("expected search tool to stay off without tool call support")
	}
}

func TestRecentUserSearchContextUsesLatestThreeUserMessages(t *testing.T) {
	messages := []globals.Message{
		{Role: globals.User, Content: "第一条用户消息"},
		{Role: globals.Assistant, Content: "助手回复"},
		{Role: globals.User, Content: "第二条用户消息"},
		{Role: globals.System, Content: "系统消息"},
		{Role: globals.User, Content: "第三条用户消息"},
		{Role: globals.User, Content: "第四条用户消息"},
	}

	got := recentUserSearchContext(messages, 3)
	want := strings.Join([]string{
		"Recent user messages:",
		"1. 第二条用户消息",
		"2. 第三条用户消息",
		"3. 第四条用户消息",
	}, "\n")

	if got != want {
		t.Fatalf("expected recent user context %q, got %q", want, got)
	}
}

func TestRecentUserSearchContextFallsBackToLastMessage(t *testing.T) {
	got := recentUserSearchContext([]globals.Message{
		{Role: globals.System, Content: "系统消息"},
		{Role: globals.Assistant, Content: "助手回复"},
	}, 3)

	if got != "助手回复" {
		t.Fatalf("expected fallback to last message, got %q", got)
	}
}

func TestBuildToolDefinitionExposesWebSearchQuery(t *testing.T) {
	tools := BuildToolDefinition()
	if tools == nil || len(*tools) != 1 {
		t.Fatalf("expected one web search tool, got %#v", tools)
	}

	tool := (*tools)[0]
	if tool.Function.Name != SearchToolName {
		t.Fatalf("expected web search tool name %q, got %q", SearchToolName, tool.Function.Name)
	}
	if tool.Function.Parameters.Required == nil || len(*tool.Function.Parameters.Required) != 1 || (*tool.Function.Parameters.Required)[0] != "query" {
		t.Fatalf("expected query to be required, got %#v", tool.Function.Parameters.Required)
	}
	if _, ok := tool.Function.Parameters.Properties["query"]; !ok {
		t.Fatalf("expected query property in tool parameters, got %#v", tool.Function.Parameters.Properties)
	}
}

func TestExecuteToolCallRejectsEmptyQuery(t *testing.T) {
	message := ExecuteToolCall(globals.ToolCall{
		Id:   "call_search",
		Type: "function",
		Function: globals.ToolCallFunction{
			Name:      SearchToolName,
			Arguments: `{"query":"   "}`,
		},
	})

	if message.ToolCallId == nil || *message.ToolCallId != "call_search" {
		t.Fatalf("expected tool call id to be preserved, got %#v", message.ToolCallId)
	}
	if !strings.Contains(message.Content, "query is required") {
		t.Fatalf("expected empty query error, got %s", message.Content)
	}
}

func TestToTavilyUsageViewCalculatesRemainingBalance(t *testing.T) {
	var usage TavilyUsageResponse
	usage.Key.Usage = 125
	usage.Key.Limit = 1000
	usage.Key.SearchUsage = 100
	usage.Account.CurrentPlan = "Project"

	got := toTavilyUsageView(&usage)

	if got.Remaining != 875 || got.Percent != 87.5 {
		t.Fatalf("expected 875 remaining and 87.5 percent, got %#v", got)
	}
	if got.SearchUsage != 100 || got.CurrentPlan != "Project" {
		t.Fatalf("expected usage details to be preserved, got %#v", got)
	}
}

func TestToTavilyUsageViewClampsNegativeBalance(t *testing.T) {
	var usage TavilyUsageResponse
	usage.Key.Usage = 1200
	usage.Key.Limit = 1000

	got := toTavilyUsageView(&usage)

	if got.Remaining != 0 || got.Percent != 0 {
		t.Fatalf("expected over-limit key to show zero remaining, got %#v", got)
	}
}
