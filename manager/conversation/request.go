package conversation

import (
	"chat/globals"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const (
	ChatRequestReserved      = "reserved"
	ChatRequestAccepted      = "accepted"
	ChatRequestCompleted     = "completed"
	chatRequestLeaseDuration = 2 * time.Minute
)

type ChatRequestRecord struct {
	RequestID      string
	ConversationID int64
	Status         string
	ReservedAt     int64
	OwnerToken     string
	LeaseExpiresAt int64
	Recovered      bool
}

func ChatRequestLeaseHeartbeatInterval() time.Duration {
	return chatRequestLeaseDuration / 4
}

func newChatRequestOwnerToken() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
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
		SELECT conversation_id, status, reserved_at, owner_token, lease_expires_at
		FROM chat_request
		WHERE user_id = ? AND request_id = ?
	`, userID, requestID).Scan(
		&record.ConversationID,
		&record.Status,
		&record.ReservedAt,
		&record.OwnerToken,
		&record.LeaseExpiresAt,
	)
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

	ownerToken, tokenErr := newChatRequestOwnerToken()
	if tokenErr != nil {
		return nil, false, tokenErr
	}
	now := time.Now().UnixMilli()
	leaseExpiresAt := now + chatRequestLeaseDuration.Milliseconds()
	_, err = globals.ExecDb(db, `
		INSERT INTO chat_request (
			user_id, request_id, conversation_id, status, reserved_at,
			owner_token, lease_expires_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, requestID, conversationID, ChatRequestReserved, now, ownerToken, leaseExpiresAt)
	if err == nil {
		return &ChatRequestRecord{
			RequestID: requestID, ConversationID: conversationID,
			Status: ChatRequestReserved, ReservedAt: now,
			OwnerToken: ownerToken, LeaseExpiresAt: leaseExpiresAt,
		}, true, nil
	}
	if !isDuplicateChatRequestError(err) {
		return nil, false, err
	}

	record, lookupErr := LookupChatRequest(db, userID, requestID)
	if lookupErr != nil || record == nil {
		return record, false, lookupErr
	}
	persisted := LoadConversation(db, userID, record.ConversationID)
	if persisted != nil && persisted.HasCompletedRequestID(requestID) {
		if err := MarkChatRequestStatus(db, userID, requestID, record.ConversationID, ChatRequestCompleted); err != nil {
			return nil, false, err
		}
		record.Status = ChatRequestCompleted
		return record, false, nil
	}

	if record.Status == ChatRequestCompleted {
		return record, false, nil
	}
	if record.LeaseExpiresAt > now {
		return record, false, nil
	}

	targetStatus := ChatRequestReserved
	targetConversationID := conversationID
	recovered := false
	if persisted != nil && persisted.HasRequestID(requestID) {
		targetStatus = ChatRequestAccepted
		targetConversationID = record.ConversationID
		recovered = true
	}

	result, deleteErr := globals.ExecDb(db, `
		UPDATE chat_request
		SET conversation_id = ?, status = ?, owner_token = ?, reserved_at = ?, lease_expires_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND request_id = ? AND status = ?
			AND owner_token = ? AND lease_expires_at = ?
	`, targetConversationID, targetStatus, ownerToken, now, leaseExpiresAt,
		userID, requestID, record.Status, record.OwnerToken, record.LeaseExpiresAt)
	if deleteErr != nil {
		return nil, false, deleteErr
	}
	updated, rowsErr := result.RowsAffected()
	if rowsErr != nil || updated == 0 {
		return record, false, rowsErr
	}
	record.Status = targetStatus
	record.ConversationID = targetConversationID
	record.ReservedAt = now
	record.OwnerToken = ownerToken
	record.LeaseExpiresAt = leaseExpiresAt
	record.Recovered = recovered
	return record, true, nil
}

func MarkChatRequestStatusOwned(
	db *sql.DB,
	userID int64,
	requestID string,
	conversationID int64,
	ownerToken string,
	status string,
) (bool, error) {
	requestID, err := normalizeChatRequestID(requestID)
	ownerToken = strings.TrimSpace(ownerToken)
	if err != nil || requestID == "" || ownerToken == "" || db == nil || userID < 0 {
		return false, err
	}
	leaseExpiresAt := time.Now().Add(chatRequestLeaseDuration).UnixMilli()
	result, err := globals.ExecDb(db, `
		UPDATE chat_request
		SET conversation_id = ?, status = ?, lease_expires_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND request_id = ? AND owner_token = ?
			AND status <> ?
	`, conversationID, status, leaseExpiresAt, userID, requestID, ownerToken, ChatRequestCompleted)
	if err != nil {
		return false, err
	}
	updated, err := result.RowsAffected()
	return updated == 1, err
}

func RenewChatRequestLease(db *sql.DB, userID int64, requestID string, ownerToken string) (bool, error) {
	requestID, err := normalizeChatRequestID(requestID)
	ownerToken = strings.TrimSpace(ownerToken)
	if err != nil || requestID == "" || ownerToken == "" || db == nil || userID < 0 {
		return false, err
	}
	leaseExpiresAt := time.Now().Add(chatRequestLeaseDuration).UnixMilli()
	result, err := globals.ExecDb(db, `
		UPDATE chat_request
		SET lease_expires_at = CASE
				WHEN lease_expires_at >= ? THEN lease_expires_at + 1
				ELSE ?
			END,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND request_id = ? AND owner_token = ?
			AND status <> ?
	`, leaseExpiresAt, leaseExpiresAt, userID, requestID, ownerToken, ChatRequestCompleted)
	if err != nil {
		return false, err
	}
	updated, err := result.RowsAffected()
	return updated == 1, err
}

func ReleaseChatRequestOwned(db *sql.DB, userID int64, requestID string, ownerToken string) error {
	requestID, err := normalizeChatRequestID(requestID)
	ownerToken = strings.TrimSpace(ownerToken)
	if err != nil || requestID == "" || ownerToken == "" || db == nil || userID < 0 {
		return err
	}
	_, err = globals.ExecDb(db, `
		DELETE FROM chat_request
		WHERE user_id = ? AND request_id = ? AND owner_token = ? AND status = ?
	`, userID, requestID, ownerToken, ChatRequestReserved)
	return err
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
