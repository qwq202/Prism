package billing

import (
	"chat/connection"
	"chat/globals"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func openBillingTestDB(t *testing.T) *sql.DB {
	t.Helper()

	previous := globals.SqliteEngine
	globals.SqliteEngine = true
	t.Cleanup(func() {
		globals.SqliteEngine = previous
	})

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "billing-test.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	connection.CreateUserTable(db)
	connection.CreateBillingTable(db)
	return db
}

func seedBillingRecord(
	t *testing.T,
	db *sql.DB,
	username string,
	recordType string,
	tokenName string,
	model string,
	createdAt string,
) {
	t.Helper()

	if _, err := globals.ExecDb(db, `
		INSERT INTO billing (
			user_id, username, type, token_name, model,
			input_tokens, output_tokens, quota, duration,
			detail, prompts, response_prompts, channel, channel_name, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 1, username, recordType, tokenName, model, 10, 20, 1.25, 0.8, "", "", "", 0, "", createdAt); err != nil {
		t.Fatalf("insert billing record: %v", err)
	}
}

func TestListRecordsFiltersByPartialTokenNameAndInclusiveDate(t *testing.T) {
	db := openBillingTestDB(t)

	seedBillingRecord(t, db, "root", "consume", "alpha-main-token", "deepseek-v4-flash", "2026-04-22 15:30:00")
	seedBillingRecord(t, db, "root", "consume", "beta-secondary-token", "deepseek-v4-flash", "2026-04-21 23:59:59")
	seedBillingRecord(t, db, "root", "topup", "alpha-topup-token", "grok-4-1-fast", "2026-04-22 09:00:00")

	data, err := ListRecords(db, true, 1, 0, RecordQuery{
		TokenName: "alpha",
		StartTime: "2026-04-22",
		EndTime:   "2026-04-22",
		Type:      "consume",
	})
	if err != nil {
		t.Fatalf("list records: %v", err)
	}

	if len(data.Records) != 1 {
		t.Fatalf("expected 1 filtered record, got %d (%#v)", len(data.Records), data.Records)
	}

	record := data.Records[0]
	if record.TokenName != "alpha-main-token" {
		t.Fatalf("expected token filter to keep partial match, got %#v", record)
	}

	if record.Type != "consume" {
		t.Fatalf("expected type filter to keep consume record, got %#v", record)
	}
}

func TestListRecordsTreatsUsernameFilterAsBoundParameter(t *testing.T) {
	db := openBillingTestDB(t)

	seedBillingRecord(t, db, "root", "consume", "alpha-main-token", "deepseek-v4-flash", "2026-04-22 15:30:00")
	seedBillingRecord(t, db, "alice", "consume", "beta-secondary-token", "grok-4-1-fast", "2026-04-23 09:00:00")

	data, err := ListRecords(db, true, 1, 0, RecordQuery{
		Username: "%' OR 1=1 --",
	})
	if err != nil {
		t.Fatalf("list records: %v", err)
	}

	if len(data.Records) != 0 {
		t.Fatalf("expected malicious username filter to match no records, got %d (%#v)", len(data.Records), data.Records)
	}
}

func TestListRecordsRejectsInvalidDateFilter(t *testing.T) {
	db := openBillingTestDB(t)

	_, err := ListRecords(db, true, 1, 0, RecordQuery{
		StartTime: "2026/04/22",
	})
	if err == nil {
		t.Fatalf("expected invalid date filter to return error")
	}
}
