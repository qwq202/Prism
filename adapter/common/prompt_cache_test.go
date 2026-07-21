package adaptercommon

import (
	"chat/globals"
	"chat/utils"
	"strings"
	"testing"
)

func withPromptCacheConfig(t *testing.T, enabled bool, minTokens int) {
	t.Helper()
	oldEnabled := globals.PromptCacheEnabled
	oldMinTokens := globals.PromptCacheMinTokens
	globals.PromptCacheEnabled = enabled
	globals.PromptCacheMinTokens = minTokens
	t.Cleanup(func() {
		globals.PromptCacheEnabled = oldEnabled
		globals.PromptCacheMinTokens = oldMinTokens
	})
}

func newPromptCacheTestProps(channelType string, model string, tokens int) *ChatProps {
	buffer := &utils.Buffer{}
	buffer.SetInputTokens(tokens)
	return &ChatProps{
		Model:         model,
		OriginalModel: model,
		ChannelType:   channelType,
		Buffer:        buffer,
	}
}

func TestApplyPromptCacheDefaultsAddsOpenAIRoutingKey(t *testing.T) {
	withPromptCacheConfig(t, true, 0)
	sessionID := "user-1-conversation-2"
	props := newPromptCacheTestProps(globals.OpenAIChannelType, "gpt-5.5", 1500)
	props.SessionID = &sessionID

	ApplyPromptCacheDefaults(props)

	if props.PromptCacheKey == nil || *props.PromptCacheKey != "coai:"+sessionID {
		t.Fatalf("expected OpenAI prompt cache key to use session id, got %#v", props.PromptCacheKey)
	}

	detail := props.Buffer.GetPromptCache()
	if detail == nil {
		t.Fatalf("expected prompt cache detail")
	}
	if detail.Provider != "openai" || detail.Mode != "routing_key" || !detail.Attempted || !detail.Eligible {
		t.Fatalf("unexpected OpenAI prompt cache detail: %#v", detail)
	}
	if !detail.PromptCacheKey {
		t.Fatalf("expected prompt cache key flag in detail: %#v", detail)
	}
	if raw := props.Buffer.GetBillingDetail(); !strings.Contains(raw, `"prompt_cache"`) || !strings.Contains(raw, `"attempted":true`) {
		t.Fatalf("expected billing detail to include prompt cache status, got %q", raw)
	}
}

func TestApplyPromptCacheDefaultsAddsClaudeAutomaticCacheControl(t *testing.T) {
	withPromptCacheConfig(t, true, 0)
	props := newPromptCacheTestProps(globals.ClaudeChannelType, "claude-opus-4-8", 1500)

	ApplyPromptCacheDefaults(props)

	if props.CacheControl == nil || props.CacheControl["type"] != "ephemeral" {
		t.Fatalf("expected Claude automatic cache_control, got %#v", props.CacheControl)
	}

	detail := props.Buffer.GetPromptCache()
	if detail == nil || detail.Provider != "claude" || detail.Mode != "automatic" || !detail.CacheControl || !detail.Attempted {
		t.Fatalf("unexpected Claude prompt cache detail: %#v", detail)
	}
}

func TestApplyPromptCacheDefaultsSkipsClaudeBelowThreshold(t *testing.T) {
	withPromptCacheConfig(t, true, 0)
	props := newPromptCacheTestProps(globals.ClaudeChannelType, "claude-opus-4-8", 512)

	ApplyPromptCacheDefaults(props)

	if props.CacheControl != nil {
		t.Fatalf("did not expect Claude cache_control below threshold, got %#v", props.CacheControl)
	}

	detail := props.Buffer.GetPromptCache()
	if detail == nil || detail.Status != "below_threshold" || detail.Attempted || detail.Eligible {
		t.Fatalf("unexpected below-threshold detail: %#v", detail)
	}
}

func TestApplyPromptCacheDefaultsRecordsGeminiImplicitThreshold(t *testing.T) {
	withPromptCacheConfig(t, true, 0)
	props := newPromptCacheTestProps(globals.PalmChannelType, globals.Gemini35Flash, 3043)

	ApplyPromptCacheDefaults(props)

	detail := props.Buffer.GetPromptCache()
	if detail == nil {
		t.Fatalf("expected Gemini prompt cache detail")
	}
	if detail.Provider != "gemini" || detail.Mode != "implicit" || detail.ThresholdTokens != 4096 {
		t.Fatalf("unexpected Gemini prompt cache detail: %#v", detail)
	}
	if detail.Status != "below_threshold" || detail.Attempted || detail.Eligible {
		t.Fatalf("expected Gemini to be below threshold, got %#v", detail)
	}
}

func TestApplyPromptCacheDefaultsMarksGeminiImplicitEligible(t *testing.T) {
	withPromptCacheConfig(t, true, 0)
	props := newPromptCacheTestProps(globals.PalmChannelType, globals.Gemini35Flash, 5000)

	ApplyPromptCacheDefaults(props)

	detail := props.Buffer.GetPromptCache()
	if detail == nil || detail.Provider != "gemini" || detail.Status != "attempted" || !detail.Attempted || !detail.Eligible {
		t.Fatalf("unexpected eligible Gemini prompt cache detail: %#v", detail)
	}
}

func TestApplyPromptCacheDefaultsUsesGemini36FlashThreshold(t *testing.T) {
	withPromptCacheConfig(t, true, 0)
	props := newPromptCacheTestProps(globals.PalmChannelType, globals.Gemini36Flash, 4096)

	ApplyPromptCacheDefaults(props)

	detail := props.Buffer.GetPromptCache()
	if detail == nil || detail.Provider != "gemini" || detail.ThresholdTokens != 4096 {
		t.Fatalf("unexpected Gemini 3.6 prompt cache detail: %#v", detail)
	}
	if detail.Status != "attempted" || !detail.Attempted || !detail.Eligible {
		t.Fatalf("expected Gemini 3.6 to be eligible at 4096 tokens, got %#v", detail)
	}
}

func TestApplyPromptCacheDefaultsHonorsDisabledConfig(t *testing.T) {
	withPromptCacheConfig(t, false, 0)
	props := newPromptCacheTestProps(globals.OpenAIChannelType, "gpt-5.5", 1500)

	ApplyPromptCacheDefaults(props)

	if props.PromptCacheKey != nil {
		t.Fatalf("did not expect prompt cache key when disabled, got %#v", props.PromptCacheKey)
	}
	if detail := props.Buffer.GetPromptCache(); detail != nil {
		t.Fatalf("did not expect prompt cache detail when disabled, got %#v", detail)
	}
}

func TestApplyPromptCacheDefaultsHonorsConfiguredMinimum(t *testing.T) {
	withPromptCacheConfig(t, true, 2000)
	props := newPromptCacheTestProps(globals.OpenAIChannelType, "gpt-5.5", 1500)

	ApplyPromptCacheDefaults(props)

	if props.PromptCacheKey != nil {
		t.Fatalf("did not expect prompt cache key below configured threshold, got %#v", props.PromptCacheKey)
	}
	detail := props.Buffer.GetPromptCache()
	if detail == nil || detail.ThresholdTokens != 2000 || detail.Status != "below_threshold" {
		t.Fatalf("unexpected configured minimum detail: %#v", detail)
	}
}
