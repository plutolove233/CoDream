package dao

import (
	"context"

	"github.com/plutolove233/co-dream/internal/dal/models"
	"github.com/plutolove233/co-dream/internal/database"
)

type AgentTasksDao struct {
	models.AgentTask
}

func (d *AgentTasksDao) Get(ctx context.Context) error {
	db := database.GetPostgreSqlDatabase()
	return db.DB().WithContext(ctx).Where(map[string]any{
		"is_deleted": false,
	}).Where(d).Take(d).Error
}

func (u *AgentTasksDao) Add(ctx context.Context) error {
	db := database.GetPostgreSqlDatabase()
	return db.DB().WithContext(ctx).Create(&u).Error
}

func (u *AgentTasksDao) Update(ctx context.Context, args map[string]any) error {
	db := database.GetPostgreSqlDatabase()
	err := u.Get(ctx)
	if err != nil {
		return err
	}
	return db.DB().WithContext(ctx).Model(&u).Updates(args).Error
}

func (u *AgentTasksDao) Delete(ctx context.Context) ([]AgentTasksDao, error) {
	db := database.GetPostgreSqlDatabase()
	user := []AgentTasksDao{}
	return user, db.DB().WithContext(ctx).Model(&AgentTasksDao{}).Where(map[string]any{
		"id":         u.ID,
		"is_deleted": false,
	}).Find(&user).Error
}
