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
