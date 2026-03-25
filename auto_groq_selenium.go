//go:build ignore

package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/tebeka/selenium"
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

// 使用Selenium注册Groq账号
func registerGroqAccount(email, password string) error {
	fmt.Println("=== Starting Selenium browser automation ===")

	// 设置Edge浏览器驱动路径
	edgeDriverPath := "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedgedriver.exe"

	// 检查驱动是否存在
	if _, err := os.Stat(edgeDriverPath); os.IsNotExist(err) {
		// 尝试其他位置
		edgeDriverPath = "C:\\WebDrivers\\msedgedriver.exe"
		if _, err := os.Stat(edgeDriverPath); os.IsNotExist(err) {
			return fmt.Errorf("Edge driver not found. Please download from: https://developer.microsoft.com/en-us/microsoft-edge/tools/webdriver/")
		}
	}

	// 设置Selenium服务器
	const (
		seleniumPort = 9515
	)

	// 创建Selenium服务
	service, err := selenium.NewChromeDriverService(edgeDriverPath, seleniumPort)
	if err != nil {
		return fmt.Errorf("failed to start selenium service: %w", err)
	}
	defer service.Stop()

	// 设置浏览器选项
	caps := selenium.Capabilities{
		"browserName": "MicrosoftEdge",
		"ms:edgeOptions": map[string]interface{}{
			"args": []string{
				"--start-maximized",
				"--disable-gpu",
				"--no-sandbox",
			},
		},
	}

	// 连接到Selenium
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", seleniumPort))
	if err != nil {
		return fmt.Errorf("failed to connect to selenium: %w", err)
	}
	defer wd.Quit()

	// 访问Groq控制台
	fmt.Printf("Navigating to: %s\n", groqConsoleURL)
	if err := wd.Get(groqConsoleURL); err != nil {
		return fmt.Errorf("failed to navigate: %w", err)
	}

	fmt.Println("Page loaded successfully")

	// 等待邮箱输入框出现并输入邮箱
	fmt.Println("Entering email...")
	emailInput, err := wd.FindElement(selenium.ByCSSSelector, "input[type='email']")
	if err != nil {
		return fmt.Errorf("failed to find email input: %w", err)
	}

	if err := emailInput.SendKeys(email); err != nil {
		return fmt.Errorf("failed to fill email: %w", err)
	}

	// 点击"Continue with email"按钮
	fmt.Println("Clicking 'Continue with email' button...")
	continueButton, err := wd.FindElement(selenium.ByCSSSelector, "button:has-text('Continue with email')")
	if err != nil {
		// 尝试其他选择器
		continueButton, err = wd.FindElement(selenium.ByXPath, "//button[contains(text(), 'Continue with email')]")
		if err != nil {
			return fmt.Errorf("failed to find continue button: %w", err)
		}
	}

	if err := continueButton.Click(); err != nil {
		return fmt.Errorf("failed to click continue button: %w", err)
	}

	// 等待密码输入框出现并输入密码
	fmt.Println("Waiting for password page...")
	time.Sleep(3 * time.Second)

	passwordInput, err := wd.FindElement(selenium.ByCSSSelector, "input[type='password']")
	if err != nil {
		return fmt.Errorf("failed to find password input: %w", err)
	}

	fmt.Println("Entering password...")
	if err := passwordInput.SendKeys(password); err != nil {
		return fmt.Errorf("failed to fill password: %w", err)
	}

	// 点击继续按钮
	fmt.Println("Clicking continue button...")
	continueButton2, err := wd.FindElement(selenium.ByCSSSelector, "button:has-text('Continue')")
	if err != nil {
		continueButton2, err = wd.FindElement(selenium.ByXPath, "//button[contains(text(), 'Continue')]")
		if err != nil {
			return fmt.Errorf("failed to find continue button: %w", err)
		}
	}

	if err := continueButton2.Click(); err != nil {
		return fmt.Errorf("failed to click continue button: %w", err)
	}

	fmt.Println("Registration request sent successfully!")
	fmt.Println("Browser will remain open for 30 seconds...")
	time.Sleep(30 * time.Second)

	return nil
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

// 使用Selenium生成API密钥
func generateAPIKey(email, password string) (string, error) {
	fmt.Println("=== Generating API key ===")

	// 设置Edge浏览器驱动路径
	edgeDriverPath := "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedgedriver.exe"

	// 检查驱动是否存在
	if _, err := os.Stat(edgeDriverPath); os.IsNotExist(err) {
		edgeDriverPath = "C:\\WebDrivers\\msedgedriver.exe"
		if _, err := os.Stat(edgeDriverPath); os.IsNotExist(err) {
			return "", fmt.Errorf("Edge driver not found. Please download from: https://developer.microsoft.com/en-us/microsoft-edge/tools/webdriver/")
		}
	}

	// 设置Selenium服务器
	const (
		seleniumPort = 9515
	)

	// 创建Selenium服务
	service, err := selenium.NewChromeDriverService(edgeDriverPath, seleniumPort)
	if err != nil {
		return "", fmt.Errorf("failed to start selenium service: %w", err)
	}
	defer service.Stop()

	// 设置浏览器选项
	caps := selenium.Capabilities{
		"browserName": "MicrosoftEdge",
		"ms:edgeOptions": map[string]interface{}{
			"args": []string{
				"--start-maximized",
				"--disable-gpu",
				"--no-sandbox",
			},
		},
	}

	// 连接到Selenium
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", seleniumPort))
	if err != nil {
		return "", fmt.Errorf("failed to connect to selenium: %w", err)
	}
	defer wd.Quit()

	// 访问Groq控制台
	if err := wd.Get(groqConsoleURL); err != nil {
		return "", fmt.Errorf("failed to navigate: %w", err)
	}

	// 登录账号
	fmt.Println("Logging in...")

	// 输入邮箱
	emailInput, err := wd.FindElement(selenium.ByCSSSelector, "input[type='email']")
	if err != nil {
		return "", fmt.Errorf("failed to find email input: %w", err)
	}

	if err := emailInput.SendKeys(email); err != nil {
		return "", fmt.Errorf("failed to fill email: %w", err)
	}

	// 点击继续按钮
	continueButton, err := wd.FindElement(selenium.ByXPath, "//button[contains(text(), 'Continue with email')]")
	if err != nil {
		return "", fmt.Errorf("failed to find continue button: %w", err)
	}

	if err := continueButton.Click(); err != nil {
		return "", fmt.Errorf("failed to click continue button: %w", err)
	}

	// 输入密码
	time.Sleep(3 * time.Second)
	passwordInput, err := wd.FindElement(selenium.ByCSSSelector, "input[type='password']")
	if err != nil {
		return "", fmt.Errorf("failed to find password input: %w", err)
	}

	if err := passwordInput.SendKeys(password); err != nil {
		return "", fmt.Errorf("failed to fill password: %w", err)
	}

	// 点击继续按钮
	continueButton2, err := wd.FindElement(selenium.ByXPath, "//button[contains(text(), 'Continue')]")
	if err != nil {
		return "", fmt.Errorf("failed to find continue button: %w", err)
	}

	if err := continueButton2.Click(); err != nil {
		return "", fmt.Errorf("failed to click continue button: %w", err)
	}

	// 等待登录成功
	fmt.Println("Waiting for login to complete...")
	time.Sleep(5 * time.Second)

	// 导航到API密钥页面
	fmt.Println("Navigating to API Keys page...")

	// 点击API Keys链接
	apiKeysLink, err := wd.FindElement(selenium.ByXPath, "//a[contains(text(), 'API Keys')]")
	if err != nil {
		apiKeysLink, err = wd.FindElement(selenium.ByCSSSelector, "a[href*='/api-keys']")
		if err != nil {
			return "", fmt.Errorf("failed to find API Keys link: %w", err)
		}
	}

	if err := apiKeysLink.Click(); err != nil {
		return "", fmt.Errorf("failed to click API Keys link: %w", err)
	}

	// 等待API密钥页面加载
	time.Sleep(3 * time.Second)

	// 点击创建API密钥按钮
	fmt.Println("Creating API key...")
	createButton, err := wd.FindElement(selenium.ByXPath, "//button[contains(text(), 'Create API Key')]")
	if err != nil {
		createButton, err = wd.FindElement(selenium.ByXPath, "//button[contains(text(), 'New API Key')]")
		if err != nil {
			return "", fmt.Errorf("failed to find create API key button: %w", err)
		}
	}

	if err := createButton.Click(); err != nil {
		return "", fmt.Errorf("failed to click create API key button: %w", err)
	}

	// 等待对话框出现
	time.Sleep(2 * time.Second)

	// 输入API密钥名称
	nameInput, err := wd.FindElement(selenium.ByCSSSelector, "input[placeholder*='Name']")
	if err != nil {
		nameInput, err = wd.FindElement(selenium.ByCSSSelector, "input[placeholder*='name']")
		if err != nil {
			return "", fmt.Errorf("failed to find name input: %w", err)
		}
	}

	if err := nameInput.SendKeys("auto-generated-key"); err != nil {
		return "", fmt.Errorf("failed to fill name: %w", err)
	}

	// 点击创建按钮
	confirmButton, err := wd.FindElement(selenium.ByXPath, "//button[contains(text(), 'Create')]")
	if err != nil {
		confirmButton, err = wd.FindElement(selenium.ByXPath, "//button[contains(text(), 'Generate')]")
		if err != nil {
			return "", fmt.Errorf("failed to find confirm button: %w", err)
		}
	}

	if err := confirmButton.Click(); err != nil {
		return "", fmt.Errorf("failed to click confirm button: %w", err)
	}

	// 等待API密钥生成
	time.Sleep(3 * time.Second)

	// 获取API密钥
	fmt.Println("Extracting API key...")
	apiKeyElement, err := wd.FindElement(selenium.ByCSSSelector, "code")
	if err != nil {
		apiKeyElement, err = wd.FindElement(selenium.ByCSSSelector, "input[readonly]")
		if err != nil {
			return "", fmt.Errorf("failed to find API key element: %w", err)
		}
	}

	apiKey, err := apiKeyElement.Text()
	if err != nil {
		return "", fmt.Errorf("failed to get API key text: %w", err)
	}

	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", fmt.Errorf("API key is empty")
	}

	fmt.Printf("API Key generated: %s\n", apiKey)
	return apiKey, nil
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
	fmt.Println("=== Groq Account Auto-Registration (Fully Automated) ===")

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

	// 步骤3: 访问确认链接
	fmt.Println("\nStep 3: Confirming email...")

	// 设置Edge浏览器驱动路径
	edgeDriverPath := "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedgedriver.exe"
	if _, err := os.Stat(edgeDriverPath); os.IsNotExist(err) {
		edgeDriverPath = "C:\\WebDrivers\\msedgedriver.exe"
		if _, err := os.Stat(edgeDriverPath); os.IsNotExist(err) {
			fmt.Printf("Edge driver not found. Please download from: https://developer.microsoft.com/en-us/microsoft-edge/tools/webdriver/\n")
			return
		}
	}

	// 设置Selenium服务器
	const (
		seleniumPort = 9515
	)

	// 创建Selenium服务
	service, err := selenium.NewChromeDriverService(edgeDriverPath, seleniumPort)
	if err != nil {
		fmt.Printf("Failed to start selenium service: %v\n", err)
		return
	}
	defer service.Stop()

	// 设置浏览器选项
	caps := selenium.Capabilities{
		"browserName": "MicrosoftEdge",
	}

	// 连接到Selenium
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", seleniumPort))
	if err != nil {
		fmt.Printf("Failed to connect to selenium: %v\n", err)
		return
	}
	defer wd.Quit()

	fmt.Printf("Accessing confirmation link: %s\n", confirmURL)
	if err := wd.Get(confirmURL); err != nil {
		fmt.Printf("Failed to access confirmation link: %v\n", err)
		return
	}

	fmt.Println("Email confirmed successfully!")
	time.Sleep(3 * time.Second)

	// 步骤4: 生成API密钥
	fmt.Println("\nStep 4: Generating API key...")
	apiKey, err := generateAPIKey(email, password)
	if err != nil {
		fmt.Printf("Failed to generate API key: %v\n", err)
		return
	}

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
	fmt.Printf("You can now use this API key: %s\n", apiKey)
}
