package memory

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMemoryDeleteRouteSupportsPostAndLegacyGet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	Register(router.Group(""))

	routes := make(map[string]bool)
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	for _, route := range []string{
		"POST /memory/delete",
		"GET /memory/delete",
	} {
		if !routes[route] {
			t.Fatalf("expected route %s to be registered", route)
		}
	}
}

func TestMemoryDeleteIDAcceptsPostJSONAndLegacyQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	postContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	postContext.Request = httptest.NewRequest(http.MethodPost, "/memory/delete", strings.NewReader(`{"id":42}`))
	postContext.Request.Header.Set("Content-Type", "application/json")

	id, err := getDeleteID(postContext)
	if err != nil {
		t.Fatalf("expected POST JSON id to parse: %v", err)
	}
	if id != 42 {
		t.Fatalf("expected POST JSON id 42, got %d", id)
	}

	getContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	getContext.Request = httptest.NewRequest(http.MethodGet, "/memory/delete?id=7", nil)

	id, err = getDeleteID(getContext)
	if err != nil {
		t.Fatalf("expected legacy query id to parse: %v", err)
	}
	if id != 7 {
		t.Fatalf("expected legacy query id 7, got %d", id)
	}
}
