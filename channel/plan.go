package channel

import (
	"chat/globals"
	"chat/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
)

type PlanManager struct {
	Enabled bool   `json:"enabled" mapstructure:"enabled"`
	Plans   []Plan `json:"plans" mapstructure:"plans"`
}

type Plan struct {
	Level         int        `json:"level" mapstructure:"level"`
	Price         float32    `json:"price" mapstructure:"price"`
	Sellable      *bool      `json:"sellable,omitempty" mapstructure:"sellable"`
	Quota         float32    `json:"quota,omitempty" mapstructure:"quota"`
	ResetInterval int64      `json:"reset_interval,omitempty" mapstructure:"resetinterval"`
	WeeklyQuota   float32    `json:"weekly_quota,omitempty" mapstructure:"weeklyquota"`
	Items         []PlanItem `json:"items" mapstructure:"items"`
}

type PlanItem struct {
	Id     string   `json:"id" mapstructure:"id"`
	Name   string   `json:"name" mapstructure:"name"`
	Icon   string   `json:"icon" mapstructure:"icon"`
	Value  int64    `json:"value" mapstructure:"value"`
	Models []string `json:"models" mapstructure:"models"`
}

type Usage struct {
	Used          float32 `json:"used" mapstructure:"used"`
	Total         float32 `json:"total" mapstructure:"total"`
	Unit          string  `json:"unit,omitempty" mapstructure:"unit"`
	ResetInterval int64   `json:"reset_interval,omitempty" mapstructure:"reset_interval"`
	ResetAt       string  `json:"reset_at,omitempty" mapstructure:"reset_at"`
}
type UsageMap map[string]Usage

var planExp int64 = 0

var errPointPoolExceeded = errors.New("subscription point pool exceeded")

const (
	PlanItemUnitTimes      = "times"
	PlanItemUnitPoints     = "points"
	PlanSharedPointsItemID = "plan_points"
	PlanWeeklyPointsItemID = "plan_points_weekly"
	pointResetInterval     = int64(5 * 60 * 60)      // 18000 seconds
	weeklyResetInterval    = int64(7 * 24 * 60 * 60) // 604800 seconds
)

type subscriptionPointUsageWindow struct {
	key           string
	limit         float32
	resetInterval int64
}

var savePlanConfig = func(manager *PlanManager) error {
	return manager.SaveConfig()
}

func NewPlanManager() *PlanManager {
	manager := &PlanManager{}
	if err := viper.UnmarshalKey("subscription", manager); err != nil {
		panic(err)
	}

	return manager
}

func (c *PlanManager) SaveConfig() error {
	return utils.SaveConfig("subscription", c)
}

func validatePlanConfigModels(data *PlanManager) error {
	if ConduitInstance == nil {
		return nil
	}

	availableModels := make(map[string]struct{}, len(ConduitInstance.GetModels()))
	for _, model := range ConduitInstance.GetModels() {
		availableModels[model] = struct{}{}
	}

	for _, plan := range data.Plans {
		for _, item := range plan.Items {
			for _, model := range item.Models {
				if _, ok := availableModels[model]; ok {
					continue
				}

				return fmt.Errorf("subscription item %q contains unavailable model %q", item.Id, model)
			}
		}
	}

	return nil
}

func (c *PlanManager) UpdateConfig(data *PlanManager) error {
	if err := validatePlanConfigModels(data); err != nil {
		return err
	}

	next := *data
	if err := savePlanConfig(&next); err != nil {
		return err
	}

	c.Enabled = next.Enabled
	c.Plans = next.Plans
	return nil
}

func (c *PlanManager) GetPlan(level int) Plan {
	for _, plan := range c.Plans {
		if plan.Level == level {
			return plan
		}
	}
	return Plan{}
}

func (c *PlanManager) GetPlans() []Plan {
	if c.Enabled {
		return c.Plans
	}

	return []Plan{}
}

func (c *PlanManager) GetRawPlans() []Plan {
	return c.Plans
}

func (c *PlanManager) IsEnabled() bool {
	return c.Enabled
}

func (p *Plan) IsSellable() bool {
	return p.Level > 0 && (p.Sellable == nil || *p.Sellable)
}

func getOffsetFormat(offset time.Time, usage int64) string {
	return fmt.Sprintf("%s/%d", offset.Format("2006-01-02:15:04:05"), usage)
}

