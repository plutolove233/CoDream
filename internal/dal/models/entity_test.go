package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/plutolove233/co-dream/internal/dao"
	"github.com/stretchr/testify/assert"
)

// TestAllModelsImplementEntity 验证所有模型都实现了 Entity 接口
func TestAllModelsImplementEntity(t *testing.T) {
	// 验证所有模型都实现了 dao.Entity 接口
	var _ dao.Entity = &Pipeline{}
	var _ dao.Entity = &PipelineExecution{}
	var _ dao.Entity = &StageExecution{}
	var _ dao.Entity = &AgentTask{}
	var _ dao.Entity = &Checkpoint{}
}

// TestPipelineEntity 测试 Pipeline 的 Entity 方法
func TestPipelineEntity(t *testing.T) {
	p := &Pipeline{}
	testID := uuid.New()

	p.SetID(testID)
	assert.Equal(t, testID, p.GetID())
	assert.Equal(t, testID, p.ID)
}

// TestPipelineExecutionEntity 测试 PipelineExecution 的 Entity 方法
func TestPipelineExecutionEntity(t *testing.T) {
	e := &PipelineExecution{}
	testID := uuid.New()

	e.SetID(testID)
	assert.Equal(t, testID, e.GetID())
	assert.Equal(t, testID, e.ID)
}
