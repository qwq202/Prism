package auth

import (
	"chat/channel"
	"chat/globals"
	"database/sql"
)

const defaultAllowSubscriptionQuotaFallback = false

func (u *User) CreateInitialQuota(db *sql.DB) bool {
	_, err := globals.ExecDb(db, `
		INSERT INTO quota (user_id, quota, used) VALUES (?, ?, ?)
	`, u.GetID(db), channel.SystemInstance.GetInitialQuota(), 0.)
	return err == nil
}

func (u *User) GetQuota(db *sql.DB) float32 {
	var quota float32
	if err := globals.QueryRowDb(db, "SELECT quota FROM quota WHERE user_id = ?", u.GetID(db)).Scan(&quota); err != nil {
		return 0.
	}
	return quota
}

func (u *User) GetUsedQuota(db *sql.DB) float32 {
	var quota float32
	if err := globals.QueryRowDb(db, "SELECT used FROM quota WHERE user_id = ?", u.GetID(db)).Scan(&quota); err != nil {
		return 0.
	}
	return quota
}

func (u *User) AllowSubscriptionQuotaFallback(db *sql.DB) bool {
	if u == nil || db == nil {
		return defaultAllowSubscriptionQuotaFallback
	}

	var allow sql.NullBool
	if err := globals.QueryRowDb(db, "SELECT allow_subscription_quota_fallback FROM quota WHERE user_id = ?", u.GetID(db)).Scan(&allow); err != nil {
		return defaultAllowSubscriptionQuotaFallback
	}
	if !allow.Valid {
		return defaultAllowSubscriptionQuotaFallback
	}
	return allow.Bool
}

func (u *User) SetAllowSubscriptionQuotaFallback(db *sql.DB, allow bool) bool {
	if u == nil || db == nil {
		return false
	}

	userID := u.GetID(db)
	result, err := globals.ExecDb(db, `
		UPDATE quota
		SET allow_subscription_quota_fallback = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, allow, userID)
	if err != nil {
		return false
	}

	affected, err := result.RowsAffected()
	if err == nil && affected > 0 {
		return true
	}

	_, err = globals.ExecDb(db, `
		INSERT INTO quota (user_id, quota, used, allow_subscription_quota_fallback)
		VALUES (?, ?, ?, ?)
	`, userID, 0., 0., allow)
	return err == nil
}

func (u *User) SetQuota(db *sql.DB, quota float32) bool {
	_, err := globals.ExecDb(db, `
		INSERT INTO quota (user_id, quota, used) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE quota = ?
	`, u.GetID(db), quota, 0., quota)
	return err == nil
}

func (u *User) SetUsedQuota(db *sql.DB, used float32) bool {
	_, err := globals.ExecDb(db, `
		INSERT INTO quota (user_id, quota, used) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE used = ?
	`, u.GetID(db), 0., used, used)
	return err == nil
}

func (u *User) IncreaseQuota(db *sql.DB, quota float32) bool {
	_, err := globals.ExecDb(db, `
		INSERT INTO quota (user_id, quota, used) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE quota = quota + ?
	`, u.GetID(db), quota, 0., quota)
	return err == nil
}

func increaseQuotaByUserIDTx(tx *sql.Tx, userID int64, quota float32) error {
	if quota <= 0 {
		return nil
	}

	_, err := globals.ExecTx(tx, `
		INSERT INTO quota (user_id, quota, used) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE quota = quota + ?
	`, userID, quota, 0., quota)
	return err
}

func (u *User) IncreaseUsedQuota(db *sql.DB, used float32) bool {
	_, err := globals.ExecDb(db, `
		INSERT INTO quota (user_id, quota, used) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE used = used + ?
	`, u.GetID(db), 0., used, used)
	return err == nil
}

func (u *User) DecreaseQuota(db *sql.DB, quota float32) bool {
	if quota <= 0 {
		return true
	}
	if u == nil || db == nil {
		return false
	}

	result, err := globals.ExecDb(db, `
		UPDATE quota
		SET quota = quota - ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND quota >= ?
	`, quota, u.GetID(db), quota)
	if err != nil {
		return false
	}

	affected, err := result.RowsAffected()
	return err == nil && affected > 0
}

func (u *User) UseQuota(db *sql.DB, quota float32) bool {
	if quota <= 0 {
		return true
	}

	result, err := globals.ExecDb(db, `
		UPDATE quota
		SET quota = quota - ?, used = used + ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND quota >= ?
	`, quota, quota, u.GetID(db), quota)
	if err != nil {
		return false
	}

	affected, err := result.RowsAffected()
	return err == nil && affected > 0
}

func (u *User) ForceUseQuota(db *sql.DB, quota float32) bool {
	if quota <= 0 {
		return true
	}
	if u == nil || db == nil {
		return false
	}

	result, err := globals.ExecDb(db, `
		UPDATE quota
		SET used = used + CASE WHEN quota >= ? THEN ? ELSE quota END,
			quota = CASE WHEN quota >= ? THEN quota - ? ELSE 0 END,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND quota > 0
	`, quota, quota, quota, quota, u.GetID(db))
	if err != nil {
		return false
	}

	affected, err := result.RowsAffected()
	return err == nil && affected > 0
}

func (u *User) ChargeQuota(db *sql.DB, quota float32) bool {
	if quota <= 0 {
		return true
	}
	if u == nil || db == nil {
		return false
	}
	if u.UseQuota(db, quota) {
		return true
	}
	u.ForceUseQuota(db, quota)
	return false
}

func (u *User) PayedQuota(db *sql.DB, quota float32) bool {
	return u.UseQuota(db, quota)
}

func (u *User) PayedQuotaAsAmount(db *sql.DB, amount float32) bool {
	return u.PayedQuota(db, amount*10)
}
