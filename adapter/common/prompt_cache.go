package adaptercommon

import (
	"chat/globals"
	"chat/utils"
	"fmt"
	"strings"
)

const (
	promptCacheProviderOpenAI = "openai"
	promptCacheProviderClaude = "claude"
	promptCacheProviderGemini = "gemini"

	promptCacheModeAutomatic     = "automatic"
	promptCacheModeImplicit      = "implicit"
	promptCacheModeRoutingKey    = "routing_key"
	promptCacheStatusAttempted   = "attempted"
	promptCacheStatusBelowLimit  = "below_threshold"
	promptCacheStatusManual      = "manual"
	promptCacheStatusUnsupported = "unsupported"

	openAIPromptCacheThreshold = 1024
	claudePromptCacheThreshold = 1024
)

func normalizePromptCacheChannel(channelType string) string {
	return strings.TrimSpace(strings.ToLower(channelType))
}

func promptCachePromptTokens(props *ChatProps) int {
	if props == nil || props.Buffer == nil {
		return 0
	}
	return props.Buffer.CountInputToken()
}

func promptCacheThreshold(defaultThreshold int) int {
	if globals.PromptCacheMinTokens > defaultThreshold {
		return globals.PromptCacheMinTokens
	}
	return defaultThreshold
}

func hasManualPromptCacheControl(props *ChatProps) bool {
	return props != nil &&
		(props.CacheControl != nil ||
			normalizedStringValue(props.PromptCacheKey) != "" ||
			normalizedStringValue(props.PromptCacheRetention) != "" ||
			normalizedStringValue(props.CachedContent) != "" ||
			normalizedStringValue(props.CachedContentSnake) != "")
}

func normalizedStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func applyPromptCacheDetail(props *ChatProps, detail globals.PromptCacheDetail) {
	if props == nil || props.Buffer == nil {
		return
	}
	props.Buffer.SetPromptCache(&detail)
}

func promptCacheSessionKey(props *ChatProps) string {
	if props == nil {
		return ""
	}
	if sessionID := normalizedStringValue(props.SessionID); sessionID != "" {
		return "coai:" + sessionID
	}
	model := strings.TrimSpace(props.OriginalModel)
	if model == "" {
		model = strings.TrimSpace(props.Model)
	}
	if model == "" {
		return ""
	}
	return "coai:model:" + utils.Md5Encrypt(model)
}

func geminiImplicitCacheThreshold(model string) int {
	model = strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasPrefix(model, "gemini-3.5-flash"),
		strings.HasPrefix(model, "gemini-3.1-pro"):
		return 4096
	case strings.HasPrefix(model, "gemini-2.5-flash"),
		strings.HasPrefix(model, "gemini-2.5-pro"):
		return 2048
	default:
		return 0
	}
}

func openAIPromptCacheSupported(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(model, "gpt-5") ||
		strings.HasPrefix(model, "gpt-4.1") ||
		strings.HasPrefix(model, "gpt-4o") ||
		strings.HasPrefix(model, "o1") ||
		strings.HasPrefix(model, "o3") ||
		strings.HasPrefix(model, "o4")
}

func applyOpenAIPromptCache(props *ChatProps, channelType string) {
	tokens := promptCachePromptTokens(props)
	manual := hasManualPromptCacheControl(props)
	threshold := promptCacheThreshold(openAIPromptCacheThreshold)
	eligible := tokens >= threshold
	detail := globals.PromptCacheDetail{
		Provider:        promptCacheProviderOpenAI,
		Mode:            promptCacheModeRoutingKey,
		ChannelType:     channelType,
		PromptTokens:    tokens,
		ThresholdTokens: threshold,
		Eligible:        eligible,
		Manual:          manual,
	}

	if !openAIPromptCacheSupported(props.Model) && !manual {
		detail.Status = promptCacheStatusUnsupported
		detail.Reason = "model is not listed for OpenAI automatic prompt caching"
		applyPromptCacheDetail(props, detail)
		return
	}

	if !eligible {
		detail.Status = promptCacheStatusBelowLimit
		detail.Reason = fmt.Sprintf("prompt tokens below %d-token OpenAI cache threshold", threshold)
		applyPromptCacheDetail(props, detail)
		return
	}

	if normalizedStringValue(props.PromptCacheKey) == "" {
		if key := promptCacheSessionKey(props); key != "" {
			props.PromptCacheKey = &key
			detail.PromptCacheKey = true
		}
	} else {
		detail.PromptCacheKey = true
	}
	if retention := normalizedStringValue(props.PromptCacheRetention); retention != "" {
		detail.PromptCacheRetention = retention
	}

	detail.Attempted = true
	if manual {
		detail.Status = promptCacheStatusManual
	} else {
		detail.Status = promptCacheStatusAttempted
	}
	applyPromptCacheDetail(props, detail)
}

