package manager

import (
	"chat/auth"
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"math"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func TestRequestBillingSessionSettlesWalletReservationDelta(t *testing.T) {
	useChatTokenTestChargeInstance(t)
	db := openChatQuotaTestDB(t)
	user := auth.GetUserByName(db, "root")
	if user == nil || !user.SetQuota(db, 10) {
		t.Fatalf("seed user quota")
	}

	buffer := utils.NewBuffer(globals.GPT3Turbo, nil, channel.ChargeInstance.GetCharge(globals.GPT3Turbo))
	session, err := newRequestBillingSession(db, nil, user, globals.GPT3Turbo, buffer, false, nil)
	if err != nil {
		t.Fatalf("reserve request quota: %v", err)
	}
	if got := user.GetQuota(db); math.Abs(float64(got-9)) > 0.001 {
		t.Fatalf("expected 1 quota to be reserved, got remaining %f", got)
	}

	session.settle(0.4)
	session.Refund()
	if got := user.GetQuota(db); math.Abs(float64(got-9.6)) > 0.001 {
		t.Fatalf("expected unused reservation to be refunded, got %f", got)
	}
	if got := user.GetUsedQuota(db); math.Abs(float64(got-0.4)) > 0.001 {
		t.Fatalf("expected actual usage 0.4, got %f", got)
	}
}

func TestRequestBillingSessionRefundIsIdempotent(t *testing.T) {
	useChatTokenTestChargeInstance(t)
	db := openChatQuotaTestDB(t)
	user := auth.GetUserByName(db, "root")
	if user == nil || !user.SetQuota(db, 10) {
		t.Fatalf("seed user quota")
	}

	buffer := utils.NewBuffer(globals.GPT3Turbo, nil, channel.ChargeInstance.GetCharge(globals.GPT3Turbo))
	session, err := newRequestBillingSession(db, nil, user, globals.GPT3Turbo, buffer, false, nil)
	if err != nil {
		t.Fatalf("reserve request quota: %v", err)
	}
	session.Refund()
	session.Refund()

	if got := user.GetQuota(db); math.Abs(float64(got-10)) > 0.001 {
		t.Fatalf("expected the full reservation to be refunded once, got %f", got)
	}
	if got := user.GetUsedQuota(db); math.Abs(float64(got)) > 0.001 {
		t.Fatalf("expected used quota to return to zero, got %f", got)
	}
}

func TestRequestBillingSessionSettlesSubscriptionPointReservation(t *testing.T) {
	useChatTestChargeInstance(t)
	db := openChatQuotaTestDB(t)
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	cache := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = cache.Close()
		server.Close()
	})

	user := auth.GetUserByName(db, "root")
	if user == nil || !user.SetQuota(db, 10) {
		t.Fatalf("seed user quota")
	}
	if !user.SetAllowSubscriptionQuotaFallback(db, false) {
		t.Fatalf("disable wallet fallback")
	}
	configureRequestBillingTestPlan(t, db, user, 10)

	buffer := utils.NewBuffer(globals.GPT3Turbo, nil, channel.ChargeInstance.GetCharge(globals.GPT3Turbo))
	session, err := newRequestBillingSession(db, cache, user, globals.GPT3Turbo, buffer, true, nil)
	if err != nil {
		t.Fatalf("reserve subscription quota: %v", err)
	}
	if !session.UsesPlan() {
		t.Fatalf("expected subscription funding")
	}
	plan := user.GetPlan(db)
	if got := plan.GetPointUsage(user, cache); math.Abs(float64(got-9)) > 0.001 {
		t.Fatalf("expected 9 subscription points to be reserved, got %f", got)
	}

	session.settle(4)
	if got := plan.GetPointUsage(user, cache); math.Abs(float64(got-4)) > 0.001 {
		t.Fatalf("expected subscription settlement to keep 4 points, got %f", got)
	}
	if got := user.GetQuota(db); math.Abs(float64(got-10)) > 0.001 {
		t.Fatalf("expected wallet quota to remain untouched, got %f", got)
	}
}

func TestRequestBillingSessionRollsBackPartialPointReservationOnFailure(t *testing.T) {
	useChatTokenTestChargeInstance(t)
	db := openChatQuotaTestDB(t)
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	cache := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = cache.Close()
		server.Close()
	})

	user := auth.GetUserByName(db, "root")
	if user == nil || !user.SetQuota(db, 10) {
		t.Fatalf("seed user quota")
	}
	if !user.SetAllowSubscriptionQuotaFallback(db, false) {
		t.Fatalf("disable wallet fallback")
	}
	configureRequestBillingTestPlan(t, db, user, 1)

	maxTokens := 2000
	buffer := utils.NewBuffer(globals.GPT3Turbo, nil, channel.ChargeInstance.GetCharge(globals.GPT3Turbo))
	if _, err := newRequestBillingSession(db, cache, user, globals.GPT3Turbo, buffer, true, &maxTokens); err == nil {
		t.Fatalf("expected reservation above the point pool to fail")
	}
	plan := user.GetPlan(db)
	if got := plan.GetPointUsage(user, cache); math.Abs(float64(got)) > 0.001 {
		t.Fatalf("expected failed reservation to roll back points, got %f", got)
	}
}

func configureRequestBillingTestPlan(t *testing.T, db *sql.DB, user *auth.User, quota float32) {
	t.Helper()

	previousPlan := channel.PlanInstance
	channel.PlanInstance = &channel.PlanManager{
		Enabled: true,
		Plans: []channel.Plan{
			{
				Level: 1,
				Quota: quota,
				Items: []channel.PlanItem{
					{Id: "included", Models: []string{globals.GPT3Turbo}},
				},
			},
		},
	}
	t.Cleanup(func() {
		channel.PlanInstance = previousPlan
	})

	if _, err := globals.ExecDb(
		db,
		"INSERT INTO subscription (user_id, level, expired_at) VALUES (?, ?, ?)",
		user.GetID(db),
		1,
		time.Now().Add(24*time.Hour).Format("2006-01-02 15:04:05"),
	); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}
}
