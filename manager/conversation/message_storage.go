package conversation

import (
	"chat/globals"
	"chat/utils"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
)

const (
	MessageStatusStreaming   = "streaming"
	MessageStatusCompleted   = "completed"
	MessageStatusFailed      = "failed"
	MessageStatusInterrupted = "interrupted"
)

type generationCheckpointWriter struct {
	db        *sql.DB
	updates   chan globals.Message
	done      chan struct{}
	closeOnce sync.Once
}

func newMessageID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err == nil {
		return hex.EncodeToString(buffer)
	}
	return utils.Md5Encrypt(fmt.Sprintf("%p", &buffer))
}

func normalizeMessageStatus(message globals.Message) string {
	status := strings.TrimSpace(message.Status)
	if status != "" {
		return status
	}
	return MessageStatusCompleted
}

func ensureMessageMetadata(message *globals.Message) {
	if message == nil {
		return
	}
	if strings.TrimSpace(message.MessageID) == "" {
		message.MessageID = newMessageID()
	}
	message.Status = normalizeMessageStatus(*message)
}

func ensureMessagesMetadata(messages []globals.Message) bool {
	changed := false
	for index := range messages {
		beforeID := messages[index].MessageID
		beforeStatus := messages[index].Status
		ensureMessageMetadata(&messages[index])
		changed = changed || beforeID != messages[index].MessageID || beforeStatus != messages[index].Status
	}
	return changed
}

func upsertMessageRecord(execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}, userID int64, conversationID int64, position int, message globals.Message, overwrite bool) error {
	ensureMessageMetadata(&message)
	if overwrite {
		result, err := execer.Exec(globals.PreflightSql(`
			UPDATE conversation_message
			SET request_id = ?, position = ?, role = ?, status = ?, data = ?, updated_at = CURRENT_TIMESTAMP
			WHERE user_id = ? AND conversation_id = ? AND message_id = ?
		`), strings.TrimSpace(message.RequestID), position, message.Role, message.Status, utils.ToJson(message), userID, conversationID, message.MessageID)
		if err != nil {
			return err
		}
		updated, err := result.RowsAffected()
		if err != nil || updated > 0 {
			return err
		}
	}

	_, err := execer.Exec(globals.PreflightSql(`
		INSERT INTO conversation_message (
			user_id, conversation_id, message_id, request_id, position, role, status, data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`), userID, conversationID, message.MessageID, strings.TrimSpace(message.RequestID), position, message.Role, message.Status, utils.ToJson(message))
	if err != nil && isDuplicateChatRequestError(err) {
		return nil
	}
	return err
}

