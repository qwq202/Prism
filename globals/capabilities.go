package globals

import "strings"

type SamplingRestriction string

const (
	SamplingRestrictionNone          SamplingRestriction = ""
	SamplingRestrictionAlways        SamplingRestriction = "always"
	SamplingRestrictionWithReasoning SamplingRestriction = "with_reasoning"
)

type ModelCapabilities struct {
	NativeWebSearch     bool
	XSearch             bool
	ReasoningControl    bool
	ReasoningEfforts    []string
	SamplingRestriction SamplingRestriction
	Vision              bool
	Video               bool
	Search              bool
}

type ProviderCapabilities struct {
	ChannelType     string
	NativeWebSearch bool
	XSearch         bool
	Reasoning       bool
	Vision          bool
	Video           bool
	Search          bool
}

func CapabilitiesFor(channelType string, model string) ModelCapabilities {
	normalizedChannel := strings.TrimSpace(strings.ToLower(channelType))
	normalizedModel := normalizeModelName(model)

	capabilities := ModelCapabilities{
		Vision: IsVisionModel(model),
		Video:  IsVideoModel(model),
	}

	switch normalizedChannel {
	case OpenAIResponsesChannelType:
		applyOpenAIResponsesCapabilities(&capabilities, normalizedModel)
	case XAIChannelType:
		applyXAICapabilities(&capabilities, normalizedModel)
	case XiaomiMiMoChannelType, XiaomiTokenPlanCNChannelType:
		applyXiaomiTokenPlanCapabilities(&capabilities, normalizedModel)
	}

	capabilities.Search = capabilities.NativeWebSearch || capabilities.XSearch
	capabilities.ReasoningControl = len(capabilities.ReasoningEfforts) > 0

	return capabilities
}

func ProviderCapabilitiesFor(channelType string) ProviderCapabilities {
	normalizedChannel := strings.TrimSpace(strings.ToLower(channelType))
	switch normalizedChannel {
	case OpenAIResponsesChannelType:
		return ProviderCapabilities{
			ChannelType:     OpenAIResponsesChannelType,
			NativeWebSearch: true,
			Reasoning:       true,
			Vision:          true,
			Video:           true,
			Search:          true,
		}
	case XAIChannelType:
		return ProviderCapabilities{
			ChannelType:     XAIChannelType,
			NativeWebSearch: true,
			XSearch:         true,
			Vision:          true,
			Search:          true,
		}
	case XiaomiMiMoChannelType, XiaomiTokenPlanCNChannelType:
		return ProviderCapabilities{
			ChannelType: normalizedChannel,
			Reasoning:   true,
			Vision:      true,
		}
	default:
		return ProviderCapabilities{ChannelType: normalizedChannel}
	}
}

func NormalizeReasoningEffort(capabilities ModelCapabilities, effort string) string {
	normalized := strings.TrimSpace(strings.ToLower(effort))
	if normalized == "" {
		return ""
	}

	for _, item := range capabilities.ReasoningEfforts {
		if item == normalized {
			return normalized
		}
	}

	return ""
}

func ShouldRestrictSampling(capabilities ModelCapabilities, reasoningEffort string) bool {
	switch capabilities.SamplingRestriction {
	case SamplingRestrictionAlways:
		return true
	case SamplingRestrictionWithReasoning:
		effort := strings.TrimSpace(strings.ToLower(reasoningEffort))
		return effort != "" && effort != "none"
	default:
		return false
	}
}

func normalizeModelName(model string) string {
	return strings.TrimSpace(strings.ToLower(model))
}

func applyOpenAIResponsesCapabilities(capabilities *ModelCapabilities, model string) {
	if model == "" {
		return
	}

	capabilities.NativeWebSearch = isOpenAIResponsesNativeWebModel(model)
	capabilities.ReasoningEfforts = openAIResponsesReasoningEfforts(model)
	capabilities.SamplingRestriction = openAIResponsesSamplingRestriction(model)
}

func applyXAICapabilities(capabilities *ModelCapabilities, model string) {
	if model == "" || !strings.HasPrefix(model, "grok") {
		return
	}

	capabilities.NativeWebSearch = true
	capabilities.XSearch = true
}

func applyXiaomiTokenPlanCapabilities(capabilities *ModelCapabilities, model string) {
	if !isXiaomiMiMoModel(model) {
		return
	}

	capabilities.ReasoningEfforts = []string{"none", "high"}
	capabilities.SamplingRestriction = SamplingRestrictionWithReasoning
}

