package admin

import (
	"chat/globals"
	"chat/utils"
	"database/sql"
	"fmt"
	"math"
	"strings"
)

type redeemExecer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func execRedeemSql(execer redeemExecer, query string, args ...interface{}) (sql.Result, error) {
	return execer.Exec(globals.PreflightSql(query), args...)
}

func GetRedeemData(db *sql.DB, page int64) PaginationForm {
	var data []interface{}
	var total int64
	if err := globals.QueryRowDb(db, `
		SELECT COUNT(*) FROM redeem
	`).Scan(&total); err != nil {
		return PaginationForm{
			Status:  false,
			Message: err.Error(),
		}
	}

	rows, err := globals.QueryDb(db, `
		SELECT code, quota, used, created_at, updated_at
		FROM redeem
		ORDER BY id DESC LIMIT ? OFFSET ?
	`, pagination, page*pagination)

	if err != nil {
		return PaginationForm{
			Status:  false,
			Message: err.Error(),
		}
	}

	for rows.Next() {
		var redeem RedeemData
		var createdAt []uint8
		var updatedAt []uint8
		if err := rows.Scan(&redeem.Code, &redeem.Quota, &redeem.Used, &createdAt, &updatedAt); err != nil {
			return PaginationForm{
				Status:  false,
				Message: err.Error(),
			}
		}

		redeem.CreatedAt = utils.ConvertTime(createdAt).Format("2006-01-02 15:04:05")
		redeem.UpdatedAt = utils.ConvertTime(updatedAt).Format("2006-01-02 15:04:05")
		data = append(data, redeem)
	}

	return PaginationForm{
		Status: true,
		Total:  int(math.Ceil(float64(total) / float64(pagination))),
		Data:   data,
	}
}

func DeleteRedeemCode(db *sql.DB, code string) error {
	_, err := globals.ExecDb(db, `
		DELETE FROM redeem WHERE code = ?
	`, code)

	return err
}

func GenerateRedeemCodes(db *sql.DB, num int, quota float32) RedeemGenerateResponse {
	batchId := utils.GenerateChar(16)

	tx, err := db.Begin()
	if err != nil {
		return RedeemGenerateResponse{Status: false, Message: err.Error()}
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	_, err = execRedeemSql(tx, `
		INSERT INTO redeem_batch (id, quota, count) VALUES (?, ?, ?)
	`, batchId, quota, num)
	if err != nil {
		return RedeemGenerateResponse{Status: false, Message: err.Error()}
	}

	arr := make([]string, 0)
	idx := 0
	for idx < num {
		code, err := createRedeemCode(tx, quota, batchId)

		if err != nil {
			return RedeemGenerateResponse{
				Status:  false,
				Message: err.Error(),
			}
		}
		arr = append(arr, code)
		idx++
	}

	if err := tx.Commit(); err != nil {
		return RedeemGenerateResponse{Status: false, Message: err.Error()}
	}
	committed = true

	return RedeemGenerateResponse{
		Status: true,
		Data:   arr,
	}
}

func createRedeemCode(execer redeemExecer, quota float32, batchId string) (string, error) {
	code := fmt.Sprintf("nio-%s", utils.GenerateChar(32))
	_, err := execRedeemSql(execer, `
		INSERT INTO redeem (code, quota, batch_id) VALUES (?, ?, ?)
	`, code, quota, batchId)

	if err != nil && isDuplicateRedeemCodeError(err) {
		return createRedeemCode(execer, quota, batchId)
	}

	return code, err
}

func isDuplicateRedeemCodeError(err error) bool {
	content := err.Error()
	return strings.Contains(content, "Duplicate entry") ||
		strings.Contains(strings.ToLower(content), "unique constraint failed")
}

func GetRedeemBatches(db *sql.DB) RedeemBatchResponse {
	rows, err := globals.QueryDb(db, `
		SELECT rb.id, rb.quota, rb.count, rb.created_at,
		       COALESCE(SUM(CASE WHEN r.used = TRUE THEN 1 ELSE 0 END), 0) as used_count
		FROM redeem_batch rb
		LEFT JOIN redeem r ON r.batch_id = rb.id
		GROUP BY rb.id, rb.quota, rb.count, rb.created_at
		ORDER BY rb.created_at DESC
	`)
	if err != nil {
		return RedeemBatchResponse{Status: false, Message: err.Error()}
	}

	var batches []RedeemBatchData
	for rows.Next() {
		var b RedeemBatchData
		var createdAt []uint8
		if err := rows.Scan(&b.Id, &b.Quota, &b.Count, &createdAt, &b.UsedCount); err != nil {
			return RedeemBatchResponse{Status: false, Message: err.Error()}
		}
		b.CreatedAt = utils.ConvertTime(createdAt).Format("2006-01-02 15:04:05")
		batches = append(batches, b)
	}

	if batches == nil {
		batches = []RedeemBatchData{}
	}

	return RedeemBatchResponse{Status: true, Data: batches}
}

func GetBatchCodes(db *sql.DB, batchId string) RedeemBatchCodesResponse {
	rows, err := globals.QueryDb(db, `
		SELECT code, quota, used, created_at, updated_at
		FROM redeem
		WHERE batch_id = ?
		ORDER BY id ASC
	`, batchId)
	if err != nil {
		return RedeemBatchCodesResponse{Status: false, Message: err.Error()}
	}

	var codes []RedeemData
	for rows.Next() {
		var r RedeemData
		var createdAt []uint8
		var updatedAt []uint8
		if err := rows.Scan(&r.Code, &r.Quota, &r.Used, &createdAt, &updatedAt); err != nil {
			return RedeemBatchCodesResponse{Status: false, Message: err.Error()}
		}
		r.CreatedAt = utils.ConvertTime(createdAt).Format("2006-01-02 15:04:05")
		r.UpdatedAt = utils.ConvertTime(updatedAt).Format("2006-01-02 15:04:05")
		codes = append(codes, r)
	}

	if codes == nil {
		codes = []RedeemData{}
	}

	return RedeemBatchCodesResponse{Status: true, Data: codes}
}
