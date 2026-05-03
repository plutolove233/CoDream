package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/plutolove233/co-dream/pkg/types"
	openai "github.com/sashabaranov/go-openai"
)

const defaultMaxToolRounds = 32

type CompletionClient interface {
	Complete(ctx context.Context, messages []types.Message, system string) (<-chan types.CompleteEvent, error)
}

type ToolExecutor interface {
	ExecuteTools(ctx context.Context, toolCalls []openai.ToolCall) ([]types.ToolCallResult, error)
}

type AgentRunRequest struct {
	Agent         AgentDefinition    `json:"agent"`
	Pipeline      PipelineDefinition `json:"pipeline"`
	Stage         PipelineStage      `json:"stage"`
	PlanItem      StagePlanItem      `json:"plan_item"`
	Context       map[string]any     `json:"context,omitempty"`
	MaxToolRounds int                `json:"max_tool_rounds,omitempty"`
}

type AgentRunResult struct {
	Content    string                 `json:"content"`
	ToolCalls  []types.ToolCallResult `json:"tool_calls,omitempty"`
	Trace      []ReActStep            `json:"trace,omitempty"`
	TokenUsage *types.TokenUsage      `json:"token_usage,omitempty"`
}

type ReActStep struct {
	Round        int                    `json:"round"`
	Thought      string                 `json:"thought,omitempty"`
	Actions      []ReActAction          `json:"actions,omitempty"`
	Observations []types.ToolCallResult `json:"observations,omitempty"`
	Final        bool                   `json:"final,omitempty"`
}

type ReActAction struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Arguments  string `json:"arguments,omitempty"`
}

type RunnerOption func(*Runner)

type Runner struct {
	client        CompletionClient
	toolExecutor  ToolExecutor
	maxToolRounds int
}

func NewRunner(client CompletionClient, opts ...RunnerOption) *Runner {
	r := &Runner{
		client:        client,
		maxToolRounds: defaultMaxToolRounds,
	}
	if toolExecutor, ok := client.(ToolExecutor); ok {
		r.toolExecutor = toolExecutor
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func WithToolExecutor(toolExecutor ToolExecutor) RunnerOption {
	return func(r *Runner) {
		r.toolExecutor = toolExecutor
	}
}

func WithMaxToolRounds(maxToolRounds int) RunnerOption {
	return func(r *Runner) {
		if maxToolRounds > 0 {
			r.maxToolRounds = maxToolRounds
		}
	}
}

func (r *Runner) Run(ctx context.Context, req AgentRunRequest) (AgentRunResult, error) {
	if r.client == nil {
		return AgentRunResult{}, fmt.Errorf("completion client is required")
	}
	if req.Agent.Name == "" {
		return AgentRunResult{}, fmt.Errorf("agent definition is required")
	}

	messages := []types.Message{{
		Role:    openai.ChatMessageRoleUser,
		Content: buildAgentUserPrompt(req),
	}}

	var final AgentRunResult
	maxToolRounds := r.maxToolRounds
	if req.MaxToolRounds > 0 {
		maxToolRounds = req.MaxToolRounds
	}
	for round := 1; round <= maxToolRounds+1; round++ {
		result, err := r.completeOnce(ctx, req.Agent, messages)
		if err != nil {
			return AgentRunResult{}, err
		}

		final.Content = result.Content
		final.TokenUsage = result.Usage
		step := ReActStep{
			Round:   round,
			Thought: result.Content,
		}
		if len(result.ToolCalls) == 0 {
			step.Final = true
			final.Trace = append(final.Trace, step)
			return final, nil
		}
		if round > maxToolRounds {
			return AgentRunResult{}, fmt.Errorf("agent %s exceeded max tool rounds", req.Agent.Name)
		}
		if r.toolExecutor == nil {
			return AgentRunResult{}, fmt.Errorf("agent %s requested tools but no tool executor is configured", req.Agent.Name)
		}

		toolResults, err := r.toolExecutor.ExecuteTools(ctx, result.ToolCalls)
		if err != nil {
			return AgentRunResult{}, fmt.Errorf("execute tools for agent %s: %w", req.Agent.Name, err)
		}
		step.Actions = buildReActActions(result.ToolCalls)
		step.Observations = toolResults
		final.Trace = append(final.Trace, step)
		final.ToolCalls = append(final.ToolCalls, toolResults...)
		messages = append(messages,
			types.Message{
				Role:      openai.ChatMessageRoleAssistant,
				Content:   result.Content,
				ToolCalls: result.ToolCalls,
			},
			types.Message{
				Role:        openai.ChatMessageRoleTool,
				ToolResults: toolResults,
			},
		)
	}

	return AgentRunResult{}, fmt.Errorf("agent %s exceeded max tool rounds", req.Agent.Name)
}

func (r *Runner) completeOnce(ctx context.Context, agent AgentDefinition, messages []types.Message) (*types.CompleteResult, error) {
	events, err := r.client.Complete(ctx, messages, buildAgentSystemPrompt(agent))
	if err != nil {
		return nil, fmt.Errorf("complete agent %s: %w", agent.Name, err)
	}

	var result *types.CompleteResult
	for event := range events {
		if event.Err != nil {
			return nil, event.Err
		}
		if event.Result != nil {
			result = event.Result
		}
	}
	if result == nil {
		return nil, fmt.Errorf("agent %s returned no result", agent.Name)
	}
	return result, nil
}

func buildAgentSystemPrompt(agent AgentDefinition) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Agent: %s\n", agent.Name)
	if agent.Description != "" {
		fmt.Fprintf(&builder, "Description: %s\n", agent.Description)
	}
	if len(agent.Tools) > 0 {
		fmt.Fprintf(&builder, "Allowed tools: %s\n", strings.Join(agent.Tools, ", "))
	}
	builder.WriteString("\nExecution protocol:\n")
	builder.WriteString("- Work as a ReAct agent: reason about the current state, call tools when observation is needed, then continue from the tool observations.\n")
	builder.WriteString("- Use multiple tool rounds when the task requires incremental implementation, inspection, testing, and fixes.\n")
	builder.WriteString("- Finish only when the assigned pipeline task has a concrete final answer and no further tool observation is needed.\n")
	if agent.Instructions != "" {
		builder.WriteString("\nInstructions:\n")
		builder.WriteString(agent.Instructions)
	}
	return strings.TrimSpace(builder.String())
}

func buildReActActions(toolCalls []openai.ToolCall) []ReActAction {
	actions := make([]ReActAction, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		actions = append(actions, ReActAction{
			ToolCallID: toolCall.ID,
			Name:       toolCall.Function.Name,
			Arguments:  toolCall.Function.Arguments,
		})
	}
	return actions
}

func buildAgentUserPrompt(req AgentRunRequest) string {
	payload := map[string]any{
		"pipeline":  req.Pipeline,
		"stage":     req.Stage,
		"plan_item": req.PlanItem,
		"context":   req.Context,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Sprintf("Run stage %s task %s.", req.Stage.Name, req.PlanItem.TaskID)
	}
	return "Execute the assigned pipeline task using the structured input below.\n\n" + string(data)
}
