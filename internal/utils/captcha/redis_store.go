package captcha

import (
	"context"
	"fmt"
	"time"

	"github.com/plutolove233/co-dream/internal/database"
)

const (
	// CaptchaKeyPrefix Redis中验证码的key前缀
	CaptchaKeyPrefix = "captcha:email:"
	// CaptchaExpireTime 验证码过期时间（5分钟）
	CaptchaExpireTime = 5 * time.Minute
)

// StoreCaptcha 将验证码存储到Redis，key为 captcha:email:{email}
func StoreCaptcha(ctx context.Context, email, code string) error {
	redisDB := database.GetRedisDatabase()
	if redisDB == nil {
		return fmt.Errorf("Redis连接未初始化")
	}

	key := CaptchaKeyPrefix + email
	err := redisDB.Client().Set(ctx, key, code, CaptchaExpireTime).Err()
	if err != nil {
		return fmt.Errorf("存储验证码失败: %w", err)
	}

	return nil
}

// VerifyCaptcha 验证验证码是否正确
func VerifyCaptcha(ctx context.Context, email, code string) (bool, error) {
	redisDB := database.GetRedisDatabase()
	if redisDB == nil {
		return false, fmt.Errorf("Redis连接未初始化")
	}

	key := CaptchaKeyPrefix + email
	storedCode, err := redisDB.Client().Get(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("获取验证码失败: %w", err)
	}

	return storedCode == code, nil
}

// DeleteCaptcha 删除验证码（验证成功后删除）
func DeleteCaptcha(ctx context.Context, email string) error {
	redisDB := database.GetRedisDatabase()
	if redisDB == nil {
		return fmt.Errorf("Redis连接未初始化")
	}

	key := CaptchaKeyPrefix + email
	err := redisDB.Client().Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("删除验证码失败: %w", err)
	}

	return nil
}
