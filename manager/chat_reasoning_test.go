package manager

import (
	"chat/globals"
	"chat/manager/conversation"
	"testing"
)

func TestBuildThinkingConfigRequestsReasoningSummary(t *testing.T) {
	instance := &conversation.Conversation{}
	instance.SetOpenAIReasoningEffort("medium")
	instance.SetOpenAIReasoningSummary("detailed")

	config, ok := buildThinkingConfig(instance, "gpt-5.4").(map[string]interface{})
	if !ok {
		t.Fatalf("expected reasoning config map, got %#v", config)
	}

	if config["effort"] != "medium" {
		t.Fatalf("expected medium reasoning effort, got %#v", config["effort"])
	}
	if config["summary"] != "detailed" {
		t.Fatalf("expected reasoning summary detailed, got %#v", config["summary"])
	}
}

func TestBuildThinkingConfigDefaultsReasoningSummaryToDetailed(t *testing.T) {
	instance := &conversation.Conversation{}
	instance.SetOpenAIReasoningEffort("medium")

	config, ok := buildThinkingConfig(instance, "gpt-5.4").(map[string]interface{})
	if !ok {
		t.Fatalf("expected reasoning config map, got %#v", config)
	}

	if config["summary"] != "detailed" {
		t.Fatalf("expected default reasoning summary detailed, got %#v", config["summary"])
	}
}

func TestBuildThinkingConfigAllowsDisablingReasoningSummary(t *testing.T) {
	instance := &conversation.Conversation{}
	instance.SetOpenAIReasoningEffort("medium")
	instance.SetOpenAIReasoningSummary("none")

	config, ok := buildThinkingConfig(instance, "gpt-5.4").(map[string]interface{})
	if !ok {
		t.Fatalf("expected reasoning config map, got %#v", config)
	}

	if _, ok := config["summary"]; ok {
		t.Fatalf("expected no summary request when summary is disabled, got %#v", config)
	}
}

func TestBuildThinkingConfigDoesNotRequestSummaryForNone(t *testing.T) {
	instance := &conversation.Conversation{}
	instance.SetOpenAIReasoningEffort("none")
	instance.SetOpenAIReasoningSummary("detailed")

	config, ok := buildThinkingConfig(instance, "gpt-5.4").(map[string]interface{})
	if !ok {
		t.Fatalf("expected reasoning config map, got %#v", config)
	}

	if config["effort"] != "none" {
		t.Fatalf("expected none reasoning effort, got %#v", config["effort"])
	}
	if _, ok := config["summary"]; ok {
		t.Fatalf("expected no summary request when reasoning is disabled, got %#v", config)
	}
}

func TestBuildThinkingConfigEnablesXiaomiTokenPlanThinking(t *testing.T) {
	instance := &conversation.Conversation{}
	instance.SetOpenAIReasoningEffort("high")
	instance.SetOpenAIReasoningSummary("detailed")

	config, ok := buildThinkingConfig(instance, "mimo-v2.5-pro").(map[string]interface{})
	if !ok {
		t.Fatalf("expected xiaomi thinking config map, got %#v", config)
	}

	if config["type"] != "enabled" {
		t.Fatalf("expected xiaomi thinking to be enabled, got %#v", config["type"])
	}
	if _, ok := config["summary"]; ok {
		t.Fatalf("expected no OpenAI reasoning summary for xiaomi thinking, got %#v", config)
	}
}

func TestBuildThinkingConfigDisablesXiaomiTokenPlanThinking(t *testing.T) {
	instance := &conversation.Conversation{}
	instance.SetOpenAIReasoningEffort("none")

	config, ok := buildThinkingConfig(instance, "mimo-v2.5").(map[string]interface{})
	if !ok {
		t.Fatalf("expected xiaomi thinking config map, got %#v", config)
	}

	if config["type"] != "disabled" {
		t.Fatalf("expected xiaomi thinking to be disabled, got %#v", config["type"])
	}
}

func TestBuildDeepseekThinkingConfigRequestsReasoningEffort(t *testing.T) {
	instance := &conversation.Conversation{}
	instance.SetDeepseekThinkingEnabled(true)
	instance.SetDeepseekReasoningEffort("max")

	config, effort := buildDeepseekThinkingConfig(instance, "deepseek-v4-pro")
	payload, ok := config.(map[string]interface{})
	if !ok {
		t.Fatalf("expected deepseek thinking config map, got %#v", config)
	}

	if payload["type"] != "enabled" {
		t.Fatalf("expected deepseek thinking to be enabled, got %#v", payload["type"])
	}
	if effort == nil || *effort != "max" {
		t.Fatalf("expected deepseek reasoning effort max, got %#v", effort)
	}
}

func TestBuildXAIReasoningEffortAppliesMaintainedLevels(t *testing.T) {
	instance := &conversation.Conversation{}
	instance.SetOpenAIReasoningEffort("medium")

	effort := buildXAIReasoningEffort(instance, "grok-4.5")
	if effort == nil || *effort != "medium" {
		t.Fatalf("expected xAI reasoning effort medium, got %#v", effort)
	}
}

func TestBuildXAIReasoningEffortDefaultsToHighestAvailableLevel(t *testing.T) {
	globals.SetCustomReasoningEfforts(map[string][]string{
		"grok-4.5": {"low", "medium"},
	})
	t.Cleanup(func() {
		globals.SetCustomReasoningEfforts(nil)
	})

	instance := &conversation.Conversation{}
	instance.SetOpenAIReasoningEffort("none")

	effort := buildXAIReasoningEffort(instance, "grok-4.5")
	if effort == nil || *effort != "medium" {
		t.Fatalf("expected restricted xAI default medium, got %#v", effort)
	}
}

func TestBuildDeepseekThinkingConfigDisablesThinking(t *testing.T) {
	instance := &conversation.Conversation{}
	instance.SetDeepseekThinkingEnabled(false)
	instance.SetDeepseekReasoningEffort("max")

	config, effort := buildDeepseekThinkingConfig(instance, "deepseek-v4-flash")
	payload, ok := config.(map[string]interface{})
	if !ok {
		t.Fatalf("expected deepseek thinking config map, got %#v", config)
	}

	if payload["type"] != "disabled" {
		t.Fatalf("expected deepseek thinking to be disabled, got %#v", payload["type"])
	}
	if effort != nil {
		t.Fatalf("expected no reasoning effort when deepseek thinking is disabled, got %#v", effort)
	}
}

func TestBuildDeepseekThinkingConfigAppliesAdminRestriction(t *testing.T) {
	globals.SetCustomReasoningEfforts(map[string][]string{
		"deepseek-v4-pro": {"high"},
	})
	t.Cleanup(func() {
		globals.SetCustomReasoningEfforts(nil)
	})

	instance := &conversation.Conversation{}
	instance.SetDeepseekThinkingEnabled(true)
	instance.SetDeepseekReasoningEffort("max")

	_, effort := buildDeepseekThinkingConfig(instance, "deepseek-v4-pro")
	if effort == nil || *effort != "high" {
		t.Fatalf("expected restricted deepseek effort high, got %#v", effort)
	}
}

func TestNormalizeConfiguredGeminiThinkingBudgetAppliesAdminRestriction(t *testing.T) {
	globals.SetCustomReasoningEfforts(map[string][]string{
		"gemini-3.1-pro-preview": {"low"},
	})
	t.Cleanup(func() {
		globals.SetCustomReasoningEfforts(nil)
	})

	if got := normalizeConfiguredGeminiThinkingBudget("gemini-3.1-pro-preview", 8192); got != 1024 {
		t.Fatalf("expected restricted gemini low budget, got %d", got)
	}
}
