//go:build ignore

package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

const (
	qqEmail        = "your_qq_email@example.com"
	qqPassword     = "your_qq_imap_password"
	domain         = "aiarchlab.xyz" // 用户的域名，用于生成随机邮箱
	groqAPIURL     = "https://api.groq.com"
	groqConsoleURL = "https://console.groq.com"
)

type GroqAccount struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	APIKey   string `json:"api_key"`
	Created  string `json:"created"`
}

// 生成随机字符串
func generateRandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}

// 生成随机邮箱
func generateRandomEmail() string {
	randomPart := generateRandomString(8)
	return fmt.Sprintf("%s@%s", randomPart, domain)
}

// 发送注册请求
func registerGroqAccount(email, password string) error {
	fmt.Printf("Sending registration request for: %s\n", email)

	// 构建注册请求数据
	formData := url.Values{}
	formData.Set("email", email)
	formData.Set("password", password)
	formData.Set("action", "register")

	// 创建HTTP请求
	req, err := http.NewRequest("POST", groqConsoleURL+"/api/auth/register", strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send registration request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("Registration response status: %d\n", resp.StatusCode)
	fmt.Printf("Registration response body: %s\n", string(body))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("registration failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	fmt.Println("Registration request sent successfully!")
	return nil
}

// 从QQ邮箱获取确认邮件并点击链接
func confirmEmail(email string) error {
	fmt.Println("Checking QQ email for confirmation message...")

	// 连接到QQ邮箱IMAP服务器
	c, err := client.DialTLS("imap.qq.com:993", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	defer c.Logout()

	// 登录
	if err := c.Login(qqEmail, qqPassword); err != nil {
		return fmt.Errorf("failed to login to IMAP server: %w", err)
	}

	// 选择收件箱
	_, err = c.Select("INBOX", false)
	if err != nil {
		return fmt.Errorf("failed to select INBOX: %w", err)
	}

	// 搜索包含特定邮箱的邮件
	criteria := imap.NewSearchCriteria()
	criteria.Header.Add("To", qqEmail)
	criteria.Header.Add("From", "no-reply@groq.com")
	criteria.SentSince = time.Now().Add(-10 * time.Minute)

	ids, err := c.Search(criteria)
	if err != nil {
		return fmt.Errorf("failed to search emails: %w", err)
	}

	if len(ids) == 0 {
		return fmt.Errorf("no confirmation email found")
	}

	// 获取最新的邮件
	seqset := new(imap.SeqSet)
	seqset.AddNum(ids[len(ids)-1])

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchBody}, messages)
	}()

	msg := <-messages
	if err := <-done; err != nil {
		return fmt.Errorf("failed to fetch email: %w", err)
	}

	// 解析邮件内容
	var body string
	for _, l := range msg.Body {
		if l != nil {
			bodyBytes, err := ioutil.ReadAll(l)
			if err == nil {
				body += string(bodyBytes)
			}
		}
	}

	fmt.Printf("Found email with subject: %s\n", msg.Envelope.Subject)

	// 提取确认链接
	re := regexp.MustCompile(`https://console\.groq\.com/verify/[^"\s]+`)
	matches := re.FindStringSubmatch(body)
	if len(matches) == 0 {
		return fmt.Errorf("confirmation link not found in email")
	}

	confirmURL := matches[0]
	fmt.Printf("Found confirmation link: %s\n", confirmURL)

	// 访问确认链接
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(confirmURL)
	if err != nil {
		return fmt.Errorf("failed to access confirmation link: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Confirmation link accessed with status: %d\n", resp.StatusCode)

	if resp.StatusCode == http.StatusOK {
		fmt.Println("Email confirmed successfully!")
		return nil
	}

	return fmt.Errorf("confirmation failed with status: %d", resp.StatusCode)
}

// 生成API密钥
func generateAPIKey(email, password string) (string, error) {
	fmt.Println("Generating API key...")

	// 先登录获取token
	loginData := url.Values{}
	loginData.Set("email", email)
	loginData.Set("password", password)

	req, err := http.NewRequest("POST", groqConsoleURL+"/api/auth/login", strings.NewReader(loginData.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to login: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read login response: %w", err)
	}

	fmt.Printf("Login response status: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed: %s", string(body))
	}

	// 解析响应获取token
	var loginResp struct {
		Token string `json:"token"`
	}

	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", fmt.Errorf("failed to parse login response: %w", err)
	}

	if loginResp.Token == "" {
		return "", fmt.Errorf("no token found in login response")
	}

	fmt.Println("Login successful, got token")

	// 创建API密钥
	apiKeyData := url.Values{}
	apiKeyData.Set("name", "auto-generated-key")

	req, err = http.NewRequest("POST", groqConsoleURL+"/api/api-keys", strings.NewReader(apiKeyData.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create API key request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err = client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create API key: %w", err)
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read API key response: %w", err)
	}

	fmt.Printf("API key creation response status: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("API key creation failed: %s", string(body))
	}

	// 解析API密钥响应
	var apiKeyResp struct {
		Key string `json:"key"`
	}

	if err := json.Unmarshal(body, &apiKeyResp); err != nil {
		return "", fmt.Errorf("failed to parse API key response: %w", err)
	}

	if apiKeyResp.Key == "" {
		return "", fmt.Errorf("no API key found in response")
	}

	fmt.Printf("API Key generated: %s\n", apiKeyResp.Key)
	return apiKeyResp.Key, nil
}

// 保存账号信息到文件
func saveAccount(account GroqAccount) error {
	data, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal account data: %w", err)
	}

	// 创建accounts目录
	if err := os.MkdirAll("accounts", 0755); err != nil {
		return fmt.Errorf("failed to create accounts directory: %w", err)
	}

	// 保存到文件
	filename := fmt.Sprintf("accounts/groq_%s.json", strings.ReplaceAll(account.Email, "@", "_"))
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write account file: %w", err)
	}

	fmt.Printf("Account saved to: %s\n", filename)
	return nil
}

func main() {
	fmt.Println("=== Groq Account Auto-Registration (Simple Version) ===")

	// 生成随机账号信息
	email := generateRandomEmail()
	password := generateRandomString(12)

	fmt.Printf("Generated account:\n")
	fmt.Printf("  Email: %s\n", email)
	fmt.Printf("  Password: %s\n", password)

	// 步骤1: 注册账号
	fmt.Println("\nStep 1: Registering Groq account...")
	if err := registerGroqAccount(email, password); err != nil {
		fmt.Printf("Registration failed: %v\n", err)
		return
	}

	// 步骤2: 等待并确认邮件
	fmt.Println("\nStep 2: Waiting for confirmation email...")
	time.Sleep(5 * time.Second) // 等待邮件发送

	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		fmt.Printf("Checking email (%d/%d)...\n", i+1, maxRetries)
		if err := confirmEmail(email); err == nil {
			break
		}
		time.Sleep(10 * time.Second)
	}

	// 步骤3: 生成API密钥
	fmt.Println("\nStep 3: Generating API key...")
	apiKey, err := generateAPIKey(email, password)
	if err != nil {
		fmt.Printf("Failed to generate API key: %v\n", err)
		return
	}

	// 步骤4: 保存账号信息
	fmt.Println("\nStep 4: Saving account information...")
	account := GroqAccount{
		Email:    email,
		Password: password,
		APIKey:   apiKey,
		Created:  time.Now().Format(time.RFC3339),
	}

	if err := saveAccount(account); err != nil {
		fmt.Printf("Failed to save account: %v\n", err)
		return
	}

	fmt.Println("\n=== Registration completed successfully! ===")
	fmt.Printf("You can now use this API key: %s\n", apiKey)
}
