# Penta

**AI-assisted offensive security workflow engine for CTFs, bug bounty, and authorized pentesting**

---

## Overview

Penta is a modular, stateful, and policy-driven reconnaissance and analysis platform designed to augment offensive security workflows.

It combines:

* deterministic tooling for data collection
* structured evidence modeling
* a persistent execution engine
* an AI reasoning layer for hypothesis generation and planning

Penta is not an autonomous exploitation system. It is an **operator-centric platform** that accelerates attack-surface discovery, reasoning, and decision-making while maintaining control, auditability, and scope enforcement.

---

## Core Philosophy

Penta is built around a few key principles:

### 1. Deterministic execution first

All data collection is performed using well-defined tools and reproducible actions.

### 2. AI as a reasoning layer, not an executor

The LLM does not run commands. It:

* analyzes structured evidence
* proposes hypotheses
* suggests next actions

### 3. Everything is an action

All work is modeled as typed actions:

* scheduled
* deduplicated
* persisted
* replayable

### 4. Evidence-driven system

Raw tool output is never trusted directly. It is:

* normalized
* deduplicated
* linked in an evidence graph

### 5. Persistent and recoverable

Every run is:

* resumable
* auditable
* reproducible

### 6. Scoped and policy-controlled

Execution is constrained by:

* scope definitions
* mode-specific policies
* approval gates

---

## Key Features

* Modular reconnaissance pipeline
* Persistent task scheduler (frontier)
* Structured evidence graph
* AI-assisted hypothesis generation
* Action planning with policy enforcement
* Multi-mode operation (CTF, bug bounty, pentest)
* JavaScript intelligence extraction
* Campaign-based workflow
* Markdown reporting and note tracking
* Fully local-first architecture

---

## Architecture

### High-level design

```
CLI / TUI
    |
    v
Campaign Manager
    |
    v
Scheduler / Frontier -------> Policy Engine
    |                               ^
    v                               |
Task Queue                          |
    v                               |
Tool Runtime ---> Artifacts ---> Normalizer ---> Evidence Store / Graph
    |                                                    |
    |                                                    v
    +-----------------------------> Context Builder ---> Planner (LLM)
                                                           |
                                                           v
                                                    Action Proposals
                                                           |
                                                           v
                                                      Reporter
```

---

## Core Components

### Campaign Manager

Represents a full engagement:

* targets
* scope rules
* execution mode
* profiles
* results
* notes

Supports:

* create / resume / pause / cancel
* multi-target workflows
* campaign comparison

---

### Scheduler / Frontier

Responsible for execution flow:

* task queuing
* deduplication
* retries
* prioritization
* dependency management
* persistence

Task lifecycle:

```
pending → runnable → running → succeeded | failed | blocked
```

---

### Tool Runtime

Executes deterministic tools:

* subfinder
* dns resolution
* http probing
* crawling
* content discovery
* template scanning

Each tool:

* runs in isolation
* produces artifacts
* emits execution metadata

---

### Normalizer

Transforms raw tool outputs into structured evidence.

Example:

* raw HTTP response → service, route, fingerprint

Ensures:

* consistency across tools
* deduplication
* graph integration

---

### Evidence Store

Stores all structured data:

Entities:

* assets (domains, IPs)
* services
* endpoints
* parameters
* findings
* hypotheses
* artifacts

Relationships:

* discovered_from
* resolves_to
* references
* exposes
* supports

Implemented with:

* SQLite (metadata)
* filesystem (artifacts)

---

### Evidence Graph

Represents relationships between entities.

Example:

```
JS file → references → API endpoint
endpoint → exposed_on → host
finding → supports → hypothesis
```

Used by the planner for reasoning.

---

### Planner (LLM)

Consumes structured context and outputs:

* summaries
* hypotheses
* action proposals

Strict constraints:

* JSON output only
* evidence citations required
* no direct command execution

---

### Policy Engine

Evaluates proposed actions:

* allow
* block
* require approval

Checks:

