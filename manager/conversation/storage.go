package conversation

import (
	"chat/auth"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"fmt"
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

func loadGlobalConversationAttachmentNames(db *sql.DB) map[string]struct{} {
	rows, err := globals.QueryDb(db, `SELECT data FROM conversation`)
	if err != nil {
		return map[string]struct{}{}
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	result := map[string]struct{}{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		for _, name := range utils.ExtractAttachmentNames(data) {
			result[name] = struct{}{}
		}
	}

	return result
}

func cleanupOrphanStoredAttachments(db *sql.DB) {
	storedNames, err := utils.ListStoredAttachmentNames()
	if err != nil {
		globals.Warn(fmt.Sprintf("[conversation] list stored attachments error: %s", err.Error()))
		return
	}

	referenced := loadGlobalConversationAttachmentNames(db)
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

	var taskID sql.NullString
	if c.TaskID != "" {
		taskID = sql.NullString{String: c.TaskID, Valid: true}
	}

	_, err = stmt.Exec(c.UserID, c.Id, c.Name, data, c.Model, taskID)
	if err != nil {
		globals.Info(fmt.Sprintf("execute error during save conversation: %s", err.Error()))
		return false
	}
	return true
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
		UserID: userId,
		Id:     conversationId,
	}

	var (
		data      string
		model     interface{}
		taskID    sql.NullString
		updatedAt interface{}
	)
	err := globals.QueryRowDb(db, `
		SELECT conversation_name, model, data, task_id, updated_at FROM conversation
		WHERE user_id = ? AND conversation_id = ?
		`, userId, conversationId).Scan(&conversation.Name, &model, &data, &taskID, &updatedAt)
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

	conversation.Message, err = utils.Unmarshal[[]globals.Message]([]byte(data))
	if err != nil {
		return nil
	}

	return &conversation
}

func LoadConversationList(db *sql.DB, userId int64) []Conversation {
	var conversationList []Conversation
	rows, err := globals.QueryDb(db, `
			SELECT conversation_id, conversation_name, updated_at FROM conversation WHERE user_id = ?
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
		err := rows.Scan(&conversation.Id, &conversation.Name, &updatedAt)
		if err != nil {
			continue
		}
		conversation.UpdatedAt = normalizeDBString(updatedAt)
		conversationList = append(conversationList, conversation)
	}

	return conversationList
}

func (c *Conversation) DeleteConversation(db *sql.DB) bool {
	_, err := globals.ExecDb(db, "DELETE FROM conversation WHERE user_id = ? AND conversation_id = ?", c.UserID, c.Id)
	return err == nil
}

func (c *Conversation) RenameConversation(db *sql.DB, name string) bool {
	_, err := globals.ExecDb(db, "UPDATE conversation SET conversation_name = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ? AND conversation_id = ?", name, c.UserID, c.Id)
	if err != nil {
		return false
	}
	return true
}

func DeleteAllConversations(db *sql.DB, user auth.User) error {
	userID := user.GetID(db)
	_, err := globals.ExecDb(db, "DELETE FROM conversation WHERE user_id = ?", userID)
	return err
}
