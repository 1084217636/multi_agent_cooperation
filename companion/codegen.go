package companion

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"multi_agent_cooperation/llm"
)

// GeneratedFileRecord 描述一次代码生成落盘后的文件摘要。
type GeneratedFileRecord struct {
	Path     string `json:"path"`
	Purpose  string `json:"purpose,omitempty"`
	Language string `json:"language,omitempty"`
	Bytes    int    `json:"bytes"`
	URL      string `json:"url,omitempty"`
}

// CodeGenerationReport 汇总代码生成与写回结果。
type CodeGenerationReport struct {
	Enabled         bool                  `json:"enabled"`
	Status          string                `json:"status,omitempty"`
	Summary         string                `json:"summary,omitempty"`
	Backend         string                `json:"backend,omitempty"`
	TargetMode      string                `json:"target_mode,omitempty"`
	PatchCandidates []string              `json:"patch_candidates,omitempty"`
	RejectedFiles   []string              `json:"rejected_files,omitempty"`
	OutputDir       string                `json:"output_dir,omitempty"`
	ManifestPath    string                `json:"manifest_path,omitempty"`
	RawResponsePath string                `json:"raw_response_path,omitempty"`
	Provider        string                `json:"provider,omitempty"`
	ParseMode       string                `json:"parse_mode,omitempty"`
	ParseError      string                `json:"parse_error,omitempty"`
	EntryPoints     []string              `json:"entry_points,omitempty"`
	RunCommands     []string              `json:"run_commands,omitempty"`
	Notes           []string              `json:"notes,omitempty"`
	Files           []GeneratedFileRecord `json:"files,omitempty"`
	StartedAt       time.Time             `json:"started_at,omitempty"`
	CompletedAt     time.Time             `json:"completed_at,omitempty"`
}

type codeBundlePayload struct {
	Summary     string              `json:"summary"`
	EntryPoints []string            `json:"entry_points"`
	RunCommands []string            `json:"run_commands"`
	Notes       []string            `json:"notes"`
	Files       []codeBundleFileDef `json:"files"`
}

type codeBundleFileDef struct {
	Path     string `json:"path"`
	Purpose  string `json:"purpose"`
	Language string `json:"language,omitempty"`
	Content  string `json:"content"`
}

type persistedCodeBundle struct {
	RunID      string              `json:"run_id"`
	Goal       string              `json:"goal"`
	Mode       string              `json:"mode"`
	TargetMode string              `json:"target_mode"`
	Candidates []string            `json:"candidates,omitempty"`
	Provider   string              `json:"provider"`
	ParseMode  string              `json:"parse_mode,omitempty"`
	Summary    string              `json:"summary"`
	Files      []codeBundleFileDef `json:"files"`
	EntryPoint []string            `json:"entry_points"`
	RunCommand []string            `json:"run_commands"`
	Notes      []string            `json:"notes"`
}

func shouldGenerateCode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case "scaffold", "closed_loop":
		return true
	default:
		return false
	}
}

