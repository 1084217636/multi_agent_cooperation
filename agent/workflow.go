package agent

import (
	"fmt"
	"strings"
	"time"

	"multi_agent_cooperation/domain"
	"multi_agent_cooperation/executor"
)

// CollaborationPhase 协作阶段
type CollaborationPhase string

const (
	PhaseInitialization CollaborationPhase = "initialization" // 初始化阶段
	PhaseTaskSplit      CollaborationPhase = "task_split"     // 任务拆分阶段
	PhaseArchitecture   CollaborationPhase = "architecture"   // 架构设计阶段
	PhaseDevelopment    CollaborationPhase = "development"    // 开发实现阶段
	PhaseCritique       CollaborationPhase = "critique"       // 批评辩论阶段
	PhaseReview         CollaborationPhase = "review"         // 审查阶段
	PhaseCompletion     CollaborationPhase = "completion"     // 完成阶段
)

// CollaborationWorkflow 协作工作流
type CollaborationWorkflow struct {
	MeetingRoom      *domain.MeetingRoom
	TaskSplitter     *TaskSplitterAgent
	Architect        *ArchitectAgent
	Developer        *DeveloperAgent
	Critic           *CriticAgent
	Runner           *executor.Runner
	CurrentPhase     CollaborationPhase
	MaxIterations    int
	CurrentIteration int
	Reports          []string
}

// NewCollaborationWorkflow 创建协作工作流
func NewCollaborationWorkflow(
	meetingRoom *domain.MeetingRoom,
	taskSplitter *TaskSplitterAgent,
	architect *ArchitectAgent,
	developer *DeveloperAgent,
	critic *CriticAgent,
	runner *executor.Runner,
) *CollaborationWorkflow {
	return &CollaborationWorkflow{
		MeetingRoom:      meetingRoom,
		TaskSplitter:     taskSplitter,
		Architect:        architect,
		Developer:        developer,
		Critic:           critic,
		Runner:           runner,
		CurrentPhase:     PhaseInitialization,
		MaxIterations:    3,
		CurrentIteration: 0,
		Reports:          []string{},
	}
}

// ExecuteWorkflow 执行协作工作流
func (cw *CollaborationWorkflow) ExecuteWorkflow(projectDescription string) error {
	fmt.Println("🚀 开始多智能体协作工作流...")
	fmt.Println(strings.Repeat("=", 50))

	// 初始化阶段
	if err := cw.executeInitializationPhase(projectDescription); err != nil {
		return fmt.Errorf("初始化阶段失败: %w", err)
	}

	// 主协作循环
	for cw.CurrentIteration < cw.MaxIterations {
		cw.CurrentIteration++
		fmt.Printf("\n🔄 第 %d 轮协作开始...\n", cw.CurrentIteration)
		fmt.Println(strings.Repeat("-", 50))

		// 任务拆分阶段
		if err := cw.executeTaskSplitPhase(); err != nil {
			return fmt.Errorf("任务拆分阶段失败: %w", err)
		}

		// 架构设计阶段
		if err := cw.executeArchitecturePhase(); err != nil {
			return fmt.Errorf("架构设计阶段失败: %w", err)
		}

		// 开发实现阶段
		if err := cw.executeDevelopmentPhase(); err != nil {
			return fmt.Errorf("开发实现阶段失败: %w", err)
		}

		// 批评辩论阶段
		if err := cw.executeCritiquePhase(); err != nil {
			return fmt.Errorf("批评辩论阶段失败: %w", err)
		}

		// 审查阶段
		if err := cw.executeReviewPhase(); err != nil {
			return fmt.Errorf("审查阶段失败: %w", err)
		}

		fmt.Printf("✅ 第 %d 轮协作完成\n", cw.CurrentIteration)
	}

	// 完成阶段
	if err := cw.executeCompletionPhase(); err != nil {
		return fmt.Errorf("完成阶段失败: %w", err)
	}

	fmt.Println("\n🎉 多智能体协作工作流完成！")
	fmt.Println(strings.Repeat("=", 50))

	return nil
}

