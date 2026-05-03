package agents

import (
	"context"
	"errors"
	"testing"

	"github.com/plutolove233/co-dream/pkg/types"
)

type recordingAgentRunner struct {
	reqs []AgentRunRequest
}

func (r *recordingAgentRunner) Run(_ context.Context, req AgentRunRequest) (AgentRunResult, error) {
	r.reqs = append(r.reqs, req)
	req.Context["mutated"] = true
	return AgentRunResult{
		Content: "generated code",
		ToolCalls: []types.ToolCallResult{{
			ToolCallID: "tc-1",
			Name:       "bash",
			Content:    "ok",
		}},
	}, nil
}

func TestCollaborationRuntimeRunSubAgentIsolatesContextAndPublishesEvents(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(AgentDefinition{Name: "code_generator"}); err != nil {
		t.Fatalf("register agent: %v", err)
	}
	runner := &recordingAgentRunner{}
	runtime := NewCollaborationRuntime(registry, func(_ AgentDefinition) (AgentRunner, error) {
		return runner, nil
	})

	ctxData := map[string]any{"requirement": "build feature"}
	result, err := runtime.RunSubAgent(context.Background(), SubAgentRunRequest{
		AgentName:    "code_generator",
		TaskID:       "implement-1",
		Instructions: "write code",
		ChannelID:    "stage-1",
		Context:      ctxData,
	})
	if err != nil {
		t.Fatalf("run sub-agent: %v", err)
	}
	if result.Content != "generated code" {
		t.Fatalf("unexpected content: %s", result.Content)
	}
	if ctxData["mutated"] == true {
		t.Fatal("sub-agent mutated caller context")
	}
	if len(runner.reqs) != 1 {
		t.Fatalf("expected one runner request, got %d", len(runner.reqs))
	}
	if runner.reqs[0].PlanItem.AgentType != "code_generator" {
		t.Fatalf("unexpected plan item: %+v", runner.reqs[0].PlanItem)
	}

	events := runtime.EventsAfter("", 0)
	if len(events) != 2 {
		t.Fatalf("expected start and completion events, got %+v", events)
	}
	if events[0].Type != CollaborationEventAgentStarted || events[1].Type != CollaborationEventAgentCompleted {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestCollaborationRuntimeRunSubAgentPublishesFailureEvent(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(AgentDefinition{Name: "code_generator"}); err != nil {
		t.Fatalf("register agent: %v", err)
	}
	runtime := NewCollaborationRuntime(registry, func(_ AgentDefinition) (AgentRunner, error) {
		return nil, errors.New("no runner")
	})

	_, err := runtime.RunSubAgent(context.Background(), SubAgentRunRequest{
		AgentName: "code_generator",
		TaskID:    "implement-1",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	events := runtime.EventsAfter("", 0)
	if len(events) != 2 || events[1].Type != CollaborationEventAgentFailed {
		t.Fatalf("expected failure event, got %+v", events)
	}
}

func TestCollaborationRuntimeChannelMessagesAreOrderedAndFiltered(t *testing.T) {
	runtime := NewCollaborationRuntime(NewRegistry(), nil)

	first, err := runtime.SendMessage("stage-1", "code_generator", "code_reviewer", "ready for review", nil)
	if err != nil {
		t.Fatalf("send first message: %v", err)
	}
	second, err := runtime.SendMessage("stage-1", "code_reviewer", "code_generator", "fix missing test", map[string]any{"severity": "high"})
	if err != nil {
		t.Fatalf("send second message: %v", err)
	}

	messages, err := runtime.ReadMessages("stage-1", "", "", 0)
	if err != nil {
		t.Fatalf("read messages: %v", err)
	}
	if len(messages) != 2 || messages[0].ID != first.ID || messages[1].ID != second.ID {
		t.Fatalf("unexpected message order: %+v", messages)
	}

	afterFirst, err := runtime.ReadMessages("stage-1", first.ID, "code_generator", 10)
	if err != nil {
		t.Fatalf("read filtered messages: %v", err)
	}
	if len(afterFirst) != 1 || afterFirst[0].ID != second.ID {
		t.Fatalf("unexpected filtered messages: %+v", afterFirst)
	}
	if afterFirst[0].Metadata["severity"] != "high" {
		t.Fatalf("metadata was not preserved: %+v", afterFirst[0].Metadata)
	}

	events := runtime.EventsAfter("", 0)
	if len(events) != 2 || events[0].MessageID != first.ID || events[1].MessageID != second.ID {
		t.Fatalf("unexpected channel events: %+v", events)
	}
}
