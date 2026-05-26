package billing

import (
	"chat/connection"
	"chat/globals"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

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
	seedBillingRecordWithQuota(t, db, username, recordType, tokenName, model, 1.25, createdAt)
}

func seedBillingRecordWithQuota(
	t *testing.T,
	db *sql.DB,
	username string,
	recordType string,
	tokenName string,
	model string,
	quota float64,
	createdAt string,
) {
	t.Helper()

	if _, err := globals.ExecDb(db, `
		INSERT INTO billing (
			user_id, username, type, token_name, model,
			input_tokens, output_tokens, quota, duration,
			detail, prompts, response_prompts, channel, channel_name, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 1, username, recordType, tokenName, model, 10, 20, quota, 0.8, "", "", "", 0, "", createdAt); err != nil {
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

func TestListRecordsConvertsOffsetFilterToRecordStorageTime(t *testing.T) {
	db := openBillingTestDB(t)

	seedBillingRecord(t, db, "root", "consume", "before-browser-day", "deepseek-v4-flash", "2026-05-23 11:59:59")
	seedBillingRecord(t, db, "root", "consume", "browser-day-start", "deepseek-v4-flash", "2026-05-23 12:00:00")
	seedBillingRecord(t, db, "root", "consume", "browser-day-end", "deepseek-v4-flash", "2026-05-24 11:59:59")
	seedBillingRecord(t, db, "root", "consume", "after-browser-day", "deepseek-v4-flash", "2026-05-24 12:00:00")

	data, err := ListRecords(db, true, 1, 0, RecordQuery{
		StartTime: "2026-05-23T00:00:00-04:00",
		EndTime:   "2026-05-23T23:59:59-04:00",
		Type:      "consume",
	})
	if err != nil {
		t.Fatalf("list records: %v", err)
	}

	if len(data.Records) != 2 {
		t.Fatalf("expected 2 records in browser-local day, got %d (%#v)", len(data.Records), data.Records)
	}

	got := map[string]bool{}
	for _, record := range data.Records {
		got[record.TokenName] = true
	}
	if !got["browser-day-start"] || !got["browser-day-end"] {
		t.Fatalf("expected converted range to keep browser day boundaries, got %#v", got)
	}
}

func TestListRecordsReturnsCreatedAtWithRecordStorageLocation(t *testing.T) {
	db := openBillingTestDB(t)

	seedBillingRecord(t, db, "root", "consume", "time-zone-token", "deepseek-v4-flash", "2026-05-23 19:55:28")

	data, err := ListRecords(db, true, 1, 0, RecordQuery{
		TokenName: "time-zone-token",
	})
	if err != nil {
		t.Fatalf("list records: %v", err)
	}

	if len(data.Records) != 1 {
		t.Fatalf("expected 1 record, got %d (%#v)", len(data.Records), data.Records)
	}

	if got := data.Records[0].CreatedAt.Format("2006-01-02T15:04:05Z07:00"); got != "2026-05-23T19:55:28+08:00" {
		t.Fatalf("expected created_at to preserve record storage zone, got %s", got)
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

func TestCreateRecordPersistsBeforeReturning(t *testing.T) {
	db := openBillingTestDB(t)

	CreateRecord(
		db,
		1,
		"root",
		"consume",
		"sync-token",
		"gpt-5.4-mini",
		12,
		34,
		0.56,
		1.2,
		"detail",
		"prompt",
		"response",
		7,
		"channel",
	)

	var count int
	if err := globals.QueryRowDb(db, "SELECT COUNT(*) FROM billing WHERE token_name = ?", "sync-token").Scan(&count); err != nil {
		t.Fatalf("count billing record: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected billing record to be persisted before return, got %d", count)
	}

	var createdAtRaw string
	if err := globals.QueryRowDb(db, "SELECT created_at FROM billing WHERE token_name = ?", "sync-token").Scan(&createdAtRaw); err != nil {
		t.Fatalf("select billing record created_at: %v", err)
	}

	createdAt, err := time.ParseInLocation("2006-01-02 15:04:05", createdAtRaw, recordStorageLocation())
	if err != nil {
		t.Fatalf("parse billing record created_at %q: %v", createdAtRaw, err)
	}

	now := time.Now().In(recordStorageLocation())
	if diff := now.Sub(createdAt); diff < -5*time.Second || diff > 5*time.Second {
		t.Fatalf("expected created_at to be written in record storage zone near now, got %s, now %s", createdAtRaw, now.Format("2006-01-02 15:04:05"))
	}
}

func TestGetRecordUsageSummaryAggregatesFilteredConsumeRecords(t *testing.T) {
	db := openBillingTestDB(t)

	seedBillingRecordWithQuota(t, db, "root", "consume", "web", "gemini-3.5-flash", 9, "2026-05-26 15:46:51")
	seedBillingRecordWithQuota(t, db, "root", "consume", "api", "gemini-3.5-flash", 3, "2026-05-25 10:00:00")
	seedBillingRecordWithQuota(t, db, "root", "consume", "web", "mimo-v2.5", 5, "2026-05-24 10:00:00")
	seedBillingRecordWithQuota(t, db, "root", "consume", "web", "old-model", 99, "2026-05-10 10:00:00")
	seedBillingRecordWithQuota(t, db, "root", "topup", "web", "gemini-3.5-flash", 100, "2026-05-26 16:00:00")

	summary, err := GetRecordUsageSummary(db, false, 1, RecordQuery{
		StartTime: "2026-05-24",
		EndTime:   "2026-05-26",
	})
	if err != nil {
		t.Fatalf("get usage summary: %v", err)
	}

	if summary.ModelCount != 2 {
		t.Fatalf("expected 2 models, got %d (%#v)", summary.ModelCount, summary.Models)
	}
	if summary.TopModel != "gemini-3.5-flash" {
		t.Fatalf("expected gemini as top model, got %q", summary.TopModel)
	}
	if summary.MaxQuota != 9 {
		t.Fatalf("expected max quota 9, got %v", summary.MaxQuota)
	}
	if summary.AverageQuota != float64(17)/3 {
		t.Fatalf("expected average quota %v, got %v", float64(17)/3, summary.AverageQuota)
	}
	if len(summary.Models) != 2 || summary.Models[0].Name != "gemini-3.5-flash" || summary.Models[0].Value != 12 {
		t.Fatalf("expected gemini aggregate first, got %#v", summary.Models)
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
