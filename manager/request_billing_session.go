package manager

import (
	"chat/auth"
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"fmt"
	"sync"

	"github.com/go-redis/redis/v8"
)

const defaultPreConsumedOutputTokens = 1000

type requestBillingSession struct {
	db                  *sql.DB
	cache               *redis.Client
	user                *auth.User
	model               string
	plan                channel.Plan
	pointPool           bool
	legacyPlan          bool
	allowWalletFallback bool

	reservedPlan   float32
	reservedWallet float32
	usesPlan       bool
	closed         bool
	mu             sync.Mutex
}

func estimateRequestBillingReservation(buffer *utils.Buffer, maxTokens *int) float32 {
	if buffer == nil || buffer.GetCharge() == nil || !buffer.GetCharge().IsBilling() {
		return 0
	}

	charge := buffer.GetCharge()
	if charge.IsBillingType(globals.TokenBilling) {
		outputTokens := defaultPreConsumedOutputTokens
		if maxTokens != nil && *maxTokens > 0 {
			outputTokens = *maxTokens
		}
		return utils.CountInputQuota(charge, buffer.CountInputToken()) +
			utils.CountOutputToken(charge, outputTokens)
	}

	reservation := charge.GetLimit()
	if current := buffer.GetQuota(); current > reservation {
		reservation = current
	}
	return reservation
}

func newRequestBillingSession(
	db *sql.DB,
	cache *redis.Client,
	user *auth.User,
	model string,
	buffer *utils.Buffer,
	usePlan bool,
	maxTokens *int,
) (*requestBillingSession, error) {
	session := &requestBillingSession{
		db:    db,
		cache: cache,
		user:  user,
		model: model,
	}
	if user == nil || buffer == nil || buffer.GetCharge() == nil || !buffer.GetCharge().IsBilling() {
		return session, nil
	}

	reservation := estimateRequestBillingReservation(buffer, maxTokens)
	if usePlan {
		session.plan = user.GetPlan(db)
		session.usesPlan = true
		if !session.plan.HasPointPool() {
			// Legacy subscriptions are counted per request during preflight.
			session.legacyPlan = true
			return session, nil
		}

		session.pointPool = true
		session.allowWalletFallback = user.AllowSubscriptionQuotaFallback(db)
		session.reservedPlan = session.plan.ConsumeAvailablePointPool(user, cache, model, reservation)
		remaining := reservation - session.reservedPlan
		if remaining <= realtimeQuotaEpsilon {
			return session, nil
		}

		if !session.allowWalletFallback || !user.UseQuota(db, remaining) {
			if session.reservedPlan > 0 {
				session.plan.RefundPointPool(user, cache, model, session.reservedPlan)
				session.reservedPlan = 0
			}
			return nil, fmt.Errorf(
				auth.ErrEstimatedCost,
				model,
				fmt.Sprintf("%.4f", reservation),
				fmt.Sprintf("%.4f", user.GetQuota(db)),
			)
		}

		session.reservedWallet = remaining
		session.usesPlan = session.reservedPlan > realtimeQuotaEpsilon
		return session, nil
	}

	if reservation > 0 && !user.UseQuota(db, reservation) {
		return nil, fmt.Errorf(
			auth.ErrEstimatedCost,
			model,
			fmt.Sprintf("%.4f", reservation),
			fmt.Sprintf("%.4f", user.GetQuota(db)),
		)
	}
	session.reservedWallet = reservation
	return session, nil
}

func (s *requestBillingSession) UsesPlan() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usesPlan
}

func (s *requestBillingSession) SettleBuffer(buffer *utils.Buffer, requestErr error) float32 {
	if s == nil || buffer == nil || buffer.IsEmpty() {
		if s != nil {
			s.Refund()
		}
		return 0
	}

	actualQuota := buffer.GetRecordQuota()
	if requestErr != nil {
		globals.Warn(fmt.Sprintf(
			"charging visible partial response after request error (model: %s): %s",
			buffer.GetModel(),
			requestErr.Error(),
		))
	}
	s.settle(actualQuota)
	return actualQuota
}

