package auth

import (
	"chat/channel"
	"chat/globals"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "github.com/mattn/go-sqlite3"
)

func TestSubscriptionAPIReturnsSingleDisabledResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	previousPlan := channel.PlanInstance
	channel.PlanInstance = &channel.PlanManager{Enabled: false}
	t.Cleanup(func() {
		channel.PlanInstance = previousPlan
	})

	previousSqlite := globals.SqliteEngine
	globals.SqliteEngine = true
	t.Cleanup(func() {
		globals.SqliteEngine = previousSqlite
	})

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "subscription-api.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	cache := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		MaxRetries:   -1,
		DialTimeout:  time.Millisecond,
		ReadTimeout:  time.Millisecond,
		WriteTimeout: time.Millisecond,
	})
	t.Cleanup(func() {
		_ = cache.Close()
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/subscription", nil)
	ctx.Set("user", "alice")
	ctx.Set("db", db)
	ctx.Set("cache", cache)

	SubscriptionAPI(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	decoder := json.NewDecoder(recorder.Body)
	var payload map[string]any
	if err := decoder.Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if status, ok := payload["status"].(bool); !ok || !status {
		t.Fatalf("expected successful disabled subscription response, got %#v", payload)
	}
	if level, ok := payload["level"].(float64); !ok || level != 0 {
		t.Fatalf("expected disabled subscription level 0, got %#v", payload["level"])
	}
	if subscribed, ok := payload["is_subscribed"].(bool); !ok || subscribed {
		t.Fatalf("expected disabled subscription to report unsubscribed, got %#v", payload["is_subscribed"])
	}

	var extra map[string]any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatalf("expected a single JSON response, got extra payload %#v and err %v", extra, err)
	}
}
