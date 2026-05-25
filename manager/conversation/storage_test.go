package conversation

import (
	"chat/connection"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupOrphanAttachmentCleanupTest(t *testing.T) (string, string) {
	t.Helper()

	previousWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	workingDir := t.TempDir()
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWorkingDir)
	})

	previousStorageMode := globals.StorageMode
	globals.StorageMode = "local"
	t.Cleanup(func() {
		globals.StorageMode = previousStorageMode
	})

	name := "0123456789abcdef0123456789abcdef.png"
	if err := os.MkdirAll(filepath.Dir(utils.AttachmentLocalPath(name)), 0o755); err != nil {
		t.Fatalf("create attachment dir: %v", err)
	}
	if err := os.WriteFile(utils.AttachmentLocalPath(name), []byte("png"), 0o644); err != nil {
		t.Fatalf("write attachment: %v", err)
	}

	return workingDir, name
}

func openOrphanAttachmentCleanupDB(t *testing.T) *sql.DB {
	t.Helper()

	previous := globals.SqliteEngine
	globals.SqliteEngine = true
	t.Cleanup(func() {
		globals.SqliteEngine = previous
	})

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "conversation-storage.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	connection.CreateConversationTable(db)
	return db
}

func TestCleanupOrphanStoredAttachmentsSkipsWhenReferencesCannotLoad(t *testing.T) {
	workingDir, name := setupOrphanAttachmentCleanupTest(t)
	db := openOrphanAttachmentCleanupDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	cleanupOrphanStoredAttachments(db)

	if _, err := os.Stat(filepath.Join(workingDir, utils.AttachmentLocalPath(name))); err != nil {
		t.Fatalf("expected attachment to remain when references cannot load: %v", err)
	}
}

func TestCleanupOrphanStoredAttachmentsKeepsReferencedAndDeletesOrphan(t *testing.T) {
	workingDir, referenced := setupOrphanAttachmentCleanupTest(t)
	db := openOrphanAttachmentCleanupDB(t)

	orphan := "fedcba9876543210fedcba9876543210.png"
	if err := os.WriteFile(utils.AttachmentLocalPath(orphan), []byte("orphan"), 0o644); err != nil {
		t.Fatalf("write orphan attachment: %v", err)
	}
	if _, err := globals.ExecDb(
		db,
		"INSERT INTO conversation (user_id, conversation_id, conversation_name, data, model) VALUES (?, ?, ?, ?, ?)",
		1,
		1,
		"chat",
		`[{"type":"image","image_url":{"url":"/attachments/`+referenced+`"}}]`,
		globals.GPT3Turbo,
	); err != nil {
		t.Fatalf("insert referenced conversation: %v", err)
	}

	cleanupOrphanStoredAttachments(db)

	if _, err := os.Stat(filepath.Join(workingDir, utils.AttachmentLocalPath(referenced))); err != nil {
		t.Fatalf("expected referenced attachment to remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workingDir, utils.AttachmentLocalPath(orphan))); !os.IsNotExist(err) {
		t.Fatalf("expected orphan attachment to be deleted, got %v", err)
	}
}
