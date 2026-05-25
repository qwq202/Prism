package auth

import (
	"bytes"
	"chat/globals"
	"chat/utils"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUpdateAccountPasswordAPIReturnsFreshToken(t *testing.T) {
	db := openAuthSecurityTestDB(t)
	setupCtx := newAuthSecurityTestContext(t, db)
	cache := utils.GetCacheFromContext(setupCtx)

	root := &User{Username: "root"}
	if err := root.UpdatePassword(db, cache, "oldsecret123"); err != nil {
		t.Fatalf("set known root password: %v", err)
	}

	var oldHash string
	if err := globals.QueryRowDb(db, "SELECT password FROM auth WHERE username = ?", "root").Scan(&oldHash); err != nil {
		t.Fatalf("fetch root password hash: %v", err)
	}

	user := &User{
		Username: "root",
		Password: oldHash,
	}
	oldToken, err := user.GenerateToken()
	if err != nil {
		t.Fatalf("generate old token: %v", err)
	}

	body := bytes.NewBufferString(`{"old_password":"oldsecret123","password":"newsecret123"}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/account/password", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("user", "root")
	ctx.Set("db", db)
	ctx.Set("cache", cache)

	UpdateAccountPasswordAPI(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var payload struct {
		Status bool   `json:"status"`
		Error  string `json:"error"`
		Token  string `json:"token"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Status {
		t.Fatalf("expected password update to succeed, got error %q", payload.Error)
	}
	if payload.Token == "" {
		t.Fatalf("expected password update response to include a fresh token")
	}

	validateCtx := newAuthSecurityTestContext(t, db)
	if parsed := ParseToken(validateCtx, oldToken); parsed != nil {
		t.Fatalf("expected old token to be rejected after password change")
	}
	if parsed := ParseToken(validateCtx, payload.Token); parsed == nil {
		t.Fatalf("expected returned token to validate after password change")
	}
}
