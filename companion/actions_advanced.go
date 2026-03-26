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

func (s *Service) performDockerSelfHealAction(workdir string, targets []string) actionOutcome {
	started := time.Now()
	if strings.TrimSpace(workdir) == "" {
		workdir = s.root
	}
	output, err := s.runDockerSelfHeal(workdir, targets)
	status := "completed"
	summary := "Docker 自愈执行闭环完成"
	if err != nil {
		status = "failed"
		summary = "Docker 自愈执行闭环失败"
		output = trimActionOutput(output + "\n" + err.Error())
	}

	report := preflight.RunWithTargets(context.Background(), workdir, targets)
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

func (s *Service) performAutoFixCommitAction(ctx context.Context, workdir string, targets []string) actionOutcome {
	started := time.Now()
	if strings.TrimSpace(workdir) == "" {
		workdir = s.root
	}
	output, err := s.runLocalAutoFix(ctx, workdir, targets)
	status := "completed"
	summary := "自动修复与自检完成"
	if err != nil {
		status = "failed"
		summary = "自动修复失败"
	}

	report := preflight.RunWithTargets(context.Background(), workdir, targets)
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

func (s *Service) runDockerSelfHeal(workdir string, targets []string) (string, error) {
	scope := strings.Join(validationArgs(targets), " ")
	script := `
set -u
export PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
export GOTOOLCHAIN=local
echo "[self-heal] initial go test"
if /usr/local/go/bin/go test ` + scope + `; then
  echo "[self-heal] initial check passed"
else
  echo "[self-heal] running go mod tidy"
  /usr/local/go/bin/go mod tidy || true
  echo "[self-heal] running gofmt"
  find . -path './.git' -prune -o -path './data' -prune -o -path './node_modules' -prune -o -name '*.go' -exec /usr/local/go/bin/gofmt -w {} +
  if command -v goimports >/dev/null 2>&1; then
    echo "[self-heal] running goimports"
    find . -path './.git' -prune -o -path './data' -prune -o -path './node_modules' -prune -o -name '*.go' -exec goimports -w {} +
  fi
  echo "[self-heal] rerunning go test"
  /usr/local/go/bin/go test ` + scope + ` || exit 1
fi
echo "[self-heal] go vet"
/usr/local/go/bin/go vet ` + scope + `
`

	cmd := exec.Command("docker", "run", "--rm",
		"-v", workdir+":/workspace",
		"-w", "/workspace",
		"golang:1.25",
		"sh", "-lc", script,
	)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (s *Service) runLocalAutoFix(ctx context.Context, workdir string, targets []string) (string, error) {
	var outputParts []string
	targetArgs := validationArgs(targets)

	goFiles, err := collectGoFiles(workdir)
	if err != nil {
		return "", err
	}
	if len(goFiles) > 0 {
		gofmtPath, pathErr := resolveGofmtPath(ctx)
		if pathErr != nil {
			return "", pathErr
		}
		cmd := exec.CommandContext(ctx, gofmtPath, append([]string{"-w"}, goFiles...)...)
		cmd.Dir = workdir
		formatted, fmtErr := cmd.CombinedOutput()
		outputParts = append(outputParts, "[gofmt]\n"+string(formatted))
		if fmtErr != nil {
			return strings.Join(outputParts, "\n"), fmtErr
		}

		if goimportsPath, lookErr := exec.LookPath("goimports"); lookErr == nil {
			goimportsCmd := exec.CommandContext(ctx, goimportsPath, append([]string{"-w"}, goFiles...)...)
			goimportsCmd.Dir = workdir
			goimportsOut, goimportsErr := goimportsCmd.CombinedOutput()
			outputParts = append(outputParts, "[goimports]\n"+string(goimportsOut))
			if goimportsErr != nil {
				return strings.Join(outputParts, "\n"), goimportsErr
			}
		} else {
			outputParts = append(outputParts, "[goimports]\nnot installed, skipped")
		}
	}

	tidyCmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	tidyCmd.Dir = workdir
	tidyCmd.Env = append(tidyCmd.Environ(), "GOTOOLCHAIN=local")
	tidyOut, tidyErr := tidyCmd.CombinedOutput()
	outputParts = append(outputParts, "[go mod tidy]\n"+string(tidyOut))
	if tidyErr != nil {
		return strings.Join(outputParts, "\n"), tidyErr
	}

	testCmd := exec.CommandContext(ctx, "go", append([]string{"test"}, targetArgs...)...)
	testCmd.Dir = workdir
	testCmd.Env = append(testCmd.Environ(), "GOTOOLCHAIN=local")
	testOut, testErr := testCmd.CombinedOutput()
	outputParts = append(outputParts, "[go test]\n"+string(testOut))
	if testErr != nil {
		return strings.Join(outputParts, "\n"), testErr
	}

	vetCmd := exec.CommandContext(ctx, "go", append([]string{"vet"}, targetArgs...)...)
	vetCmd.Dir = workdir
	vetCmd.Env = append(vetCmd.Environ(), "GOTOOLCHAIN=local")
	vetOut, vetErr := vetCmd.CombinedOutput()
	outputParts = append(outputParts, "[go vet]\n"+string(vetOut))
	if vetErr != nil {
		return strings.Join(outputParts, "\n"), vetErr
	}

	if golangciLintPath, lookErr := exec.LookPath("golangci-lint"); lookErr == nil {
		lintCmd := exec.CommandContext(ctx, golangciLintPath, append([]string{"run"}, targetArgs...)...)
		lintCmd.Dir = workdir
		lintOut, lintErr := lintCmd.CombinedOutput()
		outputParts = append(outputParts, "[golangci-lint]\n"+string(lintOut))
		if lintErr != nil {
			return strings.Join(outputParts, "\n"), lintErr
		}
	} else {
		outputParts = append(outputParts, "[golangci-lint]\nnot installed, skipped")
	}

	if isGitRepository(workdir) {
		statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
		statusCmd.Dir = workdir
		statusOut, statusErr := statusCmd.CombinedOutput()
		if statusErr == nil && strings.TrimSpace(string(statusOut)) != "" {
			addCmd := exec.CommandContext(ctx, "git", "add", "-A")
			addCmd.Dir = workdir
			addOut, addErr := addCmd.CombinedOutput()
			outputParts = append(outputParts, "[git add]\n"+string(addOut))
			if addErr != nil {
				return strings.Join(outputParts, "\n"), addErr
			}

			commitCmd := exec.CommandContext(ctx, "git", "commit", "-m", "chore: auto-fix workspace")
			commitCmd.Dir = workdir
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
			if name == ".git" || name == "data" || name == "node_modules" || name == "bin" || name == "workspace_runs" {
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
