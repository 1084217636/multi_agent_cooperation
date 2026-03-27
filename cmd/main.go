package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"multi_agent_cooperation/companion"
	_ "multi_agent_cooperation/llm/providers"
)

func main() {
	var (
		mode       = flag.String("mode", "desktop", "运行模式: desktop | run | inspect")
		configPath = flag.String("config", "config.yaml", "配置文件路径")
		addr       = flag.String("addr", "", "桌面工作台监听地址")
		goal       = flag.String("goal", "", "单次运行任务目标")
		noBrowser  = flag.Bool("no-browser", false, "启动 desktop 模式时不自动打开浏览器")
	)
	flag.Parse()

	workdir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	cfg, err := companion.LoadConfig(*configPath, workdir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	if *addr != "" {
		cfg.App.HTTPAddr = *addr
	}
	if *noBrowser {
		cfg.App.AutoOpenBrowser = false
	}

	service := companion.NewService(cfg, workdir)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch *mode {
	case "desktop":
		dashboard := companion.NewDashboard(service, cfg.App.HTTPAddr, cfg.App.AutoOpenBrowser)
		fmt.Printf("Go R&D Agent workbench listening on http://%s\n", cfg.App.HTTPAddr)
		if err := dashboard.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "dashboard error: %v\n", err)
			os.Exit(1)
		}
	case "run":
		runGoal := strings.TrimSpace(*goal)
		if runGoal == "" {
			runGoal = strings.TrimSpace(strings.Join(flag.Args(), " "))
		}
		if runGoal == "" {
			runGoal = "把当前项目推进为面向 Go 工程的研发智能体，兼顾复杂度路由、RAG、AST 符号注入、工程预检与快照回滚。"
		}
		report, err := service.Execute(ctx, runGoal)
		if err != nil {
			fmt.Fprintf(os.Stderr, "run error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Provider: %s\n", report.UsedProvider)
		fmt.Printf("Complexity: %s (%d)\n", report.Complexity.Level, report.Complexity.Score)
		fmt.Printf("Overview: %s\n", report.Plan.Overview)
		fmt.Printf("Markdown report: %s\n", report.MarkdownPath)
		fmt.Printf("JSON artifact: %s\n", report.JSONPath)
	case "inspect":
		state, err := service.State(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "inspect error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("App: %s\n", state.AppName)
		fmt.Printf("Project Root: %s\n", state.ProjectRoot)
		fmt.Printf("Generated Root: %s\n", state.GeneratedRoot)
		fmt.Printf("Providers: %d\n", len(state.Providers))
		fmt.Printf("Symbols: packages=%d structs=%d functions=%d\n", state.Symbols.PackageCount, state.Symbols.StructCount, state.Symbols.FunctionCount)
		fmt.Printf("Knowledge: files=%d chunks=%d\n", state.Knowledge.FileCount, state.Knowledge.ChunkCount)
		fmt.Printf("Workflow Backend: %s\n", state.WorkflowBackend)
		fmt.Printf("Docker: %s\n", state.DockerStatus)
		fmt.Printf("Redis: %s\n", state.RedisStatus)
		fmt.Printf("Vision: %s\n", state.VisionStatus)
	default:
		fmt.Fprintf(os.Stderr, "unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}
