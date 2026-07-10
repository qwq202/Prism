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
	"sync"
	"time"
)

const (
	emptyWorkspaceArray              = "[]"
	maxDrawingWorkspaceCount         = 64
	maxDrawingImageBytes             = 100 * 1024 * 1024
	maxDrawingActiveTasksPerUser     = 4
	maxDrawingTaskWorkspaceIDBytes   = 128
	maxDrawingTaskModelBytes         = 255
	maxDrawingTaskPromptBytes        = 32 * 1024
	maxDrawingTaskMessageBytes       = 8 * 1024 * 1024
	maxDrawingTaskOptionPayloadBytes = 64 * 1024
	drawingTaskCreateLockShardCount  = 64
)

// Task creation is serialized per user (with fixed lock striping) so the active
// task checks and insert behave atomically for concurrent in-process requests.
// Deployments with multiple application processes should additionally enforce
// their concurrency policy in shared infrastructure.
var drawingTaskCreateLocks [drawingTaskCreateLockShardCount]sync.Mutex

func drawingTaskCreateLock(userID int64) *sync.Mutex {
	shard := userID % int64(len(drawingTaskCreateLocks))
	if shard < 0 {
		shard = -shard
	}
	return &drawingTaskCreateLocks[shard]
}

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

func storeDrawingImagesInText(content string) (string, error) {
	for _, image := range utils.ExtractBase64Images(content) {
		stored, changed, err := storeDrawingDataURL(image)
		if err != nil {
			return "", err
		}
		if changed {
			content = strings.ReplaceAll(content, image, stored)
		}
	}
	return content, nil
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
		delete(workspace, "taskId")
		delete(workspace, "taskStatus")
		delete(workspace, "taskError")
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

func activeTaskWorkspaceIDs(raw json.RawMessage) map[string]bool {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil
	}

	var workspaces []map[string]interface{}
	if err := json.Unmarshal(raw, &workspaces); err != nil {
		return nil
	}

	ids := map[string]bool{}
	for _, workspace := range workspaces {
		workspaceID := normalizeOptionalText(workspace["id"])
		if workspaceID == "" {
			continue
		}

		taskStatus := normalizeOptionalText(workspace["taskStatus"])
		pending, _ := workspace["pending"].(bool)
		if pending || taskStatus == TaskStatusQueued || taskStatus == TaskStatusRunning || taskStatus == TaskStatusCanceling {
			ids[workspaceID] = true
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}

func mergeWorkspaceImagesForIDs(raw json.RawMessage, existingRaw json.RawMessage, workspaceIDs map[string]bool) (json.RawMessage, error) {
	if len(workspaceIDs) == 0 {
		return raw, nil
	}

	var incoming []map[string]interface{}
	if err := json.Unmarshal(raw, &incoming); err != nil {
		return nil, err
	}

	var existing []map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(existingRaw), &existing); err != nil {
		return raw, nil
	}

	existingImages := map[string][]interface{}{}
	for _, workspace := range existing {
		workspaceID := normalizeOptionalText(workspace["id"])
		if !workspaceIDs[workspaceID] {
			continue
		}
		images, ok := workspace["images"].([]interface{})
		if ok && len(images) > 0 {
			existingImages[workspaceID] = images
		}
	}
	if len(existingImages) == 0 {
		return raw, nil
	}

	for _, workspace := range incoming {
		workspaceID := normalizeOptionalText(workspace["id"])
		storedImages := existingImages[workspaceID]
		if len(storedImages) == 0 {
			continue
		}

		currentImages, _ := workspace["images"].([]interface{})
		seen := make(map[string]bool, len(currentImages)+len(storedImages))
		for _, rawItem := range currentImages {
			item, ok := rawItem.(map[string]interface{})
			if !ok {
				continue
			}
			if id := normalizeOptionalText(item["id"]); id != "" {
				seen["id:"+id] = true
			}
			if src := normalizeOptionalText(item["src"]); src != "" {
				seen["src:"+src] = true
			}
		}

		for _, rawItem := range storedImages {
			item, ok := rawItem.(map[string]interface{})
			if !ok {
				continue
			}
			id := normalizeOptionalText(item["id"])
			src := normalizeOptionalText(item["src"])
			if id == "" && src == "" {
				continue
			}
			if (id != "" && seen["id:"+id]) || (src != "" && seen["src:"+src]) {
				continue
			}
			currentImages = append(currentImages, rawItem)
			if id != "" {
				seen["id:"+id] = true
			}
			if src != "" {
				seen["src:"+src] = true
			}
		}
		workspace["images"] = currentImages
	}

	next, err := json.Marshal(incoming)
	if err != nil {
		return nil, err
	}
	return next, nil
}

func normalizeJSONRaw(raw json.RawMessage) json.RawMessage {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return raw
}

func normalizeAndValidateTaskOption(name string, raw json.RawMessage) (json.RawMessage, error) {
	raw = normalizeJSONRaw(raw)
	if len(raw) == 0 {
		return nil, nil
	}
	if len(raw) > maxDrawingTaskOptionPayloadBytes {
		return nil, fmt.Errorf("%s is too large", name)
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("%s must be valid JSON", name)
	}

	var object map[string]interface{}
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("%s must be a JSON object", name)
	}
	return raw, nil
}

