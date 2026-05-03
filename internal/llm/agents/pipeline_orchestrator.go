package agents

import (
	"context"
	"errors"
	"fmt"
)

type RunStatus string
type ApprovalPoint string
type ApprovalStatus string

const (
	RunStatusCompleted       RunStatus = "completed"
	RunStatusPendingApproval RunStatus = "pending_approval"
	RunStatusRejected        RunStatus = "rejected"

	ApprovalBeforeStage   ApprovalPoint = "before_stage"
	ApprovalAfterPlan     ApprovalPoint = "after_plan"
	ApprovalBeforeExecute ApprovalPoint = "before_execute"
	ApprovalAfterStage    ApprovalPoint = "after_stage"

	ApprovalApproved ApprovalStatus = "approved"
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalRejected ApprovalStatus = "rejected"
)

var (
	ErrPipelineHasNoStage      = errors.New("pipeline has no stage")
	ErrStageNameRequired       = errors.New("stage name is required")
	ErrStageNameDuplicated     = errors.New("stage name duplicated")
	ErrStageAgentTypeRequired  = errors.New("stage agent_type is required")
	ErrStageDependsOnNotFound  = errors.New("stage depends_on references unknown stage")
	ErrStageDependencyNotReady = errors.New("stage dependency has not completed")
	ErrStageDependencyCycle    = errors.New("stage dependency cycle detected")
)

type PipelineDefinition struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Stages []PipelineStage `json:"stages"`
}

type PipelineStage struct {
	Name        string         `json:"name"`
	Objective   string         `json:"objective"`
	AgentType   string         `json:"agent_type"`
	DependsOn   []string       `json:"depends_on,omitempty"`
	Checkpoint  string         `json:"checkpoint,omitempty"`
	Checkpoints []string       `json:"checkpoints,omitempty"`
	Input       map[string]any `json:"input,omitempty"`
}

type StagePlanItem struct {
	TaskID      string         `json:"task_id"`
	StageName   string         `json:"stage_name"`
	Description string         `json:"description"`
	AgentType   string         `json:"agent_type"`
	Input       map[string]any `json:"input,omitempty"`
}

type StagePlan struct {
	StageName string          `json:"stage_name"`
	Items     []StagePlanItem `json:"items"`
}

type PlanRequest struct {
	Pipeline PipelineDefinition `json:"pipeline"`
	Stage    PipelineStage      `json:"stage"`
	Context  map[string]any     `json:"context,omitempty"`
}

type StagePlanner interface {
	PlanStage(ctx context.Context, req PlanRequest) (StagePlan, error)
}

type DefaultStagePlanner struct{}

func (p DefaultStagePlanner) PlanStage(_ context.Context, req PlanRequest) (StagePlan, error) {
	description := req.Stage.Objective
	if description == "" {
		description = fmt.Sprintf("execute stage %s", req.Stage.Name)
	}
	task := StagePlanItem{
		TaskID:      req.Stage.Name + "-task-1",
		StageName:   req.Stage.Name,
		Description: description,
		AgentType:   req.Stage.AgentType,
		Input:       req.Stage.Input,
	}
	return StagePlan{
		StageName: req.Stage.Name,
		Items:     []StagePlanItem{task},
	}, nil
}

type DispatchRequest struct {
	Pipeline PipelineDefinition `json:"pipeline"`
	Stage    PipelineStage      `json:"stage"`
	PlanItem StagePlanItem      `json:"plan_item"`
	Context  map[string]any     `json:"context,omitempty"`
}

type DispatchResult struct {
	Output map[string]any `json:"output,omitempty"`
	Meta   map[string]any `json:"meta,omitempty"`
}

type AgentDispatcher interface {
	Dispatch(ctx context.Context, req DispatchRequest) (DispatchResult, error)
}

type AgentFunc func(ctx context.Context, req DispatchRequest) (DispatchResult, error)

type MapDispatcher struct {
	agents map[string]AgentFunc
}

func NewMapDispatcher() *MapDispatcher {
	return &MapDispatcher{agents: make(map[string]AgentFunc)}
}

func (d *MapDispatcher) Register(agentType string, fn AgentFunc) {
	d.agents[agentType] = fn
}

func (d *MapDispatcher) Dispatch(ctx context.Context, req DispatchRequest) (DispatchResult, error) {
	fn, ok := d.agents[req.PlanItem.AgentType]
	if !ok {
		return DispatchResult{}, fmt.Errorf("agent %q not registered", req.PlanItem.AgentType)
	}
	return fn(ctx, req)
}