func (s *Service) generateCodeBundle(ctx context.Context, runID string, goal string, plan Plan, complexity ComplexityReport, route RouteDecision, matches []KnowledgeMatch, screen ScreenContext) CodeGenerationReport {
	targetMode, outputDir, manifestPath, cleanOutput := s.resolveCodegenTarget(runID, goal, plan.Mode)
	patchCandidates := s.buildPatchCandidates(goal, matches, targetMode)
	report := CodeGenerationReport{
		Enabled:         shouldGenerateCode(plan.Mode),
		Status:          "skipped",
		TargetMode:      targetMode,
		PatchCandidates: patchCandidates,
		OutputDir:       outputDir,
		ManifestPath:    manifestPath,
		EntryPoints:     []string{},
		RunCommands:     []string{},
		Notes:           []string{},
		Files:           []GeneratedFileRecord{},
		StartedAt:       time.Now(),
	}
	if !report.Enabled {
		report.Summary = "当前任务模式无需生成代码包"
		report.CompletedAt = time.Now()
		return report
	}

	result := s.requestCodeBundle(ctx, runID, goal, plan, complexity, route, matches, screen, targetMode, patchCandidates)
	payload, providerName := result.Payload, result.ProviderName
	if providerName == "" {
		providerName = "fallback-template"
	}
	payload, rejected := restrictCodeBundlePayload(targetMode, patchCandidates, payload)
	if len(payload.Files) == 0 {
		fallback := fallbackCodeBundle(goal, plan, targetMode)
		payload = fallback
		rejected = append(rejected, "all generated files were blocked by current_repo_patch guard")
	}

	fileRecords, manifestPath, err := persistCodeBundle(report.OutputDir, report.ManifestPath, cleanOutput, runID, goal, plan.Mode, targetMode, providerName, result.ParseMode, patchCandidates, payload)
	report.Backend = coalesce(result.Backend, "builtin")
	report.Provider = providerName
	report.ParseMode = result.ParseMode
	report.ParseError = result.ParseError
	report.RejectedFiles = rejected
	report.EntryPoints = payload.EntryPoints
	report.RunCommands = payload.RunCommands
	report.Notes = payload.Notes
	if result.ParseError != "" {
		report.Notes = append(report.Notes, "parse error: "+result.ParseError)
	}
	if len(rejected) > 0 {
		report.Notes = append(report.Notes, "write guard rejected files: "+strings.Join(rejected, ", "))
	}
	if result.SourceProvider != "" && providerName == "fallback-template" {
		report.Notes = append(report.Notes, "source provider before fallback: "+result.SourceProvider)
	}
	report.ManifestPath = manifestPath
	for idx := range fileRecords {
		fileRecords[idx].URL = s.generatedFileURL(targetMode, report.OutputDir, fileRecords[idx].Path)
	}
	report.Files = fileRecords
	report.RawResponsePath = result.RawResponsePath
	report.CompletedAt = time.Now()

	if err != nil {
		report.Status = "failed"
		report.Summary = "代码包写入失败"
		report.Notes = append(report.Notes, err.Error())
		return report
	}

	report.Status = "completed"
	report.Summary = fmt.Sprintf("已生成 %d 个文件并写入 %s", len(fileRecords), report.OutputDir)
	if report.TargetMode == "current_repo_patch" {
		report.Summary = fmt.Sprintf("已向当前仓库写入 %d 个文件，并保留 bundle manifest 与快照回滚能力", len(fileRecords))
	}
	if payload.Summary != "" {
		report.Summary = payload.Summary
	}
	if len(report.RunCommands) == 0 {
		report.RunCommands = []string{"go test ./...", "go vet ./..."}
	}
	return report
}

type codeBundleResult struct {
	Payload         codeBundlePayload
	ProviderName    string
	Backend         string
	ParseMode       string
	RawResponsePath string
	ParseError      string
	SourceProvider  string
}

func (s *Service) resolveCodegenTarget(runID, goal, mode string) (targetMode, outputDir, manifestPath string, cleanOutput bool) {
	targetMode = "isolated_workspace"
	outputDir = filepath.Join(s.generatedRoot(), runID)
	manifestPath = filepath.Join(outputDir, "bundle_manifest.json")
	cleanOutput = true

	lowerGoal := strings.ToLower(strings.TrimSpace(goal))
	if shouldPatchCurrentRepo(lowerGoal, mode) {
		targetMode = "current_repo_patch"
		outputDir = s.root
		manifestPath = filepath.Join(s.config.App.DataDir, "patches", runID, "bundle_manifest.json")
		cleanOutput = false
	}
	return targetMode, outputDir, manifestPath, cleanOutput
}

func shouldPatchCurrentRepo(goal, mode string) bool {
	if strings.TrimSpace(mode) != "closed_loop" && strings.TrimSpace(mode) != "scaffold" {
		return false
	}
	return containsAny(goal,
		"当前项目", "当前仓库", "这个项目", "本项目", "在当前项目里", "在这个项目里",
		"修复当前项目", "补齐当前项目", "完善当前项目", "修改当前项目",
		"current project", "current repo", "this repo", "patch repository", "patch current project",
	)
}