func normalizeAndValidateCreateTaskForm(form createTaskForm) (createTaskForm, error) {
	form.WorkspaceID = strings.TrimSpace(form.WorkspaceID)
	form.Model = strings.TrimSpace(form.Model)
	form.Prompt = strings.TrimSpace(form.Prompt)
	form.Message = strings.TrimSpace(form.Message)

	if form.WorkspaceID == "" || form.Model == "" || form.Prompt == "" || form.Message == "" {
		return createTaskForm{}, fmt.Errorf("invalid task form")
	}
	if len(form.WorkspaceID) > maxDrawingTaskWorkspaceIDBytes {
		return createTaskForm{}, fmt.Errorf("workspace id is too long")
	}
	if len(form.Model) > maxDrawingTaskModelBytes {
		return createTaskForm{}, fmt.Errorf("model is too long")
	}
	if len(form.Prompt) > maxDrawingTaskPromptBytes {
		return createTaskForm{}, fmt.Errorf("prompt is too long")
	}
	if len(form.Message) > maxDrawingTaskMessageBytes {
		return createTaskForm{}, fmt.Errorf("message is too long")
	}

	var err error
	form.ResponseFormat, err = normalizeAndValidateTaskOption("response_format", form.ResponseFormat)
	if err != nil {
		return createTaskForm{}, err
	}
	form.Thinking, err = normalizeAndValidateTaskOption("thinking", form.Thinking)
	if err != nil {
		return createTaskForm{}, err
	}
	return form, nil
}

func taskOptionsToJSON(options TaskOptions) string {
	payload := map[string]json.RawMessage{}
	if raw := normalizeJSONRaw(options.ResponseFormat); len(raw) > 0 {
		payload["response_format"] = raw
	}
	if raw := normalizeJSONRaw(options.Thinking); len(raw) > 0 {
		payload["thinking"] = raw
	}
	return utils.Marshal(payload)
}

func parseTaskOptions(raw string) TaskOptions {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return TaskOptions{}
	}
	return TaskOptions{
		ResponseFormat: normalizeJSONRaw(payload["response_format"]),
		Thinking:       normalizeJSONRaw(payload["thinking"]),
	}
}

func parseGeneratedImages(raw string) []GeneratedImage {
	var images []GeneratedImage
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(raw), &images); err != nil {
		return nil
	}
	return images
}

func generatedImagesToJSON(images []GeneratedImage) string {
	if len(images) == 0 {
		return "[]"
	}
	return utils.Marshal(images)
}

