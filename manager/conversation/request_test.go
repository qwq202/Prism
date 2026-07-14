package conversation

import (
	"chat/connection"
	"chat/globals"
	"testing"
	"time"
)

func TestReserveChatRequestDeduplicatesAndRecoversPersistedMessage(t *testing.T) {
	db := openConversationTestDB(t, "chat-request.db")
	connection.CreateChatRequestTable(db)

	const (
		userID    = int64(1)
		requestID = "request-1"
	)
	instance := NewConversation(db, userID)

	record, owner, err := ReserveChatRequest(db, userID, requestID, instance.GetId())
	if err != nil {
		t.Fatalf("reserve request: %v", err)
	}
	if !owner || record.Status != ChatRequestReserved {
		t.Fatalf("expected the first reservation to be owned, got owner=%v record=%#v", owner, record)
	}

	duplicate, duplicateOwner, err := ReserveChatRequest(db, userID, requestID, instance.GetId())
	if err != nil {
		t.Fatalf("reserve duplicate request: %v", err)
	}
	if duplicateOwner || duplicate.Status != ChatRequestReserved {
		t.Fatalf("expected an active duplicate to remain pending, got owner=%v record=%#v", duplicateOwner, duplicate)
	}

	if !instance.HandleMessage(db, &FormMessage{
		Type:      "chat",
		Message:   "persist me once",
		RequestID: requestID,
	}) {
		t.Fatalf("persist request message")
	}
	accepted, err := MarkChatRequestStatusOwned(
		db, userID, requestID, instance.GetId(), record.OwnerToken, ChatRequestAccepted,
	)
	if err != nil || !accepted {
		t.Fatalf("mark request accepted: accepted=%v err=%v", accepted, err)
	}

	recovered, recoveredOwner, err := ReserveChatRequest(db, userID, requestID, instance.GetId())
	if err != nil {
		t.Fatalf("recover persisted request: %v", err)
	}
	if recoveredOwner || recovered.Status != ChatRequestAccepted {
		t.Fatalf("expected persisted duplicate to recover as accepted without a second owner, got owner=%v record=%#v", recoveredOwner, recovered)
	}

	stored := LoadConversation(db, userID, instance.GetId())
	if stored == nil || stored.GetMessageLength() != 1 {
		t.Fatalf("expected exactly one persisted user message, got %#v", stored)
	}
	message := stored.GetMessage()[0]
	if message.Content != "persist me once" || message.RequestID != requestID {
		t.Fatalf("unexpected persisted request message: %#v", message)
	}

	staleAt := time.Now().Add(-time.Second).UnixMilli()
	if _, err := globals.ExecDb(db, `
		UPDATE chat_request SET lease_expires_at = ?, status = ?
		WHERE user_id = ? AND request_id = ?
	`, staleAt, ChatRequestAccepted, userID, requestID); err != nil {
		t.Fatalf("age accepted request: %v", err)
	}
	aged, agedOwner, err := ReserveChatRequest(db, userID, requestID, instance.GetId())
	if err != nil {
		t.Fatalf("inspect old accepted request: %v", err)
	}
	if !agedOwner || !aged.Recovered || aged.Status != ChatRequestAccepted || aged.OwnerToken == record.OwnerToken {
		t.Fatalf("expected expired accepted request to receive a fenced recovery owner, got owner=%v record=%#v", agedOwner, aged)
	}
	if oldOwned, err := MarkChatRequestStatusOwned(
		db, userID, requestID, instance.GetId(), record.OwnerToken, ChatRequestCompleted,
	); err != nil || oldOwned {
		t.Fatalf("expected the expired owner token to be fenced, owned=%v err=%v", oldOwned, err)
	}

	stored.AddMessage(globals.Message{Role: globals.Assistant, Content: "done", RequestID: requestID})
	if !stored.SaveConversation(db) {
		t.Fatalf("persist completed assistant response")
	}
	completed, completedOwner, err := ReserveChatRequest(db, userID, requestID, instance.GetId())
	if err != nil {
		t.Fatalf("recover completed request: %v", err)
	}
	if completedOwner || completed.Status != ChatRequestCompleted {
		t.Fatalf("expected assistant metadata to recover completion, got owner=%v record=%#v", completedOwner, completed)
	}

	if !stored.DeleteConversation(db) {
		t.Fatalf("delete conversation")
	}
	deleted, err := LookupChatRequest(db, userID, requestID)
	if err != nil {
		t.Fatalf("lookup request after conversation deletion: %v", err)
	}
	if deleted != nil {
		t.Fatalf("expected conversation deletion to remove request state, got %#v", deleted)
	}
}

func TestReserveChatRequestReclaimsStaleUnpersistedReservation(t *testing.T) {
	db := openConversationTestDB(t, "chat-request-stale.db")
	connection.CreateChatRequestTable(db)

	const (
		userID    = int64(2)
		requestID = "request-stale"
	)
	record, owner, err := ReserveChatRequest(db, userID, requestID, 1)
	if err != nil || !owner {
		t.Fatalf("reserve request: owner=%v record=%#v err=%v", owner, record, err)
	}

	staleAt := time.Now().Add(-time.Second).UnixMilli()
	if _, err := globals.ExecDb(db, `
		UPDATE chat_request SET lease_expires_at = ?
		WHERE user_id = ? AND request_id = ?
	`, staleAt, userID, requestID); err != nil {
		t.Fatalf("age reservation: %v", err)
	}

	reclaimed, reclaimedOwner, err := ReserveChatRequest(db, userID, requestID, 2)
	if err != nil {
		t.Fatalf("reclaim stale request: %v", err)
	}
	if !reclaimedOwner || reclaimed.ConversationID != 2 || reclaimed.Status != ChatRequestReserved || reclaimed.OwnerToken == record.OwnerToken {
		t.Fatalf("expected stale reservation to be reclaimed, got owner=%v record=%#v", reclaimedOwner, reclaimed)
	}
}
