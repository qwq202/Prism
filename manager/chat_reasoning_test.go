package manager

import (
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

func TestBuildThinkingConfigDefaultsReasoningSummaryToAuto(t *testing.T) {
	instance := &conversation.Conversation{}
	instance.SetOpenAIReasoningEffort("medium")

	config, ok := buildThinkingConfig(instance, "gpt-5.4").(map[string]interface{})
	if !ok {
		t.Fatalf("expected reasoning config map, got %#v", config)
	}

	if config["summary"] != "auto" {
		t.Fatalf("expected default reasoning summary auto, got %#v", config["summary"])
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
