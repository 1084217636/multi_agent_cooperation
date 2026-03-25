package llm

import (
	"context"
	"time"
)

// // Message 定义LLM消息结构
// type Message struct {
// 	Role    string `json:"role"`    // system, user, assistant
// 	Content string `json:"content"` // 消息内容
// }

// CompletionRequest 定义完成请求
type CompletionRequest struct {
	Model       string    // 模型名称
	Messages    []Message // 消息列表
	Temperature float64   // 温度参数
	MaxTokens   int       // 最大token数
	TopP        float64   // Top P采样
}

// CompletionResponse 定义完成响应
type CompletionResponse struct {
	Content          string        // 生成的内容
	PromptTokens     int           // 提示词token数
	CompletionTokens int           // 完成token数
	TotalTokens      int           // 总token数
	Latency          time.Duration // 响应延迟
}

// RateLimitInfo 定义速率限制信息
type RateLimitInfo struct {
	RequestsLimit     int       // 请求限制
	RequestsRemaining int       // 剩余请求数
	TokensLimit       int       // Token限制
	TokensRemaining   int       // 剩余Token数
	ResetTime         time.Time // 重置时间
}

// Provider 定义LLM Provider接口
// 所有大模型提供商都需要实现这个接口
type Provider interface {
	// Complete 发送聊天完成请求
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, *RateLimitInfo, error)

	// GetModelList 获取支持的模型列表
	GetModelList() []string

	// GetProviderName 获取提供商名称
	GetProviderName() string

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// ProviderConfig 定义Provider配置
type ProviderConfig struct {
	Name         string            `yaml:"name" json:"name"`                   // 提供商名称
	APIKey       string            `yaml:"api_key" json:"api_key"`             // API密钥
	BaseURL      string            `yaml:"base_url" json:"base_url"`           // 基础URL（可选）
	Models       []string          `yaml:"models" json:"models"`               // 支持的模型
	DefaultModel string            `yaml:"default_model" json:"default_model"` // 默认模型
	Extra        map[string]string `yaml:"extra" json:"extra"`                 // 额外配置
}

// ProviderFactory Provider工厂函数类型
type ProviderFactory func(config ProviderConfig) (Provider, error)

// registry Provider注册表
var registry = make(map[string]ProviderFactory)

// Register 注册Provider
func Register(name string, factory ProviderFactory) {
	registry[name] = factory
}

// Create 创建Provider实例
func Create(config ProviderConfig) (Provider, error) {
	factory, ok := registry[config.Name]
	if !ok {
		return nil, ErrProviderNotFound
	}
	return factory(config)
}

// GetRegisteredProviders 获取已注册的Provider列表
func GetRegisteredProviders() []string {
	providers := make([]string, 0, len(registry))
	for name := range registry {
		providers = append(providers, name)
	}
	return providers
}
