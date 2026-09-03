---
id: FEAT-ING-006
title: Infrastructure — Config, Server, Wire, OTel
service: graphiti-ingestion
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement infrastructure layer: config (Viper), gRPC server (:9021), Wire DI, OTel tracing, Prometheus metrics, Dockerfile.

## Scope

- `internal/infra/config/config.go` — KNOWLEDGE_ADDR, STORE_ADDR, POSTGRES_URI, NATS_URL
- `internal/infra/server/grpc.go` — Server on :9021, health on :9094
- `internal/infra/wire/wire.go` — Wire providers
- `cmd/server/main.go` — Bootstrap
- `Dockerfile`

## Acceptance Criteria

- [ ] AC-1: gRPC on :9021, health on :9094
- [ ] AC-2: Config validates KNOWLEDGE_ADDR + STORE_ADDR required
- [ ] AC-3: Wire gen produces valid injector
- [ ] AC-4: Graceful shutdown drains in-flight sagas
- [ ] AC-5: Same proto as graphiti-pipeline (swappable deployment)

## Test Requirements
- **Unit tests**: Config validation
- **Minimum coverage**: 70%
