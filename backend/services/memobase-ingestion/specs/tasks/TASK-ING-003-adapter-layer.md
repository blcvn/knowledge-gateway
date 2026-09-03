---
id: TASK-ING-003
title: Implement Adapter Layer for Ingestion Service
service: memobase-ingestion
status: DONE
created: 2026-05-11
---

# Task: Implement Adapter Layer for Ingestion Service

## Objective
Implement the Adapter Layer (Layer 3) to handle gRPC communication, PostgreSQL persistence, and NATS event publishing.

## Requirements

1. **gRPC Handler**:
   - Implement `MemobaseIngestionServiceServer` for `InsertBlob`, `GetBufferStatus`, `FlushBuffer`, and `DeleteBlob`.
   - Implement data mapping between Protobuf messages and Domain entities/DTOs.
   - Extract `x-tenant-id` from gRPC metadata and map it to `project_id`.

2. **PostgreSQL Repositories**:
   - Implement `BlobRepository` to persist `general_blobs`.
   - Implement `BufferZoneRepository` to persist `buffer_zones` and handle status-based optimistic locking (`UPDATE ... WHERE status='idle'`).
   - Ensure composite primary key `(id, project_id)` is respected in queries.
   - Implement specific queries to support index strategies: `idx_blobs_user_type`, `idx_buffer_user_status`, `idx_buffer_status_idle`.

3. **NATS Event Publisher**:
   - Implement `EventPublisher` over NATS JetStream.
   - Publish `memobase.buffer.ready` events with the correct JSON payload.

## Constraints
- Repositories must use context-aware querying.
- Proper error mapping to gRPC status codes (`INVALID_ARGUMENT`, `NOT_FOUND`, `ALREADY_EXISTS`, `RESOURCE_EXHAUSTED`, `INTERNAL`, `UNAVAILABLE`).