func (s *Service) requestCodeBundle(ctx context.Context, runID, goal string, plan Plan, complexity ComplexityReport, route RouteDecision, matches []KnowledgeMatch, screen ScreenContext, targetMode string, patchCandidates []string) codeBundleResult {
	if s.shouldUseLangGraphBackend() && s.langGraph != nil {
		payload, usedProvider, err := s.langGraph.GenerateCodeBundle(ctx, langGraphRequest{
			Goal:            goal,
			Complexity:      complexity,
			Route:           route,
			Symbols:         s.symbols,
			Screen:          screen,
			Knowledge:       matches,
			Plan:            plan,
			TargetMode:      targetMode,
			PatchCandidates: patchCandidates,
		})
		if err == nil {
			return codeBundleResult{
				Payload:        payload,
				ProviderName:   coalesce(usedProvider, "langgraph-http"),
				Backend:        "langgraph_http",
				ParseMode:      "langgraph_payload",
				SourceProvider: usedProvider,
			}
		}
	}

	systemPrompt := buildCodegenSystemPrompt(targetMode)
	userPrompt := s.buildCodegenPrompt(goal, plan, complexity, matches, screen, targetMode, patchCandidates)
	lastRawPath := ""
	lastProvider := ""
	lastParseError := ""

	for _, providerName := range route.Attempts {
		if providerName == "mock" {
			continue
		}
		s.mu.RLock()
		provider, ok := s.providers[providerName]
		s.mu.RUnlock()
		if !ok {
			continue
		}
		attemptCtx, cancel := context.WithTimeout(ctx, s.providerTimeout(complexity.Level, providerName))
		response, _, err := provider.Complete(attemptCtx, llm.CompletionRequest{
			Model: s.modelFor(providerName),
			Messages: []llm.Message{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: userPrompt},
			},
			Temperature: 0.15,
			MaxTokens:   3200,
		})
		cancel()
		if err != nil {
			continue
		}
		rawPath := writeCodegenRawResponse(s.config.App.DataDir, runID, providerName, "initial", response.Content)
		lastRawPath = rawPath
		lastProvider = providerName
		payload, parseMode, parseErr := parseCodeBundleResponse(response.Content)
		if parseErr == nil {
			return codeBundleResult{
				Payload:         payload,
				ProviderName:    providerName,
				Backend:         "builtin",
				ParseMode:       parseMode,
				RawResponsePath: rawPath,
				SourceProvider:  providerName,
			}
		}
		lastParseError = parseErr.Error()
		repaired, repairedParseMode, repairRaw, repairErr := s.repairCodeBundleResponse(ctx, providerName, response.Content, targetMode)
		if repairErr == nil {
			repairPath := writeCodegenRawResponse(s.config.App.DataDir, runID, providerName, "repaired", repairRaw)
			return codeBundleResult{
				Payload:         repaired,
				ProviderName:    providerName,
				Backend:         "builtin",
				ParseMode:       repairedParseMode,
				RawResponsePath: coalesce(repairPath, rawPath),
				SourceProvider:  providerName,
			}
		}
		lastParseError = repairErr.Error()
	}

	return codeBundleResult{
		Payload:         fallbackCodeBundle(goal, plan, targetMode),
		ProviderName:    "fallback-template",
		Backend:         "builtin",
		ParseMode:       "fallback",
		RawResponsePath: lastRawPath,
		ParseError:      lastParseError,
		SourceProvider:  lastProvider,
	}
}

func buildCodegenSystemPrompt(targetMode string) string {
	if targetMode == "current_repo_patch" {
		return "你是面向 Go 单仓研发任务的仓库补丁生成器。你需要基于目标、计划、符号表和检索上下文，对当前仓库输出一个最小补丁包。只返回 JSON，不要加代码块。JSON 必须包含：summary, entry_points, run_commands, notes, files。files 中每项必须包含 path, purpose, language, content。要求：1. path 必须是相对路径，不能包含 ..；2. 优先修改当前仓库已有文件，必要时才新增测试、文档或辅助文件；3. 不要生成新的 go.mod，也不要重写整个项目骨架；4. 输出文件数控制在 1 到 5 个之间；5. 如果当前任务主要是修复预检、Docker、快照或路由链，可以只返回相关文件。"
	}
	return "你是面向 Go 单仓研发任务的代码实现器。你需要基于目标、计划、符号表和检索上下文，输出一个最小但可运行的 Go 项目代码包。只返回 JSON，不要加代码块。JSON 必须包含：summary, entry_points, run_commands, notes, files。files 中每项必须包含 path, purpose, language, content。要求：1. path 必须是相对路径，不能包含 ..；2. 至少返回 go.mod、README.md、一个可运行入口和一个测试文件；3. 代码要尽量能通过 go test ./...；4. 输出文件数控制在 4 到 8 个之间。"
}

func (s *Service) repairCodeBundleResponse(ctx context.Context, providerName, rawContent, targetMode string) (codeBundlePayload, string, string, error) {
	s.mu.RLock()
	provider, ok := s.providers[providerName]
	s.mu.RUnlock()
	if !ok {
		return codeBundlePayload{}, "", "", fmt.Errorf("provider unavailable")
	}

	repairedCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	targetHint := "独立工作区模式，可以返回最小脚手架。"
	if targetMode == "current_repo_patch" {
		targetHint = "当前仓库补丁模式，优先返回补丁文件或补丁说明，不要覆盖核心 go.mod。"
	}

	response, _, err := provider.Complete(repairedCtx, llm.CompletionRequest{
		Model: s.modelFor(providerName),
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "你是 JSON 结构修复器。请把输入内容整理成严格 JSON，字段必须包含：summary, entry_points, run_commands, notes, files。files 中每项必须包含：path, purpose, language, content。只返回 JSON。",
			},
			{
				Role:    "user",
				Content: "请把下面内容转换成严格 JSON。\n\n模式提示: " + targetHint + "\n\n原始内容:\n" + rawContent,
			},
		},
		Temperature: 0,
		MaxTokens:   2600,
	})
	if err != nil {
		return codeBundlePayload{}, "", "", err
	}
	payload, parseMode, parseErr := parseCodeBundleResponse(response.Content)
	return payload, parseMode, response.Content, parseErr
}

