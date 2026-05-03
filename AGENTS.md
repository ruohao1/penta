# AGENTS

## Repo Reality
- This is an early Go CLI for Penta, an AI-assisted offensive security workflow engine; `README.md` and `docs/architecture.md` are architecture guardrails, not proof that planned subsystems exist.
- Current entrypoint is `cmd/penta/main.go`; Cobra wiring lives in `internal/cli`, with `penta recon <target>` as the implemented command.
- Current core packages are `internal/actions`, `internal/execute`, `internal/scheduler`, `internal/events`, `internal/storage/sqlite`, `internal/targets`, and `internal/config`.
- Use dedicated branches for new work; keep slices small and compiling before committing.

## Commands
- Run all tests: `go test ./...`
- Run a focused package: `go test ./internal/scheduler`
- Run one test: `go test ./internal/targets -run TestParseDomain`
- Run the CLI without writing to the default user state DB: `PENTA_STORAGE_DB_PATH=/tmp/penta.db go run ./cmd/penta recon example.com`
- No Makefile, CI, lint, formatter, migration, or codegen config is currently present; do not invent those workflows.

## Config And Storage
- Config uses Viper: optional `penta.yaml` is read from `.`, `.penta`, or the XDG config dir.
- Env vars use the `PENTA_` prefix with `_` for nested keys, e.g. `PENTA_STORAGE_DB_PATH`.
- Default DB path is under XDG state, usually `~/.local/state/penta/penta.db`; `config.Load()` creates config/state/cache/data dirs.
- SQLite schema is embedded in `internal/storage/sqlite/schema.go`; there is no migration framework yet.
- Tests use temp SQLite DBs; follow that pattern instead of touching the default runtime DB.

## Architecture Rules
- Build v1 as a single-process modular monolith; defer microservices until there is a concrete scaling, isolation, deployment, or ownership need.
- Keep typed actions central; planner, scheduler, policy, executor, and reporting should work around action specs and evidence, not shell commands.
- The LLM is planner-only: it may summarize evidence, form hypotheses, and propose typed actions, but must not execute tools or emit shell commands.
- Policy should decide allow, block, rate-limit, or approval-required; do not scatter mode-specific `if mode == ...` behavior.
- Prefer SQLite for metadata/state and filesystem storage for raw artifacts until the repo adds a concrete reason to change.

## Action Model
- Shared action types/specs live in `internal/actions`; concrete action packages live under `internal/actions/<action_name>`.
- Runnable actions are registered in `internal/execute/registry.go` with `ActionSpec + Handler`.
- Current runnable handlers are `seed_target` and `probe_http`; `resolve_dns`, `fetch_root`, and `crawl` currently only have contracts/stubs.
- `probe_http` means HTTP service discovery and produces `service` evidence; `fetch_root` should be the action that produces `http_response`.
- Actions execute one deterministic operation and produce evidence; they should not enqueue their own follow-ups.

## Execution Flow
- Executor runs tasks from SQLite, marks status, emits events, and persists evidence through action handlers.
- Evidence rows include `task_id`; use task-linked evidence for derivation instead of reprocessing whole-run evidence.
- `internal/scheduler` derives candidate tasks from evidence, e.g. `target(domain|ip|url) -> probe_http`.
- Executor currently enqueues scheduler-derived candidates and skips exact duplicate `(run_id, action_type, input_json)` tasks.
- Future policy gates should sit between scheduler candidates and task enqueueing.

## Implementation Hygiene
- Treat docs as guardrails, but verify current code before adding planned subsystems.
- Keep diffs focused; do not mix broad refactors with feature work unless explicitly requested.
- Do not commit unless `go test ./...` passes.
- If branch intent is unclear, ask before changing architecture or package boundaries.
