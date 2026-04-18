# AGENTS

## Scope
- This repo is still early, but it now has a small Cobra CLI plus config wiring. Primary docs are `README.md` (overview) and `docs/architecture.md` (detailed target design).
- Treat architecture docs as intended design, not proof that all planned directories or subsystems are implemented yet.

## Current repo reality
- `go test ./...` currently covers only the bootstrap CLI/config packages; most planned subsystems are still not implemented.
- Do not write instructions that assume build, lint, CI, migrations, or codegen flows exist until the repo actually adds them.

## Architecture guardrails
- Build this as a single-process modular monolith, not microservices.
- Keep the main boundary explicit:
- Control plane: campaign manager, scheduler/frontier, planner, policy/approval, reporter.
- Data plane: tool runtime, raw artifacts, normalizers, evidence store/graph, retrieval.
- The typed action catalog is the center of the system. Design new modules around actions, not around prompts or tools.

## Planner and policy rules
- The LLM is a planner only. It may summarize evidence, create hypotheses, and propose typed next actions.
- The LLM must not execute tools directly, emit shell commands, or bypass the action catalog.
- Policy decides allow, block, rate-limit, or approval-required. Do not hardcode mode-specific behavior throughout the codebase.

## v1 defaults
- Prefer SQLite for metadata/state and filesystem storage for raw artifacts.
- Use a persistent frontier/task store, not an in-memory queue only.
- Keep tool execution thin and deterministic; normalization is a separate stage.
- Require structured planner output with evidence citations.

## Avoid early overengineering
- Do not start with microservices, a graph database, a vector database, or a "full autonomous agent".
- Do not let raw tool output become the DB model; normalize into typed evidence first.
- Do not make the planner the center of the architecture; the action/task model should drive scheduler, policy, runtime, and reporting.

## Implementation bias
- The next foundational work should define the task/action schema and the first small action catalog before adding broad subsystem surface area.
- When code starts landing, verify new package boundaries against the control-plane/data-plane split instead of mirroring the aspirational `README.md` tree blindly.

## Git hygiene for agent changes
- Keep changes small and reviewable (prefer focused diffs over broad rewrites).
- Before finishing, verify the modification actually matches the current branch purpose/scope.
- If branch intent is unclear, ask for clarification before making broad or unrelated edits.
- Do not mix refactors with feature work on the same branch unless explicitly requested.
