package channel

import (
	"chat/globals"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
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

func TestPlanSharedPointPoolUsesCustomResetInterval(t *testing.T) {
	server, cache := openPlanTestCache(t)
	user := planTestUser{id: 42}
	plan := Plan{
		Level:         1,
		Quota:         2,
		ResetInterval: int64((5 * time.Hour).Seconds()),
		Items: []PlanItem{
			{
				Id:     "all-models",
				Models: []string{"deepseek-v4-flash", "gpt-5.1"},
			},
		},
	}

	if !plan.CanUsePointPool(user, cache, "deepseek-v4-flash") {
		t.Fatalf("expected included model to be able to use point pool")
	}
	if plan.CanUsePointPool(user, cache, "claude-4") {
		t.Fatalf("expected excluded model to be rejected")
	}

	if !plan.ConsumePointPool(user, cache, "deepseek-v4-flash", 0.75) ||
		!plan.ConsumePointPool(user, cache, "gpt-5.1", 1.0) {
		t.Fatalf("expected shared point pool consumption to be accepted")
	}
	if plan.ConsumePointPool(user, cache, "gpt-5.1", 0.5) {
		t.Fatalf("expected over-limit shared point pool consumption to be rejected")
	}

	key := globals.GetSubscriptionLimitFormat(plan.pointUsageKey(), user.HitID())
	server.Set(key, fmt.Sprintf("%s/%.6f", time.Now().Add(-6*time.Hour).Format("2006-01-02:15:04:05"), float32(2)))

	usage := plan.GetPointUsageForm(user, cache)
	if usage.Used != 0 {
		t.Fatalf("expected usage to reset after custom interval, got %f", usage.Used)
	}
	if usage.Unit != PlanItemUnitPoints {
		t.Fatalf("expected point unit, got %q", usage.Unit)
	}
	if usage.ResetInterval != plan.ResetInterval {
		t.Fatalf("expected reset interval %d, got %d", plan.ResetInterval, usage.ResetInterval)
	}
	if usage.ResetAt == "" {
		t.Fatalf("expected next reset time to be exposed")
	}

	raw, err := server.Get(key)
	if err != nil {
		t.Fatalf("expected original point usage cache to remain readable: %v", err)
	}
	if !strings.HasSuffix(raw, "/2.000000") {
		t.Fatalf("expected point usage reads not to overwrite cache, got %q", raw)
	}
}

func TestPlanSharedPointPoolConcurrentConsumptionCannotExceedLimit(t *testing.T) {
	_, cache := openPlanTestCache(t)
	user := planTestUser{id: 88}
	plan := Plan{
		Level: 1,
		Quota: 1,
		Items: []PlanItem{
			{
				Id:     "all-models",
				Models: []string{"gpt-5.1"},
			},
		},
	}

	var success atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if plan.ConsumePointPool(user, cache, "gpt-5.1", 0.25) {
				success.Add(1)
			}
		}()
	}

	close(start)
	wg.Wait()

	if got := success.Load(); got != 4 {
		t.Fatalf("expected exactly 4 successful point consumptions, got %d", got)
	}
	if usage := plan.GetPointUsage(user, cache); usage > plan.Quota {
		t.Fatalf("expected point usage to stay within limit, got %f", usage)
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
