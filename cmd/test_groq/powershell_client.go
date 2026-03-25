package main

import (
	"fmt"
	"os"
	"os/exec"
)

func powershellClient() {
	// 配置 API
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		fmt.Println("GROQ_API_KEY 未设置，跳过 PowerShell 示例")
		return
	}
	model := "llama-3.3-70b-versatile"
	content := "Explain the importance of fast language models"

	// 构建 PowerShell 命令
	psCommand := fmt.Sprintf(`
	$headers = @{"Content-Type"="application/json"; "Authorization"="Bearer %s"}
	$body = '{"model": "%s", "messages": [{"role": "user", "content": "%s"}]}'
	Invoke-WebRequest -Uri "https://api.groq.com/openai/v1/chat/completions" -Method POST -Headers $headers -Body $body -UseBasicParsing
	`, apiKey, model, content)

	// 执行 PowerShell 命令
	cmd := exec.Command("powershell.exe", "-Command", psCommand)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("执行命令失败: %v\n", err)
		fmt.Printf("输出: %s\n", string(output))
		return
	}

	// 输出结果
	fmt.Println("✅ PowerShell 命令执行成功！")
	fmt.Println("输出:")
	fmt.Println(string(output))
}
