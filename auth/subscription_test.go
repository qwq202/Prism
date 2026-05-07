package auth

import (
	"testing"
	"time"
)

func TestCalculateDowngradeExpiredAtKeepsFreePlanExpiry(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	expired := now.AddDate(0, 1, 0)

	got := calculateDowngradeExpiredAt(expired, now, 0, 0)
	if !got.Equal(expired) {
		t.Fatalf("expected free plan downgrade to keep expiry %s, got %s", expired, got)
	}
}

func TestCalculateDowngradeExpiredAtScalesPaidPlanExpiry(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	expired := now.Add(10 * 24 * time.Hour)

	got := calculateDowngradeExpiredAt(expired, now, 20, 10)
	want := now.Add(20*24*time.Hour).AddDate(0, 0, -1)
	if !got.Equal(want) {
		t.Fatalf("expected paid plan downgrade expiry %s, got %s", want, got)
	}
}
