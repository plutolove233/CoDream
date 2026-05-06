# Pipeline Engine API

Base path: `/api/v1`

All endpoints below require `Authorization: Bearer <access_token>`.

## Pipeline CRUD

### Create Pipeline

`POST /pipelines`

Recommended simple request:

```json
{
  "name": "build-sse-endpoint",
  "requirement": "Add an SSE endpoint that streams LLM output.",
  "context_paths": ["task.md", "internal/llm/client.go"],
  "provider": "openai",
  "model": "gpt-4o-mini"
}
```

When `stages` is omitted, the server creates the standard development
pipeline:

- `design` with `solution_designer`, checkpoint after stage
- `implement` with `code_generator`
- `review` with `code_reviewer`, checkpoint after stage

Advanced custom request:

```json
{
  "name": "standard-dev",
  "description": "Requirement to code pipeline",
  "stages": [
    {
      "name": "design",
      "objective": "Create the implementation plan",
      "agent_type": "architect",
      "checkpoints": ["after_stage"],
      "input": {
        "context_paths": ["task.md", "internal/api"]
      }
    },
    {
      "name": "implement",
      "objective": "Apply the code change",
      "agent_type": "executor",
      "depends_on": ["design"]
    }
  ]
}
```

Stages may also bind multiple agents:

```json
{
  "name": "implement",
  "objective": "Build and test the change",
  "agent_type": "executor",
  "agents": [
    {
      "name": "code_writer",
      "role": "implementation",
      "agent_type": "executor",
      "system_prompt": "Write the code change.",
      "context_paths": ["internal/api"],
      "provider": "openai",
      "model": "gpt-4o-mini"
    },
    {
      "name": "test_writer",
      "role": "testing",
      "agent_type": "test-engineer",
      "context_paths": ["internal/api"]
    }
  ]
}
```

### List Pipelines

`GET /pipelines`

### Get Pipeline

`GET /pipelines/{id}`

### Update Pipeline

`PUT /pipelines/{id}`

Body accepts `name`, `description`, and `stages`.

### Delete Pipeline

`DELETE /pipelines/{id}`

## Execution Lifecycle

### Start Execution

`POST /pipelines/{id}/execute`

Recommended simple request:

```json
{
  "requirement": "Add an SSE LLM endpoint",
  "context_paths": ["task.md", "internal/llm/client.go"],
  "provider": "openai",
  "model": "gpt-4o-mini"
}
```

`input` remains available for advanced callers that need arbitrary structured
context.

The response is returned immediately after the server creates:

- `execution_id`: the persisted pipeline execution id
- `session_id`: the conversation/session id used to group this run

The pipeline itself continues in a background goroutine. Clients should use the
status query or SSE endpoint below to monitor progress.

### Get Execution Status

`GET /executions/{id}`

Execution statuses:

- `pending`
- `running`
- `paused`
- `pending_approval`
- `completed`
- `failed`
- `terminated`

### List Execution Stages

`GET /executions/{id}/stages`

### Stream Execution Events

`GET /executions/{id}/events`

This is a Server-Sent Events stream. The first event is always an
`execution.snapshot` payload with the current execution record, so a frontend
can safely connect after the execute response without missing initial state.

Event names:

- `execution.snapshot`
- `execution.status_changed`
- `stage.started`
- `stage.completed`
- `checkpoint.created`
- `error.occurred`

Example event payload:

```json
{
  "type": "execution.status_changed",
  "execution_id": "6ef...",
  "session_id": "2cc...",
  "status": "running",
  "timestamp": "2026-05-05T10:30:00Z"
}
```

### Pause, Resume, or Terminate

`PATCH /executions/{id}`

```json
{
  "action": "pause"
}
```

Allowed actions: `pause`, `resume`, `terminate`.

## Checkpoints

A stage can request human review with:

```json
{
  "checkpoints": ["before_stage", "after_stage"]
}
```

Supported checkpoint positions:

- `before_stage`
- `after_stage`

### List Pending Checkpoints

`GET /checkpoints`

### Approve Checkpoint

`POST /checkpoints/{id}/approve`

```json
{
  "context": {
    "approved_by": "human"
  }
}
```

### Reject Checkpoint

`POST /checkpoints/{id}/reject`

```json
{
  "reason": "The design does not cover rollback.",
  "context": {
    "review_notes": "Add rollback handling before implementation."
  }
}
```

Reject rewinds execution to the previous stage when possible and injects
`rejection_reason` into the execution context for the rerun.

## Runtime Notes

Pipeline definitions, executions, stage results, and checkpoints are persisted
in PostgreSQL tables:

- `pipelines`
- `pipeline_executions`
- `stage_executions`
- `checkpoints`

Agent execution uses the real Markdown Agent Runner:

`PipelineEngineService -> MarkdownDispatcher -> Runner -> internal/llm.Client`

Required LLM environment variables:

- global: `CODEDREAM_LLM_API_KEY`, `OPENAI_API_KEY`, or `API_KEY`
- optional global: `CODEDREAM_LLM_BASE_URL`, `OPENAI_BASE_URL`, or `BASE_URL`
- optional global: `CODEDREAM_LLM_MODEL`, `OPENAI_MODEL`, or `MODEL_EPIP`
- provider-specific override: `CODEDREAM_<PROVIDER>_API_KEY`,
  `CODEDREAM_<PROVIDER>_BASE_URL`, `CODEDREAM_<PROVIDER>_MODEL`

Provider-specific values are also read from Viper at
`llm.providers.<provider>.apiKey`, `llm.providers.<provider>.baseURL`, and
`llm.providers.<provider>.model`. Providers are treated as OpenAI-compatible
chat endpoints by the current runner.
