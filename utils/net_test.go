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
