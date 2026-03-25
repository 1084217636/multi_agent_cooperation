package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ChatCompletionRequest 定义聊天完成请求结构
type ChatCompletionRequest struct {
	Model    string    `json:"model"`    // 模型名称
	Messages []Message `json:"messages"` // 消息列表
}

// Message 定义消息结构
type Message struct {
	Role    string `json:"role"`    // 角色
	Content string `json:"content"` // 内容
}

// ChatCompletionResponse 定义聊天完成响应结构
type ChatCompletionResponse struct {
	Choices []Choice `json:"choices"` // 选择列表
}

// Choice 定义选择结构
type Choice struct {
	Message Message `json:"message"` // 消息
}

// Client 定义 LLM 客户端
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient 创建新的 LLM 客户端
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ChatCompletion 实现聊天完成接口
func (c *Client) ChatCompletion(model string, messages []Message) (string, error) {
	req := ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}