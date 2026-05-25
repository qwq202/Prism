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
