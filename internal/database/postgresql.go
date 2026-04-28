package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgresqlConfig 包含数据库连接配置参数。
type PostgresqlConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

// PostgresqlDatabase 封装了数据库连接实例。
type PostgresqlDatabase struct {
	db *gorm.DB
}

var (
	PostgreSqlDB *PostgresqlDatabase
)

// NewPostgreSqlConfig 创建一个新的数据库配置实例，从环境变量读取配置值。
func NewPostgreSqlConfig() *PostgresqlConfig {
	return &PostgresqlConfig{
		Host:     viper.GetString("PostgreSQL.PostgreSQLHost"),
		Port:     viper.GetString("PostgreSQL.PostgreSQLPort"),
		User:     viper.GetString("PostgreSQL.PostgreSQLUser"),
		Password: viper.GetString("PostgreSQL.PostgreSQLPassword"),
		DBName:   viper.GetString("PostgreSQL.PostgreSQLDBName"),
	}
}

// DSN 返回 PostgreSQL 数据源名称字符串。
func (c *PostgresqlConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName,
	)
}

// NewDatabase 创建一个新的数据库连接实例。
func InitPostgreSqlDatabase(ctx context.Context, config *PostgresqlConfig) error {
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(postgres.Open(config.DSN()), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 使用 context 验证数据库连接
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection established successfully")
	PostgreSqlDB = &PostgresqlDatabase{db: db}
	return nil
}

// DB 返回底层的 gorm.DB 实例。
func (d *PostgresqlDatabase) DB() *gorm.DB {
	return d.db
}

// Close 关闭数据库连接。
func (d *PostgresqlDatabase) Close(ctx context.Context) error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	return sqlDB.Close()
}

func GetPostgreSqlDatabase() *PostgresqlDatabase {
	return PostgreSqlDB
}
