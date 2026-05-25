package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWarmupAPIRejectsTooManyUrls(t *testing.T) {
	gin.SetMode(gin.TestMode)

	urls := make([]string, maxWarmupUrls+1)
	for i := range urls {
		urls[i] = "https://example.com"
	}
	payload, err := json.Marshal(WarmupForm{Urls: urls})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/warmup", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")

	WarmupAPI(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var body struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status || body.Message != "too many urls" {
		t.Fatalf("expected too many urls rejection, got %#v", body)
	}
}

func TestValidateWarmupURLRejectsUnsafeTargets(t *testing.T) {
	unsafeTargets := []string{
		"",
		"file:///etc/passwd",
		"ftp://example.com/file",
		"https://user:pass@example.com",
		"http://localhost:8080",
		"http://service.local",
		"http://127.0.0.1:8080",
		"http://10.0.0.1",
		"http://172.16.0.1",
		"http://192.168.0.1",
		"http://100.64.0.1",
		"http://169.254.169.254",
		"http://[::1]",
	}

	for _, target := range unsafeTargets {
		if _, err := validateWarmupURL(target); err == nil {
			t.Fatalf("expected warmup url %q to be rejected", target)
		}
	}
}

func TestValidateWarmupURLAllowsPublicHTTPTargets(t *testing.T) {
	for _, target := range []string{
		"https://example.com",
		"http://example.com/assets/app.js",
	} {
		if _, err := validateWarmupURL(target); err != nil {
			t.Fatalf("expected warmup url %q to be allowed: %v", target, err)
		}
	}
}

func TestWarmupDialContextRejectsPrivateLiteralIPBeforeDialing(t *testing.T) {
	_, err := warmupDialContext(context.Background(), "tcp", net.JoinHostPort("127.0.0.1", "80"))
	if err == nil || !strings.Contains(err.Error(), "unsupported url host") {
		t.Fatalf("expected private warmup host to be rejected before dialing, got %v", err)
	}
}

func TestRedeemListAPIRejectsInvalidPage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/redeem?page=invalid", nil)

	RedeemListAPI(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var body struct {
		Status bool   `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status || body.Error != "invalid page" {
		t.Fatalf("expected invalid page rejection, got %#v", body)
	}
}
