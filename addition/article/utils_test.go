package article

import (
	"strings"
	"testing"
)

func TestArticleDocxPathSanitizesTitle(t *testing.T) {
	got, err := articleDocxPath("0123456789abcdef0123456789abcdef", "../../Draft/One:Final")
	if err != nil {
		t.Fatalf("expected sanitized article path: %v", err)
	}
	if !strings.HasPrefix(got, "storage/article/data/0123456789abcdef0123456789abcdef/") {
		t.Fatalf("expected path to remain in article data directory, got %q", got)
	}
	if strings.Contains(got, "../") || strings.Contains(got, `\`) {
		t.Fatalf("expected unsafe path fragments to be removed, got %q", got)
	}
	if !strings.HasSuffix(got, ".docx") {
		t.Fatalf("expected docx filename, got %q", got)
	}
}

func TestArticleDocxPathRejectsUnsafeHash(t *testing.T) {
	if _, err := articleDocxPath("../../config", "draft"); err == nil {
		t.Fatalf("expected unsafe article hash to be rejected")
	}
}
