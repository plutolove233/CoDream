package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CheckpointStatus string

const (
	CheckpointStatusPending  CheckpointStatus = "pending"
	CheckpointStatusApproved CheckpointStatus = "approved"
	CheckpointStatusRejected CheckpointStatus = "rejected"
)

type CheckpointArtifacts struct {
	Files       []ArtifactFile         `json:"files,omitempty"`
	Outputs     map[string]interface{} `json:"outputs,omitempty"`
	Metrics     map[string]float64     `json:"metrics,omitempty"`
	Screenshots []string               `json:"screenshots,omitempty"`
}

type ArtifactFile struct {
	Path        string `json:"path"`
	Type        string `json:"type"`
	Size        int64  `json:"size"`
	Checksum    string `json:"checksum,omitempty"`
	Description string `json:"description,omitempty"`
}

type CheckpointDecision struct {
	Approved  bool   `json:"approved"`
	Reason    string `json:"reason,omitempty"`
	Feedback  string `json:"feedback,omitempty"`
	DecidedBy string `json:"decided_by,omitempty"`
}

type Checkpoint struct {
	ID          uuid.UUID            `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ExecutionID uuid.UUID            `gorm:"type:uuid;not null;index" json:"execution_id"`
	StageID     uuid.UUID            `gorm:"type:uuid;not null;index" json:"stage_id"`
	Position    CheckpointPosition   `gorm:"type:varchar(50);not null" json:"position"`
	Status      CheckpointStatus     `gorm:"type:varchar(50);not null;default:'pending';index" json:"status"`
	Artifacts   CheckpointArtifacts  `gorm:"type:jsonb;serializer:json" json:"artifacts"`
	Decision    CheckpointDecision   `gorm:"type:jsonb;serializer:json" json:"decision"`
	CreatedAt   time.Time            `gorm:"autoCreateTime;index" json:"created_at"`
	DecidedAt   *time.Time           `json:"decided_at,omitempty"`
	DeletedAt   gorm.DeletedAt       `gorm:"index" json:"deleted_at,omitempty"`

	PipelineExecution PipelineExecution `gorm:"foreignKey:ExecutionID;constraint:OnDelete:CASCADE" json:"pipeline_execution,omitempty"`
	StageExecution    StageExecution    `gorm:"foreignKey:StageID;constraint:OnDelete:CASCADE" json:"stage_execution,omitempty"`
}

func (Checkpoint) TableName() string {
	return "checkpoints"
}
