---
id: TASK-ENG-006
title: Implement Infrastructure, Bootstrap, and Observability
service: memobase-engine
layer: Infrastructure (Layer 4)
status: Done
---

# Task: Implement Infrastructure, Bootstrap, and Observability

## Objective
Establish the entry point, dependency injection, service configuration, and enterprise-grade observability.

## Requirements
1. **Configuration & Server (`internal/infra/`)**:
   - `config/config.go`: Implement Viper-based config mapping exactly to the `configuration.md` (e.g., `BIFROST_URL`, `BEST_LLM_MODEL`).
   - `server/grpc.go`: Setup gRPC server with interceptors.

2. **NATS Consumer**:
   - Implement the NATS JetStream consumer listening to `memobase.buffer.ready`.
   - Wire the consumer to the `ProcessBufferUseCase`.

3. **Observability (`internal/infra/telemetry/`)**:
   - Implement OTel tracing (spans per pipeline step: entry_summary, extract, merge, persist).
   - Implement Prometheus metrics (`llm_invocations_total`, `llm_latency_ms`, `llm_tokens_input`, `profile_updates_total`).
   - Implement structured JSON logging (`slog` or `zap`) with `tenant_id`, `request_id`, `pipeline_id`.

4. **Bootstrap (`cmd/main.go` & `internal/infra/wire/`)**:
   - `wire.go`: Implement Google Wire dependency injection binding all 4 layers.
   - `main.go`: Application entry point and graceful shutdown logic.
   - Setup `/healthz` HTTP health check on port 9099.
   - Setup `/readyz` HTTP ready check on port 9099 (DB + NATS + Bifrost connected).
   - Implement Startup Checks: LLM sanity check (Bifrost max_tokens=16), Embedding dimension validation (`EMBEDDING_DIM`), PostgreSQL pgvector extension check (`CREATE EXTENSION IF NOT EXISTS vector`).

## Constraints
- Must strictly use `x-tenant-id` context propagation for gRPC.
- Final implementation must successfully compile and bootstrap all components.
