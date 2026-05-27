package admin

import (
	"chat/auth"
	"chat/channel"
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
)

// AuthLike is to solve the problem of import cycle
type AuthLike struct {
	ID int64 `json:"id"`
}

type userAdminState struct {
	Exists   bool
	IsAdmin  bool
	IsBanned bool
}

type userListFilter struct {
	Plan  string
	Admin string
	Ban   string
	Sort  string
}

func (a *AuthLike) GetID(_ *sql.DB) int64 {
	return a.ID
}

func (a *AuthLike) HitID() int64 {
	return a.ID
}

func normalizeUserListFilter(filter userListFilter) userListFilter {
	normalizeOption := func(value string, allowed map[string]bool, fallback string) string {
		value = strings.TrimSpace(value)
		if allowed[value] {
			return value
		}
		return fallback
	}

	return userListFilter{
		Plan:  normalizeOption(filter.Plan, map[string]bool{"all": true, "yes": true, "no": true}, "all"),
		Admin: normalizeOption(filter.Admin, map[string]bool{"all": true, "yes": true, "no": true}, "all"),
		Ban:   normalizeOption(filter.Ban, map[string]bool{"all": true, "yes": true, "no": true}, "all"),
		Sort: normalizeOption(filter.Sort, map[string]bool{
			"id-asc": true, "id-desc": true,
			"quota-asc": true, "quota-desc": true,
			"used-quota-asc": true, "used-quota-desc": true,
			"plan-asc": true, "plan-desc": true,
		}, "id-asc"),
	}
}

