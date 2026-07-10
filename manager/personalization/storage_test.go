package personalization

import (
	"chat/globals"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func openPersonalizationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	previousSqlite := globals.SqliteEngine
	globals.SqliteEngine = true
	t.Cleanup(func() { globals.SqliteEngine = previousSqlite })

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "personalization.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE personalization_settings (
			user_id INTEGER PRIMARY KEY,
			data TEXT NOT NULL,
			revision INTEGER NOT NULL DEFAULT 1,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func TestSaveUsesRevisionToPreventCrossDeviceOverwrite(t *testing.T) {
	db := openPersonalizationTestDB(t)

	created, err := Save(db, 7, Settings{PersonaStyle: "friendly"}, 0)
	if err != nil {
		t.Fatalf("create settings: %v", err)
	}
	if created.Revision != 1 || created.Settings.PersonaStyle != "friendly" {
		t.Fatalf("unexpected created record: %#v", created)
	}

	updated, err := Save(db, 7, Settings{PersonaStyle: "direct"}, created.Revision)
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if updated.Revision != 2 || updated.Settings.PersonaStyle != "direct" {
		t.Fatalf("unexpected updated record: %#v", updated)
	}

	current, err := Save(db, 7, Settings{PersonaStyle: "creative"}, created.Revision)
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("expected revision conflict, got %v", err)
	}
	if current == nil || current.Revision != 2 || current.Settings.PersonaStyle != "direct" {
		t.Fatalf("expected current server record on conflict, got %#v", current)
	}

	otherUser, err := Load(db, 8)
	if err != nil {
		t.Fatalf("load other user: %v", err)
	}
	if otherUser != nil {
		t.Fatalf("expected settings to be isolated by user, got %#v", otherUser)
	}
}
