package globals

import (
	"reflect"
	"testing"
)

func TestIsOpenAIResponsesNativeWebModel(t *testing.T) {
	if !IsOpenAIResponsesNativeWebModel("gpt-5.6-terra") {
		t.Fatalf("expected gpt-5.6-terra to support native web")
	}

	if !IsOpenAIResponsesNativeWebModel("gpt-5.5") {
		t.Fatalf("expected gpt-5.5 to support native web")
	}

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

func TestIsVisionModelUsesConfiguredResolver(t *testing.T) {
	originalResolver := VisionModelResolver
	VisionModelResolver = func(model string) bool {
		return model == "custom-vision-model"
	}
	defer func() {
		VisionModelResolver = originalResolver
	}()

	if !IsVisionModel("custom-vision-model") {
		t.Fatalf("expected configured vision model to be recognized")
	}
}

func TestIsDrawingModelUsesInternalProviderAwareList(t *testing.T) {
	tests := []struct {
		name        string
		channelType string
		model       string
		want        bool
	}{
		{
			name:        "openai gpt image",
			channelType: OpenAIChannelType,
			model:       GPTImage1,
			want:        true,
		},
		{
			name:        "azure dalle",
			channelType: AzureOpenAIChannelType,
			model:       Dalle3,
			want:        true,
		},
		{
			name:        "gemini flash lite image generation",
			channelType: PalmChannelType,
			model:       Gemini31FlashLiteImage,
			want:        true,
		},
		{
			name:        "gemini image generation",
			channelType: PalmChannelType,
			model:       Gemini31FlashImage,
			want:        true,
		},
		{
			name:        "vertex express gemini image generation",
			channelType: GeminiEnterpriseAgentPlatformChannelType,
			model:       Gemini3ProImage,
			want:        true,
		},
		{
			name:        "gemini chat model is not drawing",
			channelType: PalmChannelType,
			model:       Gemini25Flash,
			want:        false,
		},
		{
			name:        "openrouter is not treated as drawing by model name alone",
			channelType: OpenRouterChannelType,
			model:       GPTImage1,
			want:        false,
		},
		{
			name:        "empty channel falls back to model list",
			channelType: "",
			model:       Gemini25FlashImage,
			want:        true,
		},
		{
			name:        "legacy gpt 4 dalle remains excluded",
			channelType: OpenAIChannelType,
			model:       GPT4Dalle,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDrawingModel(tt.channelType, tt.model); got != tt.want {
				t.Fatalf("expected drawing=%v, got %v", tt.want, got)
			}
		})
	}
}