func buildUserListWhere(search string, filter userListFilter) (string, []interface{}) {
	filter = normalizeUserListFilter(filter)
	now := time.Now().Format("2006-01-02 15:04:05")
	conditions := []string{"auth.username LIKE ?"}
	args := []interface{}{"%" + search + "%"}

	switch filter.Plan {
	case "yes":
		conditions = append(conditions, "subscription.level > 0 AND subscription.expired_at > ?")
		args = append(args, now)
	case "no":
		conditions = append(conditions, "(subscription.user_id IS NULL OR subscription.level IS NULL OR subscription.level <= 0 OR subscription.expired_at IS NULL OR subscription.expired_at <= ?)")
		args = append(args, now)
	}

	switch filter.Admin {
	case "yes":
		conditions = append(conditions, "auth.is_admin = ?")
		args = append(args, true)
	case "no":
		conditions = append(conditions, "(auth.is_admin = ? OR auth.is_admin IS NULL)")
		args = append(args, false)
	}

	switch filter.Ban {
	case "yes":
		conditions = append(conditions, "auth.is_banned = ?")
		args = append(args, true)
	case "no":
		conditions = append(conditions, "(auth.is_banned = ? OR auth.is_banned IS NULL)")
		args = append(args, false)
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

func getUserListOrderBy(sort string) string {
	switch normalizeUserListFilter(userListFilter{Sort: sort}).Sort {
	case "id-desc":
		return "auth.id DESC"
	case "quota-asc":
		return "COALESCE(quota.quota, 0) ASC, auth.id ASC"
	case "quota-desc":
		return "COALESCE(quota.quota, 0) DESC, auth.id ASC"
	case "used-quota-asc":
		return "COALESCE(quota.used, 0) ASC, auth.id ASC"
	case "used-quota-desc":
		return "COALESCE(quota.used, 0) DESC, auth.id ASC"
	case "plan-asc":
		return "COALESCE(subscription.level, 0) ASC, auth.id ASC"
	case "plan-desc":
		return "COALESCE(subscription.level, 0) DESC, auth.id ASC"
	default:
		return "auth.id ASC"
	}
}

func getUsersForm(db *sql.DB, cache *redis.Client, page int64, search string, filters ...userListFilter) PaginationForm {
	// if search is empty, then search all users

	var users []interface{}
	var total int64
	filter := userListFilter{}
	if len(filters) > 0 {
		filter = filters[0]
	}
	filter = normalizeUserListFilter(filter)
	where, args := buildUserListWhere(search, filter)

	if err := globals.QueryRowDb(db, `
			SELECT COUNT(*) FROM auth
			LEFT JOIN subscription ON subscription.user_id = auth.id
		`+where, args...).Scan(&total); err != nil {
		return PaginationForm{
			Status:  false,
			Message: err.Error(),
		}
	}

	queryArgs := append([]interface{}{}, args...)
	queryArgs = append(queryArgs, pagination, page*pagination)
	rows, err := globals.QueryDb(db, `
			SELECT
			    auth.id, auth.username, auth.email, auth.is_admin,
			    quota.quota, quota.used,
		    subscription.expired_at, subscription.total_month, subscription.enterprise, subscription.level,
		    auth.is_banned
			FROM auth
			LEFT JOIN quota ON quota.user_id = auth.id
			LEFT JOIN subscription ON subscription.user_id = auth.id
			`+where+`
			ORDER BY `+getUserListOrderBy(filter.Sort)+` LIMIT ? OFFSET ?
		`, queryArgs...)
	if err != nil {
		return PaginationForm{
			Status:  false,
			Message: err.Error(),
		}
	}
	defer rows.Close()

	for rows.Next() {
		var user UserData
		var (
			email             sql.NullString
			expired           []uint8
			quota             sql.NullFloat64
			usedQuota         sql.NullFloat64
			totalMonth        sql.NullInt64
			isEnterprise      sql.NullBool
			subscriptionLevel sql.NullInt64
			isBanned          sql.NullBool
		)
		if err := rows.Scan(&user.Id, &user.Username, &email, &user.IsAdmin, &quota, &usedQuota, &expired, &totalMonth, &isEnterprise, &subscriptionLevel, &isBanned); err != nil {
			return PaginationForm{
				Status:  false,
				Message: err.Error(),
			}
		}
		if email.Valid {
			user.Email = email.String
		}
		if quota.Valid {
			user.Quota = float32(quota.Float64)
		}
		if usedQuota.Valid {
			user.UsedQuota = float32(usedQuota.Float64)
		}
		if totalMonth.Valid {
			user.TotalMonth = totalMonth.Int64
		}
		if subscriptionLevel.Valid {
			user.Level = int(subscriptionLevel.Int64)
		}
		stamp := utils.ConvertTime(expired)
		if stamp != nil {
			user.IsSubscribed = stamp.After(time.Now())
			user.ExpiredAt = stamp.Format("2006-01-02 15:04:05")
		}
		user.Enterprise = isEnterprise.Valid && isEnterprise.Bool
		user.IsBanned = isBanned.Valid && isBanned.Bool
		user.SubscriptionWindows = getUserSubscriptionWindows(db, cache, user)

		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return PaginationForm{
			Status:  false,
			Message: err.Error(),
		}
	}

	return PaginationForm{
		Status: true,
		Total:  int(math.Ceil(float64(total) / float64(pagination))),
		Data:   users,
	}
}

func getUserSubscriptionWindows(db *sql.DB, cache *redis.Client, user UserData) []UserSubscriptionWindowData {
	if db == nil || cache == nil || channel.PlanInstance == nil {
		return nil
	}
	if !user.IsSubscribed || user.Level <= 0 {
		return nil
	}

	plan := channel.PlanInstance.GetPlan(user.Level)
	if plan.Level <= 0 {
		return nil
	}

	authLike := &AuthLike{ID: user.Id}
	usage := plan.GetUsage(authLike, db, cache)
	windows := make([]UserSubscriptionWindowData, 0, len(usage))
	appendWindow := func(id string, name string) {
		value, ok := usage[id]
		if !ok {
			return
		}
		windows = append(windows, getUserSubscriptionWindowData(id, name, value))
	}

	if plan.HasPointPool() {
		appendWindow(channel.PlanSharedPointsItemID, channel.PlanSharedPointsItemID)
		if plan.HasWeeklyPool() {
			appendWindow(channel.PlanWeeklyPointsItemID, channel.PlanWeeklyPointsItemID)
		}
		return windows
	}

	for _, item := range plan.Items {
		appendWindow(item.Id, item.Name)
	}
	return windows
}

func getUserSubscriptionWindowData(id string, name string, usage channel.Usage) UserSubscriptionWindowData {
	remaining, percent := getRemainingSubscriptionUsage(usage.Used, usage.Total)
	return UserSubscriptionWindowData{
		Id:               id,
		Name:             name,
		Used:             usage.Used,
		Total:            usage.Total,
		Remaining:        remaining,
		RemainingPercent: percent,
		Unit:             usage.Unit,
		ResetInterval:    usage.ResetInterval,
		ResetAt:          usage.ResetAt,
	}
}

func getRemainingSubscriptionUsage(used float32, total float32) (float32, float32) {
	if total < 0 {
		return -1, 100
	}
	if total == 0 {
		return 0, 0
	}

	remaining := total - used
	if remaining < 0 {
		remaining = 0
	}
	percent := remaining / total * 100
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	return remaining, percent
}

// clearUserCache clears all cache keys starting with nio:user:
func clearUserCache(cache *redis.Client) error {
	if cache == nil {
		return nil
	}

	ctx := context.Background()
	iter := cache.Scan(ctx, 0, "nio:user:*", 100).Iterator()
	for iter.Next(ctx) {
		if err := cache.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to delete cache key %s: %v", iter.Val(), err)
		}
	}
	return iter.Err()
}

func validateNewUser(username string, email string, password string) error {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)

	if len(username) < 2 || len(username) > 24 {
		return fmt.Errorf("username length must be between 2 and 24")
	}
	if len(password) < 6 || len(password) > 36 {
		return fmt.Errorf("password length must be between 6 and 36")
	}
	if !auth.ValidateEmailAddress(email) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

func createUser(db *sql.DB, username string, email string, password string) error {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)

	if err := validateNewUser(username, email, password); err != nil {
		return err
	}

	var count int64
	if err := globals.QueryRowDb(db, "SELECT COUNT(*) FROM auth WHERE username = ?", username).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("username is already taken")
	}

	if err := globals.QueryRowDb(db, "SELECT COUNT(*) FROM auth WHERE email = ?", email).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("email is already taken")
	}

	hash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var bindID int64
	if err := tx.QueryRow(globals.PreflightSql("SELECT COALESCE(MAX(bind_id), -1) + 1 FROM auth")).Scan(&bindID); err != nil {
		return err
	}

	result, err := tx.Exec(globals.PreflightSql(`
		INSERT INTO auth (username, password, email, bind_id, token)
		VALUES (?, ?, ?, ?, ?)
	`), username, hash, email, bindID, utils.Sha2Encrypt(email+username))
	if err != nil {
		return err
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	initialQuota := float32(0)
	if channel.SystemInstance != nil {
		initialQuota = float32(channel.SystemInstance.GetInitialQuota())
	}
	if _, err := tx.Exec(globals.PreflightSql(`
		INSERT INTO quota (user_id, quota, used) VALUES (?, ?, ?)
	`), userID, initialQuota, 0.); err != nil {
		return err
	}

	return tx.Commit()
}

