package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StageStatus string

const (
	StageStatusPending   StageStatus = "pending"
	StageStatusPlanning  StageStatus = "planning"
	StageStatusExecuting StageStatus = "executing"
	StageStatusChecking  StageStatus = "checking"
	StageStatusCompleted StageStatus = "completed"
	StageStatusFailed    StageStatus = "failed"
)

type StagePlan struct {
	Tasks        []TaskItem             `json:"tasks"`
	Dependencies map[string][]string    `json:"dependencies,omitempty"`
	Resources    map[string]interface{} `json:"resources,omitempty"`
}

type TaskItem struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	DependsOn   []string `json:"depends_on,omitempty"`
}

type StageInput struct {
	Data     map[string]interface{} `json:"data"`
	Context  map[string]interface{} `json:"context,omitempty"`
	FromPrev bool                   `json:"from_prev,omitempty"`
}

type StageOutput struct {
	Result   map[string]interface{} `json:"result"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type StageExecution struct {
	ID          uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ExecutionID uuid.UUID   `gorm:"type:uuid;not null;index" json:"execution_id"`
	StageName   string      `gorm:"type:varchar(255);not null" json:"stage_name"`
	StageOrder  int         `gorm:"not null" json:"stage_order"`
	Status      StageStatus `gorm:"type:varchar(50);not null;default:'pending';index" json:"status"`
	Input       StageInput  `gorm:"type:jsonb;serializer:json" json:"input"`
	Output      StageOutput `gorm:"type:jsonb;serializer:json" json:"output"`
	Plan        StagePlan   `gorm:"type:jsonb;serializer:json" json:"plan"`
	RetryCount  int         `gorm:"default:0" json:"retry_count"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	CreatedAt   time.Time   `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt   time.Time   `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	PipelineExecution PipelineExecution `gorm:"foreignKey:ExecutionID;constraint:OnDelete:CASCADE" json:"pipeline_execution,omitempty"`
	AgentTasks        []AgentTask       `gorm:"foreignKey:StageExecutionID;constraint:OnDelete:CASCADE" json:"agent_tasks,omitempty"`
	Checkpoints       []Checkpoint      `gorm:"foreignKey:StageID;constraint:OnDelete:CASCADE" json:"checkpoints,omitempty"`
}

func (StageExecution) TableName() string {
	return "stage_executions"
}

// GetID 返回 StageExecution 的 ID
func (s StageExecution) GetID() uuid.UUID {
	return s.ID
}

// SetID 设置 StageExecution 的 ID
func (s *StageExecution) SetID(id uuid.UUID) {
	s.ID = id
}