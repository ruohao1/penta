# Penta

**AI-assisted offensive security workflow engine for CTF, bug bounty, and authorized pentesting**

Penta is intended as a **modular monolith with a typed action engine**:

- deterministic tool pipeline
- persistent evidence model
- LLM planner (planning only)
- policy / approval gate

## Repository status

This repo is still architecture-led, but it now includes a small Cobra CLI and config bootstrap. The broader system design is documented below, while most planned subsystems are still not implemented yet.

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
