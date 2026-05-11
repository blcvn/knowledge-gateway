---
id: TASK-CTX-004
title: "Infrastructure Layer & Bootstrap"
status: Done
created: 2026-05-11
---

# Task: Implement Infrastructure Layer

## 1. Objective
Finalize the Infrastructure Layer (Layer 4) to bootstrap the `memobase-context` service, configuring telemetry, dependency injection, and server lifecycle.

## 2. Requirements

### 2.1. Configuration & Telemetry
- **Config (`internal/infra/config/config.go`)**: Parse environment variables via Viper:
  - Base: `GRPC_PORT` (9033), `HEALTH_PORT` (9100), `LOG_LEVEL` (info)
  - Connections: `DB_DSN`, `REDIS_URL`, `NATS_URL`, `OTEL_ENDPOINT`
  - App Logic: `PROFILE_CACHE_TTL` (1200), `DEFAULT_MAX_TOKEN_SIZE` (500), `PROFILE_EVENT_RATIO` (0.7), `EVENT_SEARCH_THRESHOLD` (0.2), `EVENT_SEARCH_WINDOW_DAYS` (21), `EVENT_SEARCH_TOPK` (10)
- **Telemetry (`internal/infra/telemetry/`)**:
  - Setup structured logging (`slog`) with context propagation (`tenant_id`, `request_id`).
  - Initialize OpenTelemetry tracing for gRPC, Postgres, and Redis.
  - Configure Prometheus metrics (e.g., `context_latency_ms`, `cache_hit_ratio`).

### 2.2. Dependency Injection
- **Wire (`internal/infra/wire/wire.go`)**: Use Google Wire to bind Domain, Usecase, Adapter, and Infra layers together and generate `wire_gen.go`.

### 2.3. Server Bootstrap
- **Entry point (`cmd/main.go`)**:
  - Initialize infra, wire dependencies.
  - Start gRPC server and NATS consumer.
  - Implement graceful shutdown (SIGINT/SIGTERM).
  - Setup health check endpoints:
    - gRPC health: `SERVING` on port 9033.
    - HTTP `/healthz`: `{"status":"healthy"}` on port 9100.
    - HTTP `/readyz`: Verify DB, Redis, and NATS connectivity before returning `200 OK`.

## 3. Acceptance Criteria
- [x] Config loading and validation completed for all variables defined in documentation.
- [x] OpenTelemetry and Prometheus exporters are active.
- [x] Graceful shutdown successfully tears down gRPC, Postgres, Redis, and NATS connections.
- [x] Liveness (`/healthz`) and Readiness (`/readyz`) probes are implemented correctly.
- [x] 100% completion of service bootstrap aligning with enterprise standards.
