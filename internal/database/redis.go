package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// RedisConfig 包含Redis连接配置参数
type RedisConfig struct {
	Host     string
	Port     string
	Password string
}

// RedisDatabase 封装了Redis连接实例
type RedisDatabase struct {
	client *redis.Client
}

var (
	RedisDB *RedisDatabase
)

// NewRedisConfig 创建一个新的Redis配置实例，从配置文件读取配置值
func NewRedisConfig() *RedisConfig {
	return &RedisConfig{
		Host:     viper.GetString("redis.RedisHost"),
		Port:     viper.GetString("redis.RedisPort"),
		Password: viper.GetString("redis.RedisPassword"),
	}
}

// InitRedisDatabase 初始化Redis连接
func InitRedisDatabase(ctx context.Context, config *RedisConfig) error {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", config.Host, config.Port),
		Password:     config.Password,
		DB:           0, // 使用默认数据库
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// 验证连接
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Println("Redis connection established successfully")
	RedisDB = &RedisDatabase{client: client}
	return nil
}

// Client 返回底层的 redis.Client 实例
func (d *RedisDatabase) Client() *redis.Client {
	return d.client
}

// Close 关闭Redis连接
func (d *RedisDatabase) Close(ctx context.Context) error {
	return d.client.Close()
}

// GetRedisDatabase 获取Redis数据库实例
func GetRedisDatabase() *RedisDatabase {
	return RedisDB
}
