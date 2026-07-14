package conversation

import (
	"chat/auth"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"
)

var orphanCleanupState struct {
	once    sync.Once
	mu      sync.Mutex
	lastRun time.Time
	running bool
}

const saveConversationQuery = `
		INSERT INTO conversation (user_id, conversation_id, conversation_name, data, model, task_id) VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE conversation_name = VALUES(conversation_name), data = VALUES(data), model = VALUES(model), task_id = VALUES(task_id), updated_at = CURRENT_TIMESTAMP
	`

const insertConversationQuery = `
		INSERT INTO conversation (user_id, conversation_id, conversation_name, data, model, task_id) VALUES (?, ?, ?, ?, ?, ?)
	`

func normalizeDBString(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case []byte:
		return string(v)
	case string:
		return v
	case time.Time:
		return v.Format(time.DateTime)
	default:
		return fmt.Sprint(v)
	}
}

func normalizeDBBool(value interface{}) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case int64:
		return v != 0
	case int:
		return v != 0
	case []byte:
		value = string(v)
	case string:
	default:
		return strings.EqualFold(fmt.Sprint(v), "true") || fmt.Sprint(v) == "1"
	}

	text := strings.TrimSpace(strings.ToLower(fmt.Sprint(value)))
	return text == "1" || text == "true" || text == "t" || text == "yes"
}

func isDuplicateConversationIDError(err error) bool {
	if err == nil {
		return false
	}

	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "error 1062") ||
		strings.Contains(lower, "duplicate entry") ||
		strings.Contains(lower, "unique constraint failed")
}

