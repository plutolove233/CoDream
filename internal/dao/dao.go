package dao

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity 定义了所有实体必须实现的接口
type Entity interface {
	GetID() uuid.UUID
	SetID(uuid.UUID)
}

// DAO 定义了通用的数据访问接口
type DAO[T Entity] interface {
	// 基础 CRUD 操作
	Create(ctx context.Context, entity *T) error
	FindByID(ctx context.Context, id uuid.UUID) (*T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, entity *T) error

	// 扩展查询操作
	FindAll(ctx context.Context) ([]*T, error)
	FindByCondition(ctx context.Context, conditionFunc func(db *gorm.DB) *gorm.DB) ([]*T, error)
	Count(ctx context.Context) (int64, error)
	Exists(ctx context.Context, id uuid.UUID) (bool, error)

	// 分页操作
	Paginate(ctx context.Context, page, pageSize int) ([]*T, int64, error)

	// 事务支持
	Transaction(ctx context.Context, fn func(txDAO DAO[T]) error) error
}
