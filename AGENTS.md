# Repository Guidelines

## Project Structure & Module Organization

CoDream is a Go service module (`github.com/plutolove233/co-dream`). Entry points live under `cmd/`: `cmd/server` runs the Gin API, `cmd/key` generates RSA keys, and `cmd/gen` regenerates GORM query/model code. Private application code is under `internal/`: API handlers, routers, middleware, and services are in `internal/api`; database clients are in `internal/database`; DAOs are in `internal/dao`; generated GORM files are in `internal/dal`; shared helpers are in `internal/utils`. Configuration is in `configs/config.yaml`, database bootstrap SQL is in `migrations/init.sql`, and design/setup notes are in `docs/`.

## Build, Test, and Development Commands

- `docker compose up -d`: start local PostgreSQL and Redis using `docker-compose.yml`.
- `go run ./cmd/server`: run the API locally; default bind is `0.0.0.0:8080` from `configs/config.yaml`.
- `go run ./cmd/key`: generate RSA key files under `configs/rsa/`.
- `go run ./cmd/gen`: regenerate GORM DAL code from the configured database schema.
- `go test ./...`: run all Go tests.
- `go build ./cmd/server`: build the server binary.

## Coding Style & Naming Conventions

Use standard Go formatting (`gofmt`) before committing. Keep packages lowercase and focused; prefer short singular names such as `router`, `service`, or `token`. Exported identifiers use `PascalCase`; unexported identifiers use `camelCase`; initialisms stay capitalized (`userID`, `apiURL`). Keep `cmd/*/main.go` thin and place reusable behavior in `internal/`. Treat files ending in `.gen.go` as generated output and avoid manual edits unless regenerating is impossible.

## Testing Guidelines

Place tests beside the code they cover using Go’s `*_test.go` convention. Name tests by behavior, for example `TestGenerateTokenRejectsExpiredInput`. Run `go test ./...` before submitting changes. Add table-driven tests for service, DAO, and token logic when adding branches or error handling. Integration tests that need PostgreSQL or Redis should document required Docker services.

## Commit & Pull Request Guidelines

Recent history uses concise Conventional Commit-style subjects, commonly `feat: ...`. Prefer `feat:`, `fix:`, `docs:`, `test:`, or `refactor:` prefixes with an imperative summary. Pull requests should include the problem, the approach, verification performed, and any config or migration impact. Include screenshots or request/response examples for API behavior changes when useful.

## Security & Configuration Tips

Do not commit real secrets. Local defaults in `configs/config.yaml` and `.env` are for development only; override credentials in the environment for shared or deployed systems. Review changes to token, RSA, session, Redis, and database configuration carefully.
