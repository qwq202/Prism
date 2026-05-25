package generation

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestProjectDownloadRejectsInvalidHash(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/generation/download/zip?hash=../../config/config", nil)

	ProjectZipDownloadAPI(ctx)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected invalid generation download hash to be rejected, got %d", recorder.Code)
	}
	if disposition := recorder.Header().Get("Content-Disposition"); disposition != "" {
		t.Fatalf("expected no download header for invalid hash, got %q", disposition)
	}
}

func TestProjectDownloadRejectsMissingFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/generation/download/zip?hash="+strings.Repeat("a", 64), nil)

	ProjectZipDownloadAPI(ctx)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected missing generation download file to be rejected, got %d", recorder.Code)
	}
	if disposition := recorder.Header().Get("Content-Disposition"); disposition != "" {
		t.Fatalf("expected no download header for missing file, got %q", disposition)
	}
}
