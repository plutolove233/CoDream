package service

import (
	"context"
	"errors"
	"testing"
)

func TestBuildPipelineDefinitionUsesSimpleRequestDefaults(t *testing.T) {
	def := buildPipelineDefinition(
		"",
		"add pipeline APIs",
		[]string{"task.md", "internal/api"},
		"openai",
		"gpt-4o-mini",
		"",
		nil,
	)

	if def.Name != "development-pipeline" {
		t.Fatalf("unexpected default name: %s", def.Name)
	}
	if len(def.Stages) != 3 {
		t.Fatalf("expected standard 3-stage pipeline, got %d", len(def.Stages))
	}
	if def.Stages[0].AgentType != "solution_designer" {
		t.Fatalf("unexpected first agent: %s", def.Stages[0].AgentType)
	}
	if def.Stages[0].Checkpoints[0] != string(ApprovalAfterStage) {
		t.Fatalf("expected design checkpoint, got %+v", def.Stages[0].Checkpoints)
	}
	if def.Stages[2].Checkpoints[0] != string(ApprovalAfterStage) {
		t.Fatalf("expected review checkpoint, got %+v", def.Stages[2].Checkpoints)
	}
	if got := def.Stages[0].Input["requirement"]; got != "add pipeline APIs" {
		t.Fatalf("expected requirement in stage input, got %v", got)
	}
}

func TestBuildPipelineDefinitionPreservesCustomStages(t *testing.T) {
	stages := []PipelineStage{
		{Name: "custom", AgentType: "solution_designer"},
	}
	def := buildPipelineDefinition("custom-pipeline", "ignored", nil, "", "", "", stages)

	if def.Name != "custom-pipeline" {
		t.Fatalf("unexpected name: %s", def.Name)
	}
	if len(def.Stages) != 1 || def.Stages[0].Name != "custom" {
		t.Fatalf("expected custom stages to be preserved, got %+v", def.Stages)
	}
}

func TestPipelineEngineRequiresPostgreSQL(t *testing.T) {
	_, err := NewPipelineEngineService().CreatePipeline(context.Background(), "user-1", PipelineCreateRequest{
		Requirement: "x",
	})
	if !errors.Is(err, ErrDatabaseUnavailable) {
		t.Fatalf("expected database unavailable, got %v", err)
	}
}
