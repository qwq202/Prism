package personalization

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterAddsPersonalizationRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	Register(engine.Group(""))

	routes := map[string]bool{}
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	for _, route := range []string{
		http.MethodGet + " /personalization",
		http.MethodPost + " /personalization",
	} {
		if !routes[route] {
			t.Fatalf("expected route %s to be registered", route)
		}
	}
}
