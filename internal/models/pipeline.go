package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PipelineStatus string

const (
	PipelineStatusPending   PipelineStatus = "pending"
	PipelineStatusRunning   PipelineStatus = "running"
	PipelineStatusPaused    PipelineStatus = "paused"
	PipelineStatusCompleted PipelineStatus = "completed"
	PipelineStatusFailed    PipelineStatus = "failed"
)

type CheckpointPosition string

const (
	CheckpointBefore CheckpointPosition = "before"
	CheckpointAfter  CheckpointPosition = "after"
)

type BackoffStrategy string

const (
	BackoffExponential BackoffStrategy = "exponential"
	BackoffLinear      BackoffStrategy = "linear"
	BackoffFixed       BackoffStrategy = "fixed"
)

type RetryPolicy struct {
	MaxRetries      int             `json:"max_retries"`
	BackoffStrategy BackoffStrategy `json:"backoff_strategy"`
	BackoffBase     string          `json:"backoff_base"`
	MaxBackoff      string          `json:"max_backoff"`
	RetryOnErrors   []string        `json:"retry_on_errors"`
}

type CheckpointConfig struct {
	Position CheckpointPosition `json:"position"`
	Required bool               `json:"required"`
}

type StageConfig struct {
	Name            string            `json:"name"`
	Order           int               `json:"order"`
	AgentType       string            `json:"agent_type"`
	Model           string            `json:"model"`
	Checkpoint      *CheckpointConfig `json:"checkpoint,omitempty"`
	RetryPolicy     *RetryPolicy      `json:"retry_policy,omitempty"`
	ParallelWith    []string          `json:"parallel_with,omitempty"`
	ToolPermissions []string          `json:"tool_permissions,omitempty"`
}

type PipelineConfig struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Stages      []StageConfig `json:"stages"`
}

type Pipeline struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"type:varchar(255);not null;index" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Config      PipelineConfig `gorm:"type:jsonb;serializer:json" json:"config"`
	Status      PipelineStatus `gorm:"type:varchar(50);not null;default:'pending';index" json:"status"`
	CreatedBy   string         `gorm:"type:varchar(255);index" json:"created_by"`
	CreatedAt   time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Executions []PipelineExecution `gorm:"foreignKey:PipelineID;constraint:OnDelete:CASCADE" json:"executions,omitempty"`
}

func (Pipeline) TableName() string {
	return "pipelines"
}