func getFloatOffsetFormat(offset time.Time, usage float32) string {
	return fmt.Sprintf("%s/%.6f", offset.Format("2006-01-02:15:04:05"), usage)
}

func advanceUsageOffset(offset time.Time, now time.Time, resetInterval int64) (time.Time, bool) {
	if resetInterval > 0 {
		interval := time.Duration(resetInterval) * time.Second
		next := offset.Add(interval)
		if next.After(now) {
			return offset, false
		}

		elapsed := now.Sub(offset)
		steps := int64(elapsed / interval)
		if steps < 1 {
			steps = 1
		}
		return offset.Add(time.Duration(steps) * interval), true
	}

	next := offset.AddDate(0, 1, 0)
	if next.Before(now) {
		for offset.AddDate(0, 1, 0).Before(now) {
			offset = offset.AddDate(0, 1, 0)
		}
		return offset, true
	}

	return offset, false
}

func parseSubscriptionUsageValue(value string, resetInterval int64, now time.Time) (usage int64, offset time.Time) {
	offset = now

	seg := strings.Split(value, "/")
	if len(seg) != 2 {
		return 0, offset
	}

	date, err := time.ParseInLocation("2006-01-02:15:04:05", seg[0], time.Local)
	usage = utils.ParseInt64(seg[1])
	if err != nil {
		return 0, offset
	}

	if nextOffset, reset := advanceUsageOffset(date, now, resetInterval); reset {
		return 0, nextOffset
	} else {
		return usage, nextOffset
	}
}

func parseSubscriptionPointUsageValue(value string, resetInterval int64, now time.Time) (usage float32, offset time.Time) {
	offset = now

	seg := strings.Split(value, "/")
	if len(seg) != 2 {
		return 0, offset
	}

	date, err := time.ParseInLocation("2006-01-02:15:04:05", seg[0], time.Local)
	usage = utils.ParseFloat32(seg[1])
	if err != nil {
		return 0, offset
	}

	if nextOffset, reset := advanceUsageOffset(date, now, resetInterval); reset {
		return 0, nextOffset
	} else {
		return usage, nextOffset
	}
}

func getSubscriptionUsage(cache *redis.Client, user globals.AuthLike, t string, resetInterval int64) (usage int64, offset time.Time) {
	// example cache value: 2021-09-01:19:00:00/100
	// if date is longer than the configured reset interval, reset usage

	key := globals.GetSubscriptionLimitFormat(t, user.HitID())
	v, err := utils.GetCache(cache, key)
	if err != nil && errors.Is(err, redis.Nil) {
		v = ""
	}

	usage, offset = parseSubscriptionUsageValue(v, resetInterval, time.Now())

	// set new cache value
	_ = utils.SetCache(cache, key, getOffsetFormat(offset, usage), planExp)

	return
}

func getSubscriptionPointUsage(cache *redis.Client, user globals.AuthLike, t string, resetInterval int64) (usage float32, offset time.Time) {
	key := globals.GetSubscriptionLimitFormat(t, user.HitID())
	v, err := utils.GetCache(cache, key)
	if err != nil && errors.Is(err, redis.Nil) {
		v = ""
	}

	usage, offset = parseSubscriptionPointUsageValue(v, resetInterval, time.Now())

	return
}

func consumeSubscriptionPointUsage(cache *redis.Client, key string, limit float32, resetInterval int64, amount float32) bool {
	return consumeSubscriptionPointUsageWindows(cache, []subscriptionPointUsageWindow{
		{key: key, limit: limit, resetInterval: resetInterval},
	}, amount)
}

