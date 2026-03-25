package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func httpClient() {
	// 配置 API
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		fmt.Println("GROQ_API_KEY 未设置，跳过 httpClient 示例")
		return
	}
	baseURL := "https://api.groq.com/openai/v1/chat/completions"

	// 构建请求数据
	requestData := map[string]interface{}{
		"model": "llama-3.3-70b-versatile",
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": "Explain the importance of fast language models",
			},
		},
	}

	// 编码请求数据
	data, err := json.Marshal(requestData)
	if err != nil {
		fmt.Printf("编码请求数据失败: %v\n", err)
		return
	}

	// 创建请求
	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(data))
	if err != nil {
		fmt.Printf("创建请求失败: %v\n", err)
		return
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "identity") // 禁用压缩
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Origin", "https://console.groq.com")
	req.Header.Set("Referer", "https://console.groq.com/")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Sec-Ch-Ua", "\"Not_A Brand\";v=\"8\", \"Chromium\";v=\"123\", \"Google Chrome\";v=\"123\"")
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", "\"Windows\"")

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
				MinVersion:         tls.VersionTLS12,
			},
			DisableKeepAlives: false,
			MaxIdleConns:      100,
			IdleConnTimeout:   90 * time.Second,
		},
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("发送请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		return
	}

	// 输出响应
	fmt.Printf("状态码: %d\n", resp.StatusCode)
	fmt.Printf("状态: %s\n", resp.Status)
	fmt.Println("响应头:")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}
	fmt.Printf("响应内容: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		fmt.Println("\n❌ API 请求失败！")
		fmt.Println("可能的原因:")
		fmt.Println("1. API Key 无效或已过期")
		fmt.Println("2. API Key 权限不足")
		fmt.Println("3. IP 地址被封禁")
		fmt.Println("4. 网络连接问题")
		fmt.Println("5. 代理设置问题")
	} else {
		fmt.Println("\n✅ API 请求成功！")
		// 解析响应
		var response map[string]interface{}
		if err := json.Unmarshal(body, &response); err == nil {
			if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if message, ok := choice["message"].(map[string]interface{}); ok {
						if content, ok := message["content"].(string); ok {
							fmt.Println("\n模型响应:")
							fmt.Println(content)
						}
					}
				}
			}
		}
	}
}
