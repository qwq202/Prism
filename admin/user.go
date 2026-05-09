package admin

import (
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/mail"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// AuthLike is to solve the problem of import cycle
type AuthLike struct {
	ID int64 `json:"id"`
}

func (a *AuthLike) GetID(_ *sql.DB) int64 {
	return a.ID
}

func (a *AuthLike) HitID() int64 {
	return a.ID
}

func getUsersForm(db *sql.DB, page int64, search string) PaginationForm {
	// if search is empty, then search all users

	var users []interface{}
	var total int64

	if err := globals.QueryRowDb(db, `
		SELECT COUNT(*) FROM auth
		WHERE username LIKE ?
	`, "%"+search+"%").Scan(&total); err != nil {
		return PaginationForm{
			Status:  false,
			Message: err.Error(),
		}
	}

	rows, err := globals.QueryDb(db, `
		SELECT 
		    auth.id, auth.username, auth.email, auth.is_admin,
		    quota.quota, quota.used,
		    subscription.expired_at, subscription.total_month, subscription.enterprise, subscription.level,
		    auth.is_banned
		FROM auth
		LEFT JOIN quota ON quota.user_id = auth.id
		LEFT JOIN subscription ON subscription.user_id = auth.id
		WHERE auth.username LIKE ?
		ORDER BY auth.id LIMIT ? OFFSET ?
	`, "%"+search+"%", pagination, page*pagination)
	if err != nil {
		return PaginationForm{
			Status:  false,
			Message: err.Error(),
		}
	}

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

		users = append(users, user)
	}

	return PaginationForm{
		Status: true,
		Total:  int(math.Ceil(float64(total) / float64(pagination))),
		Data:   users,
	}
}

// clearUserCache clears all cache keys starting with nio:user:
func clearUserCache(cache *redis.Client) error {
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
	if len(email) < 1 || len(email) > 255 {
		return fmt.Errorf("invalid email format")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
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

func passwordMigration(db *sql.DB, cache *redis.Client, id int64, password string) error {
	password = strings.TrimSpace(password)
	if len(password) < 6 || len(password) > 36 {
		return fmt.Errorf("password length must be between 6 and 36")
	}
	hash_passwd, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	// Update password in database
	_, err = globals.ExecDb(db, `
		UPDATE auth SET password = ? WHERE id = ?
	`, hash_passwd, id)

	if err != nil {
		return err
	}

	// Clear all user related cache
	if err := clearUserCache(cache); err != nil {
		return fmt.Errorf("failed to clear user cache: %v", err)
	}

	return nil
}

func emailMigration(db *sql.DB, id int64, email string) error {
	_, err := globals.ExecDb(db, `
		UPDATE auth SET email = ? WHERE id = ?
	`, email, id)

	return err
}

func setAdmin(db *sql.DB, id int64, isAdmin bool) error {
	_, err := globals.ExecDb(db, `
		UPDATE auth SET is_admin = ? WHERE id = ?
	`, isAdmin, id)

	return err
}

func banUser(db *sql.DB, id int64, isBanned bool) error {
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

	var exists int64
	if err := globals.QueryRowDb(db, "SELECT COUNT(*) FROM auth WHERE id = ?", id).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return fmt.Errorf("user not found")
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
		"DELETE FROM apikey WHERE user_id = ?",
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
	var level sql.NullInt64
	if err := globals.QueryRowDb(db, `
		SELECT level FROM subscription WHERE user_id = ?
	`, id).Scan(&level); err != nil {
		return err
	}

	if !level.Valid || level.Int64 == 0 {
		return fmt.Errorf("user is not subscribed")
	}

	u := &AuthLike{ID: id}

	plan := channel.PlanInstance.GetPlan(int(level.Int64))
	switch usageType {
	case "", releaseUsageTypeAll:
		if !plan.ReleaseAll(u, cache) {
			return fmt.Errorf("cannot reset subscription usage")
		}
	case releaseUsageTypeHour:
		if !plan.ReleasePointPool(u, cache) {
			return fmt.Errorf("cannot reset hourly subscription usage")
		}
	case releaseUsageTypeWeek:
		if !plan.ReleaseWeeklyPool(u, cache) {
			return fmt.Errorf("cannot reset weekly subscription usage")
		}
	default:
		return fmt.Errorf("invalid subscription usage reset type")
	}

	return nil
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

	if _, err := globals.ExecDb(db, `
		UPDATE auth SET password = ? WHERE username = 'root'
	`, hash); err != nil {
		return err
	}

	// Clear all user related cache
	if err := clearUserCache(cache); err != nil {
		return fmt.Errorf("failed to clear user cache: %v", err)
	}

	return nil
}
