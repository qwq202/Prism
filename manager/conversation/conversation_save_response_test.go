package conversation

import (
	"chat/connection"
	"chat/globals"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func openConversationTestDB(t *testing.T, filename string) *sql.DB {
	t.Helper()

	previous := globals.SqliteEngine
	globals.SqliteEngine = true
	t.Cleanup(func() {
		globals.SqliteEngine = previous
	})

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), filename))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	connection.CreateConversationTable(db)
	connection.CreateChatRequestTable(db)
	return db
}

func assertMessageContents(t *testing.T, messages []globals.Message, want []string) {
	t.Helper()

	if len(messages) != len(want) {
		t.Fatalf("expected %d messages, got %#v", len(want), messages)
	}
	for index, content := range want {
		if messages[index].Content != content {
			t.Fatalf("expected message %d content %q, got %#v", index, content, messages[index])
		}
	}
}

func TestSaveResponseSkipsMetadataOnlyAssistantReply(t *testing.T) {
	instance := NewAnonymousConversation()

	saved := instance.SaveResponse(nil, globals.Message{
		Content: "",
		GeminiHiddenMetadata: &globals.GeminiHiddenMetadata{
			ThoughtSignatures: []string{"sig-1"},
		},
	})

	if saved {
		t.Fatalf("expected metadata-only response not to be persisted")
	}

	if got := instance.GetMessageLength(); got != 0 {
		t.Fatalf("expected no messages to be persisted, got %d", got)
	}
}

func TestConversationMessageAccessorsHandleEmptyAndOutOfRangeIndexes(t *testing.T) {
	instance := NewAnonymousConversation()

	if got := instance.GetLastMessage(); got.Role != "" || got.Content != "" {
		t.Fatalf("expected empty last message on empty conversation, got %#v", got)
	}
	if got := instance.GetMessageById(0); got.Role != "" || got.Content != "" {
		t.Fatalf("expected empty message for out-of-range index, got %#v", got)
	}
	if instance.HasMessageId(0) {
		t.Fatalf("expected empty conversation not to have message id 0")
	}

	instance.AddMessage(globals.Message{Role: globals.User, Content: "hello"})
	if !instance.HasMessageId(0) {
		t.Fatalf("expected message id 0 to exist")
	}
	if instance.HasMessageId(-1) || instance.HasMessageId(1) {
		t.Fatalf("expected negative and out-of-range indexes to be rejected")
	}
	if got := instance.GetMessageById(0); got.Content != "hello" {
		t.Fatalf("expected valid message lookup, got %#v", got)
	}
}

func TestSaveResponsePersistsToolCallsWithoutText(t *testing.T) {
	instance := NewAnonymousConversation()
	calls := globals.ToolCalls{
		{
			Type: "function",
			Id:   "tool-call-1",
			Function: globals.ToolCallFunction{
				Name:      "lookup_weather",
				Arguments: "{\"city\":\"Shanghai\"}",
			},
		},
	}

	saved := instance.SaveResponse(nil, globals.Message{
		Role:      globals.User,
		Content:   "",
		ToolCalls: &calls,
	})

	if !saved {
		t.Fatalf("expected tool-call response to be persisted")
	}

	if got := instance.GetMessageLength(); got != 1 {
		t.Fatalf("expected one persisted message, got %d", got)
	}

	last := instance.GetLastMessage()
	if last.Role != globals.Assistant {
		t.Fatalf("expected role %q, got %q", globals.Assistant, last.Role)
	}

	if last.ToolCalls == nil || len(*last.ToolCalls) != 1 {
		t.Fatalf("expected one tool call in persisted message, got %#v", last.ToolCalls)
	}
}

func TestSaveResponsePersistsFunctionCallWithoutText(t *testing.T) {
	instance := NewAnonymousConversation()

	saved := instance.SaveResponse(nil, globals.Message{
		Content: "",
		FunctionCall: &globals.FunctionCall{
			Name:      "lookup_air_quality",
			Arguments: "{\"city\":\"Shanghai\"}",
		},
	})

	if !saved {
		t.Fatalf("expected function-call response to be persisted")
	}

	if got := instance.GetMessageLength(); got != 1 {
		t.Fatalf("expected one persisted message, got %d", got)
	}

	last := instance.GetLastMessage()
	if last.Role != globals.Assistant {
		t.Fatalf("expected role %q, got %q", globals.Assistant, last.Role)
	}

	if last.FunctionCall == nil || last.FunctionCall.Name != "lookup_air_quality" {
		t.Fatalf("expected function call payload to be preserved, got %#v", last.FunctionCall)
	}
}

