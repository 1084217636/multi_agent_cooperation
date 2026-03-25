package snapshot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureAndCompare(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "main.go")
	second := filepath.Join(root, "README.md")

	if err := os.WriteFile(first, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := Capture(root, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(first, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("# doc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	after, err := Capture(root, nil)
	if err != nil {
		t.Fatal(err)
	}

	diff := Compare(before, after)
	if len(diff.Modified) != 1 || diff.Modified[0] != "main.go" {
		t.Fatalf("unexpected modified diff: %+v", diff.Modified)
	}
	if len(diff.Added) != 1 || diff.Added[0] != "README.md" {
		t.Fatalf("unexpected added diff: %+v", diff.Added)
	}
}

func TestRestore(t *testing.T) {
	root := t.TempDir()
	mainFile := filepath.Join(root, "main.go")
	readmeFile := filepath.Join(root, "README.md")

	if err := os.WriteFile(mainFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	before, err := Capture(root, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(mainFile, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readmeFile, []byte("# changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	current, err := Capture(root, nil)
	if err != nil {
		t.Fatal(err)
	}

	diff := Compare(before, current)
	if err := Restore(root, before, diff); err != nil {
		t.Fatal(err)
	}

	restoredMain, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(restoredMain) != "package main\n" {
		t.Fatalf("unexpected restored main.go: %q", string(restoredMain))
	}

	if _, err := os.Stat(readmeFile); !os.IsNotExist(err) {
		t.Fatalf("expected README.md to be removed, got err=%v", err)
	}
}
