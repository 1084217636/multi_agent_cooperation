package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"multi_agent_cooperation/domain"
	"multi_agent_cooperation/llm"
)

// CriticAgent 批评者智能体
type CriticAgent struct {
	Agent    domain.Agent
	Provider llm.Provider
}

// NewCriticAgent 创建批评者智能体
func NewCriticAgent(provider llm.Provider) *CriticAgent {
	return &CriticAgent{
		Agent: domain.Agent{
			ID:   "critic-001",
			Name: "批评者",
			Type: domain.AgentTypeCritic,
			SystemPrompt: `你是一位严谨的批评专家，擅长发现方案中的问题和漏洞，提出建设性的改进建议。

你的职责：
1. 审查设计方案和代码实现
2. 识别潜在的风险和问题
3. 提出具体的改进建议
4. 挑战假设和设计决策
5. 确保方案的质量和可行性

批评原则：
- 建设性批评，而非否定
- 基于事实和经验
- 提供具体的改进建议
- 考虑不同的视角
- 保持客观和专业

审查维度：
- 功能完整性
- 技术可行性
- 性能和扩展性
- 安全性和可靠性
- 可维护性
- 用户体验

输出要求：
- 明确指出问题所在
- 说明问题的严重程度
- 提供具体的改进建议
- 说明改进的预期效果
- 给出优先级建议`,
			ModelName: "llama-3.3-70b-versatile",
			Priority:  4,
			Active:    true,
		},
		Provider: provider,
	}
}

// CritiqueResult 批评结果
type CritiqueResult struct {
	Target       string       `json:"target"`        // 审查目标
	Type         string       `json:"type"`          // 审查类型
	Issues       []Issue      `json:"issues"`        // 问题列表
	Suggestions  []Suggestion `json:"suggestions"`   // 建议列表
	OverallScore int          `json:"overall_score"` // 总体评分
	Summary      string       `json:"summary"`       // 总结
}

// Issue 问题定义
type Issue struct {
	ID          string `json:"id"`          // 问题ID
	Severity    string `json:"severity"`    // 严重程度 (critical/high/medium/low)
	Category    string `json:"category"`    // 问题类别
	Description string `json:"description"` // 问题描述
	Location    string `json:"location"`    // 问题位置
	Impact      string `json:"impact"`      // 影响范围
}

// Suggestion 建议定义
type Suggestion struct {
	ID                   string `json:"id"`                    // 建议ID
	Priority             string `json:"priority"`              // 优先级 (high/medium/low)
	Category             string `json:"category"`              // 建议类别
	Description          string `json:"description"`           // 建议描述
	ExpectedBenefit      string `json:"expected_benefit"`      // 预期收益
	ImplementationEffort string `json:"implementation_effort"` // 实施难度
}

// CritiqueArchitecture 批评架构设计
func (ca *CriticAgent) CritiqueArchitecture(architecture string) (*CritiqueResult, error) {
	// 构建请求
	req := llm.CompletionRequest{
		Model: ca.Agent.ModelName,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: ca.Agent.SystemPrompt,
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("请批评以下架构设计，指出问题和改进建议。请以JSON格式返回结果，格式如下：\n\n{\n  \"target\": \"Architecture Design\",\n  \"type\": \"architecture\",\n  \"issues\": [\n    {\n      \"id\": \"issue-1\",\n      \"severity\": \"high\",\n      \"category\": \"性能\",\n      \"description\": \"问题描述\",\n      \"location\": \"位置\",\n      \"impact\": \"影响范围\"\n    }\n  ],\n  \"suggestions\": [\n    {\n      \"id\": \"suggestion-1\",\n      \"priority\": \"high\",\n      \"category\": \"性能优化\",\n      \"description\": \"建议描述\",\n      \"expected_benefit\": \"预期收益\",\n      \"implementation_effort\": \"实施难度\"\n    }\n  ],\n  \"overall_score\": 85,\n  \"summary\": \"总结\"\n}\n\n架构设计：\n\n%s", architecture),
			},
		},
		Temperature: 0.7,
		MaxTokens:   4000,
	}

	// 调用LLM API
	resp, rateLimit, err := ca.Provider.Complete(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to critique architecture: %w", err)
	}

	// 打印速率限制信息
	fmt.Printf("\n[%s] 速率限制信息:\n", ca.Agent.Name)
	fmt.Printf("  请求限制: %d 次/分钟\n", rateLimit.RequestsLimit)
	fmt.Printf("  剩余请求: %d 次\n", rateLimit.RequestsRemaining)
	fmt.Printf("  Token 限制: %d tokens/分钟\n", rateLimit.TokensLimit)
	fmt.Printf("  剩余 Token: %d tokens\n", rateLimit.TokensRemaining)

	// 尝试解析JSON响应
	var result CritiqueResult
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		// 如果JSON解析失败，返回基础结果
		result = CritiqueResult{
			Target:       "Architecture Design",
			Type:         "architecture",
			Issues:       []Issue{},
			Suggestions:  []Suggestion{},
			OverallScore: 0,
			Summary:      resp.Content,
		}
	}

	return &result, nil
}

