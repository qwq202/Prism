package globals

import (
	"reflect"
	"testing"
)

func TestIsOpenAIResponsesNativeWebModel(t *testing.T) {
	if !IsOpenAIResponsesNativeWebModel("gpt-5.3-chat-latest") {
		t.Fatalf("expected gpt-5.3-chat-latest to support native web")
	}

	if !IsOpenAIResponsesNativeWebModel("gpt-5.4-pro") {
		t.Fatalf("expected gpt-5.4-pro to support native web")
	}

	if IsOpenAIResponsesNativeWebModel("o1") {
		t.Fatalf("expected o1 to not support native web")
	}

	if IsOpenAIResponsesNativeWebModel("gpt-4.5-preview") {
		t.Fatalf("expected gpt-4.5-preview to not support native web")
	}
}

func TestNormalizeOpenAIResponsesReasoningEffort(t *testing.T) {
	if got := NormalizeOpenAIResponsesReasoningEffort("gpt-5.2", "xhigh", false); got != "xhigh" {
		t.Fatalf("expected xhigh for gpt-5.2, got %q", got)
	}

	if got := NormalizeOpenAIResponsesReasoningEffort("gpt-5.4-pro", "medium", false); got != "medium" {
		t.Fatalf("expected medium for gpt-5.4-pro, got %q", got)
	}

	if got := NormalizeOpenAIResponsesReasoningEffort("gpt-5-pro", "low", false); got != "" {
		t.Fatalf("expected low to be unsupported for gpt-5-pro, got %q", got)
	}

	if got := NormalizeOpenAIResponsesReasoningEffort("gpt-5.2-chat-latest", "medium", false); got != "" {
		t.Fatalf("expected gpt-5.2-chat-latest to not expose reasoning control, got %q", got)
	}

	if got := NormalizeOpenAIResponsesReasoningEffort("gpt-5", "minimal", true); got != "low" {
		t.Fatalf("expected minimal to downgrade to low when native web is enabled, got %q", got)
	}

	if got := NormalizeOpenAIResponsesReasoningEffort("o1", "none", false); got != "" {
		t.Fatalf("expected none to be unsupported for o1, got %q", got)
	}
}

func TestNormalizeOpenAIResponsesReasoningSummary(t *testing.T) {
	if got := NormalizeOpenAIResponsesReasoningSummary(""); got != "auto" {
		t.Fatalf("expected empty summary to default to auto, got %q", got)
	}

	if got := NormalizeOpenAIResponsesReasoningSummary(" DETAILED "); got != "detailed" {
		t.Fatalf("expected detailed summary, got %q", got)
	}

	if got := NormalizeOpenAIResponsesReasoningSummary("none"); got != "none" {
		t.Fatalf("expected none summary, got %q", got)
	}

	if got := NormalizeOpenAIResponsesReasoningSummary("verbose"); got != "auto" {
		t.Fatalf("expected invalid summary to default to auto, got %q", got)
	}
}

func TestCapabilitiesForOpenAIResponsesModels(t *testing.T) {
	tests := []struct {
		name                string
		model               string
		nativeWebSearch     bool
		reasoningEfforts    []string
		samplingRestriction SamplingRestriction
	}{
		{
			name:                "gpt 5.4 reasoning model",
			model:               "gpt-5.4",
			nativeWebSearch:     true,
			reasoningEfforts:    []string{"none", "low", "medium", "high", "xhigh"},
			samplingRestriction: SamplingRestrictionWithReasoning,
		},
		{
			name:                "gpt 5 base model",
			model:               "gpt-5",
			nativeWebSearch:     true,
			reasoningEfforts:    []string{"minimal", "low", "medium", "high"},
			samplingRestriction: SamplingRestrictionAlways,
		},
		{
			name:                "gpt 5.2 pro model",
			model:               "gpt-5.2-pro",
			nativeWebSearch:     true,
			reasoningEfforts:    []string{"medium", "high", "xhigh"},
			samplingRestriction: SamplingRestrictionAlways,
		},
		{
			name:                "o3 model",
			model:               "o3",
			nativeWebSearch:     true,
			reasoningEfforts:    []string{"low", "medium", "high"},
			samplingRestriction: SamplingRestrictionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities := CapabilitiesFor(OpenAIResponsesChannelType, tt.model)
			if capabilities.NativeWebSearch != tt.nativeWebSearch {
				t.Fatalf("expected native web %v, got %v", tt.nativeWebSearch, capabilities.NativeWebSearch)
			}
			if !reflect.DeepEqual(capabilities.ReasoningEfforts, tt.reasoningEfforts) {
				t.Fatalf("unexpected reasoning efforts: got %#v want %#v", capabilities.ReasoningEfforts, tt.reasoningEfforts)
			}
			if capabilities.ReasoningControl != (len(tt.reasoningEfforts) > 0) {
				t.Fatalf("unexpected reasoning control flag: %v", capabilities.ReasoningControl)
			}
			if capabilities.SamplingRestriction != tt.samplingRestriction {
				t.Fatalf("expected sampling restriction %q, got %q", tt.samplingRestriction, capabilities.SamplingRestriction)
			}
		})
	}
}

func TestCapabilitiesForXAIModels(t *testing.T) {
	capabilities := CapabilitiesFor(XAIChannelType, "grok-4-1-fast-reasoning")
	if !capabilities.NativeWebSearch {
		t.Fatalf("expected grok to support native web search")
	}
	if !capabilities.XSearch {
		t.Fatalf("expected grok to support x search")
	}
	if !capabilities.Search {
		t.Fatalf("expected grok to expose aggregate search capability")
	}
	if capabilities.ReasoningControl {
		t.Fatalf("expected grok to not expose OpenAI-style reasoning control")
	}
}

func TestReasoningEffortNormalizationUsesCapabilities(t *testing.T) {
	capabilities := CapabilitiesFor(OpenAIResponsesChannelType, "gpt-5.2-pro")
	if got := NormalizeReasoningEffort(capabilities, "XHIGH"); got != "xhigh" {
		t.Fatalf("expected normalized xhigh, got %q", got)
	}
	if got := NormalizeReasoningEffort(capabilities, "low"); got != "" {
		t.Fatalf("expected unsupported effort to normalize to empty, got %q", got)
	}
}

func TestSamplingRestrictionUsesCapabilities(t *testing.T) {
	conditional := CapabilitiesFor(OpenAIResponsesChannelType, "gpt-5.4")
	if ShouldRestrictSampling(conditional, "none") {
		t.Fatalf("expected sampling to be allowed without reasoning")
	}
	if !ShouldRestrictSampling(conditional, "high") {
		t.Fatalf("expected sampling to be restricted with reasoning")
	}

	always := CapabilitiesFor(OpenAIResponsesChannelType, "gpt-5")
	if !ShouldRestrictSampling(always, "") {
		t.Fatalf("expected sampling to always be restricted")
	}
}
