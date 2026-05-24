package generation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateProjectRejectsPathEscape(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")

	ok := GenerateProject(project, ProjectResult{
		Result: map[string]interface{}{
			"../escape.txt": "bad",
		},
	})

	if ok {
		t.Fatalf("expected unsafe project path to be rejected")
	}
	if _, err := os.Stat(filepath.Join(root, "escape.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected escaped file not to be written, stat err: %v", err)
	}
}

func TestGenerateProjectAllowsNestedFiles(t *testing.T) {
	project := filepath.Join(t.TempDir(), "project")

	ok := GenerateProject(project, ProjectResult{
		Result: map[string]interface{}{
			"src": map[string]interface{}{
				"main.go": "package main\n",
			},
		},
	})

	if !ok {
		t.Fatalf("expected nested project file to be generated")
	}
	if data, err := os.ReadFile(filepath.Join(project, "src", "main.go")); err != nil || string(data) != "package main\n" {
		t.Fatalf("expected nested file content, got %q, err %v", data, err)
	}
}

func TestGenerateProjectRejectsUnsupportedNode(t *testing.T) {
	ok := GenerateProject(filepath.Join(t.TempDir(), "project"), ProjectResult{
		Result: map[string]interface{}{
			"README.md": []interface{}{"bad"},
		},
	})

	if ok {
		t.Fatalf("expected unsupported generated project node to be rejected")
	}
}