func (s *requestBillingSession) settle(actualQuota float32) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}

	if s.legacyPlan || s.user == nil {
		s.closed = true
		return
	}

	if s.pointPool {
		s.settlePointPoolLocked(actualQuota)
		s.closed = true
		return
	}

	s.settleWalletLocked(actualQuota)
	s.closed = true
}

func (s *requestBillingSession) settleWalletLocked(actualQuota float32) {
	delta := actualQuota - s.reservedWallet
	if delta > realtimeQuotaEpsilon {
		if !s.user.ChargeQuota(s.db, delta) {
			globals.Warn(fmt.Sprintf(
				"user quota only partially covered %.4f settlement quota; balance has been drained without creating debt (user: %s)",
				delta,
				s.user.Username,
			))
		}
		return
	}
	if delta < -realtimeQuotaEpsilon && !s.user.RefundQuotaUsage(s.db, -delta) {
		globals.Warn(fmt.Sprintf("failed to refund %.4f reserved user quota (user: %s)", -delta, s.user.Username))
	}
}

func (s *requestBillingSession) settlePointPoolLocked(actualQuota float32) {
	reservedTotal := s.reservedPlan + s.reservedWallet
	if actualQuota <= reservedTotal+realtimeQuotaEpsilon {
		planTarget := actualQuota
		if planTarget > s.reservedPlan {
			planTarget = s.reservedPlan
		}
		walletTarget := actualQuota - planTarget
		if walletTarget < 0 {
			walletTarget = 0
		}

		if refund := s.reservedPlan - planTarget; refund > realtimeQuotaEpsilon &&
			!s.plan.RefundPointPool(s.user, s.cache, s.model, refund) {
			globals.Warn(fmt.Sprintf("failed to refund %.4f reserved subscription quota (model: %s)", refund, s.model))
		}
		if refund := s.reservedWallet - walletTarget; refund > realtimeQuotaEpsilon &&
			!s.user.RefundQuotaUsage(s.db, refund) {
			globals.Warn(fmt.Sprintf("failed to refund %.4f reserved user quota (user: %s)", refund, s.user.Username))
		}
		return
	}

	extra := actualQuota - reservedTotal
	planExtra := s.plan.ConsumeAvailablePointPool(s.user, s.cache, s.model, extra)
	remaining := extra - planExtra
	if remaining <= realtimeQuotaEpsilon {
		return
	}
	if s.allowWalletFallback {
		if !s.user.ChargeQuota(s.db, remaining) {
			globals.Warn(fmt.Sprintf(
				"user quota only partially covered %.4f subscription overflow; balance has been drained without creating debt (user: %s)",
				remaining,
				s.user.Username,
			))
		}
		return
	}

	globals.Warn(fmt.Sprintf(
		"subscription usage only covered %.4f/%.4f quota and credit fallback is disabled (model: %s)",
		reservedTotal+planExtra,
		actualQuota,
		s.model,
	))
}

func (s *requestBillingSession) Refund() {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true

	if s.legacyPlan {
		if !auth.RevertSubscriptionUsage(s.db, s.cache, s.user, s.model) {
			globals.Warn(fmt.Sprintf("failed to revert subscription request usage (model: %s)", s.model))
		}
		return
	}
	if s.reservedPlan > 0 && !s.plan.RefundPointPool(s.user, s.cache, s.model, s.reservedPlan) {
		globals.Warn(fmt.Sprintf("failed to refund %.4f reserved subscription quota (model: %s)", s.reservedPlan, s.model))
	}
	if s.reservedWallet > 0 && !s.user.RefundQuotaUsage(s.db, s.reservedWallet) {
		globals.Warn(fmt.Sprintf("failed to refund %.4f reserved user quota (user: %s)", s.reservedWallet, s.user.Username))
	}
}
