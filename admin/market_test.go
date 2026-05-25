package admin

import (
	"errors"
	"testing"
)

func TestSetModelsKeepsRuntimeStateWhenSaveFails(t *testing.T) {
	previousSave := saveMarketConfig
	saveMarketConfig = func(MarketModelList) error {
		return errors.New("simulated save failure")
	}
	t.Cleanup(func() {
		saveMarketConfig = previousSave
	})

	market := &Market{
		Models: MarketModelList{
			{Id: "old-model", Name: "Old"},
		},
	}

	err := market.SetModels(MarketModelList{{Id: "new-model", Name: "New"}})
	if err == nil || err.Error() != "simulated save failure" {
		t.Fatalf("expected simulated save failure, got %v", err)
	}
	if len(market.Models) != 1 || market.Models[0].Id != "old-model" {
		t.Fatalf("expected market models to remain unchanged, got %#v", market.Models)
	}
}
