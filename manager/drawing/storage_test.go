package drawing

import (
	"chat/globals"
	"chat/utils"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
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
	_, err = globals.ExecDb(db, `
		CREATE TABLE IF NOT EXISTS drawing_task (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  task_id VARCHAR(64) UNIQUE,
		  user_id INT NOT NULL,
		  workspace_id VARCHAR(128) NOT NULL,
		  status VARCHAR(32) NOT NULL,
		  model VARCHAR(255) NOT NULL,
		  prompt TEXT,
		  message MEDIUMTEXT,
		  request_options MEDIUMTEXT,
		  result_images MEDIUMTEXT,
		  error TEXT,
		  quota DECIMAL(24, 6) DEFAULT 0,
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  started_at DATETIME NULL,
		  completed_at DATETIME NULL
		);
	`)
	if err != nil {
		t.Fatalf("create drawing task table: %v", err)
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
			"taskId": "task-1",
			"taskStatus": "running",
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
	if strings.Contains(payload, "taskStatus") || strings.Contains(payload, "taskId") {
		t.Fatalf("expected task fields to be cleared before persistence, got %s", payload)
	}

	loaded, err := LoadWorkspaceState(db, 7)
	if err != nil {
		t.Fatalf("load workspace state: %v", err)
	}
	if string(loaded.Workspaces) != string(state.Workspaces) {
		t.Fatalf("expected saved workspace payload to load back unchanged")
	}
}

func TestCreateTaskStoresMessageImagesAsAttachments(t *testing.T) {
	t.Chdir(t.TempDir())
	db := openDrawingWorkspaceTestDB(t)

	dataURL := tinyPNGDataURL(t)
	task, err := CreateTask(db, 7, createTaskForm{
		WorkspaceID: "workspace-1",
		Model:       "gemini-3-pro-image",
		Prompt:      "pig",
		Message:     "```file\n[[ref.png]]\n" + dataURL + "\n```\n\npig",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if strings.Contains(task.Message, "data:image/") {
		t.Fatalf("expected task message image data to be stored as attachment, got %s", task.Message)
	}
	if !strings.Contains(task.Message, "/attachments/") {
		t.Fatalf("expected task message to reference attachment, got %s", task.Message)
	}
}

func TestAppendImagesToWorkspaceMergesTaskImages(t *testing.T) {
	t.Chdir(t.TempDir())
	db := openDrawingWorkspaceTestDB(t)

	_, err := SaveWorkspaceState(db, 7, WorkspaceState{
		ActiveWorkspaceID: "workspace-1",
		Workspaces: json.RawMessage(`[
			{
				"id": "workspace-1",
				"model": "gemini-3-pro-image",
				"pending": true,
				"taskId": "task-1",
				"taskStatus": "running",
				"images": []
			}
	]`),
	})
	if err != nil {
		t.Fatalf("save workspace: %v", err)
	}

	err = AppendImagesToWorkspace(db, 7, "workspace-1", "gemini-3-pro-image", []GeneratedImage{
		{
			ID:        "task-1-0",
			Src:       "/attachments/image.png",
			Prompt:    "pig",
			CreatedAt: 123,
		},
	}, "pig")
	if err != nil {
		t.Fatalf("append images: %v", err)
	}

	loaded, err := LoadWorkspaceState(db, 7)
	if err != nil {
		t.Fatalf("load workspace: %v", err)
	}
	payload := string(loaded.Workspaces)
	if !strings.Contains(payload, "task-1-0") {
		t.Fatalf("expected task image to be merged, got %s", payload)
	}
	if strings.Contains(payload, `"pending":true`) {
		t.Fatalf("expected pending to be cleared, got %s", payload)
	}
	if strings.Contains(payload, "taskStatus") || strings.Contains(payload, "taskId") {
		t.Fatalf("expected completed task fields to be cleared, got %s", payload)
	}
}

func TestAppendImagesToWorkspaceCreatesMissingWorkspace(t *testing.T) {
	t.Chdir(t.TempDir())
	db := openDrawingWorkspaceTestDB(t)

	err := AppendImagesToWorkspace(db, 7, "workspace-late", "gemini-3-pro-image", []GeneratedImage{
		{
			ID:        "task-1-0",
			Src:       "/attachments/image.png",
			Prompt:    "pig",
			CreatedAt: 123,
		},
	}, "pig")
	if err != nil {
		t.Fatalf("append images: %v", err)
	}

	loaded, err := LoadWorkspaceState(db, 7)
	if err != nil {
		t.Fatalf("load workspace: %v", err)
	}
	payload := string(loaded.Workspaces)
	if !strings.Contains(payload, "workspace-late") || !strings.Contains(payload, "task-1-0") {
		t.Fatalf("expected missing workspace to be created with task image, got %s", payload)
	}
	if !strings.Contains(payload, "gemini-3-pro-image") {
		t.Fatalf("expected created workspace to keep model, got %s", payload)
	}
}

func TestSaveWorkspaceStatePreservesServerImagesForActiveTaskSnapshot(t *testing.T) {
	t.Chdir(t.TempDir())
	db := openDrawingWorkspaceTestDB(t)

	_, err := SaveWorkspaceState(db, 7, WorkspaceState{
		ActiveWorkspaceID: "workspace-1",
		Workspaces: json.RawMessage(`[
			{
				"id": "workspace-1",
				"model": "gemini-3-pro-image",
				"pending": false,
				"images": [{"id": "task-1-0", "src": "/attachments/image.png", "prompt": "pig", "createdAt": 123}]
			}
		]`),
	})
	if err != nil {
		t.Fatalf("seed workspace: %v", err)
	}

	_, err = SaveWorkspaceState(db, 7, WorkspaceState{
		ActiveWorkspaceID: "workspace-1",
		Workspaces: json.RawMessage(`[
			{
				"id": "workspace-1",
				"model": "gemini-3-pro-image",
				"pending": true,
				"taskId": "task-1",
				"taskStatus": "running",
				"images": []
			}
		]`),
	})
	if err != nil {
		t.Fatalf("save stale active task snapshot: %v", err)
	}

	loaded, err := LoadWorkspaceState(db, 7)
	if err != nil {
		t.Fatalf("load workspace: %v", err)
	}
	payload := string(loaded.Workspaces)
	if !strings.Contains(payload, "task-1-0") {
		t.Fatalf("expected existing server image to be preserved, got %s", payload)
	}
	if strings.Contains(payload, "taskStatus") || strings.Contains(payload, "taskId") {
		t.Fatalf("expected transient task fields to be cleared, got %s", payload)
	}
}

func validDrawingTaskForm(workspaceID string) createTaskForm {
	return createTaskForm{
		WorkspaceID:    workspaceID,
		Model:          "gemini-3-pro-image",
		Prompt:         "draw a pig",
		Message:        "draw a pig",
		ResponseFormat: json.RawMessage(`{"type":"image","aspect_ratio":"1:1"}`),
		Thinking:       json.RawMessage(`{"thinking_level":"minimal"}`),
	}
}

func TestNormalizeAndValidateCreateTaskFormRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*createTaskForm)
		wantError string
	}{
		{
			name: "workspace id too long",
			mutate: func(form *createTaskForm) {
				form.WorkspaceID = strings.Repeat("w", maxDrawingTaskWorkspaceIDBytes+1)
			},
			wantError: "workspace id is too long",
		},
		{
			name: "model too long",
			mutate: func(form *createTaskForm) {
				form.Model = strings.Repeat("m", maxDrawingTaskModelBytes+1)
			},
			wantError: "model is too long",
		},
		{
			name: "prompt too long",
			mutate: func(form *createTaskForm) {
				form.Prompt = strings.Repeat("p", maxDrawingTaskPromptBytes+1)
			},
			wantError: "prompt is too long",
		},
		{
			name: "message too long",
			mutate: func(form *createTaskForm) {
				form.Message = strings.Repeat("x", maxDrawingTaskMessageBytes+1)
			},
			wantError: "message is too long",
		},
		{
			name: "invalid response format json",
			mutate: func(form *createTaskForm) {
				form.ResponseFormat = json.RawMessage(`{"type":`)
			},
			wantError: "response_format must be valid JSON",
		},
		{
			name: "response format must be object",
			mutate: func(form *createTaskForm) {
				form.ResponseFormat = json.RawMessage(`[]`)
			},
			wantError: "response_format must be a JSON object",
		},
		{
			name: "thinking must be object",
			mutate: func(form *createTaskForm) {
				form.Thinking = json.RawMessage(`"high"`)
			},
			wantError: "thinking must be a JSON object",
		},
		{
			name: "option payload too large",
			mutate: func(form *createTaskForm) {
				form.ResponseFormat = json.RawMessage(`{"value":"` + strings.Repeat("x", maxDrawingTaskOptionPayloadBytes) + `"}`)
			},
			wantError: "response_format is too large",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			form := validDrawingTaskForm("workspace-1")
			test.mutate(&form)

			_, err := normalizeAndValidateCreateTaskForm(form)
			if err == nil || err.Error() != test.wantError {
				t.Fatalf("expected %q, got %v", test.wantError, err)
			}
		})
	}
}

