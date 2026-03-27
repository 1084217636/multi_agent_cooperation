package companion

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"multi_agent_cooperation/executor"
	"multi_agent_cooperation/preflight"
	"multi_agent_cooperation/snapshot"
)

// RepairRound 记录一轮自动修复的执行情况。
type RepairRound struct {
	Round             int       `json:"round"`
	Status            string    `json:"status"`
	Summary           string    `json:"summary"`
	Targets           []string  `json:"targets,omitempty"`
	Commands          []string  `json:"commands"`
	Output            string    `json:"output,omitempty"`
	ChangedFiles      int       `json:"changed_files"`
	TerminationReason string    `json:"termination_reason,omitempty"`
	StartedAt         time.Time `json:"started_at"`
	CompletedAt       time.Time `json:"completed_at"`
}

// SandboxValidation 描述一次 Docker 沙盒验证结果。
type SandboxValidation struct {
	Enabled     bool      `json:"enabled"`
	Status      string    `json:"status,omitempty"`
	Summary     string    `json:"summary,omitempty"`
	Targets     []string  `json:"targets,omitempty"`
	Command     string    `json:"command,omitempty"`
	Output      string    `json:"output,omitempty"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

func (s *Service) runAutoRepairLoop(ctx context.Context, workdir string, initial preflight.Report, targets []string) ([]RepairRound, preflight.Report) {
	maxRounds := s.config.Runtime.AutoRepairRounds
	if maxRounds <= 0 || !hasPreflightFailure(initial) {
		return nil, initial
	}
	if strings.TrimSpace(workdir) == "" {
		workdir = s.root
	}
	if len(targets) == 0 {
		targets = initial.Targets
	}

	current := initial
	previousFailure := failureSignature(initial)
	rounds := make([]RepairRound, 0, maxRounds)

	for roundNumber := 1; roundNumber <= maxRounds; roundNumber++ {
		started := time.Now()
		before, _ := snapshot.Capture(workdir, s.projectExcludeNames())

		commands, output := s.runRepairCommands(ctx, workdir, targets)
		current = preflight.RunWithTargets(ctx, workdir, targets)

		after, _ := snapshot.Capture(workdir, s.projectExcludeNames())
		diff := snapshot.Compare(before, after)
		changed := len(diff.Added) + len(diff.Modified) + len(diff.Deleted)

		status := "completed"
		summary := preflightSummary(current)
		termination := ""
		if hasPreflightFailure(current) {
			status = "failed"
			signature := failureSignature(current)
			switch {
			case changed == 0:
				termination = "自动修复命令未带来新的工作区变更，提前终止。"
			case signature == previousFailure:
				termination = "失败签名未变化，触发终止策略。"
			case roundNumber == maxRounds:
				termination = "已达到最大自动修复轮次。"
			}
			if termination != "" {
				summary = "自动修复后仍存在失败项"
			}
			previousFailure = signature
		}

		rounds = append(rounds, RepairRound{
			Round:             roundNumber,
			Status:            status,
			Summary:           summary,
			Targets:           append([]string{}, current.Targets...),
			Commands:          commands,
			Output:            trimActionOutput(output),
			ChangedFiles:      changed,
			TerminationReason: termination,
			StartedAt:         started,
			CompletedAt:       time.Now(),
		})

		if !hasPreflightFailure(current) || termination != "" {
			break
		}
	}

	return rounds, current
}

func (s *Service) runRepairCommands(ctx context.Context, workdir string, targets []string) ([]string, string) {
	if strings.TrimSpace(workdir) == "" {
		workdir = s.root
	}
	targetArgs := validationArgs(targets)
	targetLabel := validationScopeLabel(targetArgs)

	goFiles, err := collectGoFiles(workdir, s.projectExcludeNames())
	if err != nil {
		return nil, err.Error()
	}

	var commands []string
	var outputParts []string

	if len(goFiles) > 0 {
		gofmtPath, pathErr := resolveGofmtPath(ctx)
		if pathErr == nil {
			commands = append(commands, "gofmt -w <go files>")
			out, runErr := s.runCommandWithOutput(ctx, workdir, gofmtPath, append([]string{"-w"}, goFiles...)...)
			outputParts = append(outputParts, "[gofmt]\n"+out)
			if runErr != nil {
				return commands, strings.Join(outputParts, "\n")
			}
		} else {
			outputParts = append(outputParts, "[gofmt]\n"+pathErr.Error())
		}

		if goimportsPath, lookErr := exec.LookPath("goimports"); lookErr == nil {
			commands = append(commands, "goimports -w <go files>")
			out, runErr := s.runCommandWithOutput(ctx, workdir, goimportsPath, append([]string{"-w"}, goFiles...)...)
			outputParts = append(outputParts, "[goimports]\n"+out)
			if runErr != nil {
				return commands, strings.Join(outputParts, "\n")
			}
		} else {
			outputParts = append(outputParts, "[goimports]\nnot installed, skipped")
		}
	}

	commands = append(commands, "go mod tidy")
	out, tidyErr := s.runCommandWithOutput(ctx, workdir, "go", "mod", "tidy")
	outputParts = append(outputParts, "[go mod tidy]\n"+out)
	if tidyErr != nil {
		return commands, strings.Join(outputParts, "\n")
	}

	if golangciLintPath, lookErr := exec.LookPath("golangci-lint"); lookErr == nil {
		commands = append(commands, "golangci-lint run "+targetLabel)
		lintArgs := append([]string{"run"}, targetArgs...)
		out, lintErr := s.runCommandWithOutput(ctx, workdir, golangciLintPath, lintArgs...)
		outputParts = append(outputParts, "[golangci-lint]\n"+out)
		if lintErr != nil {
			return commands, strings.Join(outputParts, "\n")
		}
	} else {
		outputParts = append(outputParts, "[golangci-lint]\nnot installed, skipped")
	}

	commands = append(commands, "go test "+targetLabel)
	testArgs := append([]string{"test"}, targetArgs...)
	out, testErr := s.runCommandWithOutput(ctx, workdir, "go", testArgs...)
	outputParts = append(outputParts, "[go test]\n"+out)
	if testErr != nil {
		return commands, strings.Join(outputParts, "\n")
	}

	commands = append(commands, "go vet "+targetLabel)
	vetArgs := append([]string{"vet"}, targetArgs...)
	out, err = s.runCommandWithOutput(ctx, workdir, "go", vetArgs...)
	outputParts = append(outputParts, "[go vet]\n"+out)
	return commands, strings.Join(outputParts, "\n")
}

func (s *Service) runCommandWithOutput(ctx context.Context, workdir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workdir
	cmd.Env = append(cmd.Environ(), "GOTOOLCHAIN=local")
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func failureSignature(report preflight.Report) string {
	if len(report.Checks) == 0 {
		return "no-checks"
	}
	var parts []string
	for _, check := range report.Checks {
		if check.Status == preflight.CheckFailed {
			parts = append(parts, check.Name+":"+strings.TrimSpace(check.Summary))
		}
	}
	if len(parts) == 0 {
		return "ok"
	}
	return strings.Join(parts, "|")
}

func summarizeRepairRounds(rounds []RepairRound, report preflight.Report) string {
	if len(rounds) == 0 {
		return "未执行自动修复轮次"
	}
	last := rounds[len(rounds)-1]
	if !hasPreflightFailure(report) {
		return fmt.Sprintf("自动修复完成，共执行 %d 轮，当前预检已通过", len(rounds))
	}
	if last.TerminationReason != "" {
		return fmt.Sprintf("自动修复执行 %d 轮后终止: %s", len(rounds), last.TerminationReason)
	}
	return fmt.Sprintf("自动修复执行 %d 轮后仍存在失败项", len(rounds))
}

func (s *Service) runDockerValidation(ctx context.Context, workdir string, targets []string) SandboxValidation {
	targetArgs := validationArgs(targets)
	targetLabel := validationScopeLabel(targetArgs)
	commandScript := "export GOTOOLCHAIN=local && /usr/local/go/bin/go version && /usr/local/go/bin/go test " + strings.Join(targetArgs, " ") + " && /usr/local/go/bin/go vet " + strings.Join(targetArgs, " ")
	validation := SandboxValidation{
		Enabled:   s.config.Runtime.EnableDocker && s.config.Runtime.EnableDockerValidation,
		Targets:   append([]string{}, targetArgs...),
		Command:   "docker api runner: go test " + targetLabel + " && go vet " + targetLabel,
		StartedAt: time.Now(),
	}
	if !validation.Enabled {
		return validation
	}
	if strings.TrimSpace(workdir) == "" {
		workdir = s.root
	}
	if !s.canRunSandboxAction() {
		validation.Status = "skipped"
		validation.Summary = "docker 不可用，已跳过沙盒验证"
		validation.CompletedAt = time.Now()
		return validation
	}

	runner, err := executor.NewRunner()
	if err != nil {
		validation.Status = "failed"
		validation.Summary = "创建 Docker Runner 失败"
		validation.Output = err.Error()
		validation.CompletedAt = time.Now()
		return validation
	}

	validationCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	if workdir == s.root {
		cancel()
		validationCtx, cancel = context.WithTimeout(ctx, 90*time.Second)
	}
	defer cancel()

	result, err := runner.ValidateWorkspace(validationCtx, workdir, commandScript)
	if err != nil {
		validation.Status = "failed"
		validation.Summary = "Docker 沙盒验证失败"
		if validationCtx.Err() == context.DeadlineExceeded {
			validation.Summary = "Docker 沙盒验证超时"
		}
		validation.Output = trimActionOutput(err.Error())
		validation.CompletedAt = time.Now()
		return validation
	}

	validation.Output = trimActionOutput(strings.TrimSpace(result.Stdout + "\n" + result.Stderr))
	validation.CompletedAt = time.Now()
	if result.ExitCode != 0 {
		validation.Status = "failed"
		validation.Summary = fmt.Sprintf("Docker 沙盒退出码非 0: %d", result.ExitCode)
		return validation
	}

	validation.Status = "completed"
	validation.Summary = "Docker 沙盒验证通过 [" + targetLabel + "]"
	return validation
}
