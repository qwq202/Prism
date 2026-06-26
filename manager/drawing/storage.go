package drawing

import (
	"bytes"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"strings"
)

const (
	emptyWorkspaceArray      = "[]"
	maxDrawingWorkspaceCount = 64
	maxDrawingImageBytes     = 100 * 1024 * 1024
)

func normalizeOptionalText(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case []byte:
		return strings.TrimSpace(string(v))
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func emptyWorkspaceState() *WorkspaceState {
	return &WorkspaceState{
		Workspaces: json.RawMessage(emptyWorkspaceArray),
	}
}

func normalizeDrawingImageContentType(value string) (string, error) {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	if err != nil {
		mediaType = strings.TrimSpace(value)
	}

	mediaType = strings.ToLower(mediaType)
	if !strings.HasPrefix(mediaType, "image/") || mediaType == "image/svg+xml" {
		return "", fmt.Errorf("unsupported image type")
	}

	return mediaType, nil
}

func drawingFilenameForContentType(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg", "image/jpg":
		return "drawing.jpg"
	case "image/webp":
		return "drawing.webp"
	case "image/gif":
		return "drawing.gif"
	default:
		return "drawing.png"
	}
}

func storeDrawingDataURL(source string) (string, bool, error) {
	source = strings.TrimSpace(source)
	if !strings.HasPrefix(strings.ToLower(source), "data:image/") {
		return source, false, nil
	}

	parts := utils.SafeSplit(source, ",", 2)
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		return "", true, fmt.Errorf("invalid base64 image")
	}

	contentTypeParts := utils.SafeSplit(parts[0], ";", 2)
	if len(contentTypeParts) == 0 {
		return "", true, fmt.Errorf("invalid image type")
	}

	rawContentType := strings.TrimSpace(contentTypeParts[0])
	if strings.HasPrefix(strings.ToLower(rawContentType), "data:") {
		rawContentType = rawContentType[5:]
	}

	contentType, err := normalizeDrawingImageContentType(rawContentType)
	if err != nil {
		return "", true, err
	}

	data, err := utils.Base64Decode(strings.TrimSpace(parts[1]))
	if err != nil {
		return "", true, err
	}
	if len(data) == 0 {
		return "", true, fmt.Errorf("image data is empty")
	}
	if int64(len(data)) > maxDrawingImageBytes {
		return "", true, fmt.Errorf("image exceeds %dMB storage limit", maxDrawingImageBytes/1024/1024)
	}

	detected := strings.ToLower(http.DetectContentType(data))
	if detected != "application/octet-stream" && !strings.HasPrefix(detected, "image/") {
		return "", true, fmt.Errorf("invalid image data")
	}
	if strings.Contains(detected, "svg") || strings.Contains(strings.ToLower(string(data[:min(len(data), 512)])), "<svg") {
		return "", true, fmt.Errorf("unsupported image type")
	}

	url, err := utils.StoreAttachmentData(drawingFilenameForContentType(contentType), data, contentType)
	if err != nil {
		return "", true, err
	}

	return url, true, nil
}

func normalizeImageCollection(workspace map[string]interface{}, field string, sourceField string) error {
	rawItems, ok := workspace[field].([]interface{})
	if !ok {
		return nil
	}

	for _, rawItem := range rawItems {
		item, ok := rawItem.(map[string]interface{})
		if !ok {
			continue
		}

		source, ok := item[sourceField].(string)
		if !ok || strings.TrimSpace(source) == "" {
			continue
		}

		stored, changed, err := storeDrawingDataURL(source)
		if err != nil {
			return err
		}
		if changed {
			item[sourceField] = stored
		}
	}

	return nil
}

func normalizeWorkspaceSnapshot(raw json.RawMessage) (json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return json.RawMessage(emptyWorkspaceArray), nil
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var workspaces []map[string]interface{}
	if err := decoder.Decode(&workspaces); err != nil {
		return nil, err
	}
	if len(workspaces) > maxDrawingWorkspaceCount {
		return nil, fmt.Errorf("too many drawing workspaces")
	}

	for _, workspace := range workspaces {
		workspace["pending"] = false
		if err := normalizeImageCollection(workspace, "images", "src"); err != nil {
			return nil, err
		}
		if err := normalizeImageCollection(workspace, "references", "content"); err != nil {
			return nil, err
		}
	}

	normalized, err := json.Marshal(workspaces)
	if err != nil {
		return nil, err
	}
	return normalized, nil
}

func LoadWorkspaceState(db *sql.DB, userID int64) (*WorkspaceState, error) {
	if db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	var activeWorkspaceID, data, updatedAt interface{}
	err := globals.QueryRowDb(db, `
		SELECT active_workspace_id, data, updated_at
		FROM drawing_workspace
		WHERE user_id = ?
		LIMIT 1
	`, userID).Scan(&activeWorkspaceID, &data, &updatedAt)
	if err == sql.ErrNoRows {
		return emptyWorkspaceState(), nil
	}
	if err != nil {
		return nil, err
	}

	rawData := strings.TrimSpace(normalizeOptionalText(data))
	if rawData == "" {
		rawData = emptyWorkspaceArray
	}

	return &WorkspaceState{
		ActiveWorkspaceID: normalizeOptionalText(activeWorkspaceID),
		Workspaces:        json.RawMessage(rawData),
		UpdatedAt:         normalizeOptionalText(updatedAt),
	}, nil
}

func SaveWorkspaceState(db *sql.DB, userID int64, state WorkspaceState) (*WorkspaceState, error) {
	if db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	activeWorkspaceID := strings.TrimSpace(state.ActiveWorkspaceID)
	if len(activeWorkspaceID) > 128 {
		return nil, fmt.Errorf("active workspace id is too long")
	}

	normalized, err := normalizeWorkspaceSnapshot(state.Workspaces)
	if err != nil {
		return nil, err
	}

	result, err := globals.ExecDb(db, `
		UPDATE drawing_workspace
		SET active_workspace_id = ?, data = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, activeWorkspaceID, string(normalized), userID)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected > 0 {
		return LoadWorkspaceState(db, userID)
	}

	_, err = globals.ExecDb(db, `
		INSERT INTO drawing_workspace (user_id, active_workspace_id, data)
		VALUES (?, ?, ?)
	`, userID, activeWorkspaceID, string(normalized))
	if err != nil {
		_, updateErr := globals.ExecDb(db, `
			UPDATE drawing_workspace
			SET active_workspace_id = ?, data = ?, updated_at = CURRENT_TIMESTAMP
			WHERE user_id = ?
		`, activeWorkspaceID, string(normalized), userID)
		if updateErr != nil {
			return nil, err
		}
	}

	return LoadWorkspaceState(db, userID)
}
