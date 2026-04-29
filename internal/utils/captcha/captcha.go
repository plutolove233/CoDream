package captcha

import (
	"strings"
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateCode 生成指定长度的数字验证码
func GenerateCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("验证码长度必须大于0")
	}

	var code strings.Builder
	for range length {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("生成验证码失败: %w", err)
		}
		code.WriteString(num.String())
	}

	return code.String(), nil
}

// GenerateEmailCode 生成6位数字邮箱验证码
func GenerateEmailCode() (string, error) {
	return GenerateCode(6)
}
