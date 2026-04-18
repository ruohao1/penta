# Penta Architecture

## Current repository status

This repository is currently architecture/spec-first. The design below is the intended system shape, not proof that all directories/components already exist in code yet.

---

## Architecture stance (v1)

Build this as a **modular monolith with a typed action engine**.

Do **not** start with:

- microservices
- graph DB first (e.g., Neo4j)
- a “full autonomous agent”

The right v1 shape is:

**deterministic tool pipeline + persistent evidence model + LLM planner + policy/approval gate**

---

## Core architecture

```text
CLI / TUI / API
      |
      v
Campaign Manager
      |
      v
Scheduler / Frontier ---------> Policy Engine <------ Profiles / Mode
      |                               ^
      v                               |
Task Queue                            |
      v                               |
Tool Runtime ---> Raw Artifacts ---> Normalizer ---> Evidence Store / Graph
      |                                                    |
      |                                                    v
      +-----------------------------> Context Builder ---> Planner (LLM)
                                                           |
                                                           v
                                                    Action Proposals
                                                           |
                                                           v
                                                     Reporter / Notes
```

The split that matters:

### Control plane (decides what to do next)

- campaign manager
- scheduler / frontier
- planner
- policy engine
- approval flow
- reporter

### Data plane (collects/transforms evidence)

- tool runtime
- artifact store
- normalizers
- evidence graph/store
- search/retrieval

---

## Top-level components

### Campaign manager

Lifecycle owner for a run. A campaign should carry:

- mode: `ctf`, `bugbounty`, `pentest`
- scope
- profile
- operator notes
- run status
- linked artifacts
- approval policy

Should support:

- create
- resume
- pause
- cancel
- replay
- compare against previous run

Think in terms of **campaign state**, not one-shot scans.

### Scheduler / frontier

Most important non-AI component. Responsibilities:

- track runnable tasks
- enforce dependencies
- dedupe tasks
- retry failed work
- prioritize actions
- checkpoint progress
- resume interrupted runs

Every task should include:

- `action_type`
- normalized inputs
- fingerprint
- parent evidence IDs
- status
- priority
- retry policy
- policy decision

Use a **persistent frontier**, not an in-memory queue only.

Recommended lifecycle:

`pending -> runnable -> running -> succeeded | failed | blocked | skipped`

### Tool runtime

Thin deterministic execution layer around tools. Responsibilities:

- spawn processes
- validate inputs
- enforce timeouts
- cap stdout/stderr
- collect exit metadata
- write raw output to artifact store
- emit typed completion events

Do not let tools write directly into DB models.

Use adapters per tool:

```go
type ToolAdapter interface {
    ActionType() ActionType
    Validate(input json.RawMessage) error
    Execute(ctx context.Context, req ActionRequest) (ToolResult, error)
}
```

`ToolResult` should carry raw artifact refs and execution metadata, not final normalized findings.

### Normalizer

Converts raw output into typed evidence. Keep normalization separate from execution because:

- raw formats change
- different tools can report equivalent facts
- the system needs deduped evidence, not tool blobs

### Evidence store

Primary system of record.

Start with:

- **SQLite** for metadata/state
- **filesystem** object store for raw artifacts

Optional later:

- FTS
- Postgres (when scale/ops demands it)

Prefer relational tables + edge tables over graph DB in v1.

### Planner (LLM boundary)

Planner should **never** execute tools directly and **never** emit shell commands.

Planner outputs only:

- hypotheses
- attack-surface summaries
- ranked next actions (typed)
- rationale
- evidence citations
- uncertainty

### Policy engine

Evaluates proposed actions as:

- allowed
- blocked
- rate-limited
- approval-required

Policy checks include:

- in-scope target?
- action type allowed?
- budget/rate remaining?
- duplicate task?
- host sensitivity?
- manual approval required?

Planner proposes; policy decides.

### Reporter / notes

Continuously builds:

- timeline
- discovered assets
- key evidence
- hypotheses
- operator approvals
- finding drafts
- run summary

---

## Data model

Use **immutable events + materialized tables**.

### Immutable event log (append-only)

Examples:

