package agents

import (
	"context"
	"fmt"
)

type MarkdownDispatcher struct {
	registry *Registry
	runner   *Runner
}

func NewMarkdownDispatcher(registry *Registry, runner *Runner) *MarkdownDispatcher {
	return &MarkdownDispatcher{
		registry: registry,
		runner:   runner,
	}
}

func (d *MarkdownDispatcher) Dispatch(ctx context.Context, req DispatchRequest) (DispatchResult, error) {
	if d.registry == nil {
		return DispatchResult{}, fmt.Errorf("agent registry is required")
	}
	if d.runner == nil {
		return DispatchResult{}, fmt.Errorf("agent runner is required")
	}

	agent, ok := d.registry.Get(req.PlanItem.AgentType)
	if !ok {
		return DispatchResult{}, fmt.Errorf("agent %q not registered", req.PlanItem.AgentType)
	}

	result, err := d.runner.Run(ctx, AgentRunRequest{
		Agent:    agent,
		Pipeline: req.Pipeline,
		Stage:    req.Stage,
		PlanItem: req.PlanItem,
		Context:  req.Context,
	})
	if err != nil {
		return DispatchResult{}, err
	}

	meta := map[string]any{
		"agent": agent.Name,
	}
	if agent.Model != "" {
		meta["model"] = agent.Model
	}
	if result.TokenUsage != nil {
		meta["token_usage"] = result.TokenUsage
	}
	if len(result.ToolCalls) > 0 {
		meta["tool_calls"] = result.ToolCalls
	}

	return DispatchResult{
		Output: map[string]any{
			"content": result.Content,
		},
		Meta: meta,
	}, nil
}