// CritiqueCode 批评代码实现
func (ca *CriticAgent) CritiqueCode(code string, task domain.Task) (*CritiqueResult, error) {
	// 构建请求
	req := llm.CompletionRequest{
		Model: ca.Agent.ModelName,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: ca.Agent.SystemPrompt,
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("请批评以下代码实现，任务：%s。请以JSON格式返回结果，格式如下：\n\n{\n  \"target\": \"Code for task\",\n  \"type\": \"code\",\n  \"issues\": [\n    {\n      \"id\": \"issue-1\",\n      \"severity\": \"high\",\n      \"category\": \"性能\",\n      \"description\": \"问题描述\",\n      \"location\": \"位置\",\n      \"impact\": \"影响范围\"\n    }\n  ],\n  \"suggestions\": [\n    {\n      \"id\": \"suggestion-1\",\n      \"priority\": \"high\",\n      \"category\": \"性能优化\",\n      \"description\": \"建议描述\",\n      \"expected_benefit\": \"预期收益\",\n      \"implementation_effort\": \"实施难度\"\n    }\n  ],\n  \"overall_score\": 85,\n  \"summary\": \"总结\"\n}\n\n代码：\n%s", task.Title, code),
			},
		},
		Temperature: 0.7,
		MaxTokens:   4000,
	}

	// 调用LLM API
	resp, rateLimit, err := ca.Provider.Complete(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to critique code: %w", err)
	}

	// 打印速率限制信息
	fmt.Printf("\n[%s] 速率限制信息:\n", ca.Agent.Name)
	fmt.Printf("  请求限制: %d 次/分钟\n", rateLimit.RequestsLimit)
	fmt.Printf("  剩余请求: %d 次\n", rateLimit.RequestsRemaining)
	fmt.Printf("  Token 限制: %d tokens/分钟\n", rateLimit.TokensLimit)
	fmt.Printf("  剩余 Token: %d tokens\n", rateLimit.TokensRemaining)

	// 尝试解析JSON响应
	var result CritiqueResult
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		// 如果JSON解析失败，返回基础结果
		result = CritiqueResult{
			Target:       fmt.Sprintf("Code for task: %s", task.Title),
			Type:         "code",
			Issues:       []Issue{},
			Suggestions:  []Suggestion{},
			OverallScore: 0,
			Summary:      resp.Content,
		}
	}

	return &result, nil
}

