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
	// 获取API密钥，优先从环境变量读取
	apiKey := os.Getenv("SILICONFLOW_API_KEY")
	if apiKey == "" {
		log.Println("未设置 SILICONFLOW_API_KEY，跳过在线示例")
		return
	}

	fmt.Println("=== SiliconFlow Provider 示例 ===")

	// 方式1: 直接创建 SiliconFlow Provider
	fmt.Println("1. 直接创建 SiliconFlow Provider")
	config := llm.ProviderConfig{
		APIKey:       apiKey,
		BaseURL:      "https://api.siliconflow.cn/v1",
		DefaultModel: "Qwen/Qwen2.5-72B-Instruct",
		Models: []string{
			"Qwen/Qwen2.5-72B-Instruct",
			"deepseek-ai/DeepSeek-V2.5",
			"Qwen/Qwen2.5-7B-Instruct",
		},
	}

	provider, err := providers.NewSiliconFlowProvider(config)
	if err != nil {
		log.Fatalf("创建Provider失败: %v", err)
	}

	// 测试健康检查
	fmt.Println("   执行健康检查...")
	ctx := context.Background()
	if err := provider.HealthCheck(ctx); err != nil {
		log.Printf("   健康检查失败: %v", err)
	} else {
		fmt.Println("   ✓ 健康检查通过")
	}

	// 获取支持的模型列表
	fmt.Println("\n2. 支持的模型列表:")
	for _, model := range provider.GetModelList() {
		fmt.Printf("   - %s\n", model)
	}

	// 发送聊天请求
	fmt.Println("\n3. 发送聊天请求 (Qwen/Qwen2.5-72B-Instruct):")
	req := llm.CompletionRequest{
		Model: "Qwen/Qwen2.5-72B-Instruct",
		Messages: []llm.Message{
			{Role: "system", Content: "你是一个有帮助的AI助手。"},
			{Role: "user", Content: "有诺贝尔数学奖吗？"},
		},
		Temperature: 0.7,
		MaxTokens:   500,
	}

	resp, rateLimit, err := provider.Complete(ctx, req)
	if err != nil {
		log.Printf("   请求失败: %v", err)
	} else {
		fmt.Printf("   回答: %s\n", resp.Content)
		fmt.Printf("   Token使用: 提示词=%d, 完成=%d, 总计=%d\n",
			resp.PromptTokens, resp.CompletionTokens, resp.TotalTokens)
		fmt.Printf("   延迟: %v\n", resp.Latency)
		if rateLimit != nil {
			fmt.Printf("   速率限制: 剩余请求=%d, 剩余Token=%d\n",
				rateLimit.RequestsRemaining, rateLimit.TokensRemaining)
		}
	}

	// 使用另一个模型
	fmt.Println("\n4. 使用 DeepSeek-V2.5 模型:")
	req2 := llm.CompletionRequest{
		Model: "deepseek-ai/DeepSeek-V2.5",
		Messages: []llm.Message{
			{Role: "user", Content: "SiliconFlow公测上线，每用户送3亿token，对于整个大模型应用领域带来哪些改变？"},
		},
		Temperature: 0.8,
		MaxTokens:   800,
	}

	resp2, _, err := provider.Complete(ctx, req2)
	if err != nil {
		log.Printf("   请求失败: %v", err)
	} else {
		fmt.Printf("   回答: %s\n", resp2.Content)
		fmt.Printf("   Token使用: 总计=%d\n", resp2.TotalTokens)
	}

	// 方式2: 使用工厂函数创建
	fmt.Println("\n5. 使用工厂函数创建 Provider:")
	factoryConfig := llm.ProviderConfig{
		Name:         "siliconflow",
		APIKey:       apiKey,
		BaseURL:      "https://api.siliconflow.cn/v1",
		DefaultModel: "Qwen/Qwen2.5-72B-Instruct",
	}
	factoryProvider, err := llm.Create(factoryConfig)
	if err != nil {
		log.Fatalf("工厂创建失败: %v", err)
	}
	fmt.Printf("   ✓ 通过工厂创建成功: %s\n", factoryProvider.GetProviderName())

	fmt.Println("\n=== 示例完成 ===")
}
