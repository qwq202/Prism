package channel

import (
	"chat/globals"
	"testing"
)

func TestImageBillingEstimateUsesSizeQualityAndReferences(t *testing.T) {
	charge := &Charge{
		Type:   globals.ImageBilling,
		Output: 3,
		Image: &ImageChargeConfig{
			Default:   4,
			Request:   1,
			Reference: 0.5,
			Size: map[string]float32{
				"2K": 6,
			},
			Quality: map[string]float32{
				"high": 2,
			},
		},
	}

	got := charge.EstimateImageQuota(3, map[string]interface{}{
		"image_size": "2k",
		"quality":    "high",
	}, 2)
	want := float32(18.5)
	if got != want {
		t.Fatalf("expected quota %v, got %v", want, got)
	}
}

func TestImageBillingEstimateFallsBackToOutputPrice(t *testing.T) {
	charge := &Charge{
		Type:   globals.ImageBilling,
		Output: 2.5,
	}

	got := charge.EstimateImageQuota(0, map[string]interface{}{
		"image_size": "unsupported",
	}, 1)
	want := float32(2.5)
	if got != want {
		t.Fatalf("expected fallback quota %v, got %v", want, got)
	}
}

func TestImageBillingMatrixRuleOverridesLegacyPrices(t *testing.T) {
	charge := &Charge{
		Type: globals.ImageBilling,
		Image: &ImageChargeConfig{
			Mode: ImageBillingModeMatrix,
			Rules: []ImageChargeRule{
				{Size: "1K", Quality: "high", MimeType: "image/png", Quota: 7},
			},
			Request:   1,
			Reference: 0.5,
		},
	}

	quote, err := charge.GetImageBillingQuote(2, map[string]interface{}{
		"image_size": "1k",
		"quality":    "high",
		"mime_type":  "image/png",
	}, 3, nil)
	if err != nil {
		t.Fatalf("expected matrix quote, got error: %v", err)
	}

	if quote.TotalQuota != 23 {
		t.Fatalf("expected total quota 23, got %#v", quote)
	}
	if quote.MatchedRule == nil || quote.MatchedRuleIndex != 0 {
		t.Fatalf("expected matched matrix rule, got %#v", quote)
	}
}

func TestImageBillingMatrixRejectsMissingRule(t *testing.T) {
	charge := &Charge{
		Type: globals.ImageBilling,
		Image: &ImageChargeConfig{
			Mode:               ImageBillingModeMatrix,
			MissingPricePolicy: ImageMissingPricePolicyReject,
			Rules: []ImageChargeRule{
				{Size: "1K", Quality: "high", Quota: 7},
			},
		},
	}

	if err := charge.ValidateImageBilling(map[string]interface{}{
		"image_size": "4K",
		"quality":    "high",
	}); err == nil {
		t.Fatalf("expected missing matrix rule to reject request")
	}
}

func TestImageBillingOfficialUsageUsesUsagePrices(t *testing.T) {
	charge := &Charge{
		Type: globals.ImageBilling,
		Image: &ImageChargeConfig{
			Mode:               ImageBillingModeOfficialUsage,
			MissingPricePolicy: ImageMissingPricePolicyReject,
			Request:            1,
			Usage: &ImageUsageChargeConfig{
				Input:  0.5,
				Output: 1,
				Image:  2,
			},
		},
	}

	got := charge.EstimateImageQuotaWithUsage(0, nil, 1, &globals.TokenUsage{
		PromptTokens:     1000,
		CompletionTokens: 2000,
		ImageTokens:      3000,
	})
	want := float32(9.5)
	if got != want {
		t.Fatalf("expected official usage quota %v, got %v", want, got)
	}
}
