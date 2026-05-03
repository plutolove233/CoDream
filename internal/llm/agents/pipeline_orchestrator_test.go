package agents

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestPipelineOrchestratorRun_Success(t *testing.T) {
	dispatcher := NewMapDispatcher()
	dispatcher.Register("planner", func(_ context.Context, req DispatchRequest) (DispatchResult, error) {
		return DispatchResult{Output: map[string]any{"plan": req.PlanItem.Description}}, nil
	})
	dispatcher.Register("coder", func(_ context.Context, req DispatchRequest) (DispatchResult, error) {
		if req.Context["plan"] != "build plan" {
			t.Fatalf("expected context plan propagated, got %v", req.Context["plan"])
		}
		return DispatchResult{Output: map[string]any{"code": "done"}}, nil
	})

	o := NewPipelineOrchestrator(nil, dispatcher)
	def := PipelineDefinition{
		ID:   "p1",
		Name: "demo",
		Stages: []PipelineStage{
			{Name: "plan", Objective: "build plan", AgentType: "planner"},
			{Name: "implement", AgentType: "coder", DependsOn: []string{"plan"}},
		},
	}

	got, err := o.Run(context.Background(), def, map[string]any{"input": "x"})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if got.PipelineID != "p1" {
		t.Fatalf("unexpected pipeline id: %s", got.PipelineID)
	}
	if got.Status != RunStatusCompleted {
		t.Fatalf("unexpected status: %s", got.Status)
	}
	if len(got.Stages) != 2 {
		t.Fatalf("unexpected stage count: %d", len(got.Stages))
	}
	if got.Stages[1].StageName != "implement" {
		t.Fatalf("unexpected second stage: %s", got.Stages[1].StageName)
	}
	if got.Stages[1].Items[0].Output["code"] != "done" {
		t.Fatalf("unexpected output: %v", got.Stages[1].Items[0].Output)
	}
}

func TestPipelineOrchestratorRun_PendingApproval(t *testing.T) {
	dispatcher := NewMapDispatcher()
	dispatcher.Register("planner", func(_ context.Context, _ DispatchRequest) (DispatchResult, error) {
		t.Fatal("dispatcher should not run before pending approval is resolved")
		return DispatchResult{}, nil
	})

	o := NewPipelineOrchestrator(nil, dispatcher).WithApprovalGate(ApprovalGateFunc(
		func(_ context.Context, req ApprovalRequest) (ApprovalDecision, error) {
			if req.Point != ApprovalAfterPlan {
				return ApprovalDecision{Status: ApprovalApproved}, nil
			}
			return ApprovalDecision{Status: ApprovalPending, Reason: "wait for user"}, nil
		},
	))

	got, err := o.Run(context.Background(), PipelineDefinition{
		ID: "p5",
		Stages: []PipelineStage{
			{Name: "plan", Objective: "build plan", AgentType: "planner", Checkpoint: string(ApprovalAfterPlan)},
		},
	}, nil)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if got.Status != RunStatusPendingApproval {
		t.Fatalf("unexpected status: %s", got.Status)
	}
	if got.PendingApproval == nil || got.PendingApproval.Point != ApprovalAfterPlan {
		t.Fatalf("unexpected pending approval: %+v", got.PendingApproval)
	}
}

func TestPipelineOrchestratorRun_DependencyValidation(t *testing.T) {
	dispatcher := NewMapDispatcher()
	o := NewPipelineOrchestrator(nil, dispatcher)
	def := PipelineDefinition{
		ID: "p2",
		Stages: []PipelineStage{
			{Name: "implement", AgentType: "coder", DependsOn: []string{"plan"}},
		},
	}

	_, err := o.Run(context.Background(), def, nil)
	if !errors.Is(err, ErrStageDependsOnNotFound) {
		t.Fatalf("expected dependency not found, got: %v", err)
	}
}

func TestPipelineOrchestratorRun_OrdersStagesByDependencies(t *testing.T) {
	var order []string
	dispatcher := NewMapDispatcher()
	dispatcher.Register("agent", func(_ context.Context, req DispatchRequest) (DispatchResult, error) {
		order = append(order, req.Stage.Name)
		return DispatchResult{}, nil
	})

	o := NewPipelineOrchestrator(nil, dispatcher)
	_, err := o.Run(context.Background(), PipelineDefinition{
		ID: "p6",
		Stages: []PipelineStage{
			{Name: "implement", AgentType: "agent", DependsOn: []string{"plan"}},
			{Name: "plan", AgentType: "agent"},
		},
	}, nil)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	want := []string{"plan", "implement"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("unexpected order: got %+v want %+v", order, want)
	}
}

func TestPipelineOrchestratorRun_DependencyCycle(t *testing.T) {
	dispatcher := NewMapDispatcher()
	dispatcher.Register("agent", func(_ context.Context, _ DispatchRequest) (DispatchResult, error) {
		return DispatchResult{}, nil
	})

	o := NewPipelineOrchestrator(nil, dispatcher)
	_, err := o.Run(context.Background(), PipelineDefinition{
		ID: "p7",
		Stages: []PipelineStage{
			{Name: "a", AgentType: "agent", DependsOn: []string{"b"}},
			{Name: "b", AgentType: "agent", DependsOn: []string{"a"}},
		},
	}, nil)
	if !errors.Is(err, ErrStageDependencyCycle) {
		t.Fatalf("expected dependency cycle, got: %v", err)
	}
}

func TestPipelineOrchestratorRun_AgentFailure(t *testing.T) {
	dispatcher := NewMapDispatcher()
	dispatcher.Register("broken", func(_ context.Context, _ DispatchRequest) (DispatchResult, error) {
		return DispatchResult{}, errors.New("agent crashed")
	})

	o := NewPipelineOrchestrator(nil, dispatcher)
	def := PipelineDefinition{
		ID: "p3",
		Stages: []PipelineStage{
			{Name: "plan", AgentType: "broken"},
		},
	}

	_, err := o.Run(context.Background(), def, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDefaultStagePlanner(t *testing.T) {
	planner := DefaultStagePlanner{}
	stage := PipelineStage{Name: "design", Objective: "draft architecture", AgentType: "architect"}
	plan, err := planner.PlanStage(context.Background(), PlanRequest{
		Pipeline: PipelineDefinition{ID: "p4"},
		Stage:    stage,
		Context:  map[string]any{"k": "v"},
	})
	if err != nil {
		t.Fatalf("plan failed: %v", err)
	}

	if plan.StageName != "design" {
		t.Fatalf("unexpected stage name: %s", plan.StageName)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("unexpected item size: %d", len(plan.Items))
	}
	item := plan.Items[0]
	if item.AgentType != "architect" || item.Description != "draft architecture" {
		t.Fatalf("unexpected plan item: %+v", item)
	}
	if !reflect.DeepEqual(item.Input, stage.Input) {
		t.Fatalf("unexpected input pass-through: %+v", item.Input)
	}
}

type ApprovalGateFunc func(ctx context.Context, req ApprovalRequest) (ApprovalDecision, error)

func (f ApprovalGateFunc) RequestApproval(ctx context.Context, req ApprovalRequest) (ApprovalDecision, error) {
	return f(ctx, req)
}