func consumeSubscriptionPointUsageWindows(cache *redis.Client, windows []subscriptionPointUsageWindow, amount float32) bool {
	if amount <= 0 {
		return true
	}
	if len(windows) == 0 {
		return true
	}

	ctx := context.Background()
	keys := make([]string, 0, len(windows))
	for _, window := range windows {
		keys = append(keys, window.key)
	}

	for attempts := 0; attempts < 32; attempts++ {
		err := cache.Watch(ctx, func(tx *redis.Tx) error {
			values := make([]string, len(windows))
			offsets := make([]time.Time, len(windows))
			usages := make([]float32, len(windows))

			for i, window := range windows {
				value, err := tx.Get(ctx, window.key).Result()
				if err != nil && !errors.Is(err, redis.Nil) {
					return err
				}
				if errors.Is(err, redis.Nil) {
					value = ""
				}

				used, offset := parseSubscriptionPointUsageValue(value, window.resetInterval, time.Now())
				used += amount
				if used > window.limit {
					return errPointPoolExceeded
				}
				values[i] = getFloatOffsetFormat(offset, used)
				offsets[i] = offset
				usages[i] = used
			}

			_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				for i, window := range windows {
					if values[i] == "" {
						values[i] = getFloatOffsetFormat(offsets[i], usages[i])
					}
					pipe.Set(ctx, window.key, values[i], time.Duration(planExp)*time.Second)
				}
				return nil
			})
			return err
		}, keys...)

		if err == nil {
			return true
		}
		if errors.Is(err, errPointPoolExceeded) {
			return false
		}
		if errors.Is(err, redis.TxFailedErr) {
			continue
		}
		return false
	}

	return false
}

func consumeAvailableSubscriptionPointUsageWindows(cache *redis.Client, windows []subscriptionPointUsageWindow, amount float32) float32 {
	if amount <= 0 {
		return 0
	}
	if len(windows) == 0 {
		return amount
	}

	ctx := context.Background()
	keys := make([]string, 0, len(windows))
	for _, window := range windows {
		keys = append(keys, window.key)
	}

	for attempts := 0; attempts < 32; attempts++ {
		var consumed float32
		err := cache.Watch(ctx, func(tx *redis.Tx) error {
			offsets := make([]time.Time, len(windows))
			usages := make([]float32, len(windows))
			available := amount

			for i, window := range windows {
				value, err := tx.Get(ctx, window.key).Result()
				if err != nil && !errors.Is(err, redis.Nil) {
					return err
				}
				if errors.Is(err, redis.Nil) {
					value = ""
				}

				used, offset := parseSubscriptionPointUsageValue(value, window.resetInterval, time.Now())
				remaining := window.limit - used
				if remaining < 0 {
					remaining = 0
				}
				if remaining < available {
					available = remaining
				}
				offsets[i] = offset
				usages[i] = used
			}

			if available <= 0 {
				consumed = 0
				return nil
			}

			consumed = available
			_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				for i, window := range windows {
					pipe.Set(
						ctx,
						window.key,
						getFloatOffsetFormat(offsets[i], usages[i]+consumed),
						time.Duration(planExp)*time.Second,
					)
				}
				return nil
			})
			return err
		}, keys...)

		if err == nil {
			return consumed
		}
		if errors.Is(err, redis.TxFailedErr) {
			continue
		}
		return 0
	}

	return 0
}

func refundSubscriptionPointUsageWindows(cache *redis.Client, windows []subscriptionPointUsageWindow, amount float32) bool {
	if amount <= 0 || len(windows) == 0 {
		return true
	}

	ctx := context.Background()
	keys := make([]string, 0, len(windows))
	for _, window := range windows {
		keys = append(keys, window.key)
	}

	for attempts := 0; attempts < 32; attempts++ {
		err := cache.Watch(ctx, func(tx *redis.Tx) error {
			offsets := make([]time.Time, len(windows))
			usages := make([]float32, len(windows))

			for i, window := range windows {
				value, err := tx.Get(ctx, window.key).Result()
				if err != nil && !errors.Is(err, redis.Nil) {
					return err
				}
				if errors.Is(err, redis.Nil) {
					value = ""
				}

				used, offset := parseSubscriptionPointUsageValue(value, window.resetInterval, time.Now())
				used -= amount
				if used < 0 {
					used = 0
				}
				offsets[i] = offset
				usages[i] = used
			}

			_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				for i, window := range windows {
					pipe.Set(
						ctx,
						window.key,
						getFloatOffsetFormat(offsets[i], usages[i]),
						time.Duration(planExp)*time.Second,
					)
				}
				return nil
			})
			return err
		}, keys...)

		if err == nil {
			return true
		}
		if errors.Is(err, redis.TxFailedErr) {
			continue
		}
		return false
	}

	return false
}

func GetSubscriptionUsage(cache *redis.Client, user globals.AuthLike, t string) (usage int64, offset time.Time) {
	return getSubscriptionUsage(cache, user, t, 0)
}

