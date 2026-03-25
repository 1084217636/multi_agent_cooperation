package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"multi_agent_cooperation/domain"
	"multi_agent_cooperation/llm"
)

// ArchitectAgent 架构师智能体
type ArchitectAgent struct {
	Agent    domain.Agent
	Provider llm.Provider
}

// NewArchitectAgent 创建架构师智能体
func NewArchitectAgent(provider llm.Provider) *ArchitectAgent {
	return &ArchitectAgent{
		Agent: domain.Agent{
			ID:   domain.GenerateID(),
			Name: "架构师",
			Type: domain.AgentTypeArchitect,
			SystemPrompt: `你是一位资深的软件架构师，擅长设计稳定、可扩展的项目架构。

你的职责：
1. 分析项目需求和技术约束
2. 设计合理的系统架构
3. 确定技术栈和组件选择
4. 定义模块之间的接口和交互
5. 考虑性能、安全性和可维护性

架构设计原则：
- 高内聚低耦合
- 单一职责原则
- 开闭原则
- 依赖倒置原则
- 接口隔离原则

输出要求：
- 提供清晰的架构图描述
- 说明各模块的职责
- 定义关键接口
- 说明数据流向
- 考虑扩展性和性能优化`,
			ModelName: "llama-3.3-70b-versatile",
			Priority:  2,
			Active:    true,
		},
		Provider: provider,
	}
}

// ArchitectureDesign 架构设计方案
type ArchitectureDesign struct {
	Overview    string         `json:"overview"`    // 架构概述
	Components  []Component    `json:"components"`  // 组件列表
	Interfaces  []InterfaceDef `json:"interfaces"`  // 接口定义
	DataFlow    string         `json:"data_flow"`   // 数据流向
	TechStack   []string       `json:"tech_stack"`  // 技术栈
	Scalability string         `json:"scalability"` // 扩展性考虑
	Security    string         `json:"security"`    // 安全考虑
	Performance string         `json:"performance"` // 性能考虑
}

// Component 组件定义
type Component struct {
	Name         string   `json:"name"`         // 组件名称
	Role         string   `json:"role"`         // 组件职责
	Dependencies []string `json:"dependencies"` // 依赖的组件
	Interfaces   []string `json:"interfaces"`   // 提供的接口
}

// InterfaceDef 接口定义
type InterfaceDef struct {
	Name        string   `json:"name"`        // 接口名称
	Description string   `json:"description"` // 接口描述
	Methods     []Method `json:"methods"`     // 方法列表
}

// Method 方法定义
type Method struct {
	Name        string `json:"name"`        // 方法名
	Input       string `json:"input"`       // 输入参数
	Output      string `json:"output"`      // 输出参数
	Description string `json:"description"` // 方法描述
}

// DesignArchitecture 设计架构
func (aa *ArchitectAgent) DesignArchitecture(projectDescription string, tasks []domain.Task) (*ArchitectureDesign, error) {
	// 构建任务摘要
	taskSummary := ""
	for i, task := range tasks {
		taskSummary += fmt.Sprintf("%d. %s: %s\n", i+1, task.Title, task.Description)
	}

	// 构建请求
	req := llm.CompletionRequest{
		Model: aa.Agent.ModelName,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: aa.Agent.SystemPrompt,
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("请为以下项目设计架构：\n\n项目描述：\n%s\n\n任务列表：\n%s\n\n请以JSON格式返回架构设计。", projectDescription, taskSummary),
			},
		},
		Temperature: 0.7,
		MaxTokens:   4000,
	}

	// 调用LLM API
	resp, rateLimit, err := aa.Provider.Complete(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to get architecture design: %w", err)
	}

	// 打印速率限制信息
	fmt.Printf("\n[%s] 速率限制信息:\n", aa.Agent.Name)
	fmt.Printf("  请求限制: %d 次/分钟\n", rateLimit.RequestsLimit)
	fmt.Printf("  剩余请求: %d 次\n", rateLimit.RequestsRemaining)
	fmt.Printf("  Token 限制: %d tokens/分钟\n", rateLimit.TokensLimit)
	fmt.Printf("  剩余 Token: %d tokens\n", rateLimit.TokensRemaining)

	// 解析JSON响应
	var design ArchitectureDesign
	if err := json.Unmarshal([]byte(resp.Content), &design); err != nil {
		// 如果解析失败，返回一个基础的设计
		design = ArchitectureDesign{
			Overview:   resp.Content,
			Components: []Component{},
			Interfaces: []InterfaceDef{},
			DataFlow:   "待定义",
			TechStack:  []string{"Go", "REST API", "Database"},
		}
	}

	return &design, nil
}

// GenerateArchitectureReport 生成架构报告
func (aa *ArchitectAgent) GenerateArchitectureReport(design *ArchitectureDesign) string {
	report := fmt.Sprintf("## 架构设计报告\n\n")
	report += fmt.Sprintf("### 架构概述\n%s\n\n", design.Overview)

	report += fmt.Sprintf("### 技术栈\n")
	for _, tech := range design.TechStack {
		report += fmt.Sprintf("- %s\n", tech)
	}
	report += "\n"

	report += fmt.Sprintf("### 组件列表\n")
	for i, component := range design.Components {
		report += fmt.Sprintf("%d. **%s**\n", i+1, component.Name)
		report += fmt.Sprintf("   - 职责: %s\n", component.Role)
		if len(component.Dependencies) > 0 {
			report += fmt.Sprintf("   - 依赖: %v\n", component.Dependencies)
		}
		report += "\n"
	}

	report += fmt.Sprintf("### 数据流向\n%s\n\n", design.DataFlow)
	report += fmt.Sprintf("### 扩展性考虑\n%s\n\n", design.Scalability)
	report += fmt.Sprintf("### 安全考虑\n%s\n\n", design.Security)
	report += fmt.Sprintf("### 性能考虑\n%s\n\n", design.Performance)

	return report
}

// ValidateArchitecture 验证架构设计
func (aa *ArchitectAgent) ValidateArchitecture(design *ArchitectureDesign) []string {
	issues := []string{}

	// 检查是否有组件
	if len(design.Components) == 0 {
		issues = append(issues, "架构缺少组件定义")
	}

	// 检查是否有接口
	if len(design.Interfaces) == 0 {
		issues = append(issues, "架构缺少接口定义")
	}

	// 检查是否有技术栈
	if len(design.TechStack) == 0 {
		issues = append(issues, "架构缺少技术栈定义")
	}

	return issues
}
