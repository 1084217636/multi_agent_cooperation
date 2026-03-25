package companion

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"multi_agent_cooperation/preflight"
)

func (s *Service) performDockerSelfHealAction() actionOutcome {
	started := time.Now()
	output, err := s.runDockerSelfHeal()
	status := "completed"
	summary := "Docker 自愈执行闭环完成"
	if err != nil {
		status = "failed"
		summary = "Docker 自愈执行闭环失败"
		output = trimActionOutput(output + "\n" + err.Error())
	}

	report := preflight.Run(context.Background(), s.root)
	if hasPreflightFailure(report) {
		status = "failed"
		if err == nil {
			summary = "Docker 自愈后预检仍存在失败项"
		}
	}

	return actionOutcome{
		Status:    status,
		Summary:   summary,
		Output:    trimActionOutput(output),
		Preflight: &report,
		Step: &ExecutionStep{
			Name:        "approval-docker-self-heal",
			Status:      status,
			Summary:     summary,
			StartedAt:   started,
			CompletedAt: time.Now(),
		},
	}
}

func (s *Service) performAutoFixCommitAction(ctx context.Context) actionOutcome {
	started := time.Now()
	output, err := s.runLocalAutoFix(ctx)
	status := "completed"
	summary := "自动修复与自检完成"
	if err != nil {
		status = "failed"
		summary = "自动修复失败"
	}

	report := preflight.Run(context.Background(), s.root)
	if hasPreflightFailure(report) {
		status = "failed"
		if err == nil {
			summary = "自动修复后预检仍存在失败项"
		}
	}

	return actionOutcome{
		Status:    status,
		Summary:   summary,
		Output:    trimActionOutput(output),
		Preflight: &report,
		Step: &ExecutionStep{
			Name:        "approval-autofix-commit",
			Status:      status,
			Summary:     summary,
			StartedAt:   started,
			CompletedAt: time.Now(),
		},
	}
}

func (s *Service) runDockerSelfHeal() (string, error) {
	script := `
set -u
export GOTOOLCHAIN=local
echo "[self-heal] initial go test"
if go test ./...; then
  echo "[self-heal] initial check passed"
else
  echo "[self-heal] running go mod tidy"
  go mod tidy || true
  echo "[self-heal] running gofmt"
  find . -path './.git' -prune -o -path './data' -prune -o -path './node_modules' -prune -o -name '*.go' -exec gofmt -w {} +
  echo "[self-heal] rerunning go test"
  go test ./... || exit 1
fi
echo "[self-heal] go vet"
go vet ./...
`

	cmd := exec.Command("docker", "run", "--rm",
		"-v", s.root+":/workspace",
		"-w", "/workspace",
		"golang:1.25",
		"sh", "-lc", script,
	)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (s *Service) runLocalAutoFix(ctx context.Context) (string, error) {
	var outputParts []string

	goFiles, err := collectGoFiles(s.root)
	if err != nil {
		return "", err
	}
	if len(goFiles) > 0 {
		gofmtPath, pathErr := resolveGofmtPath(ctx)
		if pathErr != nil {
			return "", pathErr
		}
		cmd := exec.CommandContext(ctx, gofmtPath, append([]string{"-w"}, goFiles...)...)
		cmd.Dir = s.root
		formatted, fmtErr := cmd.CombinedOutput()
		outputParts = append(outputParts, "[gofmt]\n"+string(formatted))
		if fmtErr != nil {
			return strings.Join(outputParts, "\n"), fmtErr
		}
	}

	tidyCmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	tidyCmd.Dir = s.root
	tidyCmd.Env = append(tidyCmd.Environ(), "GOTOOLCHAIN=local")
	tidyOut, tidyErr := tidyCmd.CombinedOutput()
	outputParts = append(outputParts, "[go mod tidy]\n"+string(tidyOut))
	if tidyErr != nil {
		return strings.Join(outputParts, "\n"), tidyErr
	}

	if isGitRepository(s.root) {
		statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
		statusCmd.Dir = s.root
		statusOut, statusErr := statusCmd.CombinedOutput()
		if statusErr == nil && strings.TrimSpace(string(statusOut)) != "" {
			addCmd := exec.CommandContext(ctx, "git", "add", "-A")
			addCmd.Dir = s.root
			addOut, addErr := addCmd.CombinedOutput()
			outputParts = append(outputParts, "[git add]\n"+string(addOut))
			if addErr != nil {
				return strings.Join(outputParts, "\n"), addErr
			}

			commitCmd := exec.CommandContext(ctx, "git", "commit", "-m", "chore: auto-fix workspace")
			commitCmd.Dir = s.root
			commitOut, commitErr := commitCmd.CombinedOutput()
			outputParts = append(outputParts, "[git commit]\n"+string(commitOut))
			if commitErr != nil {
				return strings.Join(outputParts, "\n"), commitErr
			}
		} else {
			outputParts = append(outputParts, "[git]\nno changes to commit")
		}
	} else {
		outputParts = append(outputParts, "[git]\ncurrent workspace is not a git repository, commit skipped")
	}

	return strings.Join(outputParts, "\n"), nil
}

func collectGoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "data" || name == "node_modules" || name == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func resolveGofmtPath(ctx context.Context) (string, error) {
	gofmtPath, err := exec.LookPath("gofmt")
	if err == nil {
		return gofmtPath, nil
	}

	cmd := exec.CommandContext(ctx, "go", "env", "GOROOT")
	cmd.Env = append(cmd.Environ(), "GOTOOLCHAIN=local")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return filepath.Join(strings.TrimSpace(string(output)), "bin", "gofmt"), nil
}
