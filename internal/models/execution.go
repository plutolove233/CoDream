package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusPaused    ExecutionStatus = "paused"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
)

type ExecutionInput struct {
	Requirement string                 `json:"requirement"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

type ExecutionOutput struct {
	Result   string                 `json:"result"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type PipelineExecution struct {
	ID                uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PipelineID        uuid.UUID       `gorm:"type:uuid;not null;index" json:"pipeline_id"`
	Status            ExecutionStatus `gorm:"type:varchar(50);not null;default:'pending';index" json:"status"`
	CurrentStageIndex int             `gorm:"default:0" json:"current_stage_index"`
	Input             ExecutionInput  `gorm:"type:jsonb;serializer:json" json:"input"`
	Output            ExecutionOutput `gorm:"type:jsonb;serializer:json" json:"output"`
	StartedAt         *time.Time      `json:"started_at,omitempty"`
	CompletedAt       *time.Time      `json:"completed_at,omitempty"`
	CreatedAt         time.Time       `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt  `gorm:"index" json:"deleted_at,omitempty"`

	Pipeline        Pipeline         `gorm:"foreignKey:PipelineID;constraint:OnDelete:CASCADE" json:"pipeline,omitempty"`
	StageExecutions []StageExecution `gorm:"foreignKey:ExecutionID;constraint:OnDelete:CASCADE" json:"stage_executions,omitempty"`
	Checkpoints     []Checkpoint     `gorm:"foreignKey:ExecutionID;constraint:OnDelete:CASCADE" json:"checkpoints,omitempty"`
}

func (PipelineExecution) TableName() string {
	return "pipeline_executions"
}