func getUserAdminState(db *sql.DB, id int64) (userAdminState, error) {
	var state userAdminState
	if id <= 0 {
		return state, fmt.Errorf("invalid user id")
	}

	var isAdmin sql.NullBool
	var isBanned sql.NullBool
	err := globals.QueryRowDb(db, `
		SELECT is_admin, is_banned FROM auth WHERE id = ?
	`, id).Scan(&isAdmin, &isBanned)
	if errors.Is(err, sql.ErrNoRows) {
		return state, nil
	}
	if err != nil {
		return state, err
	}

	state.Exists = true
	state.IsAdmin = isAdmin.Valid && isAdmin.Bool
	state.IsBanned = isBanned.Valid && isBanned.Bool
	return state, nil
}

func countActiveAdmins(db *sql.DB) (int64, error) {
	var count int64
	err := globals.QueryRowDb(db, `
		SELECT COUNT(*) FROM auth WHERE is_admin = ? AND (is_banned = ? OR is_banned IS NULL)
	`, true, false).Scan(&count)
	return count, err
}

func ensureCanDisableActiveAdmin(db *sql.DB, id int64, message string) error {
	state, err := getUserAdminState(db, id)
	if err != nil {
		return err
	}
	if !state.Exists {
		return fmt.Errorf("user not found")
	}
	if !state.IsAdmin || state.IsBanned {
		return nil
	}

	count, err := countActiveAdmins(db)
	if err != nil {
		return err
	}
	if count <= 1 {
		return errors.New(message)
	}

	return nil
}

