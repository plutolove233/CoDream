package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/plutolove233/co-dream/internal/dal/models"
	"github.com/plutolove233/co-dream/internal/database"
	"github.com/plutolove233/co-dream/internal/llm"
	"github.com/plutolove233/co-dream/internal/tools"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type PipelineStatus string
type ExecutionStatus string
type CheckpointStatus string

const (
	PipelineStatusPending PipelineStatus = "pending"
	PipelineStatusActive  PipelineStatus = "active"

	ExecutionStatusPending         ExecutionStatus = "pending"
	ExecutionStatusRunning         ExecutionStatus = "running"
	ExecutionStatusPaused          ExecutionStatus = "paused"
	ExecutionStatusPendingApproval ExecutionStatus = "pending_approval"
	ExecutionStatusCompleted       ExecutionStatus = "completed"
	ExecutionStatusFailed          ExecutionStatus = "failed"
	ExecutionStatusTerminated      ExecutionStatus = "terminated"

	CheckpointStatusPending  CheckpointStatus = "pending"
	CheckpointStatusApproved CheckpointStatus = "approved"
	CheckpointStatusRejected CheckpointStatus = "rejected"
)

var (
	ErrPipelineNotFound    = errors.New("pipeline not found")
	ErrExecutionNotFound   = errors.New("execution not found")
	ErrCheckpointNotFound  = errors.New("checkpoint not found")
	ErrExecutionNotPaused  = errors.New("execution is not paused")
	ErrExecutionTerminal   = errors.New("execution is already terminal")
	ErrDatabaseUnavailable = errors.New("postgresql database is not initialized")
	ErrLLMConfigRequired   = errors.New("llm api key is required for real agent runner")
)

type PipelineRecord struct {
	ID           string             `json:"id"`
	UserID       string             `json:"user_id"`
	Name         string             `json:"name"`
	Description  string             `json:"description,omitempty"`
	Requirement  string             `json:"requirement,omitempty"`
	ContextPaths []string           `json:"context_paths,omitempty"`
	Definition   PipelineDefinition `json:"definition"`
	Status       PipelineStatus     `json:"status"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

type PipelineCreateRequest struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Requirement  string          `json:"requirement"`
	ContextPaths []string        `json:"context_paths"`
	Template     string          `json:"template"`
	Provider     string          `json:"provider"`
	Model        string          `json:"model"`
	Stages       []PipelineStage `json:"stages"`
}

type PipelineUpdateRequest struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Requirement  string          `json:"requirement"`
	ContextPaths []string        `json:"context_paths"`
	Template     string          `json:"template"`
	Provider     string          `json:"provider"`
	Model        string          `json:"model"`
	Stages       []PipelineStage `json:"stages"`
}

type PipelineExecuteRequest struct {
	Requirement  string         `json:"requirement"`
	ContextPaths []string       `json:"context_paths"`
	Provider     string         `json:"provider"`
	Model        string         `json:"model"`
	Input        map[string]any `json:"input"`
}

type ExecutionRecord struct {
	ID                string                 `json:"id"`
	SessionID         string                 `json:"session_id,omitempty"`
	UserID            string                 `json:"user_id"`
	PipelineID        string                 `json:"pipeline_id"`
	Status            ExecutionStatus        `json:"status"`
	CurrentStageIndex int                    `json:"current_stage_index"`
	Input             map[string]any         `json:"input,omitempty"`
	Context           map[string]any         `json:"context,omitempty"`
	Stages            []StageExecutionResult `json:"stages,omitempty"`
	PendingCheckpoint *CheckpointRecord      `json:"pending_checkpoint,omitempty"`
	Error             string                 `json:"error,omitempty"`
	StartedAt         *time.Time             `json:"started_at,omitempty"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
	UpdatedAt         *time.Time             `json:"updated_at,omitempty"`
}

type CheckpointRecord struct {
	ID          string           `json:"id"`
	UserID      string           `json:"user_id"`
	ExecutionID string           `json:"execution_id"`
	PipelineID  string           `json:"pipeline_id"`
	StageID     string           `json:"stage_id"`
	StageName   string           `json:"stage_name"`
	StageIndex  int              `json:"stage_index"`
	Position    ApprovalPoint    `json:"position"`
	Status      CheckpointStatus `json:"status"`
	Artifacts   map[string]any   `json:"artifacts,omitempty"`
	Reason      string           `json:"reason,omitempty"`
	CreatedAt   *time.Time       `json:"created_at,omitempty"`
	DecidedAt   *time.Time       `json:"decided_at,omitempty"`
}

type ExecutionActionRequest struct {
	Action string `json:"action" binding:"required,oneof=pause resume terminate"`
}

type CheckpointDecisionRequest struct {
	Reason  string         `json:"reason"`
	Context map[string]any `json:"context"`
}

type pipelineConfig struct {
	Definition   PipelineDefinition `json:"definition"`
	Requirement  string             `json:"requirement,omitempty"`
	ContextPaths []string           `json:"context_paths,omitempty"`
	Provider     string             `json:"provider,omitempty"`
	Model        string             `json:"model,omitempty"`
	Template     string             `json:"template,omitempty"`
}

type executionOutput struct {
	Context             map[string]any  `json:"context,omitempty"`
	SessionID           string          `json:"session_id,omitempty"`
	Error               string          `json:"error,omitempty"`
	PendingCheckpointID string          `json:"pending_checkpoint_id,omitempty"`
	ApprovedGates       map[string]bool `json:"approved_gates,omitempty"`
}