// CritiqueTaskSplit 批评任务拆分
func (ca *CriticAgent) CritiqueTaskSplit(taskSplit string) (*CritiqueResult, error) {
	// 构建请求
	req := llm.CompletionRequest{
		Model: ca.Agent.ModelName,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: ca.Agent.SystemPrompt,
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("请批评以下任务拆分方案，指出问题和改进建议。请以JSON格式返回结果，格式如下：\n\n{\n  \"target\": \"Task Split\",\n  \"type\": \"task_split\",\n  \"issues\": [\n    {\n      \"id\": \"issue-1\",\n      \"severity\": \"high\",\n      \"category\": \"性能\",\n      \"description\": \"问题描述\",\n      \"location\": \"位置\",\n      \"impact\": \"影响范围\"\n    }\n  ],\n  \"suggestions\": [\n    {\n      \"id\": \"suggestion-1\",\n      \"priority\": \"high\",\n      \"category\": \"性能优化\",\n      \"description\": \"建议描述\",\n      \"expected_benefit\": \"预期收益\",\n      \"implementation_effort\": \"实施难度\"\n    }\n  ],\n  \"overall_score\": 85,\n  \"summary\": \"总结\"\n}\n\n任务拆分方案：\n\n%s", taskSplit),
			},
		},
		Temperature: 0.7,
		MaxTokens:   4000,
	}

	// 调用LLM API
	resp, rateLimit, err := ca.Provider.Complete(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to critique task split: %w", err)
	}

	// 打印速率限制信息
	fmt.Printf("\n[%s] 速率限制信息:\n", ca.Agent.Name)
	fmt.Printf("  请求限制: %d 次/分钟\n", rateLimit.RequestsLimit)
	fmt.Printf("  剩余请求: %d 次\n", rateLimit.RequestsRemaining)
	fmt.Printf("  Token 限制: %d tokens/分钟\n", rateLimit.TokensLimit)
	fmt.Printf("  剩余 Token: %d tokens\n", rateLimit.TokensRemaining)

	// 尝试解析JSON响应
	var result CritiqueResult
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		// 如果JSON解析失败，返回基础结果
		result = CritiqueResult{
			Target:       "Task Split",
			Type:         "task_split",
			Issues:       []Issue{},
			Suggestions:  []Suggestion{},
			OverallScore: 0,
			Summary:      resp.Content,
		}
	}

	return &result, nil
}

// GenerateCritiqueReport 生成批评报告
func (ca *CriticAgent) GenerateCritiqueReport(result *CritiqueResult) string {
	report := fmt.Sprintf("## 批评报告\n\n")
	report += fmt.Sprintf("### 审查目标\n%s\n\n", result.Target)
	report += fmt.Sprintf("### 审查类型\n%s\n\n", result.Type)
	report += fmt.Sprintf("### 总体评分\n%d/100\n\n", result.OverallScore)

	report += fmt.Sprintf("### 发现的问题\n")
	if len(result.Issues) == 0 {
		report += "未发现问题\n\n"
	} else {
		for i, issue := range result.Issues {
			report += fmt.Sprintf("%d. **%s** (严重程度: %s)\n", i+1, issue.Category, issue.Severity)
			report += fmt.Sprintf("   - 描述: %s\n", issue.Description)
			report += fmt.Sprintf("   - 位置: %s\n", issue.Location)
			report += fmt.Sprintf("   - 影响: %s\n\n", issue.Impact)
		}
	}

	report += fmt.Sprintf("### 改进建议\n")
	if len(result.Suggestions) == 0 {
		report += "暂无建议\n\n"
	} else {
		for i, suggestion := range result.Suggestions {
			report += fmt.Sprintf("%d. **%s** (优先级: %s)\n", i+1, suggestion.Category, suggestion.Priority)
			report += fmt.Sprintf("   - 描述: %s\n", suggestion.Description)
			report += fmt.Sprintf("   - 预期收益: %s\n", suggestion.ExpectedBenefit)
			report += fmt.Sprintf("   - 实施难度: %s\n\n", suggestion.ImplementationEffort)
		}
	}

	report += fmt.Sprintf("### 总结\n%s\n", result.Summary)

	return report
}

// ValidateCritique 验证批评结果
func (ca *CriticAgent) ValidateCritique(result *CritiqueResult) bool {
	// 检查是否有总结
	if result.Summary == "" {
		return false
	}

	// 检查评分是否在合理范围内
	if result.OverallScore < 0 || result.OverallScore > 100 {
		return false
	}

	return true
}
