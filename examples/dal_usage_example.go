package main

import (
	"context"
	"log"

	"github.com/plutolove233/co-dream/internal/database"
	"github.com/plutolove233/co-dream/internal/dal/gen"
	"github.com/plutolove233/co-dream/internal/dal/models"
)

func main() {
	// 1. 初始化数据库连接
	ctx := context.Background()
	config := database.NewConfig()
	db, err := database.NewDatabase(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close(ctx)

	// 2. 设置默认数据库连接
	gen.SetDefault(db.DB())

	// 现在可以使用生成的查询代码了
	exampleCRUD(ctx)
}

func exampleCRUD(ctx context.Context) {
	// 获取查询对象
	p := gen.Pipeline
	pe := gen.PipelineExecution

	// ========== CREATE 创建 ==========
	log.Println("=== CREATE 示例 ===")

	// 创建一个新的 Pipeline
	newPipeline := &models.Pipeline{
		Name:        "示例流水线",
		Description: "这是一个示例流水线",
		Config: models.PipelineConfig{
			Name:        "示例配置",
			Description: "示例配置描述",
			Stages: []models.StageConfig{
				{
					Name:      "阶段1",
					Order:     1,
					AgentType: "planner",
					Model:     "claude-sonnet-4",
				},
			},
		},
		Status:    models.PipelineStatusPending,
		CreatedBy: "user123",
	}

	err := p.WithContext(ctx).Create(newPipeline)
	if err != nil {
		log.Printf("创建失败: %v", err)
		return
	}
	log.Printf("创建成功，Pipeline ID: %s", newPipeline.ID)

	// ========== READ 查询 ==========
	log.Println("\n=== READ 示例 ===")

	// 1. 根据ID查询单条记录
	pipeline, err := p.WithContext(ctx).Where(p.ID.Eq(newPipeline.ID)).First()
	if err != nil {
		log.Printf("查询失败: %v", err)
		return
	}
	log.Printf("查询到 Pipeline: %s", pipeline.Name)

	// 2. 查询多条记录（带条件）
	pipelines, err := p.WithContext(ctx).
		Where(p.Status.Eq(string(models.PipelineStatusPending))).
		Order(p.CreatedAt.Desc()).
		Limit(10).
		Find()
	if err != nil {
		log.Printf("查询列表失败: %v", err)
		return
	}
	log.Printf("查询到 %d 条 pending 状态的 Pipeline", len(pipelines))

	// 3. 复杂查询（多条件组合）
	pipelines, err = p.WithContext(ctx).
		Where(
			p.Status.In(string(models.PipelineStatusPending), string(models.PipelineStatusRunning)),
			p.CreatedBy.Eq("user123"),
		).
		Or(p.Name.Like("%示例%")).
		Find()
	if err != nil {
		log.Printf("复杂查询失败: %v", err)
		return
	}
	log.Printf("复杂查询结果: %d 条", len(pipelines))

	// ========== UPDATE 更新 ==========
	log.Println("\n=== UPDATE 示例 ===")

	// 1. 更新单个字段
	_, err = p.WithContext(ctx).
		Where(p.ID.Eq(newPipeline.ID)).
		Update(p.Status, string(models.PipelineStatusRunning))
	if err != nil {
		log.Printf("更新失败: %v", err)
		return
	}
	log.Println("更新状态成功")

	// 2. 更新多个字段
	_, err = p.WithContext(ctx).
		Where(p.ID.Eq(newPipeline.ID)).
		Updates(map[string]interface{}{
			"status":      string(models.PipelineStatusCompleted),
			"description": "已完成的示例流水线",
		})
	if err != nil {
		log.Printf("批量更新失败: %v", err)
		return
	}
	log.Println("批量更新成功")

	// ========== DELETE 删除 ==========
	log.Println("\n=== DELETE 示例 ===")

	// 软删除（如果模型有 DeletedAt 字段）
	_, err = p.WithContext(ctx).Where(p.ID.Eq(newPipeline.ID)).Delete()
	if err != nil {
		log.Printf("删除失败: %v", err)
		return
	}
	log.Println("软删除成功")

	// ========== 关联查询 ==========
	log.Println("\n=== 关联查询示例 ===")

	// 创建一个 Execution
	execution := &models.PipelineExecution{
		PipelineID: newPipeline.ID,
		Status:     models.ExecutionStatusPending,
		Input: models.ExecutionInput{
			Requirement: "测试需求",
			Context:     map[string]interface{}{"key": "value"},
		},
	}
	err = pe.WithContext(ctx).Create(execution)
	if err != nil {
		log.Printf("创建 Execution 失败: %v", err)
		return
	}

	// 查询 Pipeline 及其关联的 Executions
	pipelineWithExecs, err := p.WithContext(ctx).
		Preload(p.Executions).
		Where(p.ID.Eq(newPipeline.ID)).
		First()
	if err != nil {
		log.Printf("关联查询失败: %v", err)
		return
	}
	log.Printf("Pipeline 有 %d 个 Executions", len(pipelineWithExecs.Executions))

	// ========== 聚合查询 ==========
	log.Println("\n=== 聚合查询示例 ===")

	// 统计数量
	count, err := p.WithContext(ctx).
		Where(p.Status.Eq(string(models.PipelineStatusPending))).
		Count()
	if err != nil {
		log.Printf("统计失败: %v", err)
		return
	}
	log.Printf("Pending 状态的 Pipeline 数量: %d", count)

	log.Println("\n=== 所有示例完成 ===")
}
