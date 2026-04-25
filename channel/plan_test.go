package channel

import (
	"chat/globals"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

type planTestUser struct {
	id int64
}

func (u planTestUser) GetID(_ *sql.DB) int64 {
	return u.id
}

func (u planTestUser) HitID() int64 {
	return u.id
}

func openPlanTestCache(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()

	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}

	cache := redis.NewClient(&redis.Options{Addr: server.Addr()})
	if err := cache.Ping(cache.Context()).Err(); err != nil {
		t.Fatalf("ping miniredis: %v", err)
	}

	t.Cleanup(func() {
		_ = cache.Close()
		server.Close()
	})

	return server, cache
}

func TestValidatePlanConfigModels(t *testing.T) {
	originalConduit := ConduitInstance
	ConduitInstance = &Manager{Models: []string{"deepseek-v4-flash", "grok-4-1-fast-reasoning"}}
	defer func() {
		ConduitInstance = originalConduit
	}()

	valid := &PlanManager{
		Plans: []Plan{
			{
				Level: 1,
				Items: []PlanItem{
					{
						Id:     "valid-item",
						Models: []string{"deepseek-v4-flash"},
					},
				},
			},
		},
	}

	if err := validatePlanConfigModels(valid); err != nil {
		t.Fatalf("expected valid plan config, got error: %v", err)
	}

	invalid := &PlanManager{
		Plans: []Plan{
			{
				Level: 1,
				Items: []PlanItem{
					{
						Id:     "invalid-item",
						Models: []string{"deepseek-v4-flash", "gpt-4o"},
					},
				},
			},
		},
	}

	if err := validatePlanConfigModels(invalid); err == nil {
		t.Fatal("expected invalid plan config to be rejected")
	}
}

func TestPlanItemUsesCustomResetInterval(t *testing.T) {
	server, cache := openPlanTestCache(t)
	user := planTestUser{id: 42}
	item := PlanItem{
		Id:            "deepseek-points",
		Value:         2,
		Unit:          PlanItemUnitPoints,
		ResetInterval: int64((5 * time.Hour).Seconds()),
	}

	if !item.Increase(user, cache) || !item.Increase(user, cache) {
		t.Fatalf("expected first two point uses to be accepted")
	}
	if item.Increase(user, cache) {
		t.Fatalf("expected third point use before reset to be rejected")
	}

	key := globals.GetSubscriptionLimitFormat(item.Id, user.HitID())
	server.Set(key, fmt.Sprintf("%s/%d", time.Now().Add(-6*time.Hour).Format("2006-01-02:15:04:05"), 2))

	usage := item.GetUsageForm(user, nil, cache)
	if usage.Used != 0 {
		t.Fatalf("expected usage to reset after custom interval, got %d", usage.Used)
	}
	if usage.Unit != PlanItemUnitPoints {
		t.Fatalf("expected point unit, got %q", usage.Unit)
	}
	if usage.ResetInterval != item.ResetInterval {
		t.Fatalf("expected reset interval %d, got %d", item.ResetInterval, usage.ResetInterval)
	}
	if usage.ResetAt == "" {
		t.Fatalf("expected next reset time to be exposed")
	}

	if !item.Increase(user, cache) {
		t.Fatalf("expected point use after reset to be accepted")
	}
}

func TestPlanItemDefaultsToMonthlyTimesQuota(t *testing.T) {
	_, cache := openPlanTestCache(t)
	user := planTestUser{id: 7}
	item := PlanItem{Id: "legacy-times", Value: 1}

	usage := item.GetUsageForm(user, nil, cache)
	if usage.Unit != PlanItemUnitTimes {
		t.Fatalf("expected legacy item to default to times, got %q", usage.Unit)
	}
	if usage.ResetInterval != 0 {
		t.Fatalf("expected legacy item to keep monthly reset interval, got %d", usage.ResetInterval)
	}
	if !item.Increase(user, cache) {
		t.Fatalf("expected first use to be accepted")
	}
	if item.Increase(user, cache) {
		t.Fatalf("expected second use to be rejected")
	}
}
