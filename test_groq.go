//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"multi_agent_cooperation/llm"
	"multi_agent_cooperation/llm/providers"
)

func main() {
	// 获取API密钥
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		log.Println("GROQ_API_KEY 未设置，跳过示例")
		return
	}

	// 创建Groq Provider配置
	config := llm.ProviderConfig{
		APIKey: apiKey,
		Models: []string{
			"openai/gpt-oss-20b",
			"llama-3.3-70b-versatile",
		},
		DefaultModel: "openai/gpt-oss-20b",
	}

	// 创建Groq Provider
	groqProvider, err := providers.NewGroqProvider(config)
	if err != nil {
		log.Fatalf("Failed to create Groq provider: %v", err)
	}

	fmt.Println("Groq provider created successfully")
	fmt.Println("Supported models:", groqProvider.GetModelList())

	// 测试/responses端点（openai/gpt-oss-20b模型）
	fmt.Println("\n=== Testing /responses endpoint (openai/gpt-oss-20b) ===")
	req1 := llm.CompletionRequest{
		Model: "openai/gpt-oss-20b",
		Messages: []llm.Message{
			{
				Role:    "user",
				Content: "Explain the importance of fast language models",
			},
		},
		MaxTokens: 100,
	}

	ctx := context.Background()
	resp1, _, err := groqProvider.Complete(ctx, req1)
	if err != nil {
		fmt.Printf("Error with /responses endpoint: %v\n", err)
	} else {
		fmt.Printf("Response from /responses endpoint:\n%s\n", resp1.Content)
	}

	// 测试/chat/completions端点（llama-3.3-70b-versatile模型）
	fmt.Println("\n=== Testing /chat/completions endpoint (llama-3.3-70b-versatile) ===")
	req2 := llm.CompletionRequest{
		Model: "llama-3.3-70b-versatile",
		Messages: []llm.Message{
			{
				Role:    "user",
				Content: "Hello, what can you do?",
			},
		},
		MaxTokens: 50,
	}

	resp2, _, err := groqProvider.Complete(ctx, req2)
	if err != nil {
		fmt.Printf("Error with /chat/completions endpoint: %v\n", err)
	} else {
		fmt.Printf("Response from /chat/completions endpoint:\n%s\n", resp2.Content)
	}
}