func loadMessageRecords(db *sql.DB, userID int64, conversationID int64) ([]globals.Message, bool, error) {
	rows, err := globals.QueryDb(db, `
		SELECT status, data FROM conversation_message
		WHERE user_id = ? AND conversation_id = ?
		ORDER BY position ASC, created_at ASC, message_id ASC
	`, userID, conversationID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	messages := make([]globals.Message, 0)
	for rows.Next() {
		var status string
		var data string
		if err := rows.Scan(&status, &data); err != nil {
			return nil, false, err
		}
		message, err := utils.Unmarshal[globals.Message]([]byte(data))
		if err != nil {
			return nil, false, err
		}
		ensureMessageMetadata(&message)
		message.Status = status
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return messages, len(messages) > 0, nil
}

func persistLegacyMessages(db *sql.DB, c *Conversation) error {
	if db == nil || c == nil || c.UserID < 0 || !c.Persisted {
		return nil
	}
	ensureMessagesMetadata(c.Message)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for position, message := range c.Message {
		if err := upsertMessageRecord(tx, c.UserID, c.Id, position, message, false); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if _, err := globals.ExecTx(tx, `
		UPDATE conversation SET data = ?
		WHERE user_id = ? AND conversation_id = ?
	`, utils.ToJson(c.Message), c.UserID, c.Id); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (c *Conversation) syncNewMessageRecords(db *sql.DB) bool {
	if db == nil || c == nil || c.UserID < 0 || !c.Persisted {
		return true
	}
	ensureMessagesMetadata(c.Message)
	for position, message := range c.Message {
		if err := upsertMessageRecord(db, c.UserID, c.Id, position, message, false); err != nil {
			if isMissingMessageTableError(err) {
				return true
			}
			globals.Warn(fmt.Sprintf("[conversation] persist message %s failed: %s", message.MessageID, err.Error()))
			return false
		}
	}
	return true
}

func (c *Conversation) BeginGeneration(db *sql.DB, requestID string) (string, bool) {
	if db == nil || c == nil || c.UserID < 0 || !c.Persisted {
		return "", true
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		requestID = newMessageID()
	}
	var existingMessageID string
	lookupErr := globals.QueryRowDb(db, `
		SELECT assistant_message_id FROM generation_task
		WHERE user_id = ? AND task_id = ?
	`, c.UserID, requestID).Scan(&existingMessageID)
	if lookupErr == nil && strings.TrimSpace(existingMessageID) != "" {
		_, err := globals.ExecDb(db, `
			UPDATE generation_task
			SET status = ?, error = NULL, completed_at = NULL, updated_at = CURRENT_TIMESTAMP
			WHERE user_id = ? AND task_id = ?
		`, MessageStatusStreaming, c.UserID, requestID)
		if err != nil {
			return "", false
		}
		c.ActiveTaskID = requestID
		c.ActiveAssistantMessageID = existingMessageID
		c.startCheckpointWriter(db)
		return existingMessageID, true
	}
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		if isMissingMessageTableError(lookupErr) {
			return "", true
		}
		globals.Warn("[generation] lookup task failed: " + lookupErr.Error())
		return "", false
	}
	message := globals.Message{
		MessageID: newMessageID(),
		Role:      globals.Assistant,
		RequestID: requestID,
		Status:    MessageStatusStreaming,
		Model:     c.GetModel(),
	}
	if err := upsertMessageRecord(db, c.UserID, c.Id, len(c.Message), message, false); err != nil {
		globals.Warn("[generation] create assistant placeholder failed: " + err.Error())
		return "", false
	}
	_, err := globals.ExecDb(db, `
		INSERT INTO generation_task (
			user_id, task_id, conversation_id, assistant_message_id, status
		) VALUES (?, ?, ?, ?, ?)
	`, c.UserID, requestID, c.Id, message.MessageID, MessageStatusStreaming)
	if err != nil && !isDuplicateChatRequestError(err) {
		globals.Warn("[generation] create task failed: " + err.Error())
		_ = c.deleteMessageRecord(db, message.MessageID)
		return "", false
	}
	c.ActiveTaskID = requestID
	c.ActiveAssistantMessageID = message.MessageID
	c.startCheckpointWriter(db)
	return message.MessageID, true
}

func (c *Conversation) startCheckpointWriter(db *sql.DB) {
	if c == nil || db == nil || c.checkpointWriter != nil {
		return
	}
	writer := &generationCheckpointWriter{
		db:      db,
		updates: make(chan globals.Message, 1),
		done:    make(chan struct{}),
	}
	c.checkpointWriter = writer
	go func() {
		defer close(writer.done)
		for message := range writer.updates {
			c.checkpointGeneration(writer.db, message)
		}
	}()
}

func (c *Conversation) QueueGenerationCheckpoint(message globals.Message) {
	if c == nil || c.checkpointWriter == nil {
		return
	}
	writer := c.checkpointWriter
	select {
	case writer.updates <- message:
		return
	default:
	}
	select {
	case <-writer.updates:
	default:
	}
	select {
	case writer.updates <- message:
	default:
	}
}

func (c *Conversation) stopCheckpointWriter() {
	if c == nil || c.checkpointWriter == nil {
		return
	}
	writer := c.checkpointWriter
	writer.closeOnce.Do(func() {
		close(writer.updates)
	})
	<-writer.done
	c.checkpointWriter = nil
}

func (c *Conversation) checkpointGeneration(db *sql.DB, message globals.Message) bool {
	if db == nil || c == nil || c.UserID < 0 || strings.TrimSpace(c.ActiveAssistantMessageID) == "" {
		return false
	}
	message.MessageID = c.ActiveAssistantMessageID
	message.RequestID = c.ActiveTaskID
	message.Role = globals.Assistant
	message.Status = MessageStatusStreaming
	if message.Model == "" {
		message.Model = c.GetModel()
	}
	if err := upsertMessageRecord(db, c.UserID, c.Id, len(c.Message), message, true); err != nil {
		globals.Warn("[generation] checkpoint assistant message failed: " + err.Error())
		return false
	}
	_, err := globals.ExecDb(db, `
		UPDATE generation_task SET updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND task_id = ? AND status = ?
	`, c.UserID, c.ActiveTaskID, MessageStatusStreaming)
	return err == nil
}

func (c *Conversation) finishGeneration(db *sql.DB, message globals.Message) error {
	if db == nil || c == nil || c.UserID < 0 || strings.TrimSpace(c.ActiveAssistantMessageID) == "" {
		return nil
	}
	c.stopCheckpointWriter()
	message.MessageID = c.ActiveAssistantMessageID
	message.RequestID = c.ActiveTaskID
	ensureMessageMetadata(&message)
	if err := upsertMessageRecord(db, c.UserID, c.Id, len(c.Message), message, true); err != nil {
		return err
	}
	if message.Status == MessageStatusStreaming {
		return nil
	}
	_, err := globals.ExecDb(db, `
		UPDATE generation_task
		SET status = ?, error = ?, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND task_id = ?
	`, message.Status, generationError(message), c.UserID, c.ActiveTaskID)
	return err
}

func (c *Conversation) discardGeneration(db *sql.DB) bool {
	if db == nil || c == nil || c.UserID < 0 || strings.TrimSpace(c.ActiveAssistantMessageID) == "" {
		return false
	}
	c.stopCheckpointWriter()
	if err := c.deleteMessageRecord(db, c.ActiveAssistantMessageID); err != nil {
		return false
	}
	_, err := globals.ExecDb(db, `
		UPDATE generation_task
		SET status = ?, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND task_id = ?
	`, MessageStatusCompleted, c.UserID, c.ActiveTaskID)
	return err == nil
}

func generationError(message globals.Message) interface{} {
	if message.Status != MessageStatusFailed {
		return nil
	}
	if strings.TrimSpace(message.Content) == "" {
		return "generation failed"
	}
	return message.Content
}

func (c *Conversation) updateMessageRecord(db *sql.DB, message globals.Message, position int) error {
	if db == nil || c == nil || c.UserID < 0 || strings.TrimSpace(message.MessageID) == "" {
		return nil
	}
	return upsertMessageRecord(db, c.UserID, c.Id, position, message, true)
}

func (c *Conversation) deleteMessageRecord(db *sql.DB, messageID string) error {
	if db == nil || c == nil || c.UserID < 0 || strings.TrimSpace(messageID) == "" {
		return nil
	}
	_, err := globals.ExecDb(db, `
		DELETE FROM conversation_message
		WHERE user_id = ? AND conversation_id = ? AND message_id = ?
	`, c.UserID, c.Id, messageID)
	return err
}

func (c *Conversation) RemoveMessagePersisted(db *sql.DB, index int) bool {
	removed := c.RemoveMessage(index)
	if removed.Role == "" {
		return false
	}
	if err := c.deleteMessageRecord(db, removed.MessageID); err != nil {
		return false
	}
	return c.SaveConversation(db)
}

func (c *Conversation) EditMessagePersisted(db *sql.DB, index int, content string) bool {
	if !c.HasMessageId(index) {
		return false
	}
	c.EditMessage(index, content)
	if err := c.updateMessageRecord(db, c.Message[index], index); err != nil {
		return false
	}
	return c.SaveConversation(db)
}

func isMissingMessageTableError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return errors.Is(err, sql.ErrNoRows) || strings.Contains(lower, "no such table") || strings.Contains(lower, "doesn't exist")
}
