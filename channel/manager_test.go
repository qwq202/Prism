package channel

import (
	"chat/globals"
	"errors"
	"strings"
	"testing"
)

func TestValidateChannelTypeAllowsKnownTypes(t *testing.T) {
	knownTypes := []string{
		globals.OpenAIChannelType,
		globals.OpenRouterChannelType,
		globals.OpenAIResponsesChannelType,
		globals.XAIChannelType,
		globals.AzureOpenAIChannelType,
		globals.ClaudeChannelType,
		globals.GLMCodingPlanCNChannelType,
		globals.MiniMaxTokenPlanCNChannelType,
		globals.XiaomiMiMoChannelType,
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

func TestCreateChannelKeepsSequenceWhenSaveFails(t *testing.T) {
	previousSave := saveChannelConfig
	saveChannelConfig = func(Sequence) error {
		return errors.New("simulated save failure")
	}
	t.Cleanup(func() {
		saveChannelConfig = previousSave
	})

	manager := &Manager{
		Sequence: Sequence{
			{Id: 1, Type: globals.OpenAIChannelType, State: true},
		},
	}

	err := manager.CreateChannel(&Channel{Type: globals.OpenAIChannelType})
	if err == nil || err.Error() != "simulated save failure" {
		t.Fatalf("expected simulated save failure, got %v", err)
	}
	if len(manager.Sequence) != 1 {
		t.Fatalf("expected sequence to remain unchanged, got %#v", manager.Sequence)
	}
}

func TestActivateChannelKeepsStateWhenSaveFails(t *testing.T) {
	previousSave := saveChannelConfig
	saveChannelConfig = func(Sequence) error {
		return errors.New("simulated save failure")
	}
	t.Cleanup(func() {
		saveChannelConfig = previousSave
	})

	manager := &Manager{
		Sequence: Sequence{
			{Id: 1, Type: globals.OpenAIChannelType, State: false},
		},
	}

	err := manager.ActivateChannel(1)
	if err == nil || err.Error() != "simulated save failure" {
		t.Fatalf("expected simulated save failure, got %v", err)
	}
	if manager.Sequence[0].State {
		t.Fatalf("expected channel state to remain false")
	}
}
