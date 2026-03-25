package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"multi_agent_cooperation/llm"
)

const (
	defaultGroqBaseURL = "https://api.groq.com/openai/v1"
	groqProviderName   = "groq"
)

func init() {
	// 注册Groq Provider
	llm.Register(groqProviderName, NewGroqProvider)
}

// GroqProvider Groq LLM Provider实现
type GroqProvider struct {
	apiKey       string
	baseURL      string
	client       *http.Client
	directClient *http.Client
	models       []string
	defaultModel string
}

// groqRequest Groq API请求结构（支持/responses端点）
type groqRequest struct {
	Model       string        `json:"model"`
	Input       string        `json:"input,omitempty"`    // 用于/responses端点
	Messages    []groqMessage `json:"messages,omitempty"` // 用于/chat/completions端点
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	TopP        float64       `json:"top_p,omitempty"`
}

// groqMessage Groq消息结构
type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// groqResponse Groq API响应结构（支持/chat/completions）
type groqResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      groqMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int     `json:"prompt_tokens"`
		CompletionTokens int     `json:"completion_tokens"`
		TotalTokens      int     `json:"total_tokens"`
		PromptTime       float64 `json:"prompt_time"`
		CompletionTime   float64 `json:"completion_time"`
		TotalTime        float64 `json:"total_time"`
	} `json:"usage"`
}

// groqResponseFormat Groq /responses端点响应结构
type groqResponseFormat struct {
	OutputText string `json:"output_text"`
}

// NewGroqProvider 创建Groq Provider
func NewGroqProvider(config llm.ProviderConfig) (llm.Provider, error) {
	if config.APIKey == "" {
		return nil, llm.ErrInvalidAPIKey
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultGroqBaseURL
	}

	models := config.Models
	if len(models) == 0 {
		// 默认支持的模型
		models = []string{
			"openai/gpt-oss-20b",
			"llama-3.3-70b-versatile",
			"llama-3.1-8b-instant",
			"mixtral-8x7b-32768",
			"gemma2-9b-it",
		}
	}

	defaultModel := config.DefaultModel
	if defaultModel == "" {
		defaultModel = "llama-3.3-70b-versatile"
	}

	return &GroqProvider{
		apiKey:       config.APIKey,
		baseURL:      baseURL,
		client:       &http.Client{Timeout: 60 * time.Second},
		directClient: newDirectHTTPClient(),
		models:       models,
		defaultModel: defaultModel,
	}, nil
}

