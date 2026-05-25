package channel

import (
	"chat/globals"
	"strings"
	"testing"
)

func TestValidateChannelTypeAllowsKnownTypes(t *testing.T) {
	knownTypes := []string{
		globals.OpenAIChannelType,
		globals.OpenAIResponsesChannelType,
		globals.XAIChannelType,
		globals.AzureOpenAIChannelType,
		globals.ClaudeChannelType,
		globals.GLMCodingPlanCNChannelType,
		globals.MiniMaxTokenPlanCNChannelType,
		globals.XiaomiTokenPlanCNChannelType,
		globals.PalmChannelType,
		globals.GeminiEnterpriseAgentPlatformChannelType,
		globals.DeepseekChannelType,
	}

	for _, channelType := range knownTypes {
		if err := validateChannelType(&Channel{Type: channelType}); err != nil {
			t.Fatalf("expected channel type %q to be valid: %v", channelType, err)
		}
	}
}

func TestValidateChannelTypeRejectsUnknownTypes(t *testing.T) {
	err := validateChannelType(&Channel{Type: "unknown-provider"})
	if err == nil || !strings.Contains(err.Error(), "unknown channel type") {
		t.Fatalf("expected unknown channel type rejection, got %v", err)
	}
}

func TestValidateChannelTypeRejectsRemovedTypes(t *testing.T) {
	err := validateChannelType(&Channel{Type: "midjourney"})
	if err == nil || !strings.Contains(err.Error(), "has been removed") {
		t.Fatalf("expected removed channel type rejection, got %v", err)
	}
}