func TestSaveResponsePersistsConversationModelOnAssistantReply(t *testing.T) {
	instance := NewAnonymousConversation()
	instance.SetModel("grok-4.20-reasoning")

	saved := instance.SaveResponse(nil, globals.Message{
		Content: "hello from grok",
	})

	if !saved {
		t.Fatalf("expected assistant response to be persisted")
	}

	last := instance.GetLastMessage()
	if last.Model != "grok-4.20-reasoning" {
		t.Fatalf("expected persisted model to be preserved, got %q", last.Model)
	}
}

func TestSaveResponseMergesWithStoredConversationFromStaleSnapshot(t *testing.T) {
	db := openConversationTestDB(t, "stale-response-merge.db")

	first := &Conversation{
		UserID:  1,
		Id:      1,
		Name:    "first snapshot",
		Model:   "model-first",
		Message: []globals.Message{{Role: globals.User, Content: "金针菇有几种"}},
	}
	if !first.SaveConversation(db) {
		t.Fatalf("expected first conversation snapshot to save")
	}

	second := LoadConversation(db, 1, 1)
	if second == nil {
		t.Fatalf("expected conversation to load")
	}
	second.Name = "second snapshot"
	second.Model = "model-second"
	second.AddMessage(globals.Message{Role: globals.User, Content: "介绍一下白色金针菇和黄色金针菇"})
	if !second.SaveConversation(db) {
		t.Fatalf("expected second conversation snapshot to save")
	}

	if !first.SaveResponse(db, globals.Message{Content: "金针菇主要分为白色和黄色两类"}) {
		t.Fatalf("expected stale first response to save")
	}

	loaded := LoadConversation(db, 1, 1)
	if loaded == nil {
		t.Fatalf("expected merged conversation to load")
	}
	assertMessageContents(t, loaded.GetMessage(), []string{
		"金针菇有几种",
		"金针菇主要分为白色和黄色两类",
		"介绍一下白色金针菇和黄色金针菇",
	})
	if loaded.Name != "second snapshot" {
		t.Fatalf("expected stored conversation name to be preserved, got %q", loaded.Name)
	}
	if loaded.Model != "model-second" {
		t.Fatalf("expected stored conversation model to be preserved, got %q", loaded.Model)
	}

	if !second.SaveResponse(db, globals.Message{Content: "白色更主流，黄色更小众"}) {
		t.Fatalf("expected second response to save")
	}

	loaded = LoadConversation(db, 1, 1)
	if loaded == nil {
		t.Fatalf("expected final conversation to load")
	}
	assertMessageContents(t, loaded.GetMessage(), []string{
		"金针菇有几种",
		"金针菇主要分为白色和黄色两类",
		"介绍一下白色金针菇和黄色金针菇",
		"白色更主流，黄色更小众",
	})
}

func TestSaveResponseDoesNotDuplicateMergedAssistantResponse(t *testing.T) {
	db := openConversationTestDB(t, "response-dedupe.db")

	instance := &Conversation{
		UserID:  1,
		Id:      1,
		Name:    defaultConversationName,
		Model:   globals.GPT3Turbo,
		Message: []globals.Message{{Role: globals.User, Content: "hello"}},
	}
	if !instance.SaveConversation(db) {
		t.Fatalf("expected conversation to save")
	}

	response := globals.Message{Content: "world"}
	if !instance.SaveResponse(db, response) {
		t.Fatalf("expected first response save")
	}

	stale := &Conversation{
		UserID:    1,
		Id:        1,
		Name:      defaultConversationName,
		Model:     globals.GPT3Turbo,
		Persisted: true,
		Message:   []globals.Message{{Role: globals.User, Content: "hello"}},
	}
	if !stale.SaveResponse(db, response) {
		t.Fatalf("expected duplicate stale response save to be treated as success")
	}

	loaded := LoadConversation(db, 1, 1)
	if loaded == nil {
		t.Fatalf("expected conversation to load")
	}
	assertMessageContents(t, loaded.GetMessage(), []string{"hello", "world"})
}

