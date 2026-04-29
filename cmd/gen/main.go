package main

import (
	"context"

	"github.com/joho/godotenv"
	"github.com/plutolove233/co-dream/internal/database"
	"github.com/plutolove233/co-dream/internal/setting"
	"gorm.io/gen"
)

func main() {
	godotenv.Load()
	setting.InitViper()
	cfg := database.NewPostgreSqlConfig()
	// 初始化数据库连接
	database.InitPostgreSqlDatabase(context.Background(), cfg)
	db := database.GetPostgreSqlDatabase()
	defer db.Close(context.Background())

	// 创建生成器实例
	g := gen.NewGenerator(gen.Config{
		// 输出目录
		OutPath: "internal/dal/query",
		// 模型目录
		ModelPkgPath: "internal/dal/models",
		// 生成模式
		Mode:              gen.WithDefaultQuery | gen.WithQueryInterface,
		FieldNullable:     true,
		FieldCoverable:    true,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
	})

	// 使用数据库连接
	g.UseDB(db.DB())

	g.ApplyBasic(g.GenerateAllTable()...)

	// 执行生成
	g.Execute()
}
