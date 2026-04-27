package dao

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestEntity 是一个测试用的实体
type TestEntity struct {
	ID   uuid.UUID
	Name string
}

func (t TestEntity) GetID() uuid.UUID {
	return t.ID
}

func (t *TestEntity) SetID(id uuid.UUID) {
	t.ID = id
}

// TestEntityInterface 验证 TestEntity 实现了 Entity 接口
func TestEntityInterface(t *testing.T) {
	var _ Entity = &TestEntity{}
	var _ Entity = (*TestEntity)(nil)
}

// TestEntityGetSetID 测试 GetID 和 SetID 方法
func TestEntityGetSetID(t *testing.T) {
	entity := &TestEntity{}
	testID := uuid.New()

	entity.SetID(testID)
	assert.Equal(t, testID, entity.GetID())
	assert.Equal(t, testID, entity.ID)
}