func TestHandleMessageMergesConcurrentPersistedSnapshots(t *testing.T) {
	db := openConversationTestDB(t, "concurrent-user-message.db")
	seed := &Conversation{
		UserID:  1,
		Id:      1,
		Name:    "chat",
		Model:   globals.GPT3Turbo,
		Message: []globals.Message{{Role: globals.User, Content: "seed"}},
	}
	if !seed.SaveConversation(db) {
		t.Fatalf("save seed conversation")
	}

	first := LoadConversation(db, 1, 1)
	second := LoadConversation(db, 1, 1)
	if first == nil || second == nil {
		t.Fatalf("load concurrent snapshots")
	}
	if !first.HandleMessage(db, &FormMessage{Message: "first", Model: globals.GPT3Turbo, RequestID: "request-first"}) {
		t.Fatalf("append first concurrent message")
	}
	if !second.HandleMessage(db, &FormMessage{Message: "second", Model: globals.GPT3Turbo, RequestID: "request-second"}) {
		t.Fatalf("append second concurrent message")
	}

	loaded := LoadConversation(db, 1, 1)
	if loaded == nil {
		t.Fatalf("load merged conversation")
	}
	assertMessageContents(t, loaded.GetMessage(), []string{"seed", "first", "second"})
}

func TestHandleMessagePersistsMaskContextWithDurableRequest(t *testing.T) {
	db := openConversationTestDB(t, "durable-mask-context.db")
	instance := NewConversation(db, 1)
	if !instance.HandleMessage(db, &FormMessage{
		Message:   "hello",
		Model:     globals.GPT3Turbo,
		RequestID: "request-with-mask",
		MaskContext: []globals.Message{
			{Role: globals.System, Content: "follow the mask"},
			{Role: globals.User, Content: "mask example"},
		},
	}) {
		t.Fatalf("persist masked request")
	}

	loaded := LoadConversation(db, 1, instance.GetId())
	if loaded == nil {
		t.Fatalf("load masked request")
	}
	assertMessageContents(t, loaded.GetMessage(), []string{"follow the mask", "mask example", "hello"})
	if loaded.GetLastMessage().RequestID != "request-with-mask" {
		t.Fatalf("expected request metadata on user message, got %#v", loaded.GetLastMessage())
	}
}

func TestHandleNewMessageBindsFinalConversationIDAtomically(t *testing.T) {
	db := openConversationTestDB(t, "atomic-new-request.db")
	instance := NewConversation(db, 1)
	record, owner, err := ReserveChatRequest(db, 1, "atomic-request", instance.GetId())
	if err != nil || !owner {
		t.Fatalf("reserve new request: owner=%v record=%#v err=%v", owner, record, err)
	}

	collision := &Conversation{
		UserID: 1, Id: instance.GetId(), Name: "collision", Model: globals.GPT3Turbo,
		Message: []globals.Message{{Role: globals.User, Content: "existing"}},
	}
	if !collision.SaveConversation(db) {
		t.Fatalf("create conversation id collision")
	}
	if !instance.HandleNewMessageWithRequest(db, &FormMessage{
		Message: "atomic message", Model: globals.GPT3Turbo, RequestID: "atomic-request",
	}, record.OwnerToken) {
		t.Fatalf("persist and bind new request")
	}
	if instance.GetId() == collision.GetId() {
		t.Fatalf("expected collision retry to allocate a final conversation id")
	}

	bound, err := LookupChatRequest(db, 1, "atomic-request")
	if err != nil || bound == nil {
		t.Fatalf("load bound request: %#v err=%v", bound, err)
	}
	if bound.ConversationID != instance.GetId() || bound.Status != ChatRequestAccepted {
		t.Fatalf("expected request mapping to commit with conversation, got %#v", bound)
	}
	loaded := LoadConversation(db, 1, instance.GetId())
	if loaded == nil || loaded.GetLastMessage().RequestID != "atomic-request" {
		t.Fatalf("expected the atomic user message in the final conversation, got %#v", loaded)
	}
}

