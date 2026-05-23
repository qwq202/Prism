package conversation

import (
	"chat/globals"
	"strings"
	"testing"
)

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
