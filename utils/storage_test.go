package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateStoredAttachmentNameRejectsPathTraversal(t *testing.T) {
	valid := "0123456789abcdef0123456789abcdef.png"
	if got, err := validateStoredAttachmentName(valid); err != nil || got != valid {
		t.Fatalf("expected valid attachment name, got name=%q err=%v", got, err)
	}

	invalidNames := []string{
		"",
		"../config/config.yaml",
		"0123456789abcdef0123456789abcdef/evil.png",
		"0123456789abcdef0123456789abcdef",
		"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz.png",
	}
	for _, name := range invalidNames {
		if _, err := validateStoredAttachmentName(name); err == nil {
			t.Fatalf("expected %q to be rejected", name)
		}
	}
}

func TestDeleteConfiguredStoredAttachmentRejectsPathTraversal(t *testing.T) {
	previousWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	workingDir := t.TempDir()
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWorkingDir)
	})

	target := filepath.Join("config", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(target, []byte("secret: keep-me"), 0o644); err != nil {
		t.Fatalf("write protected file: %v", err)
	}

	if err := DeleteConfiguredStoredAttachment("../../config/config.yaml"); err == nil {
		t.Fatalf("expected traversal delete to be rejected")
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected protected file to remain: %v", err)
	}
}

func TestListConfiguredStoredAttachmentsSkipsInvalidNames(t *testing.T) {
	previousWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	workingDir := t.TempDir()
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWorkingDir)
	})

	valid := "0123456789abcdef0123456789abcdef.png"
	if err := os.MkdirAll(filepath.Dir(AttachmentLocalPath(valid)), 0o755); err != nil {
		t.Fatalf("create attachment dir: %v", err)
	}
	if err := os.WriteFile(AttachmentLocalPath(valid), []byte("png"), 0o644); err != nil {
		t.Fatalf("write valid attachment: %v", err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(AttachmentLocalPath(valid)), "not-an-attachment"), []byte("bad"), 0o644); err != nil {
		t.Fatalf("write invalid attachment: %v", err)
	}

	attachments, err := ListConfiguredStoredAttachments()
	if err != nil {
		t.Fatalf("list attachments: %v", err)
	}
	if len(attachments) != 1 || attachments[0].Name != valid {
		t.Fatalf("expected only valid attachment, got %#v", attachments)
	}
}

func TestValidateStoragePublicBaseURL(t *testing.T) {
	valid := []string{
		"",
		"https://cdn.example.com",
		"http://localhost:8094/files",
	}
	for _, value := range valid {
		if err := ValidateStoragePublicBaseURL(value); err != nil {
			t.Fatalf("expected %q to be valid, got %v", value, err)
		}
	}

	invalid := []string{
		"cdn.example.com",
		"ftp://cdn.example.com",
		"https://example.r2.cloudflarestorage.com",
	}
	for _, value := range invalid {
		if err := ValidateStoragePublicBaseURL(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestStorageConnectionAllowsLocalWithoutPublicURL(t *testing.T) {
	previousWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	workingDir := t.TempDir()
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWorkingDir)
	})

	if err := TestStorageConnection(StorageTestConfig{Mode: "local"}); err != nil {
		t.Fatalf("expected local storage test without public url to pass, got %v", err)
	}
}

func TestStorageConnectionRejectsBarePublicBaseURL(t *testing.T) {
	err := TestStorageConnection(StorageTestConfig{
		Mode: "s3",
		S3: StorageS3Config{
			PublicBaseURL: "cdn.example.com",
		},
	})
	if err == nil {
		t.Fatalf("expected bare public base url to be rejected")
	}
}
