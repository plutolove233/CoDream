package dao

import (
	"errors"
	"fmt"
	"sync"

	"gorm.io/gorm"
)

// daoFactory 是 DAO 工厂的实现
type daoFactory struct {
	db   *gorm.DB
	daos sync.Map // 缓存 DAO 实例
}

// newDAOFactory 创建新的 DAO 工厂
func newDAOFactory(db *gorm.DB) (*daoFactory, error) {
	if db == nil {
		return nil, errors.New("database connection cannot be nil")
	}
	return &daoFactory{
		db: db,
	}, nil
}

// getDAO 获取指定类型的 DAO 实例
func (f *daoFactory) getDAO(typeName string, creator func() interface{}) interface{} {
	// 尝试从缓存获取
	if cached, ok := f.daos.Load(typeName); ok {
		return cached
	}

	// 创建新的 DAO 实例
	dao := creator()

	// 存入缓存
	f.daos.Store(typeName, dao)

	return dao
}

// Close 关闭工厂
func (f *daoFactory) Close() error {
	// 清空缓存
	f.daos.Range(func(key, value interface{}) bool {
		f.daos.Delete(key)
		return true
	})
	return nil
}

// 全局工厂实例
var globalFactory *daoFactory

// InitDAOFactory 初始化全局 DAO 工厂
func InitDAOFactory(db *gorm.DB) error {
	var err error
	globalFactory, err = newDAOFactory(db)
	return err
}

// GetDAO 获取指定类型的 DAO 实例
func GetDAO[T Entity]() DAO[T] {
	if globalFactory == nil {
		panic("DAO factory not initialized, call InitDAOFactory first")
	}

	// 使用类型名作为缓存键
	var zero T
	typeName := fmt.Sprintf("%T", zero)

	// 使用工厂的 getDAO 方法
	dao := globalFactory.getDAO(typeName, func() interface{} {
		return NewBaseDAO[T](globalFactory.db)
	})

	return dao.(DAO[T])
}
