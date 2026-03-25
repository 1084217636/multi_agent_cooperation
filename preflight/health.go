package preflight

import (
	"context"
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
	Checks      []CheckResult `json:"checks"`
}

// Run 对当前工程进行轻量预检。
func Run(ctx context.Context, workdir string) Report {
	report := Report{GeneratedAt: time.Now()}

	report.Checks = append(report.Checks, runCommand(ctx, workdir, "go test", "go", "test", "./..."))
	report.Checks = append(report.Checks, runCommand(ctx, workdir, "go vet", "go", "vet", "./..."))
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
		return CheckResult{
			Name:    name,
			Status:  CheckSkipped,
			Summary: "工具未安装，已跳过",
		}
	}

	return CheckResult{
		Name:    name,
		Status:  CheckPassed,
		Summary: "工具可用: " + path,
	}
}

func trimOutput(output string) string {
	const limit = 4000
	output = strings.TrimSpace(output)
	if len(output) <= limit {
		return output
	}
	return output[:limit] + "\n...output truncated..."
}
