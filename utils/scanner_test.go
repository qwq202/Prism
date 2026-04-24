package utils

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type scannerTestReadCloser struct {
	reader   io.Reader
	closeErr error
}

func (r *scannerTestReadCloser) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *scannerTestReadCloser) Close() error {
	return r.closeErr
}

type scannerErrorReader struct {
	err error
}

func (r *scannerErrorReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestProcessLegacySSEPreservesCallbackErrorWhenCloseFails(t *testing.T) {
	callbackErr := errors.New("callback failed")
	closeErr := errors.New("close failed")
	body := &scannerTestReadCloser{
		reader:   strings.NewReader("data: {\"ok\":true}\n"),
		closeErr: closeErr,
	}

	err := processLegacySSE(body, func(string) error {
		return callbackErr
	})
	if err == nil {
		t.Fatalf("expected scanner error")
	}
	if !errors.Is(err.Error, callbackErr) {
		t.Fatalf("expected callback error to be preserved, got %v", err.Error)
	}
}

func TestProcessLegacySSEReturnsScannerError(t *testing.T) {
	readErr := errors.New("read failed")
	body := &scannerTestReadCloser{reader: &scannerErrorReader{err: readErr}}

	err := processLegacySSE(body, func(string) error {
		t.Fatalf("callback should not be called")
		return nil
	})
	if err == nil {
		t.Fatalf("expected scanner error")
	}
	if !errors.Is(err.Error, readErr) {
		t.Fatalf("expected read error, got %v", err.Error)
	}
}

func TestProcessLegacySSEAllowsLargeEvents(t *testing.T) {
	payload := strings.Repeat("x", 128*1024)
	body := &scannerTestReadCloser{
		reader: strings.NewReader("data: " + payload + "\n"),
	}

	var got string
	err := processLegacySSE(body, func(chunk string) error {
		got = chunk
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected scanner error: %v", err.Error)
	}
	if got != payload {
		t.Fatalf("expected large payload to round-trip, got length %d", len(got))
	}
}

func TestProcessFullSSEJoinsMultilineData(t *testing.T) {
	body := &scannerTestReadCloser{
		reader: strings.NewReader("event: message\ndata: {\"a\":\ndata: 1}\n\n"),
	}

	var got string
	err := processFullSSE(body, func(event string) error {
		got = event
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected scanner error: %v", err.Error)
	}

	expected := "event: message\ndata: {\"a\":\ndata: 1}"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}
