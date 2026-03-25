// LLM Provider 使用示例
// 展示如何在多智能体系统中使用不同的LLM Provider

package main

import (
	"context"
	"fmt"
	"os"

	"multi_agent_cooperation/agent"
	"multi_agent_cooperation/llm"
	_ "multi_agent_cooperation/llm/providers"
)

func main() {
	fmt.Println("=== LLM Provider 使用示例 ===")

	// 示例1: 使用Groq Provider
	fmt.Println("示例1: 使用Groq Provider")
	groqExample()

	// 示例2: 使用OpenAI Provider
	fmt.Println("\n示例2: 使用OpenAI Provider")
	openaiExample()

	// 示例3: 在Agent中使用
	fmt.Println("\n示例3: 在Agent中使用Provider")
	agentExample()
}

// groqExample 展示如何使用Groq Provider
func groqExample() {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		fmt.Println("  跳过: 未设置GROQ_API_KEY环境变量")
		return
	}

	provider, err := llm.Create(llm.ProviderConfig{
		Name:         "groq",
		APIKey:       apiKey,
		BaseURL:      "https://api.groq.com/openai/v1",
		DefaultModel: "llama-3.3-70b-versatile",
	})
	if err != nil {
		fmt.Printf("  创建Provider失败: %v\n", err)
		return
	}

	fmt.Printf("  Provider名称: %s\n", provider.GetProviderName())
	fmt.Printf("  支持模型: %v\n", provider.GetModelList())

	// 发送简单请求
	resp, rateLimit, err := provider.Complete(context.Background(), llm.CompletionRequest{
		Model: "llama-3.3-70b-versatile",
		Messages: []llm.Message{
			{Role: "system", Content: "你是一个简洁的助手"},
			{Role: "user", Content: "你好！请用一句话介绍自己。"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	})
	if err != nil {
		fmt.Printf("  请求失败: %v\n", err)
		return
	}

	fmt.Printf("  响应: %s\n", resp.Content)
	fmt.Printf("  Token使用: %d/%d\n", rateLimit.TokensRemaining, rateLimit.TokensLimit)
}

// openaiExample 展示如何使用OpenAI Provider
func openaiExample() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("  跳过: 未设置OPENAI_API_KEY环境变量")
		return
	}

	provider, err := llm.Create(llm.ProviderConfig{
		Name:         "openai",
		APIKey:       apiKey,
		BaseURL:      "https://api.openai.com/v1",
		DefaultModel: "gpt-3.5-turbo",
	})
	if err != nil {
		fmt.Printf("  创建Provider失败: %v\n", err)
		return
	}

	fmt.Printf("  Provider名称: %s\n", provider.GetProviderName())
	fmt.Printf("  支持模型: %v\n", provider.GetModelList())

	// 发送简单请求
	resp, rateLimit, err := provider.Complete(context.Background(), llm.CompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []llm.Message{
			{Role: "system", Content: "你是一个简洁的助手"},
			{Role: "user", Content: "你好！请用一句话介绍自己。"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	})
	if err != nil {
		fmt.Printf("  请求失败: %v\n", err)
		return
	}

	fmt.Printf("  响应: %s\n", resp.Content)
	fmt.Printf("  Token使用: %d/%d\n", rateLimit.TokensRemaining, rateLimit.TokensLimit)
}

// agentExample 展示如何在Agent中使用Provider
func agentExample() {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		fmt.Println("  跳过: 未设置GROQ_API_KEY环境变量")
		return
	}

	// 创建Provider
	provider, err := llm.Create(llm.ProviderConfig{
		Name:         "groq",
		APIKey:       apiKey,
		BaseURL:      "https://api.groq.com/openai/v1",
		DefaultModel: "llama-3.3-70b-versatile",
	})
	if err != nil {
		fmt.Printf("  创建Provider失败: %v\n", err)
		return
	}

	// 创建使用Provider的Agent
	architect := agent.NewArchitectAgent(provider)
	developer := agent.NewDeveloperAgent("开发者", "llama-3.3-70b-versatile", provider)
	critic := agent.NewCriticAgent(provider)

	fmt.Printf("  创建Agent成功:\n")
	fmt.Printf("    - %s (架构师)\n", architect.Agent.Name)
	fmt.Printf("    - %s (开发者)\n", developer.Agent.Name)
	fmt.Printf("    - %s (评审员)\n", critic.Agent.Name)

	// 展示Agent能力
	fmt.Printf("\n  Agent能力:\n")
	fmt.Printf("    - 架构师: 设计系统架构、API设计、技术选型\n")
	fmt.Printf("    - 开发者: 实现功能、优化代码、代码审查\n")
	fmt.Printf("    - 评审员: 评审架构、评审代码、评审任务拆分\n")
}
