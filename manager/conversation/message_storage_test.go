package conversation

import (
	"chat/connection"
	"chat/globals"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func openIndependentMessageTestDB(t *testing.T) *sql.DB {
	t.Helper()
	previous := globals.SqliteEngine
	globals.SqliteEngine = true
	t.Cleanup(func() { globals.SqliteEngine = previous })

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "messages.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	connection.CreateConversationTable(db)
	connection.CreateChatRequestTable(db)
	connection.CreateConversationMessageTable(db)
	connection.CreateGenerationTaskTable(db)
	return db
}

func TestLegacyConversationMigratesToIndependentMessages(t *testing.T) {
	db := openIndependentMessageTestDB(t)
	legacy := []globals.Message{
		{Role: globals.User, Content: "hello"},
		{Role: globals.Assistant, Content: "world"},
	}
	if _, err := globals.ExecDb(db, `
		INSERT INTO conversation (user_id, conversation_id, conversation_name, data, model)
		VALUES (?, ?, ?, ?, ?)
	`, 1, 1, "legacy", `[{"role":"user","content":"hello"},{"role":"assistant","content":"world"}]`, globals.GPT3Turbo); err != nil {
		t.Fatalf("insert legacy conversation: %v", err)
	}

	loaded := LoadConversation(db, 1, 1)
	if loaded == nil || len(loaded.Message) != len(legacy) {
		t.Fatalf("unexpected migrated messages: %#v", loaded)
	}
	for _, message := range loaded.Message {
		if message.MessageID == "" || message.Status != MessageStatusCompleted {
			t.Fatalf("expected stable metadata, got %#v", message)
		}
	}

	var count int
	if err := globals.QueryRowDb(db, `
		SELECT COUNT(*) FROM conversation_message WHERE user_id = ? AND conversation_id = ?
	`, 1, 1).Scan(&count); err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 independent messages, got %d", count)
	}
}

func TestGenerationCheckpointSurvivesReloadAndFinalReplacesPlaceholder(t *testing.T) {
	db := openIndependentMessageTestDB(t)
	instance := &Conversation{
		Auth: true, UserID: 1, Id: 1, Name: "chat", Model: globals.GPT3Turbo,
		Message: []globals.Message{},
	}
	instance.AddMessage(globals.Message{Role: globals.User, Content: "tell me more", RequestID: "request-1"})
	if !instance.SaveConversation(db) {
		t.Fatal("save user message")
	}
	messageID, ok := instance.BeginGeneration(db, "request-1")
	if !ok || messageID == "" {
		t.Fatal("begin generation")
	}
	if !instance.checkpointGeneration(db, globals.Message{
		Role: globals.Assistant, Content: "partial", Model: globals.GPT3Turbo,
	}) {
		t.Fatal("checkpoint generation")
	}
	partial := LoadConversation(db, 1, 1)
	if partial == nil || len(partial.Message) != 2 || partial.Message[1].Content != "partial" || partial.Message[1].Status != MessageStatusStreaming {
		t.Fatalf("expected reloadable partial response, got %#v", partial)
	}
	if !instance.SaveResponse(db, globals.Message{
		Role: globals.Assistant, Content: "partial and complete", Model: globals.GPT3Turbo,
	}) {
		t.Fatal("save final response")
	}

	loaded := LoadConversation(db, 1, 1)
	if loaded == nil || len(loaded.Message) != 2 {
		t.Fatalf("expected user and one assistant message, got %#v", loaded)
	}
	assistant := loaded.Message[1]
	if assistant.MessageID != messageID || assistant.Content != "partial and complete" || assistant.Status != MessageStatusCompleted {
		t.Fatalf("unexpected final assistant: %#v", assistant)
	}

	var taskStatus string
	if err := globals.QueryRowDb(db, `
		SELECT status FROM generation_task WHERE user_id = ? AND task_id = ?
	`, 1, "request-1").Scan(&taskStatus); err != nil {
		t.Fatalf("load generation task: %v", err)
	}
	if taskStatus != MessageStatusCompleted {
		t.Fatalf("expected completed task, got %q", taskStatus)
	}
}

func TestConcurrentSnapshotsCannotOverwriteIndependentMessages(t *testing.T) {
	db := openIndependentMessageTestDB(t)
	seed := &Conversation{Auth: true, UserID: 1, Id: 1, Name: "chat", Model: globals.GPT3Turbo}
	seed.AddMessage(globals.Message{Role: globals.User, Content: "first", RequestID: "request-1"})
	if !seed.SaveConversation(db) {
		t.Fatal("save seed")
	}

	first := LoadConversation(db, 1, 1)
	second := LoadConversation(db, 1, 1)
	if !first.HandleMessage(db, &FormMessage{Message: "second", RequestID: "request-2", Model: globals.GPT3Turbo}) {
		t.Fatal("append second")
	}
	second.Name = "stale rename"
	if !second.SaveConversation(db) {
		t.Fatal("save stale snapshot")
	}

	loaded := LoadConversation(db, 1, 1)
	if loaded == nil || len(loaded.Message) != 2 {
		t.Fatalf("expected both messages after stale save, got %#v", loaded)
	}
	if loaded.Message[1].Content != "second" {
		t.Fatalf("expected independently persisted second message, got %#v", loaded.Message)
	}
}
