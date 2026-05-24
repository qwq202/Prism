package main

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestNormalizeAllowedOriginHost(t *testing.T) {
	tests := map[string]string{
		"https://www.example.com/":      "example.com",
		"example.com":                   "example.com",
		"example.com/path":              "example.com",
		" http://localhost:5173/app/ ":  "localhost:5173",
		"https://api.example.com:8443/": "api.example.com:8443",
		"":                              "",
		"http://%zz":                    "",
	}

	for input, want := range tests {
		if got := normalizeAllowedOriginHost(input); got != want {
			t.Fatalf("normalizeAllowedOriginHost(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDependencyHealth(t *testing.T) {
	ok := dependencyHealth(context.Background(), func(context.Context) error {
		return nil
	})
	if ok.Status != "ok" {
		t.Fatalf("dependencyHealth success = %+v, want ok", ok)
	}

	failed := dependencyHealth(context.Background(), func(context.Context) error {
		return errors.New("boom")
	})
	if failed.Status != "unavailable" {
		t.Fatalf("dependencyHealth failure = %+v, want unavailable", failed)
	}
}

func TestHealthHTTPStatus(t *testing.T) {
	if got := healthHTTPStatus(healthDependency{Status: "ok"}, healthDependency{Status: "ok"}); got != http.StatusOK {
		t.Fatalf("healthHTTPStatus(ok dependencies) = %d, want %d", got, http.StatusOK)
	}

	if got := healthHTTPStatus(healthDependency{Status: "ok"}, healthDependency{Status: "unavailable"}); got != http.StatusServiceUnavailable {
		t.Fatalf("healthHTTPStatus(unavailable dependency) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}
