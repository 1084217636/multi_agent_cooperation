//go:build ignore

package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime"
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

// 使用系统默认浏览器打开URL
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}

// 从QQ邮箱获取确认邮件并点击链接
func confirmEmail(email string) (string, error) {
	fmt.Println("=== Checking QQ email for confirmation ===")

	// 连接到QQ邮箱IMAP服务器
	c, err := client.DialTLS("imap.qq.com:993", nil)
	if err != nil {
		return "", fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	defer c.Logout()

	// 登录
	if err := c.Login(qqEmail, qqPassword); err != nil {
		return "", fmt.Errorf("failed to login to IMAP server: %w", err)
	}

	// 选择收件箱
	_, err = c.Select("INBOX", false)
	if err != nil {
		return "", fmt.Errorf("failed to select INBOX: %w", err)
	}

	// 搜索包含特定邮箱的邮件
	criteria := imap.NewSearchCriteria()
	criteria.Header.Add("To", qqEmail)
	criteria.Header.Add("From", "no-reply@groq.com")
	criteria.SentSince = time.Now().Add(-30 * time.Minute)

	ids, err := c.Search(criteria)
	if err != nil {
		return "", fmt.Errorf("failed to search emails: %w", err)
	}

	if len(ids) == 0 {
		return "", fmt.Errorf("no confirmation email found")
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
		return "", fmt.Errorf("failed to fetch email: %w", err)
	}

	fmt.Printf("Found email with subject: %s\n", msg.Envelope.Subject)

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
		return "", fmt.Errorf("confirmation link not found in email")
	}

	confirmURL := matches[0]
	fmt.Printf("Found confirmation link: %s\n", confirmURL)
	return confirmURL, nil
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
	fmt.Println("=== Groq Account Auto-Registration (Final Version) ===")

	// 生成随机账号信息
	email := generateRandomEmail()
	password := generateRandomString(12)

	fmt.Printf("Generated account:\n")
	fmt.Printf("  Email: %s\n", email)
	fmt.Printf("  Password: %s\n", password)

	// 步骤1: 打开浏览器并导航到Groq控制台
	fmt.Println("\nStep 1: Opening Groq console in browser...")
	if err := openBrowser(groqConsoleURL); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
		return
	}

	// 步骤2: 提示用户手动完成注册
	fmt.Println("\nStep 2: Please complete registration manually")
	fmt.Println("1. Use the email and password above to register")
	fmt.Println("2. Complete the registration process")
	fmt.Println("3. Check your QQ email for confirmation")
	fmt.Println("\nPress Enter when you have completed registration...")

	var input string
	fmt.Scanln(&input)

	// 步骤3: 自动确认邮件
	fmt.Println("\nStep 3: Auto-confirming email...")
	maxRetries := 20
	var confirmURL string
	var err error

	for i := 0; i < maxRetries; i++ {
		fmt.Printf("Checking email (%d/%d)...\n", i+1, maxRetries)
		confirmURL, err = confirmEmail(email)
		if err == nil {
			break
		}
		fmt.Printf("Failed to find confirmation link: %v\n", err)
		time.Sleep(10 * time.Second)
	}

	if err != nil {
		fmt.Printf("Failed to find confirmation link after %d attempts: %v\n", maxRetries, err)
		return
	}

	// 打开确认链接
	fmt.Printf("Opening confirmation link: %s\n", confirmURL)
	if err := openBrowser(confirmURL); err != nil {
		fmt.Printf("Failed to open confirmation link: %v\n", err)
		return
	}

	fmt.Println("Email confirmed successfully!")
	time.Sleep(2 * time.Second)

	// 步骤4: 提示用户生成API密钥
	fmt.Println("\nStep 4: Please generate API key")
	fmt.Println("1. Login to Groq console")
	fmt.Println("2. Go to API Keys section")
	fmt.Println("3. Create a new API key")
	fmt.Println("4. Copy the API key")
	fmt.Println("\nEnter the API key:")

	var apiKey string
	fmt.Scanln(&apiKey)

	// 步骤5: 保存账号信息
	fmt.Println("\nStep 5: Saving account information...")
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
	fmt.Printf("Email: %s\n", email)
	fmt.Printf("Password: %s\n", password)
	fmt.Printf("API Key: %s\n", apiKey)
	fmt.Println("Account information saved to accounts directory")
}
