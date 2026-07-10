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
	connection.CreateDrawingWorkspaceTable(db)
	connection.CreateDrawingTaskTable(db)
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

func TestCleanupOrphanStoredAttachmentsTracksDrawingReferences(t *testing.T) {
	tests := []struct {
		name      string
		insertRef func(t *testing.T, db *sql.DB, attachmentName string)
		deleteRef func(t *testing.T, db *sql.DB)
	}{
		{
			name: "workspace data",
			insertRef: func(t *testing.T, db *sql.DB, attachmentName string) {
				t.Helper()
				if _, err := globals.ExecDb(
					db,
					"INSERT INTO drawing_workspace (user_id, active_workspace_id, data) VALUES (?, ?, ?)",
					1,
					"workspace-1",
					`[{"id":"workspace-1","images":[{"src":"/attachments/`+attachmentName+`"}]}]`,
				); err != nil {
					t.Fatalf("insert drawing workspace reference: %v", err)
				}
			},
			deleteRef: func(t *testing.T, db *sql.DB) {
				t.Helper()
				if _, err := globals.ExecDb(db, "DELETE FROM drawing_workspace"); err != nil {
					t.Fatalf("delete drawing workspace reference: %v", err)
				}
			},
		},
		{
			name: "task message",
			insertRef: func(t *testing.T, db *sql.DB, attachmentName string) {
				t.Helper()
				if _, err := globals.ExecDb(
					db,
					`INSERT INTO drawing_task (
						task_id, user_id, workspace_id, status, model, message, result_images
					) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					"task-1",
					1,
					"workspace-1",
					"running",
					"image-model",
					"```file\n[[reference.png]]\n/attachments/"+attachmentName+"\n```",
					"[]",
				); err != nil {
					t.Fatalf("insert drawing task message reference: %v", err)
				}
			},
			deleteRef: func(t *testing.T, db *sql.DB) {
				t.Helper()
				if _, err := globals.ExecDb(db, "DELETE FROM drawing_task"); err != nil {
					t.Fatalf("delete drawing task message reference: %v", err)
				}
			},
		},
		{
			name: "task result images",
			insertRef: func(t *testing.T, db *sql.DB, attachmentName string) {
				t.Helper()
				if _, err := globals.ExecDb(
					db,
					`INSERT INTO drawing_task (
						task_id, user_id, workspace_id, status, model, message, result_images
					) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					"task-1",
					1,
					"workspace-1",
					"succeeded",
					"image-model",
					"draw a cat",
					`[{"src":"/attachments/`+attachmentName+`"}]`,
				); err != nil {
					t.Fatalf("insert drawing task result reference: %v", err)
				}
			},
			deleteRef: func(t *testing.T, db *sql.DB) {
				t.Helper()
				if _, err := globals.ExecDb(db, "DELETE FROM drawing_task"); err != nil {
					t.Fatalf("delete drawing task result reference: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workingDir, attachmentName := setupOrphanAttachmentCleanupTest(t)
			db := openOrphanAttachmentCleanupDB(t)
			attachmentPath := filepath.Join(workingDir, utils.AttachmentLocalPath(attachmentName))

			tt.insertRef(t, db, attachmentName)
			cleanupOrphanStoredAttachments(db)
			if _, err := os.Stat(attachmentPath); err != nil {
				t.Fatalf("expected drawing-only attachment to remain: %v", err)
			}

			tt.deleteRef(t, db)
			cleanupOrphanStoredAttachments(db)
			if _, err := os.Stat(attachmentPath); !os.IsNotExist(err) {
				t.Fatalf("expected unreferenced drawing attachment to be deleted, got %v", err)
			}
		})
	}
}

func TestConversationFavoritePersistsInDetailAndList(t *testing.T) {
	db := openOrphanAttachmentCleanupDB(t)

	conversation := &Conversation{
		UserID:    1,
		Id:        1,
		Name:      "favorite chat",
		Model:     globals.GPT3Turbo,
		Message:   []globals.Message{},
		Persisted: false,
	}
	if !conversation.SaveConversation(db) {
		t.Fatalf("save conversation")
	}
	if !conversation.SetFavorite(db, true) {
		t.Fatalf("set conversation favorite")
	}

	loaded := LoadConversation(db, 1, 1)
	if loaded == nil {
		t.Fatalf("load conversation")
	}
	if !loaded.Favorite {
		t.Fatalf("expected loaded conversation to be favorite")
	}

	list := LoadConversationList(db, 1)
	if len(list) != 1 {
		t.Fatalf("expected one conversation, got %d", len(list))
	}
	if !list[0].Favorite {
		t.Fatalf("expected listed conversation to be favorite")
	}
}
