---
id: TASK-007
title: "Documentation — Architecture, API, Runbook"
app: apps/graphiti
version: 1.0.0
status: Done
priority: P2
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-006]
---

## Mục Tiêu

Tạo documentation cho Graphiti monolith app theo monorepo standards.

## Scope

### In Scope
- `docs/README.md` — Quick start, overview
- `docs/architecture.md` — Architecture diagram, design decisions
- `docs/configuration.md` — All config values, ENV vars
- `docs/runbook.md` — Startup, shutdown, troubleshooting
- `docs/changelog.md` — Initial changelog entry

### Out of Scope
- API docs (gateway đã có)
- Service-level docs (services đã có)

## Deliverables

### README.md
- Purpose: Single-binary deployment of Graphiti knowledge graph system
- Quick start: `make run` / `docker compose up`
- Architecture overview diagram
- Link to detailed docs

### architecture.md
- Embedded Service Supervisor pattern
- Phase-ordered startup diagram
- Inter-service communication (gRPC localhost + NATS)
- Zero-change constraint explanation
- Comparison: monolith vs microservices deployment

### configuration.md
- Full ENV var table with types, defaults, descriptions
- Config file format (YAML)
- Service-specific config mapping

### runbook.md
- Startup procedure + expected logs
- Health check endpoints
- Troubleshooting: service won't start, gRPC errors, NATS issues
- Graceful shutdown procedure
- Rolling update procedure

## Acceptance Criteria

- [x] AC-1: README contains working quick-start instructions
- [x] AC-2: Architecture doc has up-to-date diagrams
- [x] AC-3: All ENV vars documented in configuration.md
- [x] AC-4: Runbook covers common failure scenarios
- [x] AC-5: Changelog has initial v1.0.0 entry

## Definition of Done

- [x] All 5 docs created and reviewed
- [x] No broken links
