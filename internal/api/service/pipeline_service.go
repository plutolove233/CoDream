package service

import "github.com/plutolove233/co-dream/internal/llm/agents"

type RunStatus = agents.RunStatus
type ApprovalPoint = agents.ApprovalPoint
type ApprovalStatus = agents.ApprovalStatus
type AgentDefinition = agents.AgentDefinition
type PipelineDefinition = agents.PipelineDefinition
type PipelineStage = agents.PipelineStage
type StagePlanItem = agents.StagePlanItem
type StagePlan = agents.StagePlan
type PlanRequest = agents.PlanRequest
type StagePlanner = agents.StagePlanner
type DefaultStagePlanner = agents.DefaultStagePlanner
type DispatchRequest = agents.DispatchRequest
type DispatchResult = agents.DispatchResult
type AgentDispatcher = agents.AgentDispatcher
type AgentFunc = agents.AgentFunc
type MapDispatcher = agents.MapDispatcher
type StageExecutionResult = agents.StageExecutionResult
type DispatchItem = agents.DispatchItem
type PipelineRunResult = agents.PipelineRunResult
type PipelineOrchestrator = agents.PipelineOrchestrator
type ApprovalRequest = agents.ApprovalRequest
type ApprovalDecision = agents.ApprovalDecision
type ApprovalGate = agents.ApprovalGate
type Registry = agents.Registry
type Loader = agents.Loader
type Runner = agents.Runner
type MarkdownDispatcher = agents.MarkdownDispatcher
type CompletionClient = agents.CompletionClient
type ToolExecutor = agents.ToolExecutor
type AgentRunRequest = agents.AgentRunRequest
type AgentRunResult = agents.AgentRunResult
type ReActStep = agents.ReActStep
type ReActAction = agents.ReActAction
type RunnerOption = agents.RunnerOption
type CollaborationEventType = agents.CollaborationEventType
type CollaborationEvent = agents.CollaborationEvent
type ChannelMessage = agents.ChannelMessage
type SubAgentRunRequest = agents.SubAgentRunRequest
type SubAgentRunResult = agents.SubAgentRunResult
type AgentRunner = agents.AgentRunner
type RunnerFactory = agents.RunnerFactory
type CollaborationRuntime = agents.CollaborationRuntime

var (
	ErrPipelineHasNoStage      = agents.ErrPipelineHasNoStage
	ErrStageNameRequired       = agents.ErrStageNameRequired
	ErrStageNameDuplicated     = agents.ErrStageNameDuplicated
	ErrStageAgentTypeRequired  = agents.ErrStageAgentTypeRequired
	ErrStageDependsOnNotFound  = agents.ErrStageDependsOnNotFound
	ErrStageDependencyNotReady = agents.ErrStageDependencyNotReady
	ErrStageDependencyCycle    = agents.ErrStageDependencyCycle
)

const (
	RunStatusCompleted       = agents.RunStatusCompleted
	RunStatusPendingApproval = agents.RunStatusPendingApproval
	RunStatusRejected        = agents.RunStatusRejected

	ApprovalBeforeStage   = agents.ApprovalBeforeStage
	ApprovalAfterPlan     = agents.ApprovalAfterPlan
	ApprovalBeforeExecute = agents.ApprovalBeforeExecute
	ApprovalAfterStage    = agents.ApprovalAfterStage

	ApprovalApproved = agents.ApprovalApproved
	ApprovalPending  = agents.ApprovalPending
	ApprovalRejected = agents.ApprovalRejected

	CollaborationEventAgentStarted   = agents.CollaborationEventAgentStarted
	CollaborationEventAgentCompleted = agents.CollaborationEventAgentCompleted
	CollaborationEventAgentFailed    = agents.CollaborationEventAgentFailed
	CollaborationEventChannelMessage = agents.CollaborationEventChannelMessage
)

func NewAgentRegistry() *Registry {
	return agents.NewRegistry()
}

func NewAgentLoader(agentsDir string) *Loader {
	return agents.NewLoader(agentsDir)
}

func NewRunner(client CompletionClient, opts ...RunnerOption) *Runner {
	return agents.NewRunner(client, opts...)
}

func WithToolExecutor(toolExecutor ToolExecutor) RunnerOption {
	return agents.WithToolExecutor(toolExecutor)
}

func WithMaxToolRounds(maxToolRounds int) RunnerOption {
	return agents.WithMaxToolRounds(maxToolRounds)
}

func NewMarkdownDispatcher(registry *Registry, runner *Runner) *MarkdownDispatcher {
	return agents.NewMarkdownDispatcher(registry, runner)
}

func NewMapDispatcher() *MapDispatcher {
	return agents.NewMapDispatcher()
}

func NewPipelineOrchestrator(planner StagePlanner, dispatcher AgentDispatcher) *PipelineOrchestrator {
	return agents.NewPipelineOrchestrator(planner, dispatcher)
}

func NewCollaborationRuntime(registry *Registry, runnerFactory RunnerFactory) *CollaborationRuntime {
	return agents.NewCollaborationRuntime(registry, runnerFactory)
}
