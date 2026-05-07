package admin

import (
	"chat/connection"
	"chat/globals"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func openAdminUserTestDB(t *testing.T) *sql.DB {
	t.Helper()

	previousSqlite := globals.SqliteEngine
	globals.SqliteEngine = true
	t.Cleanup(func() {
		globals.SqliteEngine = previousSqlite
	})

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "admin-user.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})

	connection.CreateUserTable(db)
	connection.CreateQuotaTable(db)

	return db
}

func TestCreateUserCreatesAuthAndQuotaRows(t *testing.T) {
	db := openAdminUserTestDB(t)

	if err := createUser(db, "alice", "alice@example.com", "secret123"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	var id int64
	var bindID int64
	if err := globals.QueryRowDb(db, "SELECT id, bind_id FROM auth WHERE username = ?", "alice").Scan(&id, &bindID); err != nil {
		t.Fatalf("fetch created user: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected created user id")
	}
	if bindID != 1 {
		t.Fatalf("expected bind id 1 after root user, got %d", bindID)
	}

	var quota float32
	if err := globals.QueryRowDb(db, "SELECT quota FROM quota WHERE user_id = ?", id).Scan(&quota); err != nil {
		t.Fatalf("fetch created user quota: %v", err)
	}
	if quota != 0 {
		t.Fatalf("expected default quota 0 without system config, got %f", quota)
	}
}

func TestCreateUserRejectsDuplicateUsername(t *testing.T) {
	db := openAdminUserTestDB(t)

	if err := createUser(db, "alice", "alice@example.com", "secret123"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := createUser(db, "alice", "other@example.com", "secret123"); err == nil {
		t.Fatalf("expected duplicate username to be rejected")
	}
}
