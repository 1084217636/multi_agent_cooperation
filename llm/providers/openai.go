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
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
	openaiProviderName   = "openai"
)

func init() {
	// 注册OpenAI Provider
	llm.Register(openaiProviderName, NewOpenAIProvider)
}

// OpenAIProvider OpenAI LLM Provider实现
type OpenAIProvider struct {
	apiKey       string
	baseURL      string
	client       *http.Client
	models       []string
	defaultModel string
	organization string // 可选的组织ID
}

// openaiRequest OpenAI API请求结构
type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

// openaiMessage OpenAI消息结构
type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiResponse OpenAI API响应结构
type openaiResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openaiMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewOpenAIProvider 创建OpenAI Provider
func NewOpenAIProvider(config llm.ProviderConfig) (llm.Provider, error) {
	if config.APIKey == "" {
		return nil, llm.ErrInvalidAPIKey
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}

	models := config.Models
	if len(models) == 0 {
		// 默认支持的模型
		models = []string{
			"gpt-4o",
			"gpt-4o-mini",
			"gpt-4-turbo",
			"gpt-3.5-turbo",
		}
	}

	defaultModel := config.DefaultModel
	if defaultModel == "" {
		defaultModel = "gpt-4o-mini"
	}

	return &OpenAIProvider{
		apiKey:       config.APIKey,
		baseURL:      baseURL,
		client:       &http.Client{Timeout: 60 * time.Second},
		models:       models,
		defaultModel: defaultModel,
		organization: config.Extra["organization"],
	}, nil
}

// Complete 实现聊天完成
func (o *OpenAIProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, *llm.RateLimitInfo, error) {
	startTime := time.Now()

	// 转换消息格式
	messages := make([]openaiMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openaiMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// 使用默认模型或请求中的模型
	model := req.Model
	if model == "" {
		model = o.defaultModel
	}

	openaiReq := openaiRequest{
		Model:       model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Stream:      false,
	}

	data, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/chat/completions", bytes.NewBuffer(data))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	if o.organization != "" {
		httpReq.Header.Set("OpenAI-Organization", o.organization)
	}

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

	var openaiResp openaiResponse
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, nil, fmt.Errorf("no choices in response")
	}

	// 解析速率限制信息
	rateLimit := parseOpenAIRateLimit(resp.Header)

	return &llm.CompletionResponse{
		Content:          openaiResp.Choices[0].Message.Content,
		PromptTokens:     openaiResp.Usage.PromptTokens,
		CompletionTokens: openaiResp.Usage.CompletionTokens,
		TotalTokens:      openaiResp.Usage.TotalTokens,
		Latency:          time.Since(startTime),
	}, rateLimit, nil
}

// GetModelList 获取支持的模型列表
func (o *OpenAIProvider) GetModelList() []string {
	return o.models
}

// GetProviderName 获取提供商名称
func (o *OpenAIProvider) GetProviderName() string {
	return openaiProviderName
}

// HealthCheck 健康检查
func (o *OpenAIProvider) HealthCheck(ctx context.Context) error {
	req := llm.CompletionRequest{
		Model:     o.defaultModel,
		Messages:  []llm.Message{{Role: "user", Content: "Hi"}},
		MaxTokens: 5,
	}
	_, _, err := o.Complete(ctx, req)
	return err
}

// parseOpenAIRateLimit 解析OpenAI速率限制头
func parseOpenAIRateLimit(headers http.Header) *llm.RateLimitInfo {
	return &llm.RateLimitInfo{
		RequestsLimit:     parseIntHeader(headers, "x-ratelimit-limit-requests"),
		RequestsRemaining: parseIntHeader(headers, "x-ratelimit-remaining-requests"),
		TokensLimit:       parseIntHeader(headers, "x-ratelimit-limit-tokens"),
		TokensRemaining:   parseIntHeader(headers, "x-ratelimit-remaining-tokens"),
		ResetTime:         time.Unix(parseInt64Header(headers, "x-ratelimit-reset-tokens"), 0),
	}
}