func (s *Service) buildCodegenPrompt(goal string, plan Plan, complexity ComplexityReport, matches []KnowledgeMatch, screen ScreenContext, targetMode string, patchCandidates []string) string {
	var builder strings.Builder
	builder.WriteString("目标:\n")
	builder.WriteString(goal)
	builder.WriteString("\n\n")
	builder.WriteString("任务模式:\n")
	builder.WriteString(plan.Mode)
	builder.WriteString("\n\n")
	builder.WriteString("代码落盘策略:\n")
	builder.WriteString(targetMode)
	builder.WriteString("\n")
	if targetMode == "current_repo_patch" {
		builder.WriteString("要求: 当前任务是补丁模式，优先修改或补充当前仓库已有结构，不要重新生成一套全新项目骨架。\n\n")
	} else {
		builder.WriteString("要求: 当前任务是独立工作区模式，可以生成最小可运行脚手架。\n\n")
	}
	if len(patchCandidates) > 0 {
		builder.WriteString("补丁候选文件:\n")
		for _, path := range patchCandidates {
			builder.WriteString("- ")
			builder.WriteString(path)
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
		if targetMode == "current_repo_patch" {
			builder.WriteString("优先在上述候选文件中选择修改目标；如果确实需要新文件，优先放到已有目录旁边的测试、文档或辅助文件里。\n\n")
		}
	}
	builder.WriteString("执行摘要:\n")
	builder.WriteString(plan.Overview)
	builder.WriteString("\n\n")
	builder.WriteString("复杂度:\n")
	builder.WriteString(fmt.Sprintf("- level: %s\n- score: %d\n", complexity.Level, complexity.Score))
	builder.WriteString("计划动作:\n")
	for _, action := range plan.Actions {
		builder.WriteString("- ")
		builder.WriteString(action)
		builder.WriteString("\n")
	}
	builder.WriteString("交付物:\n")
	for _, item := range plan.Deliverables {
		builder.WriteString("- ")
		builder.WriteString(item)
		builder.WriteString("\n")
	}
	builder.WriteString("符号快照:\n")
	builder.WriteString(s.symbols.Preview)
	builder.WriteString("\n")
	if screen.Available {
		builder.WriteString("屏幕上下文:\n")
		builder.WriteString(fmt.Sprintf("- source: %s\n", screen.SourceLabel))
		if screen.VisionSummary != "" {
			builder.WriteString("- vision: ")
			builder.WriteString(screen.VisionSummary)
			builder.WriteString("\n")
		}
		if screen.OCRText != "" {
			builder.WriteString("- ocr: ")
			builder.WriteString(truncate(screen.OCRText, 320))
			builder.WriteString("\n")
		}
	}
	if len(matches) > 0 {
		builder.WriteString("RAG 命中:\n")
		for _, match := range matches {
			builder.WriteString("- ")
			builder.WriteString(match.Path)
			builder.WriteString(": ")
			builder.WriteString(match.Content)
			builder.WriteString("\n")
		}
	}
	if targetMode == "current_repo_patch" {
		builder.WriteString("候选文件片段:\n")
		for _, snippet := range s.loadPatchSnippets(patchCandidates, 6) {
			builder.WriteString(snippet)
			builder.WriteString("\n")
		}
	}
	builder.WriteString("请返回可直接落盘、可自检的 Go 代码包。")
	return builder.String()
}

func parseCodeBundleResponse(content string) (codeBundlePayload, string, error) {
	content = strings.TrimSpace(content)
	for _, candidate := range extractJSONCandidates(content) {
		var payload codeBundlePayload
		if err := json.Unmarshal([]byte(candidate), &payload); err == nil {
			normalized, normalizeErr := normalizeCodeBundlePayload(payload)
			if normalizeErr == nil {
				return normalized, "json", nil
			}
		}
		if payload, ok := parseFlexibleCodeBundleJSON(candidate); ok {
			normalized, normalizeErr := normalizeCodeBundlePayload(payload)
			if normalizeErr == nil {
				return normalized, "json_flexible", nil
			}
		}
	}

	if payload, ok := parseCodeBundleFromFileBlocks(content); ok {
		normalized, err := normalizeCodeBundlePayload(payload)
		if err == nil {
			return normalized, "file_blocks", nil
		}
	}
	return codeBundlePayload{}, "", fmt.Errorf("unable to parse code bundle response")
}

func parseFlexibleCodeBundleJSON(candidate string) (codeBundlePayload, bool) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(candidate), &raw); err != nil {
		return codeBundlePayload{}, false
	}

	payload := codeBundlePayload{
		Summary:     stringValue(raw["summary"]),
		EntryPoints: stringListValue(raw["entry_points"]),
		RunCommands: stringListValue(raw["run_commands"]),
		Notes:       stringListValue(raw["notes"]),
		Files:       []codeBundleFileDef{},
	}

	if files, ok := raw["files"].([]any); ok {
		for _, item := range files {
			fileMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			payload.Files = append(payload.Files, codeBundleFileDef{
				Path:     stringValue(fileMap["path"]),
				Purpose:  stringValue(fileMap["purpose"]),
				Language: stringValue(fileMap["language"]),
				Content:  stringValue(fileMap["content"]),
			})
		}
	}

	if len(payload.Files) == 0 {
		return codeBundlePayload{}, false
	}
	return payload, true
}

