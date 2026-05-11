---
id: TASK-ING-004
title: Implement Infrastructure Layer and Bootstrap for Ingestion Service
service: memobase-ingestion
status: DONE
created: 2026-05-11
---

# Task: Implement Infrastructure Layer and Bootstrap for Ingestion Service

## Objective
Implement the Infrastructure Layer (Layer 4) to bootstrap the `memobase-ingestion` service, wiring all dependencies and initializing the gRPC server.

## Requirements

1. **Configuration**:
   - Initialize Viper to load environment variables and config files.

2. **Dependency Injection**:
   - Use Google Wire to generate dependency injection graph (`wire.go` -> `wire_gen.go`).
   - Wire domain usecases, PostgreSQL repos, NATS publishers, and gRPC handlers together.

3. **Server Setup**:
   - Setup gRPC Server on port `9031`.
   - Setup Health Check endpoints (gRPC health and HTTP `/healthz`) on port `9098`.

4. **Telemetry and Observability**:
   - Integrate OpenTelemetry (OTel) for tracing (custom spans for `InsertBlob`, `FlushBuffer` RPCs).
   - Integrate Prometheus metrics (`insert_blob_total`, `buffer_flush_total`, `buffer_token_sum`, `flush_latency_ms`).
   - Configure structured JSON logging (`slog`) with context injection (`request_id`, `tenant_id`, `user_id`).

5. **Main Entrypoint**:
   - Implement `cmd/main.go` for graceful startup and shutdown.
   - Ensure graceful shutdown properly closes PostgreSQL connections, NATS publishers, and gRPC servers.

## Constraints
- Ensure strict adherence to VNP Memory monorepo conventions.
- No business logic in the infrastructure or `cmd/main.go` files.
