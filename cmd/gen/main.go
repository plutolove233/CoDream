package main

import (
	"context"
	"log"

	"github.com/plutolove233/co-dream/internal/database"
	"github.com/plutolove233/co-dream/internal/dal/models"
	"gorm.io/gen"
)

func main() {
	// 初始化数据库连接
	ctx := context.Background()
	config := database.NewConfig()
	db, err := database.NewDatabase(ctx, config)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close(ctx)

	// 配置代码生成器
	g := gen.NewGenerator(gen.Config{
		OutPath:           "internal/dal/gen",
		Mode:              gen.WithDefaultQuery | gen.WithQueryInterface,
		FieldNullable:     true,
		FieldCoverable:    true,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
	})

	g.UseDB(db.DB())

	// 从现有模型生成代码
	g.ApplyBasic(
		models.Pipeline{},
		models.PipelineExecution{},
		models.StageExecution{},
		models.AgentTask{},
		models.Checkpoint{},
	)

	// 执行代码生成
	g.Execute()

	log.Println("Code generation completed successfully!")
}
