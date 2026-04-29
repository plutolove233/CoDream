package email

import (
	"encoding/base64"
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"sync"
)

type SMTPClient struct {
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
}

var (
	SMTPClientInstance *SMTPClient
	once               sync.Once
)

/**
 * @Description 通过smtp发送邮件
 * @param to 接收方邮箱，如有多个，以;隔开
 * @param subject 邮件主题
 * @param format 发送格式，可选html和plain
 * @param body
 * @return error 返回错误
 */
func (s *SMTPClient) SMTPSendEmail(userNikeName, to, subject, format, body string) error {
	auth := smtp.PlainAuth("", s.SMTPUser, s.SMTPPass, s.SMTPHost)

	bs64 := base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/")
	header := make(map[string]string)
	header["From"] = userNikeName + "<" + s.SMTPUser + ">"
	header["To"] = to
	header["Subject"] = fmt.Sprintf("=?UTF-8?B?%s?=", bs64.EncodeToString([]byte(subject)))
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/" + format + "; charset=UTF-8"
	header["Content-Transfer-Encoding"] = "base64"

	var data strings.Builder
	for k, v := range header {
		data.WriteString(k + ": " + v + "\r\n")
	}
	data.WriteString("\r\n" + bs64.EncodeToString([]byte(body)))
	sendTo := strings.Split(to, ";")

	err := smtp.SendMail(s.SMTPHost+":"+s.SMTPPort, auth, s.SMTPUser, sendTo, []byte(data.String()))
	return err
}

func GetSMTPClient() *SMTPClient {
	once.Do(func() {
		SMTPClientInstance = &SMTPClient{
			SMTPHost: os.Getenv("SMTP_HOST"),
			SMTPPort: os.Getenv("SMTP_PORT"),
			SMTPUser: os.Getenv("SMTP_USER"),
			SMTPPass: os.Getenv("SMTP_PASSWORD"),
		}
	})
	return SMTPClientInstance
}
