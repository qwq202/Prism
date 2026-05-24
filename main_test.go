package main

import "testing"

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
