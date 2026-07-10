package personalization

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newPersonalizationAPIContext(
	t *testing.T,
	method string,
	body string,
) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	db := openPersonalizationTestDB(t)
	if _, err := db.Exec(`CREATE TABLE auth (id INTEGER PRIMARY KEY, username TEXT UNIQUE)`); err != nil {
		t.Fatalf("create auth table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO auth (id, username) VALUES (7, 'alice')`); err != nil {
		t.Fatalf("insert auth user: %v", err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, "/personalization", strings.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("db", db)
	context.Set("auth", true)
	context.Set("user", "alice")
	return context, recorder
}

func decodePersonalizationAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return response
}

func TestSaveAndLoadPersonalizationAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	saveContext, saveRecorder := newPersonalizationAPIContext(t, http.MethodPost, `{
		"settings": {
			"persona_style": "friendly",
			"persona_warmth": "high",
			"memory_enabled": true
		},
		"base_revision": 0
	}`)
	SaveAPI(saveContext)
	if response := decodePersonalizationAPIResponse(t, saveRecorder); response["status"] != true {
		t.Fatalf("expected save to succeed, got %#v", response)
	}

	loadRecorder := httptest.NewRecorder()
	loadContext, _ := gin.CreateTestContext(loadRecorder)
	loadContext.Request = httptest.NewRequest(http.MethodGet, "/personalization", nil)
	loadContext.Set("db", saveContext.MustGet("db"))
	loadContext.Set("auth", true)
	loadContext.Set("user", "alice")
	LoadAPI(loadContext)

	response := decodePersonalizationAPIResponse(t, loadRecorder)
	if response["status"] != true {
		t.Fatalf("expected load to succeed, got %#v", response)
	}
	data, ok := response["data"].(map[string]interface{})
	if !ok || data["revision"] != float64(1) {
		t.Fatalf("expected revision 1, got %#v", response["data"])
	}
	settings, ok := data["settings"].(map[string]interface{})
	if !ok || settings["persona_style"] != "friendly" || settings["memory_enabled"] != true {
		t.Fatalf("unexpected saved settings: %#v", data["settings"])
	}
}

func TestSavePersonalizationAPIRejectsMissingSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, recorder := newPersonalizationAPIContext(t, http.MethodPost, `{}`)
	SaveAPI(context)

	response := decodePersonalizationAPIResponse(t, recorder)
	if response["status"] != false {
		t.Fatalf("expected missing settings to fail, got %#v", response)
	}
}