func taskFromScan(
	id int64,
	taskID interface{},
	userID int64,
	workspaceID interface{},
	status interface{},
	model interface{},
	prompt interface{},
	message interface{},
	options interface{},
	resultImages interface{},
	errText interface{},
	quota sql.NullFloat64,
	createdAt interface{},
	updatedAt interface{},
	startedAt interface{},
	completedAt interface{},
) *Task {
	return &Task{
		ID:          id,
		TaskID:      normalizeOptionalText(taskID),
		UserID:      userID,
		WorkspaceID: normalizeOptionalText(workspaceID),
		Status:      normalizeOptionalText(status),
		Model:       normalizeOptionalText(model),
		Prompt:      normalizeOptionalText(prompt),
		Message:     normalizeOptionalText(message),
		Options:     parseTaskOptions(normalizeOptionalText(options)),
		Images:      parseGeneratedImages(normalizeOptionalText(resultImages)),
		Error:       normalizeOptionalText(errText),
		Quota:       utils.Multi[float64](quota.Valid, quota.Float64, 0),
		CreatedAt:   normalizeOptionalText(createdAt),
		UpdatedAt:   normalizeOptionalText(updatedAt),
		StartedAt:   normalizeOptionalText(startedAt),
		CompletedAt: normalizeOptionalText(completedAt),
	}
}

