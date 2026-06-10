package auth

import (
	"chat/globals"
	"chat/utils"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
)

type Redeem struct {
	Id    int64   `json:"id"`
	Code  string  `json:"code"`
	Quota float32 `json:"quota"`
	Used  bool    `json:"used"`
}

func GenerateRedeemCodes(db *sql.DB, num int, quota float32) ([]string, error) {
	arr := make([]string, 0)
	idx := 0
	for idx < num {
		code := fmt.Sprintf("nio-%s", utils.GenerateChar(32))
		if err := CreateRedeemCode(db, code, quota); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, fmt.Errorf("failed to generate code: %w", err)
		}
		arr = append(arr, code)
		idx++
	}

	return arr, nil
}

func CreateRedeemCode(db *sql.DB, code string, quota float32) error {
	_, err := globals.ExecDb(db, `
		INSERT INTO redeem (code, quota) VALUES (?, ?)
	`, code, quota)
	return err
}

func GetRedeemCode(db *sql.DB, code string) (*Redeem, error) {
	row := globals.QueryRowDb(db, `
		SELECT id, code, quota, used
		FROM redeem
		WHERE code = ?
	`, code)
	var redeem Redeem
	err := row.Scan(&redeem.Id, &redeem.Code, &redeem.Quota, &redeem.Used)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("redeem code not found")
		}
		return nil, fmt.Errorf("failed to get redeem code: %w", err)
	}
	return &redeem, nil
}
func (r *Redeem) IsUsed() bool {
	return r.Used
}

func (r *Redeem) Use(db *sql.DB) error {
	result, err := globals.ExecDb(db, `
		UPDATE redeem SET used = TRUE, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND used = FALSE
	`, r.Id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Redeem) GetQuota() float32 {
	return r.Quota
}

func (r *Redeem) UseRedeem(db *sql.DB, user *User) error {
	if r == nil || user == nil {
		return fmt.Errorf("invalid redeem request")
	}
	if r.IsUsed() {
		return fmt.Errorf("this redeem code has been used")
	}

	userID := user.GetID(db)
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start redeem transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := globals.ExecTx(tx, `
		UPDATE redeem SET used = TRUE, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND used = FALSE
	`, r.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("redeem code not found")
		} else if errors.Is(err, sql.ErrTxDone) {
			return fmt.Errorf("transaction has been closed")
		}
		return fmt.Errorf("failed to use redeem code: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to confirm redeem code usage: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("this redeem code has been used")
	}

	if err := increaseQuotaByUserIDTx(tx, userID, r.GetQuota()); err != nil {
		return fmt.Errorf("failed to increase quota for user: %w", err)
	}

	return tx.Commit()
}

func (u *User) UseRedeem(db *sql.DB, cache *redis.Client, code string) (float32, error) {
	if useDeeptrain() {
		return 0, errors.New("redeem code is not available in deeptrain mode")
	}

	if redeem, err := GetRedeemCode(db, code); err != nil {
		return 0, err
	} else {
		if err := redeem.UseRedeem(db, u); err != nil {
			return 0, fmt.Errorf("failed to use redeem code: %w", err)
		}

		incrBillingRequest(cache, int64(redeem.GetQuota()*10))
		return redeem.GetQuota(), nil
	}
}
