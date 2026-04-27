package dao

import (
	"context"

	"github.com/plutolove233/co-dream/internal/dal/models"
	"github.com/plutolove233/co-dream/internal/database"
)

type PipelineExecutionsDao struct {
	models.PipelineExecution
}

func (d *PipelineExecutionsDao) Get(ctx context.Context) error {
	db := database.GetPostgreSqlDatabase()
	return db.DB().WithContext(ctx).Where(map[string]any{
		"is_deleted": false,
	}).Where(d).Take(d).Error
}

func (d *PipelineExecutionsDao) Add(ctx context.Context) error {
	db := database.GetPostgreSqlDatabase()
	return db.DB().WithContext(ctx).Create(&d).Error
}

func (d *PipelineExecutionsDao) Update(ctx context.Context, args map[string]any) error {
	db := database.GetPostgreSqlDatabase()
	err := d.Get(ctx)
	if err != nil {
		return err
	}
	return db.DB().WithContext(ctx).Model(&d).Updates(args).Error
}

func (d *PipelineExecutionsDao) Delete(ctx context.Context) ([]PipelineExecutionsDao, error) {
	db := database.GetPostgreSqlDatabase()
	pipelineExecutions := []PipelineExecutionsDao{}
	return pipelineExecutions, db.DB().WithContext(ctx).Model(&PipelineExecutionsDao{}).Where(map[string]any{
		"id":         d.ID,
		"is_deleted": false,
	}).Find(&pipelineExecutions).Error
}
