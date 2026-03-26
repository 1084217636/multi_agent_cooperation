package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// ExecutionResult 执行结果
type ExecutionResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// Runner Docker执行引擎
type Runner struct {
	client *client.Client
}

// NewRunner 创建新的执行引擎
func NewRunner() (*Runner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &Runner{client: cli}, nil
}

// ValidateWorkspace 在 Docker 中挂载工作区并执行验证脚本。
func (r *Runner) ValidateWorkspace(ctx context.Context, workdir, script string) (*ExecutionResult, error) {
	if strings.TrimSpace(workdir) == "" {
		return nil, fmt.Errorf("workdir cannot be empty")
	}
	if strings.TrimSpace(script) == "" {
		return nil, fmt.Errorf("script cannot be empty")
	}

	started := time.Now()
	reader, err := r.client.ImagePull(ctx, "golang:1.25", image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}
	io.Copy(io.Discard, reader)
	reader.Close()

	containerConfig := &container.Config{
		Image:      "golang:1.25",
		WorkingDir: "/workspace",
		Cmd:        []string{"sh", "-lc", script},
		Env: []string{
			"PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			"GOCACHE=/tmp/go-cache",
			"GOMODCACHE=/tmp/go-mod-cache",
			"GOTOOLCHAIN=local",
		},
	}

	hostConfig := &container.HostConfig{
		Binds: []string{workdir + ":/workspace"},
	}

	resp, err := r.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create validation container: %w", err)
	}
	defer r.client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})

	if err := r.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start validation container: %w", err)
	}

	statusCh, errCh := r.client.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			return nil, fmt.Errorf("validation wait error: %w", waitErr)
		}
	case <-statusCh:
	}

	logs, err := r.client.ContainerLogs(ctx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get validation logs: %w", err)
	}
	defer logs.Close()

	var stdoutBuf, stderrBuf strings.Builder
	if _, err := stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logs); err != nil {
		return nil, fmt.Errorf("failed to parse validation logs: %w", err)
	}

	inspect, err := r.client.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect validation container: %w", err)
	}

	return &ExecutionResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: inspect.State.ExitCode,
		Duration: time.Since(started),
	}, nil
}

// ExecuteCode 执行Go代码
func (r *Runner) ExecuteCode(code string) (*ExecutionResult, error) {
	ctx := context.Background()

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "gopher-os-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 写入代码到main.go
	mainFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write code file: %w", err)
	}

	// 创建go.mod文件
	goModFile := filepath.Join(tempDir, "go.mod")
	goModContent := `module gopher-os-exec

go 1.21
`
	if err := os.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write go.mod: %w", err)
	}

	// 拉取golang镜像
	reader, err := r.client.ImagePull(ctx, "golang:1.21-alpine", image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}
	io.Copy(io.Discard, reader)
	reader.Close()

	// 创建容器配置
	containerConfig := &container.Config{
		Image: "golang:1.21-alpine",
		Cmd:   []string{"go", "run", "/app/main.go"},
	}

	hostConfig := &container.HostConfig{
		Binds: []string{tempDir + ":/app"},
	}

	// 创建容器
	resp, err := r.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}
	defer r.client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})

	// 启动容器
	if err := r.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// 等待容器完成
	statusCh, errCh := r.client.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("container wait error: %w", err)
		}
	case <-statusCh:
	}

	// 获取容器日志
	logs, err := r.client.ContainerLogs(ctx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}
	defer logs.Close()

	// 分离stdout和stderr
	var stdoutBuf, stderrBuf strings.Builder
	_, err = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse logs: %w", err)
	}

	// 获取退出码
	inspect, err := r.client.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return &ExecutionResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: inspect.State.ExitCode,
	}, nil
}

// ExecuteWithRetry 带重试的执行
func (r *Runner) ExecuteWithRetry(code string, maxRetries int) (*ExecutionResult, error) {
	var lastError error
	for i := 0; i < maxRetries; i++ {
		result, err := r.ExecuteCode(code)
		if err == nil && result.ExitCode == 0 {
			return result, nil
		}
		if err != nil {
			lastError = err
		} else if result.Stderr != "" {
			lastError = fmt.Errorf("execution failed: %s", result.Stderr)
		}
		time.Sleep(time.Second * 2)
	}
	return nil, fmt.Errorf("max retries exceeded: %w", lastError)
}

// ExecuteWithGoModTidy 执行代码并自动运行go mod tidy
func (r *Runner) ExecuteWithGoModTidy(code string) (*ExecutionResult, error) {
	ctx := context.Background()

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "gopher-os-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 写入代码到main.go
	mainFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write code file: %w", err)
	}

	// 创建go.mod文件
	goModFile := filepath.Join(tempDir, "go.mod")
	goModContent := `module gopher-os-exec

go 1.21
`
	if err := os.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write go.mod: %w", err)
	}

	// 拉取golang镜像
	reader, err := r.client.ImagePull(ctx, "golang:1.21-alpine", image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}
	io.Copy(io.Discard, reader)
	reader.Close()

	// 先运行go mod tidy
	tidyContainerConfig := &container.Config{
		Image: "golang:1.21-alpine",
		Cmd:   []string{"go", "mod", "tidy"},
	}

	tidyHostConfig := &container.HostConfig{
		Binds: []string{tempDir + ":/app"},
	}

	tidyResp, err := r.client.ContainerCreate(ctx, tidyContainerConfig, tidyHostConfig, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create tidy container: %w", err)
	}
	defer r.client.ContainerRemove(ctx, tidyResp.ID, container.RemoveOptions{Force: true})

	if err := r.client.ContainerStart(ctx, tidyResp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start tidy container: %w", err)
	}

	// 等待tidy完成
	tidyStatusCh, tidyErrCh := r.client.ContainerWait(ctx, tidyResp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-tidyErrCh:
		if err != nil {
			return nil, fmt.Errorf("tidy wait error: %w", err)
		}
	case <-tidyStatusCh:
	}

	// 然后执行代码
	runContainerConfig := &container.Config{
		Image: "golang:1.21-alpine",
		Cmd:   []string{"go", "run", "main.go"},
	}

	runHostConfig := &container.HostConfig{
		Binds: []string{tempDir + ":/app"},
	}

	runResp, err := r.client.ContainerCreate(ctx, runContainerConfig, runHostConfig, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create run container: %w", err)
	}
	defer r.client.ContainerRemove(ctx, runResp.ID, container.RemoveOptions{Force: true})

	if err := r.client.ContainerStart(ctx, runResp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start run container: %w", err)
	}

	// 等待执行完成
	runStatusCh, runErrCh := r.client.ContainerWait(ctx, runResp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-runErrCh:
		if err != nil {
			return nil, fmt.Errorf("run wait error: %w", err)
		}
	case <-runStatusCh:
	}

	// 获取容器日志
	logs, err := r.client.ContainerLogs(ctx, runResp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}
	defer logs.Close()

	// 分离stdout和stderr
	var stdoutBuf, stderrBuf strings.Builder
	_, err = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse logs: %w", err)
	}

	// 获取退出码
	inspect, err := r.client.ContainerInspect(ctx, runResp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return &ExecutionResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: inspect.State.ExitCode,
	}, nil
}
