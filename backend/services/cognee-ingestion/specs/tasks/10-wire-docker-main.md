---
id: TASK-ING-10
title: Implement Main Entrypoint, Wire, and Dockerfile
service: cognee-ingestion
feature: FEAT-ING-003
status: Done
---

## Objective
Finalize the service wiring, define the application entry point, and setup deployment artifacts.

## Files to Create/Update
- `internal/infra/wire/wire.go`: Define Google Wire provider sets.
- `cmd/server/main.go`: Implement the main application entry point.
- `Dockerfile`: Multi-stage Docker build producing a minimal image.
- `Makefile`: Commands for building, running, and wire generation.
- (Generated) `internal/infra/wire/wire_gen.go` by running wire.

## Acceptance Criteria
- Wire `wire_gen.go` script successfully generates without errors.
- `go run cmd/server/main.go` boots up the service successfully without errors (Smoke test).
- Docker image builds efficiently and size is minimal (<=50MB).
