---
id: TASK-PIPE-004
title: Implement Infrastructure Layer & Testing
layer: infrastructure
status: Done
---

## Objective
Implement infrastructure integration, service bootstrap, and E2E acceptance tests.

## Requirements
1. **Config**: Load and validate configuration variables from `configuration.md`: `GRPC_PORT`, `HEALTH_PORT`, `DATABASE_URL`, `REDIS_URL`, `NATS_URL`, `LOG_LEVEL`, and `OTEL_ENDPOINT`. Ensure token threshold defaults to `1024`.
2. **LLM Integration (Bifrost)**: Implement Bifrost client for YOLO merge. Ensure token counting matches the LLM tokenizer.
3. **Bootstrap & Ops**: Set up gRPC server on `GRPC_PORT`, health checks on `HEALTH_PORT` (`grpc.health.v1.Health/Check`), graceful shutdown (SIGTERM with 30s grace period as per `runbook.md`), and dependency injection via Wire.
4. **Acceptance Testing**: Verify AC-1 to AC-5 defined in TDD:
   - AC-1: Buffer Zone FSM transitions.
   - AC-2: Exactly 3 LLM calls per flush.
   - AC-3: NATS events emitted.
   - AC-4: Token threshold respected.
   - AC-5: Blob cleanup on success.

## Constraints
- Maintain clear separation of infrastructure code from core domain/usecase logic.
