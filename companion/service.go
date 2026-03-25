package companion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"multi_agent_cooperation/executor"
	"multi_agent_cooperation/llm"
	"multi_agent_cooperation/mcp"
	"multi_agent_cooperation/preflight"
	"multi_agent_cooperation/rag"
	"multi_agent_cooperation/snapshot"
	"multi_agent_cooperation/vision"
)

// SymbolSnapshot 是 MCP 风格项目符号表的摘要。
type SymbolSnapshot struct {
	PackageCount  int      `json:"package_count"`
	StructCount   int      `json:"struct_count"`
	FunctionCount int      `json:"function_count"`
	TopStructs    []string `json:"top_structs"`
	TopFunctions  []string `json:"top_functions"`
	Preview       string   `json:"preview"`
}

// KnowledgeSnapshot 是轻量 RAG 知识库摘要。
type KnowledgeSnapshot struct {
	FileCount  int `json:"file_count"`
	ChunkCount int `json:"chunk_count"`
}

// KnowledgeMatch 表示一次检索命中。
type KnowledgeMatch struct {
	Path    string  `json:"path"`
	Title   string  `json:"title"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// ProviderAttempt 记录一次模型尝试。
type ProviderAttempt struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Note    string `json:"note,omitempty"`
}

// Plan 是伴侣生成的结构化输出。
type Plan struct {
	Mode            string   `json:"mode,omitempty"`
	Overview        string   `json:"overview"`
	Actions         []string `json:"actions"`
	Deliverables    []string `json:"deliverables,omitempty"`
	ProgressSignals []string `json:"progress_signals,omitempty"`
	Risks           []string `json:"risks"`
	Innovations     []string `json:"innovations"`
	DesktopPetGaps  []string `json:"desktop_pet_gaps"`
	RAGUseCases     []string `json:"rag_use_cases"`
	NextSteps       []string `json:"next_steps"`
}

// GeneratedArtifact 描述一次运行自动导出的文档或结构化产物。
type GeneratedArtifact struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Summary string `json:"summary"`
	Path    string `json:"path"`
	URL     string `json:"url"`
}

// SnapshotReport 描述运行前后工作区的快照信息。
type SnapshotReport struct {
	Enabled        bool          `json:"enabled"`
	BeforePath     string        `json:"before_path,omitempty"`
	AfterPath      string        `json:"after_path,omitempty"`
	Diff           snapshot.Diff `json:"diff"`
	ChangedFiles   int           `json:"changed_files"`
	RollbackAdvice string        `json:"rollback_advice,omitempty"`
}

// TroubleshootingReport 汇总一次运行的排障结果。
type TroubleshootingReport struct {
	Status          string   `json:"status"`
	Issues          []string `json:"issues"`
	Recommendations []string `json:"recommendations"`
}

// ExecutionAction 描述一个可选的手动执行动作。
type ExecutionAction struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Kind             string     `json:"kind"`
	Status           string     `json:"status"`
	Summary          string     `json:"summary"`
	Command          string     `json:"command,omitempty"`
	Output           string     `json:"output,omitempty"`
	RequiresApproval bool       `json:"requires_approval"`
	LastExecutedAt   *time.Time `json:"last_executed_at,omitempty"`
}

// RunReport 是一次伴侣执行的完整报告。
type RunReport struct {
	ID               string                `json:"id"`
	Goal             string                `json:"goal"`
	CreatedAt        time.Time             `json:"created_at"`
	Complexity       ComplexityReport      `json:"complexity"`
	Route            RouteDecision         `json:"route"`
	UsedProvider     string                `json:"used_provider"`
	Attempts         []ProviderAttempt     `json:"attempts"`
	Steps            []ExecutionStep       `json:"steps"`
	Symbols          SymbolSnapshot        `json:"symbols"`
	Screen           ScreenContext         `json:"screen"`
	Knowledge        []KnowledgeMatch      `json:"knowledge"`
	Plan             Plan                  `json:"plan"`
	Artifacts        []GeneratedArtifact   `json:"artifacts,omitempty"`
	Snapshot         SnapshotReport        `json:"snapshot"`
	Troubleshoot     TroubleshootingReport `json:"troubleshoot"`
	ExecutionActions []ExecutionAction     `json:"execution_actions,omitempty"`
	Preflight        preflight.Report      `json:"preflight"`
	MarkdownPath     string                `json:"markdown_path"`
	JSONPath         string                `json:"json_path"`
	MarkdownURL      string                `json:"markdown_url"`
	JSONURL          string                `json:"json_url"`
}

// RunDigest 用于首页展示历史记录摘要。
type RunDigest struct {
	ID           string           `json:"id"`
	Goal         string           `json:"goal"`
	CreatedAt    time.Time        `json:"created_at"`
	UsedProvider string           `json:"used_provider"`
	Complexity   ComplexityReport `json:"complexity"`
	MarkdownURL  string           `json:"markdown_url"`
}

// DashboardState 是桌面工作台首屏状态。
type DashboardState struct {
	AppName         string            `json:"app_name"`
	Workspace       string            `json:"workspace"`
	GeneratedAt     time.Time         `json:"generated_at"`
	Providers       []ProviderStatus  `json:"providers"`
	Symbols         SymbolSnapshot    `json:"symbols"`
	Knowledge       KnowledgeSnapshot `json:"knowledge"`
	Screen          ScreenContext     `json:"screen"`
	Workflow        WorkflowState     `json:"workflow"`
	WorkflowBackend string            `json:"workflow_backend"`
	DockerStatus    string            `json:"docker_status"`
	RedisStatus     string            `json:"redis_status"`
	VisionStatus    string            `json:"vision_status"`
	LatestRun       *RunReport        `json:"latest_run,omitempty"`
	RecentRuns      []RunDigest       `json:"recent_runs"`
}

// Service 负责串联研发智能体工作台的各项能力。
type Service struct {
	config *Config
	root   string

	mu           sync.RWMutex
	bootstrapped bool
	providers    map[string]llm.Provider
	statuses     map[string]ProviderStatus
	index        *rag.Index
	symbols      SymbolSnapshot
	screen       ScreenContext
	workflow     WorkflowState
	dockerStatus string
	redisStatus  string
	visionStatus string
	runs         []*RunReport
	store        stateStore
	langGraph    *langGraphBridge
	vision       *vision.Analyzer
}

// NewService 创建研发智能体服务。
func NewService(config *Config, root string) *Service {
	return &Service{
		config:    config,
		root:      root,
		providers: make(map[string]llm.Provider),
		statuses:  make(map[string]ProviderStatus),
		workflow: WorkflowState{
			Status:    "idle",
			Phase:     "ready",
			Message:   "工作台初始化完成，等待任务",
			UpdatedAt: time.Now(),
		},
		redisStatus:  "redis not initialized",
		visionStatus: "vision not initialized",
		store:        &noopStateStore{status: "redis not initialized"},
	}
}

// Bootstrap 初始化 Provider、符号快照、RAG 索引和历史运行记录。
func (s *Service) Bootstrap(ctx context.Context) error {
	s.mu.Lock()
	if s.bootstrapped {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	s.initProviders()
	s.detectDocker()
	s.initEnhancements(ctx)

	if s.config.Runtime.EnableSymbols {
		if err := s.refreshSymbols(); err != nil {
			return err
		}
	}

	if s.config.Runtime.EnableRAG {
		if err := s.refreshKnowledge(); err != nil {
			return err
		}
	}

	if err := s.loadHistory(); err != nil {
		return err
	}

	s.mu.Lock()
	s.bootstrapped = true
	s.mu.Unlock()

	_ = ctx
	return nil
}

// State 返回桌面工作台需要的状态数据。
func (s *Service) State(ctx context.Context) (*DashboardState, error) {
	if err := s.Bootstrap(ctx); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	providers := make([]ProviderStatus, 0, len(s.statuses))
	for _, status := range s.statuses {
		providers = append(providers, status)
	}
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Name < providers[j].Name
	})

	recentRuns := make([]RunDigest, 0, len(s.runs))
	for _, run := range s.runs {
		recentRuns = append(recentRuns, RunDigest{
			ID:           run.ID,
			Goal:         run.Goal,
			CreatedAt:    run.CreatedAt,
			UsedProvider: run.UsedProvider,
			Complexity:   run.Complexity,
			MarkdownURL:  run.MarkdownURL,
		})
		if len(recentRuns) == 6 {
			break
		}
	}

	var latest *RunReport
	if len(s.runs) > 0 {
		latest = s.runs[0]
	}

	state := &DashboardState{
		AppName:         s.config.App.Name,
		Workspace:       s.config.App.Workspace,
		GeneratedAt:     time.Now(),
		Providers:       providers,
		Symbols:         s.symbols,
		Knowledge:       s.knowledgeSnapshot(),
		Screen:          s.screen,
		Workflow:        s.workflow,
		WorkflowBackend: s.config.Workflow.Backend,
		DockerStatus:    s.dockerStatus,
		RedisStatus:     s.redisStatus,
		VisionStatus:    s.visionStatus,
		LatestRun:       latest,
		RecentRuns:      recentRuns,
	}
	return state, nil
}

// Execute 对目标执行一次完整分析并写出报告。
func (s *Service) Execute(ctx context.Context, goal string) (*RunReport, error) {
	if err := s.Bootstrap(ctx); err != nil {
		return nil, err
	}

	goal = strings.TrimSpace(goal)
	if goal == "" {
		return nil, errors.New("goal cannot be empty")
	}

	s.updateWorkflow("running", "context", "正在收集 RAG、符号表和屏幕上下文", goal)
	var steps []ExecutionStep
	var err error
	snapshotReport := SnapshotReport{Enabled: s.config.Runtime.EnableSnapshots}
	var beforeSnapshot *snapshot.Manifest

	if s.config.Runtime.EnableSnapshots {
		s.updateWorkflow("running", "snapshot", "正在创建执行前快照", goal)
		snapshotStarted := time.Now()
		var beforePath string
		beforeSnapshot, beforePath, err = s.createSnapshot("before")
		if err == nil {
			snapshotReport.BeforePath = beforePath
			steps = append(steps, ExecutionStep{
				Name:        "snapshot-before",
				Status:      "completed",
				Summary:     fmt.Sprintf("已生成执行前快照: %s (%d files)", filepath.Base(beforePath), len(beforeSnapshot.Files)),
				StartedAt:   snapshotStarted,
				CompletedAt: time.Now(),
			})
		} else {
			steps = append(steps, ExecutionStep{
				Name:        "snapshot-before",
				Status:      "failed",
				Summary:     snapErrSummary(err),
				StartedAt:   snapshotStarted,
				CompletedAt: time.Now(),
			})
		}
	}

	contextStarted := time.Now()
	complexity := AssessComplexity(goal)
	statuses := s.copyStatuses()
	route := DecideRoute(goal, complexity, statuses)
	matches := s.retrieveKnowledge(goal)
	screen := s.screenSnapshot()
	screen = s.ensureScreenAnalysis(ctx, screen)
	steps = append(steps, ExecutionStep{
		Name:        "context",
		Status:      "completed",
		Summary:     screenSummary(screen, len(matches)),
		StartedAt:   contextStarted,
		CompletedAt: time.Now(),
	})

	s.updateWorkflow("running", "planning", "正在生成执行方案", goal)
	planningStarted := time.Now()
	plan, usedProvider, attempts, err := s.generatePlan(ctx, goal, complexity, route, matches, screen)
	if err != nil {
		s.updateWorkflow("error", "planning", err.Error(), goal)
		return nil, err
	}
	steps = append(steps, ExecutionStep{
		Name:        "planning",
		Status:      "completed",
		Summary:     fmt.Sprintf("已使用 %s 生成方案", usedProvider),
		StartedAt:   planningStarted,
		CompletedAt: time.Now(),
	})

	if s.config.Runtime.EnableDocker && s.config.Runtime.EnableSandboxSmoke {
		s.updateWorkflow("running", "sandbox", "正在验证 Docker 沙箱执行链路", goal)
		sandboxStarted := time.Now()
		summary := s.runSandboxSmoke()
		steps = append(steps, ExecutionStep{
			Name:        "sandbox-smoke",
			Status:      summary.Status,
			Summary:     summary.Summary,
			StartedAt:   sandboxStarted,
			CompletedAt: time.Now(),
		})
	}

	preflightReport := preflight.Report{}
	if s.config.Runtime.EnablePreflight {
		s.updateWorkflow("running", "preflight", "正在执行工程预检", goal)
		preflightStarted := time.Now()
		preflightReport = preflight.Run(ctx, s.root)
		steps = append(steps, ExecutionStep{
			Name:        "preflight",
			Status:      "completed",
			Summary:     preflightSummary(preflightReport),
			StartedAt:   preflightStarted,
			CompletedAt: time.Now(),
		})
	}

	if snapshotReport.Enabled && beforeSnapshot != nil {
		snapshotStarted := time.Now()
		afterSnapshot, afterPath, snapErr := s.createSnapshot("after")
		if snapErr == nil {
			snapshotReport.AfterPath = afterPath
			snapshotReport.Diff = snapshot.Compare(beforeSnapshot, afterSnapshot)
			snapshotReport.ChangedFiles = len(snapshotReport.Diff.Added) + len(snapshotReport.Diff.Modified) + len(snapshotReport.Diff.Deleted)
			if snapshotReport.ChangedFiles > 0 {
				snapshotReport.RollbackAdvice = "如果后续自动执行产生异常，可基于 before 快照对差异文件回滚。"
			}
			steps = append(steps, ExecutionStep{
				Name:        "snapshot-after",
				Status:      "completed",
				Summary:     fmt.Sprintf("已生成执行后快照: %s，差异文件 %d 个", filepath.Base(afterPath), snapshotReport.ChangedFiles),
				StartedAt:   snapshotStarted,
				CompletedAt: time.Now(),
			})
		} else {
			steps = append(steps, ExecutionStep{
				Name:        "snapshot-after",
				Status:      "failed",
				Summary:     snapErrSummary(snapErr),
				StartedAt:   snapshotStarted,
				CompletedAt: time.Now(),
			})
		}
	}

	s.updateWorkflow("running", "persisting", "正在写入运行报告", goal)
	report := &RunReport{
		ID:               time.Now().Format("20060102150405"),
		Goal:             goal,
		CreatedAt:        time.Now(),
		Complexity:       complexity,
		Route:            route,
		UsedProvider:     usedProvider,
		Attempts:         attempts,
		Steps:            steps,
		Symbols:          s.symbols,
		Screen:           screen,
		Knowledge:        matches,
		Plan:             plan,
		Snapshot:         snapshotReport,
		Troubleshoot:     buildTroubleshooting(attempts, preflightReport, snapshotReport),
		ExecutionActions: s.buildExecutionActions(snapshotReport, preflightReport),
		Preflight:        preflightReport,
	}

	if err := s.persistRun(report); err != nil {
		s.updateWorkflow("error", "persisting", err.Error(), goal)
		return nil, err
	}

	s.mu.Lock()
	s.runs = append([]*RunReport{report}, s.runs...)
	s.mu.Unlock()
	s.persistRuntimeState()
	s.updateWorkflow("idle", "ready", "工作台就绪，等待下一次任务", "")

	return report, nil
}

// ExecuteAction 对指定运行报告执行一个审批动作，并把结果回写到同一份报告。
func (s *Service) ExecuteAction(ctx context.Context, runID, actionID string) (*RunReport, error) {
	if err := s.Bootstrap(ctx); err != nil {
		return nil, err
	}

	runID = strings.TrimSpace(runID)
	actionID = strings.TrimSpace(actionID)
	if runID == "" || actionID == "" {
		return nil, errors.New("run_id and action_id cannot be empty")
	}

	s.updateWorkflow("running", "approval-action", "正在执行审批动作", "")

	s.mu.Lock()
	report, actionIndex, err := s.lookupRunActionLocked(runID, actionID)
	if err != nil {
		s.mu.Unlock()
		s.updateWorkflow("error", "approval-action", err.Error(), "")
		return nil, err
	}

	report.ExecutionActions[actionIndex].Status = "running"
	report.ExecutionActions[actionIndex].Summary = "已批准，正在执行"
	startedAt := time.Now()
	report.ExecutionActions[actionIndex].LastExecutedAt = &startedAt
	action := report.ExecutionActions[actionIndex]
	reportGoal := report.Goal
	s.mu.Unlock()

	outcome := s.performAction(ctx, action, report)

	s.mu.Lock()
	report, actionIndex, err = s.lookupRunActionLocked(runID, actionID)
	if err != nil {
		s.mu.Unlock()
		s.updateWorkflow("error", "approval-action", err.Error(), reportGoal)
		return nil, err
	}

	report.ExecutionActions[actionIndex].Status = outcome.Status
	report.ExecutionActions[actionIndex].Summary = outcome.Summary
	report.ExecutionActions[actionIndex].Output = outcome.Output
	finishedAt := time.Now()
	report.ExecutionActions[actionIndex].LastExecutedAt = &finishedAt
	if outcome.Step != nil {
		report.Steps = append(report.Steps, *outcome.Step)
	}
	if outcome.Preflight != nil {
		report.Preflight = *outcome.Preflight
	}
	report.Troubleshoot = buildTroubleshooting(report.Attempts, report.Preflight, report.Snapshot)
	updated := report
	s.mu.Unlock()

	if err := s.persistRun(updated); err != nil {
		s.updateWorkflow("error", "approval-action", err.Error(), reportGoal)
		return nil, err
	}
	s.persistRuntimeState()

	if outcome.Status == "failed" {
		s.updateWorkflow("error", "approval-action", outcome.Summary, reportGoal)
		return updated, nil
	}
	s.updateWorkflow("idle", "ready", "审批动作执行完成，等待下一次任务", "")
	return updated, nil
}

func (s *Service) initProviders() {
	statuses := map[string]ProviderStatus{}
	providers := map[string]llm.Provider{}

	for _, providerConfig := range s.config.Providers {
		status := ProviderStatus{
			Name:         providerConfig.Name,
			DefaultModel: providerConfig.DefaultModel,
			Models:       providerConfig.Models,
		}

		if providerConfig.Name != "mock" && providerConfig.APIKey == "" && providerConfig.Name != "ollama" {
			status.Reason = "未配置 API Key"
			statuses[providerConfig.Name] = status
			continue
		}

		provider, err := llm.Create(providerConfig)
		if err != nil {
			status.Reason = err.Error()
			statuses[providerConfig.Name] = status
			continue
		}

		status.Ready = true
		status.Reason = "已配置"

		if providerConfig.Name == "ollama" {
			if err := probeHTTP(providerConfig.BaseURL); err != nil {
				status.Ready = false
				status.Reason = "本地 Ollama 未响应"
			}
		}

		statuses[providerConfig.Name] = status
		if status.Ready {
			providers[providerConfig.Name] = provider
		}
	}

	if _, ok := providers["mock"]; !ok {
		mockProvider, _ := llm.Create(llm.ProviderConfig{Name: "mock"})
		providers["mock"] = mockProvider
		statuses["mock"] = ProviderStatus{
			Name:         "mock",
			Ready:        true,
			Reason:       "离线回退可用",
			DefaultModel: "companion-mock-v1",
			Models:       []string{"companion-mock-v1"},
		}
	}

	s.mu.Lock()
	s.statuses = statuses
	s.providers = providers
	s.mu.Unlock()
}

func (s *Service) refreshSymbols() error {
	inspector := mcp.NewInspector(s.root)
	codeInfo, err := inspector.ScanProject()
	if err != nil {
		return fmt.Errorf("failed to scan symbols: %w", err)
	}

	structNames := make([]string, 0, len(codeInfo.Structs))
	for _, item := range codeInfo.Structs {
		structNames = append(structNames, fmt.Sprintf("%s.%s", item.Package, item.Name))
	}
	functionNames := make([]string, 0, len(codeInfo.Functions))
	for _, item := range codeInfo.Functions {
		functionNames = append(functionNames, fmt.Sprintf("%s.%s", item.Package, item.Name))
	}

	sort.Strings(structNames)
	sort.Strings(functionNames)

	snapshot := SymbolSnapshot{
		PackageCount:  len(codeInfo.Packages),
		StructCount:   len(codeInfo.Structs),
		FunctionCount: len(codeInfo.Functions),
		TopStructs:    topN(structNames, 8),
		TopFunctions:  topN(functionNames, 10),
		Preview:       buildSymbolPreview(codeInfo.Packages, structNames, functionNames),
	}

	s.mu.Lock()
	s.symbols = snapshot
	s.mu.Unlock()
	return nil
}

func (s *Service) refreshKnowledge() error {
	index, err := rag.Build(s.root, rag.Options{
		IncludePaths:     s.config.Knowledge.IncludePaths,
		ExcludeNames:     s.config.Knowledge.ExcludeNames,
		MaxFileSizeBytes: int64(s.config.Knowledge.MaxFileSizeKB) * 1024,
		ChunkSize:        s.config.Knowledge.ChunkSize,
	})
	if err != nil {
		return fmt.Errorf("failed to build rag index: %w", err)
	}

	s.mu.Lock()
	s.index = index
	s.mu.Unlock()
	return nil
}

func (s *Service) detectDocker() {
	if _, err := exec.LookPath("docker"); err != nil {
		s.dockerStatus = "docker CLI 未安装或不可见"
		return
	}
	s.dockerStatus = "docker CLI 可用，后续可接入隔离执行"
}

func (s *Service) knowledgeSnapshot() KnowledgeSnapshot {
	if s.index == nil {
		return KnowledgeSnapshot{}
	}
	return KnowledgeSnapshot{
		FileCount:  s.index.FileCount(),
		ChunkCount: s.index.ChunkCount(),
	}
}

func (s *Service) retrieveKnowledge(goal string) []KnowledgeMatch {
	s.mu.RLock()
	index := s.index
	s.mu.RUnlock()

	if index == nil || !s.config.Runtime.EnableRAG {
		return nil
	}

	results := index.Search(goal, s.config.Runtime.MaxKnowledgeResults)
	matches := make([]KnowledgeMatch, 0, len(results))
	for _, result := range results {
		matches = append(matches, KnowledgeMatch{
			Path:    result.Path,
			Title:   result.Title,
			Content: truncate(result.Content, 260),
			Score:   result.Score,
		})
	}
	return matches
}

func (s *Service) generatePlan(ctx context.Context, goal string, complexity ComplexityReport, route RouteDecision, matches []KnowledgeMatch, screen ScreenContext) (Plan, string, []ProviderAttempt, error) {
	if s.config.Workflow.Backend == "langgraph_http" && s.langGraph != nil {
		plan, usedProvider, err := s.langGraph.GeneratePlan(ctx, langGraphRequest{
			Goal:       goal,
			Complexity: complexity,
			Route:      route,
			Symbols:    s.symbols,
			Screen:     screen,
			Knowledge:  matches,
		})
		if err == nil {
			return plan, coalesce(usedProvider, "langgraph-http"), []ProviderAttempt{
				{Name: "langgraph-http", Success: true, Note: "external orchestration backend"},
			}, nil
		}
	}

	systemPrompt := "你是面向 Go 工程的研发智能体与开发伴侣，不是只会泛泛谈架构升级的顾问。你必须先判断当前任务更接近页面分析、文档导出、轻量产物生成，还是复杂闭环开发。除非用户明确要求改造平台，否则不要把回答强行写成“先重做整个工作台或交互层”。请基于当前问题、屏幕内容、RAG 命中和符号快照直接回答用户任务。只返回 JSON，不要加代码块。JSON 必须包含：mode, overview, actions, deliverables, progress_signals, risks, innovations, desktop_pet_gaps, rag_use_cases, next_steps。"
	userPrompt := s.buildAnalysisPrompt(goal, complexity, route, matches, screen)

	var attempts []ProviderAttempt
	for _, providerName := range route.Attempts {
		s.mu.RLock()
		provider, ok := s.providers[providerName]
		s.mu.RUnlock()
		if !ok {
			attempts = append(attempts, ProviderAttempt{Name: providerName, Error: "provider unavailable"})
			continue
		}

		model := s.modelFor(providerName)
		attemptCtx, cancel := context.WithTimeout(ctx, s.providerTimeout(complexity.Level, providerName))
		started := time.Now()
		response, _, err := provider.Complete(attemptCtx, llm.CompletionRequest{
			Model: model,
			Messages: []llm.Message{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: userPrompt},
			},
			Temperature: 0.2,
			MaxTokens:   1400,
		})
		cancel()
		if err != nil {
			attempts = append(attempts, ProviderAttempt{
				Name:    providerName,
				Error:   fmt.Sprintf("%s (latency=%s)", err.Error(), time.Since(started).Round(time.Millisecond)),
				Success: false,
			})
			continue
		}

		plan := parsePlanResponse(response.Content, goal, matches)
		attempts = append(attempts, ProviderAttempt{
			Name:    providerName,
			Success: true,
			Note:    fmt.Sprintf("latency=%s", time.Since(started).Round(time.Millisecond)),
		})
		return plan, providerName, attempts, nil
	}

	plan := fallbackPlan(goal, matches)
	return plan, "mock", attempts, nil
}

func (s *Service) buildAnalysisPrompt(goal string, complexity ComplexityReport, route RouteDecision, matches []KnowledgeMatch, screen ScreenContext) string {
	var builder strings.Builder
	mode := inferTaskMode(goal, screen)
	builder.WriteString("目标: ")
	builder.WriteString(goal)
	builder.WriteString("\n")
	builder.WriteString("任务模式建议: ")
	builder.WriteString(mode)
	builder.WriteString("\n")
	builder.WriteString("复杂度等级: ")
	builder.WriteString(string(complexity.Level))
	builder.WriteString("\n")
	builder.WriteString("复杂度原因:\n")
	for _, reason := range complexity.Reasons {
		builder.WriteString("- ")
		builder.WriteString(reason)
		builder.WriteString("\n")
	}
	builder.WriteString("建议路由: ")
	builder.WriteString(route.Reason)
	builder.WriteString("\n")
	builder.WriteString("执行要求:\n")
	builder.WriteString("- overview 必须先直接回答用户当前问题，不要先讲平台升级\n")
	builder.WriteString("- deliverables 必须给出至少 2 个可导出的文档、代码或项目产物名称\n")
	builder.WriteString("- progress_signals 必须告诉用户闭环运行时如何判断进行到哪一步\n")
	builder.WriteString("- 如果低阶模型超时或失败，应升级到更高能力模型\n")
	builder.WriteString("- 输出要能支持闭环执行、排障和后续回滚\n")
	builder.WriteString("- 如果用户只是让你分析当前页面、整理结构或导出文档，不要把回答升级成大型平台改造\n")
	builder.WriteString("- 如果用户明确要求开发项目或持续闭环，再输出工程化开发方案与代码/文档交付物\n")
	builder.WriteString("模式提示:\n")
	builder.WriteString(taskModePrompt(mode))
	builder.WriteString("MCP 风格符号快照:\n")
	builder.WriteString(s.symbols.Preview)
	builder.WriteString("\n")

	if screen.Available {
		builder.WriteString("桌面屏幕上下文:\n")
		builder.WriteString(fmt.Sprintf("- 来源: %s\n", screen.SourceLabel))
		builder.WriteString(fmt.Sprintf("- 分辨率: %dx%d\n", screen.Width, screen.Height))
		builder.WriteString(fmt.Sprintf("- 捕捉时间: %s\n", screen.CapturedAt.Format("2006-01-02 15:04:05")))
		builder.WriteString(fmt.Sprintf("- 工件路径: %s\n", screen.ImagePath))
		if strings.TrimSpace(screen.AppHint) != "" {
			builder.WriteString(fmt.Sprintf("- 应用猜测: %s\n", screen.AppHint))
		}
		if strings.TrimSpace(screen.VisionSummary) != "" {
			builder.WriteString("- 视觉摘要: ")
			builder.WriteString(screen.VisionSummary)
			builder.WriteString("\n")
		}
		if strings.TrimSpace(screen.OCRText) != "" {
			builder.WriteString("- OCR 文本: ")
			builder.WriteString(truncate(screen.OCRText, 400))
			builder.WriteString("\n")
		}
	} else {
		builder.WriteString("桌面屏幕上下文:\n- 当前没有可用的屏幕捕捉工件\n")
	}

	if len(matches) > 0 {
		builder.WriteString("RAG 检索命中:\n")
		for _, match := range matches {
			builder.WriteString("- ")
			builder.WriteString(match.Path)
			builder.WriteString(": ")
			builder.WriteString(match.Content)
			builder.WriteString("\n")
		}
	}

	builder.WriteString("请输出一个既能回答当前任务，又能支撑后续研发闭环的结构化方案。")
	return builder.String()
}

func inferTaskMode(goal string, screen ScreenContext) string {
	lower := strings.ToLower(strings.TrimSpace(goal))
	switch {
	case containsAny(lower, "文档", "markdown", "readme", "报告", "导出", "总结成", "整理成"):
		return "document"
	case containsAny(lower, "分析", "结构", "页面", "这个页面", "浏览器内容", "讲讲", "介绍", "梳理") && !containsAny(lower, "开发", "实现", "生成代码", "搭建", "闭环"):
		return "analysis"
	case containsAny(lower, "新建项目", "生成项目", "脚手架", "项目代码", "创建文件", "生成代码", "实现功能", "搭建"):
		return "scaffold"
	case containsAny(lower, "闭环", "持续运行", "自动修复", "自动提交", "长流程", "复杂项目"):
		return "closed_loop"
	case screen.Available:
		return "analysis"
	default:
		return "general"
	}
}

func taskModePrompt(mode string) string {
	switch mode {
	case "analysis":
		return "- 当前更像页面/项目分析任务，先给出对当前浏览器内容的结构判断，再补简单建议与导出文档。\n"
	case "document":
		return "- 当前更像文档导出任务，输出重点应是可落盘的文档结构、标题、章节和关键结论。\n"
	case "scaffold":
		return "- 当前更像轻量项目生成任务，输出重点应是项目结构、交付文件、代码骨架和最小可运行说明。\n"
	case "closed_loop":
		return "- 当前更像复杂闭环开发任务，输出重点应是阶段划分、实时进度信号、代码/文档交付物和风险控制。\n"
	default:
		return "- 先回答当前问题，再给出最小必要的后续动作和导出产物。\n"
	}
}

func containsAny(input string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(input, needle) {
			return true
		}
	}
	return false
}

func (s *Service) loadHistory() error {
	runDir := filepath.Join(s.config.App.DataDir, "runs")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(runDir)
	if err != nil {
		return err
	}

	var loaded []*RunReport
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(runDir, entry.Name()))
		if err != nil {
			continue
		}

		var report RunReport
		if err := json.Unmarshal(data, &report); err != nil {
			continue
		}
		loaded = append(loaded, &report)
	}

	sort.Slice(loaded, func(i, j int) bool {
		return loaded[i].CreatedAt.After(loaded[j].CreatedAt)
	})

	s.mu.Lock()
	s.runs = loaded
	s.mu.Unlock()
	return nil
}

func (s *Service) persistRun(report *RunReport) error {
	reportDir := filepath.Join(s.config.App.DataDir, "reports")
	runDir := filepath.Join(s.config.App.DataDir, "runs")
	exportDir := filepath.Join(s.config.App.DataDir, "exports")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		return err
	}

	report.MarkdownPath = filepath.Join(reportDir, report.ID+".md")
	report.JSONPath = filepath.Join(runDir, report.ID+".json")
	report.MarkdownURL = "/reports/" + filepath.Base(report.MarkdownPath)
	report.JSONURL = "/runs/" + filepath.Base(report.JSONPath)
	if err := s.persistRunArtifacts(report); err != nil {
		return err
	}

	if err := os.WriteFile(report.MarkdownPath, []byte(renderMarkdownReport(report)), 0o644); err != nil {
		return err
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(report.JSONPath, data, 0o644)
}

func (s *Service) persistRunArtifacts(report *RunReport) error {
	exportDir := filepath.Join(s.config.App.DataDir, "exports", report.ID)
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		return err
	}

	docName := "deliverable.md"
	if report.Plan.Mode == "analysis" {
		docName = "page_analysis.md"
	} else if report.Plan.Mode == "document" {
		docName = "document_export.md"
	} else if report.Plan.Mode == "scaffold" || report.Plan.Mode == "closed_loop" {
		docName = "project_package.md"
	}

	docPath := filepath.Join(exportDir, docName)
	if err := os.WriteFile(docPath, []byte(renderDeliverableDocument(report)), 0o644); err != nil {
		return err
	}

	planPath := filepath.Join(exportDir, "plan.json")
	planPayload, err := json.MarshalIndent(report.Plan, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(planPath, planPayload, 0o644); err != nil {
		return err
	}

	report.Artifacts = []GeneratedArtifact{
		{
			Name:    filepath.Base(docPath),
			Kind:    "markdown",
			Summary: "根据本次任务自动导出的交付文档，可直接查看或二次编辑。",
			Path:    docPath,
			URL:     "/exports/" + report.ID + "/" + filepath.Base(docPath),
		},
		{
			Name:    filepath.Base(planPath),
			Kind:    "json",
			Summary: "本次运行的结构化计划产物，适合后续脚本或二次处理。",
			Path:    planPath,
			URL:     "/exports/" + report.ID + "/" + filepath.Base(planPath),
		},
	}
	return nil
}

func (s *Service) buildExecutionActions(snapshotReport SnapshotReport, preflightReport preflight.Report) []ExecutionAction {
	var actions []ExecutionAction
	preflightFailed := hasPreflightFailure(preflightReport)
	workspaceChanged := snapshotReport.Enabled && snapshotReport.ChangedFiles > 0
	gitRepo := isGitRepository(s.root)

	if s.config.Runtime.EnablePreflight && preflightFailed {
		actions = append(actions, ExecutionAction{
			ID:               "rerun-preflight",
			Title:            "重新执行工程预检",
			Kind:             "preflight-rerun",
			Status:           "pending",
			Summary:          "上一轮预检存在失败项，建议先重新执行工程预检。",
			Command:          "go test ./... && go vet ./...",
			RequiresApproval: true,
		})
	}

	if snapshotReport.Enabled && snapshotReport.BeforePath != "" && workspaceChanged {
		actions = append(actions, ExecutionAction{
			ID:               "rollback-before-snapshot",
			Title:            "回滚到运行前快照",
			Kind:             "snapshot-rollback",
			Status:           "pending",
			Summary:          fmt.Sprintf("检测到 %d 个工作区差异文件，可按 before 快照执行恢复。", snapshotReport.ChangedFiles),
			Command:          "restore workspace from before snapshot",
			RequiresApproval: true,
		})
	}

	if preflightFailed && s.config.Runtime.EnableDocker && s.canRunSandboxAction() {
		actions = append(actions, ExecutionAction{
			ID:               "docker-self-heal",
			Title:            "Docker 自愈执行闭环",
			Kind:             "docker-self-heal",
			Status:           "pending",
			Summary:          "预检失败后，可在 Docker 中挂载当前仓库并执行 go test / go mod tidy / gofmt / go vet 的自愈链路。",
			Command:          "docker self-heal workspace",
			RequiresApproval: true,
		})
	}

	if preflightFailed {
		title := "自动修复后自检"
		summary := "执行 gofmt、go mod tidy、工程自检，尝试修复常见 Go 工程问题。"
		command := "gofmt -w && go mod tidy && go test ./..."
		if gitRepo {
			title = "自动修复后自检并提交"
			summary = "执行 gofmt、go mod tidy、工程自检；如果修复成功，会自动提交当前 Git 工作区变更。"
			command = "gofmt -w && go mod tidy && go test ./... && git commit"
		}
		actions = append(actions, ExecutionAction{
			ID:               "autofix-commit",
			Title:            title,
			Kind:             "autofix-commit",
			Status:           "pending",
			Summary:          summary,
			Command:          command,
			RequiresApproval: true,
		})
	}

	return actions
}

func (s *Service) canRunSandboxAction() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	return true
}

func (s *Service) lookupRunActionLocked(runID, actionID string) (*RunReport, int, error) {
	for _, run := range s.runs {
		if run.ID != runID {
			continue
		}
		for idx := range run.ExecutionActions {
			if run.ExecutionActions[idx].ID == actionID {
				return run, idx, nil
			}
		}
		return nil, -1, fmt.Errorf("action %s not found in run %s", actionID, runID)
	}
	return nil, -1, fmt.Errorf("run %s not found", runID)
}

type actionOutcome struct {
	Status    string
	Summary   string
	Output    string
	Preflight *preflight.Report
	Step      *ExecutionStep
}

func (s *Service) performAction(ctx context.Context, action ExecutionAction, report *RunReport) actionOutcome {
	switch action.Kind {
	case "preflight-rerun":
		return s.performPreflightAction(ctx)
	case "snapshot-rollback":
		return s.performRollbackAction(report)
	case "sandbox-smoke":
		return s.performSandboxAction()
	case "docker-self-heal":
		return s.performDockerSelfHealAction()
	case "autofix-commit":
		return s.performAutoFixCommitAction(ctx)
	default:
		return actionOutcome{
			Status:  "failed",
			Summary: "未知动作类型: " + action.Kind,
			Output:  "action kind is not supported",
		}
	}
}

func (s *Service) performPreflightAction(ctx context.Context) actionOutcome {
	started := time.Now()
	report := preflight.Run(ctx, s.root)
	status := "completed"
	summary := preflightSummary(report)
	if hasPreflightFailure(report) {
		status = "failed"
		summary = "工程预检仍存在失败项"
	}

	return actionOutcome{
		Status:    status,
		Summary:   summary,
		Output:    formatPreflightReport(report),
		Preflight: &report,
		Step: &ExecutionStep{
			Name:        "approval-preflight",
			Status:      status,
			Summary:     summary,
			StartedAt:   started,
			CompletedAt: time.Now(),
		},
	}
}

func (s *Service) performRollbackAction(report *RunReport) actionOutcome {
	started := time.Now()
	if report.Snapshot.BeforePath == "" {
		return actionOutcome{
			Status:  "failed",
			Summary: "缺少 before 快照，无法回滚",
			Output:  "snapshot.before_path is empty",
			Step: &ExecutionStep{
				Name:        "approval-rollback",
				Status:      "failed",
				Summary:     "缺少 before 快照，无法回滚",
				StartedAt:   started,
				CompletedAt: time.Now(),
			},
		}
	}

	before, err := snapshot.Load(report.Snapshot.BeforePath)
	if err != nil {
		return actionOutcome{
			Status:  "failed",
			Summary: "读取 before 快照失败",
			Output:  err.Error(),
			Step: &ExecutionStep{
				Name:        "approval-rollback",
				Status:      "failed",
				Summary:     "读取 before 快照失败",
				StartedAt:   started,
				CompletedAt: time.Now(),
			},
		}
	}

	current, err := snapshot.Capture(s.root, []string{
		".git", ".vscode", "bin", "node_modules", "data", "accounts",
	})
	if err != nil {
		return actionOutcome{
			Status:  "failed",
			Summary: "创建当前工作区快照失败",
			Output:  err.Error(),
			Step: &ExecutionStep{
				Name:        "approval-rollback",
				Status:      "failed",
				Summary:     "创建当前工作区快照失败",
				StartedAt:   started,
				CompletedAt: time.Now(),
			},
		}
	}

	diff := snapshot.Compare(before, current)
	total := len(diff.Added) + len(diff.Modified) + len(diff.Deleted)
	if total == 0 {
		return actionOutcome{
			Status:  "completed",
			Summary: "当前工作区与运行前快照一致，无需回滚",
			Output:  "no diff detected between workspace and before snapshot",
			Step: &ExecutionStep{
				Name:        "approval-rollback",
				Status:      "completed",
				Summary:     "当前工作区与运行前快照一致，无需回滚",
				StartedAt:   started,
				CompletedAt: time.Now(),
			},
		}
	}

	if err := snapshot.Restore(s.root, before, diff); err != nil {
		return actionOutcome{
			Status:  "failed",
			Summary: "快照回滚失败",
			Output:  err.Error(),
			Step: &ExecutionStep{
				Name:        "approval-rollback",
				Status:      "failed",
				Summary:     "快照回滚失败",
				StartedAt:   started,
				CompletedAt: time.Now(),
			},
		}
	}

	summary := fmt.Sprintf("已按 before 快照恢复工作区，处理 %d 个差异文件", total)
	return actionOutcome{
		Status:  "completed",
		Summary: summary,
		Output:  formatSnapshotDiff(diff),
		Step: &ExecutionStep{
			Name:        "approval-rollback",
			Status:      "completed",
			Summary:     summary,
			StartedAt:   started,
			CompletedAt: time.Now(),
		},
	}
}

func (s *Service) performSandboxAction() actionOutcome {
	started := time.Now()
	summary := s.runSandboxSmoke()
	status := summary.Status
	if status == "completed" {
		status = "completed"
	} else {
		status = "failed"
	}

	return actionOutcome{
		Status:  status,
		Summary: summary.Summary,
		Output:  summary.Summary,
		Step: &ExecutionStep{
			Name:        "approval-sandbox",
			Status:      status,
			Summary:     summary.Summary,
			StartedAt:   started,
			CompletedAt: time.Now(),
		},
	}
}

func hasPreflightFailure(report preflight.Report) bool {
	for _, check := range report.Checks {
		if check.Status == preflight.CheckFailed {
			return true
		}
	}
	return false
}

func formatPreflightReport(report preflight.Report) string {
	if len(report.Checks) == 0 {
		return "no preflight checks executed"
	}

	var builder strings.Builder
	for _, check := range report.Checks {
		builder.WriteString(check.Name)
		builder.WriteString(": ")
		builder.WriteString(string(check.Status))
		builder.WriteString(" | ")
		builder.WriteString(check.Summary)
		if check.Output != "" {
			builder.WriteString("\n")
			builder.WriteString(trimActionOutput(check.Output))
		}
		builder.WriteString("\n\n")
	}
	return strings.TrimSpace(builder.String())
}

func formatSnapshotDiff(diff snapshot.Diff) string {
	var sections []string
	if len(diff.Added) > 0 {
		sections = append(sections, "added: "+strings.Join(diff.Added, ", "))
	}
	if len(diff.Modified) > 0 {
		sections = append(sections, "modified: "+strings.Join(diff.Modified, ", "))
	}
	if len(diff.Deleted) > 0 {
		sections = append(sections, "deleted: "+strings.Join(diff.Deleted, ", "))
	}
	if len(sections) == 0 {
		return "no diff"
	}
	return trimActionOutput(strings.Join(sections, "\n"))
}

func trimActionOutput(output string) string {
	const limit = 2400
	output = strings.TrimSpace(output)
	if len(output) <= limit {
		return output
	}
	return output[:limit] + "\n...output truncated..."
}

func (s *Service) copyStatuses() map[string]ProviderStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]ProviderStatus, len(s.statuses))
	for name, status := range s.statuses {
		result[name] = status
	}
	return result
}

func (s *Service) modelFor(providerName string) string {
	config, ok := s.config.ProviderConfig(providerName)
	if !ok {
		return ""
	}
	return config.DefaultModel
}

func renderMarkdownReport(report *RunReport) string {
	var builder strings.Builder
	builder.WriteString("# Desk Companion Run Report\n\n")
	builder.WriteString("## Goal\n")
	builder.WriteString(report.Goal)
	builder.WriteString("\n\n")
	builder.WriteString("## Task Mode\n")
	builder.WriteString(fmt.Sprintf("- `%s`\n\n", coalesce(report.Plan.Mode, "general")))
	builder.WriteString("## Routing\n")
	builder.WriteString(fmt.Sprintf("- Complexity: `%s` (%d)\n", report.Complexity.Level, report.Complexity.Score))
	builder.WriteString(fmt.Sprintf("- Provider: `%s`\n", report.UsedProvider))
	builder.WriteString(fmt.Sprintf("- Decision: %s\n\n", report.Route.Reason))
	builder.WriteString("## Snapshot\n")
	if report.Snapshot.Enabled {
		builder.WriteString(fmt.Sprintf("- Before: `%s`\n", report.Snapshot.BeforePath))
		builder.WriteString(fmt.Sprintf("- After: `%s`\n", report.Snapshot.AfterPath))
		builder.WriteString(fmt.Sprintf("- Changed Files: `%d`\n", report.Snapshot.ChangedFiles))
		if report.Snapshot.RollbackAdvice != "" {
			builder.WriteString(fmt.Sprintf("- Rollback Advice: %s\n", report.Snapshot.RollbackAdvice))
		}
	} else {
		builder.WriteString("- Disabled\n")
	}
	builder.WriteString("\n")
	builder.WriteString("## Workflow Steps\n")
	for _, step := range report.Steps {
		builder.WriteString(fmt.Sprintf("- `%s`: %s (%s)\n", step.Name, step.Status, step.Summary))
	}
	builder.WriteString("\n")
	builder.WriteString("## Screen Context\n")
	if report.Screen.Available {
		builder.WriteString(fmt.Sprintf("- Source: `%s`\n", report.Screen.SourceLabel))
		builder.WriteString(fmt.Sprintf("- Size: `%dx%d`\n", report.Screen.Width, report.Screen.Height))
		builder.WriteString(fmt.Sprintf("- Captured At: `%s`\n", report.Screen.CapturedAt.Format("2006-01-02 15:04:05")))
		builder.WriteString(fmt.Sprintf("- Artifact: `%s`\n\n", report.Screen.ImagePath))
	} else {
		builder.WriteString("- None\n\n")
	}
	builder.WriteString("## Plan Overview\n")
	builder.WriteString(report.Plan.Overview)
	builder.WriteString("\n\n")

	writeList(&builder, "Actions", report.Plan.Actions)
	writeList(&builder, "Deliverables", report.Plan.Deliverables)
	writeList(&builder, "Progress Signals", report.Plan.ProgressSignals)
	writeList(&builder, "Risks", report.Plan.Risks)
	writeList(&builder, "Innovations", report.Plan.Innovations)
	writeList(&builder, "Desktop Pet Gaps", report.Plan.DesktopPetGaps)
	writeList(&builder, "RAG Use Cases", report.Plan.RAGUseCases)
	writeList(&builder, "Next Steps", report.Plan.NextSteps)

	builder.WriteString("## Knowledge Matches\n")
	if len(report.Knowledge) == 0 {
		builder.WriteString("- None\n\n")
	} else {
		for _, match := range report.Knowledge {
			builder.WriteString(fmt.Sprintf("- `%s`: %s\n", match.Path, match.Content))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("## Preflight\n")
	for _, check := range report.Preflight.Checks {
		builder.WriteString(fmt.Sprintf("- `%s`: %s (%s)\n", check.Name, check.Status, check.Summary))
	}
	builder.WriteString("\n")
	builder.WriteString("## Troubleshooting\n")
	builder.WriteString(fmt.Sprintf("- Status: `%s`\n", report.Troubleshoot.Status))
	for _, issue := range report.Troubleshoot.Issues {
		builder.WriteString(fmt.Sprintf("- Issue: %s\n", issue))
	}
	for _, recommendation := range report.Troubleshoot.Recommendations {
		builder.WriteString(fmt.Sprintf("- Recommendation: %s\n", recommendation))
	}
	builder.WriteString("\n")
	builder.WriteString("## Exported Artifacts\n")
	if len(report.Artifacts) == 0 {
		builder.WriteString("- None\n\n")
	} else {
		for _, artifact := range report.Artifacts {
			builder.WriteString(fmt.Sprintf("- `%s`: %s\n", artifact.Name, artifact.Summary))
			builder.WriteString(fmt.Sprintf("  - Path: `%s`\n", artifact.Path))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("## Manual Recovery Actions\n")
	if len(report.ExecutionActions) == 0 {
		builder.WriteString("- None\n\n")
	} else {
		for _, action := range report.ExecutionActions {
			builder.WriteString(fmt.Sprintf("- `%s`: %s (%s)\n", action.Title, action.Status, action.Summary))
			if action.Command != "" {
				builder.WriteString(fmt.Sprintf("  - Command: `%s`\n", action.Command))
			}
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

func renderDeliverableDocument(report *RunReport) string {
	var builder strings.Builder
	builder.WriteString("# Companion Deliverable\n\n")
	builder.WriteString("## Task\n")
	builder.WriteString(report.Goal)
	builder.WriteString("\n\n")
	builder.WriteString("## Mode\n")
	builder.WriteString(coalesce(report.Plan.Mode, "general"))
	builder.WriteString("\n\n")
	builder.WriteString("## Direct Answer\n")
	builder.WriteString(report.Plan.Overview)
	builder.WriteString("\n\n")
	if report.Screen.Available {
		builder.WriteString("## Browser / Screen Context\n")
		builder.WriteString(fmt.Sprintf("- Source: %s\n", report.Screen.SourceLabel))
		if report.Screen.AppHint != "" {
			builder.WriteString(fmt.Sprintf("- App Hint: %s\n", report.Screen.AppHint))
		}
		if report.Screen.VisionSummary != "" {
			builder.WriteString(fmt.Sprintf("- Visual Summary: %s\n", report.Screen.VisionSummary))
		}
		if report.Screen.OCRText != "" {
			builder.WriteString(fmt.Sprintf("- OCR: %s\n", truncate(report.Screen.OCRText, 400)))
		}
		builder.WriteString("\n")
	}
	writeList(&builder, "Deliverables", report.Plan.Deliverables)
	writeList(&builder, "Actions", report.Plan.Actions)
	writeList(&builder, "Progress Signals", report.Plan.ProgressSignals)
	writeList(&builder, "Risks", report.Plan.Risks)
	writeList(&builder, "Next Steps", report.Plan.NextSteps)
	if len(report.Knowledge) > 0 {
		builder.WriteString("## Knowledge Evidence\n")
		for _, match := range report.Knowledge {
			builder.WriteString(fmt.Sprintf("- %s: %s\n", match.Path, match.Content))
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

func writeList(builder *strings.Builder, title string, items []string) {
	builder.WriteString("## ")
	builder.WriteString(title)
	builder.WriteString("\n")
	if len(items) == 0 {
		builder.WriteString("- None\n\n")
		return
	}
	for _, item := range items {
		builder.WriteString("- ")
		builder.WriteString(item)
		builder.WriteString("\n")
	}
	builder.WriteString("\n")
}

func parsePlanResponse(content, goal string, matches []KnowledgeMatch) Plan {
	content = strings.TrimSpace(content)
	jsonText := content
	if start := strings.Index(content, "{"); start >= 0 {
		if end := strings.LastIndex(content, "}"); end > start {
			jsonText = content[start : end+1]
		}
	}

	var plan Plan
	if err := json.Unmarshal([]byte(jsonText), &plan); err == nil && plan.Overview != "" {
		return normalizePlan(plan)
	}

	return fallbackPlan(goal, matches)
}

func normalizePlan(plan Plan) Plan {
	if strings.TrimSpace(plan.Mode) == "" {
		plan.Mode = "general"
	}
	plan.Actions = ensureList(plan.Actions)
	plan.Deliverables = ensureList(plan.Deliverables)
	plan.ProgressSignals = ensureList(plan.ProgressSignals)
	plan.Risks = ensureList(plan.Risks)
	plan.Innovations = ensureList(plan.Innovations)
	plan.DesktopPetGaps = ensureList(plan.DesktopPetGaps)
	plan.RAGUseCases = ensureList(plan.RAGUseCases)
	plan.NextSteps = ensureList(plan.NextSteps)
	return plan
}

func ensureList(items []string) []string {
	if len(items) == 0 {
		return []string{"待补充"}
	}
	return items
}

func fallbackPlan(goal string, matches []KnowledgeMatch) Plan {
	mode := inferTaskMode(goal, ScreenContext{})
	ragHint := "优先把项目文档、代码和设计记录纳入检索上下文。"
	if len(matches) > 0 {
		ragHint = "已命中现有项目知识，可直接作为后续生成与重构的上下文。"
	}

	switch mode {
	case "analysis":
		return Plan{
			Mode:     mode,
			Overview: fmt.Sprintf("围绕“%s”，当前更适合先输出页面/项目分析，而不是直接上升为平台改造。建议先根据浏览器内容整理项目定位、功能模块、目标用户和演示路径，再导出成文档。", goal),
			Actions: []string{
				"先按页面内容梳理项目定位、核心模块和交互入口。",
				"把识别到的信息整理成一份结构化分析文档。",
				"给出 3 到 5 条轻量优化建议，而不是直接扩成复杂工程。",
			},
			Deliverables: []string{
				"页面项目分析.md",
				"项目结构梳理.md",
			},
			ProgressSignals: []string{
				"context: 已抓到屏幕、OCR 与知识上下文",
				"planning: 已整理项目结构、功能点和文档结论",
				"persisting: 已导出 Markdown / JSON 结果文件",
			},
			Risks: []string{
				"如果页面信息不足，分析会偏概括，需要结合更多上下文。",
				"没有真实代码仓库结构时，项目判断会更依赖屏幕内容。",
			},
			Innovations: []string{
				"把页面分析和项目交付文档导出合并成一条本地研发闭环。",
				"让工作台既能看上下文，也能沉淀结构化分析结果。",
			},
			DesktopPetGaps: []string{
				"还缺少跨应用持续监听与主动提醒。",
			},
			RAGUseCases: []string{
				ragHint,
			},
			NextSteps: []string{
				"继续补充页面截图或仓库上下文，让分析更具体。",
				"如果确认要实现项目，再切到 scaffold 或 closed_loop 模式。",
			},
		}
	case "document":
		return Plan{
			Mode:     mode,
			Overview: fmt.Sprintf("围绕“%s”，当前任务更适合输出可导出的文档包。建议先生成一份对当前页面/项目的说明文档，再补执行清单和后续实现路线。", goal),
			Actions: []string{
				"整理当前问题、页面线索和项目目标。",
				"输出一份可直接导出的 Markdown 文档。",
				"补一份后续实现或优化清单。",
			},
			Deliverables: []string{
				"任务说明文档.md",
				"执行清单.md",
			},
			ProgressSignals: []string{
				"context: 已收集页面与检索证据",
				"planning: 已生成文档结构和关键结论",
				"persisting: 已写入导出文档",
			},
			Risks: []string{
				"如果目标不够具体，文档会更偏通用模板。",
			},
			Innovations: []string{
				"把桌面问答直接转成可导出的项目文档。",
			},
			DesktopPetGaps: []string{
				"缺少一键同步到外部知识库或云端文档。",
			},
			RAGUseCases: []string{
				ragHint,
			},
			NextSteps: []string{
				"确认文档后，再决定是否让闭环继续生成代码或文件结构。",
			},
		}
	}

	return Plan{
		Mode:     mode,
		Overview: fmt.Sprintf("围绕“%s”，建议让研发智能体在回答当前任务的同时，输出可导出的文档和可继续执行的工程计划。对于复杂项目，再进入持续闭环开发。", goal),
		Actions: []string{
			"根据问题类型在分析、文档、脚手架和闭环开发之间切换。",
			"在生成前注入项目符号快照、屏幕上下文和知识检索结果。",
			"建立离线 mock 与在线 provider 的双运行模式。",
			"把测试、vet、格式化与后续沙箱执行纳入闭环。",
		},
		Deliverables: []string{
			"交付文档.md",
			"结构化计划.json",
			"后续项目实现清单.md",
		},
		ProgressSignals: []string{
			"context: 已收集屏幕、RAG 和符号快照",
			"planning: 已生成任务方案与交付物",
			"preflight: 已执行工程预检",
			"persisting: 已写入文档、JSON 和导出文件",
		},
		Risks: []string{
			"没有真实 API 时，结果偏演示和规划。",
			"如果要走产品化交互，还需要更稳定的 IDE、终端和通知集成。",
			"自动执行能力需要更明确的审批与回滚边界。",
		},
		Innovations: []string{
			"以本地研发工作台承载开发智能体，而不是单次对话工具。",
			"把 RAG 与符号表合并进上下文，减少模型幻觉。",
			"采用复杂度驱动的阶梯式路由，平衡成本与成功率。",
		},
		DesktopPetGaps: []string{
			"缺少更深的 IDE/终端上下文接入。",
			"缺少真正的代码写回、复测和自动提交主链。",
			"缺少跨周期任务的重试、熔断和恢复机制。",
		},
		RAGUseCases: []string{
			ragHint,
			"用项目历史设计、错误复盘和接口文档支撑代码生成。",
			"把使用文档、教学文档和配置文档作为长期记忆的一部分。",
		},
		NextSteps: []string{
			"先让简单分析和文档导出稳定可用。",
			"再让项目生成任务补到代码骨架和文件写入。",
			"最后补复杂闭环开发、自动修复和更稳定的本地入口。",
		},
	}
}

func buildSymbolPreview(packages, structs, functions []string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("包数量: %d\n", len(packages)))
	builder.WriteString("结构体样例:\n")
	for _, item := range topN(structs, 6) {
		builder.WriteString("- ")
		builder.WriteString(item)
		builder.WriteString("\n")
	}
	builder.WriteString("函数样例:\n")
	for _, item := range topN(functions, 8) {
		builder.WriteString("- ")
		builder.WriteString(item)
		builder.WriteString("\n")
	}
	return builder.String()
}

func topN(items []string, limit int) []string {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func truncate(input string, limit int) string {
	runes := []rune(strings.TrimSpace(input))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "..."
}

func coalesce(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}

func screenSummary(screen ScreenContext, matchCount int) string {
	if screen.Available {
		return fmt.Sprintf("已收集 %d 条知识命中，并带入桌面屏幕上下文（%s, %dx%d）", matchCount, screen.SourceLabel, screen.Width, screen.Height)
	}
	return fmt.Sprintf("已收集 %d 条知识命中，当前无屏幕上下文", matchCount)
}

func preflightSummary(report preflight.Report) string {
	if len(report.Checks) == 0 {
		return "未执行预检"
	}

	passed := 0
	failed := 0
	for _, check := range report.Checks {
		switch check.Status {
		case preflight.CheckPassed:
			passed++
		case preflight.CheckFailed:
			failed++
		}
	}
	return fmt.Sprintf("预检完成: passed=%d failed=%d total=%d", passed, failed, len(report.Checks))
}

func (s *Service) providerTimeout(level ComplexityLevel, providerName string) time.Duration {
	seconds := s.config.Runtime.ProviderTimeoutSec
	if seconds <= 0 {
		seconds = 20
	}
	switch level {
	case ComplexityHigh:
		seconds += 8
	case ComplexityLow:
		seconds -= 6
	}

	if providerName == "mock" {
		seconds = 8
	}
	if providerName == "ollama" {
		seconds += 6
	}
	if seconds < 5 {
		seconds = 5
	}
	return time.Duration(seconds) * time.Second
}

func (s *Service) createSnapshot(stage string) (*snapshot.Manifest, string, error) {
	manifest, err := snapshot.Capture(s.root, []string{
		".git", ".vscode", "bin", "node_modules", "data", "accounts",
	})
	if err != nil {
		return nil, "", err
	}

	dir := filepath.Join(s.config.App.DataDir, "snapshots")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, "", err
	}
	path := filepath.Join(dir, manifest.ID+"-"+stage+".json")
	if err := snapshot.Save(manifest, path); err != nil {
		return nil, "", err
	}
	return manifest, path, nil
}

type sandboxSummary struct {
	Status  string
	Summary string
}

func (s *Service) runSandboxSmoke() sandboxSummary {
	runner, err := executor.NewRunner()
	if err != nil {
		return sandboxSummary{Status: "failed", Summary: "创建 Docker Runner 失败: " + err.Error()}
	}

	result, err := runner.ExecuteCode(`package main
import "fmt"
func main() { fmt.Println("sandbox-ok") }
`)
	if err != nil {
		return sandboxSummary{Status: "failed", Summary: "Docker 沙箱执行失败: " + err.Error()}
	}
	if result.ExitCode != 0 {
		return sandboxSummary{Status: "failed", Summary: "Docker 沙箱退出码非 0"}
	}
	return sandboxSummary{Status: "completed", Summary: "Docker 沙箱验证通过: " + strings.TrimSpace(result.Stdout)}
}

func buildTroubleshooting(attempts []ProviderAttempt, preflightReport preflight.Report, snapshotReport SnapshotReport) TroubleshootingReport {
	report := TroubleshootingReport{
		Status:          "ok",
		Issues:          []string{},
		Recommendations: []string{},
	}

	for _, attempt := range attempts {
		if attempt.Success {
			continue
		}
		report.Issues = append(report.Issues, fmt.Sprintf("Provider %s 失败: %s", attempt.Name, attempt.Error))
	}

	for _, check := range preflightReport.Checks {
		if check.Status == preflight.CheckFailed {
			report.Issues = append(report.Issues, fmt.Sprintf("预检失败: %s (%s)", check.Name, check.Summary))
		}
	}

	if snapshotReport.Enabled && snapshotReport.ChangedFiles > 0 {
		report.Recommendations = append(report.Recommendations, "如果自动执行引入异常，可以基于 before 快照对变更文件回滚。")
	}
	if len(report.Issues) == 0 {
		report.Recommendations = append(report.Recommendations, "当前闭环运行正常，可继续接入自动执行和审批机制。")
		return report
	}

	report.Status = "attention"
	report.Recommendations = append(report.Recommendations, "优先检查失败的 Provider 返回值与超时策略。")
	report.Recommendations = append(report.Recommendations, "根据 preflight 结果先修复编译、导包和静态检查问题。")
	return report
}

func snapErrSummary(err error) string {
	if err == nil {
		return ""
	}
	return "快照创建失败: " + err.Error()
}

func probeHTTP(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	host := parsed.Host
	if !strings.Contains(host, ":") {
		switch parsed.Scheme {
		case "https":
			host += ":443"
		default:
			host += ":80"
		}
	}
	conn, err := net.DialTimeout("tcp", host, 1500*time.Millisecond)
	if err != nil {
		return err
	}
	return conn.Close()
}
