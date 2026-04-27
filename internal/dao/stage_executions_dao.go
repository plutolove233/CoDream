package dao

import (
	"context"

	"github.com/plutolove233/co-dream/internal/dal/models"
	"github.com/plutolove233/co-dream/internal/database"
)

type StageExecutionsDao struct {
	models.StageExecution
}

func (c *StageExecutionsDao) Get(ctx context.Context) error {
	db := database.GetPostgreSqlDatabase()
	return db.DB().WithContext(ctx).Where(map[string]any{
		"is_deleted": false,
	}).Where(c).Take(c).Error
}

func (c *StageExecutionsDao) Add(ctx context.Context) error {
	db := database.GetPostgreSqlDatabase()
	return db.DB().WithContext(ctx).Create(&c).Error
}

func (c *StageExecutionsDao) Update(ctx context.Context, args map[string]any) error {
	db := database.GetPostgreSqlDatabase()
	err := c.Get(ctx)
	if err != nil {
		return err
	}
	return db.DB().WithContext(ctx).Model(&c).Updates(args).Error
}

func (c *StageExecutionsDao) Delete(ctx context.Context) ([]StageExecutionsDao, error) {
	db := database.GetPostgreSqlDatabase()
	stageExecutions := []StageExecutionsDao{}
	return stageExecutions, db.DB().WithContext(ctx).Model(&StageExecutionsDao{}).Where(map[string]any{
		"id":         c.ID,
		"is_deleted": false,
	}).Find(&stageExecutions).Error
}
