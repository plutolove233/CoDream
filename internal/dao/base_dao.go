package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseDAO 是 DAO 接口的通用实现
type BaseDAO[T Entity] struct {
	db *gorm.DB
}

// NewBaseDAO 创建新的 BaseDAO 实例
func NewBaseDAO[T Entity](db *gorm.DB) *BaseDAO[T] {
	return &BaseDAO[T]{db: db}
}

// Create 创建新记录
func (d *BaseDAO[T]) Create(ctx context.Context, entity *T) error {
	if entity == nil {
		return ErrEntityNil
	}
	result := d.db.WithContext(ctx).Create(entity)
	if result.Error != nil {
		return fmt.Errorf("failed to create entity: %w", result.Error)
	}
	return nil
}

// FindByID 根据 ID 查询单条记录
func (d *BaseDAO[T]) FindByID(ctx context.Context, id uuid.UUID) (*T, error) {
	var entity T
	result := d.db.WithContext(ctx).Where("id = ?", id).First(&entity)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to find entity: %w", result.Error)
	}
	return &entity, nil
}

// Update 更新记录
func (d *BaseDAO[T]) Update(ctx context.Context, entity *T) error {
	if entity == nil {
		return ErrEntityNil
	}
	result := d.db.WithContext(ctx).Save(entity)
	if result.Error != nil {
		return fmt.Errorf("failed to update entity: %w", result.Error)
	}
	return nil
}

// Delete 软删除记录（从 entity 的主键字段获取 ID）
func (d *BaseDAO[T]) Delete(ctx context.Context, entity *T) error {
	if entity == nil {
		return ErrEntityNil
	}
	result := d.db.WithContext(ctx).Delete(entity)
	if result.Error != nil {
		return fmt.Errorf("failed to delete entity: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// FindAll 查询所有记录
func (d *BaseDAO[T]) FindAll(ctx context.Context) ([]*T, error) {
	var entities []*T
	result := d.db.WithContext(ctx).Find(&entities)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find all entities: %w", result.Error)
	}
	return entities, nil
}

// FindByCondition 根据条件查询
func (d *BaseDAO[T]) FindByCondition(ctx context.Context, conditionFunc func(db *gorm.DB) *gorm.DB) ([]*T, error) {
	var entities []*T
	// 使用 GORM 的查询接口
	query := d.db.WithContext(ctx)
	query = conditionFunc(query)
	result := query.Find(&entities)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find entities by condition: %w", result.Error)
	}
	return entities, nil
}

// Count 统计记录数
func (d *BaseDAO[T]) Count(ctx context.Context) (int64, error) {
	var count int64
	result := d.db.WithContext(ctx).Model(new(T)).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count entities: %w", result.Error)
	}
	return count, nil
}

// Exists 检查记录是否存在
func (d *BaseDAO[T]) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int64
	result := d.db.WithContext(ctx).Model(new(T)).Where("id = ?", id).Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("failed to check existence: %w", result.Error)
	}
	return count > 0, nil
}

// Paginate 分页查询
func (d *BaseDAO[T]) Paginate(ctx context.Context, page, pageSize int) ([]*T, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 10
	}

	var total int64
	var items []*T

	// 统计总数
	countResult := d.db.WithContext(ctx).Model(new(T)).Count(&total)
	if countResult.Error != nil {
		return nil, 0, fmt.Errorf("failed to count total: %w", countResult.Error)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	queryResult := d.db.WithContext(ctx).Offset(offset).Limit(pageSize).Find(&items)
	if queryResult.Error != nil {
		return nil, 0, fmt.Errorf("failed to paginate: %w", queryResult.Error)
	}

	return items, total, nil
}

// Transaction 事务支持
func (d *BaseDAO[T]) Transaction(ctx context.Context, fn func(txDAO DAO[T]) error) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDAO := NewBaseDAO[T](tx)
		return fn(txDAO)
	})
}