func getNextResetAt(offset time.Time, resetInterval int64) time.Time {
	if resetInterval > 0 {
		return offset.Add(time.Duration(resetInterval) * time.Second)
	}

	return offset.AddDate(0, 1, 0)
}

func (p *PlanItem) GetResetAt(user globals.AuthLike, cache *redis.Client) time.Time {
	_, offset := getSubscriptionUsage(cache, user, p.Id, 0)
	return getNextResetAt(offset, 0)
}

func (p *Plan) pointUsageKey() string {
	return fmt.Sprintf("%s:%d", PlanSharedPointsItemID, p.Level)
}

func (p *Plan) pointResetInterval() int64 {
	if p.ResetInterval > 0 {
		return p.ResetInterval
	}
	return pointResetInterval
}

func (p *Plan) weeklyUsageKey() string {
	return fmt.Sprintf("%s:%d", PlanWeeklyPointsItemID, p.Level)
}

func (p *Plan) HasPointPool() bool {
	return p.Quota > 0 || p.Quota == -1
}

func (p *Plan) IsPointPoolInfinity() bool {
	return p.Quota == -1
}

func (p *Plan) HasWeeklyPool() bool {
	return p.WeeklyQuota > 0 || p.WeeklyQuota == -1
}

func (p *Plan) IsWeeklyPoolInfinity() bool {
	return p.WeeklyQuota == -1
}

func (p *Plan) IncludesModel(model string) bool {
	if p.HasPointPool() && len(p.Items) == 0 {
		return true
	}

	for _, item := range p.Items {
		if utils.Contains(model, item.Models) {
			return true
		}
	}

	return false
}

func (p *Plan) GetPointResetAt(user globals.AuthLike, cache *redis.Client) time.Time {
	resetInterval := p.pointResetInterval()
	_, offset := getSubscriptionPointUsage(cache, user, p.pointUsageKey(), resetInterval)
	return getNextResetAt(offset, resetInterval)
}

func (p *Plan) GetPointUsage(user globals.AuthLike, cache *redis.Client) float32 {
	usage, _ := getSubscriptionPointUsage(cache, user, p.pointUsageKey(), p.pointResetInterval())
	return usage
}

func (p *Plan) GetPointUsageForm(user globals.AuthLike, cache *redis.Client) Usage {
	resetInterval := p.pointResetInterval()
	used, offset := getSubscriptionPointUsage(cache, user, p.pointUsageKey(), resetInterval)
	return Usage{
		Used:          used,
		Total:         p.Quota,
		Unit:          PlanItemUnitPoints,
		ResetInterval: resetInterval,
		ResetAt:       getNextResetAt(offset, resetInterval).Format(time.RFC3339),
	}
}

func (p *Plan) GetWeeklyPointUsage(user globals.AuthLike, cache *redis.Client) float32 {
	usage, _ := getSubscriptionPointUsage(cache, user, p.weeklyUsageKey(), weeklyResetInterval)
	return usage
}

func (p *Plan) GetWeeklyPointUsageForm(user globals.AuthLike, cache *redis.Client) Usage {
	used, offset := getSubscriptionPointUsage(cache, user, p.weeklyUsageKey(), weeklyResetInterval)
	return Usage{
		Used:          used,
		Total:         p.WeeklyQuota,
		Unit:          PlanItemUnitPoints,
		ResetInterval: weeklyResetInterval,
		ResetAt:       getNextResetAt(offset, weeklyResetInterval).Format(time.RFC3339),
	}
}

func (p *Plan) CanUseWeeklyPool(user globals.AuthLike, cache *redis.Client, model string) bool {
	return p.CanUseWeeklyPoolForQuota(user, cache, model, 0)
}

func (p *Plan) CanUseWeeklyPoolForQuota(user globals.AuthLike, cache *redis.Client, model string, quota float32) bool {
	if !p.HasWeeklyPool() || !p.IncludesModel(model) {
		return false
	}
	if p.IsWeeklyPoolInfinity() {
		return true
	}
	remaining := p.WeeklyQuota - p.GetWeeklyPointUsage(user, cache)
	if quota <= 0 {
		return remaining > 0
	}
	return remaining >= quota
}

