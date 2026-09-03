---
id: TASK-ING-09
title: Implement gRPC Server Infrastructure
service: cognee-ingestion
feature: FEAT-ING-003
status: Done
---

## Objective
Setup the gRPC server infrastructure including health checks and graceful shutdown.

## Files to Create/Update
- `internal/infra/server/grpc.go`: Implement server startup, gRPC Health v1, HTTP health endpoint (/healthz on port 9091), and graceful shutdown logic.
- Related `*_test.go` files.

## Acceptance Criteria
- Server accurately listens on the configured gRPC port.
- HTTP `/healthz` endpoint correctly returns 200 OK with health status.
- Receiving SIGTERM/SIGINT signals triggers connection draining and shuts down gracefully within 30s.
- Server lifecycle unit tests pass.