- `run.created`
- `task.enqueued`
- `task.started`
- `artifact.created`
- `task.completed`
- `evidence.upserted`
- `hypothesis.created`
- `action.proposed`
- `action.blocked`
- `action.approved`
- `finding.promoted`

Useful for replay, auditing, debugging, projection rebuilds, planner behavior comparisons.

### Materialized current-state tables

Suggested core tables:

- `runs`
- `scope_rules`
- `tasks`
- `task_attempts`
- `artifacts`
- `evidence`
- `graph_edges`
- `hypotheses`
- `proposals`
- `findings`
- `llm_calls`

---

## Action model (architecture hinge)

Everything should operate via a typed **action catalog**:

```go
type ActionType string

const (
    ActionPassiveEnum       ActionType = "passive_enum"
    ActionResolveDNS        ActionType = "resolve_dns"
    ActionProbeHTTP         ActionType = "probe_http"
    ActionCrawl             ActionType = "crawl"
    ActionExtractJSRoutes   ActionType = "extract_js_routes"
    ActionContentDiscovery  ActionType = "content_discovery"
    ActionTemplateScan      ActionType = "template_scan"
    ActionCompareBehavior   ActionType = "compare_behavior"
    ActionSummarizeSurface  ActionType = "summarize_surface"
    ActionDraftFinding      ActionType = "draft_finding"
)
```

Each action type must define:

- parameter schema
- validator
- tool adapter or internal handler
- dedupe fingerprint rule
- policy rule
- output contract

---

## Control loop

1. **Seed** campaign and initial tasks (target normalization, passive enum, DNS, HTTP probe)
2. **Execute** via scheduler + bounded worker pools
3. **Normalize** raw outputs to typed evidence/edges
4. **Update context** per host/surface
5. **Plan** only on meaningful state change
6. **Gate** proposed actions through policy
7. **Enqueue** approved actions
8. **Iterate** until frontier exhausted / operator stop / goal reached

Do **not** run planner after every single tool completion.

---

## Context builder

Planner input should be compact structured context, not raw scan logs.

Prioritize evidence by:

- novelty
- centrality
- severity
- freshness
- cross-tool agreement

Snapshot should include scope, current surface, key evidence, recent actions, and profile constraints.

---

## Concurrency and reliability

Use bounded worker pools per action class and enforce domain/host-level rate limits.

Each task should have:

- timeout
- cancellation
- retry limit
- stdout/stderr caps
- deterministic fingerprint
- per-task temp directory

For recovery:

- persist task transitions
- persist attempt metadata
- persist artifact refs before normalization
- keep normalizers idempotent
- replay from event log if projections break

---

## Sandboxing (practical order)

Minimum:

- `exec.CommandContext`
- environment allowlist
- per-task temp dir
- output caps
- timeout
- separate stdout/stderr
- record exact args

Later:

- per-tool container sandbox
- network namespace
- seccomp / ulimit
- filesystem restrictions

Do not overengineer sandboxing before core loop correctness.

---

## Profiles over hardcoded mode branches

Use declarative profiles instead of scattered `if mode == ...` logic.

Profiles should define allowed actions, approval requirements, rate/concurrency limits, and planner constraints.

---

## Intended Go package layout (future)

```text
/internal
  /campaign
  /scheduler
  /policy
  /profiles
  /actions
  /runtime
  /tools
  /normalize
  /evidence
  /graph
  /planner
  /contextbuilder
  /storage
  /artifacts
  /reporting
  /telemetry
  /models
/cmd/penta
/configs/profiles
/schemas
/testdata
```

Keep interfaces narrow.

---

## What not to add yet

- No microservices
- No vector DB
- No direct model-to-shell tool calling

---

## v1 architecture to actually ship

- Storage: SQLite + filesystem artifacts
- Runtime: one binary, bounded workers, persistent DB-backed frontier
- Tools: small fixed adapter set + normalization stage
- AI: single planner, strict JSON output, required evidence citations
- UX: CLI-first, markdown reporting, TUI later

---

## Single biggest design rule

Make the **action catalog** the center of the system, not the LLM.

If actions are central:

- tools implement actions
- scheduler queues actions
- policy evaluates actions
- planner proposes actions
- reports explain actions

Next concrete implementation step: define the task/action schema and first 8–10 action types.