type StageExecutionResult struct {
	StageName string         `json:"stage_name"`
	Plan      StagePlan      `json:"plan"`
	Items     []DispatchItem `json:"items"`
}

type DispatchItem struct {
	TaskID string         `json:"task_id"`
	Output map[string]any `json:"output,omitempty"`
	Meta   map[string]any `json:"meta,omitempty"`
}

type PipelineRunResult struct {
	PipelineID      string                 `json:"pipeline_id"`
	Status          RunStatus              `json:"status"`
	Stages          []StageExecutionResult `json:"stages"`
	PendingApproval *ApprovalRequest       `json:"pending_approval,omitempty"`
	RejectionReason string                 `json:"rejection_reason,omitempty"`
}

type PipelineOrchestrator struct {
	planner      StagePlanner
	dispatcher   AgentDispatcher
	approvalGate ApprovalGate
}

func NewPipelineOrchestrator(planner StagePlanner, dispatcher AgentDispatcher) *PipelineOrchestrator {
	if planner == nil {
		planner = DefaultStagePlanner{}
	}
	return &PipelineOrchestrator{
		planner:    planner,
		dispatcher: dispatcher,
	}
}

func (o *PipelineOrchestrator) WithApprovalGate(gate ApprovalGate) *PipelineOrchestrator {
	o.approvalGate = gate
	return o
}

func (o *PipelineOrchestrator) Run(ctx context.Context, def PipelineDefinition, input map[string]any) (PipelineRunResult, error) {
	if err := validatePipeline(def); err != nil {
		return PipelineRunResult{}, err
	}
	if o.dispatcher == nil {
		return PipelineRunResult{}, errors.New("dispatcher is required")
	}
	stages, err := orderStages(def.Stages)
	if err != nil {
		return PipelineRunResult{}, err
	}

	completed := make(map[string]bool, len(stages))
	ctxData := cloneMap(input)
	result := PipelineRunResult{
		PipelineID: def.ID,
		Status:     RunStatusCompleted,
		Stages:     make([]StageExecutionResult, 0, len(stages)),
	}

	for _, stage := range stages {
		if err := ensureStageReady(stage, completed); err != nil {
			return PipelineRunResult{}, fmt.Errorf("stage %s: %w", stage.Name, err)
		}

		if stopped, err := o.requestApproval(ctx, def, stage, ApprovalBeforeStage, nil, nil, ctxData, &result); stopped || err != nil {
			return result, err
		}

		plan, err := o.planner.PlanStage(ctx, PlanRequest{
			Pipeline: def,
			Stage:    stage,
			Context:  cloneMap(ctxData),
		})
		if err != nil {
			return PipelineRunResult{}, fmt.Errorf("plan stage %s: %w", stage.Name, err)
		}

		if stopped, err := o.requestApproval(ctx, def, stage, ApprovalAfterPlan, &plan, nil, ctxData, &result); stopped || err != nil {
			return result, err
		}

		stageResult := StageExecutionResult{
			StageName: stage.Name,
			Plan:      plan,
			Items:     make([]DispatchItem, 0, len(plan.Items)),
		}

		if stopped, err := o.requestApproval(ctx, def, stage, ApprovalBeforeExecute, &plan, &stageResult, ctxData, &result); stopped || err != nil {
			return result, err
		}

		for _, item := range plan.Items {
			dispatchResult, err := o.dispatcher.Dispatch(ctx, DispatchRequest{
				Pipeline: def,
				Stage:    stage,
				PlanItem: item,
				Context:  cloneMap(ctxData),
			})
			if err != nil {
				return PipelineRunResult{}, fmt.Errorf("dispatch %s/%s: %w", stage.Name, item.TaskID, err)
			}

			stageResult.Items = append(stageResult.Items, DispatchItem{
				TaskID: item.TaskID,
				Output: cloneMap(dispatchResult.Output),
				Meta:   cloneMap(dispatchResult.Meta),
			})
			mergeInto(ctxData, dispatchResult.Output)
		}

		if stopped, err := o.requestApproval(ctx, def, stage, ApprovalAfterStage, &plan, &stageResult, ctxData, &result); stopped || err != nil {
			return result, err
		}

		completed[stage.Name] = true
		result.Stages = append(result.Stages, stageResult)
	}

	return result, nil
}

