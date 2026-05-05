package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/plutolove233/co-dream/internal/llm/agents"
	"github.com/plutolove233/co-dream/internal/tools"
	"github.com/plutolove233/co-dream/pkg/interfaces"
)

const demoChannelID = "demo-code-change"

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Printf("demo failed: %v\n", err)
	}
}

func run(ctx context.Context) error {
	agentRegistry := agents.NewRegistry()
	for _, agent := range []agents.AgentDefinition{
		{
			Name:        "code_generator",
			Description: "Generates a small code patch from a requirement.",
			Tools:       []string{"send_agent_message", "read_agent_messages"},
		},
		{
			Name:        "code_reviewer",
			Description: "Reviews generated code and asks for follow-up changes.",
			Tools:       []string{"send_agent_message", "read_agent_messages"},
		},
	} {
		if err := agentRegistry.Register(agent); err != nil {
			return fmt.Errorf("register agent %s: %w", agent.Name, err)
		}
	}

	var runtime *agents.CollaborationRuntime
	runtime = agents.NewCollaborationRuntime(agentRegistry, func(agent agents.AgentDefinition) (agents.AgentRunner, error) {
		return demoAgentRunner{runtime: runtime, agent: agent}, nil
	})

	pipeline := agents.PipelineDefinition{
		ID:   "demo-pipeline",
		Name: "Requirement to Reviewed Patch",
		Stages: []agents.PipelineStage{
			{
				Name:      "generate-code",
				Objective: "Generate a focused patch for the requirement.",
				AgentType: "main_agent",
				Input: map[string]any{
					"delegate_agent": "code_generator",
				},
			},
			{
				Name:      "review-code",
				Objective: "Review the generated patch and produce actionable feedback.",
				AgentType: "main_agent",
				DependsOn: []string{
					"generate-code",
				},
				Input: map[string]any{
					"delegate_agent": "code_reviewer",
				},
			},
		},
	}

	dispatcher := agents.NewMapDispatcher()
	dispatcher.Register("main_agent", func(ctx context.Context, req agents.DispatchRequest) (agents.DispatchResult, error) {
		return dispatchWithAgentTools(ctx, runtime, req)
	})

	done := make(chan struct{})
	go printRuntimeEvents(runtime, done)
	defer close(done)

	fmt.Printf("PIPELINE started: %s (%s)\n", pipeline.Name, pipeline.ID)
	orchestrator := agents.NewPipelineOrchestrator(consolePlanner{}, dispatcher)
	result, err := orchestrator.Run(ctx, pipeline, map[string]any{
		"requirement": "Add a /health endpoint that returns HTTP 200 and a JSON status payload.",
		"repo_path":   ".",
	})
	if err != nil {
		return fmt.Errorf("run pipeline: %w", err)
	}

	time.Sleep(150 * time.Millisecond)
	fmt.Printf("\nPIPELINE completed: status=%s stages=%d\n", result.Status, len(result.Stages))
	for _, stage := range result.Stages {
		fmt.Printf("  - %s: %d item(s)\n", stage.StageName, len(stage.Items))
	}
	return nil
}

type consolePlanner struct{}

func (consolePlanner) PlanStage(_ context.Context, req agents.PlanRequest) (agents.StagePlan, error) {
	fmt.Printf("\nSTAGE planning: %s\n", req.Stage.Name)
	description := req.Stage.Objective
	if description == "" {
		description = "execute stage " + req.Stage.Name
	}
	return agents.StagePlan{
		StageName: req.Stage.Name,
		Items: []agents.StagePlanItem{{
			TaskID:      req.Stage.Name + "-task-1",
			StageName:   req.Stage.Name,
			Description: description,
			AgentType:   req.Stage.AgentType,
			Input:       req.Stage.Input,
		}},
	}, nil
}

func dispatchWithAgentTools(ctx context.Context, runtime *agents.CollaborationRuntime, req agents.DispatchRequest) (agents.DispatchResult, error) {
	delegateAgent, ok := req.PlanItem.Input["delegate_agent"].(string)
	if !ok || delegateAgent == "" {
		return agents.DispatchResult{}, fmt.Errorf("stage %s missing delegate_agent", req.Stage.Name)
	}

	fmt.Printf("STAGE dispatch: %s -> main_agent uses run_agent(%s)\n", req.Stage.Name, delegateAgent)

	toolRegistry := tools.NewRegistry()
	if err := tools.RegisterAgentCollaborationTools(toolRegistry, runtime, tools.AgentToolContext{
		Pipeline:    req.Pipeline,
		Stage:       req.Stage,
		BaseContext: req.Context,
	}); err != nil {
		return agents.DispatchResult{}, fmt.Errorf("register agent tools: %w", err)
	}

	if req.Stage.Name == "review-code" {
		if err := printExistingChannelMessages(ctx, toolRegistry); err != nil {
			return agents.DispatchResult{}, err
		}
	}

	runTool, ok := toolRegistry.Get("run_agent")
	if !ok {
		return agents.DispatchResult{}, fmt.Errorf("run_agent tool not registered")
	}

	input, err := json.Marshal(tools.RunAgentInput{
		AgentName:    delegateAgent,
		TaskID:       req.PlanItem.TaskID,
		Instructions: req.PlanItem.Description,
		ChannelID:    demoChannelID,
		Context: map[string]any{
			"stage_input": req.PlanItem.Input,
		},
		Metadata: map[string]any{
			"stage_name": req.Stage.Name,
		},
	})
	if err != nil {
		return agents.DispatchResult{}, fmt.Errorf("marshal run_agent input: %w", err)
	}

	raw, err := runTool.Execute(ctx, input)
	if err != nil {
		return agents.DispatchResult{}, fmt.Errorf("execute run_agent: %w", err)
	}

	var subAgentResult agents.SubAgentRunResult
	if err := json.Unmarshal([]byte(raw), &subAgentResult); err != nil {
		return agents.DispatchResult{}, fmt.Errorf("decode run_agent output: %w", err)
	}

	fmt.Printf("MODEL final output [%s]:\n%s\n", subAgentResult.AgentName, indent(subAgentResult.Content, "  "))

	return agents.DispatchResult{
		Output: map[string]any{
			req.Stage.Name + "_output": subAgentResult.Content,
			"last_agent_output":        subAgentResult.Content,
		},
		Meta: map[string]any{
			"agent_name": subAgentResult.AgentName,
			"task_id":    subAgentResult.TaskID,
		},
	}, nil
}