func TestCreateTaskSerializesConcurrentRequestsForSameWorkspace(t *testing.T) {
	db := openDrawingWorkspaceTestDB(t)

	const attempts = 12
	start := make(chan struct{})
	errors := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := CreateTask(db, 7, validDrawingTaskForm("workspace-1"))
			errors <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errors)

	succeeded := 0
	for err := range errors {
		if err == nil {
			succeeded++
			continue
		}
		if err.Error() != "workspace already has a running drawing task" {
			t.Fatalf("unexpected create error: %v", err)
		}
	}
	if succeeded != 1 {
		t.Fatalf("expected exactly one task to be created, got %d", succeeded)
	}

	count, err := countActiveTasks(db, 7)
	if err != nil {
		t.Fatalf("count active tasks: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one active task, got %d", count)
	}
}

func TestCreateTaskLimitsConcurrentTasksPerUser(t *testing.T) {
	db := openDrawingWorkspaceTestDB(t)

	attempts := maxDrawingActiveTasksPerUser + 8
	start := make(chan struct{})
	errors := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		workspaceID := fmt.Sprintf("workspace-%d", i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := CreateTask(db, 7, validDrawingTaskForm(workspaceID))
			errors <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errors)

	succeeded := 0
	for err := range errors {
		if err == nil {
			succeeded++
			continue
		}
		if err.Error() != "too many active drawing tasks" {
			t.Fatalf("unexpected create error: %v", err)
		}
	}
	if succeeded != maxDrawingActiveTasksPerUser {
		t.Fatalf("expected %d tasks to be created, got %d", maxDrawingActiveTasksPerUser, succeeded)
	}

	count, err := countActiveTasks(db, 7)
	if err != nil {
		t.Fatalf("count active tasks: %v", err)
	}
	if count != maxDrawingActiveTasksPerUser {
		t.Fatalf("expected %d active tasks, got %d", maxDrawingActiveTasksPerUser, count)
	}

	if _, err := CreateTask(db, 8, validDrawingTaskForm("workspace-other-user")); err != nil {
		t.Fatalf("expected another user to have an independent task limit: %v", err)
	}
}