func (p *Plan) ConsumeWeeklyPool(user globals.AuthLike, cache *redis.Client, model string, quota float32) bool {
	if !p.HasWeeklyPool() || !p.IncludesModel(model) {
		return false
	}
	if p.IsWeeklyPoolInfinity() {
		return true
	}
	if quota <= 0 {
		return true
	}
	key := globals.GetSubscriptionLimitFormat(p.weeklyUsageKey(), user.HitID())
	return consumeSubscriptionPointUsage(cache, key, p.WeeklyQuota, weeklyResetInterval, quota)
}

func (p *Plan) CanUsePointPool(user globals.AuthLike, cache *redis.Client, model string) bool {
	return p.CanUsePointPoolForQuota(user, cache, model, 0)
}

func (p *Plan) CanUsePointPoolForQuota(user globals.AuthLike, cache *redis.Client, model string, quota float32) bool {
	if !p.HasPointPool() || !p.IncludesModel(model) {
		return false
	}
	if !p.IsPointPoolInfinity() {
		remaining := p.Quota - p.GetPointUsage(user, cache)
		if quota <= 0 {
			if remaining <= 0 {
				return false
			}
		} else if remaining < quota {
			return false
		}
	}
	if p.HasWeeklyPool() && !p.CanUseWeeklyPoolForQuota(user, cache, model, quota) {
		return false
	}
	return true
}

func (p *Plan) PointPoolRemaining(user globals.AuthLike, cache *redis.Client, model string) (float32, bool) {
	if !p.HasPointPool() || !p.IncludesModel(model) {
		return 0, false
	}

	remaining := float32(math.MaxFloat32)
	hasFiniteWindow := false
	if !p.IsPointPoolInfinity() {
		remaining = p.Quota - p.GetPointUsage(user, cache)
		hasFiniteWindow = true
	}
	if p.HasWeeklyPool() && !p.IsWeeklyPoolInfinity() {
		weeklyRemaining := p.WeeklyQuota - p.GetWeeklyPointUsage(user, cache)
		if !hasFiniteWindow || weeklyRemaining < remaining {
			remaining = weeklyRemaining
		}
		hasFiniteWindow = true
	}

	if !hasFiniteWindow {
		return float32(math.MaxFloat32), true
	}
	if remaining < 0 {
		remaining = 0
	}
	return remaining, true
}

func (p *Plan) ConsumePointPool(user globals.AuthLike, cache *redis.Client, model string, quota float32) bool {
	if !p.HasPointPool() || !p.IncludesModel(model) {
		return false
	}
	if quota <= 0 {
		return true
	}

	windows := make([]subscriptionPointUsageWindow, 0, 2)
	if !p.IsPointPoolInfinity() {
		windows = append(windows, subscriptionPointUsageWindow{
			key:           globals.GetSubscriptionLimitFormat(p.pointUsageKey(), user.HitID()),
			limit:         p.Quota,
			resetInterval: p.pointResetInterval(),
		})
	}
	if p.HasWeeklyPool() && !p.IsWeeklyPoolInfinity() {
		windows = append(windows, subscriptionPointUsageWindow{
			key:           globals.GetSubscriptionLimitFormat(p.weeklyUsageKey(), user.HitID()),
			limit:         p.WeeklyQuota,
			resetInterval: weeklyResetInterval,
		})
	}

	return consumeSubscriptionPointUsageWindows(cache, windows, quota)
}

func (p *Plan) ConsumeAvailablePointPool(user globals.AuthLike, cache *redis.Client, model string, quota float32) float32 {
	if !p.HasPointPool() || !p.IncludesModel(model) {
		return 0
	}
	if quota <= 0 {
		return 0
	}

	windows := make([]subscriptionPointUsageWindow, 0, 2)
	if !p.IsPointPoolInfinity() {
		windows = append(windows, subscriptionPointUsageWindow{
			key:           globals.GetSubscriptionLimitFormat(p.pointUsageKey(), user.HitID()),
			limit:         p.Quota,
			resetInterval: p.pointResetInterval(),
		})
	}
	if p.HasWeeklyPool() && !p.IsWeeklyPoolInfinity() {
		windows = append(windows, subscriptionPointUsageWindow{
			key:           globals.GetSubscriptionLimitFormat(p.weeklyUsageKey(), user.HitID()),
			limit:         p.WeeklyQuota,
			resetInterval: weeklyResetInterval,
		})
	}

	return consumeAvailableSubscriptionPointUsageWindows(cache, windows, quota)
}

