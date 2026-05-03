package agents

import (
	"context"
	"strings"
	"testing"

	"github.com/plutolove233/co-dream/pkg/types"
	openai "github.com/sashabaranov/go-openai"
)

type fakeCompletionClient struct {
	result *types.CompleteResult
}

func (c fakeCompletionClient) Complete(_ context.Context, _ []types.Message, _ string) (<-chan types.CompleteEvent, error) {
	events := make(chan types.CompleteEvent, 1)
	events <- types.CompleteEvent{Result: c.result}
	close(events)
	return events, nil
}

type sequenceCompletionClient struct {
	results  []*types.CompleteResult
	messages [][]types.Message
	systems  []string
}

func (c *sequenceCompletionClient) Complete(_ context.Context, messages []types.Message, system string) (<-chan types.CompleteEvent, error) {
	c.messages = append(c.messages, messages)
	c.systems = append(c.systems, system)

	events := make(chan types.CompleteEvent, 1)
	if len(c.results) > 0 {
		events <- types.CompleteEvent{Result: c.results[0]}
		c.results = c.results[1:]
	}
	close(events)
	return events, nil
}

type fakeToolExecutor struct {
	results []types.ToolCallResult
	calls   [][]openai.ToolCall
}

func (e *fakeToolExecutor) ExecuteTools(_ context.Context, toolCalls []openai.ToolCall) ([]types.ToolCallResult, error) {
	e.calls = append(e.calls, toolCalls)
	return e.results, nil
}

func TestRunnerRun(t *testing.T) {
	runner := NewRunner(fakeCompletionClient{
		result: &types.CompleteResult{Content: "implementation complete"},
	})

	result, err := runner.Run(context.Background(), AgentRunRequest{
		Agent: AgentDefinition{
			Name:         "code_generator",
			Instructions: "Write code.",
		},
		Pipeline: PipelineDefinition{ID: "p1"},
		Stage:    PipelineStage{Name: "implementation", AgentType: "code_generator"},
		PlanItem: StagePlanItem{TaskID: "t1", AgentType: "code_generator"},
		Context:  map[string]any{"requirement": "build feature"},
	})
	if err != nil {
		t.Fatalf("run agent: %v", err)
	}
	if result.Content != "implementation complete" {
		t.Fatalf("unexpected content: %s", result.Content)
	}
	if len(result.Trace) != 1 || !result.Trace[0].Final {
		t.Fatalf("expected final trace step, got %+v", result.Trace)
	}
}

func TestRunnerRun_ReActToolLoopAllowsManyRounds(t *testing.T) {
	client := &sequenceCompletionClient{
		results: []*types.CompleteResult{
			{Content: "inspect repo", ToolCalls: []openai.ToolCall{newToolCall("tc-1", "read_file", `{"path":"a.go"}`)}},
			{Content: "patch file", ToolCalls: []openai.ToolCall{newToolCall("tc-2", "edit_file", `{"path":"a.go"}`)}},
			{Content: "run tests", ToolCalls: []openai.ToolCall{newToolCall("tc-3", "bash", `{"cmd":"go test ./..."}`)}},
			{Content: "fix tests", ToolCalls: []openai.ToolCall{newToolCall("tc-4", "edit_file", `{"path":"a_test.go"}`)}},
			{Content: "verify again", ToolCalls: []openai.ToolCall{newToolCall("tc-5", "bash", `{"cmd":"go test ./..."}`)}},
			{Content: "implementation complete"},
		},
	}
	executor := &fakeToolExecutor{
		results: []types.ToolCallResult{{ToolCallID: "result", Name: "tool", Content: "ok"}},
	}
	runner := NewRunner(client, WithToolExecutor(executor))

	result, err := runner.Run(context.Background(), AgentRunRequest{
		Agent:    AgentDefinition{Name: "code_generator", Instructions: "Write code."},
		Pipeline: PipelineDefinition{ID: "p1"},
		Stage:    PipelineStage{Name: "implementation", AgentType: "code_generator"},
		PlanItem: StagePlanItem{TaskID: "t1", AgentType: "code_generator"},
	})
	if err != nil {
		t.Fatalf("run agent: %v", err)
	}
	if result.Content != "implementation complete" {
		t.Fatalf("unexpected content: %s", result.Content)
	}
	if len(executor.calls) != 5 {
		t.Fatalf("expected five tool rounds, got %d", len(executor.calls))
	}
	if len(result.Trace) != 6 {
		t.Fatalf("unexpected trace length: %d", len(result.Trace))
	}
	if !result.Trace[5].Final {
		t.Fatalf("expected final trace step, got %+v", result.Trace[5])
	}
	if got := len(client.messages[1]); got != 3 {
		t.Fatalf("expected second completion to include user, assistant, and tool messages, got %d", got)
	}
}

func TestRunnerRun_RequestMaxToolRounds(t *testing.T) {
	client := &sequenceCompletionClient{
		results: []*types.CompleteResult{
			{Content: "first action", ToolCalls: []openai.ToolCall{newToolCall("tc-1", "bash", "{}")}},
			{Content: "second action", ToolCalls: []openai.ToolCall{newToolCall("tc-2", "bash", "{}")}},
		},
	}
	runner := NewRunner(client, WithToolExecutor(&fakeToolExecutor{
		results: []types.ToolCallResult{{ToolCallID: "tc-1", Name: "bash", Content: "ok"}},
	}))

	_, err := runner.Run(context.Background(), AgentRunRequest{
		Agent:         AgentDefinition{Name: "code_generator"},
		Pipeline:      PipelineDefinition{ID: "p1"},
		Stage:         PipelineStage{Name: "implementation", AgentType: "code_generator"},
		PlanItem:      StagePlanItem{TaskID: "t1", AgentType: "code_generator"},
		MaxToolRounds: 1,
	})
	if err == nil {
		t.Fatal("expected max tool round error, got nil")
	}
}

func TestBuildAgentSystemPromptIncludesReActProtocol(t *testing.T) {
	system := buildAgentSystemPrompt(AgentDefinition{
		Name:         "reviewer",
		Instructions: "Review code.",
	})
	if !strings.Contains(system, "ReAct agent") {
		t.Fatalf("expected ReAct protocol in system prompt: %s", system)
	}
}

func TestMarkdownDispatcher(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(AgentDefinition{
		Name:         "reviewer",
		Instructions: "Review code.",
	}); err != nil {
		t.Fatalf("register agent: %v", err)
	}

	dispatcher := NewMarkdownDispatcher(registry, NewRunner(fakeCompletionClient{
		result: &types.CompleteResult{Content: "no findings"},
	}))

	result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
		Pipeline: PipelineDefinition{ID: "p1"},
		Stage:    PipelineStage{Name: "review", AgentType: "reviewer"},
		PlanItem: StagePlanItem{TaskID: "review-task-1", AgentType: "reviewer"},
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if result.Output["content"] != "no findings" {
		t.Fatalf("unexpected output: %+v", result.Output)
	}
	if result.Meta["agent"] != "reviewer" {
		t.Fatalf("unexpected meta: %+v", result.Meta)
	}
}

func newToolCall(id string, name string, arguments string) openai.ToolCall {
	return openai.ToolCall{
		ID:   id,
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      name,
			Arguments: arguments,
		},
	}
}
