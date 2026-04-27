package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AgentTaskStatus string

const (
	AgentTaskStatusQueued    AgentTaskStatus = "queued"
	AgentTaskStatusRunning   AgentTaskStatus = "running"
	AgentTaskStatusCompleted AgentTaskStatus = "completed"
	AgentTaskStatusFailed    AgentTaskStatus = "failed"
)

type AgentTaskInput struct {
	Prompt      string                 `json:"prompt"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Tools       []string               `json:"tools,omitempty"`
	Constraints map[string]interface{} `json:"constraints,omitempty"`
}

type AgentTaskOutput struct {
	Result      string                 `json:"result"`
	Artifacts   []string               `json:"artifacts,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Error       string                 `json:"error,omitempty"`
	ToolCalls   []ToolCall             `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	Result    interface{}            `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

type ModelConfig struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
}

type TokenUsage struct {
	InputTokens          int `json:"input_tokens"`
	OutputTokens         int `json:"output_tokens"`
	CacheCreationTokens  int `json:"cache_creation_tokens,omitempty"`
	CacheReadTokens      int `json:"cache_read_tokens,omitempty"`
}

type AgentTask struct {
	ID                uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	StageExecutionID  uuid.UUID       `gorm:"type:uuid;not null;index" json:"stage_execution_id"`
	AgentType         string          `gorm:"type:varchar(255);not null;index" json:"agent_type"`
	Status            AgentTaskStatus `gorm:"type:varchar(50);not null;default:'queued';index" json:"status"`
	Input             AgentTaskInput  `gorm:"type:jsonb;serializer:json" json:"input"`
	Output            AgentTaskOutput `gorm:"type:jsonb;serializer:json" json:"output"`
	ModelConfig       ModelConfig     `gorm:"type:jsonb;serializer:json" json:"model_config"`
	TokenUsage        TokenUsage      `gorm:"type:jsonb;serializer:json" json:"token_usage"`
	StartedAt         *time.Time      `json:"started_at,omitempty"`
	CompletedAt       *time.Time      `json:"completed_at,omitempty"`
	CreatedAt         time.Time       `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt  `gorm:"index" json:"deleted_at,omitempty"`

	StageExecution StageExecution `gorm:"foreignKey:StageExecutionID;constraint:OnDelete:CASCADE" json:"stage_execution,omitempty"`
}

func (AgentTask) TableName() string {
	return "agent_tasks"
}

// GetID 返回 AgentTask 的 ID
func (a AgentTask) GetID() uuid.UUID {
	return a.ID
}

// SetID 设置 AgentTask 的 ID
func (a *AgentTask) SetID(id uuid.UUID) {
	a.ID = id
}