func (p *Plan) RefundPointPool(user globals.AuthLike, cache *redis.Client, model string, quota float32) bool {
	if !p.HasPointPool() || !p.IncludesModel(model) {
		return false
	}
	if quota <= 0 {
		return true
	}

	windows := make([]subscriptionPointUsageWindow, 0, 2)
	if !p.IsPointPoolInfinity() {
		windows = append(windows, subscriptionPointUsageWindow{
			key:           globals.GetSubscriptionLimitFormat(p.pointUsageKey(), user.HitID()),
			limit:         p.Quota,
			resetInterval: p.pointResetInterval(),
		})
	}
	if p.HasWeeklyPool() && !p.IsWeeklyPoolInfinity() {
		windows = append(windows, subscriptionPointUsageWindow{
			key:           globals.GetSubscriptionLimitFormat(p.weeklyUsageKey(), user.HitID()),
			limit:         p.WeeklyQuota,
			resetInterval: weeklyResetInterval,
		})
	}

	return refundSubscriptionPointUsageWindows(cache, windows, quota)
}

func increaseSubscriptionUsage(cache *redis.Client, user globals.AuthLike, t string, limit int64, resetInterval int64, amount int64) bool {
	key := globals.GetSubscriptionLimitFormat(t, user.HitID())
	usage, offset := getSubscriptionUsage(cache, user, t, resetInterval)

	if amount <= 0 {
		amount = 1
	}

	usage += amount
	if usage > limit {
		return false
	}

	// set new cache value
	err := utils.SetCache(cache, key, getOffsetFormat(offset, usage), planExp)
	return err == nil
}

func IncreaseSubscriptionUsage(cache *redis.Client, user globals.AuthLike, t string, limit int64) bool {
	return increaseSubscriptionUsage(cache, user, t, limit, 0, 1)
}

func decreaseSubscriptionUsage(cache *redis.Client, user globals.AuthLike, t string, resetInterval int64, amount int64) bool {
	key := globals.GetSubscriptionLimitFormat(t, user.HitID())
	usage, offset := getSubscriptionUsage(cache, user, t, resetInterval)

	if amount <= 0 {
		amount = 1
	}

	usage -= amount
	if usage < 0 {
		return true
	}

	// set new cache value
	err := utils.SetCache(cache, key, getOffsetFormat(offset, usage), planExp)
	return err == nil
}

func DecreaseSubscriptionUsage(cache *redis.Client, user globals.AuthLike, t string) bool {
	return decreaseSubscriptionUsage(cache, user, t, 0, 1)
}

func releaseSubscriptionUsage(cache *redis.Client, user globals.AuthLike, t string, resetInterval int64) bool {
	key := globals.GetSubscriptionLimitFormat(t, user.HitID())
	_, offset := getSubscriptionUsage(cache, user, t, resetInterval)

	// set new cache value
	err := utils.SetCache(cache, key, getOffsetFormat(offset, 0), planExp)
	return err == nil
}

func ReleaseSubscriptionUsage(cache *redis.Client, user globals.AuthLike, t string) bool {
	return releaseSubscriptionUsage(cache, user, t, 0)
}

func (p *Plan) GetUsage(user globals.AuthLike, db *sql.DB, cache *redis.Client) UsageMap {
	if p.HasPointPool() {
		m := UsageMap{
			PlanSharedPointsItemID: p.GetPointUsageForm(user, cache),
		}
		if p.HasWeeklyPool() {
			m[PlanWeeklyPointsItemID] = p.GetWeeklyPointUsageForm(user, cache)
		}
		return m
	}

	return utils.EachObject[PlanItem, Usage](p.Items, func(usage PlanItem) (string, Usage) {
		return usage.Id, usage.GetUsageForm(user, db, cache)
	})
}

func (p *PlanItem) GetUsage(user globals.AuthLike, db *sql.DB, cache *redis.Client) int64 {
	// preflight check
	user.GetID(db)
	usage, _ := getSubscriptionUsage(cache, user, p.Id, 0)
	return usage
}

func (p *PlanItem) ResetUsage(user globals.AuthLike, cache *redis.Client) bool {
	key := globals.GetSubscriptionLimitFormat(p.Id, user.HitID())
	_, offset := getSubscriptionUsage(cache, user, p.Id, 0)

	err := utils.SetCache(cache, key, getOffsetFormat(offset, 0), planExp)
	return err == nil
}

