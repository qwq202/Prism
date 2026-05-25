package auth

import (
	"testing"
	"time"

	"chat/channel"

	"github.com/spf13/viper"
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

func TestBuySubscriptionReturnsErrorWhenLocalUpdateFailsAfterPayment(t *testing.T) {
	db := openAuthSecurityTestDB(t)
	user := GetUserByName(db, "root")
	if user == nil {
		t.Fatalf("expected root user")
	}
	if !user.SetQuota(db, 20) {
		t.Fatalf("set quota")
	}

	previousUseDeeptrain := viper.GetBool("auth.use_deeptrain")
	viper.Set("auth.use_deeptrain", false)
	t.Cleanup(func() {
		viper.Set("auth.use_deeptrain", previousUseDeeptrain)
	})

	previousPlanInstance := channel.PlanInstance
	channel.PlanInstance = &channel.PlanManager{
		Enabled: true,
		Plans: []channel.Plan{
			{Level: 1, Price: 1},
		},
	}
	t.Cleanup(func() {
		channel.PlanInstance = previousPlanInstance
	})

	err := BuySubscription(db, nil, user, 1, 1)
	if err == nil || err.Error() != "payment succeeded but failed to update subscription" {
		t.Fatalf("expected local subscription update failure, got %v", err)
	}
	if quota := user.GetQuota(db); quota != 10 {
		t.Fatalf("expected payment quota to be deducted before update failure, got %f", quota)
	}
}
