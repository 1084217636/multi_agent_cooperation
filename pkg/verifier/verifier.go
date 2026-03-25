package verifier

import (
	"fmt"
	"regexp"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// Config 是 IMAP 配置
type Config struct {
	Username string
	Password string
	Server   string
	Port     int
}

// FetchLatestCode 从 QQ 邮箱抓取最新的验证码
func FetchLatestCode(targetEmail string, senderFilter string, config Config) (string, error) {
	// 连接到 IMAP 服务器
	addr := fmt.Sprintf("%s:%d", config.Server, config.Port)
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return "", err
	}
	defer c.Logout()

	// 登录
	if err := c.Login(config.Username, config.Password); err != nil {
		return "", err
	}

	// 选择收件箱
	_, err = c.Select("INBOX", false)
	if err != nil {
		return "", err
	}

	// 设置超时
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// 循环检查邮件
	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("timeout: no verification code received within 2 minutes")
		case <-ticker.C:
			// 搜索符合条件的邮件
			criteria := imap.NewSearchCriteria()
			criteria.Header.Add("To", targetEmail)
			if senderFilter != "" {
				criteria.Header.Add("From", senderFilter)
			}

			// 搜索邮件
			ids, err := c.Search(criteria)
			if err != nil {
				return "", err
			}

			if len(ids) > 0 {
				// 获取最新的邮件
				n := len(ids)
				set := &imap.SeqSet{}
				set.AddNum(ids[n-1])

				// 获取邮件内容
				items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchBody}
				ch := make(chan *imap.Message, 1)
				if err := c.Fetch(set, items, ch); err != nil {
					return "", err
				}

				msg := <-ch
				if msg == nil {
					return "", fmt.Errorf("no message found")
				}

				// 解析邮件内容
				for _, part := range msg.Body {
					if part == nil {
						continue
					}

					// 读取邮件内容
					buf := make([]byte, 1024)
					n, _ := part.Read(buf)
					body := string(buf[:n])

					// 提取验证码
					code := extractVerificationCode(body)
					if code != "" {
						return code, nil
					}
				}
			}
		}
	}
}

// extractVerificationCode 从邮件正文中提取6位数字验证码
func extractVerificationCode(body string) string {
	// 匹配6位数字
	re := regexp.MustCompile(`\b\d{6}\b`)
	matches := re.FindStringSubmatch(body)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}
