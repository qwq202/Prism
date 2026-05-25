package main

import (
	"os"
	"strings"
	"testing"
)

func TestDockerImageProvidesPrismCLI(t *testing.T) {
	data, err := os.ReadFile("Dockerfile")
	if err != nil {
		t.Fatalf("read Dockerfile: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "ln -sf /chat /usr/local/bin/prism") {
		t.Fatalf("expected Docker image to expose prism CLI alias for documented maintenance commands")
	}
	if !strings.Contains(content, "ln -sf /chat /usr/local/bin/chat") {
		t.Fatalf("expected Docker image to expose chat CLI alias for binary-name maintenance commands")
	}
}
