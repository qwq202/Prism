package utils

import (
	"chat/globals"
	"os"
	"path/filepath"
	"strings"
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

func withStorageGlobals(t *testing.T) {
	t.Helper()

	previousNotifyURL := globals.NotifyUrl
	previousMode := globals.StorageMode
	previousS3Endpoint := globals.StorageS3Endpoint
	previousS3Region := globals.StorageS3Region
	previousS3Bucket := globals.StorageS3Bucket
	previousS3AccessKey := globals.StorageS3AccessKey
	previousS3SecretKey := globals.StorageS3SecretKey
	previousS3PublicBaseURL := globals.StorageS3PublicBaseURL
	previousS3ForcePathStyle := globals.StorageS3ForcePathStyle
	previousR2AccountID := globals.StorageR2AccountID
	previousR2Jurisdiction := globals.StorageR2Jurisdiction
	previousR2Bucket := globals.StorageR2Bucket
	previousR2AccessKey := globals.StorageR2AccessKey
	previousR2SecretKey := globals.StorageR2SecretKey
	previousR2PublicBaseURL := globals.StorageR2PublicBaseURL

	t.Cleanup(func() {
		globals.NotifyUrl = previousNotifyURL
		globals.StorageMode = previousMode
		globals.StorageS3Endpoint = previousS3Endpoint
		globals.StorageS3Region = previousS3Region
		globals.StorageS3Bucket = previousS3Bucket
		globals.StorageS3AccessKey = previousS3AccessKey
		globals.StorageS3SecretKey = previousS3SecretKey
		globals.StorageS3PublicBaseURL = previousS3PublicBaseURL
		globals.StorageS3ForcePathStyle = previousS3ForcePathStyle
		globals.StorageR2AccountID = previousR2AccountID
		globals.StorageR2Jurisdiction = previousR2Jurisdiction
		globals.StorageR2Bucket = previousR2Bucket
		globals.StorageR2AccessKey = previousR2AccessKey
		globals.StorageR2SecretKey = previousR2SecretKey
		globals.StorageR2PublicBaseURL = previousR2PublicBaseURL
	})

	globals.NotifyUrl = ""
	globals.StorageMode = "local"
	globals.StorageS3Endpoint = ""
	globals.StorageS3Region = ""
	globals.StorageS3Bucket = ""
	globals.StorageS3AccessKey = ""
	globals.StorageS3SecretKey = ""
	globals.StorageS3PublicBaseURL = ""
	globals.StorageS3ForcePathStyle = false
	globals.StorageR2AccountID = ""
	globals.StorageR2Jurisdiction = ""
	globals.StorageR2Bucket = ""
	globals.StorageR2AccessKey = ""
	globals.StorageR2SecretKey = ""
	globals.StorageR2PublicBaseURL = ""
}

func TestAttachmentPublicURLIgnoresIncompleteRemotePublicBaseURL(t *testing.T) {
	withStorageGlobals(t)

	globals.NotifyUrl = ""
	globals.StorageMode = "s3"
	globals.StorageS3PublicBaseURL = "https://cdn.example.com"

	name := "0123456789abcdef0123456789abcdef.png"
	if got := AttachmentPublicURL(name); got != "/attachments/"+name {
		t.Fatalf("expected local attachment url for incomplete remote storage, got %q", got)
	}
}

func TestStoreAttachmentDataRejectsIncompleteRemoteStorage(t *testing.T) {
	withStorageGlobals(t)

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

	data := []byte("png")
	name := attachmentNameForUpload("image.png", data, "image/png")
	globals.StorageMode = "s3"
	globals.StorageS3PublicBaseURL = "https://cdn.example.com"

	if _, err := StoreAttachmentData("image.png", data, "image/png"); err == nil ||
		!strings.Contains(err.Error(), "s3 storage is not fully configured") {
		t.Fatalf("expected incomplete s3 storage error, got %v", err)
	}
	if _, err := os.Stat(AttachmentLocalPath(name)); !os.IsNotExist(err) {
		t.Fatalf("expected incomplete remote upload not to write a local fallback, got %v", err)
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

func TestStorageConnectionRejectsR2WithoutAccountID(t *testing.T) {
	err := TestStorageConnection(StorageTestConfig{
		Mode: "r2",
		R2: StorageR2Config{
			Bucket:        "bucket",
			AccessKey:     "access",
			SecretKey:     "secret",
			PublicBaseURL: "https://files.example.com",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "r2 storage is not fully configured") {
		t.Fatalf("expected incomplete r2 storage error, got %v", err)
	}
}
