package conversation

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestConversationDestructiveRoutesSupportPostAndLegacyGet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	Register(router.Group(""))

	routes := make(map[string]bool)
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	for _, route := range []string{
		"POST /conversation/delete",
		"GET /conversation/delete",
		"POST /conversation/clean",
		"GET /conversation/clean",
		"POST /conversation/share/delete",
		"GET /conversation/share/delete",
	} {
		if !routes[route] {
			t.Fatalf("expected route %s to be registered", route)
		}
	}
}

func TestDeleteConversationIDAcceptsPostJSONAndLegacyQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	postContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	postContext.Request = httptest.NewRequest(http.MethodPost, "/conversation/delete", strings.NewReader(`{"id":42}`))
	postContext.Request.Header.Set("Content-Type", "application/json")

	id, err := getDeleteConversationID(postContext)
	if err != nil {
		t.Fatalf("expected POST JSON id to parse: %v", err)
	}
	if id != 42 {
		t.Fatalf("expected POST JSON id 42, got %d", id)
	}

	getContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	getContext.Request = httptest.NewRequest(http.MethodGet, "/conversation/delete?id=7", nil)

	id, err = getDeleteConversationID(getContext)
	if err != nil {
		t.Fatalf("expected legacy query id to parse: %v", err)
	}
	if id != 7 {
		t.Fatalf("expected legacy query id 7, got %d", id)
	}
}

func TestDeleteSharingHashAcceptsPostJSONAndLegacyQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	postContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	postContext.Request = httptest.NewRequest(http.MethodPost, "/conversation/share/delete", strings.NewReader(`{"hash":" abc123 "}`))
	postContext.Request.Header.Set("Content-Type", "application/json")

	if got := getDeleteSharingHash(postContext); got != "abc123" {
		t.Fatalf("expected trimmed POST JSON hash, got %q", got)
	}

	getContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	getContext.Request = httptest.NewRequest(http.MethodGet, "/conversation/share/delete?hash=legacy", nil)

	if got := getDeleteSharingHash(getContext); got != "legacy" {
		t.Fatalf("expected legacy query hash, got %q", got)
	}
}
