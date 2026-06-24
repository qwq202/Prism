package admin

import (
	"errors"
	"testing"
)

func hasMarketTag(tags ModelTag, target string) bool {
	for _, tag := range tags {
		if tag == target {
			return true
		}
	}
	return false
}

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

func TestGetViewModelsAddsDrawingTag(t *testing.T) {
	market := &Market{
		Models: MarketModelList{
			{Id: "gemini-3-pro-image", Name: "Gemini 3 Pro Image", Tag: ModelTag{"official"}},
		},
	}

	views := market.GetViewModels()
	if len(views) != 1 {
		t.Fatalf("expected one market view, got %d", len(views))
	}
	if !views[0].DrawingModel {
		t.Fatalf("expected drawing model to be detected")
	}
	if !hasMarketTag(views[0].Tag, imageGenerationTag) {
		t.Fatalf("expected drawing model view to include %q tag, got %#v", imageGenerationTag, views[0].Tag)
	}
	if hasMarketTag(market.Models[0].Tag, imageGenerationTag) {
		t.Fatalf("expected view normalization not to mutate stored market tags")
	}
}
