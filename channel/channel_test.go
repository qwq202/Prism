package channel

import (
	"errors"
	"strings"
	"testing"
)

func TestProcessErrorIgnoresBlankEndpoint(t *testing.T) {
	err := (&Channel{Id: 7}).ProcessError(errors.New("unknown channel type test"))
	if err == nil {
		t.Fatalf("expected processed error")
	}

	if got := err.Error(); got != "unknown channel type test" {
		t.Fatalf("expected blank endpoint to leave error intact, got %q", got)
	}
}

func TestProcessErrorRedactsEndpoint(t *testing.T) {
	err := (&Channel{
		Id:       7,
		Endpoint: "https://api.example.com/v1",
	}).ProcessError(errors.New("post https://api.example.com/v1/chat failed"))
	if err == nil {
		t.Fatalf("expected processed error")
	}

	if got := err.Error(); strings.Contains(got, "api.example.com") || !strings.Contains(got, "channel://7") {
		t.Fatalf("expected endpoint to be redacted, got %q", got)
	}
}
