package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/plutolove233/co-dream/internal/llm"
	"github.com/plutolove233/co-dream/internal/llm/agents"
	"github.com/plutolove233/co-dream/internal/tools"
	"github.com/plutolove233/co-dream/pkg/interfaces"
	"github.com/plutolove233/co-dream/pkg/types"
	openai "github.com/sashabaranov/go-openai"
)

const realAPIChannelID = "real-api-code-change"

func main() {
	godotenv.Load()
	if err := run(context.Background()); err != nil {
		fmt.Printf("real api demo failed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := readConfig()
	if err != nil {
		return err
	}

	agentRegistry := agents.NewRegistry()
	for _, agent := range realAPIAgents() {
		if err := agentRegistry.Register(agent); err != nil {
			return fmt.Errorf("register agent %s: %w", agent.Name, err)
		}
	}

	var runtime *agents.CollaborationRuntime
	runtime = agents.NewCollaborationRuntime(agentRegistry, func(agent agents.AgentDefinition) (agents.AgentRunner, error) {
		toolRegistry := tools.NewRegistry()
		if err := tools.RegisterAgentCollaborationTools(toolRegistry, runtime, tools.AgentToolContext{}); err != nil {
			return nil, fmt.Errorf("register sub-agent tools: %w", err)
		}
		filteredTools := toolRegistry.Filter(agent.Tools)
		client := llm.NewClient(cfg.apiKey, cfg.baseURL, cfg.model, filteredTools)
		streamingClient := streamReportingClient{
			provider:  client,
			runtime:   runtime,
			agentName: agent.Name,
			channelID: realAPIChannelID,
		}
		return agents.NewRunner(streamingClient, agents.WithMaxToolRounds(4)), nil
	})

	pipeline := agents.PipelineDefinition{
		ID:   "real-api-demo-pipeline",
		Name: "Real API Requirement to Reviewed Patch",
		Stages: []agents.PipelineStage{
			{
				Name:      "generate-code",
				Objective: "Generate a small implementation plan for a /health endpoint, then share it with the reviewer through the channel.",
				AgentType: "main_agent",
				Input: map[string]any{
					"delegate_agent": "code_generator",
				},
			},
			{
				Name:      "review-code",
				Objective: "Read the generator's channel message and review the proposed implementation.",
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
		return dispatchWithRealAgentTool(ctx, runtime, req)
	})

	done := make(chan struct{})
	go printRuntimeEvents(runtime, done)
	defer close(done)

	fmt.Printf("PIPELINE started: %s (%s)\n", pipeline.Name, pipeline.ID)
	fmt.Printf("LLM config: model=%s base_url=%s\n", cfg.model, cfg.baseURL)

	orchestrator := agents.NewPipelineOrchestrator(consolePlanner{}, dispatcher)
	result, err := orchestrator.Run(ctx, pipeline, map[string]any{
		"requirement": "Add a /health endpoint that returns HTTP 200 and a JSON payload like {\"status\":\"ok\"}.",
		"repo_path":   ".",
	})
	if err != nil {
		return fmt.Errorf("run pipeline: %w", err)
	}

	time.Sleep(200 * time.Millisecond)
	fmt.Printf("\nPIPELINE completed: status=%s stages=%d\n", result.Status, len(result.Stages))
	for _, stage := range result.Stages {
		fmt.Printf("  - %s: %d item(s)\n", stage.StageName, len(stage.Items))
	}
	return nil
}

type config struct {
	apiKey  string
	baseURL string
	model   string
}

func readConfig() (config, error) {
	cfg := config{
		apiKey:  os.Getenv("API_KEY"),
		baseURL: os.Getenv("BASE_URL"),
		model:   os.Getenv("MODEL_EPIP"),
	}
	if cfg.apiKey == "" {
		return config{}, fmt.Errorf("CODEDREAM_LLM_API_KEY or OPENAI_API_KEY is required")
	}
	return cfg, nil
}

func realAPIAgents() []agents.AgentDefinition {
	return []agents.AgentDefinition{
		{
			Name:        "code_generator",
			Description: "Generates a small code patch plan from a requirement.",
			Tools:       []string{"send_agent_message", "read_agent_messages"},
			Instructions: strings.TrimSpace(`
You are a CoDream code generator sub-agent.

Work as a ReAct agent. You must:
- Use send_agent_message exactly once to share your proposed patch with code_reviewer.
- Keep the proposed patch small and aligned with router -> handler -> service boundaries.
- Do not claim you modified files; this example validates orchestration, not filesystem edits.
- Then provide a concise final answer with the proposed files, behavior, and tests.
`),
		},
		{
			Name:        "code_reviewer",
			Description: "Reviews the generated plan and asks for concrete fixes when needed.",
			Tools:       []string{"read_agent_messages", "send_agent_message"},
			Instructions: strings.TrimSpace(`
You are a CoDream code reviewer sub-agent.

Work as a ReAct agent. You must:
- Use read_agent_messages to inspect messages in the shared channel before reviewing.
- Review the generator proposal for layering, scope, and tests.
- Use send_agent_message only if you need to send a correction back.
- Then provide a concise final review verdict.
`),
		},
	}
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

func dispatchWithRealAgentTool(ctx context.Context, runtime *agents.CollaborationRuntime, req agents.DispatchRequest) (agents.DispatchResult, error) {
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
		AgentName:     delegateAgent,
		TaskID:        req.PlanItem.TaskID,
		Instructions:  req.PlanItem.Description,
		ChannelID:     realAPIChannelID,
		Context:       map[string]any{"stage_input": req.PlanItem.Input},
		MaxToolRounds: 4,
		Metadata:      map[string]any{"stage_name": req.Stage.Name},
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
	if subAgentResult.TokenUsage != nil {
		fmt.Printf("TOKEN usage [%s]: prompt=%d completion=%d total=%d\n",
			subAgentResult.AgentName,
			subAgentResult.TokenUsage.PromptTokens,
			subAgentResult.TokenUsage.CompletionTokens,
			subAgentResult.TokenUsage.TotalTokens,
		)
	}

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
		ChannelID: realAPIChannelID,
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
		fmt.Printf("  %s -> %s: %s\n", msg.FromAgent, msg.ToAgent, truncate(msg.Content, 180))
	}
	return nil
}

type streamReportingClient struct {
	provider  interfaces.LLMProvider
	runtime   *agents.CollaborationRuntime
	agentName string
	channelID string
}

func (c streamReportingClient) Model() string {
	return c.provider.Model()
}

func (c streamReportingClient) ExecuteTools(ctx context.Context, toolCalls []openai.ToolCall) ([]types.ToolCallResult, error) {
	return c.provider.ExecuteTools(ctx, toolCalls)
}

func (c streamReportingClient) Complete(ctx context.Context, messages []types.Message, system string) (<-chan types.CompleteEvent, error) {
	events, err := c.provider.Complete(ctx, messages, system)
	if err != nil {
		return nil, err
	}

	out := make(chan types.CompleteEvent, 16)
	go func() {
		defer close(out)
		var buffer strings.Builder
		flush := func() {
			text := strings.TrimSpace(buffer.String())
			if text == "" {
				buffer.Reset()
				return
			}
			_, _ = c.runtime.SendMessage(c.channelID, c.agentName, "", "MODEL stream: "+text, nil)
			buffer.Reset()
		}

		for event := range events {
			if event.Delta != "" {
				buffer.WriteString(event.Delta)
				if strings.ContainsAny(event.Delta, "\n。.!?") || buffer.Len() >= 160 {
					flush()
				}
			}
			if event.Result != nil || event.Err != nil {
				flush()
			}

			select {
			case out <- event:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

func printRuntimeEvents(runtime *agents.CollaborationRuntime, done <-chan struct{}) {
	ticker := time.NewTicker(120 * time.Millisecond)
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
			fmt.Printf("EVENT channel_message: from=%s to=%v content=%v\n",
				event.AgentName,
				event.Payload["to_agent"],
				truncate(fmt.Sprint(event.Payload["content"]), 220),
			)
		}
	}
}

func indent(s, prefix string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
