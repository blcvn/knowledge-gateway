---
id: FEAT-KNW-008
title: Infrastructure — Config, Server, Wire, OTel
service: graphiti-knowledge
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement infrastructure layer cho graphiti-knowledge: config, gRPC server, Wire DI, OTel, Dockerfile.

## Scope

- `internal/infra/config/config.go` — LLM_PROVIDER, LLM_MODEL, EMBEDDER_*, STORE_ADDR, NATS_URL
- `internal/infra/server/grpc.go` — Server on :9023, health on :9096
- `internal/infra/wire/wire.go` — Wire providers
- `cmd/server/main.go` — Bootstrap
- `Dockerfile`

## Acceptance Criteria

- [ ] AC-1: gRPC on :9023, health on :9096
- [ ] AC-2: Config validates LLM_API_KEY + STORE_ADDR required
- [ ] AC-3: Wire gen produces valid injector
- [ ] AC-4: Graceful shutdown flushes LLM metrics
- [ ] AC-5: Same proto as graphiti-pipeline/knowledge (swappable)

## Test Requirements
- **Unit tests**: Config validation
- **Minimum coverage**: 70%
