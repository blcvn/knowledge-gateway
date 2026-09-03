---
id: TASK-SRCH-005
title: "Bootstrap Service and Observability"
service: ov-search
status: Done
priority: High
created_at: 2026-05-11
---

# TASK-SRCH-005: Bootstrap Service and Observability

## Objective
Initialize the service lifecycle, configure dependency injection, and set up comprehensive observability metrics and tracing.

## Requirements

1. **Service Bootstrap (`cmd/server/main.go`)**:
   - Set up the main application entry point.
   - Start the gRPC server on port 9052.
   - Configure HTTP `/healthz` (serving status) and `/readyz` (dependency checks for db, qdrant, nats, bifrost) on port 9105.
   - Implement graceful shutdown via SIGTERM with a 30-second timeout.

2. **Dependency Injection (`internal/infra/wire/wire.go`)**:
   - Use Google Wire to stitch together Domain, Usecase, Adapter, and Infrastructure dependencies.

3. **Observability**:
   - Implement OpenTelemetry tracing spans (e.g., `ov-search.HierarchicalSearch` with sub-spans for pipeline steps).
   - Implement Prometheus metrics: `ov_search_query_total`, `ov_search_query_duration_seconds`, `ov_search_embedding_upsert_total`, `ov_search_hotness_recompute_duration_seconds`, `ov_search_propagation_depth`.
   - Implement structured JSON logging (`slog`) with context injection (`request_id`, `account_id`, `query_hash`).

## Constraints
- Ensure strict adherence to project standards for production-readiness.
- All configuration points must be environment-variable driven.
