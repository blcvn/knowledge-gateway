---
id: TASK-PIPE-003
title: Implement Adapter Layer
layer: adapter
status: Done
---

## Objective
Implement repository, gRPC, and event publisher adapters.

## Requirements
1. **PostgreSQL Repository**: Implement CRUD operations for `Blobs`, `BufferState`, `Profiles`, and `EventGists` (using `pgvector`). All queries must enforce tenant isolation.
2. **Redis Cache**: Implement fast lookup for Buffer state FSM and token counting cache.
3. **gRPC Handler**: Implement `MemobaseIngestionService` (`ingestion_handler.go`) using existing proto definitions. Enforce tenant isolation via `x-tenant-id` gRPC metadata. Note: JWT/API key validation is handled via Gateway as per `api.md`.
4. **NATS Publisher**: Implement publisher for `memobase.pipeline.completed`, `memobase.profile.changed`, and `memobase.event.created` events.

## Constraints
- Interface-driven implementations based on Usecase ports.