func TestSaveResponseDoesNotRecreateDeletedConversation(t *testing.T) {
	db := openConversationTestDB(t, "deleted-response.db")
	instance := &Conversation{
		UserID:  1,
		Id:      1,
		Name:    "chat",
		Model:   globals.GPT3Turbo,
		Message: []globals.Message{{Role: globals.User, Content: "delete me"}},
	}
	if !instance.SaveConversation(db) {
		t.Fatalf("save conversation")
	}
	if !instance.DeleteConversation(db) {
		t.Fatalf("delete conversation")
	}
	if instance.SaveResponse(db, globals.Message{Content: "late response"}) {
		t.Fatalf("late response must not recreate a deleted conversation")
	}
	if loaded := LoadConversation(db, 1, 1); loaded != nil {
		t.Fatalf("expected deleted conversation to stay deleted, got %#v", loaded)
	}
}

func TestSaveConversationQueryUpdatesModelColumn(t *testing.T) {
	if !strings.Contains(saveConversationQuery, "model = VALUES(model)") {
		t.Fatalf("expected save conversation query to update model column, got %q", saveConversationQuery)
	}
	if !strings.Contains(saveConversationQuery, "updated_at = CURRENT_TIMESTAMP") {
		t.Fatalf("expected save conversation query to bump updated_at, got %q", saveConversationQuery)
	}
}

func TestSaveConversationQuerySqlitePreflightUpdatesModelColumn(t *testing.T) {
	previous := globals.SqliteEngine
	globals.SqliteEngine = true
	t.Cleanup(func() {
		globals.SqliteEngine = previous
	})

	query := globals.PreflightSql(saveConversationQuery)
	if !strings.Contains(query, "model = excluded.model") {
		t.Fatalf("expected sqlite save conversation query to update model column, got %q", query)
	}
	if !strings.Contains(query, "updated_at = CURRENT_TIMESTAMP") {
		t.Fatalf("expected sqlite save conversation query to bump updated_at, got %q", query)
	}
	if strings.Contains(query, "DUPLICATE KEY") {
		t.Fatalf("expected sqlite save conversation query to remove mysql upsert syntax, got %q", query)
	}
}

func TestNewConversationRetriesOnConversationIDCollision(t *testing.T) {
	db := openConversationTestDB(t, "conversation-collision.db")

	first := &Conversation{
		UserID:  1,
		Id:      1,
		Name:    "first",
		Model:   globals.GPT3Turbo,
		Message: []globals.Message{{Role: globals.User, Content: "first"}},
	}
	if !first.SaveConversation(db) {
		t.Fatalf("expected first conversation to save")
	}

	second := &Conversation{
		UserID:  1,
		Id:      1,
		Name:    "second",
		Model:   globals.GPT3Turbo,
		Message: []globals.Message{{Role: globals.User, Content: "second"}},
	}
	if !second.SaveConversation(db) {
		t.Fatalf("expected second conversation to retry and save")
	}
	if second.Id == first.Id {
		t.Fatalf("expected second conversation id to change after collision")
	}

	loadedFirst := LoadConversation(db, 1, first.Id)
	if loadedFirst == nil || loadedFirst.GetMessage()[0].Content != "first" {
		t.Fatalf("expected first conversation to remain intact, got %#v", loadedFirst)
	}

	loadedSecond := LoadConversation(db, 1, second.Id)
	if loadedSecond == nil || loadedSecond.GetMessage()[0].Content != "second" {
		t.Fatalf("expected retried second conversation to persist, got %#v", loadedSecond)
	}
}

func TestDefaultConversationContextIsFive(t *testing.T) {
	instance := NewAnonymousConversation()
	if got := instance.GetContextLength(); got != 5 {
		t.Fatalf("expected default context length 5, got %d", got)
	}
}

