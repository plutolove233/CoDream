# Repository Guidelines

## Project Structure & Architecture

CoDream is a Go service module (`github.com/plutolove233/co-dream`) with a single HTTP server entrypoint and several supporting command binaries.

- `cmd/server`: boots the Gin API server, loads Viper config, initializes PostgreSQL, and conditionally initializes Redis.
- `cmd/key`: generates RSA key material used by auth-related flows.
- `cmd/gen`: regenerates GORM DAL query/models into `internal/dal`.

Application code under `internal/` follows the current layered structure:

- `internal/api/router`: route registration only. Keep HTTP/WS URL wiring here.
- `internal/api/handler`: transport-facing request handlers. Parse input, call services, and shape responses. Do not place persistence logic here.
- `internal/api/service`: business logic and orchestration. This is the main place for workflow decisions.
- `internal/dao`: persistence access for app-owned queries and mutations.
- `internal/dal`: generated GORM models and query helpers. Treat `*.gen.go` as generated artifacts.
- `internal/database`: PostgreSQL and Redis initialization plus connection access.
- `internal/llm`: model client integration and tool-call bridging.
- `internal/tools`: tool definitions, registry, and tool execution plumbing.
- `internal/skill`: skill loading, parsing, and execution support.
- `internal/globals`: shared response helpers, status codes, and common constants/errors.
- `internal/utils`: focused utilities such as RSA, token, captcha, and email helpers.
- `internal/setting`: configuration bootstrap.

Configuration and assets:

- `configs/config.yaml`: runtime config.
- `configs/agents/`: agent config payloads.
- `configs/skills/`: bundled skill content.
- `migrations/init.sql`: database bootstrap SQL.
- `docs/`: product, design, and implementation notes.

## Current Task Brief

The active product task is to build the pipeline-engine platform described in `task.md`. Keep implementation and future guidance aligned with these requirements:

- Build a configurable Pipeline engine with stage definitions, ordering, dependency management, inter-stage data flow, and lifecycle controls for start, pause, resume, and terminate.
- Model each stage as one or more AI Agents with explicit roles, system prompts, and input/output contracts.
- Ensure Agents can consume codebase context, at minimum via directory or file-path based context input.
- Support at least two configurable LLM providers and allow switching providers at runtime.
- Include at least two human-in-the-loop checkpoints with `Approve` and `Reject` outcomes, where `Reject` sends work back to the previous stage with the rejection reason.
- Expose core operations through RESTful APIs, including Pipeline CRUD, execution trigger, status query, and checkpoint actions.
- Keep API documentation complete enough to serve as the public contract for the workflow.
- Produce at least one end-to-end demo run that takes a requirement as input and results in a runnable code change, ideally against this repository itself.
- Treat multi-agent collaboration, automatic regression/retry, observability, codebase indexing, pipeline templates, and Git integration as good-to-have enhancements rather than must-haves.
- If front-end work is pursued, keep the experience aligned with the task brief: a SPA dashboard, floating chat and selection affordances, source-linked edits, hot reload preview, and optional MR summary generation.

## Layering Rules

Follow the existing request path unless the task explicitly requires a new pattern:

`router -> handler -> service -> dao/dal -> database`

- Routers register endpoints and middleware only.
- Handlers own request binding, auth/context extraction, and HTTP/WebSocket response translation.
- Services own business rules, cross-component orchestration, and error semantics.
- DAOs own storage access and should use `context.Context`.
- Generated DAL code is a dependency, not a customization point.

For agent capabilities:

- `internal/llm` owns provider client behavior and tool-call streaming.
- `internal/tools` owns tool registration and execution contracts.
- `internal/skill` owns loading/parsing/executing skill definitions from disk.

Keep these boundaries explicit. Do not move business logic into routers, handlers, generated files, or generic utils just to make a change quicker.

## Generated Code Policy

- Never hand-edit `internal/dal/models/*.gen.go` or `internal/dal/query/*.gen.go` unless regeneration is impossible and the user explicitly accepts the tradeoff.
- When schema-driven DAL output must change, prefer updating the source schema/config and rerunning `go run ./cmd/gen`.
- If generated code is changed manually as an emergency fix, call it out clearly in the final report as a risk.

## Development Commands

- `docker compose up -d`: start local PostgreSQL and Redis.
- `go run ./cmd/server`: run the API server locally.
- `go run ./cmd/key`: generate RSA keys.
- `go run ./cmd/gen`: regenerate GORM DAL code.
- `go test ./...`: run the full test suite.
- `go build ./cmd/server`: build the server binary.

Prefer targeted test runs while iterating, then `go test ./...` before concluding substantial code changes.

## Go Conventions

- Use `gofmt` on changed Go files.
- Keep `cmd/*/main.go` thin; put reusable logic under `internal/`.
- Package names stay short, lowercase, and purpose-specific.
- Exported identifiers use `PascalCase`; unexported identifiers use `camelCase`; preserve Go initialisms such as `userID`, `apiURL`, and `HTTP`.
- Thread `context.Context` through handlers, services, DAOs, and I/O-heavy utilities.
- Return wrapped errors with context; avoid log-and-return duplication in lower layers.

## Testing Expectations

- Place tests beside the code they cover using `*_test.go`.
- Prefer table-driven tests for service, DAO, token, and tool behaviors.
- When changing auth, session, captcha, Redis, token, or DAO behavior, add or update regression coverage unless equivalent protection already exists.
- Integration tests that require PostgreSQL or Redis should document that dependency clearly.

## Change Strategy

- Prefer small, reviewable diffs that preserve the existing layering.
- Reuse existing service, DAO, tool, and utility patterns before adding new abstractions.
- Prefer deletion or simplification over introducing new packages or helpers.
- No new dependencies without explicit user approval.
- For refactor or cleanup work, write a cleanup plan first and lock behavior with tests before broad edits.

## Verification

Before claiming completion, run the lightest verification that proves the change:

- docs-only changes: proofread and check referenced paths/commands against the repo.
- code changes: run relevant tests, then broader validation as needed.
- structural or cross-layer changes: run `go test ./...` unless blocked.

Final reports should include:

- changed files
- architecture or simplification decisions
- verification performed
- remaining risks or unverified areas

## Security & Configuration

- Do not commit real secrets, tokens, or private keys.
- Treat `configs/config.yaml`, `.env`, RSA paths, session config, Redis config, and token settings as security-sensitive.
- Preserve current auth/session behavior unless the task explicitly changes it.
- Review changes touching login, refresh token, captcha, email, RSA, Redis, or database initialization with extra care.
