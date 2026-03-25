package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"multi_agent_cooperation/llm"
)

const (
	defaultOllamaBaseURL = "http://localhost:11434/v1"
	ollamaProviderName   = "ollama"
)

func init() {
	// 注册Ollama Provider
	llm.Register(ollamaProviderName, NewOllamaProvider)
}

// OllamaProvider Ollama LLM Provider实现
// Ollama是本地LLM运行工具，提供兼容OpenAI的API接口
// 文档: https://ollama.com/docs/api
// 默认地址: http://localhost:11434/v1
// 特点: 本地运行，不需要API密钥，支持多种本地模型

type OllamaProvider struct {
	baseURL      string
	client       *http.Client
	models       []string
	defaultModel string
}

// ollamaRequest Ollama API请求结构 (兼容OpenAI格式)
type ollamaRequest struct {
	Model       string          `json:"model"`
	Messages    []ollamaMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

// ollamaMessage Ollama消息结构 (兼容OpenAI格式)
type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaResponse Ollama API响应结构 (兼容OpenAI格式)
type ollamaResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      ollamaMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewOllamaProvider 创建Ollama Provider
func NewOllamaProvider(config llm.ProviderConfig) (llm.Provider, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	}

	models := config.Models
	if len(models) == 0 {
		// 默认支持的模型列表（需要用户提前下载）
		models = []string{
			"llama3",          // Meta Llama 3
			"llama3.1",        // Meta Llama 3.1
			"qwen2",           // 通义千问2
			"gemma2",          // Google Gemma 2
			"mistral",         // Mistral
			"phi3",            // Microsoft Phi-3
			"deepseek-coder",  // DeepSeek Coder
		}
	}

	defaultModel := config.DefaultModel
	if defaultModel == "" {
		defaultModel = "llama3"
	}

	return &OllamaProvider{
		baseURL:      baseURL,
		client:       &http.Client{Timeout: 120 * time.Second},
		models:       models,
		defaultModel: defaultModel,
	}, nil
}

// Complete 实现聊天完成
func (o *OllamaProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, *llm.RateLimitInfo, error) {
	startTime := time.Now()

	// 转换消息格式
	messages := make([]ollamaMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = ollamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// 使用默认模型或请求中的模型
	model := req.Model
	if model == "" {
		model = o.defaultModel
	}

	ollamaReq := ollamaRequest{
		Model:       model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Stream:      false, // 非流式请求
	}

	data, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/chat/completions", bytes.NewBuffer(data))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	// Ollama不需要API密钥

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", llm.ErrNetworkError, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, nil, llm.ErrRateLimitExceeded
		}
		return nil, nil, fmt.Errorf("%w: status %d: %s", llm.ErrAPIError, resp.StatusCode, string(body))
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(ollamaResp.Choices) == 0 {
		return nil, nil, fmt.Errorf("no choices in response")
	}

	// Ollama通常不提供速率限制信息，返回默认值
	rateLimit := &llm.RateLimitInfo{
		RequestsLimit:     1000,
		RequestsRemaining: 999,
		TokensLimit:       100000,
		TokensRemaining:   99999,
		ResetTime:         time.Now().Add(time.Hour),
	}

	return &llm.CompletionResponse{
		Content:          ollamaResp.Choices[0].Message.Content,
		PromptTokens:     ollamaResp.Usage.PromptTokens,
		CompletionTokens: ollamaResp.Usage.CompletionTokens,
		TotalTokens:      ollamaResp.Usage.TotalTokens,
		Latency:          time.Since(startTime),
	}, rateLimit, nil
}

// GetModelList 获取支持的模型列表
func (o *OllamaProvider) GetModelList() []string {
	return o.models
}

// GetProviderName 获取提供商名称
func (o *OllamaProvider) GetProviderName() string {
	return ollamaProviderName
}

// HealthCheck 健康检查
func (o *OllamaProvider) HealthCheck(ctx context.Context) error {
	req := llm.CompletionRequest{
		Model:     o.defaultModel,
		Messages:  []llm.Message{{Role: "user", Content: "hello"}},
		MaxTokens: 5,
	}
	_, _, err := o.Complete(ctx, req)
	return err
}