func applyClaudePromptCache(props *ChatProps, channelType string) {
	tokens := promptCachePromptTokens(props)
	manual := hasManualPromptCacheControl(props)
	threshold := promptCacheThreshold(claudePromptCacheThreshold)
	eligible := tokens >= threshold
	detail := globals.PromptCacheDetail{
		Provider:        promptCacheProviderClaude,
		Mode:            promptCacheModeAutomatic,
		ChannelType:     channelType,
		PromptTokens:    tokens,
		ThresholdTokens: threshold,
		Eligible:        eligible,
		Manual:          manual,
	}

	if !eligible {
		detail.Status = promptCacheStatusBelowLimit
		detail.Reason = fmt.Sprintf("prompt tokens below %d-token Claude cache threshold", threshold)
		applyPromptCacheDetail(props, detail)
		return
	}

	if props.CacheControl == nil {
		props.CacheControl = map[string]interface{}{"type": "ephemeral"}
	}
	detail.CacheControl = props.CacheControl != nil
	detail.Attempted = detail.CacheControl
	if manual {
		detail.Status = promptCacheStatusManual
	} else {
		detail.Status = promptCacheStatusAttempted
	}
	applyPromptCacheDetail(props, detail)
}

func applyGeminiPromptCache(props *ChatProps, channelType string) {
	model := strings.TrimSpace(props.Model)
	if model == "" {
		model = strings.TrimSpace(props.OriginalModel)
	}
	baseThreshold := geminiImplicitCacheThreshold(model)
	if baseThreshold <= 0 {
		applyPromptCacheDetail(props, globals.PromptCacheDetail{
			Provider:    promptCacheProviderGemini,
			Mode:        promptCacheModeImplicit,
			ChannelType: channelType,
			Status:      promptCacheStatusUnsupported,
			Reason:      "model is not listed for Gemini implicit prompt caching",
			Manual:      hasManualPromptCacheControl(props),
		})
		return
	}
	threshold := promptCacheThreshold(baseThreshold)

	tokens := promptCachePromptTokens(props)
	manual := hasManualPromptCacheControl(props)
	eligible := tokens >= threshold
	detail := globals.PromptCacheDetail{
		Provider:        promptCacheProviderGemini,
		Mode:            promptCacheModeImplicit,
		ChannelType:     channelType,
		PromptTokens:    tokens,
		ThresholdTokens: threshold,
		Eligible:        eligible,
		Attempted:       eligible,
		Manual:          manual,
	}
	if manual {
		detail.Status = promptCacheStatusManual
	} else if eligible {
		detail.Status = promptCacheStatusAttempted
	} else {
		detail.Status = promptCacheStatusBelowLimit
		detail.Reason = fmt.Sprintf("prompt tokens below %d-token Gemini implicit cache threshold", threshold)
	}
	if normalizedStringValue(props.CachedContent) != "" || normalizedStringValue(props.CachedContentSnake) != "" {
		detail.Mode = "explicit"
		detail.Attempted = true
	}
	applyPromptCacheDetail(props, detail)
}

// ApplyPromptCacheDefaults mutates props with provider-supported automatic prompt
// cache controls and records a small, non-sensitive billing detail.
func ApplyPromptCacheDefaults(props *ChatProps) {
	if props == nil || props.Buffer == nil {
		return
	}
	if !globals.PromptCacheEnabled {
		return
	}

	channelType := normalizePromptCacheChannel(props.ChannelType)
	switch channelType {
	case globals.OpenAIChannelType,
		globals.AzureOpenAIChannelType,
		globals.OpenAIResponsesChannelType:
		applyOpenAIPromptCache(props, channelType)
	case globals.ClaudeChannelType,
		globals.GLMCodingPlanCNChannelType,
		globals.MiniMaxTokenPlanCNChannelType:
		applyClaudePromptCache(props, channelType)
	case globals.PalmChannelType,
		globals.GeminiEnterpriseAgentPlatformChannelType:
		if globals.IsGeminiModel(props.Model) {
			applyGeminiPromptCache(props, channelType)
		}
	default:
		applyPromptCacheDetail(props, globals.PromptCacheDetail{
			ChannelType:  channelType,
			Status:       promptCacheStatusUnsupported,
			PromptTokens: promptCachePromptTokens(props),
			Reason:       "channel does not expose automatic prompt cache controls",
			Manual:       hasManualPromptCacheControl(props),
		})
	}
}
