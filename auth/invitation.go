package auth

import (
	"chat/globals"
	"chat/utils"
	"database/sql"
	"errors"
	"fmt"
)

type Invitation struct {
	Id     int64   `json:"id"`
	Code   string  `json:"code"`
	Quota  float32 `json:"quota"`
	Type   string  `json:"type"`
	Used   bool    `json:"used"`
	UsedId int64   `json:"used_id"`
}

func GenerateInvitations(db *sql.DB, num int, quota float32, t string) ([]string, error) {
	arr := make([]string, 0)
	idx := 0
	for idx < num {
		code := fmt.Sprintf("%s-%s", t, utils.GenerateChar(24))
		if err := CreateInvitationCode(db, code, quota, t); err != nil {
			// unique constraint
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

func CreateInvitationCode(db *sql.DB, code string, quota float32, t string) error {
	_, err := globals.ExecDb(db, `
		INSERT INTO invitation (code, quota, type)
		VALUES (?, ?, ?)
	`, code, quota, t)
	return err
}

func GetInvitation(db *sql.DB, code string) (*Invitation, error) {
	row := globals.QueryRowDb(db, `
		SELECT id, code, quota, type, used, used_id
		FROM invitation
		WHERE code = ?
	`, code)
	var invitation Invitation
	var id sql.NullInt64
	err := row.Scan(&invitation.Id, &invitation.Code, &invitation.Quota, &invitation.Type, &invitation.Used, &id)
	if id.Valid {
		invitation.UsedId = id.Int64
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("invitation code not found")
		}
		return nil, fmt.Errorf("failed to get invitation: %w", err)
	}
	return &invitation, nil
}

func (i *Invitation) IsUsed() bool {
	return i.Used
}

func (i *Invitation) Use(db *sql.DB, userId int64) error {
	result, err := globals.ExecDb(db, `
		UPDATE invitation SET used = TRUE, used_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND used = FALSE
	`, userId, i.Id)
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

func (i *Invitation) GetQuota() float32 {
	return i.Quota
}

func (i *Invitation) UseInvitation(db *sql.DB, user User) error {
	if i == nil {
		return fmt.Errorf("invalid invitation request")
	}
	if i.IsUsed() {
		return fmt.Errorf("this invitation has been used")
	}

	userID := user.GetID(db)
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start invitation transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := globals.ExecTx(tx, `
		UPDATE invitation SET used = TRUE, used_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND used = FALSE
	`, userID, i.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("invitation code not found")
		} else if errors.Is(err, sql.ErrTxDone) {
			return fmt.Errorf("transaction has been closed")
		}
		return fmt.Errorf("failed to use invitation: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to confirm invitation usage: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("this invitation has been used")
	}

	if err := increaseQuotaByUserIDTx(tx, userID, i.GetQuota()); err != nil {
		return fmt.Errorf("failed to increase quota for user: %w", err)
	}

	return tx.Commit()
}

func (u *User) UseInvitation(db *sql.DB, code string) (float32, error) {
	if invitation, err := GetInvitation(db, code); err != nil {
		return 0, err
	} else {
		if err := invitation.UseInvitation(db, *u); err != nil {
			return 0, fmt.Errorf("failed to use invitation: %w", err)
		}

		return invitation.GetQuota(), nil
	}
}
