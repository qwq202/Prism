package auth

import (
	"chat/channel"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type memoryOTPCache struct {
	values map[string]string
}

func newMemoryOTPCache() *memoryOTPCache {
	return &memoryOTPCache{values: map[string]string{}}
}

func (c *memoryOTPCache) Get(_ context.Context, key string) *redis.StringCmd {
	value, ok := c.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(value, nil)
}

func (c *memoryOTPCache) Set(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	c.values[key] = fmt.Sprint(value)
	return redis.NewStatusResult("OK", nil)
}

func (c *memoryOTPCache) Del(_ context.Context, keys ...string) *redis.IntCmd {
	var removed int64
	for _, key := range keys {
		if _, ok := c.values[key]; ok {
			delete(c.values, key)
			removed++
		}
	}
	return redis.NewIntResult(removed, nil)
}

func TestCheckCodeDoesNotConsumeOTPBeforeMutationSucceeds(t *testing.T) {
	ctx := context.Background()
	cache := newMemoryOTPCache()
	email := "user@example.com"

	setCode(ctx, cache, email, "123456")
	if !checkCode(ctx, cache, email, "123456") {
		t.Fatalf("expected first otp check to pass")
	}
	if !checkCode(ctx, cache, email, "123456") {
		t.Fatalf("expected otp to remain available until caller consumes it")
	}

	deleteCode(ctx, cache, email)
	if checkCode(ctx, cache, email, "123456") {
		t.Fatalf("expected consumed otp to be rejected")
	}
}

func TestValidateVerificationEmailForResetRequiresRegisteredEmail(t *testing.T) {
	db := openAuthSecurityTestDB(t)

	if err := validateVerificationEmail(db, "not-an-email", false, true); err == nil {
		t.Fatalf("expected invalid email to be rejected")
	}

	if err := validateVerificationEmail(db, "missing@example.com", false, true); err == nil {
		t.Fatalf("expected reset verification for unregistered email to be rejected")
	}

	if err := validateVerificationEmail(db, "root@example.com", false, true); err != nil {
		t.Fatalf("expected reset verification for registered email to pass: %v", err)
	}

	if err := validateVerificationEmail(db, "missing@example.com", false, false); err != nil {
		t.Fatalf("expected non-reset verification to allow unregistered test email: %v", err)
	}
}

func withSystemBackend(t *testing.T, backend string) {
	t.Helper()

	previousInstance := channel.SystemInstance
	if channel.SystemInstance == nil {
		channel.SystemInstance = &channel.SystemConfig{}
	}
	previousBackend := channel.SystemInstance.General.Backend
	channel.SystemInstance.General.Backend = backend

	t.Cleanup(func() {
		if previousInstance == nil {
			channel.SystemInstance = nil
			return
		}
		channel.SystemInstance = previousInstance
		channel.SystemInstance.General.Backend = previousBackend
	})
}

func TestBuildPasswordResetLinkUsesRequestOrigin(t *testing.T) {
	withSystemBackend(t, "")

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/verify", nil)
	c.Request.Header.Set("Origin", "https://prism.example.com")

	link, err := buildPasswordResetLink(c, "root@example.com", "reset-token")
	if err != nil {
		t.Fatalf("build reset link: %v", err)
	}

	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatalf("parse reset link: %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "prism.example.com" || parsed.Path != "/forgot" {
		t.Fatalf("unexpected reset link target: %s", link)
	}
	if parsed.Query().Get("email") != "root@example.com" || parsed.Query().Get("token") != "reset-token" {
		t.Fatalf("unexpected reset link query: %s", link)
	}
}

func TestBuildPasswordResetLinkStripsConfiguredBackendAPIPath(t *testing.T) {
	withSystemBackend(t, "https://prism.example.com/api")

	link, err := buildPasswordResetLink(&gin.Context{}, "root@example.com", "reset-token")
	if err != nil {
		t.Fatalf("build reset link: %v", err)
	}

	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatalf("parse reset link: %v", err)
	}
	if parsed.String() != "https://prism.example.com/forgot?email=root%40example.com&token=reset-token" {
		t.Fatalf("unexpected reset link: %s", link)
	}
}
