package preflight

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CheckStatus 描述检查结果状态。
type CheckStatus string

const (
	CheckPassed  CheckStatus = "passed"
	CheckFailed  CheckStatus = "failed"
	CheckSkipped CheckStatus = "skipped"
)

// CheckResult 描述单项工程检查的结果。
type CheckResult struct {
	Name    string      `json:"name"`
	Status  CheckStatus `json:"status"`
	Summary string      `json:"summary"`
	Output  string      `json:"output,omitempty"`
}

// Report 汇总本地工程预检结果。
type Report struct {
	GeneratedAt time.Time     `json:"generated_at"`
	Scope       string        `json:"scope,omitempty"`
	Targets     []string      `json:"targets,omitempty"`
	Checks      []CheckResult `json:"checks"`
}

// Run 对当前工程进行轻量预检。
func Run(ctx context.Context, workdir string) Report {
	return RunWithTargets(ctx, workdir, nil)
}

// RunWithTargets 对指定包范围执行预检；targets 为空时回退到 ./...
func RunWithTargets(ctx context.Context, workdir string, targets []string) Report {
	normalized := normalizeTargets(targets)
	report := Report{
		GeneratedAt: time.Now(),
		Scope:       scopeLabel(normalized),
		Targets:     normalized,
	}

	report.Checks = append(report.Checks, annotateScope(runCommand(ctx, workdir, "go test", append([]string{"go", "test"}, normalized...)...), report.Scope))
	report.Checks = append(report.Checks, annotateScope(runCommand(ctx, workdir, "go vet", append([]string{"go", "vet"}, normalized...)...), report.Scope))
	report.Checks = append(report.Checks, checkTool("goimports"))
	report.Checks = append(report.Checks, checkTool("golangci-lint"))

	return report
}

func runCommand(parent context.Context, workdir, name string, args ...string) CheckResult {
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = workdir
	cmd.Env = append(cmd.Environ(), "GOTOOLCHAIN=local")
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return CheckResult{
			Name:    name,
			Status:  CheckFailed,
			Summary: "执行超时",
			Output:  trimOutput(string(output)),
		}
	}

	if err != nil {
		return CheckResult{
			Name:    name,
			Status:  CheckFailed,
			Summary: err.Error(),
			Output:  trimOutput(string(output)),
		}
	}

	return CheckResult{
		Name:    name,
		Status:  CheckPassed,
		Summary: "检查通过",
		Output:  trimOutput(string(output)),
	}
}

func checkTool(name string) CheckResult {
	path, err := exec.LookPath(name)
	if err != nil {
		summary := "可选工具未安装，已跳过增强纠偏"
		output := ""
		switch name {
		case "goimports":
			output = "未安装 goimports；当前仍会执行 gofmt、go test、go vet。安装后可自动整理 import。\n示例：go install golang.org/x/tools/cmd/goimports@latest"
		case "golangci-lint":
			output = "未安装 golangci-lint；当前仍会执行 go test、go vet。安装后可补充更严格的静态检查。\n示例：go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
		}
		return CheckResult{
			Name:    name,
			Status:  CheckSkipped,
			Summary: summary,
			Output:  output,
		}
	}

	return CheckResult{
		Name:    name,
		Status:  CheckPassed,
		Summary: "工具可用: " + path,
	}
}

func normalizeTargets(targets []string) []string {
	if len(targets) == 0 {
		return []string{"./..."}
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(targets))
	for _, target := range targets {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		normalized = append(normalized, target)
	}
	if len(normalized) == 0 {
		return []string{"./..."}
	}
	return normalized
}

func scopeLabel(targets []string) string {
	switch len(targets) {
	case 0:
		return "./..."
	case 1:
		return targets[0]
	case 2:
		return targets[0] + ", " + targets[1]
	default:
		return targets[0] + ", " + targets[1] + fmt.Sprintf(" ... (%d targets)", len(targets))
	}
}

func annotateScope(result CheckResult, scope string) CheckResult {
	scope = strings.TrimSpace(scope)
	if scope == "" || scope == "./..." || result.Status == CheckSkipped {
		return result
	}
	if result.Summary == "" {
		result.Summary = "scope: " + scope
		return result
	}
	result.Summary = result.Summary + " [" + scope + "]"
	return result
}

func trimOutput(output string) string {
	const limit = 4000
	output = strings.TrimSpace(output)
	if len(output) <= limit {
		return output
	}
	return output[:limit] + "\n...output truncated..."
}
