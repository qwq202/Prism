package utils

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func withRemoteImageResponse(t *testing.T, response *http.Response) {
	t.Helper()

	previous := remoteImageHTTPClient
	remoteImageHTTPClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return response, nil
		}),
	}
	t.Cleanup(func() {
		remoteImageHTTPClient = previous
	})
}

func TestReadRemoteImageBytesRejectsLargeContentLength(t *testing.T) {
	withRemoteImageResponse(t, &http.Response{
		StatusCode:    http.StatusOK,
		Header:        make(http.Header),
		ContentLength: 5,
		Body:          io.NopCloser(strings.NewReader("")),
	})

	if _, _, err := readRemoteImageBytes("https://example.com/image.png", 4); err == nil {
		t.Fatalf("expected oversized content length to be rejected")
	}
}

func TestReadRemoteImageBytesRejectsOversizedStream(t *testing.T) {
	header := make(http.Header)
	header.Set("Content-Type", "image/png")
	withRemoteImageResponse(t, &http.Response{
		StatusCode:    http.StatusOK,
		Header:        header,
		ContentLength: -1,
		Body:          io.NopCloser(strings.NewReader("12345")),
	})

	if _, _, err := readRemoteImageBytes("https://example.com/image.png", 4); err == nil {
		t.Fatalf("expected oversized response body to be rejected")
	}
}

func TestReadRemoteImageBytesReturnsContentType(t *testing.T) {
	header := make(http.Header)
	header.Set("Content-Type", "image/png; charset=binary")
	withRemoteImageResponse(t, &http.Response{
		StatusCode:    http.StatusOK,
		Header:        header,
		ContentLength: 3,
		Body:          io.NopCloser(strings.NewReader("png")),
	})

	data, contentType, err := readRemoteImageBytes("https://example.com/image.png", 4)
	if err != nil {
		t.Fatalf("read remote image: %v", err)
	}
	if string(data) != "png" {
		t.Fatalf("unexpected body: %q", string(data))
	}
	if contentType != "image/png" {
		t.Fatalf("expected normalized content type, got %q", contentType)
	}
}

func TestOpenRemoteImageResponseRejectsInvalidTargets(t *testing.T) {
	if _, err := openRemoteImageResponse("file:///tmp/image.png"); err == nil {
		t.Fatalf("expected unsupported scheme to be rejected")
	}

	withRemoteImageResponse(t, &http.Response{
		StatusCode:    http.StatusNotFound,
		Header:        make(http.Header),
		ContentLength: -1,
		Body:          io.NopCloser(strings.NewReader("not found")),
	})

	_, err := openRemoteImageResponse("https://example.com/missing.png")
	if err == nil || !strings.Contains(err.Error(), "unexpected status code") {
		t.Fatalf("expected non-2xx response to be rejected, got %v", err)
	}
}
