---
id: TASK-SRCH-004
title: "Implement Adapter Layer and Clients"
service: ov-search
status: Done
priority: High
created_at: 2026-05-11
---

# TASK-SRCH-004: Implement Adapter Layer and Clients

## Objective
Implement the Adapter Layer (Layer 3) to expose gRPC handlers, integrate with message brokers (NATS), and implement external service clients.

## Requirements

1. **gRPC Handlers (`internal/adapter/grpc/handler.go`)**:
   - Implement `OvSearchService` with endpoints: `HierarchicalSearch`, `RetrieveContext`, `GetHotness`, `UpsertEmbedding`, `DeleteEmbedding`.
   - Extract `x-tenant-id` from gRPC metadata for authentication/isolation.
   - Map domain errors to standard gRPC error codes (NOT_FOUND, INVALID_ARGUMENT, INTERNAL, UNAVAILABLE).

2. **NATS Event Subscribers (`internal/adapter/event/subscriber.go`)**:
   - Implement subscribers for events: `ov.content.written`, `ov.content.deleted`, `ov.resource.ingested`, `ov.session.committed`.
   - Connect NATS events to corresponding Usecase methods (e.g., trigger upsert/delete/index/boost).

3. **External Clients (`internal/adapter/client/`)**:
   - Implement `bifrost_client.go` to connect to the Bifrost service for generating embeddings.
   - Implement `fs_client.go` to connect to the `ov-fs` service for tiered context loading (fetching L1/L2 content).

## Constraints
- Rely on defined output ports from the Usecase layer.
- Ensure correct data mapping from DTOs to Protobuf and vice versa.
