package companion

import "testing"

func TestResolveValidationTargets(t *testing.T) {
	report := CodeGenerationReport{
		TargetMode: "isolated_workspace",
		Files: []GeneratedFileRecord{
			{Path: "companion/auto_loop.go"},
			{Path: "preflight/health.go"},
			{Path: "README.md"},
		},
	}

	targets := resolveValidationTargets("/repo", "/repo/workspace_runs/123", report)
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d (%v)", len(targets), targets)
	}
	if targets[0] != "./companion" || targets[1] != "./preflight" {
		t.Fatalf("unexpected validation targets: %v", targets)
	}
}

func TestRestrictCodeBundlePayloadForCurrentRepoPatch(t *testing.T) {
	payload := codeBundlePayload{
		Files: []codeBundleFileDef{
			{Path: "companion/auto_loop.go", Content: "package companion\n"},
			{Path: "companion/auto_loop_test.go", Content: "package companion\n"},
			{Path: "cmd/main.go", Content: "package main\n"},
		},
	}

	filtered, rejected := restrictCodeBundlePayload("current_repo_patch", []string{"companion/auto_loop.go"}, payload)
	if len(filtered.Files) != 2 {
		t.Fatalf("expected 2 allowed files, got %d", len(filtered.Files))
	}
	if len(rejected) != 1 || rejected[0] != "cmd/main.go" {
		t.Fatalf("unexpected rejected files: %v", rejected)
	}
}
