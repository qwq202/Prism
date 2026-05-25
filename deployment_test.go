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

func TestDockerImagePersistsSQLiteDirectory(t *testing.T) {
	dockerfile, err := os.ReadFile("Dockerfile")
	if err != nil {
		t.Fatalf("read Dockerfile: %v", err)
	}
	if !strings.Contains(string(dockerfile), `VOLUME ["/config", "/logs", "/storage", "/db"]`) {
		t.Fatalf("expected Docker image to declare /db as a volume for SQLite deployments")
	}

	entrypoint, err := os.ReadFile("docker-entrypoint.sh")
	if err != nil {
		t.Fatalf("read docker-entrypoint.sh: %v", err)
	}
	if !strings.Contains(string(entrypoint), "/config /logs /storage /db") {
		t.Fatalf("expected entrypoint to repair /db permissions when mounted by users")
	}
}

func TestDockerEnvExampleDocumentsComposeVariables(t *testing.T) {
	data, err := os.ReadFile(".env.example")
	if err != nil {
		t.Fatalf("read .env.example: %v", err)
	}

	content := string(data)
	for _, name := range []string{
		"MYSQL_ROOT_PASSWORD",
		"MYSQL_DATABASE",
		"MYSQL_USER",
		"MYSQL_PASSWORD",
		"PRISM_IMAGE_TAG",
		"SECRET",
		"ROOT_INITIAL_PASSWORD",
	} {
		if !strings.Contains(content, name+"=") {
			t.Fatalf("expected .env.example to document %s", name)
		}
	}
}
