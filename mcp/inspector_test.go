package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectorCollectsImportsAndCalls(t *testing.T) {
	root := t.TempDir()
	source := `package demo

import (
	"fmt"
	"strings"
)

type Example struct {
	Name string
}

func Build(input string) string {
	value := strings.TrimSpace(input)
	fmt.Println(value)
	return value
}
`

	path := filepath.Join(root, "demo.go")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	inspector := NewInspector(root)
	info, err := inspector.ScanProject()
	if err != nil {
		t.Fatalf("scan project: %v", err)
	}

	if len(info.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(info.Imports))
	}
	if len(info.Calls) < 2 {
		t.Fatalf("expected at least 2 call edges, got %d", len(info.Calls))
	}

	var foundTrim, foundPrintln bool
	for _, edge := range info.Calls {
		if strings.Contains(edge.Callee, "TrimSpace") {
			foundTrim = true
		}
		if strings.Contains(edge.Callee, "Println") {
			foundPrintln = true
		}
	}

	if !foundTrim || !foundPrintln {
		t.Fatalf("missing expected call edges: %#v", info.Calls)
	}
}
