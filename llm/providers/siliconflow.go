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
	defaultSiliconFlowBaseURL = "https://api.siliconflow.cn/v1"
	siliconflowProviderName   = "siliconflow"
)

func init() {
	// 注册SiliconFlow Provider
	llm.Register(siliconflowProviderName, NewSiliconFlowProvider)
}

// SiliconFlowProvider SiliconFlow LLM Provider实现
// SiliconFlow API兼容OpenAI格式
// 文档: https://docs.siliconflow.cn
// 支持模型: Qwen/Qwen2.5-72B-Instruct, deepseek-ai/DeepSeek-V2.5 等
// API端点: https://api.siliconflow.cn/v1
// 特点: 每用户送3亿token，支持流式输出

type SiliconFlowProvider struct {
	apiKey       string
	baseURL      string
	client       *http.Client
	models       []string
	defaultModel string
}

// siliconflowRequest SiliconFlow API请求结构 (兼容OpenAI格式)
type siliconflowRequest struct {
	Model       string          `json:"model"`
	Messages    []siliconflowMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

// siliconflowMessage SiliconFlow消息结构 (兼容OpenAI格式)
type siliconflowMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// siliconflowResponse SiliconFlow API响应结构 (兼容OpenAI格式)
type siliconflowResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int              `json:"index"`
		Message      siliconflowMessage `json:"message"`
		FinishReason string           `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewSiliconFlowProvider 创建SiliconFlow Provider
func NewSiliconFlowProvider(config llm.ProviderConfig) (llm.Provider, error) {
	if config.APIKey == "" {
		return nil, llm.ErrInvalidAPIKey
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultSiliconFlowBaseURL
	}

	models := config.Models
	if len(models) == 0 {
		// 默认支持的模型列表
		models = []string{
			"Qwen/Qwen2.5-72B-Instruct",       // 通义千问2.5 72B
			"deepseek-ai/DeepSeek-V2.5",       // DeepSeek V2.5
			"Qwen/Qwen2.5-7B-Instruct",        // 通义千问2.5 7B
			"Qwen/Qwen2.5-14B-Instruct",       // 通义千问2.5 14B
			"Qwen/Qwen2.5-32B-Instruct",       // 通义千问2.5 32B
			"meta-llama/Llama-3.1-70B-Instruct", // Llama 3.1 70B
			"meta-llama/Llama-3.1-8B-Instruct",  // Llama 3.1 8B
		}
	}

	defaultModel := config.DefaultModel
	if defaultModel == "" {
		defaultModel = "Qwen/Qwen2.5-72B-Instruct"
	}

	return &SiliconFlowProvider{
		apiKey:       config.APIKey,
		baseURL:      baseURL,
		client:       &http.Client{Timeout: 120 * time.Second},
		models:       models,
		defaultModel: defaultModel,
	}, nil
}

// Complete 实现聊天完成
func (s *SiliconFlowProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, *llm.RateLimitInfo, error) {
	startTime := time.Now()

	// 转换消息格式
	messages := make([]siliconflowMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = siliconflowMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// 使用默认模型或请求中的模型
	model := req.Model
	if model == "" {
		model = s.defaultModel
	}

	sfReq := siliconflowRequest{
		Model:       model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Stream:      false, // 非流式请求
	}

	data, err := json.Marshal(sfReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/chat/completions", bytes.NewBuffer(data))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(httpReq)
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

	var sfResp siliconflowResponse
	if err := json.Unmarshal(body, &sfResp); err != nil {
		return nil, nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(sfResp.Choices) == 0 {
		return nil, nil, fmt.Errorf("no choices in response")
	}

	// 解析速率限制信息
	rateLimit := parseSiliconFlowRateLimit(resp.Header)

	return &llm.CompletionResponse{
		Content:          sfResp.Choices[0].Message.Content,
		PromptTokens:     sfResp.Usage.PromptTokens,
		CompletionTokens: sfResp.Usage.CompletionTokens,
		TotalTokens:      sfResp.Usage.TotalTokens,
		Latency:          time.Since(startTime),
	}, rateLimit, nil
}

// GetModelList 获取支持的模型列表
func (s *SiliconFlowProvider) GetModelList() []string {
	return s.models
}

// GetProviderName 获取提供商名称
func (s *SiliconFlowProvider) GetProviderName() string {
	return siliconflowProviderName
}

// HealthCheck 健康检查
func (s *SiliconFlowProvider) HealthCheck(ctx context.Context) error {
	req := llm.CompletionRequest{
		Model:     s.defaultModel,
		Messages:  []llm.Message{{Role: "user", Content: "你好"}},
		MaxTokens: 5,
	}
	_, _, err := s.Complete(ctx, req)
	return err
}

// parseSiliconFlowRateLimit 解析SiliconFlow速率限制头
func parseSiliconFlowRateLimit(headers http.Header) *llm.RateLimitInfo {
	return &llm.RateLimitInfo{
		RequestsLimit:     parseIntHeader(headers, "x-ratelimit-limit-requests"),
		RequestsRemaining: parseIntHeader(headers, "x-ratelimit-remaining-requests"),
		TokensLimit:       parseIntHeader(headers, "x-ratelimit-limit-tokens"),
		TokensRemaining:   parseIntHeader(headers, "x-ratelimit-remaining-tokens"),
		ResetTime:         time.Unix(parseInt64Header(headers, "x-ratelimit-reset-tokens"), 0),
	}
}