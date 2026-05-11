---
id: TASK-ING-002
title: Implement Usecase Layer for Ingestion Service
service: memobase-ingestion
status: DONE
created: 2026-05-11
---

# Task: Implement Usecase Layer for Ingestion Service

## Objective
Implement the Usecase Layer (Layer 2) to orchestrate blob insertion, token-aware batching, and buffer flushing.

## Requirements

1. **Ports and DTOs**:
   - Define input ports: `BlobInserter`, `BufferFlusher`.
   - Define output ports: `BlobStore`, `BufferStore`, `EventPublisher`.
   - Define DTOs corresponding to gRPC requests and responses (`InsertBlobRequest`, `InsertBlobResponse`, `BufferStatusRequest`, `BufferStatus`, etc.).

2. **InsertBlobUseCase**:
   - Store the incoming blob via `BlobStore`.
   - Create a buffer entry and calculate `token_size` using `tiktoken-go` (gpt-4o encoder).
   - Check if total token sum for the user's idle blobs `≥ 1024`.
   - If threshold reached, trigger the flush process (fire-and-forget background goroutine).
   - Support `persistent` flag: if false, raw blobs should be configured/marked for deletion after processing.

3. **FlushBufferUseCase**:
   - Query FSM `idle` entries via `BufferStore`.
   - Update status to `processing` (optimistic concurrency via `WHERE status='idle'`).
   - Publish `memobase.buffer.ready` event via `EventPublisher` with payload `{user_id, project_id, buffer_ids[], blob_type}`.

4. **GetBufferStatusUseCase**:
   - Aggregate metrics (`idle_count`, `processing_count`, `failed_count`, `total_tokens`) per user.

5. **DeleteBlobUseCase**:
   - Handle the deletion of a specific blob and its associated buffer zone entry.

## Constraints
- Only imports the domain layer and `tiktoken-go` (for token sizing).
- Fire-and-forget flush must not block the InsertBlob response.
