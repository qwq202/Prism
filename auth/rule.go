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
	ErrNotAuthenticated = "not authenticated error (model: %s)"
	ErrNotSetPrice      = "the price of the model is not set (model: %s)"
	ErrNotEnoughQuota   = "user quota is not enough error (model: %s, minimum quota: %s, your quota: %s)"
	ErrEstimatedCost    = "estimated cost exceeds user quota (model: %s, estimated cost: %s, your quota: %s)"
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

// CanEnableModel returns whether the model can be enabled (without subscription)
func CanEnableModel(db *sql.DB, user *User, model string, messages []globals.Message) error {
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
	// use subscription quota first
	minimumCost := channel.ChargeInstance.GetCharge(model).GetLimit()
	if user != nil && HandleSubscriptionUsage(db, cache, user, model, minimumCost) {
		return nil, true
	}
	return CanEnableModel(db, user, model, messages), false
}
