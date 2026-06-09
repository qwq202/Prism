package web

import (
	"chat/globals"
	"strings"
	"testing"
)

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