func passwordMigration(db *sql.DB, cache *redis.Client, id int64, password string) error {
	if id <= 0 {
		return fmt.Errorf("invalid user id")
	}

	password = strings.TrimSpace(password)
	if len(password) < 6 || len(password) > 36 {
		return fmt.Errorf("password length must be between 6 and 36")
	}
	hash_passwd, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	// Update password in database
	result, err := globals.ExecDb(db, `
			UPDATE auth SET password = ? WHERE id = ?
	`, hash_passwd, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("user not found")
	}

	// Clear all user related cache
	if err := clearUserCache(cache); err != nil {
		return fmt.Errorf("failed to clear user cache: %v", err)
	}

	return nil
}

func emailMigration(db *sql.DB, id int64, email string) error {
	if id <= 0 {
		return fmt.Errorf("invalid user id")
	}
	email = strings.TrimSpace(email)
	if !auth.ValidateEmailAddress(email) {
		return fmt.Errorf("invalid email format")
	}

	var count int64
	if err := globals.QueryRowDb(db, "SELECT COUNT(*) FROM auth WHERE id = ?", id).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("user not found")
	}

	if err := globals.QueryRowDb(db, "SELECT COUNT(*) FROM auth WHERE email = ? AND id <> ?", email, id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("email is already taken")
	}

	_, err := globals.ExecDb(db, `
		UPDATE auth SET email = ? WHERE id = ?
	`, email, id)

	return err
}

func setAdmin(db *sql.DB, id int64, isAdmin bool) error {
	if !isAdmin {
		if err := ensureCanDisableActiveAdmin(db, id, "cannot remove the last active admin"); err != nil {
			return err
		}
	} else {
		state, err := getUserAdminState(db, id)
		if err != nil {
			return err
		}
		if !state.Exists {
			return fmt.Errorf("user not found")
		}
	}

	_, err := globals.ExecDb(db, `
		UPDATE auth SET is_admin = ? WHERE id = ?
	`, isAdmin, id)

	return err
}

func banUser(db *sql.DB, id int64, isBanned bool) error {
	if isBanned {
		if err := ensureCanDisableActiveAdmin(db, id, "cannot ban the last active admin"); err != nil {
			return err
		}
	} else {
		state, err := getUserAdminState(db, id)
		if err != nil {
			return err
		}
		if !state.Exists {
			return fmt.Errorf("user not found")
		}
	}

	_, err := globals.ExecDb(db, `
		UPDATE auth SET is_banned = ? WHERE id = ?
	`, isBanned, id)

	return err
}

func clearDeletedUserCache(cache *redis.Client, id int64) error {
	if cache == nil {
		return nil
	}

	ctx := context.Background()
	if err := clearUserCache(cache); err != nil {
		return err
	}

	iter := cache.Scan(ctx, 0, fmt.Sprintf("usage-*:%d", id), 100).Iterator()
	for iter.Next(ctx) {
		if err := cache.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to delete cache key %s: %v", iter.Val(), err)
		}
	}
	return iter.Err()
}

