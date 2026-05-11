---
id: FEAT-SEA-007
title: Infrastructure — Config, Server, Wire, OTel
service: graphiti-search
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement infrastructure layer cho graphiti-search: config, gRPC server, Wire DI, OTel telemetry, Dockerfile.

## Scope

- `internal/infra/config/config.go` — STORE_ADDR, REDIS_URL, NATS_URL, cache TTL, reranker weights
- `internal/infra/server/grpc.go` — gRPC server on :9022
- `internal/infra/wire/wire.go` — Wire providers + reranker factory
- `cmd/server/main.go` — Bootstrap
- `Dockerfile`

## Acceptance Criteria

- [ ] AC-1: gRPC on :9022, health on :9095
- [ ] AC-2: Config supports reranker weight customization
- [ ] AC-3: Wire gen produces valid injector
- [ ] AC-4: NATS subscriber starts on boot for cache invalidation
- [ ] AC-5: Graceful shutdown closes Redis + NATS connections

## Test Requirements
- **Unit tests**: Config validation
- **Minimum coverage**: 70%