func TestConversationContextLengthBounds(t *testing.T) {
	if got := normalizeContextLength(1); got != 5 {
		t.Fatalf("expected generic normalization to clamp 1 to 5, got %d", got)
	}

	instance := NewAnonymousConversation()

	instance.SetContextLength(1, false)
	if got := instance.GetContextLength(); got != 5 {
		t.Fatalf("expected non-ignore context length 1 to clamp to 5, got %d", got)
	}

	instance.SetContextLength(3, false)
	if got := instance.GetContextLength(); got != 5 {
		t.Fatalf("expected context length below minimum to clamp to 5, got %d", got)
	}

	instance.SetContextLength(30, false)
	if got := instance.GetContextLength(); got != 25 {
		t.Fatalf("expected context length above maximum to clamp to 25, got %d", got)
	}
}

func TestGetChatMessageTruncatesCleanedHistory(t *testing.T) {
	instance := NewAnonymousConversation()
	instance.SetContextLength(5, false)
	instance.Message = []globals.Message{
		{Role: globals.User, Content: "u1"},
		{Role: globals.Assistant, Content: "a1"},
		{Role: globals.Assistant, Content: "   "},
		{Role: globals.User, Content: "u2"},
		{Role: globals.Assistant, Content: "a2"},
		{Role: globals.User, Content: "u3"},
		{Role: globals.Assistant, Content: "a3"},
		{Role: globals.User, Content: "u4"},
	}

	got := instance.GetChatMessage(false)
	if len(got) != 5 {
		t.Fatalf("expected 5 context messages, got %#v", got)
	}

	want := []string{"u2", "a2", "u3", "a3", "u4"}
	for index, content := range want {
		if got[index].Content != content {
			t.Fatalf("expected message %d content %q, got %#v", index, content, got)
		}
	}
}

func TestGetChatMessageStartsAfterLastContextClear(t *testing.T) {
	instance := NewAnonymousConversation()
	instance.SetContextLength(10, false)
	instance.Message = []globals.Message{
		{Role: globals.User, Content: "old user"},
		{Role: globals.Assistant, Content: "old assistant"},
		{Role: globals.User, Content: "fresh user", ContextCleared: true},
		{Role: globals.Assistant, Content: "fresh assistant"},
		{Role: globals.User, Content: "current user"},
	}

	got := instance.GetChatMessage(false)
	if len(got) != 3 {
		t.Fatalf("expected messages after context clear, got %#v", got)
	}

	if got[0].Content != "fresh user" || !got[0].ContextCleared {
		t.Fatalf("expected context clear message first, got %#v", got[0])
	}
	if got[1].Content != "fresh assistant" || got[2].Content != "current user" {
		t.Fatalf("unexpected messages after context clear: %#v", got)
	}
}

func TestGetChatMessageDropsAbandonedConsecutiveAssistantReply(t *testing.T) {
	instance := NewAnonymousConversation()
	instance.SetContextLength(10, false)
	instance.Message = []globals.Message{
		{Role: globals.User, Content: "question"},
		{Role: globals.Assistant, Content: "old answer"},
		{Role: globals.Assistant, Content: "regenerated answer"},
		{Role: globals.User, Content: "follow up"},
	}

	got := instance.GetChatMessage(false)
	if len(got) != 3 {
		t.Fatalf("expected one abandoned assistant reply to be dropped, got %#v", got)
	}

	if got[1].Content != "regenerated answer" {
		t.Fatalf("expected regenerated answer to be kept, got %#v", got)
	}
}

func TestAddMessageFromFormMarksContextClear(t *testing.T) {
	instance := NewAnonymousConversation()
	form := &FormMessage{
		Message:       " reset here ",
		IgnoreContext: true,
	}

	if err := instance.AddMessageFromForm(form); err != nil {
		t.Fatalf("unexpected add message error: %v", err)
	}

	got := instance.GetLastMessage()
	if got.Content != "reset here" {
		t.Fatalf("expected trimmed user content, got %q", got.Content)
	}
	if !got.ContextCleared {
		t.Fatalf("expected context clear marker on user message")
	}
	if instance.GetContextLength() != 1 {
		t.Fatalf("expected ignore context to use current message only, got %d", instance.GetContextLength())
	}
}
