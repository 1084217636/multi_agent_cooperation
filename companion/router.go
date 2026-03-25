package companion

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ComplexityLevel 表示任务复杂度等级。
type ComplexityLevel string

const (
	ComplexityLow    ComplexityLevel = "low"
	ComplexityMedium ComplexityLevel = "medium"
	ComplexityHigh   ComplexityLevel = "high"
)

// ComplexityReport 描述任务复杂度评估结果。
type ComplexityReport struct {
	Score   int             `json:"score"`
	Level   ComplexityLevel `json:"level"`
	Reasons []string        `json:"reasons"`
}

// ProviderStatus 描述某个 Provider 的可用性。
type ProviderStatus struct {
	Name         string   `json:"name"`
	Ready        bool     `json:"ready"`
	Reason       string   `json:"reason"`
	DefaultModel string   `json:"default_model"`
	Models       []string `json:"models"`
}

// RouteDecision 描述阶梯路由的结果。
type RouteDecision struct {
	Primary  string   `json:"primary"`
	Attempts []string `json:"attempts"`
	Reason   string   `json:"reason"`
}

// AssessComplexity 根据目标描述进行轻量复杂度评分。
func AssessComplexity(goal string) ComplexityReport {
	score := 1
	reasons := []string{"默认基础复杂度"}
	trimmed := strings.TrimSpace(goal)
	lower := strings.ToLower(trimmed)

	if len([]rune(trimmed)) > 80 {
		score++
		reasons = append(reasons, "需求描述较长")
	}
	if len([]rune(trimmed)) > 180 {
		score++
		reasons = append(reasons, "包含较多约束与上下文")
	}

	for keyword, reason := range map[string]string{
		"rag":       "涉及 RAG 检索设计",
		"mcp":       "涉及 MCP / 符号暴露",
		"docker":    "涉及沙箱执行",
		"redis":     "涉及状态与缓存设计",
		"langgraph": "涉及工作流编排",
		"桌宠":        "涉及桌面入口或角色化交互",
		"桌面":        "涉及本地工作台与运行态",
		"重构":        "涉及工程级重构",
		"架构":        "涉及系统架构设计",
		"多智能体":      "涉及多角色协作编排",
		"自动执行":      "涉及任务自动化闭环",
		"代码生成":      "涉及代码生成与验证",
		"编译":        "涉及工程可构建性",
		"llm":       "涉及模型编排与接入",
		".go":       "显式提到了代码文件",
		"cmd/":      "显式提到了项目路径",
		"agent/":    "显式提到了项目路径",
		"executor/": "显式提到了项目路径",
	} {
		if strings.Contains(lower, keyword) {
			score++
			reasons = append(reasons, reason)
		}
	}

	level := ComplexityLow
	switch {
	case score >= 7:
		level = ComplexityHigh
	case score >= 4:
		level = ComplexityMedium
	}

	return ComplexityReport{
		Score:   score,
		Level:   level,
		Reasons: reasons,
	}
}

// DecideRoute 根据复杂度和 Provider 状态生成阶梯式路由。
func DecideRoute(goal string, complexity ComplexityReport, statuses map[string]ProviderStatus) RouteDecision {
	preferred := preferredProviders(goal, complexity.Level)
	attempts := make([]string, 0, len(preferred))

	for _, name := range preferred {
		status, ok := statuses[name]
		if !ok || !status.Ready {
			continue
		}
		attempts = append(attempts, name)
	}

	if len(attempts) == 0 {
		attempts = []string{"mock"}
	}

	reason := fmt.Sprintf("复杂度=%s，优先按 %s 路由；低阶模型超时或失败后自动升级。", complexity.Level, strings.Join(attempts, " -> "))
	if complexity.Level == ComplexityHigh && attempts[0] == "mock" {
		reason = "任务复杂度较高，但当前没有可用在线模型，已回退到离线 mock 演示链路。"
	}
	if strings.Contains(strings.ToLower(goal), filepath.Base(goal)) {
		reason += " 请求中包含工程上下文，适合结合符号快照与知识检索。"
	}

	return RouteDecision{
		Primary:  attempts[0],
		Attempts: attempts,
		Reason:   reason,
	}
}

func preferredProviders(goal string, level ComplexityLevel) []string {
	if wantsOfflineRoute(goal) {
		switch level {
		case ComplexityHigh:
			return []string{"mock", "groq", "openai", "siliconflow", "ollama"}
		case ComplexityMedium:
			return []string{"mock", "groq", "ollama", "siliconflow", "openai"}
		default:
			return []string{"mock", "groq", "ollama", "siliconflow", "openai"}
		}
	}

	switch level {
	case ComplexityHigh:
		return []string{"openai", "groq", "siliconflow", "ollama", "mock"}
	case ComplexityMedium:
		return []string{"groq", "ollama", "siliconflow", "openai", "mock"}
	default:
		return []string{"groq", "ollama", "siliconflow", "openai", "mock"}
	}
}

func wantsOfflineRoute(goal string) bool {
	lower := strings.ToLower(strings.TrimSpace(goal))
	return strings.Contains(lower, "mock") ||
		strings.Contains(lower, "离线") ||
		strings.Contains(lower, "演示") ||
		strings.Contains(lower, "无 api") ||
		strings.Contains(lower, "没额度")
}