func CreateTask(db *sql.DB, userID int64, form createTaskForm) (*Task, error) {
	if db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	form, err := normalizeAndValidateCreateTaskForm(form)
	if err != nil {
		return nil, err
	}

	createLock := drawingTaskCreateLock(userID)
	createLock.Lock()
	defer createLock.Unlock()

	active, err := HasActiveTask(db, userID, form.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, fmt.Errorf("workspace already has a running drawing task")
	}
	activeCount, err := countActiveTasks(db, userID)
	if err != nil {
		return nil, err
	}
	if activeCount >= maxDrawingActiveTasksPerUser {
		return nil, fmt.Errorf("too many active drawing tasks")
	}

	storedMessage, err := storeDrawingImagesInText(form.Message)
	if err != nil {
		return nil, err
	}

	options := TaskOptions{
		ResponseFormat: normalizeJSONRaw(form.ResponseFormat),
		Thinking:       normalizeJSONRaw(form.Thinking),
	}
	now := time.Now().Format("20060102150405")
	var lastErr error
	for i := 0; i < 3; i++ {
		taskID := fmt.Sprintf("draw_%s_%s", now, utils.GenerateChar(12))
		_, err := globals.ExecDb(db, `
			INSERT INTO drawing_task (
				task_id, user_id, workspace_id, status, model, prompt,
				message, request_options, result_images
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, taskID, userID, form.WorkspaceID, TaskStatusQueued, form.Model, form.Prompt, storedMessage, taskOptionsToJSON(options), "[]")
		if err == nil {
			return LoadTask(db, userID, taskID)
		}
		lastErr = err
	}
	return nil, lastErr
}

func countActiveTasks(db *sql.DB, userID int64) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("database is not initialized")
	}

	var count int
	err := globals.QueryRowDb(db, `
		SELECT COUNT(*)
		FROM drawing_task
		WHERE user_id = ? AND status IN (?, ?, ?)
	`, userID, TaskStatusQueued, TaskStatusRunning, TaskStatusCanceling).Scan(&count)
	return count, err
}

func HasActiveTask(db *sql.DB, userID int64, workspaceID string) (bool, error) {
	var count int
	err := globals.QueryRowDb(db, `
		SELECT COUNT(*)
		FROM drawing_task
		WHERE user_id = ? AND workspace_id = ? AND status IN (?, ?, ?)
	`, userID, strings.TrimSpace(workspaceID), TaskStatusQueued, TaskStatusRunning, TaskStatusCanceling).Scan(&count)
	return count > 0, err
}

func scanTask(row interface {
	Scan(dest ...interface{}) error
}) (*Task, error) {
	var id, userID int64
	var taskID, workspaceID, status, model, prompt, message, options, resultImages, errText interface{}
	var quota sql.NullFloat64
	var createdAt, updatedAt, startedAt, completedAt interface{}
	err := row.Scan(
		&id,
		&taskID,
		&userID,
		&workspaceID,
		&status,
		&model,
		&prompt,
		&message,
		&options,
		&resultImages,
		&errText,
		&quota,
		&createdAt,
		&updatedAt,
		&startedAt,
		&completedAt,
	)
	if err != nil {
		return nil, err
	}
	return taskFromScan(
		id, taskID, userID, workspaceID, status, model, prompt, message,
		options, resultImages, errText, quota, createdAt, updatedAt, startedAt, completedAt,
	), nil
}

const taskSelectColumns = `
	id, task_id, user_id, workspace_id, status, model, prompt, message,
	request_options, result_images, error, quota, created_at, updated_at,
	started_at, completed_at
`

func LoadTask(db *sql.DB, userID int64, taskID string) (*Task, error) {
	if db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}
	return scanTask(globals.QueryRowDb(db, `
		SELECT `+taskSelectColumns+`
		FROM drawing_task
		WHERE user_id = ? AND task_id = ?
		LIMIT 1
	`, userID, strings.TrimSpace(taskID)))
}

func LoadLatestWorkspaceTasks(db *sql.DB, userID int64) ([]Task, error) {
	if db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	rows, err := globals.QueryDb(db, `
		SELECT `+taskSelectColumns+`
		FROM drawing_task
		WHERE user_id = ?
		  AND id IN (
			SELECT MAX(id)
			FROM drawing_task
			WHERE user_id = ?
			GROUP BY workspace_id
		  )
		ORDER BY id DESC
		LIMIT ?
	`, userID, userID, maxDrawingWorkspaceCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *task)
	}
	return tasks, rows.Err()
}

func IsTaskCancellationRequested(db *sql.DB, taskID string) bool {
	var status interface{}
	err := globals.QueryRowDb(db, `
		SELECT status
		FROM drawing_task
		WHERE task_id = ?
		LIMIT 1
	`, strings.TrimSpace(taskID)).Scan(&status)
	if err != nil {
		return false
	}
	value := normalizeOptionalText(status)
	return value == TaskStatusCanceling || value == TaskStatusCanceled
}

func MarkTaskRunning(db *sql.DB, taskID string) error {
	_, err := globals.ExecDb(db, `
		UPDATE drawing_task
		SET status = ?, started_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE task_id = ? AND status = ?
	`, TaskStatusRunning, strings.TrimSpace(taskID), TaskStatusQueued)
	return err
}

func MarkTaskSucceeded(db *sql.DB, taskID string, images []GeneratedImage, quota float64) error {
	_, err := globals.ExecDb(db, `
		UPDATE drawing_task
		SET status = ?, result_images = ?, error = '', quota = ?,
		    completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE task_id = ? AND status NOT IN (?, ?)
	`, TaskStatusSucceeded, generatedImagesToJSON(images), quota, strings.TrimSpace(taskID), TaskStatusCanceled, TaskStatusCanceling)
	return err
}

func MarkTaskFailed(db *sql.DB, taskID string, err error) error {
	message := ""
	if err != nil {
		message = err.Error()
	}
	_, execErr := globals.ExecDb(db, `
		UPDATE drawing_task
		SET status = ?, error = ?, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE task_id = ? AND status NOT IN (?, ?)
	`, TaskStatusFailed, message, strings.TrimSpace(taskID), TaskStatusCanceled, TaskStatusCanceling)
	return execErr
}

func RequestTaskCancellation(db *sql.DB, userID int64, taskID string) (*Task, error) {
	if db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("invalid task id")
	}

	result, err := globals.ExecDb(db, `
		UPDATE drawing_task
		SET status = ?, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND task_id = ? AND status = ?
	`, TaskStatusCanceled, userID, taskID, TaskStatusQueued)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		_, err = globals.ExecDb(db, `
			UPDATE drawing_task
			SET status = ?, completed_at = NULL, updated_at = CURRENT_TIMESTAMP
			WHERE user_id = ? AND task_id = ? AND status = ?
		`, TaskStatusCanceling, userID, taskID, TaskStatusRunning)
		if err != nil {
			return nil, err
		}
	}
	return LoadTask(db, userID, taskID)
}

func FinalizeTaskCancellation(db *sql.DB, taskID string) error {
	if db == nil {
		return fmt.Errorf("database is not initialized")
	}
	_, err := globals.ExecDb(db, `
		UPDATE drawing_task
		SET status = ?, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE task_id = ? AND status = ?
	`, TaskStatusCanceled, strings.TrimSpace(taskID), TaskStatusCanceling)
	return err
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

	activeTaskWorkspaces := activeTaskWorkspaceIDs(state.Workspaces)
	normalized, err := normalizeWorkspaceSnapshot(state.Workspaces)
	if err != nil {
		return nil, err
	}

	existingState, existingErr := LoadWorkspaceState(db, userID)
	if existingErr != nil && existingErr != sql.ErrNoRows {
		return nil, existingErr
	}
	if existingState != nil {
		normalized, err = mergeWorkspaceImagesForIDs(normalized, existingState.Workspaces, activeTaskWorkspaces)
		if err != nil {
			return nil, err
		}
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

func appendImagesToWorkspaceSnapshot(raw json.RawMessage, workspaceID string, model string, images []GeneratedImage, prompt string) (json.RawMessage, bool, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" || len(images) == 0 {
		return raw, false, nil
	}

	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		raw = json.RawMessage(emptyWorkspaceArray)
	}

	var workspaces []map[string]interface{}
	if err := json.Unmarshal(raw, &workspaces); err != nil {
		return nil, false, err
	}

	changed := false
	found := false
	for _, workspace := range workspaces {
		if normalizeOptionalText(workspace["id"]) != workspaceID {
			continue
		}
		found = true

		existing, _ := workspace["images"].([]interface{})
		seen := make(map[string]bool, len(existing))
		for _, rawItem := range existing {
			item, ok := rawItem.(map[string]interface{})
			if !ok {
				continue
			}
			id := normalizeOptionalText(item["id"])
			src := normalizeOptionalText(item["src"])
			if id != "" {
				seen["id:"+id] = true
			}
			if src != "" {
				seen["src:"+src] = true
			}
		}

		for _, image := range images {
			if image.ID == "" || image.Src == "" {
				continue
			}
			if seen["id:"+image.ID] || seen["src:"+image.Src] {
				continue
			}
			storedImage := map[string]interface{}{
				"id":        image.ID,
				"src":       image.Src,
				"prompt":    image.Prompt,
				"createdAt": image.CreatedAt,
			}
			if strings.TrimSpace(image.Model) != "" {
				storedImage["model"] = image.Model
			}
			if image.Options != nil {
				storedImage["options"] = image.Options
			}
			existing = append(existing, storedImage)
			seen["id:"+image.ID] = true
			seen["src:"+image.Src] = true
			changed = true
		}

		workspace["images"] = existing
		workspace["pending"] = false
		delete(workspace, "taskId")
		delete(workspace, "taskStatus")
		delete(workspace, "taskError")
		if strings.TrimSpace(prompt) != "" {
			workspace["lastPrompt"] = prompt
		}
		break
	}

	if !found {
		workspace := map[string]interface{}{
			"id":         workspaceID,
			"model":      strings.TrimSpace(model),
			"mode":       "generate",
			"prompt":     "",
			"options":    map[string]interface{}{},
			"references": []interface{}{},
			"images":     []interface{}{},
			"pending":    false,
			"lastPrompt": prompt,
			"createdAt":  time.Now().UnixMilli(),
			"accent":     len(workspaces),
		}
		workspaces = append(workspaces, workspace)
		next, _, err := appendImagesToWorkspaceSnapshot(mustMarshalRaw(workspaces), workspaceID, model, images, prompt)
		return next, true, err
	}

	if !changed {
		return raw, false, nil
	}

	next, err := json.Marshal(workspaces)
	if err != nil {
		return nil, false, err
	}
	return next, true, nil
}

func mustMarshalRaw(value interface{}) json.RawMessage {
	return json.RawMessage(utils.Marshal(value))
}

func AppendImagesToWorkspace(db *sql.DB, userID int64, workspaceID string, model string, images []GeneratedImage, prompt string) error {
	if db == nil {
		return fmt.Errorf("database is not initialized")
	}
	if len(images) == 0 {
		return nil
	}

	state, err := LoadWorkspaceState(db, userID)
	if err != nil {
		return err
	}

	next, changed, err := appendImagesToWorkspaceSnapshot(state.Workspaces, workspaceID, model, images, prompt)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	_, err = SaveWorkspaceState(db, userID, WorkspaceState{
		ActiveWorkspaceID: state.ActiveWorkspaceID,
		Workspaces:        next,
	})
	return err
}