func loadGlobalAttachmentNames(db *sql.DB) (map[string]struct{}, error) {
	if db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	rows, err := globals.QueryDb(db, `
		SELECT data FROM conversation
		UNION ALL
		SELECT data FROM drawing_workspace
		UNION ALL
		SELECT message FROM drawing_task
		UNION ALL
		SELECT result_images FROM drawing_task
	`)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	result := map[string]struct{}{}
	for rows.Next() {
		var data sql.NullString
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		if !data.Valid {
			continue
		}

		for _, name := range utils.ExtractAttachmentNames(data.String) {
			result[name] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func cleanupOrphanStoredAttachments(db *sql.DB) {
	referenced, err := loadGlobalAttachmentNames(db)
	if err != nil {
		globals.Warn(fmt.Sprintf("[conversation] load attachment references error: %s", err.Error()))
		return
	}

	storedNames, err := utils.ListStoredAttachmentNames()
	if err != nil {
		globals.Warn(fmt.Sprintf("[conversation] list stored attachments error: %s", err.Error()))
		return
	}

	for _, name := range storedNames {
		if _, ok := referenced[name]; ok {
			continue
		}

		if err := utils.DeleteStoredAttachment(name); err != nil {
			globals.Warn(fmt.Sprintf("[conversation] delete orphan attachment error: %s (attachment: %s)", err.Error(), name))
		}
	}
}

func ResetOrphanAttachmentCleanupSchedule() {
	orphanCleanupState.mu.Lock()
	defer orphanCleanupState.mu.Unlock()
	orphanCleanupState.lastRun = time.Time{}
}

func RunOrphanAttachmentCleanup(db *sql.DB) {
	orphanCleanupState.mu.Lock()
	if orphanCleanupState.running {
		orphanCleanupState.mu.Unlock()
		return
	}
	orphanCleanupState.running = true
	orphanCleanupState.lastRun = time.Now()
	orphanCleanupState.mu.Unlock()

	defer func() {
		orphanCleanupState.mu.Lock()
		orphanCleanupState.running = false
		orphanCleanupState.mu.Unlock()
	}()

	cleanupOrphanStoredAttachments(db)
}

func StartOrphanAttachmentCleanupWorker(db *sql.DB) {
	if db == nil {
		return
	}

	orphanCleanupState.once.Do(func() {
		go func() {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()

			for range ticker.C {
				if !globals.OrphanCleanupEnabled {
					continue
				}

				interval := globals.OrphanCleanupInterval
				if interval <= 0 {
					interval = 60
				}

				orphanCleanupState.mu.Lock()
				lastRun := orphanCleanupState.lastRun
				running := orphanCleanupState.running
				orphanCleanupState.mu.Unlock()

				if running {
					continue
				}

				if !lastRun.IsZero() && time.Since(lastRun) < time.Duration(interval)*time.Minute {
					continue
				}

				RunOrphanAttachmentCleanup(db)
			}
		}()
	})
}

func (c *Conversation) SaveConversation(db *sql.DB) bool {
	if c.UserID == -1 {
		// anonymous request
		return true
	}

	data := utils.ToJson(c.GetMessage())
	var taskID sql.NullString
	if c.TaskID != "" {
		taskID = sql.NullString{String: c.TaskID, Valid: true}
	}

	if !c.Persisted {
		return c.insertNewConversation(db, data, taskID)
	}

	stmt, err := globals.PrepareDb(db, saveConversationQuery)
	if err != nil {
		return false
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			globals.Warn(err)
		}
	}(stmt)

	_, err = stmt.Exec(c.UserID, c.Id, c.Name, data, c.Model, taskID)
	if err != nil {
		globals.Info(fmt.Sprintf("execute error during save conversation: %s", err.Error()))
		return false
	}
	return true
}

// HandleNewMessageWithRequest atomically persists the first user message and
// binds the idempotency record to the final conversation id. This closes the
// allocation race where insertNewConversation may have to change c.Id after a
// duplicate-key conflict.
func (c *Conversation) HandleNewMessageWithRequest(db *sql.DB, form *FormMessage, ownerToken string) bool {
	if db == nil || c.UserID < 0 || c.Persisted || strings.TrimSpace(form.RequestID) == "" || strings.TrimSpace(ownerToken) == "" {
		return false
	}

	previousName := c.Name
	previousLength := len(c.Message)
	if err := c.AddMessageFromForm(form); err != nil {
		return false
	}
	if previousLength == 0 || c.Name == defaultConversationName {
		c.Name = utils.Extract(form.Message, 50, "...")
	}
	data := utils.ToJson(c.GetMessage())
	var taskID sql.NullString
	if c.TaskID != "" {
		taskID = sql.NullString{String: c.TaskID, Valid: true}
	}

	for attempt := 0; attempt < 5; attempt++ {
		tx, err := db.Begin()
		if err != nil {
			break
		}
		if _, err = globals.ExecTx(tx, insertConversationQuery, c.UserID, c.Id, c.Name, data, c.Model, taskID); err != nil {
			_ = tx.Rollback()
			if !isDuplicateConversationIDError(err) {
				break
			}
			c.Id = GetConversationLengthByUserID(db, c.UserID) + 1
			continue
		}

		leaseExpiresAt := time.Now().Add(chatRequestLeaseDuration).UnixMilli()
		result, err := globals.ExecTx(tx, `
			UPDATE chat_request
			SET conversation_id = ?, status = ?, lease_expires_at = ?, updated_at = CURRENT_TIMESTAMP
			WHERE user_id = ? AND request_id = ? AND owner_token = ? AND status = ?
		`, c.Id, ChatRequestAccepted, leaseExpiresAt,
			c.UserID, strings.TrimSpace(form.RequestID), strings.TrimSpace(ownerToken), ChatRequestReserved)
		if err != nil {
			_ = tx.Rollback()
			break
		}
		updated, err := result.RowsAffected()
		if err != nil || updated != 1 {
			_ = tx.Rollback()
			break
		}
		if err := tx.Commit(); err != nil {
			break
		}

		c.Persisted = true
		return true
	}

	c.Message = c.Message[:previousLength]
	c.Name = previousName
	return false
}

func (c *Conversation) insertNewConversation(db *sql.DB, data string, taskID sql.NullString) bool {
	for attempt := 0; attempt < 5; attempt++ {
		stmt, err := globals.PrepareDb(db, insertConversationQuery)
		if err != nil {
			return false
		}

		_, err = stmt.Exec(c.UserID, c.Id, c.Name, data, c.Model, taskID)
		if closeErr := stmt.Close(); closeErr != nil {
			globals.Warn(closeErr)
		}

		if err == nil {
			c.Persisted = true
			return true
		}

		if !isDuplicateConversationIDError(err) {
			globals.Info(fmt.Sprintf("execute error during insert conversation: %s", err.Error()))
			return false
		}

		c.Id = GetConversationLengthByUserID(db, c.UserID) + 1
	}

	globals.Info(fmt.Sprintf("failed to allocate unique conversation id for user %d", c.UserID))
	return false
}

func GetConversationLengthByUserID(db *sql.DB, userId int64) int64 {
	var length int64
	err := globals.QueryRowDb(db, "SELECT MAX(conversation_id) FROM conversation WHERE user_id = ?", userId).Scan(&length)
	if err != nil || length < 0 {
		return 0
	}
	return length
}

func LoadConversation(db *sql.DB, userId int64, conversationId int64) *Conversation {
	conversation := Conversation{
		UserID:    userId,
		Id:        conversationId,
		Persisted: true,
	}

	var (
		data      string
		model     interface{}
		taskID    sql.NullString
		updatedAt interface{}
	)
	var favorite interface{}
	err := globals.QueryRowDb(db, `
		SELECT conversation_name, model, data, task_id, updated_at, favorite FROM conversation
		WHERE user_id = ? AND conversation_id = ?
		`, userId, conversationId).Scan(&conversation.Name, &model, &data, &taskID, &updatedAt, &favorite)
	if err != nil {
		return nil
	}

	conversation.Model = normalizeDBString(model)
	if conversation.Model == "" {
		conversation.Model = globals.GPT3Turbo
	}
	if taskID.Valid {
		conversation.TaskID = taskID.String
	}
	conversation.UpdatedAt = normalizeDBString(updatedAt)
	conversation.Favorite = normalizeDBBool(favorite)

	conversation.Message, err = utils.Unmarshal[[]globals.Message]([]byte(data))
	if err != nil {
		return nil
	}

	return &conversation
}

func LoadConversationList(db *sql.DB, userId int64) []Conversation {
	var conversationList []Conversation
	rows, err := globals.QueryDb(db, `
			SELECT conversation_id, conversation_name, updated_at, favorite FROM conversation WHERE user_id = ?
			ORDER BY conversation_id DESC LIMIT 100
	`, userId)
	if err != nil {
		return conversationList
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			return
		}
	}(rows)

	for rows.Next() {
		var conversation Conversation
		var updatedAt interface{}
		var favorite interface{}
		err := rows.Scan(&conversation.Id, &conversation.Name, &updatedAt, &favorite)
		if err != nil {
			continue
		}
		conversation.UpdatedAt = normalizeDBString(updatedAt)
		conversation.Favorite = normalizeDBBool(favorite)
		conversationList = append(conversationList, conversation)
	}

	return conversationList
}

func (c *Conversation) DeleteConversation(db *sql.DB) bool {
	tx, err := db.Begin()
	if err != nil {
		return false
	}
	if _, err = globals.ExecTx(tx, "DELETE FROM chat_request WHERE user_id = ? AND conversation_id = ?", c.UserID, c.Id); err != nil {
		_ = tx.Rollback()
		return false
	}
	if _, err = globals.ExecTx(tx, "DELETE FROM conversation WHERE user_id = ? AND conversation_id = ?", c.UserID, c.Id); err != nil {
		_ = tx.Rollback()
		return false
	}
	return tx.Commit() == nil
}

func (c *Conversation) RenameConversation(db *sql.DB, name string) bool {
	_, err := globals.ExecDb(db, "UPDATE conversation SET conversation_name = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ? AND conversation_id = ?", name, c.UserID, c.Id)
	if err != nil {
		return false
	}
	return true
}

func (c *Conversation) SetFavorite(db *sql.DB, favorite bool) bool {
	_, err := globals.ExecDb(db, "UPDATE conversation SET favorite = ? WHERE user_id = ? AND conversation_id = ?", favorite, c.UserID, c.Id)
	if err != nil {
		return false
	}
	c.Favorite = favorite
	return true
}

func DeleteAllConversations(db *sql.DB, user auth.User) error {
	userID := user.GetID(db)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if _, err = globals.ExecTx(tx, "DELETE FROM chat_request WHERE user_id = ?", userID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err = globals.ExecTx(tx, "DELETE FROM conversation WHERE user_id = ?", userID); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
