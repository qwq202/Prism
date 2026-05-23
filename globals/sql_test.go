package globals

import (
	"strings"
	"testing"
)

func TestPreflightSqlConvertsConversationUpsertWithModel(t *testing.T) {
	previous := SqliteEngine
	SqliteEngine = true
	t.Cleanup(func() {
		SqliteEngine = previous
	})

	query := `
		INSERT INTO conversation (user_id, conversation_id, conversation_name, data, model, task_id) VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE conversation_name = VALUES(conversation_name), data = VALUES(data), model = VALUES(model), task_id = VALUES(task_id), updated_at = CURRENT_TIMESTAMP
	`

	got := PreflightSql(query)
	if strings.Contains(got, "ON DUPLICATE KEY") || strings.Contains(got, "VALUES(model)") {
		t.Fatalf("expected sqlite-compatible upsert, got %q", got)
	}
	if !strings.Contains(got, "ON CONFLICT(user_id, conversation_id)") {
		t.Fatalf("expected conflict target in sqlite upsert, got %q", got)
	}
	if !strings.Contains(got, "model = excluded.model") {
		t.Fatalf("expected model column to be updated in sqlite upsert, got %q", got)
	}
	if !strings.Contains(got, "updated_at = CURRENT_TIMESTAMP") {
		t.Fatalf("expected updated_at to be bumped in sqlite upsert, got %q", got)
	}
}
