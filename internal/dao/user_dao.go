package dao

import (
	"context"

	"github.com/plutolove233/co-dream/internal/dal/models"
	"github.com/plutolove233/co-dream/internal/database"
)

type UserDao struct {
	models.User
}

func (u *UserDao) Get(ctx context.Context) error {
	db := database.GetPostgreSqlDatabase()
	return db.DB().WithContext(ctx).Where(map[string]any{
		"is_deleted": false,
	}).Where(u).Take(u).Error
}

func (u *UserDao) Add(ctx context.Context) error {
	db := database.GetPostgreSqlDatabase()
	return db.DB().WithContext(ctx).Create(&u).Error
}

func (u *UserDao) Update(ctx context.Context, args map[string]any) error {
	db := database.GetPostgreSqlDatabase()
	err := u.Get(ctx)
	if err != nil {
		return err
	}
	return db.DB().WithContext(ctx).Model(&u).Updates(args).Error
}

func (u *UserDao) Delete(ctx context.Context) ([]UserDao, error) {
	db := database.GetPostgreSqlDatabase()
	user := []UserDao{}
	return user, db.DB().WithContext(ctx).Model(&UserDao{}).Where(map[string]any{
		"id":         u.ID,
		"is_deleted": false,
	}).Find(&user).Error
}
