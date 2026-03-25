package groq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ChatCompletionRequest 定义聊天完成请求结构
type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message 定义消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse 定义聊天完成响应结构
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 定义选择结构
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage 定义使用统计结构
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	PromptTime       int `json:"prompt_time"`
	CompletionTokens int `json:"completion_tokens"`
	CompletionTime   int `json:"completion_time"`
	TotalTokens      int `json:"total_tokens"`
	TotalTime        int `json:"total_time"`
}

// RateLimitInfo 定义速率限制信息结构
type RateLimitInfo struct {
	RequestsLimit  int   `json:"requests_limit"`
	RequestsRemaining int `json:"requests_remaining"`
	TokensLimit    int   `json:"tokens_limit"`
	TokensRemaining int `json:"tokens_remaining"`
	Reset          int64 `json:"reset"`
}

// Client 定义 Groq 客户端
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient 创建新的 Groq 客户端
func NewClient(apiKey string) *Client {
	return &Client{
		BaseURL: "https://api.groq.com/openai/v1/chat/completions",
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ChatCompletion 实现聊天完成接口
func (c *Client) ChatCompletion(model string, messages []Message) (string, *ChatCompletionResponse, *RateLimitInfo, error) {
	req := ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(data))
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil, nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", nil, nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", nil, nil, fmt.Errorf("no choices in response")
	}

	// 解析速率限制信息
	rateLimitInfo := &RateLimitInfo{
		RequestsLimit:  parseHeader(resp.Header, "x-ratelimit-limit-requests"),
		RequestsRemaining: parseHeader(resp.Header, "x-ratelimit-remaining-requests"),
		TokensLimit:    parseHeader(resp.Header, "x-ratelimit-limit-tokens"),
		TokensRemaining: parseHeader(resp.Header, "x-ratelimit-remaining-tokens"),
		Reset:          parseResetHeader(resp.Header, "x-ratelimit-reset-requests"),
	}

	return chatResp.Choices[0].Message.Content, &chatResp, rateLimitInfo, nil
}

// parseHeader 解析HTTP头
func parseHeader(headers http.Header, key string) int {
	val := headers.Get(key)
	if val == "" {
		return 0
	}
	var result int
	fmt.Sscanf(val, "%d", &result)
	return result
}

// parseResetHeader 解析重置时间头
func parseResetHeader(headers http.Header, key string) int64 {
	val := headers.Get(key)
	if val == "" {
		return 0
	}
	var result int64
	fmt.Sscanf(val, "%d", &result)
	return result
}

// PrintUsageStats 打印使用统计信息
func PrintUsageStats(response *ChatCompletionResponse, rateLimit *RateLimitInfo) {
	fmt.Println("\n=== 使用统计 ===")
	fmt.Printf("提示词 Tokens: %d\n", response.Usage.PromptTokens)
	fmt.Printf("完成 Tokens: %d\n", response.Usage.CompletionTokens)
	fmt.Printf("总 Tokens: %d\n", response.Usage.TotalTokens)
	fmt.Printf("总时间: %.3f 秒\n", float64(response.Usage.TotalTime)/1000)
	fmt.Printf("队列时间: %.3f 秒\n", float64(response.Usage.PromptTime)/1000)
	fmt.Printf("提示词处理时间: %.3f 秒\n", float64(response.Usage.PromptTime)/1000)
	fmt.Printf("完成处理时间: %.3f 秒\n", float64(response.Usage.CompletionTime)/1000)
	
	fmt.Println("\n=== 速率限制信息 ===")
	fmt.Printf("请求限制: %d 次/分钟\n", rateLimit.RequestsLimit)
	fmt.Printf("剩余请求: %d 次\n", rateLimit.RequestsRemaining)
	fmt.Printf("Token 限制: %d tokens/分钟\n", rateLimit.TokensLimit)
	fmt.Printf("剩余 Token: %d tokens\n", rateLimit.TokensRemaining)
	if rateLimit.Reset > 0 {
		resetTime := time.Unix(rateLimit.Reset, 0)
		fmt.Printf("重置时间: %s\n", resetTime.Format("2006-01-02 15:04:05"))
	}
}
