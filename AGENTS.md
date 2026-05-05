# AGENTS

## Repo Reality
- Early Go CLI for Penta, an AI-assisted offensive-security workflow engine; `README.md` and `docs/architecture.md` are guardrails, not proof planned subsystems exist.
- Real entrypoint: `cmd/penta/main.go`; Cobra commands live in `internal/cli`.
- Implemented user-facing flows are `penta recon <target>` and `penta session ...`; there is no TUI/API service yet.
- No Makefile, CI workflow, linter, formatter config, codegen, or task runner is present; do not invent those workflows.

## Commands
- All tests: `go test ./...`
- Focused package: `go test ./internal/scheduler`
- Single test: `go test ./internal/targets -run TestParseDomain`
- Run recon without touching the default user DB: `PENTA_STORAGE_DB_PATH=/tmp/penta.db go run ./cmd/penta recon example.com`
- Local lab recon needs explicit session scope, e.g. `go run ./cmd/penta session create local-dev --kind lab`, then `go run ./cmd/penta session scope add <session-id> ip 127.0.0.1 --include`, then `go run ./cmd/penta recon --session <session-id> localhost:8080`.
- Do not commit unless `go test ./...` passes.

## Config, Storage, Artifacts
- Config uses Viper; optional `penta.yaml` is read from `.`, `.penta`, or the XDG config dir.
- Env vars use `PENTA_` with `_` for nested keys, e.g. `PENTA_STORAGE_DB_PATH`.
- Default DB path is under XDG state, usually `~/.local/state/penta/penta.db`; tests should use temp DBs.
- SQLite schema is embedded in `internal/storage/sqlite/schema.go`; migrations live in `internal/storage/sqlite/migrations.go` and current `PRAGMA user_version` is `2`.
- HTML response bodies are stored as capped filesystem artifacts beside the DB (`artifacts/`) and referenced by SQLite artifact rows; raw bodies should not be printed in normal output.

## Action Model
- Shared action types/specs live in `internal/actions`; runnable packages live under `internal/actions/<action_name>` and are registered in `internal/execute/registry.go` with `ActionSpec + Handler`.
- Current runnable actions: `seed_target`, `probe_http`, `resolve_dns`, `http_request`, and `crawl`.
- Actions execute one deterministic operation and emit evidence; they must not enqueue follow-up tasks directly.
- `probe_http` discovers HTTP services and emits `service`; `http_request` performs bounded `GET`/`HEAD` requests and emits `http_response`; `crawl` reads stored HTML body artifacts and emits `crawl` evidence.
- `fetch_root` was replaced by `http_request`; do not reintroduce root-fetch-specific behavior unless there is a strong reason.

## Execution Flow
- `penta recon` requests `probe_http`; executor runs SQLite tasks, persists evidence, emits events, then asks `internal/scheduler` for follow-on candidates from task-linked evidence.
- Frontier admission in `internal/execute/frontier.go` is the gate for policy, session scope, crawl depth/budget, dedupe, and `candidate.blocked` events; keep those checks centralized there.
- Scheduler derives current chains such as `target(domain) -> resolve_dns + probe_http`, `service(http|https) -> http_request GET /`, `http_response(html artifact) -> crawl`, and `crawl -> http_request` for discovered URLs.
- Duplicate suppression is exact `(run_id, action_type, input_json)`; changing action input JSON changes dedupe behavior.

## Scope, Policy, and Network Safety
- Sessions are explicit for v1: `penta recon --session <session-id> <target>`; there is no persistent active session.
- Session scope is checked before run creation for the initial target and by frontier for derived candidates; exclude rules win over include rules.
- IP/CIDR scope can authorize service/URL hosts, including `localhost` as loopback for local lab work.
- `http_request` uses a guarded dialer: loopback/private/link-local/unspecified/multicast destinations are blocked unless the run has explicit matching session `ip`/`cidr` scope.
- `http_request` does not follow redirects; if that changes, apply the same network guard to every redirect target.
- Crawl defaults are intentionally small: per-page extraction cap is `50`, max crawl depth is `1`, and crawl-derived URL budget is `100` per run.

## Output and Reports
- Human output is event-driven in `internal/cli/output.go`; action handlers/scheduler/executor should stay silent.
- Stdout is for progress/reports; stderr is for errors/warnings/log diagnostics.
- `penta recon -o <file>` writes Markdown and fails if the file exists; there is no `--force` yet.
- `--redact-report` redacts final terminal/Markdown report strings only; stored evidence/events/artifacts remain unredacted except bounded header redaction done by `http_request`.
- Normal terminal reports are intentionally compact; Markdown can retain detailed evidence such as hashes and artifact IDs.

## Architecture Constraints
- Keep v1 a single-process modular monolith; defer microservices, graph DB first, or autonomous command execution until there is a concrete need.
- The LLM is planner-only: it may summarize evidence and propose typed actions, but must not execute tools or emit shell commands.
- Prefer SQLite for metadata/state and filesystem artifacts for raw data until the repo adds a concrete reason to change.
- Treat docs as direction, but verify current code before adding planned subsystems.
- Keep diffs focused; do not mix broad refactors with feature work unless explicitly requested.
