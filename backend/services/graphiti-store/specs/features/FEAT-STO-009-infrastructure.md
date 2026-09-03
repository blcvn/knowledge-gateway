---
id: FEAT-STO-009
title: Infrastructure — Config, Server, Wire, OTel
service: graphiti-store
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement infrastructure layer: config loader, gRPC server, Wire DI, OTel telemetry, và Dockerfile. Bao gồm driver factory cho backend selection.

## Scope

- `internal/infra/config/config.go` — Viper with DRIVER_PROVIDER, NEO4J_URI, etc.
- `internal/infra/server/grpc.go` — Server with interceptors
- `internal/infra/wire/wire.go` — Wire providers + driver factory integration
- `internal/adapter/factory/driver_factory.go` — DRIVER_PROVIDER → GraphDriver
- `cmd/server/main.go` — Bootstrap → inject → serve → shutdown
- `Dockerfile` — Multi-stage build

### Driver Factory

```go
func NewGraphDriver(cfg Config) (domain.GraphDriver, error) {
    switch cfg.DriverProvider {
    case "neo4j":
        return neo4j.NewDriver(cfg.Neo4j)
    case "falkordb":
        return nil, ErrDriverNotImplemented
    default:
        return nil, ErrDriverNotSupported
    }
}
```

## Acceptance Criteria

- [ ] AC-1: `DRIVER_PROVIDER=neo4j` selects Neo4j driver
- [ ] AC-2: Config validation fails fast on missing NEO4J_URI
- [ ] AC-3: Wire gen produces valid injector
- [ ] AC-4: gRPC server starts on :9024, health on :9097
- [ ] AC-5: Graceful shutdown closes Neo4j driver (connection pool cleanup)

## Test Requirements
- **Unit tests**: Config validation, driver factory
- **Minimum coverage**: 70%
