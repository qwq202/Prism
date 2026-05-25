package connection

import (
	"chat/globals"
	"chat/utils"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

func openConnectionTestDB(t *testing.T) *sql.DB {
	t.Helper()

	previousSqlite := globals.SqliteEngine
	globals.SqliteEngine = true
	t.Cleanup(func() {
		globals.SqliteEngine = previousSqlite
	})

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "connection.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func withInitialRootPassword(t *testing.T, password string) {
	t.Helper()

	previous := viper.GetString("root.initial_password")
	viper.Set("root.initial_password", password)
	t.Cleanup(func() {
		viper.Set("root.initial_password", previous)
	})
}

func fetchRootPasswordHash(t *testing.T, db *sql.DB) string {
	t.Helper()

	var hash string
	if err := globals.QueryRowDb(db, "SELECT password FROM auth WHERE username = ?", defaultRootUsername).Scan(&hash); err != nil {
		t.Fatalf("fetch root password hash: %v", err)
	}

	return hash
}

func expectPanicContaining(t *testing.T, want string, run func()) {
	t.Helper()

	defer func() {
		value := recover()
		if value == nil {
			t.Fatalf("expected panic containing %q", want)
		}
		if got := fmt.Sprint(value); !strings.Contains(got, want) {
			t.Fatalf("expected panic containing %q, got %q", want, got)
		}
	}()

	run()
}

func TestCreateUserTablePanicsOnSchemaError(t *testing.T) {
	db := openConnectionTestDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	expectPanicContaining(t, "create auth table failed", func() {
		CreateUserTable(db)
	})
}

func TestMigrateDatabasePanicsOnMigrationError(t *testing.T) {
	db := openConnectionTestDB(t)

	expectPanicContaining(t, "database migration failed", func() {
		migrateDatabaseOrPanic(db)
	})
}

func TestInitRootUserUsesConfiguredInitialPassword(t *testing.T) {
	withInitialRootPassword(t, "customRoot123")
	db := openConnectionTestDB(t)

	CreateUserTable(db)

	if ok, _ := utils.VerifyPassword("customRoot123", fetchRootPasswordHash(t, db)); !ok {
		t.Fatalf("expected configured initial root password to verify")
	}
}

func TestInitRootUserGeneratesPasswordWhenUnset(t *testing.T) {
	withInitialRootPassword(t, "")
	db := openConnectionTestDB(t)

	CreateUserTable(db)

	if ok, _ := utils.VerifyPassword("chatnio123456", fetchRootPasswordHash(t, db)); ok {
		t.Fatalf("expected fixed legacy root password not to be used by default")
	}
}

func TestInitRootUserIgnoresInvalidConfiguredInitialPassword(t *testing.T) {
	withInitialRootPassword(t, "short")
	db := openConnectionTestDB(t)

	CreateUserTable(db)

	if ok, _ := utils.VerifyPassword("short", fetchRootPasswordHash(t, db)); ok {
		t.Fatalf("expected invalid configured root password to be ignored")
	}
}
