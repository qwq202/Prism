package admin

import (
	"chat/globals"
	"errors"
	"reflect"
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
			{
				Id:               "old-model",
				Name:             "Old",
				ReasoningModel:   true,
				ReasoningEfforts: []string{"high"},
			},
		},
	}
	globals.SetCustomReasoningEfforts(map[string][]string{"old-model": {"high"}})
	t.Cleanup(func() {
		globals.SetCustomReasoningEfforts(nil)
	})

	err := market.SetModels(MarketModelList{{
		Id:               "new-model",
		Name:             "New",
		ReasoningModel:   true,
		ReasoningEfforts: []string{"low"},
	}})
	if err == nil || err.Error() != "simulated save failure" {
		t.Fatalf("expected simulated save failure, got %v", err)
	}
	if len(market.Models) != 1 || market.Models[0].Id != "old-model" {
		t.Fatalf("expected market models to remain unchanged, got %#v", market.Models)
	}
	if globals.CapabilitiesFor(globals.OpenAIResponsesChannelType, "new-model").ReasoningControl {
		t.Fatalf("expected failed save not to publish new reasoning capabilities")
	}
	if !globals.CapabilitiesFor(globals.OpenAIResponsesChannelType, "old-model").ReasoningControl {
		t.Fatalf("expected failed save to retain old reasoning capabilities")
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

func TestNormalizeMarketModelsDefaultsCustomReasoningEfforts(t *testing.T) {
	models := normalizeMarketModels(MarketModelList{{
		Id:             "future-reasoner",
		Name:           "Future Reasoner",
		ReasoningModel: true,
	}})

	if len(models) != 1 {
		t.Fatalf("expected one normalized model, got %d", len(models))
	}
	if !reflect.DeepEqual(models[0].ReasoningEfforts, defaultCustomReasoningEfforts) {
		t.Fatalf("unexpected default efforts: got %#v want %#v", models[0].ReasoningEfforts, defaultCustomReasoningEfforts)
	}
}

func TestNormalizeMarketModelsKeepsNoneOnlyReasoningConfiguration(t *testing.T) {
	models := normalizeMarketModels(MarketModelList{{
		Id:               "future-reasoner",
		Name:             "Future Reasoner",
		ReasoningModel:   true,
		ReasoningEfforts: []string{"none"},
	}})

	if !reflect.DeepEqual(models[0].ReasoningEfforts, []string{"none"}) {
		t.Fatalf("expected none-only configuration to be retained, got %#v", models[0].ReasoningEfforts)
	}
}

func TestGetViewModelsExposesCustomReasoningConfiguration(t *testing.T) {
	models := normalizeMarketModels(MarketModelList{{
		Id:               "future-reasoner",
		Name:             "Future Reasoner",
		ReasoningModel:   true,
		ReasoningEfforts: []string{"max", "low"},
		Tag:              ModelTag{"official"},
	}})
	market := &Market{Models: models}

	views := market.GetViewModels()
	if len(views) != 1 {
		t.Fatalf("expected one market view, got %d", len(views))
	}
	view := views[0]
	if !view.ReasoningConfigurable || !view.ReasoningModel {
		t.Fatalf("expected custom reasoning configuration in view, got %#v", view)
	}
	if !reflect.DeepEqual(view.ReasoningEfforts, []string{"low", "max"}) {
		t.Fatalf("unexpected reasoning efforts: %#v", view.ReasoningEfforts)
	}
	if !hasMarketTag(view.Tag, reasoningTag) {
		t.Fatalf("expected reasoning tag in view, got %#v", view.Tag)
	}
	if hasMarketTag(market.Models[0].Tag, reasoningTag) {
		t.Fatalf("expected view normalization not to mutate stored market tags")
	}
}

func TestNormalizeMarketModelsDefaultsMaintainedReasoningEfforts(t *testing.T) {
	models := normalizeMarketModels(MarketModelList{{
		Id:   "gpt-5.6",
		Name: "GPT 5.6",
	}})

	want := globals.ManagedReasoningEfforts("gpt-5.6")
	if !reflect.DeepEqual(models[0].ReasoningEfforts, want) {
		t.Fatalf("unexpected maintained defaults: got %#v want %#v", models[0].ReasoningEfforts, want)
	}
	if models[0].ReasoningModel {
		t.Fatalf("expected maintained model not to persist the custom reasoning switch")
	}
}

func TestGetViewModelsExposesMaintainedReasoningRestriction(t *testing.T) {
	market := &Market{Models: normalizeMarketModels(MarketModelList{{
		Id:               "gpt-5.6",
		Name:             "GPT 5.6",
		ReasoningModel:   true,
		ReasoningEfforts: []string{"low", "high", "invalid"},
	}})}

	views := market.GetViewModels()
	if len(views) != 1 {
		t.Fatalf("expected one market view, got %d", len(views))
	}
	view := views[0]
	if view.ReasoningConfigurable || !view.ReasoningModel {
		t.Fatalf("expected maintained model to expose levels without the custom switch, got %#v", view)
	}
	if !reflect.DeepEqual(view.ReasoningEfforts, []string{"low", "high"}) {
		t.Fatalf("unexpected maintained selected efforts: %#v", view.ReasoningEfforts)
	}
	if !reflect.DeepEqual(view.ReasoningAvailable, globals.ManagedReasoningEfforts("gpt-5.6")) {
		t.Fatalf("unexpected maintained available efforts: %#v", view.ReasoningAvailable)
	}
}

func TestGetViewModelsTreatsGrok45AsMaintainedReasoningModel(t *testing.T) {
	market := &Market{Models: normalizeMarketModels(MarketModelList{{
		Id:             "grok-4.5",
		Name:           "Grok 4.5",
		ReasoningModel: true,
	}})}

	view := market.GetViewModels()[0]
	if view.ReasoningConfigurable || !view.ReasoningModel {
		t.Fatalf("expected Grok 4.5 to hide the custom reasoning switch, got %#v", view)
	}
	if !reflect.DeepEqual(view.ReasoningAvailable, []string{"low", "medium", "high"}) {
		t.Fatalf("unexpected Grok 4.5 reasoning levels: %#v", view.ReasoningAvailable)
	}
	if !view.VisionModel {
		t.Fatalf("expected Grok 4.5 multimodal input capability to be maintained")
	}
}
