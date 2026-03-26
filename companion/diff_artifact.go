package companion

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"multi_agent_cooperation/snapshot"
)

func writeSnapshotDiffArtifact(exportDir string, report *RunReport) (*GeneratedArtifact, error) {
	if report == nil || !report.Snapshot.Enabled || report.Snapshot.BeforePath == "" || report.Snapshot.AfterPath == "" {
		return nil, nil
	}
	total := len(report.Snapshot.Diff.Added) + len(report.Snapshot.Diff.Modified) + len(report.Snapshot.Diff.Deleted)
	if total == 0 {
		return nil, nil
	}

	before, err := snapshot.Load(report.Snapshot.BeforePath)
	if err != nil {
		return nil, err
	}
	after, err := snapshot.Load(report.Snapshot.AfterPath)
	if err != nil {
		return nil, err
	}

	beforeMap, err := manifestContentMap(before)
	if err != nil {
		return nil, err
	}
	afterMap, err := manifestContentMap(after)
	if err != nil {
		return nil, err
	}

	var builder strings.Builder
	builder.WriteString("# Workspace Diff\n\n")
	appendDiffGroup(&builder, report.Snapshot.Diff.Added, beforeMap, afterMap)
	appendDiffGroup(&builder, report.Snapshot.Diff.Modified, beforeMap, afterMap)
	appendDiffGroup(&builder, report.Snapshot.Diff.Deleted, beforeMap, afterMap)

	path := filepath.Join(exportDir, "workspace_diff.patch")
	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		return nil, err
	}

	return &GeneratedArtifact{
		Name:    filepath.Base(path),
		Kind:    "diff",
		Summary: "本次运行的工作区差异补丁，包含新增、修改、删除文件的 diff 内容。",
		Path:    path,
		URL:     "/exports/" + report.ID + "/" + filepath.Base(path),
	}, nil
}

func appendDiffGroup(builder *strings.Builder, paths []string, beforeMap, afterMap map[string]string) {
	for _, path := range paths {
		diffBody := renderUnifiedDiff(path, beforeMap[path], afterMap[path])
		if !strings.HasSuffix(diffBody, "\n") {
			diffBody += "\n"
		}
		builder.WriteString("## ")
		builder.WriteString(path)
		builder.WriteString("\n\n")
		builder.WriteString("```diff\n")
		builder.WriteString(diffBody)
		builder.WriteString("```\n\n")
	}
}

func manifestContentMap(manifest *snapshot.Manifest) (map[string]string, error) {
	result := make(map[string]string, len(manifest.Files))
	for _, file := range manifest.Files {
		data, err := base64.StdEncoding.DecodeString(file.Data)
		if err != nil {
			return nil, err
		}
		result[file.Path] = string(data)
	}
	return result, nil
}

func renderUnifiedDiff(path, before, after string) string {
	diffPath, err := exec.LookPath("diff")
	if err == nil {
		output, renderErr := externalDiff(diffPath, path, before, after)
		if renderErr == nil && strings.TrimSpace(output) != "" {
			return output
		}
	}
	return fallbackDiff(path, before, after)
}

func externalDiff(diffPath, path, before, after string) (string, error) {
	tempDir, err := os.MkdirTemp("", "companion-diff-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)

	beforePath := filepath.Join(tempDir, "before.txt")
	afterPath := filepath.Join(tempDir, "after.txt")
	if err := os.WriteFile(beforePath, []byte(before), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(afterPath, []byte(after), 0o644); err != nil {
		return "", err
	}

	cmd := exec.Command(diffPath, "-u", "--label", "a/"+path, beforePath, "--label", "b/"+path, afterPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return string(output), nil
		}
		return "", err
	}
	return string(output), nil
}

func fallbackDiff(path, before, after string) string {
	var builder bytes.Buffer
	builder.WriteString(fmt.Sprintf("--- a/%s\n", path))
	builder.WriteString(fmt.Sprintf("+++ b/%s\n", path))
	if before == after {
		builder.WriteString(" no textual diff captured\n")
		return builder.String()
	}
	if before == "" {
		for _, line := range strings.Split(after, "\n") {
			builder.WriteString("+")
			builder.WriteString(line)
			builder.WriteString("\n")
		}
		return builder.String()
	}
	if after == "" {
		for _, line := range strings.Split(before, "\n") {
			builder.WriteString("-")
			builder.WriteString(line)
			builder.WriteString("\n")
		}
		return builder.String()
	}

	builder.WriteString("@@ before @@\n")
	for _, line := range strings.Split(before, "\n") {
		builder.WriteString("-")
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	builder.WriteString("@@ after @@\n")
	for _, line := range strings.Split(after, "\n") {
		builder.WriteString("+")
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	return builder.String()
}