func normalizeCodeBundlePayload(payload codeBundlePayload) (codeBundlePayload, error) {
	payload.Summary = strings.TrimSpace(payload.Summary)
	payload.EntryPoints = compactStrings(payload.EntryPoints)
	payload.RunCommands = compactStrings(payload.RunCommands)
	payload.Notes = compactStrings(payload.Notes)
	if len(payload.Files) == 0 {
		return codeBundlePayload{}, fmt.Errorf("files cannot be empty")
	}

	files := make([]codeBundleFileDef, 0, len(payload.Files))
	for _, file := range payload.Files {
		normalizedPath, ok := sanitizeRelativePath(file.Path)
		if !ok {
			continue
		}
		content := strings.TrimSpace(file.Content)
		if content == "" {
			continue
		}
		files = append(files, codeBundleFileDef{
			Path:     normalizedPath,
			Purpose:  strings.TrimSpace(file.Purpose),
			Language: strings.TrimSpace(file.Language),
			Content:  content + "\n",
		})
	}
	if len(files) == 0 {
		return codeBundlePayload{}, fmt.Errorf("no valid files after normalization")
	}
	payload.Files = files
	return payload, nil
}

func persistCodeBundle(outputDir, manifestPath string, cleanOutput bool, runID, goal, mode, targetMode, provider, parseMode string, candidates []string, payload codeBundlePayload) ([]GeneratedFileRecord, string, error) {
	if cleanOutput {
		if err := os.RemoveAll(outputDir); err != nil {
			return nil, "", err
		}
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, "", err
	}

	files := make([]GeneratedFileRecord, 0, len(payload.Files))
	for _, file := range payload.Files {
		fullPath := filepath.Join(outputDir, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return nil, "", err
		}
		if err := os.WriteFile(fullPath, []byte(file.Content), 0o644); err != nil {
			return nil, "", err
		}
		files = append(files, GeneratedFileRecord{
			Path:     file.Path,
			Purpose:  file.Purpose,
			Language: file.Language,
			Bytes:    len([]byte(file.Content)),
		})
	}

	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		return nil, "", err
	}
	manifestPayload := persistedCodeBundle{
		RunID:      runID,
		Goal:       goal,
		Mode:       mode,
		TargetMode: targetMode,
		Candidates: candidates,
		Provider:   provider,
		ParseMode:  parseMode,
		Summary:    payload.Summary,
		Files:      payload.Files,
		EntryPoint: payload.EntryPoints,
		RunCommand: payload.RunCommands,
		Notes:      payload.Notes,
	}
	data, err := json.MarshalIndent(manifestPayload, "", "  ")
	if err != nil {
		return nil, "", err
	}
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		return nil, "", err
	}
	return files, manifestPath, nil
}

