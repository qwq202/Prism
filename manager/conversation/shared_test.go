package conversation

import (
	"chat/auth"
	"chat/connection"
	"chat/globals"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func newSharingTestDB(t *testing.T) (*sql.DB, *auth.User) {
	t.Helper()

	previous := globals.SqliteEngine
	globals.SqliteEngine = true
	t.Cleanup(func() {
		globals.SqliteEngine = previous
	})

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "sharing.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	connection.CreateUserTable(db)
	connection.CreateConversationTable(db)
	connection.CreateSharingTable(db)

	return db, &auth.User{ID: 1, Username: "root"}
}

func saveSharingTestConversation(t *testing.T, db *sql.DB, userID int64) *Conversation {
	t.Helper()

	instance := &Conversation{
		Auth:   true,
		UserID: userID,
		Id:     1,
		Name:   "share me",
		Model:  "claude-share-model",
		Message: []globals.Message{
			{Role: globals.User, Content: "question"},
			{Role: globals.Assistant, Content: "answer"},
		},
	}
	if !instance.SaveConversation(db) {
		t.Fatalf("expected conversation to save")
	}
	return instance
}

func TestShareConversationRequiresExistingConversation(t *testing.T) {
	db, user := newSharingTestDB(t)

	hash, err := ShareConversation(db, user, 404, []int{-1})
	if err == nil {
		t.Fatalf("expected missing conversation to fail")
	}
	if hash != "" {
		t.Fatalf("expected no hash for missing conversation, got %q", hash)
	}

	var count int
	if err := globals.QueryRowDb(db, "SELECT COUNT(*) FROM sharing").Scan(&count); err != nil {
		t.Fatalf("count sharing rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no sharing rows for missing conversation, got %d", count)
	}
}

func TestShareConversationRejectsInvalidRefsWithoutSharingAll(t *testing.T) {
	db, user := newSharingTestDB(t)
	saveSharingTestConversation(t, db, user.ID)

	hash, err := ShareConversation(db, user, 1, []int{99, -2})
	if err == nil {
		t.Fatalf("expected invalid refs to fail")
	}
	if hash != "" {
		t.Fatalf("expected no hash for invalid refs, got %q", hash)
	}

	var count int
	if err := globals.QueryRowDb(db, "SELECT COUNT(*) FROM sharing").Scan(&count); err != nil {
		t.Fatalf("count sharing rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected invalid refs not to create sharing rows, got %d", count)
	}
}

func TestShareConversationNormalizesRefsAndPreservesModel(t *testing.T) {
	db, user := newSharingTestDB(t)
	saveSharingTestConversation(t, db, user.ID)

	hash, err := ShareConversation(db, user, 1, []int{1, 1, 99})
	if err != nil {
		t.Fatalf("share conversation: %v", err)
	}
	if hash == "" {
		t.Fatalf("expected share hash")
	}

	shared, err := GetSharedConversation(db, hash)
	if err != nil {
		t.Fatalf("load shared conversation: %v", err)
	}
	if len(shared.Messages) != 1 || shared.Messages[0].Content != "answer" {
		t.Fatalf("expected only the valid selected message, got %#v", shared.Messages)
	}

	imported := UseSharedConversation(db, user, hash)
	if imported == nil {
		t.Fatalf("expected shared conversation import")
	}
	if imported.Model != "claude-share-model" {
		t.Fatalf("expected shared model to be preserved, got %q", imported.Model)
	}
}
