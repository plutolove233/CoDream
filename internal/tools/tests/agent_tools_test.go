package tests

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/plutolove233/co-dream/internal/llm/agents"
	"github.com/plutolove233/co-dream/internal/tools"
)

type fakeSubAgentRunner struct {
	reqs []agents.AgentRunRequest
}

func (r *fakeSubAgentRunner) Run(_ context.Context, req agents.AgentRunRequest) (agents.AgentRunResult, error) {
	r.reqs = append(r.reqs, req)
	req.Context["mutated"] = true
	return agents.AgentRunResult{Content: "review complete"}, nil
}

func TestRunAgentToolExecutesIsolatedSubAgent(t *testing.T) {
	registry := agents.NewRegistry()
	if err := registry.Register(agents.AgentDefinition{
		Name:  "code_reviewer",
		Tools: []string{"file_handler"},
	}); err != nil {
		t.Fatalf("register agent: %v", err)
	}

	runner := &fakeSubAgentRunner{}
	var factoryAgent agents.AgentDefinition
	runtime := agents.NewCollaborationRuntime(registry, func(agent agents.AgentDefinition) (agents.AgentRunner, error) {
		factoryAgent = agent
		return runner, nil
	})

	baseContext := map[string]any{"requirement": "build feature"}
	tool := tools.NewRunAgentTool(runtime, tools.AgentToolContext{
		Pipeline:    agents.PipelineDefinition{ID: "p1"},
		Stage:       agents.PipelineStage{Name: "implement"},
		BaseContext: baseContext,
	})

	input, err := json.Marshal(tools.RunAgentInput{
		AgentName:    "code_reviewer",
		TaskID:       "review-1",
		Instructions: "review the generated patch",
		ChannelID:    "stage-implement",
		Context:      map[string]any{"patch": "diff --git"},
	})
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}

	output, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("execute run_agent: %v", err)
	}

	var result agents.SubAgentRunResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if result.AgentName != "code_reviewer" || result.Content != "review complete" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if factoryAgent.Name != "code_reviewer" || len(factoryAgent.Tools) != 1 || factoryAgent.Tools[0] != "file_handler" {
		t.Fatalf("factory did not receive agent tool contract: %+v", factoryAgent)
	}
	if len(runner.reqs) != 1 {
		t.Fatalf("expected one runner request, got %d", len(runner.reqs))
	}
	if runner.reqs[0].Context["requirement"] != "build feature" || runner.reqs[0].Context["patch"] != "diff --git" {
		t.Fatalf("unexpected merged context: %+v", runner.reqs[0].Context)
	}
	if baseContext["mutated"] == true {
		t.Fatal("base context was mutated by sub-agent")
	}
}

func TestAgentMessageToolsSendAndReadChannelMessages(t *testing.T) {
	runtime := agents.NewCollaborationRuntime(agents.NewRegistry(), nil)
	sendTool := tools.NewSendAgentMessageTool(runtime)
	readTool := tools.NewReadAgentMessagesTool(runtime)

	firstInput, err := json.Marshal(tools.SendAgentMessageInput{
		ChannelID: "stage-implement",
		FromAgent: "code_generator",
		ToAgent:   "code_reviewer",
		Content:   "patch is ready",
	})
	if err != nil {
		t.Fatalf("marshal first input: %v", err)
	}
	if _, err := sendTool.Execute(context.Background(), firstInput); err != nil {
		t.Fatalf("send first message: %v", err)
	}

	secondInput, err := json.Marshal(tools.SendAgentMessageInput{
		ChannelID: "stage-implement",
		FromAgent: "code_reviewer",
		ToAgent:   "code_generator",
		Content:   "add a regression test",
	})
	if err != nil {
		t.Fatalf("marshal second input: %v", err)
	}
	if _, err := sendTool.Execute(context.Background(), secondInput); err != nil {
		t.Fatalf("send second message: %v", err)
	}

	readInput, err := json.Marshal(tools.ReadAgentMessagesInput{
		ChannelID: "stage-implement",
		ToAgent:   "code_generator",
	})
	if err != nil {
		t.Fatalf("marshal read input: %v", err)
	}
	output, err := readTool.Execute(context.Background(), readInput)
	if err != nil {
		t.Fatalf("read messages: %v", err)
	}

	var payload struct {
		Messages []agents.ChannelMessage `json:"messages"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal messages: %v", err)
	}
	if len(payload.Messages) != 1 || payload.Messages[0].Content != "add a regression test" {
		t.Fatalf("unexpected messages: %+v", payload.Messages)
	}
}

func TestRegistryFilterRemovesRecursiveRunAgentTool(t *testing.T) {
	registry := tools.NewRegistry()
	if err := registry.Register(tools.NewRunAgentTool(nil, tools.AgentToolContext{})); err != nil {
		t.Fatalf("register run_agent: %v", err)
	}
	if err := registry.Register(tools.NewReadAgentMessagesTool(nil)); err != nil {
		t.Fatalf("register read_agent_messages: %v", err)
	}

	filtered := registry.Filter([]string{"run_agent", "read_agent_messages"})
	if _, ok := filtered.Get("run_agent"); ok {
		t.Fatal("expected run_agent to be removed from filtered sub-agent registry")
	}
	if _, ok := filtered.Get("read_agent_messages"); !ok {
		t.Fatal("expected read_agent_messages to remain available")
	}
}
