package auth

import (
	"chat/globals"
	"database/sql"
)

type GiftResponse struct {
	Cert     bool `json:"cert"`
	Teenager bool `json:"teenager"`
}

func (u *User) HasPackage(db *sql.DB, _t string) bool {
	var count int
	if err := globals.QueryRowDb(db, `SELECT COUNT(*) FROM package where user_id = ? AND type = ?`, u.ID, _t).Scan(&count); err != nil {
		return false
	}

	return count > 0
}

func (u *User) HasCertPackage(db *sql.DB) bool {
	return u.HasPackage(db, "cert")
}

func (u *User) HasTeenagerPackage(db *sql.DB) bool {
	return u.HasPackage(db, "teenager")
}

func NewPackage(db *sql.DB, user *User, _t string) bool {
	return grantPackageQuota(db, user, _t, 0)
}

func grantPackageQuota(db *sql.DB, user *User, _t string, quota float32) bool {
	if user == nil {
		return false
	}

	id := user.GetID(db)

	tx, err := db.Begin()
	if err != nil {
		return false
	}
	defer tx.Rollback()

	if _, err := globals.ExecTx(tx, `INSERT INTO package (user_id, type) VALUES (?, ?)`, id, _t); err != nil {
		return false
	}

	if err := increaseQuotaByUserIDTx(tx, id, quota); err != nil {
		return false
	}

	return tx.Commit() == nil
}

func NewCertPackage(db *sql.DB, user *User) bool {
	return grantPackageQuota(db, user, "cert", 50)
}

func NewTeenagerPackage(db *sql.DB, user *User) bool {
	return grantPackageQuota(db, user, "teenager", 150)
}

func RefreshPackage(db *sql.DB, user *User) *GiftResponse {
	if !useDeeptrain() {
		return nil
	}

	resp := Cert(user.Username)
	if resp == nil || resp.Status == false {
		return nil
	}

	if resp.Cert {
		NewCertPackage(db, user)
	}
	if resp.Teenager {
		NewTeenagerPackage(db, user)
	}

	return &GiftResponse{
		Cert:     resp.Cert,
		Teenager: resp.Teenager,
	}
}
