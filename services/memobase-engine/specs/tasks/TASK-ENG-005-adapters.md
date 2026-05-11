---
id: TASK-ENG-005
title: Implement Adapters (PostgreSQL, NATS, Bifrost, Embedder)
service: memobase-engine
layer: Adapter (Layer 3)
status: Done
---

# Task: Implement Adapters

## Objective
Implement all secondary driven adapters and the primary gRPC/NATS adapters as specified in the Layer 3 architecture.

## Requirements
1. **Repository Adapters (`internal/adapter/repository/postgres/`)**:
   - `profile_repo.go`: Implement `ProfileRepository` using PostgreSQL with `(user_id, project_id)` isolation.
   - `event_repo.go`: Implement `EventRepository` and `EventGistRepository` utilizing `pgvector` for semantic embeddings and HNSW indexes.

2. **Client Adapters (`internal/adapter/client/`)**:
   - `bifrost_llm.go`: Implement LLM client integrating with the Bifrost Gateway.
   - `embedder.go`: Implement Embedder client supporting OpenAI/Jina/Ollama.

3. **Event Adapters (`internal/adapter/event/`)**:
   - `nats_publisher.go`: Implement NATS JetStream publisher for `memobase.engine.completed`, `memobase.profile.changed`, and `memobase.event.created`.

4. **gRPC Adapters (`internal/adapter/grpc/`)**:
   - `handler.go` & `mapper.go`: Implement `ProcessBuffer`, `GetPipelineStatus`, `GetProfileConfig`, and `UpdateProfileConfig`.

## Constraints
- Database queries must enforce the multi-tenant `(id, project_id)` composite primary key logic.
- External client implementations must adhere strictly to the Usecase Output Ports.
