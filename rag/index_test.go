package rag

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildAndSearch(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "# RAG Guide\n\n桌面 AI 开发伴侣需要把 README、代码和设计文档一起做轻量检索。"
	if err := os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	index, err := Build(root, Options{
		IncludePaths: []string{docsDir},
		ChunkSize:    120,
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	results := index.Search("桌面 AI RAG 检索", 3)
	if len(results) == 0 {
		t.Fatal("expected at least one search result")
	}
	if results[0].Path != filepath.ToSlash("docs/guide.md") && results[0].Path != "docs\\guide.md" {
		t.Fatalf("unexpected path: %s", results[0].Path)
	}
}
