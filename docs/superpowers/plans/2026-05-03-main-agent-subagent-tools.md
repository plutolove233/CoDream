# Main Agent Sub-Agent Tools Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep Pipeline as the visible execution skeleton while allowing one main ReAct agent to coordinate isolated ReAct sub-agents through tools and channel messages.

**Architecture:** Pipeline stages remain the public lifecycle and frontend status contract. Inside a stage, the main agent can call sub-agent tools backed by isolated `Runner` sessions, and agents can coordinate through named in-memory channels that also emit timeline events for frontend streaming.

**Tech Stack:** Go, existing `internal/llm/agents` ReAct runner, existing `internal/tools` tool registry, `go test`.

---

### Task 1: Collaboration Runtime

**Files:**
- Create: `internal/llm/agents/collaboration.go`
- Test: `internal/llm/agents/collaboration_test.go`

- [x] Define thread-safe channel storage, agent session inputs, run results, and pipeline-friendly events.
- [x] Add tests for isolated context cloning, channel append/read order, and event publication.

### Task 2: Sub-Agent Tool Surface

**Files:**
- Create: `internal/tools/agent_tools.go`
- Test: `internal/tools/tests/agent_tools_test.go`

- [x] Add `run_agent`, `send_agent_message`, and `read_agent_messages` tools.
- [x] Ensure `run_agent` filters tools per sub-agent definition and returns structured JSON.
- [x] Ensure channel tools write/read through the shared collaboration runtime.

### Task 3: Service Exports And Verification

**Files:**
- Modify: `internal/api/service/pipeline_service.go`

- [x] Re-export collaboration runtime types and constructors for API/service wiring.
- [x] Run `gofmt` and targeted tests, then run broader package tests if practical.
