package connection

import (
	"os"
	"strings"
	"testing"
)

func TestDebugRedisCleanupDoesNotFlushAllDatabases(t *testing.T) {
	data, err := os.ReadFile("cache.go")
	if err != nil {
		t.Fatalf("read cache.go: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "FlushAll(") {
		t.Fatalf("debug Redis cleanup must not flush every Redis database")
	}
	if !strings.Contains(content, "FlushDB(") {
		t.Fatalf("expected debug Redis cleanup to stay scoped to the configured Redis database")
	}
}
