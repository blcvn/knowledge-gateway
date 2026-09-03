---
id: TASK-COG-010
title: Implement Infrastructure Bootstrap and Dependency Injection
feature: FEAT-COG-003
status: Done
---
# Task: Implement Service Bootstrap

## Objective
Finalize Layer 4 (Infrastructure) by implementing application configuration, telemetry setup, server runtimes, and the Wire dependency injection container.

## Files to Create/Modify
- `internal/infra/config/config.go`
- `internal/infra/telemetry/telemetry.go`
- `internal/infra/server/server.go`
- `internal/infra/di/wire.go`
- `cmd/server/main.go`
- `Dockerfile`

## Requirements
- **Config**: Use Viper to load settings (gRPC ports, NATS, Neo4j, Qdrant, Postgres, Bifrost, Bulkhead config).
- **Telemetry**: Initialize OpenTelemetry for Traces, Metrics, and `slog` structure logging.
- **Server**: Setup gRPC server runtime. Implement a graceful shutdown to safely complete or pause in-progress pipeline stages.
- **Wire DI**: Create `wire.go` to construct the application dependency graph.
- **Dockerfile**: Provide a multi-stage Dockerfile keeping the target image lightweight (≤50MB).
