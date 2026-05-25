package channel

import (
	"chat/globals"
	"errors"
	"testing"
)

func withFailingChargeSave(t *testing.T) {
	t.Helper()

	previousSave := saveChargeConfig
	saveChargeConfig = func(ChargeSequence) error {
		return errors.New("simulated save failure")
	}
	t.Cleanup(func() {
		saveChargeConfig = previousSave
	})
}

func TestSetRuleKeepsSequenceWhenSaveFails(t *testing.T) {
	withFailingChargeSave(t)

	manager := &ChargeManager{
		Sequence: ChargeSequence{
			{Id: 1, Type: globals.TokenBilling, Models: []string{"gpt-old"}, Input: 1, Output: 2},
		},
	}

	err := manager.SetRule(Charge{Id: 1, Type: globals.NonBilling, Models: []string{"gpt-new"}})
	if err == nil || err.Error() != "simulated save failure" {
		t.Fatalf("expected simulated save failure, got %v", err)
	}
	if manager.Sequence[0].Type != globals.TokenBilling {
		t.Fatalf("expected charge type to remain unchanged, got %q", manager.Sequence[0].Type)
	}
	if got := manager.Sequence[0].Models; len(got) != 1 || got[0] != "gpt-old" {
		t.Fatalf("expected charge models to remain unchanged, got %#v", got)
	}
}

func TestSyncRulesKeepsNestedSequenceWhenSaveFails(t *testing.T) {
	withFailingChargeSave(t)

	manager := &ChargeManager{
		Sequence: ChargeSequence{
			{Id: 1, Type: globals.TokenBilling, Models: []string{"gpt-a", "gpt-b"}, Input: 1, Output: 2},
		},
	}

	err := manager.SyncRules(ChargeSequence{
		{Type: globals.NonBilling, Models: []string{"gpt-a"}},
	}, true)
	if err == nil || err.Error() != "simulated save failure" {
		t.Fatalf("expected simulated save failure, got %v", err)
	}
	if got := manager.Sequence[0].Models; len(got) != 2 || got[0] != "gpt-a" || got[1] != "gpt-b" {
		t.Fatalf("expected nested charge models to remain unchanged, got %#v", got)
	}
}
