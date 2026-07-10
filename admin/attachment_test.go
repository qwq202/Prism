package admin

import (
	"chat/globals"
	"chat/utils"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func setupAttachmentDeleteTest(t *testing.T) (string, string) {
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

func callDeleteAttachmentAPI(t *testing.T, name string, force bool) struct {
	Status         bool   `json:"status"`
	Error          string `json:"error"`
	Referenced     bool   `json:"referenced"`
	ReferenceCount int64  `json:"reference_count"`
} {
	t.Helper()

	db := openAdminUserTestDB(t)
	if _, err := globals.ExecDb(
		db,
		"INSERT INTO conversation (user_id, conversation_id, conversation_name, data, model) VALUES (?, ?, ?, ?, ?)",
		1,
		1,
		"chat",
		`[{"type":"image","image_url":{"url":"/attachments/`+name+`"}}]`,
		"gpt-4o-mini",
	); err != nil {
		t.Fatalf("insert conversation: %v", err)
	}

	url := "/admin/attachment/delete?name=" + name
	if force {
		url += "&force=true"
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, url, nil)
	c.Set("db", db)

	DeleteAttachmentAPI(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body struct {
		Status         bool   `json:"status"`
		Error          string `json:"error"`
		Referenced     bool   `json:"referenced"`
		ReferenceCount int64  `json:"reference_count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return body
}

func TestDeleteAttachmentRejectsReferencedWithoutForce(t *testing.T) {
	gin.SetMode(gin.TestMode)

	workingDir, name := setupAttachmentDeleteTest(t)
	body := callDeleteAttachmentAPI(t, name, false)

	if body.Status || body.Error != "attachment is still referenced" || !body.Referenced || body.ReferenceCount != 1 {
		t.Fatalf("expected referenced attachment rejection, got %#v", body)
	}
	if _, err := os.Stat(filepath.Join(workingDir, utils.AttachmentLocalPath(name))); err != nil {
		t.Fatalf("expected referenced attachment to remain: %v", err)
	}
}

func TestDeleteAttachmentAllowsReferencedWithForce(t *testing.T) {
	gin.SetMode(gin.TestMode)

	workingDir, name := setupAttachmentDeleteTest(t)
	body := callDeleteAttachmentAPI(t, name, true)

	if !body.Status {
		t.Fatalf("expected forced delete to succeed, got %#v", body)
	}
	if _, err := os.Stat(filepath.Join(workingDir, utils.AttachmentLocalPath(name))); !os.IsNotExist(err) {
		t.Fatalf("expected forced delete to remove attachment, got %v", err)
	}
}

func TestDeleteOrphanAttachmentsKeepsReferencedFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	workingDir, referenced := setupAttachmentDeleteTest(t)
	db := openAdminUserTestDB(t)
	if _, err := globals.ExecDb(
		db,
		"INSERT INTO conversation (user_id, conversation_id, conversation_name, data, model) VALUES (?, ?, ?, ?, ?)",
		1,
		1,
		"chat",
		`[{"type":"image","image_url":{"url":"/attachments/`+referenced+`"}}]`,
		"gpt-4o-mini",
	); err != nil {
		t.Fatalf("insert referenced conversation: %v", err)
	}

	orphan := "fedcba9876543210fedcba9876543210.png"
	if err := os.WriteFile(utils.AttachmentLocalPath(orphan), []byte("orphan"), 0o644); err != nil {
		t.Fatalf("write orphan attachment: %v", err)
	}
	drawingReferenced := "00112233445566778899aabbccddeeff.png"
	if err := os.WriteFile(utils.AttachmentLocalPath(drawingReferenced), []byte("drawing"), 0o644); err != nil {
		t.Fatalf("write drawing attachment: %v", err)
	}
	if _, err := globals.ExecDb(
		db,
		"INSERT INTO drawing_workspace (user_id, active_workspace_id, data) VALUES (?, ?, ?)",
		1,
		"workspace-1",
		`[{"id":"workspace-1","images":[{"src":"/attachments/`+drawingReferenced+`"}]}]`,
	); err != nil {
		t.Fatalf("insert drawing workspace: %v", err)
	}

	deleted, err := DeleteOrphanAttachments(db)
	if err != nil {
		t.Fatalf("delete orphan attachments: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected one deleted attachment, got %d", deleted)
	}
	if _, err := os.Stat(filepath.Join(workingDir, utils.AttachmentLocalPath(referenced))); err != nil {
		t.Fatalf("expected referenced attachment to remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workingDir, utils.AttachmentLocalPath(drawingReferenced))); err != nil {
		t.Fatalf("expected drawing attachment to remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workingDir, utils.AttachmentLocalPath(orphan))); !os.IsNotExist(err) {
		t.Fatalf("expected orphan attachment to be deleted, got %v", err)
	}
}

func TestListAttachmentsReturnsRFC3339UpdatedAt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	_, name := setupAttachmentDeleteTest(t)
	modTime := time.Date(2026, 5, 25, 10, 30, 45, 0, time.FixedZone("CST", 8*60*60))
	if err := os.Chtimes(utils.AttachmentLocalPath(name), modTime, modTime); err != nil {
		t.Fatalf("set attachment mod time: %v", err)
	}

	items, err := ListAttachments(openAdminUserTestDB(t))
	if err != nil {
		t.Fatalf("list attachments: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 attachment, got %d (%#v)", len(items), items)
	}

	parsed, err := time.Parse(time.RFC3339, items[0].UpdatedAt)
	if err != nil {
		t.Fatalf("expected RFC3339 updated_at, got %q: %v", items[0].UpdatedAt, err)
	}
	if !parsed.Equal(modTime) {
		t.Fatalf("expected updated_at %s, got %s", modTime.Format(time.RFC3339), items[0].UpdatedAt)
	}
}
