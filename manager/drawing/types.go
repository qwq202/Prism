package drawing

import "encoding/json"

const (
	TaskStatusQueued    = "queued"
	TaskStatusRunning   = "running"
	TaskStatusSucceeded = "succeeded"
	TaskStatusFailed    = "failed"
	TaskStatusCanceled  = "canceled"
)

type WorkspaceState struct {
	ActiveWorkspaceID string          `json:"active_workspace_id"`
	Workspaces        json.RawMessage `json:"workspaces"`
	UpdatedAt         string          `json:"updated_at,omitempty"`
}

type GeneratedImage struct {
	ID        string `json:"id"`
	Src       string `json:"src"`
	Prompt    string `json:"prompt"`
	CreatedAt int64  `json:"createdAt"`
}

type TaskOptions struct {
	ResponseFormat json.RawMessage `json:"response_format,omitempty"`
	Thinking       json.RawMessage `json:"thinking,omitempty"`
}

type Task struct {
	ID          int64            `json:"-"`
	TaskID      string           `json:"task_id"`
	UserID      int64            `json:"-"`
	WorkspaceID string           `json:"workspace_id"`
	Status      string           `json:"status"`
	Model       string           `json:"model"`
	Prompt      string           `json:"prompt"`
	Message     string           `json:"-"`
	Options     TaskOptions      `json:"options,omitempty"`
	Images      []GeneratedImage `json:"images,omitempty"`
	Error       string           `json:"error,omitempty"`
	Quota       float64          `json:"quota,omitempty"`
	CreatedAt   string           `json:"created_at,omitempty"`
	UpdatedAt   string           `json:"updated_at,omitempty"`
	StartedAt   string           `json:"started_at,omitempty"`
	CompletedAt string           `json:"completed_at,omitempty"`
}

type saveWorkspaceForm struct {
	ActiveWorkspaceID string          `json:"active_workspace_id"`
	Workspaces        json.RawMessage `json:"workspaces" binding:"required"`
}

type createTaskForm struct {
	WorkspaceID    string          `json:"workspace_id" binding:"required"`
	Model          string          `json:"model" binding:"required"`
	Prompt         string          `json:"prompt" binding:"required"`
	Message        string          `json:"message" binding:"required"`
	ResponseFormat json.RawMessage `json:"response_format,omitempty"`
	Thinking       json.RawMessage `json:"thinking,omitempty"`
}