func fallbackCodeBundle(goal string, plan Plan, targetMode string) codeBundlePayload {
	moduleName := fallbackModuleName(goal)
	overview := strings.TrimSpace(plan.Overview)
	if overview == "" {
		overview = "A generated Go workspace scaffold for the requested task."
	}
	actionLines := "- 初始化最小 Go 模块\n- 保留可测试的内部服务层\n- 提供 README 与开发说明\n"
	if len(plan.Actions) > 0 {
		var builder strings.Builder
		for _, action := range plan.Actions {
			builder.WriteString("- ")
			builder.WriteString(action)
			builder.WriteString("\n")
		}
		actionLines = builder.String()
	}

	appMessage := fmt.Sprintf("任务目标：%s", goal)
	if targetMode == "current_repo_patch" {
		patchDir := "generated_patches/" + strings.TrimPrefix(moduleName, "generated/")
		return codeBundlePayload{
			Summary:     "在线代码生成不可用，已在当前仓库内生成保守的补丁说明目录，避免直接覆盖核心工程文件。",
			EntryPoints: []string{},
			RunCommands: []string{
				"go test ./...",
				"go vet ./...",
			},
			Notes: []string{
				"当前为保守兜底，不会覆盖主工程的 go.mod 或现有入口文件。",
				"如果要真正补丁当前仓库，优先保证在线模型输出可解析的文件清单。",
			},
			Files: []codeBundleFileDef{
				{
					Path:     patchDir + "/PATCH_PLAN.md",
					Purpose:  "保留当前仓库补丁任务的说明和下一步动作。",
					Language: "markdown",
					Content: fmt.Sprintf(`# Repository Patch Plan

## Goal
%s

## Overview
%s

## Planned Actions
%s

## Notes
- 当前是保守兜底，不直接覆盖主仓库核心文件。
- 建议下一步用在线模型生成精确的补丁文件列表。
`, goal, overview, actionLines),
				},
				{
					Path:     patchDir + "/NEXT_STEPS.md",
					Purpose:  "沉淀当前仓库补丁模式下的后续操作建议。",
					Language: "markdown",
					Content: fmt.Sprintf(`# Next Steps

- 检查当前仓库哪些文件需要真正修改。
- 基于 Snapshot 和 diff 结果决定是否应用更激进的 patch。
- 在通过预检后再考虑自动提交。

任务上下文：%s
`, goal),
				},
			},
		}
	}
	return codeBundlePayload{
		Summary: fmt.Sprintf("已使用兜底模板生成最小 Go 项目骨架，输出目录可直接执行 go test ./...。"),
		EntryPoints: []string{
			"cmd/app/main.go",
		},
		RunCommands: []string{
			"gofmt -w .",
			"go test ./...",
			"go vet ./...",
		},
		Notes: []string{
			"当前使用兜底模板，适合先打通代码写回、自检与 Docker 验证链路。",
			"后续可把文件生成器升级成模型驱动 patch，而不是整包脚手架。",
		},
		Files: []codeBundleFileDef{
			{
				Path:     "go.mod",
				Purpose:  "定义独立的 Go 模块，避免影响当前主仓库。",
				Language: "go",
				Content: fmt.Sprintf(`module %s

go 1.25
`, moduleName),
			},
			{
				Path:     "README.md",
				Purpose:  "解释生成代码包的目标、结构和运行方式。",
				Language: "markdown",
				Content: fmt.Sprintf(`# Generated Go Workspace

## Goal
%s

## Overview
%s

## Planned Actions
%s

## Run
~~~bash
go test ./...
go vet ./...
~~~
`, goal, overview, actionLines),
			},
			{
				Path:     "cmd/app/main.go",
				Purpose:  "提供最小可运行入口。",
				Language: "go",
				Content: fmt.Sprintf(`package main

import (
	"fmt"

	"%s/internal/app"
)

func main() {
	fmt.Println(app.BuildSummary())
}
`, moduleName),
			},
			{
				Path:     "internal/app/app.go",
				Purpose:  "承载可测试的核心摘要逻辑。",
				Language: "go",
				Content: fmt.Sprintf(`package app

const summary = %q

// BuildSummary returns the generated task summary for the workspace.
func BuildSummary() string {
	return summary
}
`, appMessage),
			},
			{
				Path:     "internal/app/app_test.go",
				Purpose:  "验证生成的核心逻辑可通过单测。",
				Language: "go",
				Content: `package app

import "testing"

func TestBuildSummary(t *testing.T) {
	if BuildSummary() == "" {
		t.Fatal("summary should not be empty")
	}
}
`,
			},
			{
				Path:     "docs/implementation_plan.md",
				Purpose:  "沉淀实现说明与后续扩展点。",
				Language: "markdown",
				Content: fmt.Sprintf(`# Implementation Notes

## Goal
%s

## Next Steps
- 将当前脚手架替换为模型驱动的局部 patch 输出。
- 将验证链从全量 go test ./... 提升为按目标包/目标文件的局部验证。
- 接入更细粒度的 diff 审计与失败重试策略。
`, goal),
			},
		},
	}
}

