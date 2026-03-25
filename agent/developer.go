package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"multi_agent_cooperation/domain"
	"multi_agent_cooperation/llm"
)

// DeveloperAgent 开发者智能体
type DeveloperAgent struct {
	Agent    domain.Agent
	Provider llm.Provider
}

// NewDeveloperAgent 创建开发者智能体
func NewDeveloperAgent(name, modelName string, provider llm.Provider) *DeveloperAgent {
	return &DeveloperAgent{
		Agent: domain.Agent{
			ID:   domain.GenerateID(),
			Name: name,
			Type: domain.AgentTypeDeveloper,
			SystemPrompt: "你是一个专业的开发工程师。你的职责是：\n" +
				"1. 理解需求和设计文档\n" +
				"2. 编写清晰、可维护的代码\n" +
				"3. 遵循编码规范和最佳实践\n" +
				"4. 编写单元测试和集成测试\n" +
				"5. 优化代码性能和资源使用\n\n" +
				"编码原则：\n" +
				"- 代码可读性优先\n" +
				"- 遵循SOLID原则\n" +
				"- 错误处理完善\n" +
				"- 代码注释清晰\n" +
				"- 模块化设计",
			ModelName: modelName,
			Priority:  3,
			Active:    true,
		},
		Provider: provider,
	}
}

// CodeImplementation 代码实现
type CodeImplementation struct {
	FileName     string   `json:"file_name"`
	Language     string   `json:"language"`
	Code         string   `json:"code"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`
	Tests        string   `json:"tests"`
	Usage        string   `json:"usage"`
}

// ImplementTask 实现任务
func (da *DeveloperAgent) ImplementTask(task domain.Task, architecture string) (*CodeImplementation, error) {
	// 构建请求
	req := llm.CompletionRequest{
		Model: da.Agent.ModelName,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: da.Agent.SystemPrompt,
			},
			{
				Role: "user",
				Content: fmt.Sprintf("请实现以下任务：\n\n任务标题：%s\n任务描述：%s\n\n架构参考：\n%s\n\n请以JSON格式返回代码实现。",
					task.Title, task.Description, architecture),
			},
		},
		Temperature: 0.7,
		MaxTokens:   4000,
	}

	// 调用LLM API
	resp, rateLimit, err := da.Provider.Complete(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to get code implementation: %w", err)
	}

	// 打印速率限制信息
	fmt.Printf("\n[%s] 速率限制信息:\n", da.Agent.Name)
	fmt.Printf("  请求限制: %d 次/分钟\n", rateLimit.RequestsLimit)
	fmt.Printf("  剩余请求: %d 次\n", rateLimit.RequestsRemaining)
	fmt.Printf("  Token 限制: %d tokens/分钟\n", rateLimit.TokensLimit)
	fmt.Printf("  剩余 Token: %d tokens\n", rateLimit.TokensRemaining)

	// 解析JSON响应
	var implementation CodeImplementation
	if err := json.Unmarshal([]byte(resp.Content), &implementation); err != nil {
		// 如果解析失败，返回一个基础的实现
		implementation = CodeImplementation{
			FileName:     fmt.Sprintf("%s.go", task.ID),
			Language:     "Go",
			Code:         resp.Content,
			Description:  fmt.Sprintf("实现任务: %s", task.Title),
			Dependencies: []string{},
			Tests:        "// TODO: 添加测试",
			Usage:        "// TODO: 添加使用说明",
		}
	}

	return &implementation, nil
}

// GenerateCodeReport 生成代码报告
func (da *DeveloperAgent) GenerateCodeReport(implementation *CodeImplementation) string {
	report := fmt.Sprintf("## 代码实现报告\n\n")
	report += fmt.Sprintf("### 文件信息\n")
	report += fmt.Sprintf("- 文件名: %s\n", implementation.FileName)
	report += fmt.Sprintf("- 语言: %s\n", implementation.Language)
	report += fmt.Sprintf("- 描述: %s\n\n", implementation.Description)

	report += fmt.Sprintf("### 依赖项\n")
	for _, dep := range implementation.Dependencies {
		report += fmt.Sprintf("- %s\n", dep)
	}
	report += "\n"

	report += fmt.Sprintf("### 代码实现\n```%s\n%s\n```\n\n", implementation.Language, implementation.Code)

	report += fmt.Sprintf("### 测试代码\n```go\n%s\n```\n\n", implementation.Tests)

	report += fmt.Sprintf("### 使用说明\n%s\n", implementation.Usage)

	return report
}

// OptimizeCode 优化代码
func (da *DeveloperAgent) OptimizeCode(originalCode string, performanceIssues []string) (string, error) {
	// 构建性能问题摘要
	issuesSummary := ""
	for i, issue := range performanceIssues {
		issuesSummary += fmt.Sprintf("%d. %s\n", i+1, issue)
	}

	// 构建请求
	req := llm.CompletionRequest{
		Model: da.Agent.ModelName,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: da.Agent.SystemPrompt,
			},
			{
				Role: "user",
				Content: fmt.Sprintf("请优化以下代码，解决以下性能问题：\n\n性能问题：\n%s\n\n原始代码：\n%s",
					issuesSummary, originalCode),
			},
		},
		Temperature: 0.7,
		MaxTokens:   4000,
	}

	// 调用LLM API
	resp, rateLimit, err := da.Provider.Complete(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("failed to optimize code: %w", err)
	}

	// 打印速率限制信息
	fmt.Printf("\n[%s] 速率限制信息:\n", da.Agent.Name)
	fmt.Printf("  请求限制: %d 次/分钟\n", rateLimit.RequestsLimit)
	fmt.Printf("  剩余请求: %d 次\n", rateLimit.RequestsRemaining)
	fmt.Printf("  Token 限制: %d tokens/分钟\n", rateLimit.TokensLimit)
	fmt.Printf("  剩余 Token: %d tokens\n", rateLimit.TokensRemaining)

	return resp.Content, nil
}

// ReviewCode 审查代码
func (da *DeveloperAgent) ReviewCode(code string) ([]string, error) {
	// 构建请求
	req := llm.CompletionRequest{
		Model: da.Agent.ModelName,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: da.Agent.SystemPrompt,
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("请审查以下代码，指出潜在问题和改进建议：\n\n%s", code),
			},
		},
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	// 调用LLM API
	resp, rateLimit, err := da.Provider.Complete(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to review code: %w", err)
	}

	// 打印速率限制信息
	fmt.Printf("\n[%s] 速率限制信息:\n", da.Agent.Name)
	fmt.Printf("  请求限制: %d 次/分钟\n", rateLimit.RequestsLimit)
	fmt.Printf("  剩余请求: %d 次\n", rateLimit.RequestsRemaining)
	fmt.Printf("  Token 限制: %d tokens/分钟\n", rateLimit.TokensLimit)
	fmt.Printf("  剩余 Token: %d tokens\n", rateLimit.TokensRemaining)

	// 这里简化处理，返回LLM的响应
	return []string{resp.Content}, nil
}
