package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-resty/resty/v2"
)

// GroqRequest 定义 Groq API 请求结构
type GroqRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

// GroqResponse 定义 Groq API 响应结构
type GroqResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Logprobs     interface{} `json:"logprobs"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		QueueTime        float64 `json:"queue_time"`
		PromptTokens     int     `json:"prompt_tokens"`
		PromptTime       float64 `json:"prompt_time"`
		CompletionTokens int     `json:"completion_tokens"`
		CompletionTime   float64 `json:"completion_time"`
		TotalTokens      int     `json:"total_tokens"`
		TotalTime        float64 `json:"total_time"`
	} `json:"usage"`
	SystemFingerprint string `json:"system_fingerprint"`
	XGroq             struct {
		ID string `json:"id"`
	} `json:"x_groq"`
}

func main() {
	// 配置 API
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		log.Println("GROQ_API_KEY 未设置，跳过示例")
		return
	}
	baseURL := "https://api.groq.com/openai/v1/chat/completions"
	model := "llama-3.3-70b-versatile"

	// 创建 resty 客户端
	client := resty.New()

	// 构建请求
	request := GroqRequest{
		Model: model,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "user",
				Content: "请用中文解释快速语言模型的重要性",
			},
		},
	}

	// 发送请求
	var response GroqResponse
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+apiKey).
		SetHeader("Content-Type", "application/json").
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36").
		SetHeader("Accept", "application/json, text/plain, */*").
		SetHeader("Accept-Language", "en-US,en;q=0.9").
		SetHeader("Accept-Encoding", "identity"). // 禁用压缩
		SetHeader("Connection", "keep-alive").
		SetHeader("Origin", "https://console.groq.com").
		SetHeader("Referer", "https://console.groq.com/").
		SetBody(request).
		SetResult(&response).
		SetError(&map[string]interface{}{}).
		Post(baseURL)

	if err != nil {
		log.Fatalf("请求失败: %v", err)
	}

	// 检查响应状态码
	if resp.StatusCode() != http.StatusOK {
		fmt.Println("=== 详细错误信息 ===")
		fmt.Println("状态码:", resp.StatusCode())
		fmt.Println("状态:", resp.Status())
		fmt.Println("响应头:", resp.Header())
		fmt.Println("响应内容:", resp.String())
		fmt.Println("===================")
		log.Fatalf("API 请求失败")
	}

	// 输出结果
	fmt.Println("✅ Groq API 连接成功！")
	fmt.Println("ID:", response.ID)
	fmt.Println("模型:", response.Model)
	fmt.Println("状态码:", resp.StatusCode())

	// 输出速率限制信息
	fmt.Println("\n📊 速率限制信息:")
	if limitRequests := resp.Header().Get("x-ratelimit-limit-requests"); limitRequests != "" {
		fmt.Printf("请求限制: %s 次/分钟\n", limitRequests)
	}
	if remainingRequests := resp.Header().Get("x-ratelimit-remaining-requests"); remainingRequests != "" {
		fmt.Printf("剩余请求: %s 次\n", remainingRequests)
	}
	if limitTokens := resp.Header().Get("x-ratelimit-limit-tokens"); limitTokens != "" {
		fmt.Printf("Token 限制: %s tokens/分钟\n", limitTokens)
	}
	if remainingTokens := resp.Header().Get("x-ratelimit-remaining-tokens"); remainingTokens != "" {
		fmt.Printf("剩余 Token: %s tokens\n", remainingTokens)
	}
	if reset := resp.Header().Get("x-ratelimit-reset-requests"); reset != "" {
		fmt.Printf("请求重置时间: %s\n", reset)
	}
	if resetTokens := resp.Header().Get("x-ratelimit-reset-tokens"); resetTokens != "" {
		fmt.Printf("Token 重置时间: %s\n", resetTokens)
	}

	// 输出响应内容
	fmt.Println("\n📝 模型响应:")
	if len(response.Choices) > 0 {
		fmt.Println(response.Choices[0].Message.Content)
	} else {
		fmt.Println("未收到响应内容")
	}

	// 输出使用统计
	fmt.Println("\n📈 本次请求使用统计:")
	fmt.Printf("提示词 tokens: %d\n", response.Usage.PromptTokens)
	fmt.Printf("完成 tokens: %d\n", response.Usage.CompletionTokens)
	fmt.Printf("总 tokens: %d\n", response.Usage.TotalTokens)
	fmt.Printf("总时间: %.3f 秒\n", response.Usage.TotalTime)
	fmt.Printf("队列时间: %.3f 秒\n", response.Usage.QueueTime)
	fmt.Printf("提示词处理时间: %.3f 秒\n", response.Usage.PromptTime)
	fmt.Printf("完成处理时间: %.3f 秒\n", response.Usage.CompletionTime)
}