func fallbackModuleName(goal string) string {
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug := strings.ToLower(strings.TrimSpace(goal))
	slug = re.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "generated-agent-workspace"
	}
	if len(slug) > 32 {
		slug = slug[:32]
	}
	return "generated/" + slug
}

func sanitizeRelativePath(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	path = strings.ReplaceAll(path, "\\", "/")
	if strings.HasPrefix(path, "/") {
		return "", false
	}
	cleaned := filepath.ToSlash(filepath.Clean(path))
	if cleaned == "." || cleaned == "" {
		return "", false
	}
	if strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", false
	}
	return cleaned, true
}

func compactStrings(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func restrictCodeBundlePayload(targetMode string, candidates []string, payload codeBundlePayload) (codeBundlePayload, []string) {
	if targetMode != "current_repo_patch" {
		return payload, nil
	}

	allowedFiles := map[string]struct{}{}
	allowedDirs := map[string]struct{}{}
	for _, candidate := range candidates {
		candidate, ok := sanitizeRelativePath(candidate)
		if !ok {
			continue
		}
		allowedFiles[candidate] = struct{}{}
		dir := filepath.ToSlash(filepath.Dir(candidate))
		if dir == "" {
			dir = "."
		}
		allowedDirs[dir] = struct{}{}
	}

	filtered := make([]codeBundleFileDef, 0, len(payload.Files))
	rejected := make([]string, 0)
	for _, file := range payload.Files {
		path, ok := sanitizeRelativePath(file.Path)
		if !ok {
			rejected = append(rejected, file.Path)
			continue
		}
		if _, ok := allowedFiles[path]; ok || isAllowedAuxiliaryPatchFile(path, allowedDirs) {
			file.Path = path
			filtered = append(filtered, file)
			continue
		}
		rejected = append(rejected, path)
	}
	payload.Files = filtered
	return payload, rejected
}

func isAllowedAuxiliaryPatchFile(path string, allowedDirs map[string]struct{}) bool {
	dir := filepath.ToSlash(filepath.Dir(path))
	if dir == "" {
		dir = "."
	}
	if _, ok := allowedDirs[dir]; !ok {
		return false
	}

	ext := strings.ToLower(filepath.Ext(path))
	base := filepath.Base(path)
	if ext == ".md" || ext == ".txt" || ext == ".json" {
		return true
	}
	return strings.HasSuffix(base, "_test.go")
}

func (r CodeGenerationReport) HasGoChanges() bool {
	for _, file := range r.Files {
		if strings.EqualFold(filepath.Ext(file.Path), ".go") {
			return true
		}
	}
	return false
}

func (s *Service) buildPatchCandidates(goal string, matches []KnowledgeMatch, targetMode string) []string {
	candidates := make([]string, 0, 8)
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		fullPath := filepath.Join(s.root, filepath.FromSlash(path))
		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			return
		}
		for _, existing := range candidates {
			if existing == path {
				return
			}
		}
		candidates = append(candidates, path)
	}

	for _, match := range matches {
		if len(candidates) >= 6 {
			break
		}
		add(match.Path)
	}

	lowerGoal := strings.ToLower(goal)
	if containsAny(lowerGoal, "预检", "preflight", "lint", "vet", "编译", "测试") {
		add("preflight/health.go")
		add("companion/auto_loop.go")
		add("companion/actions_advanced.go")
		add("companion/service.go")
	}
	if containsAny(lowerGoal, "docker", "沙盒", "sandbox") {
		add("executor/docker_run.go")
		add("companion/auto_loop.go")
	}
	if containsAny(lowerGoal, "路由", "router", "模型", "provider") {
		add("companion/router.go")
		add("companion/service.go")
	}
	if containsAny(lowerGoal, "langgraph", "workflow", "编排") {
		add("companion/langgraph_bridge.go")
		add("companion/service.go")
	}
	if containsAny(lowerGoal, "snapshot", "快照", "回滚") {
		add("snapshot/manifest.go")
		add("companion/service.go")
	}
	if containsAny(lowerGoal, "rag", "检索", "知识") {
		add("rag/index.go")
		add("companion/service.go")
	}
	if containsAny(lowerGoal, "readme", "文档", "教程") {
		add("README.md")
		add("PROJECT_TEACHING_GUIDE.md")
	}
	if targetMode == "current_repo_patch" {
		add("companion/service.go")
	}
	if len(candidates) > 8 {
		return candidates[:8]
	}
	return candidates
}

