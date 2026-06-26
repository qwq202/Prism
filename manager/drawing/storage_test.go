package drawing

import (
	"chat/globals"
	"chat/utils"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func openDrawingWorkspaceTestDB(t *testing.T) *sql.DB {
	t.Helper()

	previousSqlite := globals.SqliteEngine
	globals.SqliteEngine = true
	t.Cleanup(func() {
		globals.SqliteEngine = previousSqlite
	})

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "drawing.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = globals.ExecDb(db, `
		CREATE TABLE IF NOT EXISTS drawing_workspace (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  user_id INT UNIQUE,
		  active_workspace_id VARCHAR(128),
		  data MEDIUMTEXT,
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatalf("create drawing workspace table: %v", err)
	}

	return db
}

func tinyPNGDataURL(t *testing.T) string {
	t.Helper()

	data, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}

	return "data:image/png;base64," + utils.Base64EncodeBytes(data)
}

func TestSaveWorkspaceStateStoresDataURLImagesAsAttachments(t *testing.T) {
	t.Chdir(t.TempDir())
	db := openDrawingWorkspaceTestDB(t)

	dataURL := tinyPNGDataURL(t)
	rawWorkspaces := json.RawMessage(`[
		{
			"id": "workspace-1",
			"model": "gemini-3-pro-image",
			"pending": true,
			"references": [{"name": "ref.png", "content": "` + dataURL + `"}],
			"images": [{"id": "image-1", "src": "` + dataURL + `", "prompt": "pig", "createdAt": 1}]
		}
	]`)

	state, err := SaveWorkspaceState(db, 7, WorkspaceState{
		ActiveWorkspaceID: "workspace-1",
		Workspaces:        rawWorkspaces,
	})
	if err != nil {
		t.Fatalf("save workspace state: %v", err)
	}

	payload := string(state.Workspaces)
	if strings.Contains(payload, "data:image/") {
		t.Fatalf("expected data URLs to be stored as attachments, got %s", payload)
	}
	if !strings.Contains(payload, "/attachments/") {
		t.Fatalf("expected attachment URLs in workspace payload, got %s", payload)
	}
	if strings.Contains(payload, `"pending":true`) {
		t.Fatalf("expected pending state to be cleared before persistence, got %s", payload)
	}

	loaded, err := LoadWorkspaceState(db, 7)
	if err != nil {
		t.Fatalf("load workspace state: %v", err)
	}
	if string(loaded.Workspaces) != string(state.Workspaces) {
		t.Fatalf("expected saved workspace payload to load back unchanged")
	}
}
