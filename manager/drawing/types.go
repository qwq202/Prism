package drawing

import "encoding/json"

type WorkspaceState struct {
	ActiveWorkspaceID string          `json:"active_workspace_id"`
	Workspaces        json.RawMessage `json:"workspaces"`
	UpdatedAt         string          `json:"updated_at,omitempty"`
}

type saveWorkspaceForm struct {
	ActiveWorkspaceID string          `json:"active_workspace_id"`
	Workspaces        json.RawMessage `json:"workspaces" binding:"required"`
}
