package admin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempLogWorkdir(t *testing.T) string {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change cwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	if err := os.Mkdir(logDirectory, 0o755); err != nil {
		t.Fatalf("create logs dir: %v", err)
	}

	return dir
}

func TestResolveLogPathRejectsTraversal(t *testing.T) {
	withTempLogWorkdir(t)

	tests := []string{
		"",
		".",
		"..",
		"../etc/passwd",
		filepath.Join("nested", "..", "..", "etc", "passwd"),
		string(filepath.Separator) + filepath.Join("etc", "passwd"),
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			if got, err := resolveLogPath(path); err == nil {
				t.Fatalf("expected %q to be rejected, got %q", path, got)
			}
		})
	}
}

func TestResolveLogPathAllowsLogDirectoryFiles(t *testing.T) {
	withTempLogWorkdir(t)

	got, err := resolveLogPath(filepath.Join("archive", "..", "chatnio.log"))
	if err != nil {
		t.Fatalf("expected log path to resolve: %v", err)
	}

	root, err := filepath.Abs(logDirectory)
	if err != nil {
		t.Fatalf("resolve log root: %v", err)
	}
	wantPrefix := root + string(filepath.Separator)
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("expected resolved path to stay in logs dir, got %q", got)
	}
	if filepath.Base(got) != "chatnio.log" {
		t.Fatalf("expected resolved log file basename, got %q", got)
	}
}

func TestResolveLogPathRejectsSymlinkEscape(t *testing.T) {
	dir := withTempLogWorkdir(t)

	outside := filepath.Join(dir, "outside.log")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	link := filepath.Join(logDirectory, "linked.log")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	if got, err := resolveLogPath("linked.log"); err == nil {
		t.Fatalf("expected symlink escape to be rejected, got %q", got)
	}
}

func TestDeleteLogFileRejectsTraversal(t *testing.T) {
	dir := withTempLogWorkdir(t)

	outside := filepath.Join(dir, "outside.log")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	if err := deleteLogFile("../outside.log"); err == nil {
		t.Fatalf("expected traversal delete to be rejected")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("expected outside file to remain: %v", err)
	}
}
