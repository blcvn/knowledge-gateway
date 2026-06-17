# Design

## Current Behavior

The repository currently contains only `docs/KG_Service_TDD_v1.md`. There is no existing service code, no OpenSpec workspace, and no implementation backlog tied to the TDD. As a result, the design is documented but not yet decomposed into deliverable units.

## Problem Statement

The TDD covers a large platform surface:

- Identity and access with tenant/app resolution and grant-based sharing
- Ontology management for domain-specific schemas and query templates
- PostgreSQL as source of truth plus Graph DB and Qdrant replicas
- Generic query-template compilation and status handling
- REST and MCP APIs
- Security, audit, consistency, and performance constraints

Without a phased OpenSpec change, implementation is likely to drift from the TDD invariants, mix platform concerns with seed-domain concerns, or skip critical guardrails like deny-by-default ACL and no-raw-query enforcement.

## Goals

- Translate the TDD into a single implementation-ready change package.
- Preserve the TDD's domain-agnostic architecture boundaries.
- Separate platform engine work from legal-domain seed/configuration work.
- Provide a task sequence that can be executed incrementally with measurable acceptance.

## Non-Goals

- Implement the KG Service runtime in this change.
- Fully restate every endpoint and table definition from the TDD.
- Define detailed sprint staffing or project management assignments.

## Key Decisions

### 1. Treat the TDD as the source architecture and the OpenSpec change as the delivery contract

The TDD remains the full technical reference. The OpenSpec artifacts will summarize the required implementation intent, decisions, and checkpoints rather than duplicate all low-level details.

### 2. Use one umbrella change for end-to-end bootstrap

The repo has no prior OpenSpec history. Starting with one umbrella change keeps the implementation scope traceable to a single canonical design document. Follow-up sub-changes can later split out optimizations or domain expansions after the platform baseline exists.

### 3. Split the spec by implementation capability instead of keeping one umbrella spec

The first draft used a single broad platform spec, which was enough to capture direction but too coarse for implementation tracking. The refined change separates requirements into capability-focused specs:

- `identity-access`
- `ontology-plane`
- `write-path`
- `read-templates`
- `semantic-search`
- `sync-consistency`
- `admin-mcp-observability`

This makes each area easier to implement, test, review, and evolve independently while still rolling up to one change.

### 4. Keep capability requirements centered on platform invariants

Each capability spec focuses on stable behaviors that must hold regardless of domain:

- identity is resolved from credentials, never request body
- access is deny-by-default and grant-scoped
- write path commits to PostgreSQL before replica sync
- read path executes registered templates only with ACL injection
- search path filters by ACL and deletion state
- sync and admin surfaces preserve the same security model

This keeps the spec durable even if seed domains or storage engines evolve.

### 5. Separate platform work from legal ontology seed tasks

The TDD uses legal ontology as the first domain, but explicitly states that it is not hardcoded platform behavior. Tasks therefore distinguish:

- generic engine/platform implementation
- legal seed ontology/config and template registration
- later validation with a non-legal domain

### 6. Align tasks to the TDD rollout phases

The TDD already defines four implementation phases with acceptance criteria. Reusing those phases reduces translation loss and gives the team a natural delivery order.

## Risks And Mitigations

- Scope is broad for a single change.
  - Mitigation: tasks are grouped by phase with explicit dependencies and acceptance gates.
- Seed-domain details could leak into core service code.
  - Mitigation: tasks explicitly mark legal ontology artifacts as configuration through Ontology APIs, not hardcoded service logic.
- Cross-store consistency can be underdesigned if workers are implemented late.
  - Mitigation: sync workers and outbox processing are introduced before read/search is considered complete.
- Security controls may be postponed behind feature delivery pressure.
  - Mitigation: ACL, RLS, audit, and raw-query prohibition are elevated into the spec requirements and acceptance tests.

## Validation Strategy

- Review implementation against the spec requirements in this change.
- Review implementation against each capability spec rather than a single umbrella checklist.
- Validate each phase using the TDD acceptance statements:
  - Phase A: tenant isolation integration test
  - Phase B: ACL-protected read/search and five legal templates through generic routes
  - Phase C: grant propagation visible within five seconds and MCP coverage
  - Phase D: NFR validation, reconciliation, pentest, and non-legal domain trial
- Keep `docs/KG_Service_TDD_v1.md` as the detailed reference for schemas, API payloads, and NFR thresholds.
