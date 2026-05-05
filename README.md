# Penta

**AI-assisted offensive security workflow engine for CTF, bug bounty, and authorized pentesting**

Penta is intended as a **modular monolith with a typed action engine**:

- deterministic tool pipeline
- persistent evidence model
- LLM planner (planning only)
- policy / approval gate

## Repository status

Penta is an early Go CLI. The implemented user-facing flows are scoped recon,
session management, run listing, and list-first inspection of stored evidence and
artifact metadata. The broader system design is documented below; planned TUI/API
surfaces are not implemented yet.

## Read this first

- Detailed architecture: [`docs/architecture.md`](docs/architecture.md)
- Agent operating constraints: [`AGENTS.md`](AGENTS.md)

## v1 guardrails

- No microservices
- No graph DB as initial storage layer
- No “full autonomous agent” that executes commands directly
- Keep the LLM as planner-only over a typed action catalog

## Near-term implementation priority

Define the task/action schema and first action catalog before adding broad subsystem surface area.

## Build and test

```bash
go install ./cmd/penta
go test ./...
```

## Quick start

Create an explicit lab session, authorize loopback scope, then run recon:

```bash
penta session create local-dev --kind lab
penta session scope add <session-id> ip 127.0.0.1 --include
penta recon --session <session-id> localhost:8080
```

Penta uses its default XDG state database unless configured otherwise.

## Inspecting results

Use list-first commands to find short, human-friendly selectors before showing
details:

```bash
penta runs list
penta evidence list
penta evidence show 5
penta evidence show http_response:/docs
penta artifacts list
penta artifacts show 1
```

Evidence and artifact `show` commands accept list indexes, stable IDs, and
supported semantic selectors. Artifact inspection is metadata-only and does not
print stored response bodies.

## Shell completion

Generate completions with Cobra's built-in completion command:

```bash
source <(penta completion zsh)
penta completion bash
penta completion fish
```

Completions include run selectors such as `latest`, numbered evidence/artifact
selectors, semantic selectors such as `http_response:/docs`, and exact IDs when
typing an ID prefix.

## Safety notes

- Sessions are explicit for v1: pass `--session <session-id>` to `penta recon`.
- Local, private, and loopback recon targets require explicit matching session
  scope.
- Normal terminal reports are compact; detailed evidence and artifact metadata
  can be inspected on demand.