func (s *Service) loadPatchSnippets(candidates []string, limit int) []string {
	if limit <= 0 {
		limit = 4
	}
	snippets := make([]string, 0, limit)
	for _, path := range candidates {
		if len(snippets) >= limit {
			break
		}
		data, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(path)))
		if err != nil {
			continue
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			continue
		}
		snippet := truncate(content, 900)
		var builder strings.Builder
		builder.WriteString("### ")
		builder.WriteString(path)
		builder.WriteString("\n")
		builder.WriteString(snippet)
		builder.WriteString("\n")
		snippets = append(snippets, builder.String())
	}
	return snippets
}

func extractJSONCandidates(content string) []string {
	candidates := []string{}
	content = strings.TrimSpace(content)
	if content == "" {
		return candidates
	}
	candidates = append(candidates, content)
	if start := strings.Index(content, "{"); start >= 0 {
		if end := strings.LastIndex(content, "}"); end > start {
			candidates = append(candidates, content[start:end+1])
		}
	}

	re := regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")
	matches := re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			candidates = append(candidates, strings.TrimSpace(match[1]))
		}
	}
	return uniqueStrings(candidates)
}

func parseCodeBundleFromFileBlocks(content string) (codeBundlePayload, bool) {
	re := regexp.MustCompile("(?ms)(?:^|\\n)(?:#{2,4}\\s*)?(?:FILE|File|Path|路径)\\s*[:：]\\s*([^\\n]+)\\n(?:Purpose|用途)\\s*[:：]?\\s*([^\\n]*)\\n?```([a-zA-Z0-9_-]*)\\n(.*?)\\n```")
	matches := re.FindAllStringSubmatch(content, -1)
	files := make([]codeBundleFileDef, 0, len(matches))
	for _, match := range matches {
		if len(match) < 5 {
			continue
		}
		path, ok := sanitizeRelativePath(match[1])
		if !ok {
			continue
		}
		files = append(files, codeBundleFileDef{
			Path:     path,
			Purpose:  strings.TrimSpace(match[2]),
			Language: strings.TrimSpace(match[3]),
			Content:  strings.TrimSpace(match[4]) + "\n",
		})
	}
	if len(files) == 0 {
		reSimple := regexp.MustCompile("(?ms)(?:^|\\n)###\\s+([^\\n]+\\.(?:go|md|yaml|yml|json|txt))\\n```([a-zA-Z0-9_-]*)\\n(.*?)\\n```")
		matches = reSimple.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) < 4 {
				continue
			}
			path, ok := sanitizeRelativePath(match[1])
			if !ok {
				continue
			}
			files = append(files, codeBundleFileDef{
				Path:     path,
				Purpose:  "从 Markdown 文件块提取",
				Language: strings.TrimSpace(match[2]),
				Content:  strings.TrimSpace(match[3]) + "\n",
			})
		}
	}
	if len(files) == 0 {
		return codeBundlePayload{}, false
	}
	summary := strings.TrimSpace(content)
	if idx := strings.Index(summary, "\n"); idx > 0 {
		summary = strings.TrimSpace(summary[:idx])
	}
	if summary == "" {
		summary = "从自由文本中提取文件块成功"
	}
	return codeBundlePayload{
		Summary:     truncate(summary, 180),
		EntryPoints: inferEntryPoints(files),
		RunCommands: []string{"go test ./...", "go vet ./..."},
		Notes:       []string{"本次结果由自由文本文件块解析获得，建议结合 raw response 检查格式稳定性。"},
		Files:       files,
	}, true
}

func inferEntryPoints(files []codeBundleFileDef) []string {
	points := []string{}
	for _, file := range files {
		if strings.HasSuffix(file.Path, "/main.go") || file.Path == "main.go" {
			points = append(points, file.Path)
		}
	}
	return points
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			text := stringValue(item)
			if text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func stringListValue(value any) []string {
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil
		}
		if strings.Contains(text, "\n") {
			lines := []string{}
			for _, line := range strings.Split(text, "\n") {
				line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
				if line != "" {
					lines = append(lines, line)
				}
			}
			if len(lines) > 0 {
				return lines
			}
		}
		return []string{text}
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			text := stringValue(item)
			if text != "" {
				result = append(result, text)
			}
		}
		return result
	case []string:
		return compactStrings(typed)
	default:
		text := stringValue(typed)
		if text == "" {
			return nil
		}
		return []string{text}
	}
}

func writeCodegenRawResponse(dataDir, runID, providerName, stage, content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	dir := filepath.Join(dataDir, "patches", runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	filename := fmt.Sprintf("%s_%s_raw.txt", providerName, stage)
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return ""
	}
	return path
}
