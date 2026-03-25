package domain

import (
	"time"
)

// AgentType 定义智能体类型
type AgentType string

const (
	AgentTypeTaskSplitter AgentType = "task_splitter" // 任务拆分者
	AgentTypeArchitect    AgentType = "architect"     // 架构师
	AgentTypeDeveloper    AgentType = "developer"     // 开发者
	AgentTypeCritic       AgentType = "critic"        // 批评者
	AgentTypeCoordinator  AgentType = "coordinator"   // 协调者
	AgentTypeReviewer     AgentType = "reviewer"      // 审查者
)

// MessageType 定义消息类型
type MessageType string

const (
	MessageTypeSystem       MessageType = "system"       // 系统消息
	MessageTypeTaskSplit    MessageType = "task_split"   // 任务拆分
	MessageTypeArchitecture MessageType = "architecture" // 架构设计
	MessageTypeDevelopment  MessageType = "development"  // 开发实现
	MessageTypeCritique     MessageType = "critique"     // 批评辩论
	MessageTypeReview       MessageType = "review"       // 审查
	MessageTypeCompletion   MessageType = "completion"   // 完成
	MessageTypeCode         MessageType = "code"         // 代码
)

// Agent 定义智能体结构
type Agent struct {
	ID           string    `json:"id"`            // 智能体ID
	Name         string    `json:"name"`          // 角色名称
	Type         AgentType `json:"type"`          // 智能体类型
	SystemPrompt string    `json:"system_prompt"` // 系统提示
	ModelName    string    `json:"model_name"`    // 绑定的模型名称
	Priority     int       `json:"priority"`      // 优先级
	Active       bool      `json:"active"`        // 是否活跃
}

// Message 定义统一的消息结构
type Message struct {
	ID        string                 `json:"id"`        // 消息ID
	Sender    string                 `json:"sender"`    // 发送者
	Receiver  string                 `json:"receiver"`  // 接收者
	Content   string                 `json:"content"`   // 内容
	Timestamp time.Time              `json:"timestamp"` // 时间戳
	Type      MessageType            `json:"type"`      // 消息类型
	Metadata  map[string]interface{} `json:"metadata"`  // 元数据
}

// Task 定义任务结构
type Task struct {
	ID           string                 `json:"id"`           // 任务ID
	Title        string                 `json:"title"`        // 任务标题
	Description  string                 `json:"description"`  // 任务描述
	Status       string                 `json:"status"`       // 任务状态
	Priority     int                    `json:"priority"`     // 优先级
	Assignee     string                 `json:"assignee"`     // 负责人
	SubTasks     []Task                 `json:"sub_tasks"`    // 子任务
	Dependencies []string               `json:"dependencies"` // 依赖任务ID
	CreatedAt    time.Time              `json:"created_at"`   // 创建时间
	UpdatedAt    time.Time              `json:"updated_at"`   // 更新时间
	Metadata     map[string]interface{} `json:"metadata"`     // 元数据
}

// Project 定义项目结构
type Project struct {
	ID          string    `json:"id"`          // 项目ID
	Name        string    `json:"name"`        // 项目名称
	Description string    `json:"description"` // 项目描述
	Tasks       []Task    `json:"tasks"`       // 任务列表
	Status      string    `json:"status"`      // 项目状态
	CreatedAt   time.Time `json:"created_at"`  // 创建时间
	UpdatedAt   time.Time `json:"updated_at"`  // 更新时间
}

// MeetingRoom 核心调度器
type MeetingRoom struct {
	ID             string    `json:"id"`              // 会议室ID
	Agents         []Agent   `json:"agents"`          // 智能体列表
	Messages       []Message `json:"messages"`        // 消息历史
	Project        *Project  `json:"project"`         // 当前项目
	MaxRounds      int       `json:"max_rounds"`      // 最大流转轮次
	CurrentRound   int       `json:"current_round"`   // 当前轮次
	CurrentSpeaker int       `json:"current_speaker"` // 当前发言者索引
	Phase          string    `json:"phase"`           // 当前阶段
	CreatedAt      time.Time `json:"created_at"`      // 创建时间
}

// NewMeetingRoom 创建新的会议室
func NewMeetingRoom(agents []Agent, maxRounds int) *MeetingRoom {
	return &MeetingRoom{
		ID:             GenerateID(),
		Agents:         agents,
		Messages:       []Message{},
		MaxRounds:      maxRounds,
		CurrentRound:   0,
		CurrentSpeaker: 0,
		Phase:          "initialization",
		CreatedAt:      time.Now(),
	}
}

// AddMessage 添加消息到会议室
func (mr *MeetingRoom) AddMessage(msg Message) {
	msg.ID = GenerateID()
	mr.Messages = append(mr.Messages, msg)
}

// GetNextSpeaker 获取下一个发言者
func (mr *MeetingRoom) GetNextSpeaker() int {
	mr.CurrentSpeaker = (mr.CurrentSpeaker + 1) % len(mr.Agents)
	if mr.CurrentSpeaker == 0 {
		mr.CurrentRound++
	}
	return mr.CurrentSpeaker
}

// IsFinished 检查会议是否结束
func (mr *MeetingRoom) IsFinished() bool {
	return mr.CurrentRound >= mr.MaxRounds
}

// GetAgentByName 根据名称获取智能体
func (mr *MeetingRoom) GetAgentByName(name string) *Agent {
	for i := range mr.Agents {
		if mr.Agents[i].Name == name {
			return &mr.Agents[i]
		}
	}
	return nil
}

// GetAgentByType 根据类型获取智能体
func (mr *MeetingRoom) GetAgentByType(agentType AgentType) *Agent {
	for i := range mr.Agents {
		if mr.Agents[i].Type == agentType {
			return &mr.Agents[i]
		}
	}
	return nil
}

// GenerateID 生成唯一ID
func GenerateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

// randomString 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
