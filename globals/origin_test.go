package globals

import "testing"

func TestOriginIsAllowedRejectsExternalOriginWhenAllowListEmpty(t *testing.T) {
	previous := AllowedOrigins
	AllowedOrigins = nil
	t.Cleanup(func() {
		AllowedOrigins = previous
	})

	if OriginIsAllowed("https://evil.example") {
		t.Fatalf("expected empty allow list to reject external origin")
	}
	if OriginIsAllowed("http://%zz") {
		t.Fatalf("expected malformed origin to be rejected")
	}
	if !OriginIsAllowed("http://localhost:5173") {
		t.Fatalf("expected localhost to remain allowed for local development")
	}
	if !OriginIsAllowed("http://127.0.0.1:1420") {
		t.Fatalf("expected 127.0.0.1 to remain allowed for local desktop development")
	}
	if !OriginIsAllowed("http://[::1]:1420") {
		t.Fatalf("expected ::1 to remain allowed for local desktop development")
	}
	if !OriginIsAllowed("file://local-app") {
		t.Fatalf("expected file origins to remain allowed")
	}
}

func TestOriginIsAllowedUsesConfiguredHosts(t *testing.T) {
	previous := AllowedOrigins
	AllowedOrigins = []string{"example.com"}
	t.Cleanup(func() {
		AllowedOrigins = previous
	})

	if !OriginIsAllowed("https://www.example.com") {
		t.Fatalf("expected configured host to be allowed")
	}
	if OriginIsAllowed("https://evil.example") {
		t.Fatalf("expected unconfigured host to be rejected")
	}
}
