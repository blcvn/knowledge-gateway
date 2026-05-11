---
id: TASK-ING-006
title: Implement Infrastructure — Config, Server, Wire, OTel
service: graphiti-ingestion
type: task
status: done
priority: P0
created: 2026-05-11
dependencies: [TASK-ING-003, TASK-ING-004, TASK-ING-005]
estimated_time: 4h
linked_feat: FEAT-ING-006
---

## Objective
Implement infrastructure layer: config (Viper), gRPC server (:9021), Wire DI, OTel tracing, Prometheus metrics, Dockerfile.

## Scope
- `internal/infra/config/config.go` — KNOWLEDGE_ADDR, STORE_ADDR, POSTGRES_URI, NATS_URL
- `internal/infra/server/grpc.go` — Server on :9021, health on :9094
- `internal/infra/wire/wire.go` — Wire providers
- `cmd/server/main.go` — Bootstrap
- `Dockerfile`

## Acceptance Criteria
- [x] gRPC on :9021, health on :9094
- [x] Config validates KNOWLEDGE_ADDR + STORE_ADDR required
- [x] Wire gen produces valid injector
- [x] Graceful shutdown drains in-flight sagas
- [x] Same proto as graphiti-pipeline (swappable deployment)

## Test Requirements
- Unit tests: Config validation
- Minimum coverage: 70%