func TestNormalizeOpenAIResponsesReasoningEffort(t *testing.T) {
	if got := NormalizeOpenAIResponsesReasoningEffort("gpt-5.6", "max", false); got != "max" {
		t.Fatalf("expected max for gpt-5.6, got %q", got)
	}

	if got := NormalizeOpenAIResponsesReasoningEffort("gpt-5.2", "xhigh", false); got != "xhigh" {
		t.Fatalf("expected xhigh for gpt-5.2, got %q", got)
	}

	if got := NormalizeOpenAIResponsesReasoningEffort("gpt-5.4-pro", "medium", false); got != "medium" {
		t.Fatalf("expected medium for gpt-5.4-pro, got %q", got)
	}

	if got := NormalizeOpenAIResponsesReasoningEffort("gpt-5.4-mini", "xhigh", false); got != "xhigh" {
		t.Fatalf("expected xhigh for gpt-5.4-mini, got %q", got)
	}

	if got := NormalizeOpenAIResponsesReasoningEffort("gpt-5.5", "xhigh", false); got != "xhigh" {
		t.Fatalf("expected xhigh for gpt-5.5, got %q", got)
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
	if got := NormalizeOpenAIResponsesReasoningSummary(""); got != "detailed" {
		t.Fatalf("expected empty summary to default to detailed, got %q", got)
	}

	if got := NormalizeOpenAIResponsesReasoningSummary(" DETAILED "); got != "detailed" {
		t.Fatalf("expected detailed summary, got %q", got)
	}

	if got := NormalizeOpenAIResponsesReasoningSummary("none"); got != "none" {
		t.Fatalf("expected none summary, got %q", got)
	}

	if got := NormalizeOpenAIResponsesReasoningSummary("verbose"); got != "detailed" {
		t.Fatalf("expected invalid summary to default to detailed, got %q", got)
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
			name:                "gpt 5.6 reasoning model",
			model:               "gpt-5.6-terra",
			nativeWebSearch:     true,
			reasoningEfforts:    []string{"none", "low", "medium", "high", "xhigh", "max"},
			samplingRestriction: SamplingRestrictionAlways,
		},
		{
			name:                "gpt 5.5 reasoning model",
			model:               "gpt-5.5",
			nativeWebSearch:     true,
			reasoningEfforts:    []string{"none", "low", "medium", "high", "xhigh"},
			samplingRestriction: SamplingRestrictionAlways,
		},
		{
			name:                "gpt 5.4 reasoning model",
			model:               "gpt-5.4",
			nativeWebSearch:     true,
			reasoningEfforts:    []string{"none", "low", "medium", "high", "xhigh"},
			samplingRestriction: SamplingRestrictionAlways,
		},
		{
			name:                "gpt 5.4 mini reasoning model",
			model:               "gpt-5.4-mini",
			nativeWebSearch:     true,
			reasoningEfforts:    []string{"none", "low", "medium", "high", "xhigh"},
			samplingRestriction: SamplingRestrictionAlways,
		},
		{
			name:                "gpt 5.2 reasoning model",
			model:               "gpt-5.2",
			nativeWebSearch:     true,
			reasoningEfforts:    []string{"none", "low", "medium", "high", "xhigh"},
			samplingRestriction: SamplingRestrictionAlways,
		},
		{
			name:                "gpt 5.1 reasoning model",
			model:               "gpt-5.1",
			nativeWebSearch:     true,
			reasoningEfforts:    []string{"none", "low", "medium", "high"},
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
			samplingRestriction: SamplingRestrictionAlways,
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
	capabilities := CapabilitiesFor(XAIChannelType, "grok-4.5")
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

func TestCapabilitiesForXiaomiTokenPlanModels(t *testing.T) {
	capabilities := CapabilitiesFor(XiaomiTokenPlanCNChannelType, "mimo-v2.5-pro")
	if !reflect.DeepEqual(capabilities.ReasoningEfforts, []string{"none", "high"}) {
		t.Fatalf("unexpected xiaomi reasoning efforts: %#v", capabilities.ReasoningEfforts)
	}
	if !capabilities.ReasoningControl {
		t.Fatalf("expected xiaomi mimo to expose thinking control")
	}
	if capabilities.SamplingRestriction != SamplingRestrictionWithReasoning {
		t.Fatalf("expected xiaomi sampling restriction with reasoning, got %q", capabilities.SamplingRestriction)
	}

	prefixed := CapabilitiesFor(XiaomiTokenPlanCNChannelType, "xiaomi/mimo-v2.5")
	if !prefixed.ReasoningControl {
		t.Fatalf("expected prefixed xiaomi mimo id to expose thinking control")
	}
}

func TestCapabilitiesForXiaomiMiMoModels(t *testing.T) {
	capabilities := CapabilitiesFor(XiaomiMiMoChannelType, "mimo-v2.5-pro")
	if !reflect.DeepEqual(capabilities.ReasoningEfforts, []string{"none", "high"}) {
		t.Fatalf("unexpected xiaomi mimo reasoning efforts: %#v", capabilities.ReasoningEfforts)
	}
	if !capabilities.ReasoningControl {
		t.Fatalf("expected xiaomi mimo to expose thinking control")
	}
	if capabilities.SamplingRestriction != SamplingRestrictionWithReasoning {
		t.Fatalf("expected xiaomi mimo sampling restriction with reasoning, got %q", capabilities.SamplingRestriction)
	}
}

func TestCapabilitiesForCustomReasoningModel(t *testing.T) {
	SetCustomReasoningEfforts(map[string][]string{
		"future-reasoner": {"MAX", "low", "low", "invalid", "none"},
	})
	t.Cleanup(func() {
		SetCustomReasoningEfforts(nil)
	})

	capabilities := CapabilitiesFor(OpenAIResponsesChannelType, "future-reasoner")
	want := []string{"none", "low", "max"}
	if !reflect.DeepEqual(capabilities.ReasoningEfforts, want) {
		t.Fatalf("unexpected custom reasoning efforts: got %#v want %#v", capabilities.ReasoningEfforts, want)
	}
	if !capabilities.ReasoningControl {
		t.Fatalf("expected custom reasoning model to expose reasoning control")
	}
	if capabilities.SamplingRestriction != SamplingRestrictionWithReasoning {
		t.Fatalf("expected custom reasoning to restrict sampling only while enabled, got %q", capabilities.SamplingRestriction)
	}
}

func TestConfiguredReasoningCanRestrictMaintainedModel(t *testing.T) {
	SetCustomReasoningEfforts(map[string][]string{
		"gpt-5.6": {"low", "high"},
	})
	t.Cleanup(func() {
		SetCustomReasoningEfforts(nil)
	})

	capabilities := CapabilitiesFor(OpenAIResponsesChannelType, "gpt-5.6")
	want := []string{"low", "high"}
	if !reflect.DeepEqual(capabilities.ReasoningEfforts, want) {
		t.Fatalf("expected configured maintained restriction: got %#v want %#v", capabilities.ReasoningEfforts, want)
	}
}

func TestConfiguredReasoningCannotExpandMaintainedModel(t *testing.T) {
	SetCustomReasoningEfforts(map[string][]string{
		"gpt-5.6": {"minimal"},
	})
	t.Cleanup(func() {
		SetCustomReasoningEfforts(nil)
	})

	capabilities := CapabilitiesFor(OpenAIResponsesChannelType, "gpt-5.6")
	want := []string{"none", "low", "medium", "high", "xhigh", "max"}
	if !reflect.DeepEqual(capabilities.ReasoningEfforts, want) {
		t.Fatalf("expected unsupported configured level to be ignored: got %#v want %#v", capabilities.ReasoningEfforts, want)
	}
}

func TestHasManagedReasoningCapabilities(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{model: "gpt-5.6-terra", want: true},
		{model: "gemini-3.1-pro-preview", want: true},
		{model: "deepseek-v4-pro", want: true},
		{model: "xiaomi/mimo-v2.5-pro", want: true},
		{model: "future-reasoner", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := HasManagedReasoningCapabilities("", tt.model); got != tt.want {
				t.Fatalf("expected managed=%v for %q, got %v", tt.want, tt.model, got)
			}
		})
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
	conditional := CapabilitiesFor(OpenAIResponsesChannelType, "gpt-5.1")
	if ShouldRestrictSampling(conditional, "") {
		t.Fatalf("expected sampling to be allowed for default gpt-5.1 none reasoning")
	}
	if ShouldRestrictSampling(conditional, "none") {
		t.Fatalf("expected sampling to be allowed with gpt-5.1 none reasoning")
	}
	if !ShouldRestrictSampling(conditional, "high") {
		t.Fatalf("expected sampling to be restricted with reasoning")
	}

	always := CapabilitiesFor(OpenAIResponsesChannelType, "gpt-5")
	if !ShouldRestrictSampling(always, "") {
		t.Fatalf("expected sampling to always be restricted")
	}
}
