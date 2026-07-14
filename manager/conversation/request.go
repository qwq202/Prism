package conversation

import (
	"chat/globals"
	"database/sql"
	"errors"
	"strings"
	"time"
)

const (
	ChatRequestReserved   = "reserved"
	ChatRequestAccepted   = "accepted"
	ChatRequestCompleted  = "completed"
	chatRequestStaleAfter = 30 * time.Second
)

type ChatRequestRecord struct {
	RequestID      string
	ConversationID int64
	Status         string
	ReservedAt     int64
}

func normalizeChatRequestID(requestID string) (string, error) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return "", nil
	}
	if len(requestID) > 128 {
		return "", errors.New("request id is too long")
	}
	return requestID, nil
}

func isDuplicateChatRequestError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "duplicate") ||
		strings.Contains(lower, "unique constraint failed")
}

func LookupChatRequest(db *sql.DB, userID int64, requestID string) (*ChatRequestRecord, error) {
	requestID, err := normalizeChatRequestID(requestID)
	if err != nil || requestID == "" || db == nil || userID < 0 {
		return nil, err
	}

	record := &ChatRequestRecord{RequestID: requestID}
	err = globals.QueryRowDb(db, `
		SELECT conversation_id, status, reserved_at
		FROM chat_request
		WHERE user_id = ? AND request_id = ?
	`, userID, requestID).Scan(&record.ConversationID, &record.Status, &record.ReservedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func ReserveChatRequest(db *sql.DB, userID int64, requestID string, conversationID int64) (*ChatRequestRecord, bool, error) {
	requestID, err := normalizeChatRequestID(requestID)
	if err != nil {
		return nil, false, err
	}
	if requestID == "" || db == nil || userID < 0 {
		return &ChatRequestRecord{RequestID: requestID, ConversationID: conversationID}, true, nil
	}

	now := time.Now().UnixMilli()
	_, err = globals.ExecDb(db, `
		INSERT INTO chat_request (user_id, request_id, conversation_id, status, reserved_at)
		VALUES (?, ?, ?, ?, ?)
	`, userID, requestID, conversationID, ChatRequestReserved, now)
	if err == nil {
		return &ChatRequestRecord{
			RequestID: requestID, ConversationID: conversationID,
			Status: ChatRequestReserved, ReservedAt: now,
		}, true, nil
	}
	if !isDuplicateChatRequestError(err) {
		return nil, false, err
	}

	record, lookupErr := LookupChatRequest(db, userID, requestID)
	if lookupErr != nil || record == nil {
		return record, false, lookupErr
	}
	if record.Status != ChatRequestReserved {
		return record, false, nil
	}

	if persisted := LoadConversation(db, userID, record.ConversationID); persisted != nil && persisted.HasRequestID(requestID) {
		_ = MarkChatRequestStatus(db, userID, requestID, record.ConversationID, ChatRequestAccepted)
		record.Status = ChatRequestAccepted
		return record, false, nil
	}

	if time.Since(time.UnixMilli(record.ReservedAt)) <= chatRequestStaleAfter {
		return record, false, nil
	}

	result, deleteErr := globals.ExecDb(db, `
		DELETE FROM chat_request
		WHERE user_id = ? AND request_id = ? AND status = ? AND reserved_at = ?
	`, userID, requestID, ChatRequestReserved, record.ReservedAt)
	if deleteErr != nil {
		return nil, false, deleteErr
	}
	removed, rowsErr := result.RowsAffected()
	if rowsErr != nil || removed == 0 {
		return record, false, rowsErr
	}
	return ReserveChatRequest(db, userID, requestID, conversationID)
}

func MarkChatRequestStatus(db *sql.DB, userID int64, requestID string, conversationID int64, status string) error {
	requestID, err := normalizeChatRequestID(requestID)
	if err != nil || requestID == "" || db == nil || userID < 0 {
		return err
	}
	_, err = globals.ExecDb(db, `
		UPDATE chat_request
		SET conversation_id = ?, status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND request_id = ?
	`, conversationID, status, userID, requestID)
	return err
}

func ReleaseChatRequest(db *sql.DB, userID int64, requestID string) error {
	requestID, err := normalizeChatRequestID(requestID)
	if err != nil || requestID == "" || db == nil || userID < 0 {
		return err
	}
	_, err = globals.ExecDb(db, `
		DELETE FROM chat_request
		WHERE user_id = ? AND request_id = ? AND status = ?
	`, userID, requestID, ChatRequestReserved)
	return err
}
