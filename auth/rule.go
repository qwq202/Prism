package auth

import (
	"chat/channel"
	"database/sql"
	"fmt"
	"math"

	"chat/globals"
	"chat/utils"

	"github.com/go-redis/redis/v8"
)

const (
	ErrNotAuthenticated  = "not authenticated error (model: %s)"
	ErrNotSetPrice       = "the price of the model is not set (model: %s)"
	ErrNotEnoughQuota    = "user quota is not enough error (model: %s, minimum quota: %s, your quota: %s)"
	ErrEstimatedCost     = "estimated cost exceeds user quota (model: %s, estimated cost: %s, your quota: %s)"
	ErrSubscriptionQuota = "subscription quota is not enough error (model: %s, remaining quota: %s, minimum quota: %s, remaining percent: %s, minimum percent: %s, credit fallback: disabled)"
)

func formatQuotaValue(value float32) string {
	abs := math.Abs(float64(value))

	switch {
	case abs == 0:
		return "0.0000"
	case abs < 0.0001:
		return "<0.0001"
	default:
		return fmt.Sprintf("%.4f", value)
	}
}

func formatPercentValue(value float32) string {
	if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
		return "0.00"
	}
	abs := math.Abs(float64(value))
	if abs > 0 && abs < 0.01 {
		return "<0.01"
	}
	return fmt.Sprintf("%.2f", value)
}

type subscriptionPointWindowSummary struct {
	RemainingQuota   float32
	RemainingPercent float32
	MinimumPercent   float32
}

func summarizeSubscriptionPointWindow(plan channel.Plan, user *User, cache *redis.Client, minimumQuota float32) subscriptionPointWindowSummary {
	if !plan.HasPointPool() {
		return subscriptionPointWindowSummary{}
	}
	if plan.IsPointPoolInfinity() {
		return subscriptionPointWindowSummary{
			RemainingQuota:   float32(math.MaxFloat32),
			RemainingPercent: 100,
			MinimumPercent:   0,
		}
	}

	total := plan.Quota
	if total <= 0 {
		return subscriptionPointWindowSummary{}
	}

	remainingQuota := plan.Quota - plan.GetPointUsage(user, cache)
	if remainingQuota < 0 {
		remainingQuota = 0
	}
	return subscriptionPointWindowSummary{
		RemainingQuota:   remainingQuota,
		RemainingPercent: remainingQuota / total * 100,
		MinimumPercent:   minimumQuota / total * 100,
	}
}

// CanEnableModel returns whether the model can be enabled (without subscription)
func estimateImageBillingCost(charge *channel.Charge, messages []globals.Message, responseFormat interface{}) float32 {
	if charge == nil || !charge.IsBillingType(globals.ImageBilling) {
		return 0
	}

	outputImages := 1
	if config := charge.GetImageChargeConfig(); config.OutputCount > 1 {
		outputImages = config.OutputCount
	}

	cost := charge.EstimateImageQuota(
		utils.CountReferenceImagesFromMessages(messages),
		responseFormat,
		outputImages,
	)
	if cost <= 0 {
		return charge.GetLimit()
	}
	return cost
}

func CanEnableModel(db *sql.DB, user *User, model string, messages []globals.Message) error {
	return CanEnableModelForRequest(db, user, model, messages, nil)
}

// CanEnableModelForRequest returns whether the model can be enabled for a
// specific request. responseFormat is used by non-token billing modes such as
// image generation where size or quality changes the estimated cost.
func CanEnableModelForRequest(db *sql.DB, user *User, model string, messages []globals.Message, responseFormat interface{}) error {
	isAuth := user != nil
	isAdmin := isAuth && user.IsAdmin(db)

	charge := channel.ChargeInstance.GetCharge(model)

	if charge.IsUnsetType() && !isAdmin {
		return fmt.Errorf(ErrNotSetPrice, model)
	}

	if !charge.IsBilling() {
		// return if is the user is authenticated or anonymous is allowed for this model
		if charge.SupportAnonymous() || isAuth {
			return nil
		}

		return fmt.Errorf(ErrNotAuthenticated, model)
	}

	if !isAuth {
		return fmt.Errorf(ErrNotAuthenticated, model)
	}

	// Get user's current quota
	quota := user.GetQuota(db)
	minimumCost := charge.GetLimit()
	if minimumCost > 0 && quota < minimumCost {
		return fmt.Errorf(
			ErrNotEnoughQuota,
			model,
			formatQuotaValue(minimumCost),
			formatQuotaValue(quota),
		)
	}

	if charge.IsBillingType(globals.ImageBilling) {
		estimatedCost := estimateImageBillingCost(charge, messages, responseFormat)
		if estimatedCost > 0 && quota < estimatedCost {
			return fmt.Errorf(
				ErrEstimatedCost,
				model,
				formatQuotaValue(estimatedCost),
				formatQuotaValue(quota),
			)
		}
		return nil
	}

	// Calculate estimated input cost
	inputTokens := utils.NumTokensFromMessages(messages, model, false)
	estimatedInputCost := float32(inputTokens) / 1000 * charge.GetInput()

	if quota < estimatedInputCost {
		return fmt.Errorf(
			ErrEstimatedCost,
			model,
			formatQuotaValue(estimatedInputCost),
			formatQuotaValue(quota),
		)
	}

	return nil
}

func CanEnableModelWithSubscription(db *sql.DB, cache *redis.Client, user *User, model string, messages []globals.Message) (canEnable error, usePlan bool) {
	return CanEnableModelWithSubscriptionForRequest(db, cache, user, model, messages, nil)
}

func CanEnableModelWithSubscriptionForRequest(db *sql.DB, cache *redis.Client, user *User, model string, messages []globals.Message, responseFormat interface{}) (canEnable error, usePlan bool) {
	// use subscription quota first
	charge := channel.ChargeInstance.GetCharge(model)
	minimumCost := charge.GetLimit()
	inputTokens := utils.NumTokensFromMessages(messages, model, false)
	estimatedInputCost := float32(inputTokens) / 1000 * charge.GetInput()
	if charge.IsBillingType(globals.ImageBilling) {
		estimatedInputCost = estimateImageBillingCost(charge, messages, responseFormat)
	}
	subscriptionPreflightCost := minimumCost
	if estimatedInputCost > subscriptionPreflightCost {
		subscriptionPreflightCost = estimatedInputCost
	}
	if user != nil && HandleSubscriptionUsage(db, cache, user, model, subscriptionPreflightCost) {
		return nil, true
	}
	if !disableSubscription() && user != nil && user.IsSubscribe(db) {
		plan := user.GetPlan(db)
		if plan.IncludesModel(model) && !user.AllowSubscriptionQuotaFallback(db) {
			summary := summarizeSubscriptionPointWindow(plan, user, cache, subscriptionPreflightCost)
			return fmt.Errorf(
				ErrSubscriptionQuota,
				model,
				formatQuotaValue(summary.RemainingQuota),
				formatQuotaValue(subscriptionPreflightCost),
				formatPercentValue(summary.RemainingPercent),
				formatPercentValue(summary.MinimumPercent),
			), false
		}
	}
	return CanEnableModelForRequest(db, user, model, messages, responseFormat), false
}
