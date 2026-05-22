package memory

import (
	"chat/globals"
	"testing"
)

func TestParseToolInput(t *testing.T) {
	input, err := parseToolInput(`{"action":"create","category":"preference","content":"用户喜欢玩原神游戏","reason":"用户明确表示自己爱玩原神，这是一个稳定的游戏偏好"}`)
	if err != nil {
		t.Fatalf("parseToolInput returned error: %v", err)
	}

	if input.Action != "create" {
		t.Fatalf("unexpected action: %q", input.Action)
	}

	if input.Category != "preference" {
		t.Fatalf("unexpected category: %q", input.Category)
	}

	if input.Content != "用户喜欢玩原神游戏" {
		t.Fatalf("unexpected content: %q", input.Content)
	}

	if input.Reason != "用户明确表示自己爱玩原神，这是一个稳定的游戏偏好" {
		t.Fatalf("unexpected reason: %q", input.Reason)
	}
}

func TestWritableToolChannelsIncludeXiaomiTokenPlan(t *testing.T) {
	if _, ok := writableToolChannelTypes[globals.XiaomiTokenPlanCNChannelType]; !ok {
		t.Fatalf("expected xiaomi token plan channel to allow tool definitions")
	}
}

func TestToolCallChannelsIncludeFunctionToolCapableAdapters(t *testing.T) {
	expected := []string{
		globals.OpenAIChannelType,
		globals.OpenAIResponsesChannelType,
		globals.AzureOpenAIChannelType,
		globals.ClaudeChannelType,
		globals.GLMCodingPlanCNChannelType,
		globals.MiniMaxTokenPlanCNChannelType,
		globals.XiaomiTokenPlanCNChannelType,
		globals.PalmChannelType,
		globals.GeminiEnterpriseAgentPlatformChannelType,
		globals.DeepseekChannelType,
		globals.XAIChannelType,
	}

	for _, channelType := range expected {
		if _, ok := toolCallChannelTypes[channelType]; !ok {
			t.Fatalf("expected %s channel to allow function tool calls", channelType)
		}
	}
}