func (p *PlanItem) CreateUsage(user globals.AuthLike, cache *redis.Client) bool {
	key := globals.GetSubscriptionLimitFormat(p.Id, user.HitID())

	err := utils.SetCache(cache, key, getOffsetFormat(time.Now(), 0), planExp)
	return err == nil
}

func (p *PlanItem) GetUsageForm(user globals.AuthLike, db *sql.DB, cache *redis.Client) Usage {
	used, offset := getSubscriptionUsage(cache, user, p.Id, 0)
	return Usage{
		Used:          float32(used),
		Total:         float32(p.Value),
		Unit:          PlanItemUnitTimes,
		ResetInterval: 0,
		ResetAt:       getNextResetAt(offset, 0).Format(time.RFC3339),
	}
}

func (p *PlanItem) IsInfinity() bool {
	return p.Value == -1
}

func (p *PlanItem) IsExceeded(user globals.AuthLike, db *sql.DB, cache *redis.Client) bool {
	return p.IsInfinity() || p.GetUsage(user, db, cache) < p.Value
}

func (p *PlanItem) Increase(user globals.AuthLike, cache *redis.Client) bool {
	state := increaseSubscriptionUsage(cache, user, p.Id, p.Value, 0, 1)
	return state || p.IsInfinity()
}

func (p *PlanItem) Decrease(user globals.AuthLike, cache *redis.Client) bool {
	if p.Value == -1 {
		return true
	}
	return decreaseSubscriptionUsage(cache, user, p.Id, 0, 1)
}

func (p *PlanItem) Release(user globals.AuthLike, cache *redis.Client) bool {
	return releaseSubscriptionUsage(cache, user, p.Id, 0)
}

func (p *Plan) IncreaseUsage(user globals.AuthLike, cache *redis.Client, model string, minimumQuota float32) bool {
	if p.HasPointPool() {
		return p.CanUsePointPoolForQuota(user, cache, model, minimumQuota)
	}

	for _, usage := range p.Items {
		if utils.Contains(model, usage.Models) {
			return usage.Increase(user, cache)
		}
	}

	return false
}

func (p *Plan) DecreaseUsage(user globals.AuthLike, cache *redis.Client, model string) bool {
	if p.HasPointPool() {
		return true
	}

	for _, usage := range p.Items {
		if utils.Contains(model, usage.Models) {
			return usage.Decrease(user, cache)
		}
	}

	return false
}

func (p *Plan) ReleaseUsage(user globals.AuthLike, cache *redis.Client, model string) bool {
	if p.HasPointPool() {
		return true
	}

	for _, usage := range p.Items {
		if utils.Contains(model, usage.Models) {
			return usage.Release(user, cache)
		}
	}

	return false
}

func releasePointUsage(cache *redis.Client, user globals.AuthLike, key string, usageKey string, resetInterval int64) bool {
	_, offset := getSubscriptionPointUsage(cache, user, usageKey, resetInterval)
	return utils.SetCache(cache, key, getFloatOffsetFormat(offset, 0), planExp) == nil
}

func (p *Plan) ReleasePointPool(user globals.AuthLike, cache *redis.Client) bool {
	if !p.HasPointPool() {
		return false
	}

	key := globals.GetSubscriptionLimitFormat(p.pointUsageKey(), user.HitID())
	return releasePointUsage(cache, user, key, p.pointUsageKey(), p.pointResetInterval())
}

func (p *Plan) ReleaseWeeklyPool(user globals.AuthLike, cache *redis.Client) bool {
	if !p.HasWeeklyPool() {
		return false
	}

	key := globals.GetSubscriptionLimitFormat(p.weeklyUsageKey(), user.HitID())
	return releasePointUsage(cache, user, key, p.weeklyUsageKey(), weeklyResetInterval)
}

func (p *Plan) ReleaseAll(user globals.AuthLike, cache *redis.Client) bool {
	if p.HasPointPool() && !p.ReleasePointPool(user, cache) {
		return false
	}

	if p.HasWeeklyPool() && !p.ReleaseWeeklyPool(user, cache) {
		return false
	}

	for _, usage := range p.Items {
		if !usage.Release(user, cache) {
			return false
		}
	}

	return true
}

func IsValidPlan(level int) bool {
	return utils.InRange(level, 1, 3)
}