func deleteUser(db *sql.DB, cache *redis.Client, id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid user id")
	}

	state, err := getUserAdminState(db, id)
	if err != nil {
		return err
	}
	if !state.Exists {
		return fmt.Errorf("user not found")
	}
	if state.IsAdmin && !state.IsBanned {
		if err := ensureCanDisableActiveAdmin(db, id, "cannot delete the last active admin"); err != nil {
			return err
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	queries := []string{
		"UPDATE invitation SET used = ?, used_id = NULL WHERE used_id = ?",
		"DELETE FROM package WHERE user_id = ?",
		"DELETE FROM quota WHERE user_id = ?",
		"DELETE FROM sharing WHERE user_id = ?",
		"DELETE FROM conversation WHERE user_id = ?",
		"DELETE FROM memories WHERE user_id = ?",
		"DELETE FROM mask WHERE user_id = ?",
		"DELETE FROM subscription WHERE user_id = ?",
		"DELETE FROM passkey_credential WHERE user_id = ?",
		"DELETE FROM broadcast WHERE poster_id = ?",
		"DELETE FROM billing WHERE user_id = ?",
		"DELETE FROM payment_orders WHERE user_id = ?",
		"DELETE FROM auth WHERE id = ?",
	}

	for idx, query := range queries {
		if idx == 0 {
			_, err = tx.Exec(globals.PreflightSql(query), false, id)
		} else {
			_, err = tx.Exec(globals.PreflightSql(query), id)
		}
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return clearDeletedUserCache(cache, id)
}

func batchUsers(db *sql.DB, ids []int64, action string, value float32) error {
	switch action {
	case "ban", "unban", "add_quota":
	default:
		return fmt.Errorf("invalid batch action")
	}

	if action == "ban" {
		activeAdmins, err := countActiveAdmins(db)
		if err != nil {
			return err
		}

		seen := make(map[int64]bool)
		selectedActiveAdmins := int64(0)
		for _, id := range ids {
			if id <= 0 || seen[id] {
				continue
			}
			seen[id] = true

			state, err := getUserAdminState(db, id)
			if err != nil {
				return err
			}
			if state.Exists && state.IsAdmin && !state.IsBanned {
				selectedActiveAdmins++
			}
		}

		if activeAdmins > 0 && activeAdmins-selectedActiveAdmins <= 0 {
			return fmt.Errorf("cannot ban the last active admin")
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for _, id := range ids {
		switch action {
		case "ban":
			_, err = tx.Exec(globals.PreflightSql(`
				UPDATE auth SET is_banned = ? WHERE id = ?
			`), true, id)
		case "unban":
			_, err = tx.Exec(globals.PreflightSql(`
				UPDATE auth SET is_banned = ? WHERE id = ?
			`), false, id)
		case "add_quota":
			_, err = tx.Exec(globals.PreflightSql(
				"INSERT INTO quota (user_id, quota, used) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE quota = quota + ?",
			), id, value, 0., value)
		}
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

func quotaMigration(db *sql.DB, id int64, quota float32, override bool) error {
	// if quota is negative, then decrease quota
	// if quota is positive, then increase quota

	if override {
		_, err := globals.ExecDb(db, `
			INSERT INTO quota (user_id, quota, used) VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE quota = ?
		`, id, quota, 0., quota)

		return err
	}

	_, err := globals.ExecDb(db, `
		INSERT INTO quota (user_id, quota, used) VALUES (?, ?, ?) 
		ON DUPLICATE KEY UPDATE quota = quota + ?
	`, id, quota, 0., quota)

	return err
}

func subscriptionMigration(db *sql.DB, id int64, expired string) error {
	_, err := globals.ExecDb(db, `
		INSERT INTO subscription (user_id, expired_at) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE expired_at = ?
	`, id, expired, expired)
	return err
}

func subscriptionLevelMigration(db *sql.DB, id int64, level int64) error {
	if level < 0 || level > 3 {
		return fmt.Errorf("invalid subscription level")
	}

	_, err := globals.ExecDb(db, `
		INSERT INTO subscription (user_id, level) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE level = ?
	`, id, level, level)

	return err
}

const (
	releaseUsageTypeAll  = "all"
	releaseUsageTypeHour = "hour"
	releaseUsageTypeWeek = "week"
)

func releaseUsage(db *sql.DB, cache *redis.Client, id int64, usageType string) error {
	if channel.PlanInstance == nil {
		return fmt.Errorf("subscription plan is not configured")
	}

	var (
		level     sql.NullInt64
		expiredAt sql.NullString
	)
	if err := globals.QueryRowDb(db, `
			SELECT level, expired_at FROM subscription WHERE user_id = ?
		`, id).Scan(&level, &expiredAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("user is not subscribed")
		}
		return err
	}

	if !level.Valid || level.Int64 == 0 || !expiredAt.Valid {
		return fmt.Errorf("user is not subscribed")
	}
	stamp, err := time.Parse("2006-01-02 15:04:05", expiredAt.String)
	if err != nil || !stamp.After(time.Now()) {
		return fmt.Errorf("user is not subscribed")
	}

	u := &AuthLike{ID: id}

	plan := channel.PlanInstance.GetPlan(int(level.Int64))
	return releasePlanUsage(plan, u, cache, usageType)
}

func releasePlanUsage(plan channel.Plan, user *AuthLike, cache *redis.Client, usageType string) error {
	if plan.Level <= 0 {
		return fmt.Errorf("invalid subscription level")
	}

	switch usageType {
	case "", releaseUsageTypeAll:
		if !plan.ReleaseAll(user, cache) {
			return fmt.Errorf("cannot reset subscription usage")
		}
	case releaseUsageTypeHour:
		if !plan.ReleasePointPool(user, cache) {
			return fmt.Errorf("cannot reset hourly subscription usage")
		}
	case releaseUsageTypeWeek:
		if !plan.ReleaseWeeklyPool(user, cache) {
			return fmt.Errorf("cannot reset weekly subscription usage")
		}
	default:
		return fmt.Errorf("invalid subscription usage reset type")
	}

	return nil
}

func planHasReleaseUsageType(plan channel.Plan, usageType string) bool {
	switch usageType {
	case "", releaseUsageTypeAll:
		return plan.HasPointPool() || plan.HasWeeklyPool() || len(plan.Items) > 0
	case releaseUsageTypeHour:
		return plan.HasPointPool()
	case releaseUsageTypeWeek:
		return plan.HasWeeklyPool()
	default:
		return false
	}
}

func isReleaseUsageTypeValid(usageType string) bool {
	switch usageType {
	case "", releaseUsageTypeAll, releaseUsageTypeHour, releaseUsageTypeWeek:
		return true
	default:
		return false
	}
}

func releaseUsageForSubscribedUsers(db *sql.DB, cache *redis.Client, usageType string) (int64, error) {
	if channel.PlanInstance == nil {
		return 0, fmt.Errorf("subscription plan is not configured")
	}
	if !isReleaseUsageTypeValid(usageType) {
		return 0, fmt.Errorf("invalid subscription usage reset type")
	}

	rows, err := globals.QueryDb(db, `
		SELECT user_id, level FROM subscription
		WHERE level > 0 AND expired_at > ?
	`, time.Now().Format("2006-01-02 15:04:05"))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		var userID int64
		var level int64
		if err := rows.Scan(&userID, &level); err != nil {
			return count, err
		}

		plan := channel.PlanInstance.GetPlan(int(level))
		if plan.Level <= 0 || !planHasReleaseUsageType(plan, usageType) {
			continue
		}

		if err := releasePlanUsage(plan, &AuthLike{ID: userID}, cache, usageType); err != nil {
			return count, err
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return count, err
	}

	return count, nil
}

func UpdateRootPassword(db *sql.DB, cache *redis.Client, password string) error {
	password = strings.TrimSpace(password)
	if len(password) < 6 || len(password) > 36 {
		return fmt.Errorf("password length must be between 6 and 36")
	}

	hash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	result, err := globals.ExecDb(db, `
			UPDATE auth SET password = ? WHERE username = 'root'
	`, hash)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("root user not found")
	}

	// Clear all user related cache
	if err := clearUserCache(cache); err != nil {
		return fmt.Errorf("failed to clear user cache: %v", err)
	}

	return nil
}
