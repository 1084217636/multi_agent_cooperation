package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"multi_agent_cooperation/llm"
)

const (
	mockProviderName   = "mock"
	mockDefaultModel   = "companion-mock-v1"
	mockOverviewKey    = "\"overview\""
	mockActionsKey     = "\"actions\""
	mockDeliverableKey = "\"deliverables\""
	mockDesktopGapKey  = "\"desktop_pet_gaps\""
)

func init() {
	llm.Register(mockProviderName, NewMockProvider)
}

// MockProvider 在没有真实 API 配额时提供稳定的离线回退。
type MockProvider struct {
	models       []string
	defaultModel string
}

// NewMockProvider 创建离线 Mock Provider。
func NewMockProvider(config llm.ProviderConfig) (llm.Provider, error) {
	models := config.Models
	if len(models) == 0 {
		models = []string{mockDefaultModel}
	}

	defaultModel := config.DefaultModel
	if defaultModel == "" {
		defaultModel = models[0]
	}

	return &MockProvider{
		models:       models,
		defaultModel: defaultModel,
	}, nil
}

// Complete 返回结构化、可预测的离线结果。
func (m *MockProvider) Complete(_ context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, *llm.RateLimitInfo, error) {
	start := time.Now()
	prompt := joinPrompt(req.Messages)
	content := buildMockContent(prompt)

	return &llm.CompletionResponse{
			Content:          content,
			PromptTokens:     len(prompt) / 4,
			CompletionTokens: len(content) / 4,
			TotalTokens:      (len(prompt) + len(content)) / 4,
			Latency:          time.Since(start),
		}, &llm.RateLimitInfo{
			RequestsLimit:     999999,
			RequestsRemaining: 999998,
			TokensLimit:       999999,
			TokensRemaining:   999998,
			ResetTime:         time.Now().Add(24 * time.Hour),
		}, nil
}

func (m *MockProvider) GetModelList() []string {
	return m.models
}

func (m *MockProvider) GetProviderName() string {
	return mockProviderName
}

func (m *MockProvider) HealthCheck(_ context.Context) error {
	return nil
}

func joinPrompt(messages []llm.Message) string {
	var builder strings.Builder
	for _, msg := range messages {
		builder.WriteString(msg.Role)
		builder.WriteString(": ")
		builder.WriteString(msg.Content)
		builder.WriteString("\n")
	}
	return builder.String()
}

func buildMockContent(prompt string) string {
	lower := strings.ToLower(prompt)
	if strings.Contains(lower, "json") && strings.Contains(lower, mockOverviewKey) &&
		strings.Contains(lower, mockActionsKey) && strings.Contains(lower, mockDesktopGapKey) &&
		strings.Contains(lower, mockDeliverableKey) {
		return buildCompanionPlanJSON(prompt)
	}

	if strings.Contains(lower, "json") && strings.Contains(lower, "subtasks") {
		return `{"main_task":"离线模拟任务","subtasks":[{"id":"task-1","title":"整理需求","description":"梳理目标与约束","priority":1,"dependencies":[],"estimated_time":"30m"},{"id":"task-2","title":"输出方案","description":"生成初步实现方案","priority":2,"dependencies":["task-1"],"estimated_time":"45m"}]}`
	}

	if strings.Contains(lower, "json") && strings.Contains(lower, "components") {
		return `{"overview":"采用 Companion Orchestrator + RAG Index + MCP Symbol Snapshot 的三层架构。","components":[{"name":"Companion Orchestrator","role":"负责任务接入、路由和报告输出","dependencies":["RAG Index","Provider Router"],"interfaces":["RunTask","GetState"]},{"name":"RAG Index","role":"为桌面伴侣提供项目文档与代码知识检索","dependencies":[],"interfaces":["Build","Search"]}],"interfaces":[{"name":"TaskService","description":"负责任务执行","methods":[{"name":"RunTask","input":"TaskRequest","output":"TaskReport","description":"执行一次伴侣任务"}]}],"data_flow":"用户请求进入 Orchestrator，检索 RAG 上下文后交给 Provider，再输出执行报告。","tech_stack":["Go","HTTP","RAG","MCP-style AST inspection"],"scalability":"Provider 与检索层可独立扩展。","security":"执行前保留人工确认与本地沙箱。","performance":"优先使用轻量级离线检索和阶梯路由。"}`
	}

	return "离线 Mock Provider 已响应。当前未配置真实大模型 API，因此返回的是可用于演示和联调的结构化模拟结果。"
}