type ApprovalRequest struct {
	Pipeline PipelineDefinition    `json:"pipeline"`
	Stage    PipelineStage         `json:"stage"`
	Point    ApprovalPoint         `json:"point"`
	Plan     *StagePlan            `json:"plan,omitempty"`
	Result   *StageExecutionResult `json:"result,omitempty"`
	Context  map[string]any        `json:"context,omitempty"`
}

type ApprovalDecision struct {
	Status  ApprovalStatus `json:"status"`
	Reason  string         `json:"reason,omitempty"`
	Context map[string]any `json:"context,omitempty"`
}

type ApprovalGate interface {
	RequestApproval(ctx context.Context, req ApprovalRequest) (ApprovalDecision, error)
}

func (o *PipelineOrchestrator) requestApproval(
	ctx context.Context,
	def PipelineDefinition,
	stage PipelineStage,
	point ApprovalPoint,
	plan *StagePlan,
	stageResult *StageExecutionResult,
	ctxData map[string]any,
	runResult *PipelineRunResult,
) (bool, error) {
	if !stageRequiresApproval(stage, point) {
		return false, nil
	}
	if o.approvalGate == nil {
		return false, nil
	}

	req := ApprovalRequest{
		Pipeline: def,
		Stage:    stage,
		Point:    point,
		Plan:     plan,
		Result:   stageResult,
		Context:  cloneMap(ctxData),
	}
	decision, err := o.approvalGate.RequestApproval(ctx, req)
	if err != nil {
		return true, fmt.Errorf("request approval for stage %s at %s: %w", stage.Name, point, err)
	}

	switch decision.Status {
	case "", ApprovalApproved:
		mergeInto(ctxData, decision.Context)
		return false, nil
	case ApprovalPending:
		runResult.Status = RunStatusPendingApproval
		runResult.PendingApproval = &req
		return true, nil
	case ApprovalRejected:
		runResult.Status = RunStatusRejected
		runResult.RejectionReason = decision.Reason
		return true, nil
	default:
		return true, fmt.Errorf("unknown approval status %q", decision.Status)
	}
}

func validatePipeline(def PipelineDefinition) error {
	if len(def.Stages) == 0 {
		return ErrPipelineHasNoStage
	}
	knownStages := make(map[string]bool, len(def.Stages))
	for _, stage := range def.Stages {
		if stage.Name == "" {
			return ErrStageNameRequired
		}
		if stage.AgentType == "" {
			return ErrStageAgentTypeRequired
		}
		if knownStages[stage.Name] {
			return fmt.Errorf("%w: %s", ErrStageNameDuplicated, stage.Name)
		}
		knownStages[stage.Name] = true
	}
	for _, stage := range def.Stages {
		for _, dep := range stage.DependsOn {
			if !knownStages[dep] {
				return fmt.Errorf("%w: stage=%s dependency=%s", ErrStageDependsOnNotFound, stage.Name, dep)
			}
		}
	}
	return nil
}

func orderStages(stages []PipelineStage) ([]PipelineStage, error) {
	completed := make(map[string]bool, len(stages))
	remaining := make(map[string]PipelineStage, len(stages))
	for _, stage := range stages {
		remaining[stage.Name] = stage
	}

	ordered := make([]PipelineStage, 0, len(stages))
	for len(ordered) < len(stages) {
		progress := false
		for _, stage := range stages {
			if _, ok := remaining[stage.Name]; !ok {
				continue
			}
			if !dependenciesCompleted(stage, completed) {
				continue
			}
			ordered = append(ordered, stage)
			completed[stage.Name] = true
			delete(remaining, stage.Name)
			progress = true
		}
		if !progress {
			return nil, ErrStageDependencyCycle
		}
	}
	return ordered, nil
}

func dependenciesCompleted(stage PipelineStage, completed map[string]bool) bool {
	for _, dep := range stage.DependsOn {
		if !completed[dep] {
			return false
		}
	}
	return true
}

func ensureStageReady(stage PipelineStage, completed map[string]bool) error {
	for _, dep := range stage.DependsOn {
		if !completed[dep] {
			return fmt.Errorf("%w: stage=%s dependency=%s", ErrStageDependencyNotReady, stage.Name, dep)
		}
	}
	return nil
}

func stageRequiresApproval(stage PipelineStage, point ApprovalPoint) bool {
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

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func mergeInto(dst map[string]any, src map[string]any) {
	if dst == nil || src == nil {
		return
	}
	for k, v := range src {
		dst[k] = v
	}
}