// Complete 实现聊天完成
func (g *GroqProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, *llm.RateLimitInfo, error) {
	startTime := time.Now()

	// 使用默认模型或请求中的模型
	model := req.Model
	if model == "" {
		model = g.defaultModel
	}

	var groqReq groqRequest
	var endpoint string

	// 根据模型选择不同的端点
	if model == "openai/gpt-oss-20b" {
		// 使用/responses端点（用户提供的示例格式）
		endpoint = "/responses"

		// 构建input内容
		var inputBuilder strings.Builder
		for _, msg := range req.Messages {
			inputBuilder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
		}

		groqReq = groqRequest{
			Model:       model,
			Input:       inputBuilder.String(),
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
			TopP:        req.TopP,
		}
	} else {
		// 使用/chat/completions端点（OpenAI兼容格式）
		endpoint = "/chat/completions"

		// 转换消息格式
		messages := make([]groqMessage, len(req.Messages))
		for i, msg := range req.Messages {
			messages[i] = groqMessage{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}

		groqReq = groqRequest{
			Model:       model,
			Messages:    messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
			TopP:        req.TopP,
		}
	}

	data, err := json.Marshal(groqReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := g.doRequest(ctx, endpoint, data)
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

	var rateLimit *llm.RateLimitInfo
	var content string

	// 根据端点使用不同的响应解析
	if endpoint == "/responses" {
		// 解析/responses端点响应
		var groqResp groqResponseFormat
		if err := json.Unmarshal(body, &groqResp); err != nil {
			return nil, nil, fmt.Errorf("failed to decode /responses response: %w", err)
		}
		content = groqResp.OutputText
	} else {
		// 解析/chat/completions端点响应
		var groqResp groqResponse
		if err := json.Unmarshal(body, &groqResp); err != nil {
			return nil, nil, fmt.Errorf("failed to decode /chat/completions response: %w", err)
		}

		if len(groqResp.Choices) == 0 {
			return nil, nil, fmt.Errorf("no choices in response")
		}
		content = groqResp.Choices[0].Message.Content
	}

	// 解析速率限制信息
	rateLimit = parseGroqRateLimit(resp.Header)

	return &llm.CompletionResponse{
		Content:          content,
		PromptTokens:     0, // /responses端点不返回token信息
		CompletionTokens: 0,
		TotalTokens:      0,
		Latency:          time.Since(startTime),
	}, rateLimit, nil
}

// GetModelList 获取支持的模型列表
func (g *GroqProvider) GetModelList() []string {
	return g.models
}

// GetProviderName 获取提供商名称
func (g *GroqProvider) GetProviderName() string {
	return groqProviderName
}

// HealthCheck 健康检查
func (g *GroqProvider) HealthCheck(ctx context.Context) error {
	// 发送一个简单的请求来检查服务是否可用
	req := llm.CompletionRequest{
		Model:     g.defaultModel,
		Messages:  []llm.Message{{Role: "user", Content: "Hi"}},
		MaxTokens: 5,
	}
	_, _, err := g.Complete(ctx, req)
	return err
}

func (g *GroqProvider) doRequest(ctx context.Context, endpoint string, data []byte) (*http.Response, error) {
	resp, err := g.doRequestWithClient(ctx, g.client, endpoint, data)
	if err == nil || !shouldRetryWithoutProxy(err) {
		return resp, err
	}
	return g.doRequestWithClient(ctx, g.directClient, endpoint, data)
}

func (g *GroqProvider) doRequestWithClient(ctx context.Context, client *http.Client, endpoint string, data []byte) (*http.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", g.baseURL+endpoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)

	return client.Do(httpReq)
}

func newDirectHTTPClient() *http.Client {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Client{Timeout: 60 * time.Second}
	}
	cloned := transport.Clone()
	cloned.Proxy = nil
	return &http.Client{
		Timeout:   60 * time.Second,
		Transport: cloned,
	}
}

func shouldRetryWithoutProxy(err error) bool {
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "proxyconnect") ||
		(strings.Contains(lower, "proxy") && strings.Contains(lower, "127.0.0.1")) ||
		(strings.Contains(lower, "proxy") && strings.Contains(lower, "localhost"))
}

// parseGroqRateLimit 解析Groq速率限制头
func parseGroqRateLimit(headers http.Header) *llm.RateLimitInfo {
	return &llm.RateLimitInfo{
		RequestsLimit:     parseIntHeader(headers, "x-ratelimit-limit-requests"),
		RequestsRemaining: parseIntHeader(headers, "x-ratelimit-remaining-requests"),
		TokensLimit:       parseIntHeader(headers, "x-ratelimit-limit-tokens"),
		TokensRemaining:   parseIntHeader(headers, "x-ratelimit-remaining-tokens"),
		ResetTime:         time.Unix(parseInt64Header(headers, "x-ratelimit-reset-requests"), 0),
	}
}

func parseIntHeader(headers http.Header, key string) int {
	val := headers.Get(key)
	if val == "" {
		return 0
	}
	var result int
	fmt.Sscanf(val, "%d", &result)
	return result
}

func parseInt64Header(headers http.Header, key string) int64 {
	val := headers.Get(key)
	if val == "" {
		return 0
	}
	var result int64
	fmt.Sscanf(val, "%d", &result)
	return result
}
