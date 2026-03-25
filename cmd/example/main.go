package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"multi_agent_cooperation/pkg/identity"
	"multi_agent_cooperation/pkg/verifier"
)

// Config 是配置结构
type Config struct {
	IMAP verifier.Config `yaml:"imap"`
}

func main() {
	// 加载配置
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 生成新邮箱
	email := identity.GenerateNewEmail()
	fmt.Printf("已生成新邮箱: %s\n", email)
	fmt.Println("邮箱已复制到剪贴板")

	// 保存身份信息
	purpose := "SiliconFlow Registration"
	if err := identity.SaveIdentity(email, purpose); err != nil {
		fmt.Printf("保存身份信息失败: %v\n", err)
	}

	// 提示用户
	fmt.Printf("请前往官网输入该邮箱：%s\n", email)
	fmt.Println("正在等待验证码...")

	// 抓取验证码
	senderFilter := "SiliconFlow"
	code, err := verifier.FetchLatestCode(email, senderFilter, config.IMAP)
	if err != nil {
		fmt.Printf("抓取验证码失败: %v\n", err)
		os.Exit(1)
	}

	// 高亮显示验证码
	fmt.Printf("\033[32m验证码: %s\033[0m\n", code)
	fmt.Println("流程完成！")
}

// loadConfig 加载配置文件
func loadConfig() (Config, error) {
	var config Config

	// 读取配置文件
	data, err := os.ReadFile("imap_config.yaml")
	if err != nil {
		return config, err
	}

	// 解析 YAML
	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, err
	}

	return config, nil
}