type PipelineEvent struct {
	Type        string            `json:"type"`
	ExecutionID string            `json:"execution_id"`
	SessionID   string            `json:"session_id,omitempty"`
	Status      ExecutionStatus   `json:"status,omitempty"`
	StageID     string            `json:"stage_id,omitempty"`
	StageName   string            `json:"stage_name,omitempty"`
	Checkpoint  *CheckpointRecord `json:"checkpoint,omitempty"`
	Payload     map[string]any    `json:"payload,omitempty"`
	Error       string            `json:"error,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

const (
	PipelineEventExecutionStatusChanged = "execution.status_changed"
	PipelineEventStageStarted           = "stage.started"
	PipelineEventStageCompleted         = "stage.completed"
	PipelineEventCheckpointCreated      = "checkpoint.created"
	PipelineEventErrorOccurred          = "error.occurred"
	PipelineEventSnapshot               = "execution.snapshot"
)

type checkpointDecision struct {
	Status  CheckpointStatus `json:"status,omitempty"`
	Reason  string           `json:"reason,omitempty"`
	Context map[string]any   `json:"context,omitempty"`
}

type PipelineEngineService struct {
	db *gorm.DB

	mu          sync.Mutex
	running     map[string]struct{}
	subscribers map[string]map[chan PipelineEvent]struct{}
}

var defaultPipelineEngine = NewPipelineEngineService()

func DefaultPipelineEngine() *PipelineEngineService {
	return defaultPipelineEngine
}

func NewPipelineEngineService() *PipelineEngineService {
	return &PipelineEngineService{
		running:     make(map[string]struct{}),
		subscribers: make(map[string]map[chan PipelineEvent]struct{}),
	}
}

func NewPipelineEngineServiceWithDB(db *gorm.DB) *PipelineEngineService {
	s := NewPipelineEngineService()
	s.db = db
	return s
}

func (s *PipelineEngineService) CreatePipeline(ctx context.Context, userID string, req PipelineCreateRequest) (PipelineRecord, error) {
	db, err := s.database()
	if err != nil {
		return PipelineRecord{}, err
	}
	def := buildPipelineDefinition(req.Name, req.Requirement, req.ContextPaths, req.Provider, req.Model, req.Template, req.Stages)
	if err := validateDefinition(def); err != nil {
		return PipelineRecord{}, err
	}
	cfg := pipelineConfig{
		Definition:   def,
		Requirement:  req.Requirement,
		ContextPaths: append([]string(nil), req.ContextPaths...),
		Provider:     req.Provider,
		Model:        req.Model,
		Template:     req.Template,
	}
	configJSON, err := marshalJSON(cfg)
	if err != nil {
		return PipelineRecord{}, err
	}

	status := string(PipelineStatusPending)
	id := def.ID
	description := req.Description
	row := models.Pipeline{
		ID:          &id,
		UserID:      userID,
		Name:        def.Name,
		Description: stringPtrOrNil(description),
		Config:      configJSON,
		Status:      &status,
		CreatedBy:   &userID,
	}
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return PipelineRecord{}, fmt.Errorf("create pipeline: %w", err)
	}
	return pipelineRecordFromModel(row)
}

func (s *PipelineEngineService) ListPipelines(ctx context.Context, userID string) []PipelineRecord {
	db, err := s.database()
	if err != nil {
		return nil
	}
	var rows []models.Pipeline
	if err := db.WithContext(ctx).
		Where("user_id = ? AND is_deleted = false", userID).
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil
	}
	records := make([]PipelineRecord, 0, len(rows))
	for _, row := range rows {
		record, err := pipelineRecordFromModel(row)
		if err == nil {
			records = append(records, record)
		}
	}
	return records
}

func (s *PipelineEngineService) GetPipeline(ctx context.Context, userID, pipelineID string) (PipelineRecord, error) {
	row, err := s.getPipelineModel(ctx, userID, pipelineID)
	if err != nil {
		return PipelineRecord{}, err
	}
	return pipelineRecordFromModel(row)
}

func (s *PipelineEngineService) UpdatePipeline(ctx context.Context, userID, pipelineID string, req PipelineUpdateRequest) (PipelineRecord, error) {
	db, err := s.database()
	if err != nil {
		return PipelineRecord{}, err
	}
	row, err := s.getPipelineModel(ctx, userID, pipelineID)
	if err != nil {
		return PipelineRecord{}, err
	}
	current, err := pipelineRecordFromModel(row)
	if err != nil {
		return PipelineRecord{}, err
	}

	name := firstNonEmpty(req.Name, current.Name)
	requirement := firstNonEmpty(req.Requirement, current.Requirement)
	contextPaths := current.ContextPaths
	if req.ContextPaths != nil {
		contextPaths = req.ContextPaths
	}
	stages := current.Definition.Stages
	if len(req.Stages) > 0 {
		stages = req.Stages
	}
	cfg := pipelineConfig{
		Definition:   buildPipelineDefinition(name, requirement, contextPaths, req.Provider, req.Model, req.Template, stages),
		Requirement:  requirement,
		ContextPaths: append([]string(nil), contextPaths...),
		Provider:     req.Provider,
		Model:        req.Model,
		Template:     req.Template,
	}
	cfg.Definition.ID = pipelineID
	if err := validateDefinition(cfg.Definition); err != nil {
		return PipelineRecord{}, err
	}
	configJSON, err := marshalJSON(cfg)
	if err != nil {
		return PipelineRecord{}, err
	}
	updates := map[string]any{
		"name":       cfg.Definition.Name,
		"config":     configJSON,
		"updated_at": time.Now(),
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if err := db.WithContext(ctx).
		Model(&models.Pipeline{}).
		Where("id = ? AND user_id = ? AND is_deleted = false", pipelineID, userID).
		Updates(updates).Error; err != nil {
		return PipelineRecord{}, fmt.Errorf("update pipeline: %w", err)
	}
	return s.GetPipeline(ctx, userID, pipelineID)
}

func (s *PipelineEngineService) DeletePipeline(ctx context.Context, userID, pipelineID string) error {
	db, err := s.database()
	if err != nil {
		return err
	}
	res := db.WithContext(ctx).
		Model(&models.Pipeline{}).
		Where("id = ? AND user_id = ? AND is_deleted = false", pipelineID, userID).
		Updates(map[string]any{"is_deleted": true, "deleted_at": time.Now()})
	if res.Error != nil {
		return fmt.Errorf("delete pipeline: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrPipelineNotFound
	}
	return nil
}

func (s *PipelineEngineService) StartExecution(ctx context.Context, userID, pipelineID string, req PipelineExecuteRequest) (ExecutionRecord, error) {
	db, err := s.database()
	if err != nil {
		return ExecutionRecord{}, err
	}
	pipeline, err := s.GetPipeline(ctx, userID, pipelineID)
	if err != nil {
		return ExecutionRecord{}, err
	}
	input := cloneAnyMap(req.Input)
	if req.Requirement != "" {
		input["requirement"] = req.Requirement
	}
	if len(req.ContextPaths) > 0 {
		input["context_paths"] = append([]string(nil), req.ContextPaths...)
	}
	if req.Provider != "" {
		input["provider"] = req.Provider
	}
	if req.Model != "" {
		input["model"] = req.Model
	}
	if _, ok := input["requirement"]; !ok && pipeline.Requirement != "" {
		input["requirement"] = pipeline.Requirement
	}
	if _, ok := input["context_paths"]; !ok && len(pipeline.ContextPaths) > 0 {
		input["context_paths"] = append([]string(nil), pipeline.ContextPaths...)
	}
	now := time.Now()
	status := string(ExecutionStatusPending)
	index := int32(0)
	executionID := uuid.NewString()
	sessionID, err := s.createPipelineSession(ctx, userID, pipeline, executionID, input)
	if err != nil {
		return ExecutionRecord{}, err
	}
	out := executionOutput{Context: cloneAnyMap(input), SessionID: sessionID, ApprovedGates: map[string]bool{}}
	inputJSON, err := marshalJSON(input)
	if err != nil {
		return ExecutionRecord{}, err
	}
	outputJSON, err := marshalJSON(out)
	if err != nil {
		return ExecutionRecord{}, err
	}
	row := models.PipelineExecution{
		ID:                &executionID,
		UserID:            userID,
		PipelineID:        pipelineID,
		Status:            &status,
		CurrentStageIndex: &index,
		Input:             inputJSON,
		Output:            outputJSON,
		StartedAt:         &now,
	}
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return ExecutionRecord{}, fmt.Errorf("create execution: %w", err)
	}
	record, err := s.GetExecution(ctx, userID, executionID)
	if err != nil {
		return ExecutionRecord{}, err
	}
	s.publishExecutionEvent(ctx, userID, executionID, PipelineEvent{
		Type:        PipelineEventExecutionStatusChanged,
		ExecutionID: executionID,
		SessionID:   sessionID,
		Status:      ExecutionStatusPending,
	})
	s.launchExecution(context.WithoutCancel(ctx), userID, executionID, pipeline.Definition)
	return record, nil
}

func (s *PipelineEngineService) GetExecution(ctx context.Context, userID, executionID string) (ExecutionRecord, error) {
	row, err := s.getExecutionModel(ctx, userID, executionID)
	if err != nil {
		return ExecutionRecord{}, err
	}
	return s.executionRecordFromModel(ctx, row)
}

func (s *PipelineEngineService) ListExecutionStages(ctx context.Context, userID, executionID string) ([]StageExecutionResult, error) {
	db, err := s.database()
	if err != nil {
		return nil, err
	}
	var rows []models.StageExecution
	if err := db.WithContext(ctx).
		Where("execution_id = ? AND user_id = ? AND is_deleted = false", executionID, userID).
		Order("stage_order ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list stage executions: %w", err)
	}
	stages := make([]StageExecutionResult, 0, len(rows))
	for _, row := range rows {
		stage, err := stageResultFromModel(row)
		if err != nil {
			return nil, err
		}
		stages = append(stages, stage)
	}
	return stages, nil
}

func (s *PipelineEngineService) UpdateExecutionAction(ctx context.Context, userID, executionID string, req ExecutionActionRequest) (ExecutionRecord, error) {
	exec, err := s.getExecutionModel(ctx, userID, executionID)
	if err != nil {
		return ExecutionRecord{}, err
	}
	status := ExecutionStatus(valueOfStringPtr(exec.Status))
	if isTerminalExecution(status) {
		return ExecutionRecord{}, ErrExecutionTerminal
	}
	switch req.Action {
	case "pause":
		if err := s.updateExecutionFields(ctx, userID, executionID, map[string]any{"status": string(ExecutionStatusPaused)}); err != nil {
			return ExecutionRecord{}, err
		}
	case "terminate":
		now := time.Now()
		if err := s.updateExecutionFields(ctx, userID, executionID, map[string]any{"status": string(ExecutionStatusTerminated), "completed_at": &now}); err != nil {
			return ExecutionRecord{}, err
		}
	case "resume":
		if status != ExecutionStatusPaused {
			return ExecutionRecord{}, ErrExecutionNotPaused
		}
		if err := s.updateExecutionFields(ctx, userID, executionID, map[string]any{"status": string(ExecutionStatusPending)}); err != nil {
			return ExecutionRecord{}, err
		}
		pipeline, err := s.GetPipeline(ctx, userID, exec.PipelineID)
		if err != nil {
			return ExecutionRecord{}, err
		}
		s.launchExecution(context.WithoutCancel(ctx), userID, executionID, pipeline.Definition)
	default:
		return ExecutionRecord{}, fmt.Errorf("unsupported action %q", req.Action)
	}
	return s.GetExecution(ctx, userID, executionID)
}

func (s *PipelineEngineService) ListPendingCheckpoints(ctx context.Context, userID string) []CheckpointRecord {
	db, err := s.database()
	if err != nil {
		return nil
	}
	var rows []models.Checkpoint
	if err := db.WithContext(ctx).
		Where("user_id = ? AND status = ? AND is_deleted = false", userID, string(CheckpointStatusPending)).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil
	}
	records := make([]CheckpointRecord, 0, len(rows))
	for _, row := range rows {
		record, err := s.checkpointRecordFromModel(ctx, row)
		if err == nil {
			records = append(records, record)
		}
	}
	return records
}

func (s *PipelineEngineService) ApproveCheckpoint(ctx context.Context, userID, checkpointID string, req CheckpointDecisionRequest) (ExecutionRecord, error) {
	return s.decideCheckpoint(ctx, userID, checkpointID, CheckpointStatusApproved, req)
}

func (s *PipelineEngineService) RejectCheckpoint(ctx context.Context, userID, checkpointID string, req CheckpointDecisionRequest) (ExecutionRecord, error) {
	return s.decideCheckpoint(ctx, userID, checkpointID, CheckpointStatusRejected, req)
}

func (s *PipelineEngineService) decideCheckpoint(ctx context.Context, userID, checkpointID string, status CheckpointStatus, req CheckpointDecisionRequest) (ExecutionRecord, error) {
	db, err := s.database()
	if err != nil {
		return ExecutionRecord{}, err
	}
	var checkpoint models.Checkpoint
	if err := db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND is_deleted = false", checkpointID, userID).
		Take(&checkpoint).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ExecutionRecord{}, ErrCheckpointNotFound
		}
		return ExecutionRecord{}, fmt.Errorf("get checkpoint: %w", err)
	}
	if valueOfStringPtr(checkpoint.Status) != string(CheckpointStatusPending) {
		return ExecutionRecord{}, fmt.Errorf("checkpoint is already %s", valueOfStringPtr(checkpoint.Status))
	}
	exec, err := s.getExecutionModel(ctx, userID, checkpoint.ExecutionID)
	if err != nil {
		return ExecutionRecord{}, err
	}
	state, err := executionState(exec)
	if err != nil {
		return ExecutionRecord{}, err
	}
	decision := checkpointDecision{Status: status, Reason: req.Reason, Context: cloneAnyMap(req.Context)}
	decisionJSON, err := marshalJSON(decision)
	if err != nil {
		return ExecutionRecord{}, err
	}
	now := time.Now()
	if err := db.WithContext(ctx).Model(&models.Checkpoint{}).
		Where("id = ? AND user_id = ?", checkpointID, userID).
		Updates(map[string]any{
			"status":     string(status),
			"decision":   decisionJSON,
			"decided_at": &now,
		}).Error; err != nil {
		return ExecutionRecord{}, fmt.Errorf("decide checkpoint: %w", err)
	}
	stageIndex, point, err := checkpointPosition(ctx, db, userID, checkpoint)
	if err != nil {
		return ExecutionRecord{}, err
	}
	state.PendingCheckpointID = ""
	mergeAnyMap(state.Context, req.Context)
	switch status {
	case CheckpointStatusApproved:
		if state.ApprovedGates == nil {
			state.ApprovedGates = map[string]bool{}
		}
		state.ApprovedGates[checkpointGateKey(stageIndex, point)] = true
	case CheckpointStatusRejected:
		state.Context["rejection_reason"] = req.Reason
		if stageIndex > 0 {
			stageIndex--
		}
		if err := s.deleteStagesFrom(ctx, userID, valueOfStringPtr(exec.ID), stageIndex); err != nil {
			return ExecutionRecord{}, err
		}
		exec.CurrentStageIndex = int32Ptr(int32(stageIndex))
	}
	outputJSON, err := marshalJSON(state)
	if err != nil {
		return ExecutionRecord{}, err
	}
	if err := s.updateExecutionFields(ctx, userID, checkpoint.ExecutionID, map[string]any{
		"status":              string(ExecutionStatusPending),
		"current_stage_index": valueOfInt32Ptr(exec.CurrentStageIndex),
		"output":              outputJSON,
	}); err != nil {
		return ExecutionRecord{}, err
	}
	pipeline, err := s.GetPipeline(ctx, userID, exec.PipelineID)
	if err != nil {
		return ExecutionRecord{}, err
	}
	record, err := s.GetExecution(ctx, userID, checkpoint.ExecutionID)
	if err != nil {
		return ExecutionRecord{}, err
	}
	s.launchExecution(context.WithoutCancel(ctx), userID, checkpoint.ExecutionID, pipeline.Definition)
	return record, nil
}

func (s *PipelineEngineService) advanceExecution(ctx context.Context, userID, executionID string, def PipelineDefinition) error {
	if err := validateDefinition(def); err != nil {
		return err
	}
	for {
		exec, err := s.getExecutionModel(ctx, userID, executionID)
		if err != nil {
			return err
		}
		status := ExecutionStatus(valueOfStringPtr(exec.Status))
		if status == ExecutionStatusPaused || status == ExecutionStatusPendingApproval || isTerminalExecution(status) {
			return nil
		}
		stageIndex := int(valueOfInt32Ptr(exec.CurrentStageIndex))
		if stageIndex >= len(def.Stages) {
			now := time.Now()
			return s.updateExecutionFields(ctx, userID, executionID, map[string]any{
				"status":       string(ExecutionStatusCompleted),
				"completed_at": &now,
			})
		}
		state, err := executionState(exec)
		if err != nil {
			return err
		}
		stage := def.Stages[stageIndex]
		stageRow, err := s.ensureStageRow(ctx, userID, executionID, stage, stageIndex, state.Context)
		if err != nil {
			return err
		}
		if shouldCheckpoint(stage, ApprovalBeforeStage) && !state.ApprovedGates[checkpointGateKey(stageIndex, ApprovalBeforeStage)] {
			return s.createCheckpoint(ctx, userID, exec.PipelineID, executionID, *stageRow.ID, stage, stageIndex, ApprovalBeforeStage, map[string]any{
				"context": cloneAnyMap(state.Context),
				"stage":   stage,
			}, state)
		}
		if err := s.updateExecutionFields(ctx, userID, executionID, map[string]any{"status": string(ExecutionStatusRunning)}); err != nil {
			return err
		}
		if err := s.startStage(ctx, userID, *stageRow.ID); err != nil {
			return err
		}
		s.publishExecutionEvent(ctx, userID, executionID, PipelineEvent{
			Type:        PipelineEventStageStarted,
			ExecutionID: executionID,
			SessionID:   state.SessionID,
			StageID:     *stageRow.ID,
			StageName:   stage.Name,
			Status:      ExecutionStatusRunning,
		})
		result, err := s.runStage(ctx, def, stage, stageIndex, state.Context)
		if err != nil {
			return fmt.Errorf("run stage %s: %w", stage.Name, err)
		}
		if err := s.completeStage(ctx, userID, *stageRow.ID, result); err != nil {
			return err
		}
		mergeStageOutput(state.Context, result)
		nextIndex := stageIndex + 1
		stateJSON, err := marshalJSON(state)
		if err != nil {
			return err
		}
		afterStageExec, err := s.getExecutionModel(ctx, userID, executionID)
		if err != nil {
			return err
		}
		afterStageStatus := ExecutionStatus(valueOfStringPtr(afterStageExec.Status))
		if isTerminalExecution(afterStageStatus) {
			return nil
		}
		nextStatus := ExecutionStatusPending
		if afterStageStatus == ExecutionStatusPaused {
			nextStatus = ExecutionStatusPaused
		}
		if err := s.updateExecutionFields(ctx, userID, executionID, map[string]any{
			"status":              string(nextStatus),
			"current_stage_index": nextIndex,
			"output":              stateJSON,
		}); err != nil {
			return err
		}
		if nextStatus == ExecutionStatusPaused {
			return nil
		}
		if shouldCheckpoint(stage, ApprovalAfterStage) && !state.ApprovedGates[checkpointGateKey(stageIndex, ApprovalAfterStage)] {
			return s.createCheckpoint(ctx, userID, exec.PipelineID, executionID, *stageRow.ID, stage, stageIndex, ApprovalAfterStage, map[string]any{
				"context": cloneAnyMap(state.Context),
				"stage":   stage,
				"result":  result,
			}, state)
		}
	}
}

func (s *PipelineEngineService) runStage(ctx context.Context, def PipelineDefinition, stage PipelineStage, stageIndex int, input map[string]any) (StageExecutionResult, error) {
	planner := DefaultStagePlanner{}
	plan, err := planner.PlanStage(ctx, PlanRequest{Pipeline: def, Stage: stage, Context: cloneAnyMap(input)})
	if err != nil {
		return StageExecutionResult{}, err
	}
	dispatcher, err := s.newRealAgentDispatcher(stage)
	if err != nil {
		return StageExecutionResult{}, err
	}
	result := StageExecutionResult{StageName: stage.Name, Plan: plan, Items: make([]DispatchItem, 0, len(plan.Items))}
	for _, item := range plan.Items {
		out, err := dispatcher.Dispatch(ctx, DispatchRequest{
			Pipeline: def,
			Stage:    stage,
			PlanItem: item,
			Context:  cloneAnyMap(input),
		})
		if err != nil {
			return StageExecutionResult{}, err
		}
		result.Items = append(result.Items, DispatchItem{TaskID: item.TaskID, Output: out.Output, Meta: mergeMeta(out.Meta, map[string]any{"stage_order": stageIndex})})
	}
	return result, nil
}

func (s *PipelineEngineService) newRealAgentDispatcher(stage PipelineStage) (AgentDispatcher, error) {
	provider := stageProvider(stage)
	apiKey := firstNonEmpty(
		providerEnv(provider, "API_KEY"),
		viper.GetString(fmt.Sprintf("llm.providers.%s.apiKey", provider)),
		os.Getenv("CODEDREAM_LLM_API_KEY"),
		os.Getenv("OPENAI_API_KEY"),
		os.Getenv("API_KEY"),
		viper.GetString("llm.apiKey"),
	)
	if apiKey == "" {
		return nil, ErrLLMConfigRequired
	}
	model := firstNonEmpty(
		stageModel(stage),
		providerEnv(provider, "MODEL"),
		viper.GetString(fmt.Sprintf("llm.providers.%s.model", provider)),
		os.Getenv("CODEDREAM_LLM_MODEL"),
		os.Getenv("OPENAI_MODEL"),
		os.Getenv("MODEL_EPIP"),
		viper.GetString("llm.model"),
		"gpt-4o-mini",
	)
	baseURL := firstNonEmpty(
		providerEnv(provider, "BASE_URL"),
		viper.GetString(fmt.Sprintf("llm.providers.%s.baseURL", provider)),
		os.Getenv("CODEDREAM_LLM_BASE_URL"),
		os.Getenv("OPENAI_BASE_URL"),
		os.Getenv("BASE_URL"),
		viper.GetString("llm.baseURL"),
	)
	agentRegistry := NewAgentRegistry()
	agentsDir := firstNonEmpty(viper.GetString("agent.agentsPath"), "configs/agents")
	if err := agentRegistry.LoadFromDir(agentsDir); err != nil {
		return nil, err
	}
	registerInlineAgents(agentRegistry, stage)
	toolRegistry := tools.NewRegistry()
	registerTool := func(t any) error {
		tool, ok := t.(interface {
			Name() string
			Description() string
			Parameters() any
		})
		_ = tool
		if !ok {
			return nil
		}
		return nil
	}
	_ = registerTool
	if err := toolRegistry.Register(tools.NewFileHandler()); err != nil {
		return nil, err
	}
	if err := toolRegistry.Register(tools.NewBashTool()); err != nil {
		return nil, err
	}
	if err := toolRegistry.Register(tools.NewTodoManager()); err != nil {
		return nil, err
	}
	client := llm.NewClient(apiKey, baseURL, model, toolRegistry)
	return NewMarkdownDispatcher(agentRegistry, NewRunner(client)), nil
}

func (s *PipelineEngineService) launchExecution(ctx context.Context, userID, executionID string, def PipelineDefinition) {
	s.mu.Lock()
	if _, ok := s.running[executionID]; ok {
		s.mu.Unlock()
		return
	}
	s.running[executionID] = struct{}{}
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.running, executionID)
			s.mu.Unlock()
		}()
		if err := s.advanceExecution(ctx, userID, executionID, def); err != nil {
			_ = s.markExecutionFailed(ctx, userID, executionID, err)
		}
	}()
}

func (s *PipelineEngineService) SubscribeExecution(ctx context.Context, userID, executionID string) (<-chan PipelineEvent, func(), error) {
	record, err := s.GetExecution(ctx, userID, executionID)
	if err != nil {
		return nil, nil, err
	}
	ch := make(chan PipelineEvent, 32)
	s.mu.Lock()
	if s.subscribers[executionID] == nil {
		s.subscribers[executionID] = make(map[chan PipelineEvent]struct{})
	}
	s.subscribers[executionID][ch] = struct{}{}
	s.mu.Unlock()

	ch <- PipelineEvent{
		Type:        PipelineEventSnapshot,
		ExecutionID: executionID,
		SessionID:   record.SessionID,
		Status:      record.Status,
		Payload: map[string]any{
			"execution": record,
		},
		Timestamp: time.Now(),
	}

	unsubscribe := func() {
		s.mu.Lock()
		if subs := s.subscribers[executionID]; subs != nil {
			delete(subs, ch)
			if len(subs) == 0 {
				delete(s.subscribers, executionID)
			}
		}
		s.mu.Unlock()
		close(ch)
	}
	return ch, unsubscribe, nil
}

func (s *PipelineEngineService) publishExecutionEvent(ctx context.Context, userID, executionID string, event PipelineEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.ExecutionID == "" {
		event.ExecutionID = executionID
	}
	if event.SessionID == "" || event.Status == "" {
		if exec, err := s.getExecutionModel(ctx, userID, executionID); err == nil {
			if event.Status == "" {
				event.Status = ExecutionStatus(valueOfStringPtr(exec.Status))
			}
			if event.SessionID == "" {
				if state, err := executionState(exec); err == nil {
					event.SessionID = state.SessionID
				}
			}
		}
	}

	s.mu.Lock()
	subs := s.subscribers[executionID]
	for ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
	s.mu.Unlock()
}

func (s *PipelineEngineService) createPipelineSession(ctx context.Context, userID string, pipeline PipelineRecord, executionID string, input map[string]any) (string, error) {
	db, err := s.database()
	if err != nil {
		return "", err
	}
	sessionID := uuid.NewString()
	status := "active"
	title := firstNonEmpty(pipeline.Name, "Pipeline Execution")
	metadata, err := marshalJSON(map[string]any{
		"type":         "pipeline_execution",
		"pipeline_id":  pipeline.ID,
		"execution_id": executionID,
		"requirement":  input["requirement"],
	})
	if err != nil {
		return "", err
	}
	now := time.Now()
	row := models.Session{
		ID:            &sessionID,
		UserID:        userID,
		Title:         &title,
		Status:        &status,
		Metadata:      &metadata,
		LastMessageAt: &now,
	}
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return "", fmt.Errorf("create pipeline session: %w", err)
	}
	return sessionID, nil
}

func registerInlineAgents(registry *Registry, stage PipelineStage) {
	for _, agent := range stage.Agents {
		if agent.Name == "" {
			continue
		}
		_ = registry.Register(AgentDefinition{
			Name:         agent.Name,
			Description:  agent.Role,
			Model:        agent.Model,
			Instructions: agent.SystemPrompt,
		})
	}
}

func buildPipelineDefinition(name, requirement string, contextPaths []string, provider, model, template string, stages []PipelineStage) PipelineDefinition {
	if name == "" {
		name = "development-pipeline"
	}
	if len(stages) == 0 {
		stages = standardPipelineStages(requirement, contextPaths, provider, model)
	}
	return PipelineDefinition{
		ID:     uuid.NewString(),
		Name:   name,
		Stages: stages,
	}
}

func standardPipelineStages(requirement string, contextPaths []string, provider, model string) []PipelineStage {
	baseInput := map[string]any{
		"requirement":    requirement,
		"context_paths":  append([]string(nil), contextPaths...),
		"llm_provider":   provider,
		"provider_model": model,
	}
	return []PipelineStage{
		{
			Name:        "design",
			Objective:   "Create an implementation design for the requirement.",
			AgentType:   "solution_designer",
			Checkpoints: []string{string(ApprovalAfterStage)},
			Input:       cloneAnyMap(baseInput),
		},
		{
			Name:      "implement",
			Objective: "Implement the approved design with focused tests.",
			AgentType: "code_generator",
			DependsOn: []string{
				"design",
			},
			Input: cloneAnyMap(baseInput),
		},
		{
			Name:        "review",
			Objective:   "Review the implementation and decide whether it is ready.",
			AgentType:   "code_reviewer",
			DependsOn:   []string{"implement"},
			Checkpoints: []string{string(ApprovalAfterStage)},
			Input:       cloneAnyMap(baseInput),
		},
	}
}

func (s *PipelineEngineService) database() (*gorm.DB, error) {
	if s.db != nil {
		return s.db, nil
	}
	db := database.GetPostgreSqlDatabase()
	if db == nil || db.DB() == nil {
		return nil, ErrDatabaseUnavailable
	}
	return db.DB(), nil
}

func (s *PipelineEngineService) getPipelineModel(ctx context.Context, userID, pipelineID string) (models.Pipeline, error) {
	db, err := s.database()
	if err != nil {
		return models.Pipeline{}, err
	}
	var row models.Pipeline
	if err := db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND is_deleted = false", pipelineID, userID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Pipeline{}, ErrPipelineNotFound
		}
		return models.Pipeline{}, fmt.Errorf("get pipeline: %w", err)
	}
	return row, nil
}

func (s *PipelineEngineService) getExecutionModel(ctx context.Context, userID, executionID string) (models.PipelineExecution, error) {
	db, err := s.database()
	if err != nil {
		return models.PipelineExecution{}, err
	}
	var row models.PipelineExecution
	if err := db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND is_deleted = false", executionID, userID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.PipelineExecution{}, ErrExecutionNotFound
		}
		return models.PipelineExecution{}, fmt.Errorf("get execution: %w", err)
	}
	return row, nil
}

func pipelineRecordFromModel(row models.Pipeline) (PipelineRecord, error) {
	cfg, err := parsePipelineConfig(row.Config)
	if err != nil {
		return PipelineRecord{}, err
	}
	return PipelineRecord{
		ID:           valueOfStringPtr(row.ID),
		UserID:       row.UserID,
		Name:         row.Name,
		Description:  valueOfStringPtr(row.Description),
		Requirement:  cfg.Requirement,
		ContextPaths: append([]string(nil), cfg.ContextPaths...),
		Definition:   cfg.Definition,
		Status:       PipelineStatus(valueOfStringPtr(row.Status)),
		CreatedAt:    valueOfTimePtr(row.CreatedAt),
		UpdatedAt:    valueOfTimePtr(row.UpdatedAt),
	}, nil
}

func (s *PipelineEngineService) executionRecordFromModel(ctx context.Context, row models.PipelineExecution) (ExecutionRecord, error) {
	state, err := executionState(row)
	if err != nil {
		return ExecutionRecord{}, err
	}
	stages, err := s.ListExecutionStages(ctx, row.UserID, valueOfStringPtr(row.ID))
	if err != nil {
		return ExecutionRecord{}, err
	}
	var pending *CheckpointRecord
	if state.PendingCheckpointID != "" {
		var checkpoint models.Checkpoint
		db, dbErr := s.database()
		if dbErr != nil {
			return ExecutionRecord{}, dbErr
		}
		if err := db.WithContext(ctx).Where("id = ? AND user_id = ?", state.PendingCheckpointID, row.UserID).Take(&checkpoint).Error; err == nil {
			record, err := s.checkpointRecordFromModel(ctx, checkpoint)
			if err != nil {
				return ExecutionRecord{}, err
			}
			pending = &record
		}
	}
	input := map[string]any{}
	_ = json.Unmarshal([]byte(row.Input), &input)
	return ExecutionRecord{
		ID:                valueOfStringPtr(row.ID),
		SessionID:         state.SessionID,
		UserID:            row.UserID,
		PipelineID:        row.PipelineID,
		Status:            ExecutionStatus(valueOfStringPtr(row.Status)),
		CurrentStageIndex: int(valueOfInt32Ptr(row.CurrentStageIndex)),
		Input:             input,
		Context:           state.Context,
		Stages:            stages,
		PendingCheckpoint: pending,
		Error:             state.Error,
		StartedAt:         row.StartedAt,
		CompletedAt:       row.CompletedAt,
		UpdatedAt:         row.UpdatedAt,
	}, nil
}

func (s *PipelineEngineService) checkpointRecordFromModel(ctx context.Context, row models.Checkpoint) (CheckpointRecord, error) {
	db, err := s.database()
	if err != nil {
		return CheckpointRecord{}, err
	}
	var stage models.StageExecution
	if err := db.WithContext(ctx).Where("id = ? AND user_id = ?", row.StageID, row.UserID).Take(&stage).Error; err != nil {
		return CheckpointRecord{}, fmt.Errorf("get checkpoint stage: %w", err)
	}
	artifacts := map[string]any{}
	_ = json.Unmarshal([]byte(row.Artifacts), &artifacts)
	decision := checkpointDecision{}
	_ = json.Unmarshal([]byte(row.Decision), &decision)
	pipelineID := ""
	var exec models.PipelineExecution
	if err := db.WithContext(ctx).Where("id = ? AND user_id = ?", row.ExecutionID, row.UserID).Take(&exec).Error; err == nil {
		pipelineID = exec.PipelineID
	}
	return CheckpointRecord{
		ID:          valueOfStringPtr(row.ID),
		UserID:      row.UserID,
		ExecutionID: row.ExecutionID,
		PipelineID:  pipelineID,
		StageID:     row.StageID,
		StageName:   stage.StageName,
		StageIndex:  int(stage.StageOrder),
		Position:    ApprovalPoint(row.Position),
		Status:      CheckpointStatus(valueOfStringPtr(row.Status)),
		Artifacts:   artifacts,
		Reason:      decision.Reason,
		CreatedAt:   row.CreatedAt,
		DecidedAt:   row.DecidedAt,
	}, nil
}

func stageResultFromModel(row models.StageExecution) (StageExecutionResult, error) {
	plan := StagePlan{}
	if err := json.Unmarshal([]byte(row.Plan), &plan); err != nil {
		return StageExecutionResult{}, fmt.Errorf("parse stage plan: %w", err)
	}
	items := []DispatchItem{}
	if err := json.Unmarshal([]byte(row.Output), &items); err != nil {
		return StageExecutionResult{}, fmt.Errorf("parse stage output: %w", err)
	}
	return StageExecutionResult{StageName: row.StageName, Plan: plan, Items: items}, nil
}

func (s *PipelineEngineService) ensureStageRow(ctx context.Context, userID, executionID string, stage PipelineStage, stageIndex int, input map[string]any) (*models.StageExecution, error) {
	db, err := s.database()
	if err != nil {
		return nil, err
	}
	var row models.StageExecution
	err = db.WithContext(ctx).
		Where("execution_id = ? AND user_id = ? AND stage_order = ? AND is_deleted = false", executionID, userID, stageIndex).
		Take(&row).Error
	if err == nil {
		return &row, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get stage execution: %w", err)
	}
	status := string(ExecutionStatusPending)
	stageID := uuid.NewString()
	inputJSON, err := marshalJSON(input)
	if err != nil {
		return nil, err
	}
	emptyObject, _ := marshalJSON(map[string]any{})
	emptyItems, _ := marshalJSON([]DispatchItem{})
	row = models.StageExecution{
		ID:          &stageID,
		UserID:      userID,
		ExecutionID: executionID,
		StageName:   stage.Name,
		StageOrder:  int32(stageIndex),
		Status:      &status,
		Input:       inputJSON,
		Output:      emptyItems,
		Plan:        emptyObject,
	}
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, fmt.Errorf("create stage execution: %w", err)
	}
	return &row, nil
}

func (s *PipelineEngineService) startStage(ctx context.Context, userID, stageID string) error {
	db, err := s.database()
	if err != nil {
		return err
	}
	status := string(ExecutionStatusRunning)
	now := time.Now()
	return db.WithContext(ctx).
		Model(&models.StageExecution{}).
		Where("id = ? AND user_id = ?", stageID, userID).
		Updates(map[string]any{
			"status":     status,
			"started_at": &now,
			"updated_at": now,
		}).Error
}

func (s *PipelineEngineService) completeStage(ctx context.Context, userID, stageID string, result StageExecutionResult) error {
	planJSON, err := marshalJSON(result.Plan)
	if err != nil {
		return err
	}
	outputJSON, err := marshalJSON(result.Items)
	if err != nil {
		return err
	}
	status := string(ExecutionStatusCompleted)
	now := time.Now()
	db, err := s.database()
	if err != nil {
		return err
	}
	if err := db.WithContext(ctx).
		Model(&models.StageExecution{}).
		Where("id = ? AND user_id = ?", stageID, userID).
		Updates(map[string]any{
			"status":       status,
			"plan":         planJSON,
			"output":       outputJSON,
			"completed_at": &now,
			"updated_at":   now,
		}).Error; err != nil {
		return err
	}
	var stage models.StageExecution
	if err := db.WithContext(ctx).Where("id = ? AND user_id = ?", stageID, userID).Take(&stage).Error; err == nil {
		s.publishExecutionEvent(ctx, userID, stage.ExecutionID, PipelineEvent{
			Type:        PipelineEventStageCompleted,
			ExecutionID: stage.ExecutionID,
			StageID:     stageID,
			StageName:   stage.StageName,
			Status:      ExecutionStatusCompleted,
			Payload: map[string]any{
				"output": result,
			},
		})
	}
	return nil
}

func (s *PipelineEngineService) createCheckpoint(ctx context.Context, userID, pipelineID, executionID, stageID string, stage PipelineStage, stageIndex int, point ApprovalPoint, artifacts map[string]any, state executionOutput) error {
	db, err := s.database()
	if err != nil {
		return err
	}
	checkpointID := uuid.NewString()
	status := string(CheckpointStatusPending)
	artifactsJSON, err := marshalJSON(artifacts)
	if err != nil {
		return err
	}
	decisionJSON, _ := marshalJSON(checkpointDecision{})
	row := models.Checkpoint{
		ID:          &checkpointID,
		UserID:      userID,
		ExecutionID: executionID,
		StageID:     stageID,
		Position:    string(point),
		Status:      &status,
		Artifacts:   artifactsJSON,
		Decision:    decisionJSON,
	}
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("create checkpoint: %w", err)
	}
	state.PendingCheckpointID = checkpointID
	outputJSON, err := marshalJSON(state)
	if err != nil {
		return err
	}
	if err := s.updateExecutionFields(ctx, userID, executionID, map[string]any{
		"status": string(ExecutionStatusPendingApproval),
		"output": outputJSON,
	}); err != nil {
		return err
	}
	checkpoint, err := s.checkpointRecordFromModel(ctx, row)
	if err == nil {
		s.publishExecutionEvent(ctx, userID, executionID, PipelineEvent{
			Type:        PipelineEventCheckpointCreated,
			ExecutionID: executionID,
			SessionID:   state.SessionID,
			Status:      ExecutionStatusPendingApproval,
			Checkpoint:  &checkpoint,
			Payload: map[string]any{
				"artifacts": artifacts,
				"position":  point,
			},
		})
	}
	return nil
}

func (s *PipelineEngineService) updateExecutionFields(ctx context.Context, userID, executionID string, fields map[string]any) error {
	db, err := s.database()
	if err != nil {
		return err
	}
	fields["updated_at"] = time.Now()
	res := db.WithContext(ctx).
		Model(&models.PipelineExecution{}).
		Where("id = ? AND user_id = ? AND is_deleted = false", executionID, userID).
		Updates(fields)
	if res.Error != nil {
		return fmt.Errorf("update execution: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrExecutionNotFound
	}
	if rawStatus, ok := fields["status"]; ok {
		status, _ := rawStatus.(string)
		s.publishExecutionEvent(ctx, userID, executionID, PipelineEvent{
			Type:        PipelineEventExecutionStatusChanged,
			ExecutionID: executionID,
			Status:      ExecutionStatus(status),
		})
	}
	return nil
}

func (s *PipelineEngineService) markExecutionFailed(ctx context.Context, userID, executionID string, runErr error) error {
	exec, err := s.getExecutionModel(ctx, userID, executionID)
	if err != nil {
		return err
	}
	state, _ := executionState(exec)
	state.Error = runErr.Error()
	outputJSON, _ := marshalJSON(state)
	now := time.Now()
	if err := s.updateExecutionFields(ctx, userID, executionID, map[string]any{
		"status":       string(ExecutionStatusFailed),
		"output":       outputJSON,
		"completed_at": &now,
	}); err != nil {
		return err
	}
	s.publishExecutionEvent(ctx, userID, executionID, PipelineEvent{
		Type:        PipelineEventErrorOccurred,
		ExecutionID: executionID,
		SessionID:   state.SessionID,
		Status:      ExecutionStatusFailed,
		Error:       runErr.Error(),
	})
	return nil
}

func (s *PipelineEngineService) deleteStagesFrom(ctx context.Context, userID, executionID string, stageIndex int) error {
	db, err := s.database()
	if err != nil {
		return err
	}
	return db.WithContext(ctx).
		Model(&models.StageExecution{}).
		Where("execution_id = ? AND user_id = ? AND stage_order >= ?", executionID, userID, stageIndex).
		Updates(map[string]any{"is_deleted": true, "deleted_at": time.Now()}).Error
}

func parsePipelineConfig(raw string) (pipelineConfig, error) {
	cfg := pipelineConfig{}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return pipelineConfig{}, fmt.Errorf("parse pipeline config: %w", err)
	}
	if cfg.Definition.ID == "" {
		return pipelineConfig{}, errors.New("pipeline config missing definition")
	}
	return cfg, nil
}

func executionState(row models.PipelineExecution) (executionOutput, error) {
	state := executionOutput{Context: map[string]any{}, ApprovedGates: map[string]bool{}}
	if row.Output == "" {
		return state, nil
	}
	if err := json.Unmarshal([]byte(row.Output), &state); err != nil {
		return executionOutput{}, fmt.Errorf("parse execution output: %w", err)
	}
	if state.Context == nil {
		state.Context = map[string]any{}
	}
	if state.ApprovedGates == nil {
		state.ApprovedGates = map[string]bool{}
	}
	return state, nil
}

func validateDefinition(def PipelineDefinition) error {
	dispatcher := NewMapDispatcher()
	for _, stage := range def.Stages {
		if stage.AgentType != "" {
			dispatcher.Register(stage.AgentType, func(_ context.Context, _ DispatchRequest) (DispatchResult, error) {
				return DispatchResult{}, nil
			})
		}
		for _, agent := range stage.Agents {
			if agent.AgentType != "" {
				dispatcher.Register(agent.AgentType, func(_ context.Context, _ DispatchRequest) (DispatchResult, error) {
					return DispatchResult{}, nil
				})
			}
		}
	}
	_, err := NewPipelineOrchestrator(nil, dispatcher).Run(context.Background(), def, map[string]any{})
	return err
}

func checkpointPosition(ctx context.Context, db *gorm.DB, userID string, checkpoint models.Checkpoint) (int, ApprovalPoint, error) {
	var stage models.StageExecution
	if err := db.WithContext(ctx).Where("id = ? AND user_id = ?", checkpoint.StageID, userID).Take(&stage).Error; err != nil {
		return 0, "", fmt.Errorf("get checkpoint stage: %w", err)
	}
	return int(stage.StageOrder), ApprovalPoint(checkpoint.Position), nil
}

func shouldCheckpoint(stage PipelineStage, point ApprovalPoint) bool {
	if stage.Checkpoint == string(point) {
		return true
	}
	for _, checkpoint := range stage.Checkpoints {
		if checkpoint == string(point) {
			return true
		}
	}
	return false
}

func checkpointGateKey(stageIndex int, point ApprovalPoint) string {
	return fmt.Sprintf("%d:%s", stageIndex, point)
}

func isTerminalExecution(status ExecutionStatus) bool {
	return status == ExecutionStatusCompleted || status == ExecutionStatusFailed || status == ExecutionStatusTerminated
}

func mergeStageOutput(ctx map[string]any, result StageExecutionResult) {
	stageOutput := map[string]any{}
	for _, item := range result.Items {
		for k, v := range item.Output {
			stageOutput[k] = v
		}
	}
	ctx[result.StageName] = stageOutput
}

func mergeMeta(left, right map[string]any) map[string]any {
	out := cloneAnyMap(left)
	mergeAnyMap(out, right)
	return out
}

func stageModel(stage PipelineStage) string {
	if len(stage.Agents) == 0 {
		if model, ok := stage.Input["provider_model"].(string); ok {
			return model
		}
		if model, ok := stage.Input["model"].(string); ok {
			return model
		}
		return ""
	}
	return firstNonEmpty(stage.Agents[0].Model, stringFromMap(stage.Input, "provider_model"), stringFromMap(stage.Input, "model"))
}

func stageProvider(stage PipelineStage) string {
	if len(stage.Agents) > 0 && stage.Agents[0].Provider != "" {
		return stage.Agents[0].Provider
	}
	return firstNonEmpty(stringFromMap(stage.Input, "llm_provider"), stringFromMap(stage.Input, "provider"), "openai")
}

func providerEnv(provider, suffix string) string {
	if provider == "" {
		return ""
	}
	key := strings.ToUpper(provider)
	key = strings.NewReplacer("-", "_", ".", "_").Replace(key)
	return os.Getenv("CODEDREAM_" + key + "_" + suffix)
}

func stringFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return value
}

func cloneAnyMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func mergeAnyMap(dst map[string]any, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}

func marshalJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func stringPtrOrNil(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func valueOfStringPtr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func valueOfInt32Ptr(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}

func int32Ptr(v int32) *int32 {
	return &v
}

func valueOfTimePtr(v *time.Time) time.Time {
	if v == nil {
		return time.Time{}
	}
	return *v
}