func buildCompanionPlanJSON(prompt string) string {
	goal := extractGoal(prompt)
	mode := inferMockGoalMode(goal)
	screenHint := extractPromptCue(prompt, "视觉摘要:")
	if screenHint == "" {
		screenHint = extractPromptCue(prompt, "OCR 文本:")
	}
	if screenHint == "" {
		screenHint = extractPromptCue(prompt, "应用猜测:")
	}
	overview := fmt.Sprintf("围绕“%s”，建议将项目收敛为常驻桌面的 AI 开发伴侣：前台负责任务接入与结果呈现，后台串联复杂度路由、RAG 检索、符号快照和工程预检。", goal)
	actions := []string{
		"把项目入口改为桌面工作台，而不是单纯群聊 demo。",
		"对接轻量 RAG，把 README、代码与设计文档作为长期上下文。",
		"在调用 LLM 前先注入 AST 符号快照，减少虚构函数和结构体。",
		"增加 go test / go vet / goimports 检查，让自动化执行形成闭环。",
	}
	deliverables := []string{
		"交付文档.md",
		"结构化计划.json",
	}
	progressSignals := []string{
		"context: 已收集页面与知识上下文",
		"planning: 已生成方案与交付物",
		"persisting: 已导出 Markdown / JSON 结果",
	}

	switch mode {
	case "analysis":
		overview = fmt.Sprintf("围绕“%s”，当前更适合先根据浏览器页面内容输出项目分析与结构说明，而不是直接上升为平台改造。", goal)
		if screenHint != "" {
			overview += " 当前页面线索显示：" + screenHint
		}
		actions = []string{
			"根据当前页面识别项目定位、功能模块和使用路径。",
			"整理一份简洁的项目结构分析文档。",
			"给出轻量优化建议和可继续追问的方向。",
		}
		deliverables = []string{
			"页面项目分析.md",
			"项目结构梳理.md",
		}
	case "document":
		overview = fmt.Sprintf("围绕“%s”，当前任务更适合输出可导出的文档，而不是直接进入复杂开发闭环。", goal)
		if screenHint != "" {
			overview += " 当前页面线索显示：" + screenHint
		}
		actions = []string{
			"整理当前页面与问题中的关键信息。",
			"输出一份可导出的 Markdown 文档。",
			"补充后续实施建议和必要输入。",
		}
		deliverables = []string{
			"任务说明文档.md",
			"执行清单.md",
		}
	case "scaffold":
		overview = fmt.Sprintf("围绕“%s”，当前更像轻量项目生成任务，应先给出项目结构、最小代码骨架和配套文档。", goal)
		actions = []string{
			"给出最小可运行项目结构和模块划分。",
			"列出需要生成的代码文件和说明文档。",
			"为后续闭环开发保留测试、自检和回滚入口。",
		}
		deliverables = []string{
			"项目结构草案.md",
			"README.md",
			"tasks.md",
		}
	case "closed_loop":
		overview = fmt.Sprintf("围绕“%s”，当前属于复杂闭环开发任务，需要阶段化执行、实时进度反馈和代码/文档双交付。", goal)
		actions = []string{
			"先拆解阶段目标、交付物和风险控制点。",
			"在闭环中持续输出代码、文档和运行状态。",
			"保留失败后的自检、回滚和再次执行入口。",
		}
		deliverables = []string{
			"项目交付包.md",
			"README.md",
			"开发计划.md",
			"代码骨架清单.md",
		}
		progressSignals = []string{
			"context: 已抓到页面、OCR 与 RAG 证据",
			"planning: 已产出阶段计划和交付清单",
			"preflight: 已完成工程预检",
			"persisting: 已写入报告与导出文件",
		}
	}

	response := map[string]any{
		"mode":             mode,
		"overview":         overview,
		"actions":          actions,
		"deliverables":     deliverables,
		"progress_signals": progressSignals,
		"risks": []string{
			"没有真实 API 时只能走离线演示，结果更偏规划而非强生成。",
			"桌宠如果要做到悬浮、动画、语音，需要额外的桌面 GUI 能力。",
			"执行能力越强，越需要显式审批和沙箱边界。",
		},
		"innovations": []string{
			"把代码符号表和文档检索一起作为上下文，形成 RAG + MCP 双保险。",
			"根据任务复杂度自动升阶模型，平衡成本与成功率。",
			"让桌面端既是入口，也是项目健康看板与教学界面。",
		},
		"desktop_pet_gaps": []string{
			"缺少常驻悬浮窗和角色动画系统。",
			"缺少语音输入输出与情绪状态表达。",
			"缺少系统级提醒、日程联动和主动唤醒机制。",
			"缺少可审计的动作审批面板。",
		},
		"rag_use_cases": []string{
			"为代码生成提供项目历史约束、接口文档和已有模块语义。",
			"在重构时召回旧方案、设计记录和错误复盘，避免重复踩坑。",
			"为桌面伴侣积累长期记忆，如偏好、规范、近期任务和常见问题。",
		},
		"next_steps": []string{
			"先用离线 Mock 模式打通桌面入口、报告输出和知识检索。",
			"再接入 Groq / OpenAI / Ollama 等真实 Provider。",
			"最后补桌宠形态的 GUI、语音和主动工作流。",
		},
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return string(data)
}

func inferMockGoalMode(goal string) string {
	lower := strings.ToLower(strings.TrimSpace(goal))
	switch {
	case strings.Contains(lower, "文档") || strings.Contains(lower, "markdown") || strings.Contains(lower, "导出"):
		return "document"
	case (strings.Contains(lower, "分析") || strings.Contains(lower, "结构") || strings.Contains(lower, "页面")) &&
		!strings.Contains(lower, "开发") && !strings.Contains(lower, "实现") && !strings.Contains(lower, "闭环"):
		return "analysis"
	case strings.Contains(lower, "闭环") || strings.Contains(lower, "持续运行") || strings.Contains(lower, "自动修复"):
		return "closed_loop"
	case strings.Contains(lower, "生成项目") || strings.Contains(lower, "脚手架") || strings.Contains(lower, "生成代码") || strings.Contains(lower, "创建文件"):
		return "scaffold"
	default:
		return "general"
	}
}

func extractPromptCue(prompt, key string) string {
	lines := strings.Split(prompt, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, key) {
			parts := strings.SplitN(trimmed, key, 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func extractGoal(prompt string) string {
	lines := strings.Split(prompt, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "目标:") || strings.HasPrefix(trimmed, "Goal:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	if len(lines) == 0 {
		return "当前项目改造"
	}

	last := strings.TrimSpace(lines[len(lines)-1])
	if last == "" {
		return "当前项目改造"
	}
	return last
}
