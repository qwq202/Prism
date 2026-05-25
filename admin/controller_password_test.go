package admin

import (
	"bytes"
	"chat/auth"
	"chat/globals"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
)

func setAdminControllerTestSecret(t *testing.T) {
	t.Helper()

	previousSecret := viper.GetString("secret")
	viper.Set("secret", strings.Repeat("s", 32))
	t.Cleanup(func() {
		viper.Set("secret", previousSecret)
	})
}

func newAdminControllerTestContext(t *testing.T, db *sql.DB, username string, body string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/admin/user/password", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("db", db)
	ctx.Set("cache", (*redis.Client)(nil))
	ctx.Set("user", username)

	return ctx, recorder
}

func decodeAdminPasswordResponse(t *testing.T, recorder *httptest.ResponseRecorder) struct {
	Status bool   `json:"status"`
	Error  string `json:"error"`
	Token  string `json:"token"`
} {
	t.Helper()

	var payload struct {
		Status bool   `json:"status"`
		Error  string `json:"error"`
		Token  string `json:"token"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return payload
}

func TestUpdatePasswordAPIReturnsFreshTokenForCurrentUser(t *testing.T) {
	setAdminControllerTestSecret(t)
	db := openAdminUserTestDB(t)

	if err := createUser(db, "alice", "alice@example.com", "oldsecret123"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := setAdmin(db, 2, true); err != nil {
		t.Fatalf("promote user: %v", err)
	}

	var oldHash string
	if err := globals.QueryRowDb(db, "SELECT password FROM auth WHERE username = ?", "alice").Scan(&oldHash); err != nil {
		t.Fatalf("fetch old password hash: %v", err)
	}
	oldToken, err := (&auth.User{Username: "alice", Password: oldHash}).GenerateToken()
	if err != nil {
		t.Fatalf("generate old token: %v", err)
	}

	ctx, recorder := newAdminControllerTestContext(t, db, "alice", `{"id":2,"password":"newsecret123"}`)
	UpdatePasswordAPI(ctx)

	payload := decodeAdminPasswordResponse(t, recorder)
	if !payload.Status {
		t.Fatalf("expected password update to succeed, got error %q", payload.Error)
	}
	if payload.Token == "" {
		t.Fatalf("expected current user password update to return a fresh token")
	}
	if parsed := auth.ParseToken(ctx, oldToken); parsed != nil {
		t.Fatalf("expected old token to be rejected after admin password update")
	}
	if parsed := auth.ParseToken(ctx, payload.Token); parsed == nil {
		t.Fatalf("expected returned token to validate after admin password update")
	}
}

func TestUpdateRootPasswordAPIReturnsFreshTokenForRoot(t *testing.T) {
	setAdminControllerTestSecret(t)
	db := openAdminUserTestDB(t)

	var oldHash string
	if err := globals.QueryRowDb(db, "SELECT password FROM auth WHERE username = ?", "root").Scan(&oldHash); err != nil {
		t.Fatalf("fetch old root password hash: %v", err)
	}
	oldToken, err := (&auth.User{Username: "root", Password: oldHash}).GenerateToken()
	if err != nil {
		t.Fatalf("generate old token: %v", err)
	}

	ctx, recorder := newAdminControllerTestContext(t, db, "root", `{"password":"newsecret123"}`)
	UpdateRootPasswordAPI(ctx)

	payload := decodeAdminPasswordResponse(t, recorder)
	if !payload.Status {
		t.Fatalf("expected root password update to succeed, got error %q", payload.Error)
	}
	if payload.Token == "" {
		t.Fatalf("expected root password update to return a fresh token")
	}
	if parsed := auth.ParseToken(ctx, oldToken); parsed != nil {
		t.Fatalf("expected old root token to be rejected after root password update")
	}
	if parsed := auth.ParseToken(ctx, payload.Token); parsed == nil {
		t.Fatalf("expected returned root token to validate after password update")
	}
}