func isXiaomiMiMoModel(model string) bool {
	model = strings.TrimPrefix(normalizeModelName(model), "xiaomi/")
	return strings.HasPrefix(model, "mimo-v2") && !strings.Contains(model, "tts")
}

func isOpenAIResponsesNativeWebModel(model string) bool {
	switch {
	case model == "gpt-5.5" || strings.HasPrefix(model, "gpt-5.5-"):
		return true
	case strings.HasPrefix(model, "gpt-5.4"):
		return true
	case model == "gpt-5.3-chat-latest":
		return true
	case strings.HasPrefix(model, "gpt-5.2"):
		return true
	case strings.HasPrefix(model, "gpt-5.1"):
		return true
	case model == "gpt-5" || strings.HasPrefix(model, "gpt-5-"):
		return true
	case model == "o3" || strings.HasPrefix(model, "o3-"):
		return true
	default:
		return false
	}
}

func openAIResponsesReasoningEfforts(model string) []string {
	var efforts []string
	switch {
	case model == "gpt-5.5" || strings.HasPrefix(model, "gpt-5.5-"):
		efforts = []string{"none", "low", "medium", "high", "xhigh"}
	case strings.HasPrefix(model, "gpt-5.4-pro"):
		efforts = []string{"medium", "high", "xhigh"}
	case strings.HasPrefix(model, "gpt-5.4-mini"):
		efforts = []string{"none", "low", "medium", "high", "xhigh"}
	case strings.HasPrefix(model, "gpt-5.4-nano"):
		return nil
	case strings.HasPrefix(model, "gpt-5.4"):
		efforts = []string{"none", "low", "medium", "high", "xhigh"}
	case model == "gpt-5.2-pro" || strings.HasPrefix(model, "gpt-5.2-pro-"):
		efforts = []string{"medium", "high", "xhigh"}
	case model == "gpt-5.2-chat-latest":
		return nil
	case strings.HasPrefix(model, "gpt-5.2"):
		efforts = []string{"none", "low", "medium", "high", "xhigh"}
	case strings.HasPrefix(model, "gpt-5.1"):
		efforts = []string{"none", "low", "medium", "high"}
	case model == "gpt-5-pro" || strings.HasPrefix(model, "gpt-5-pro-"):
		efforts = []string{"high"}
	case model == "gpt-5-mini" || strings.HasPrefix(model, "gpt-5-mini-"):
		return nil
	case model == "gpt-5-nano" || strings.HasPrefix(model, "gpt-5-nano-"):
		return nil
	case model == "gpt-5":
		efforts = []string{"minimal", "low", "medium", "high"}
	case model == "o3" || strings.HasPrefix(model, "o3-"):
		efforts = []string{"low", "medium", "high"}
	case model == "o1" || strings.HasPrefix(model, "o1-"):
		efforts = []string{"low", "medium", "high"}
	default:
		return nil
	}

	return append([]string(nil), efforts...)
}

func openAIResponsesSamplingRestriction(model string) SamplingRestriction {
	switch {
	case model == "gpt-5.5" || strings.HasPrefix(model, "gpt-5.5-"):
		return SamplingRestrictionAlways
	case strings.HasPrefix(model, "gpt-5.4"):
		return SamplingRestrictionAlways
	case model == "gpt-5.2-pro" || strings.HasPrefix(model, "gpt-5.2-pro-"):
		return SamplingRestrictionAlways
	case strings.HasPrefix(model, "gpt-5.2"):
		return SamplingRestrictionAlways
	case strings.HasPrefix(model, "gpt-5.1"):
		return SamplingRestrictionWithReasoning
	case model == "gpt-5" || strings.HasPrefix(model, "gpt-5-"):
		return SamplingRestrictionAlways
	case model == "o3" || strings.HasPrefix(model, "o3-"):
		return SamplingRestrictionAlways
	case model == "o1" || strings.HasPrefix(model, "o1-"):
		return SamplingRestrictionAlways
	case model == "o4-mini" || strings.HasPrefix(model, "o4-mini-"):
		return SamplingRestrictionAlways
	default:
		return SamplingRestrictionNone
	}
}

func isOpenAIGPT54Model(model string) bool {
	return strings.HasPrefix(model, "gpt-5.4") && !strings.Contains(model, "pro")
}
