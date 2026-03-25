package engine

import (
	"context"
	"fmt"
	"time"

	"multi_agent_cooperation/domain"
	"multi_agent_cooperation/llm"
)

// ChatEngine 定义群聊引擎
type ChatEngine struct {
	MeetingRoom     *domain.MeetingRoom
	PrimaryProvider llm.Provider   // 主要provider（本地Ollama）
	BackupProvider  llm.Provider   // 备份provider（云端SiliconFlow）
	FailureCount    map[string]int // 失败计数，按智能体名称
	UseBackup       bool           // 是否使用备份provider
}

// NewChatEngine 创建新的群聊引擎
func NewChatEngine(agents []domain.Agent, maxRounds int, primaryProvider, backupProvider llm.Provider) *ChatEngine {
	return &ChatEngine{
		MeetingRoom:     domain.NewMeetingRoom(agents, maxRounds),
		PrimaryProvider: primaryProvider,
		BackupProvider:  backupProvider,
		FailureCount:    make(map[string]int),
		UseBackup:       false,
	}
}

// StartMeeting 开始会议
func (ce *ChatEngine) StartMeeting(initialPrompt string) error {
	// 初始化会议，添加初始需求消息
	initialMsg := domain.Message{
		Sender:    "System",
		Receiver:  "All",
		Content:   initialPrompt,
		Timestamp: time.Now(),
	}
	ce.MeetingRoom.AddMessage(initialMsg)

	// 循环处理轮次
	for !ce.MeetingRoom.IsFinished() {
		// 获取当前发言者
		speakerIndex := ce.MeetingRoom.CurrentSpeaker
		speaker := ce.MeetingRoom.Agents[speakerIndex]

		// 构建消息历史
		var messages []llm.Message
		// 添加系统提示
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: speaker.SystemPrompt,
		})
		// 添加对话历史
		for _, msg := range ce.MeetingRoom.Messages {
			role := "user"
			if msg.Sender == speaker.Name {
				role = "assistant"
			}
			messages = append(messages, llm.Message{
				Role:    role,
				Content: fmt.Sprintf("%s: %s", msg.Sender, msg.Content),
			})
		}

		// 构建请求
		req := llm.CompletionRequest{
			Model:       speaker.ModelName,
			Messages:    messages,
			Temperature: 0.7,
			MaxTokens:   2000,
		}

		// 调用 LLM API 获取回复（带模型路由）
		resp, rateLimitInfo, err := ce.completeWithFallback(context.Background(), req, speaker.Name)
		if err != nil {
			return fmt.Errorf("failed to get response from %s: %w", speaker.Name, err)
		}

		// 打印速率限制信息
		fmt.Printf("[ChatEngine] Rate Limit Info - Requests: %d/%d, Tokens: %d/%d\n",
			rateLimitInfo.RequestsRemaining,
			rateLimitInfo.RequestsLimit,
			rateLimitInfo.TokensRemaining,
			rateLimitInfo.TokensLimit)

		// 创建新消息
		newMsg := domain.Message{
			Sender:    speaker.Name,
			Receiver:  "All",
			Content:   resp.Content,
			Timestamp: time.Now(),
		}

		// 添加消息到会议室
		ce.MeetingRoom.AddMessage(newMsg)

		// 打印消息
		fmt.Printf("[%s] %s: %s\n", newMsg.Timestamp.Format("2006-01-02 15:04:05"), newMsg.Sender, newMsg.Content)

		// 获取下一个发言者
		ce.MeetingRoom.GetNextSpeaker()
	}

	return nil
}

// completeWithFallback 带失败回退的模型调用
func (ce *ChatEngine) completeWithFallback(ctx context.Context, req llm.CompletionRequest, agentName string) (*llm.CompletionResponse, *llm.RateLimitInfo, error) {
	const maxFailures = 3

	// 如果已经切换到备份provider，直接使用备份
	if ce.UseBackup && ce.BackupProvider != nil {
		fmt.Printf("[ChatEngine] Using backup provider for %s\n", agentName)
		return ce.BackupProvider.Complete(ctx, req)
	}

	// 优先使用主要provider（本地Ollama）
	fmt.Printf("[ChatEngine] Using primary provider for %s\n", agentName)
	resp, rateLimitInfo, err := ce.PrimaryProvider.Complete(ctx, req)

	// 如果成功，重置失败计数
	if err == nil {
		// 检查回复内容是否包含特定的失败标识
		if ce.shouldSwitchToBackup(resp.Content) {
			fmt.Printf("[ChatEngine] Response contains failure indicator, switching to backup\n")
			ce.FailureCount[agentName] = maxFailures
		} else {
			ce.FailureCount[agentName] = 0
			return resp, rateLimitInfo, nil
		}
	} else {
		// 失败，增加失败计数
		ce.FailureCount[agentName]++
		fmt.Printf("[ChatEngine] Primary provider failed (%d/%d): %v\n",
			ce.FailureCount[agentName], maxFailures, err)
	}

	// 检查是否需要切换到备份provider
	if ce.FailureCount[agentName] >= maxFailures && ce.BackupProvider != nil {
		fmt.Printf("[ChatEngine] Switching to backup provider after %d failures\n", maxFailures)
		ce.UseBackup = true
		return ce.BackupProvider.Complete(ctx, req)
	}

	// 如果没有备份provider或还没达到切换条件，返回原始错误
	return nil, nil, err
}

// shouldSwitchToBackup 检查是否应该切换到备份provider
func (ce *ChatEngine) shouldSwitchToBackup(content string) bool {
	// 检查回复内容是否包含特定的失败标识
	failureIndicators := []string{
		"I cannot solve this",
		"I don't know",
		"无法解决",
		"不知道",
		"抱歉，我无法",
		"Error",
		"error",
		"Exception",
		"exception",
	}

	for _, indicator := range failureIndicators {
		if containsIgnoreCase(content, indicator) {
			return true
		}
	}

	return false
}

// containsIgnoreCase 忽略大小写检查字符串是否包含子串
func containsIgnoreCase(s, substr string) bool {
	return contains([]rune(s), []rune(substr), true)
}

// contains 检查字符串是否包含子串
func contains(s, substr []rune, ignoreCase bool) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			a, b := s[i+j], substr[j]
			if ignoreCase {
				if a >= 'A' && a <= 'Z' {
					a += 'a' - 'A'
				}
				if b >= 'A' && b <= 'Z' {
					b += 'a' - 'A'
				}
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// GenerateReport 生成会议报告
func (ce *ChatEngine) GenerateReport() string {
	report := "# 会议报告\n\n"
	report += "## 会议参与者\n"
	for _, agent := range ce.MeetingRoom.Agents {
		report += fmt.Sprintf("- %s (模型: %s)\n", agent.Name, agent.ModelName)
	}
	report += "\n## 会议记录\n"
	for _, msg := range ce.MeetingRoom.Messages {
		report += fmt.Sprintf("### %s - %s\n", msg.Timestamp.Format("2006-01-02 15:04:05"), msg.Sender)
		report += msg.Content + "\n\n"
	}
	report += "## 会议总结\n"
	report += fmt.Sprintf("会议已完成，共进行了 %d 轮讨论。\n", ce.MeetingRoom.CurrentRound)
	return report
}
