package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/plutolove233/co-dream/internal/database"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// 创建 context
	ctx := context.Background()

	// 创建数据库配置
	config := database.NewConfig()
	log.Printf("Connecting to database: %s@%s:%s/%s\n",
		config.User, config.Host, config.Port, config.DBName)

	// 连接数据库
	db, err := database.NewDatabase(ctx, config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close(ctx)

	// 执行自动迁移
	log.Println("Running database migrations...")
	if err := database.AutoMigrate(ctx, db.DB()); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 创建额外的索引
	if err := database.CreateIndexes(ctx, db.DB()); err != nil {
		log.Fatalf("Failed to create indexes: %v", err)
	}

	log.Println("\n✅ Database setup completed successfully!")
	log.Println("All tables created and indexes established.")

	os.Exit(0)
}
