---
id: TASK-005
title: "Health Aggregation + Main Entry Point"
app: apps/memobase
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-004]
---

## Mục Tiêu

Implement `cmd/memobase/main.go` (entry point) và `cmd/memobase/health.go` (aggregated health server).

## Scope

### In Scope
- `apps/memobase/cmd/memobase/main.go` — Main entry point
- `apps/memobase/cmd/memobase/health.go` — Aggregated health HTTP server

### Out of Scope
- Supervisor (TASK-002), Services (TASK-003), Gateway (TASK-004)

## Thiết Kế Kỹ Thuật

### main.go — Registers all services with supervisor in phase order:
- Phase 0 (Data): memobase-ingestion
- Phase 1 (Intelligence): memobase-engine
- Phase 2 (Application): memobase-context, memobase-pipeline
- Phase 3 (Gateway): vnp-gateway

### Health Endpoints
| Endpoint | Purpose | Response |
|----------|---------|----------|
| `/healthz` | Liveness | `200 {"status":"alive"}` |
| `/readyz` | Readiness | `200` if all serving, else `503` |
| `/status` | Detail | `200 {service→status}` |

## Acceptance Criteria

- [x] AC-1: `main.go` registers all 4 services + gateway with correct phases
- [x] AC-2: Handles SIGINT/SIGTERM for graceful shutdown
- [x] AC-3: `/healthz` returns 200 when process alive
- [x] AC-4: `/readyz` returns 503 during startup, 200 when all serving
- [x] AC-5: `/status` returns per-service status JSON
- [x] AC-6: `go build ./cmd/memobase/` produces single binary
- [x] AC-7: Binary starts services in correct phase order
