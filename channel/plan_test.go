package channel

import "testing"

func TestValidatePlanConfigModels(t *testing.T) {
	originalConduit := ConduitInstance
	ConduitInstance = &Manager{Models: []string{"deepseek-v4-flash", "grok-4-1-fast-reasoning"}}
	defer func() {
		ConduitInstance = originalConduit
	}()

	valid := &PlanManager{
		Plans: []Plan{
			{
				Level: 1,
				Items: []PlanItem{
					{
						Id:     "valid-item",
						Models: []string{"deepseek-v4-flash"},
					},
				},
			},
		},
	}

	if err := validatePlanConfigModels(valid); err != nil {
		t.Fatalf("expected valid plan config, got error: %v", err)
	}

	invalid := &PlanManager{
		Plans: []Plan{
			{
				Level: 1,
				Items: []PlanItem{
					{
						Id:     "invalid-item",
						Models: []string{"deepseek-v4-flash", "gpt-4o"},
					},
				},
			},
		},
	}

	if err := validatePlanConfigModels(invalid); err == nil {
		t.Fatal("expected invalid plan config to be rejected")
	}
}
