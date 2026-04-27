package dao

import (
	"context"

	"github.com/plutolove233/co-dream/internal/dal/models"
	"github.com/plutolove233/co-dream/internal/database"
)

type CheckpointsDao struct {
	models.Checkpoint
}

func (c *CheckpointsDao) Get(ctx context.Context) error {
	db := database.GetPostgreSqlDatabase()
	return db.DB().WithContext(ctx).Where(map[string]any{
		"is_deleted": false,
	}).Where(c).Take(c).Error
}

func (c *CheckpointsDao) Add(ctx context.Context) error {
	db := database.GetPostgreSqlDatabase()
	return db.DB().WithContext(ctx).Create(&c).Error
}

func (c *CheckpointsDao) Update(ctx context.Context, args map[string]any) error {
	db := database.GetPostgreSqlDatabase()
	err := c.Get(ctx)
	if err != nil {
		return err
	}
	return db.DB().WithContext(ctx).Model(&c).Updates(args).Error
}

func (c *CheckpointsDao) Delete(ctx context.Context) ([]CheckpointsDao, error) {
	db := database.GetPostgreSqlDatabase()
	checkpoints := []CheckpointsDao{}
	return checkpoints, db.DB().WithContext(ctx).Model(&CheckpointsDao{}).Where(map[string]any{
		"id":         c.ID,
		"is_deleted": false,
	}).Find(&checkpoints).Error
}
