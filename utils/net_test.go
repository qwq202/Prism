package utils

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFormatBodyForLogTruncatesLargeText(t *testing.T) {
	content := strings.Repeat("a", maxHTTPLogBodyBytes+100)
	got := formatBodyForLog([]byte(content), "text/plain")

	if len(got) >= len(content) {
		t.Fatalf("expected logged body to be truncated")
	}
	if !strings.Contains(got, "[truncated to 64.0 KB]") {
		t.Fatalf("expected truncation notice, got %q", got[len(got)-80:])
	}
}

func TestFormatBodyForLogSummarizesBinary(t *testing.T) {
	got := formatBodyForLog([]byte{0, 1, 2, 3, 4}, "application/octet-stream")
	if !strings.Contains(got, "[Binary Content]") {
		t.Fatalf("expected binary content summary, got %q", got)
	}
}

func TestFormatBodyForLogRedactsGeminiInlineData(t *testing.T) {
	imageData := strings.Repeat("A", minBase64LogRedactLength+20)
	body := `{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":"` + imageData + `"}}]}}]}`

	got := formatBodyForLog([]byte(body), "application/json")
	if strings.Contains(got, imageData) {
		t.Fatalf("expected inline base64 data to be redacted, got %q", got)
	}
	if !strings.Contains(got, "[base64 omitted") {
		t.Fatalf("expected base64 redaction notice, got %q", got)
	}
	if !strings.Contains(got, `"mimeType": "image/png"`) {
		t.Fatalf("expected non-sensitive metadata to remain, got %q", got)
	}
}

func TestFormatBodyForLogRedactsDataURLBase64Text(t *testing.T) {
	imageData := strings.Repeat("B", minBase64LogRedactLength+20)
	body := "before data:image/png;base64," + imageData + " after"

	got := formatBodyForLog([]byte(body), "text/plain")
	if strings.Contains(got, imageData) {
		t.Fatalf("expected data URL base64 to be redacted, got %q", got)
	}
	if !strings.Contains(got, "data:image/png;base64,[base64 omitted") {
		t.Fatalf("expected data URL prefix and redaction notice, got %q", got)
	}
}

func TestFormatBodyForLogKeepsLongTextData(t *testing.T) {
	text := strings.Repeat("ordinary words ", 40)
	body := `{"data":"` + text + `"}`

	got := formatBodyForLog([]byte(body), "application/json")
	if strings.Contains(got, "[base64 omitted") {
		t.Fatalf("did not expect ordinary text data to be redacted, got %q", got)
	}
	if !strings.Contains(got, text) {
		t.Fatalf("expected ordinary text data to remain, got %q", got)
	}
}

func TestFormatBodyForLogRedactsSSEDataJSON(t *testing.T) {
	imageData := strings.Repeat("D", minBase64LogRedactLength+20)
	body := `data: {"inlineData":{"data":"` + imageData + `"}}`

	got := formatBodyForLog([]byte(body), "text/event-stream")
	if strings.Contains(got, imageData) {
		t.Fatalf("expected SSE inline base64 data to be redacted, got %q", got)
	}
	if !strings.Contains(got, "data:") || !strings.Contains(got, "[base64 omitted") {
		t.Fatalf("expected SSE data prefix and redaction notice, got %q", got)
	}
}

func TestFormatHeadersForLogRedactsSecrets(t *testing.T) {
	got := formatHeadersForLog(map[string]string{
		"Content-Type":   "application/json",
		"x-goog-api-key": "secret-key",
		"Authorization":  "Bearer secret-token",
	})

	if strings.Contains(got, "secret-key") || strings.Contains(got, "secret-token") {
		t.Fatalf("expected sensitive headers to be redacted, got %q", got)
	}
	if !strings.Contains(got, `"Content-Type":"application/json"`) {
		t.Fatalf("expected non-sensitive header to remain, got %q", got)
	}
}

func TestFormatURIForLogRedactsSecrets(t *testing.T) {
	got := formatURIForLog("https://example.com/v1beta/models/gemini:generateContent?key=secret-key&alt=json")
	if strings.Contains(got, "secret-key") {
		t.Fatalf("expected sensitive query value to be redacted, got %q", got)
	}
	if !strings.Contains(got, "alt=json") {
		t.Fatalf("expected non-sensitive query value to remain, got %q", got)
	}
}

func TestReadErrorBodyCapsLargeResponse(t *testing.T) {
	body := strings.NewReader(strings.Repeat("x", maxHTTPErrorBodyBytes+1))
	data, truncated, err := readErrorBody(body)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if !truncated {
		t.Fatalf("expected error body to be marked truncated")
	}
	if len(data) != maxHTTPErrorBodyBytes {
		t.Fatalf("expected capped body length %d, got %d", maxHTTPErrorBodyBytes, len(data))
	}
}

func TestGetErrorBodyIncludesTruncationNotice(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(strings.Repeat("x", maxHTTPErrorBodyBytes+1))),
	}

	got := getErrorBody(resp)
	if !strings.Contains(got, "[truncated to 1.0 MB]") {
		t.Fatalf("expected error body truncation notice")
	}
}

func TestGetErrorBodyRedactsInlineBase64(t *testing.T) {
	imageData := strings.Repeat("C", minBase64LogRedactLength+20)
	resp := &http.Response{
		Header: http.Header{"Content-Type": []string{"application/json"}},
		Body:   io.NopCloser(strings.NewReader(`{"inlineData":{"data":"` + imageData + `"}}`)),
	}

	got := getErrorBody(resp)
	if strings.Contains(got, imageData) {
		t.Fatalf("expected error body inline base64 to be redacted, got %q", got)
	}
	if !strings.Contains(got, "[base64 omitted") {
		t.Fatalf("expected base64 redaction notice, got %q", got)
	}
}
