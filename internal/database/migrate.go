package database

import (
	"context"
	"fmt"
	"log"

	"github.com/plutolove233/co-dream/internal/models"
	"gorm.io/gorm"
)

// AutoMigrate 执行数据库自动迁移，创建或更新表结构。
func AutoMigrate(ctx context.Context, db *gorm.DB) error {
	log.Println("Starting database migration...")

	err := db.WithContext(ctx).AutoMigrate(
		&models.Pipeline{},
		&models.PipelineExecution{},
		&models.StageExecution{},
		&models.Checkpoint{},
		&models.AgentTask{},
	)

	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database migration completed successfully")
	return nil
}

// CreateIndexes 创建额外的数据库索引以优化查询性能。
func CreateIndexes(ctx context.Context, db *gorm.DB) error {
	log.Println("Creating additional indexes...")

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_pipeline_executions_status_created ON pipeline_executions(status, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_stage_executions_execution_status ON stage_executions(execution_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_checkpoints_execution_status ON checkpoints(execution_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_agent_tasks_stage_status ON agent_tasks(stage_execution_id, status)",
	}

	for _, idx := range indexes {
		if err := db.WithContext(ctx).Exec(idx).Error; err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	log.Println("Indexes created successfully")
	return nil
}
