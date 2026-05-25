package utils

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func TestRegisterStaticRouteSetsCacheHeaders(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("restore wd: %v", err)
		}
		viper.Reset()
	})

	viper.Reset()
	viper.Set("serve_static", true)
	gin.SetMode(gin.TestMode)

	root := t.TempDir()
	dist := filepath.Join(root, "app", "dist")
	if err := os.MkdirAll(filepath.Join(dist, "assets"), 0o755); err != nil {
		t.Fatalf("create dist: %v", err)
	}
	writeTestFile(t, filepath.Join(dist, "index.html"), "<html><head><title>Prism</title></head><body>chatnio</body></html>")
	writeTestFile(t, filepath.Join(dist, "site.webmanifest"), `{"name":"Prism"}`)
	writeTestFile(t, filepath.Join(dist, "assets", "app.123.js"), "console.log('ok');")
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir temp root: %v", err)
	}

	router := gin.New()
	RegisterStaticRoute(router)

	tests := []struct {
		name         string
		path         string
		cacheControl string
	}{
		{name: "root", path: "/", cacheControl: staticPageCacheControl},
		{name: "spa fallback", path: "/settings/profile", cacheControl: staticPageCacheControl},
		{name: "manifest", path: "/site.webmanifest", cacheControl: staticManifestCacheControl},
		{name: "direct cached manifest", path: "/site.cache.webmanifest", cacheControl: staticManifestCacheControl},
		{name: "fingerprinted asset", path: "/assets/app.123.js", cacheControl: staticImmutableCacheControl},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if got := rec.Header().Get("Cache-Control"); got != tt.cacheControl {
				t.Fatalf("unexpected cache-control for %s: got %q want %q", tt.path, got, tt.cacheControl)
			}
		})
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
