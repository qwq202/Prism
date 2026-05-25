package fetch

import (
	"chat/globals"
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildToolDefinition(t *testing.T) {
	tools := BuildToolDefinition()
	if tools == nil || len(*tools) != 1 {
		t.Fatalf("expected one fetch tool, got %#v", tools)
	}

	tool := (*tools)[0]
	if tool.Type != "function" {
		t.Fatalf("expected function tool, got %q", tool.Type)
	}
	if tool.Function.Name != ToolName {
		t.Fatalf("expected tool name %q, got %q", ToolName, tool.Function.Name)
	}
	if _, ok := tool.Function.Parameters.Properties["url"]; !ok {
		t.Fatalf("expected url parameter in fetch tool schema")
	}
	if tool.Function.Parameters.Required == nil || len(*tool.Function.Parameters.Required) != 1 || (*tool.Function.Parameters.Required)[0] != "url" {
		t.Fatalf("expected url to be required, got %#v", tool.Function.Parameters.Required)
	}
}

func TestExecuteToolCallRejectsInvalidArguments(t *testing.T) {
	message := ExecuteToolCall(globals.ToolCall{
		Id:   "call_1",
		Type: "function",
		Function: globals.ToolCallFunction{
			Name:      ToolName,
			Arguments: `{`,
		},
	})

	if message.Role != globals.Tool {
		t.Fatalf("expected tool message, got %q", message.Role)
	}
	if message.ToolCallId == nil || *message.ToolCallId != "call_1" {
		t.Fatalf("expected tool call id to be preserved, got %#v", message.ToolCallId)
	}
	if !strings.Contains(message.Content, "invalid tool arguments") {
		t.Fatalf("expected invalid arguments error, got %s", message.Content)
	}
}

func TestExecuteToolCallBlocksLocalNetworkURL(t *testing.T) {
	message := ExecuteToolCall(globals.ToolCall{
		Id:   "call_2",
		Type: "function",
		Function: globals.ToolCallFunction{
			Name:      ToolName,
			Arguments: `{"url":"http://127.0.0.1:8080/private"}`,
		},
	})

	var result ToolResult
	if err := json.Unmarshal([]byte(message.Content), &result); err != nil {
		t.Fatalf("failed to parse tool result: %v", err)
	}
	if result.Status != "error" {
		t.Fatalf("expected error status, got %#v", result)
	}
	if !strings.Contains(result.Error, "local or private network") {
		t.Fatalf("expected local network rejection, got %#v", result)
	}
}

func TestValidateURLRejectsUnsafeTargets(t *testing.T) {
	unsafeTargets := []string{
		"ftp://example.com/file",
		"https://user:pass@example.com/private",
		"http://service.local/private",
		"http://100.64.0.1/private",
		"http://[fe80::1%25lo0]/private",
	}

	for _, target := range unsafeTargets {
		if err := validateURL(target); err == nil {
			t.Fatalf("expected %q to be rejected", target)
		}
	}
}