// executeInitializationPhase 执行初始化阶段
func (cw *CollaborationWorkflow) executeInitializationPhase(projectDescription string) error {
	cw.CurrentPhase = PhaseInitialization
	fmt.Println("\n📋 初始化阶段")

	// 创建项目
	project := &domain.Project{
		ID:          domain.GenerateID(),
		Name:        "协作项目",
		Description: projectDescription,
		Tasks:       []domain.Task{},
		Status:      "initialized",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	cw.MeetingRoom.Project = project
	cw.MeetingRoom.Phase = string(PhaseInitialization)

	// 添加初始化消息
	msg := domain.Message{
		Sender:    "System",
		Receiver:  "All",
		Content:   fmt.Sprintf("项目初始化完成。项目描述：%s", projectDescription),
		Timestamp: time.Now(),
		Type:      domain.MessageTypeSystem,
	}
	cw.MeetingRoom.AddMessage(msg)

	fmt.Printf("✅ 项目初始化完成: %s\n", project.ID)
	return nil
}

// executeTaskSplitPhase 执行任务拆分阶段
func (cw *CollaborationWorkflow) executeTaskSplitPhase() error {
	cw.CurrentPhase = PhaseTaskSplit
	fmt.Println("\n🔧 任务拆分阶段")

	// 任务拆分者拆分任务
	taskSplitResult, err := cw.TaskSplitter.SplitTask(cw.MeetingRoom.Project.Description)
	if err != nil {
		return err
	}

	// 转换为领域任务
	tasks := cw.TaskSplitter.ConvertToDomainTasks(taskSplitResult)

	// 更新项目任务
	cw.MeetingRoom.Project.Tasks = tasks

	// 生成报告
	report := cw.TaskSplitter.GenerateTaskSummary(taskSplitResult)
	cw.Reports = append(cw.Reports, report)

	// 添加消息
	msg := domain.Message{
		Sender:    cw.TaskSplitter.Agent.Name,
		Receiver:  "All",
		Content:   report,
		Timestamp: time.Now(),
		Type:      domain.MessageTypeTaskSplit,
	}
	cw.MeetingRoom.AddMessage(msg)

	fmt.Printf("✅ 任务拆分完成，共拆分出 %d 个子任务\n", len(tasks))
	return nil
}

// executeArchitecturePhase 执行架构设计阶段
func (cw *CollaborationWorkflow) executeArchitecturePhase() error {
	cw.CurrentPhase = PhaseArchitecture
	fmt.Println("\n🏗️ 架构设计阶段")

	// 架构师设计架构
	architecture, err := cw.Architect.DesignArchitecture(
		cw.MeetingRoom.Project.Description,
		cw.MeetingRoom.Project.Tasks,
	)
	if err != nil {
		return err
	}

	// 生成报告
	report := cw.Architect.GenerateArchitectureReport(architecture)
	cw.Reports = append(cw.Reports, report)

	// 添加消息
	msg := domain.Message{
		Sender:    cw.Architect.Agent.Name,
		Receiver:  "All",
		Content:   report,
		Timestamp: time.Now(),
		Type:      domain.MessageTypeArchitecture,
	}
	cw.MeetingRoom.AddMessage(msg)

	fmt.Printf("✅ 架构设计完成\n")
	return nil
}

// executeDevelopmentPhase 执行开发实现阶段
func (cw *CollaborationWorkflow) executeDevelopmentPhase() error {
	cw.CurrentPhase = PhaseDevelopment
	fmt.Println("\n💻 开发实现阶段")

	// 开发者实现任务
	for i, task := range cw.MeetingRoom.Project.Tasks {
		if task.Status != "completed" {
			// 获取最新的架构报告
			architectureReport := ""
			for _, report := range cw.Reports {
				if len(report) > 0 && report[0:2] == "##" {
					architectureReport = report
					break
				}
			}

			// 实现任务（带自愈循环）
			if err := cw.implementTaskWithSelfHeal(task, architectureReport); err != nil {
				return err
			}

			// 更新任务状态
			cw.MeetingRoom.Project.Tasks[i].Status = "completed"

			fmt.Printf("✅ 任务 %s 实现完成\n", task.Title)
		}
	}

	return nil
}

// implementTaskWithSelfHeal 带自愈循环的任务实现
func (cw *CollaborationWorkflow) implementTaskWithSelfHeal(task domain.Task, architectureReport string) error {
	maxRetries := 3
	retryCount := 0

	for retryCount < maxRetries {
		retryCount++
		fmt.Printf("\n🔄 第 %d 次尝试实现任务: %s\n", retryCount, task.Title)

		// 开发者实现任务
		implementation, err := cw.Developer.ImplementTask(task, architectureReport)
		if err != nil {
			return fmt.Errorf("failed to implement task: %w", err)
		}

		// 生成报告
		report := cw.Developer.GenerateCodeReport(implementation)
		cw.Reports = append(cw.Reports, report)

		// 添加消息
		msg := domain.Message{
			Sender:    cw.Developer.Agent.Name,
			Receiver:  "All",
			Content:   report,
			Timestamp: time.Now(),
			Type:      domain.MessageTypeCode,
		}
		cw.MeetingRoom.AddMessage(msg)

		// 如果有Runner，执行代码
		if cw.Runner != nil {
			fmt.Printf("🐳 在Docker中执行代码...\n")

			// 提取代码内容
			code := implementation.Code

			// 执行代码
			result, err := cw.Runner.ExecuteWithGoModTidy(code)
			if err != nil {
				fmt.Printf("❌ Docker执行失败: %v\n", err)
				continue
			}

			// 检查执行结果
			if result.ExitCode != 0 || result.Stderr != "" {
				fmt.Printf("❌ 代码执行失败\n")
				fmt.Printf("   Stderr: %s\n", result.Stderr)

				// 让批评者分析错误
				critiqueResult, err := cw.Critic.CritiqueCode(code, task)
				if err != nil {
					fmt.Printf("⚠️  批评者分析失败: %v\n", err)
					continue
				}

				// 生成批评报告
				critiqueReport := cw.Critic.GenerateCritiqueReport(critiqueResult)
				cw.Reports = append(cw.Reports, critiqueReport)

				// 添加批评消息
				critiqueMsg := domain.Message{
					Sender:    cw.Critic.Agent.Name,
					Receiver:  cw.Developer.Agent.Name,
					Content:   critiqueReport,
					Timestamp: time.Now(),
					Type:      domain.MessageTypeCritique,
				}
				cw.MeetingRoom.AddMessage(critiqueMsg)

				// 更新架构报告，包含错误信息，供下次实现参考
				architectureReport += "\n\n## 错误分析\n"
				architectureReport += fmt.Sprintf("任务: %s\n", task.Title)
				architectureReport += fmt.Sprintf("错误信息: %s\n", result.Stderr)
				architectureReport += critiqueReport

				continue
			}

			// 执行成功
			fmt.Printf("✅ 代码执行成功\n")
			fmt.Printf("   Stdout: %s\n", result.Stdout)

			// 添加执行成功消息
			successMsg := domain.Message{
				Sender:    "System",
				Receiver:  "All",
				Content:   fmt.Sprintf("任务 %s 代码执行成功！输出: %s", task.Title, result.Stdout),
				Timestamp: time.Now(),
				Type:      domain.MessageTypeSystem,
			}
			cw.MeetingRoom.AddMessage(successMsg)

			return nil
		} else {
			// 如果没有Runner，直接返回成功
			fmt.Printf("⚠️  没有配置Docker执行引擎，跳过执行验证\n")
			return nil
		}
	}

	return fmt.Errorf("failed to implement task after %d retries", maxRetries)
}

// executeCritiquePhase 执行批评辩论阶段
func (cw *CollaborationWorkflow) executeCritiquePhase() error {
	cw.CurrentPhase = PhaseCritique
	fmt.Println("\n🔍 批评辩论阶段")

	// 批评者批评架构
	architectureReport := ""
	for _, report := range cw.Reports {
		if len(report) > 0 && report[0:2] == "##" {
			architectureReport = report
			break
		}
	}

	critiqueResult, err := cw.Critic.CritiqueArchitecture(architectureReport)
	if err != nil {
		return err
	}

	// 生成报告
	report := cw.Critic.GenerateCritiqueReport(critiqueResult)
	cw.Reports = append(cw.Reports, report)

	// 添加消息
	msg := domain.Message{
		Sender:    cw.Critic.Agent.Name,
		Receiver:  "All",
		Content:   report,
		Timestamp: time.Now(),
		Type:      domain.MessageTypeCritique,
	}
	cw.MeetingRoom.AddMessage(msg)

	fmt.Printf("✅ 批评辩论完成\n")
	return nil
}

// executeReviewPhase 执行审查阶段
func (cw *CollaborationWorkflow) executeReviewPhase() error {
	cw.CurrentPhase = PhaseReview
	fmt.Println("\n📊 审查阶段")

	// 检查是否需要继续迭代
	// 这里简化处理，总是继续到最大迭代次数
	continueIteration := cw.CurrentIteration < cw.MaxIterations

	if continueIteration {
		fmt.Printf("⏭️ 需要继续改进，进入下一轮协作\n")
	} else {
		fmt.Printf("✅ 审查通过，准备完成项目\n")
	}

	return nil
}

// executeCompletionPhase 执行完成阶段
func (cw *CollaborationWorkflow) executeCompletionPhase() error {
	cw.CurrentPhase = PhaseCompletion
	fmt.Println("\n🎯 完成阶段")

	// 更新项目状态
	cw.MeetingRoom.Project.Status = "completed"
	cw.MeetingRoom.Project.UpdatedAt = time.Now()

	// 添加完成消息
	msg := domain.Message{
		Sender:    "System",
		Receiver:  "All",
		Content:   "项目已完成！",
		Timestamp: time.Now(),
		Type:      domain.MessageTypeCompletion,
	}
	cw.MeetingRoom.AddMessage(msg)

	fmt.Printf("✅ 项目完成: %s\n", cw.MeetingRoom.Project.ID)
	return nil
}

// GenerateFinalReport 生成最终报告
func (cw *CollaborationWorkflow) GenerateFinalReport() string {
	report := "# 多智能体协作最终报告\n\n"
	report += fmt.Sprintf("## 项目信息\n")
	report += fmt.Sprintf("- 项目ID: %s\n", cw.MeetingRoom.Project.ID)
	report += fmt.Sprintf("- 项目名称: %s\n", cw.MeetingRoom.Project.Name)
	report += fmt.Sprintf("- 项目描述: %s\n", cw.MeetingRoom.Project.Description)
	report += fmt.Sprintf("- 项目状态: %s\n", cw.MeetingRoom.Project.Status)
	report += fmt.Sprintf("- 创建时间: %s\n", cw.MeetingRoom.Project.CreatedAt.Format("2006-01-02 15:04:05"))
	report += fmt.Sprintf("- 完成时间: %s\n\n", cw.MeetingRoom.Project.UpdatedAt.Format("2006-01-02 15:04:05"))

	report += fmt.Sprintf("## 协作统计\n")
	report += fmt.Sprintf("- 总轮次: %d\n", cw.CurrentIteration)
	report += fmt.Sprintf("- 参与智能体: %d\n", len(cw.MeetingRoom.Agents))
	report += fmt.Sprintf("- 总消息数: %d\n", len(cw.MeetingRoom.Messages))
	report += fmt.Sprintf("- 总任务数: %d\n\n", len(cw.MeetingRoom.Project.Tasks))

	report += fmt.Sprintf("## 详细报告\n\n")
	for i, r := range cw.Reports {
		report += fmt.Sprintf("### 报告 %d\n%s\n\n", i+1, r)
	}

	report += fmt.Sprintf("## 消息历史\n\n")
	for _, msg := range cw.MeetingRoom.Messages {
		report += fmt.Sprintf("### %s - %s\n", msg.Timestamp.Format("2006-01-02 15:04:05"), msg.Sender)
		report += msg.Content + "\n\n"
	}

	return report
}

// extractTasks 从消息中提取任务
func extractTasks(messages []domain.Message) []domain.Task {
	var tasks []domain.Task
	for _, msg := range messages {
		if msg.Type == domain.MessageTypeTaskSplit {
			if taskData, ok := msg.Metadata["tasks"].([]domain.Task); ok {
				tasks = append(tasks, taskData...)
			}
		}
	}
	return tasks
}

// extractArchitecture 从消息中提取架构信息
func extractArchitecture(messages []domain.Message) map[string]interface{} {
	for _, msg := range messages {
		if msg.Type == domain.MessageTypeArchitecture {
			if archData, ok := msg.Metadata["architecture"].(map[string]interface{}); ok {
				return archData
			}
		}
	}
	return nil
}

// extractCode 从消息中提取代码信息
func extractCode(messages []domain.Message) map[string]interface{} {
	codeData := make(map[string]interface{})
	for _, msg := range messages {
		if msg.Type == domain.MessageTypeCode {
			if data, ok := msg.Metadata["code"].(map[string]interface{}); ok {
				for k, v := range data {
					codeData[k] = v
				}
			}
		}
	}
	return codeData
}

// extractIssues 从消息中提取问题
func extractIssues(messages []domain.Message) []Issue {
	var issues []Issue
	for _, msg := range messages {
		if msg.Type == domain.MessageTypeCritique {
			if issueData, ok := msg.Metadata["issues"].([]Issue); ok {
				issues = append(issues, issueData...)
			}
		}
	}
	return issues
}

// generateContext 生成上下文信息
func generateContext(messages []domain.Message) string {
	var context strings.Builder
	context.WriteString("## 协作历史\n\n")

	for i, msg := range messages {
		context.WriteString(fmt.Sprintf("### %d. %s -> %s\n", i+1, msg.Sender, msg.Receiver))
		context.WriteString(fmt.Sprintf("**时间**: %s\n", msg.Timestamp.Format("2006-01-02 15:04:05")))
		context.WriteString(fmt.Sprintf("**类型**: %s\n", msg.Type))
		context.WriteString(fmt.Sprintf("**内容**: %s\n\n", msg.Content))
	}

	return context.String()
}

// generateSummary 生成摘要
func generateSummary(messages []domain.Message) string {
	summary := &strings.Builder{}
	summary.WriteString("## 协作摘要\n\n")

	// 统计各类消息数量
	messageCounts := make(map[domain.MessageType]int)
	for _, msg := range messages {
		messageCounts[msg.Type]++
	}

	summary.WriteString("### 消息统计\n\n")
	for msgType, count := range messageCounts {
		summary.WriteString(fmt.Sprintf("- %s: %d 条\n", msgType, count))
	}

	summary.WriteString("\n### 关键信息\n\n")

	// 提取关键信息
	tasks := extractTasks(messages)
	if len(tasks) > 0 {
		summary.WriteString(fmt.Sprintf("**任务数量**: %d\n", len(tasks)))
	}

	issues := extractIssues(messages)
	if len(issues) > 0 {
		summary.WriteString(fmt.Sprintf("**问题数量**: %d\n", len(issues)))
	}

	return summary.String()
}
