//go:build ignore

package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

// 获取确认链接
func getConfirmationLink() (string, error) {
	fmt.Println("Checking QQ email for Groq confirmation...")

	// 连接到QQ邮箱IMAP服务器
	c, err := client.DialTLS("imap.qq.com:993", nil)
	if err != nil {
		return "", fmt.Errorf("failed to connect to IMAP: %w", err)
	}
	defer c.Logout()

	// 登录
	if err := c.Login(qqEmail, qqPassword); err != nil {
		return "", fmt.Errorf("failed to login: %w", err)
	}

	// 选择收件箱
	_, err = c.Select("INBOX", false)
	if err != nil {
		return "", fmt.Errorf("failed to select INBOX: %w", err)
	}

	// 搜索最近的Groq邮件
	criteria := imap.NewSearchCriteria()
	criteria.Header.Add("From", "no-reply@groq.com")
	criteria.SentSince = time.Now().Add(-30 * time.Minute)

	ids, err := c.Search(criteria)
	if err != nil {
		return "", fmt.Errorf("failed to search emails: %w", err)
	}

	if len(ids) == 0 {
		return "", fmt.Errorf("no Groq emails found")
	}

	// 获取最新的邮件
	seqset := new(imap.SeqSet)
	seqset.AddNum(ids[len(ids)-1])

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchBody}, messages)
	}()

	msg := <-messages
	if err := <-done; err != nil {
		return "", fmt.Errorf("failed to fetch email: %w", err)
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

	// 提取确认链接
	re := regexp.MustCompile(`https://console\.groq\.com/verify/[^\s]+`)
	matches := re.FindStringSubmatch(body)
	if len(matches) == 0 {
		return "", fmt.Errorf("confirmation link not found")
	}

	confirmURL := matches[0]
	fmt.Printf("Found confirmation link: %s\n", confirmURL)
	return confirmURL, nil
}

// 保存账号信息
func saveAccount(email, password, apiKey string) error {
	account := GroqAccount{
		Email:    email,
		Password: password,
		APIKey:   apiKey,
		Created:  time.Now().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal account: %w", err)
	}

	// 创建accounts目录
	if err := os.MkdirAll("accounts", 0755); err != nil {
		return fmt.Errorf("failed to create accounts directory: %w", err)
	}

	// 保存到文件
	filename := fmt.Sprintf("accounts/groq_%s.json", strings.ReplaceAll(email, "@", "_"))
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write account file: %w", err)
	}

	fmt.Printf("Account saved to: %s\n", filename)
	return nil
}

func main() {
	fmt.Println("=== Groq Account Auto-Registration ===")

	// 生成随机账号信息
	email := generateRandomEmail()
	password := generateRandomString(12)

	fmt.Printf("Generated account:\n")
	fmt.Printf("  Email: %s\n", email)
	fmt.Printf("  Password: %s\n", password)

	// 步骤1: 提示用户手动注册
	fmt.Println("\nStep 1: Please manually register using this email and password")
	fmt.Printf("Go to: %s\n", groqConsoleURL)
	fmt.Printf("Use email: %s\n", email)
	fmt.Printf("Use password: %s\n", password)
	fmt.Println("After registration, check your QQ email for confirmation")

	// 步骤2: 等待用户确认注册完成
	fmt.Println("\nPress Enter when you have completed the registration...")
	var input string
	fmt.Scanln(&input)

	// 步骤3: 自动确认邮箱
	fmt.Println("\nStep 2: Auto-confirming email...")
	maxRetries := 20
	var confirmURL string
	var err error

	for i := 0; i < maxRetries; i++ {
		fmt.Printf("Checking email (%d/%d)...\n", i+1, maxRetries)
		confirmURL, err = getConfirmationLink()
		if err == nil {
			break
		}
		fmt.Printf("Failed to find confirmation link: %v\n", err)
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		fmt.Printf("Failed to find confirmation link after %d attempts: %v\n", maxRetries, err)
		return
	}

	// 访问确认链接
	fmt.Println("Accessing confirmation link...")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(confirmURL)
	if err != nil {
		fmt.Printf("Failed to access confirmation link: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Confirmation link accessed with status: %d\n", resp.StatusCode)

	if resp.StatusCode == http.StatusOK {
		fmt.Println("Email confirmed successfully!")
	} else {
		fmt.Printf("Confirmation failed with status: %d\n", resp.StatusCode)
		return
	}

	// 步骤4: 提示用户生成API密钥
	fmt.Println("\nStep 3: Please generate API key manually")
	fmt.Println("1. Login to Groq console")
	fmt.Println("2. Go to API Keys section")
	fmt.Println("3. Create a new API key")
	fmt.Println("4. Copy the API key")
	fmt.Println("\nEnter the API key:")
	var apiKey string
	fmt.Scanln(&apiKey)

	// 步骤5: 保存账号信息
	fmt.Println("\nStep 4: Saving account information...")
	if err := saveAccount(email, password, apiKey); err != nil {
		fmt.Printf("Failed to save account: %v\n", err)
		return
	}

	fmt.Println("\n=== Registration completed successfully! ===")
	fmt.Printf("Email: %s\n", email)
	fmt.Printf("Password: %s\n", password)
	fmt.Printf("API Key: %s\n", apiKey)
	fmt.Println("Account information saved to accounts directory")
}
