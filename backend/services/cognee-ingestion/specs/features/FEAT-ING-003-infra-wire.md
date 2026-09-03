---
id: FEAT-ING-003
title: Ingestion Service — Infrastructure + Wire DI
service: cognee-ingestion
version: 1.0.0
status: InProgress
priority: P0
created: 2026-05-10
updated: 2026-05-11
implementation_started: 2026-05-11
linked_sol: SOL-001
linked_feat: FEAT-ING-002
---

## Mục Tiêu

Implement Layer 4 (Infrastructure) cho cognee-ingestion — config loading, gRPC server bootstrap, graceful shutdown, OTel telemetry, Google Wire DI, Dockerfile, health checks.

## Scope

### In Scope
- Viper config loader with validation
- gRPC server setup + graceful shutdown
- OTel tracer/metrics/logger initialization
- Google Wire providers + injector
- gRPC health check service (gRPC Health v1)
- HTTP health endpoint (/healthz on port 9091)
- Dockerfile (multi-stage build)
- cmd/server/main.go entry point

### Out of Scope
- Domain/Usecase/Adapter (FEAT-ING-001, FEAT-ING-002)

## Thiết Kế Kỹ Thuật

### Directory Structure

```
services/cognee-ingestion/
├── cmd/server/main.go         # Entry point
├── internal/infra/
│   ├── config/config.go       # Viper config struct
│   ├── server/grpc.go         # gRPC server + health + graceful shutdown
│   ├── telemetry/
│   │   ├── tracer.go          # OTel tracer provider
│   │   ├── metrics.go         # Prometheus metrics
│   │   └── logger.go          # slog JSON logger
│   └── wire/
│       ├── wire.go            # Wire provider sets
│       └── wire_gen.go        # Generated
├── Dockerfile
├── Makefile
└── go.mod / go.sum
```

### Config Structure

```go
type Config struct {
    Service   ServiceConfig   `mapstructure:"service"`
    GRPC      GRPCConfig      `mapstructure:"grpc"`
    Health    HealthConfig    `mapstructure:"health"`
    Postgres  PostgresConfig  `mapstructure:"postgres"`
    MinIO     MinIOConfig     `mapstructure:"minio"`
    NATS      NATSConfig      `mapstructure:"nats"`
    Telemetry TelemetryConfig `mapstructure:"telemetry"`
}

type ServiceConfig struct {
    Name    string `mapstructure:"name" default:"cognee-ingestion"`
    Version string `mapstructure:"version"`
}

type GRPCConfig struct {
    Port int `mapstructure:"port" default:"9011"`
}
```

## Acceptance Criteria

- [ ] AC-1: Given valid config (env vars or yaml), When service starts, Then gRPC server listens on configured port
- [ ] AC-2: Given SIGTERM/SIGINT, When signal received, Then service drains connections and shuts down gracefully within 30s
- [ ] AC-3: Given /healthz HTTP endpoint, When service is healthy, Then return 200 OK with status
- [ ] AC-4: Given Wire providers, When `wire` command runs, Then `wire_gen.go` is generated without errors
- [ ] AC-5: Given Dockerfile, When `docker build`, Then multi-stage build produces minimal image ≤50MB
- [ ] AC-6: Given OTel config, When service runs, Then traces and metrics are exported to configured collector

## Test Requirements

- **Unit tests**: Config validation, server lifecycle
- **Smoke test**: `go run cmd/server/main.go` starts without error
- **Coverage**: ≥ 80%