func printExistingChannelMessages(ctx context.Context, registry interfaces.ToolRegistry) error {
	readTool, ok := registry.Get("read_agent_messages")
	if !ok {
		return fmt.Errorf("read_agent_messages tool not registered")
	}
	input, err := json.Marshal(tools.ReadAgentMessagesInput{
		ChannelID: demoChannelID,
		ToAgent:   "code_reviewer",
		Limit:     10,
	})
	if err != nil {
		return fmt.Errorf("marshal read_agent_messages input: %w", err)
	}
	raw, err := readTool.Execute(ctx, input)
	if err != nil {
		return fmt.Errorf("execute read_agent_messages: %w", err)
	}

	var payload struct {
		Messages []agents.ChannelMessage `json:"messages"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return fmt.Errorf("decode read_agent_messages output: %w", err)
	}
	fmt.Printf("CHANNEL preload for reviewer: %d message(s)\n", len(payload.Messages))
	for _, msg := range payload.Messages {
		fmt.Printf("  %s -> %s: %s\n", msg.FromAgent, msg.ToAgent, msg.Content)
	}
	return nil
}

type demoAgentRunner struct {
	runtime *agents.CollaborationRuntime
	agent   agents.AgentDefinition
}

func (r demoAgentRunner) Run(ctx context.Context, req agents.AgentRunRequest) (agents.AgentRunResult, error) {
	switch r.agent.Name {
	case "code_generator":
		if err := r.publishModelChunk(ctx, "I inspected the requirement and will produce a small handler-level patch plan."); err != nil {
			return agents.AgentRunResult{}, err
		}
		if err := r.publishModelChunk(ctx, "Generated patch sketch: add GET /health route, return {\"status\":\"ok\"}."); err != nil {
			return agents.AgentRunResult{}, err
		}
		if _, err := r.runtime.SendMessage(demoChannelID, r.agent.Name, "code_reviewer", "Patch is ready for review: add /health route and JSON status response.", map[string]any{
			"task_id": req.PlanItem.TaskID,
		}); err != nil {
			return agents.AgentRunResult{}, fmt.Errorf("send generator message: %w", err)
		}
		return agents.AgentRunResult{Content: strings.TrimSpace(`
Proposed code change:
- Register GET /health in the HTTP router.
- Add a lightweight handler returning HTTP 200.
- Response body: {"status":"ok"}.
`)}, nil
	case "code_reviewer":
		if err := r.publishModelChunk(ctx, "I read the generator message and checked the expected API behavior."); err != nil {
			return agents.AgentRunResult{}, err
		}
		if err := r.publishModelChunk(ctx, "Review result: implementation is acceptable if covered by a handler test."); err != nil {
			return agents.AgentRunResult{}, err
		}
		return agents.AgentRunResult{Content: strings.TrimSpace(`
Review:
- The /health endpoint behavior is clear and low-risk.
- Add a regression test for HTTP 200 and the JSON status payload before merging.
- Keep the route registration in router layer and response shaping in handler layer.
`)}, nil
	default:
		return agents.AgentRunResult{}, fmt.Errorf("unsupported demo agent %s", r.agent.Name)
	}
}

func (r demoAgentRunner) publishModelChunk(ctx context.Context, content string) error {
	if err := sleep(ctx, 350*time.Millisecond); err != nil {
		return err
	}
	if _, err := r.runtime.SendMessage(demoChannelID, r.agent.Name, "", "MODEL stream: "+content, nil); err != nil {
		return fmt.Errorf("publish model chunk: %w", err)
	}
	return nil
}

func printRuntimeEvents(runtime *agents.CollaborationRuntime, done <-chan struct{}) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var afterID string
	for {
		select {
		case <-done:
			printEvents(runtime.EventsAfter(afterID, 0), &afterID)
			return
		case <-ticker.C:
			printEvents(runtime.EventsAfter(afterID, 0), &afterID)
		}
	}
}

func printEvents(events []agents.CollaborationEvent, afterID *string) {
	for _, event := range events {
		*afterID = event.ID
		switch event.Type {
		case agents.CollaborationEventAgentStarted:
			fmt.Printf("EVENT agent_started: agent=%s task=%v\n", event.AgentName, event.Payload["task_id"])
		case agents.CollaborationEventAgentCompleted:
			fmt.Printf("EVENT agent_completed: agent=%s task=%v\n", event.AgentName, event.Payload["task_id"])
		case agents.CollaborationEventAgentFailed:
			fmt.Printf("EVENT agent_failed: agent=%s error=%v\n", event.AgentName, event.Payload["error"])
		case agents.CollaborationEventChannelMessage:
			fmt.Printf("EVENT channel_message: from=%s to=%v content=%v\n", event.AgentName, event.Payload["to_agent"], event.Payload["content"])
		}
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func indent(s, prefix string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
