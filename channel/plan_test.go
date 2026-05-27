package channel

import (
	"chat/globals"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
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

func TestPlanConfigReadsSavedWindowQuotaKeys(t *testing.T) {
	conf := viper.New()
	conf.SetConfigType("yaml")

	if err := conf.ReadConfig(strings.NewReader(`
subscription:
  enabled: true
  plans:
    - level: 1
      price: 0
      quota: 20
      resetinterval: 18000
      weeklyquota: 200
      items: []
`)); err != nil {
		t.Fatalf("read config: %v", err)
	}

	var manager PlanManager
	if err := conf.UnmarshalKey("subscription", &manager); err != nil {
		t.Fatalf("unmarshal subscription: %v", err)
	}

	if len(manager.Plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(manager.Plans))
	}
	plan := manager.Plans[0]
	if plan.ResetInterval != int64((5 * time.Hour).Seconds()) {
		t.Fatalf("expected reset interval 18000, got %d", plan.ResetInterval)
	}
	if plan.WeeklyQuota != 200 {
		t.Fatalf("expected weekly quota 200, got %f", plan.WeeklyQuota)
	}
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

func TestUpdatePlanConfigKeepsRuntimeStateWhenSaveFails(t *testing.T) {
	originalConduit := ConduitInstance
	ConduitInstance = nil
	t.Cleanup(func() {
		ConduitInstance = originalConduit
	})

	previousSave := savePlanConfig
	savePlanConfig = func(*PlanManager) error {
		return errors.New("simulated save failure")
	}
	t.Cleanup(func() {
		savePlanConfig = previousSave
	})

	manager := &PlanManager{
		Enabled: false,
		Plans: []Plan{
			{Level: 1, Quota: 10},
		},
	}
	next := &PlanManager{
		Enabled: true,
		Plans: []Plan{
			{Level: 2, Quota: 20},
		},
	}

	err := manager.UpdateConfig(next)
	if err == nil || err.Error() != "simulated save failure" {
		t.Fatalf("expected simulated save failure, got %v", err)
	}
	if manager.Enabled {
		t.Fatalf("expected enabled flag to remain false")
	}
	if len(manager.Plans) != 1 || manager.Plans[0].Level != 1 {
		t.Fatalf("expected plans to remain unchanged, got %#v", manager.Plans)
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
	if plan.CanUsePointPoolForQuota(user, cache, "gpt-5.1", 0.5) {
		t.Fatalf("expected preflight to reject a request above remaining point quota")
	}
	if !plan.CanUsePointPoolForQuota(user, cache, "gpt-5.1", 0.25) {
		t.Fatalf("expected preflight to accept a request within remaining point quota")
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

func TestPlanPointAndWeeklyPoolsCanResetIndependently(t *testing.T) {
	_, cache := openPlanTestCache(t)
	user := planTestUser{id: 99}
	plan := Plan{
		Level:       1,
		Quota:       10,
		WeeklyQuota: 10,
		Items: []PlanItem{
			{
				Id:     "all-models",
				Models: []string{"gpt-5.1"},
			},
		},
	}

	if !plan.ConsumePointPool(user, cache, "gpt-5.1", 3) {
		t.Fatalf("expected point and weekly usage to be consumed")
	}
	if got := plan.GetPointUsage(user, cache); got != 3 {
		t.Fatalf("expected point usage 3, got %f", got)
	}
	if got := plan.GetWeeklyPointUsage(user, cache); got != 3 {
		t.Fatalf("expected weekly usage 3, got %f", got)
	}

	if !plan.ReleasePointPool(user, cache) {
		t.Fatalf("expected point usage reset to succeed")
	}
	if got := plan.GetPointUsage(user, cache); got != 0 {
		t.Fatalf("expected point usage 0 after reset, got %f", got)
	}
	if got := plan.GetWeeklyPointUsage(user, cache); got != 3 {
		t.Fatalf("expected weekly usage to remain 3, got %f", got)
	}

	if !plan.ReleaseWeeklyPool(user, cache) {
		t.Fatalf("expected weekly usage reset to succeed")
	}
	if got := plan.GetWeeklyPointUsage(user, cache); got != 0 {
		t.Fatalf("expected weekly usage 0 after reset, got %f", got)
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
