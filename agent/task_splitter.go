package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"multi_agent_cooperation/domain"
	"multi_agent_cooperation/llm"
)

// TaskSplitterAgent 任务拆分智能体
type TaskSplitterAgent struct {
	Agent    domain.Agent
	Provider llm.Provider
}

// NewTaskSplitterAgent 创建任务拆分智能体
func NewTaskSplitterAgent(name, modelName string, provider llm.Provider) *TaskSplitterAgent {
	return &TaskSplitterAgent{
		Agent: domain.Agent{
			ID:   domain.GenerateID(),
			Name: name,
			Type: domain.AgentTypeTaskSplitter,
			SystemPrompt: "你是一个专业的任务拆分专家。你的职责是：\n" +
				"1. 理解复杂的项目需求\n" +
				"2. 将复杂任务拆分为可管理的子任务\n" +
				"3. 为每个子任务定义清晰的职责和依赖关系\n" +
				"4. 识别任务之间的依赖关系和执行顺序\n" +
				"5. 评估每个子任务的复杂度和优先级\n\n" +
				"输出格式要求：\n" +
				"- 使用JSON格式输出任务列表\n" +
				"- 每个任务包含：id、title、description、priority、dependencies、estimatedTime\n" +
				"- priority字段使用：high、medium、low\n" +
				"- dependencies字段列出依赖的任务ID列表",
			ModelName: modelName,
			Priority:  1,
			Active:    true,
		},
		Provider: provider,
	}
}

// TaskSplitResult 任务拆分结果
type TaskSplitResult struct {
	MainTask string        `json:"main_task"`
	Subtasks []SubTaskInfo `json:"subtasks"`
}

// SubTaskInfo 子任务信息
type SubTaskInfo struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Priority      int      `json:"priority"`
	Dependencies  []string `json:"dependencies"`
	EstimatedTime string   `json:"estimated_time"`
}

// SplitTask 拆分任务
func (tsa *TaskSplitterAgent) SplitTask(taskDescription string) (*TaskSplitResult, error) {
	// 构建请求
	req := llm.CompletionRequest{
		Model: tsa.Agent.ModelName,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: tsa.Agent.SystemPrompt,
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("请拆分以下任务：\n%s", taskDescription),
			},
		},
		Temperature: 0.7,
		MaxTokens:   4000,
	}

	// 调用LLM API
	resp, rateLimit, err := tsa.Provider.Complete(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to get task split response: %w", err)
	}

	// 打印速率限制信息
	fmt.Printf("\n[%s] 速率限制信息:\n", tsa.Agent.Name)
	fmt.Printf("  请求限制: %d 次/分钟\n", rateLimit.RequestsLimit)
	fmt.Printf("  剩余请求: %d 次\n", rateLimit.RequestsRemaining)
	fmt.Printf("  Token 限制: %d tokens/分钟\n", rateLimit.TokensLimit)
	fmt.Printf("  剩余 Token: %d tokens\n", rateLimit.TokensRemaining)

	// 解析JSON响应
	var result TaskSplitResult
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse task split result: %w", err)
	}

	return &result, nil
}

// ConvertToDomainTasks 转换为领域任务
func (tsa *TaskSplitterAgent) ConvertToDomainTasks(result *TaskSplitResult) []domain.Task {
	tasks := make([]domain.Task, len(result.Subtasks))
	for i, subtask := range result.Subtasks {
		tasks[i] = domain.Task{
			ID:           subtask.ID,
			Title:        subtask.Title,
			Description:  subtask.Description,
			Status:       "pending",
			Priority:     subtask.Priority,
			Dependencies: subtask.Dependencies,
		}
	}
	return tasks
}

// GenerateTaskSummary 生成任务摘要
func (tsa *TaskSplitterAgent) GenerateTaskSummary(result *TaskSplitResult) string {
	summary := fmt.Sprintf("## 任务拆分摘要\n\n")
	summary += fmt.Sprintf("**主任务**: %s\n\n", result.MainTask)
	summary += fmt.Sprintf("**子任务数量**: %d\n\n", len(result.Subtasks))
	summary += fmt.Sprintf("### 子任务列表\n\n")

	for i, subtask := range result.Subtasks {
		summary += fmt.Sprintf("%d. **%s** (优先级: %d)\n", i+1, subtask.Title, subtask.Priority)
		summary += fmt.Sprintf("   - 描述: %s\n", subtask.Description)
		if len(subtask.Dependencies) > 0 {
			summary += fmt.Sprintf("   - 依赖: %v\n", subtask.Dependencies)
		}
		summary += fmt.Sprintf("   - 预计时间: %s\n\n", subtask.EstimatedTime)
	}

	return summary
}
