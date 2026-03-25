package identity

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/atotto/clipboard"
)

const (
	// Domain 是邮箱域名
	Domain = "aiarchlab.xyz"
	// LogFile 是身份记录文件
	LogFile = "identities.log"
)

// GenerateNewEmail 生成一个新的随机邮箱
func GenerateNewEmail() string {
	// 初始化随机数生成器
	rand.Seed(time.Now().UnixNano())

	// 生成随机前缀
	prefix := fmt.Sprintf("dev_%s", generateRandomString(4))

	// 拼接邮箱地址
	email := fmt.Sprintf("%s@%s", prefix, Domain)

	// 复制到剪贴板
	clipboard.WriteAll(email)

	return email
}

// SaveIdentity 保存身份信息到日志文件
func SaveIdentity(email string, purpose string) error {
	// 打开日志文件，如果不存在则创建
	file, err := os.OpenFile(LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入身份信息
	_, err = fmt.Fprintf(file, "%s | %s | %s\n", time.Now().Format("2006-01-02 15:04:05"), email, purpose)
	return err
}

// generateRandomString 生成指定长度的随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}