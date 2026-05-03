package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAgentMarkdown(t *testing.T) {
	agent, err := ParseAgentMarkdown(`---
name: code_generator
description: Write code
model: gpt-test
tools:
  - file_handler
  - bash
---

Follow the plan.
`, "AGENT.md")
	if err != nil {
		t.Fatalf("parse agent failed: %v", err)
	}
	if agent.Name != "code_generator" {
		t.Fatalf("unexpected name: %s", agent.Name)
	}
	if agent.Model != "gpt-test" {
		t.Fatalf("unexpected model: %s", agent.Model)
	}
	if len(agent.Tools) != 2 {
		t.Fatalf("unexpected tools: %+v", agent.Tools)
	}
	if agent.Instructions != "Follow the plan." {
		t.Fatalf("unexpected instructions: %q", agent.Instructions)
	}
}

func TestRegistryLoadFromDir(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "reviewer")
	if err := os.Mkdir(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	content := `---
name: reviewer
description: Review code
---

Find concrete defects.
`
	if err := os.WriteFile(filepath.Join(agentDir, "AGENT.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write agent file: %v", err)
	}

	registry := NewRegistry()
	if err := registry.LoadFromDir(dir); err != nil {
		t.Fatalf("load agents: %v", err)
	}
	agent, ok := registry.Get("reviewer")
	if !ok {
		t.Fatal("expected reviewer agent to be registered")
	}
	if agent.Description != "Review code" {
		t.Fatalf("unexpected description: %s", agent.Description)
	}
}