* scope validity
* action type
* rate limits
* duplication
* sensitivity

---

### Reporter

Generates:

* markdown reports
* run summaries
* findings drafts
* execution timelines

Also acts as:

* CTF writeup assistant
* pentest notes system

---

## Execution Flow

1. Campaign initialized
2. Initial tasks seeded
3. Scheduler dispatches actions
4. Tools generate artifacts
5. Normalizer produces evidence
6. Evidence graph updated
7. Context builder prepares snapshot
8. Planner generates proposals
9. Policy engine evaluates actions
10. Approved actions are scheduled
11. Loop continues until completion

---

## Action Model

Everything is expressed as typed actions.

### Example action types

* passive_enum
* resolve_dns
* probe_http
* crawl
* extract_js_routes
* content_discovery
* template_scan
* compare_behavior
* summarize_surface
* draft_finding

Each action includes:

* parameters
* validation rules
* deduplication fingerprint
* execution handler

---

## Data Model

### Runs

Campaign metadata and lifecycle

### Tasks

Queued and executed actions

### Artifacts

Raw outputs from tools

### Evidence

Normalized structured facts

### Graph Edges

Relationships between evidence

### Hypotheses

AI-generated attack theories

### Proposals

Suggested actions from planner

### Findings

Validated results and reports

---

## Modes

### CTF Mode

* fast iteration
* minimal approval
* aggressive exploration
* heuristic reasoning

### Bug Bounty Mode

* wide surface discovery
* strict scope enforcement
* noise reduction
* prioritization

### Pentest Mode

* full audit trail
* approval gates
* conservative execution
* reporting focus

---

## Profiles

Profiles define behavior:

```yaml
name: bugbounty-web
allowed_actions:
  - passive_enum
  - probe_http
  - crawl
  - content_discovery
approval_required_for:
  - content_discovery
max_rps_per_host: 5
```

---

## Example Usage

```bash
penta recon example.com --mode bugbounty --profile web
```

```bash
penta resume run_123
```

```bash
penta report run_123
```

---

## Example Output (Planner)

```json
{
  "summary": "Multiple API endpoints discovered via JS analysis. Staging environment is accessible.",
  "hypotheses": [
    {
      "statement": "Undocumented admin API may be exposed",
      "confidence": 0.82
    }
  ],
  "proposed_actions": [
    {
      "type": "content_discovery",
      "target": "https://staging.example.com/api/",
      "priority": 1
    }
  ]
}
```

---

## Repository Structure

```
penta/
├── cmd/
├── internal/
│   ├── campaign/
│   ├── scheduler/
│   ├── policy/
│   ├── tools/
│   ├── runtime/
│   ├── normalize/
│   ├── evidence/
│   ├── graph/
│   ├── planner/
│   ├── contextbuilder/
│   ├── storage/
│   ├── reporting/
│   └── models/
├── configs/
├── schemas/
├── testdata/
└── docs/
```

---

## Roadmap

### v1

* core pipeline
* SQLite storage
* basic tool integration
* simple planner
* markdown reporting

### v2

* JS intelligence
* hypothesis tracking
* improved context building
* interactive TUI

### v3

* differential analysis
* collaborative campaigns
* advanced policy controls
* plugin system

---

## Security Considerations

* strict scope enforcement
* no autonomous exploitation
* audit logging
* rate limiting
* tool sandboxing (future)

---

## Limitations

* not a replacement for human expertise
* dependent on quality of evidence normalization
* LLM reasoning is probabilistic
* requires careful prompt design

---

## Contributing

Contributions should focus on:

* new action types
* tool integrations
* normalization improvements
* planner robustness
* policy rules

---

## License

TBD

---

## Final Notes

Penta is designed as a **serious offensive security platform**, not a demo.

The goal is to:

* reduce cognitive load during recon
* structure messy data into actionable insight
* provide a reasoning layer on top of tooling
* maintain control, safety, and reproducibility

AI is a component of Penta, not its foundation.

---

