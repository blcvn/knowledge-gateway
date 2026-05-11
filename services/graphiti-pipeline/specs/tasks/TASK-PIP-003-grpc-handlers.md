---
id: TASK-PIP-003
title: Implement gRPC Handler Adapters
feature: FEAT-PIP-003
status: Done
---

## Objective
Thực thi implement gRPC handler adapters cho 2 proto services dựa trên FEAT-PIP-003.

## Tasks
1. Tạo file `internal/adapter/grpc/mapper.go`
   - Implement bidirectional Proto ↔ Domain mapping.

2. Tạo file `internal/adapter/grpc/ingestion_handler.go`
   - Implement methods: `IngestEpisode`, `BulkIngest`, `GetEpisodeStatus`, `ListEpisodes`, `RemoveEpisode`.
   - Propagate proper gRPC status codes.

3. Tạo file `internal/adapter/grpc/knowledge_handler.go`
   - Implement methods: `ExtractEntities`, `ResolveEntities`, `ExtractEdges`, `ResolveEdges`, `GenerateEmbedding`, `Rerank`, `UpdateCommunity`.
   - Propagate proper gRPC status codes.

4. gRPC Interceptors
   - Tích hợp OTel tracing (tạo span per RPC).
   - Tích hợp panic recovery, structured logging.
   - Tích hợp tenant extraction (`x-tenant-id` metadata thành `GroupID`).

5. Unit Tests
   - Viết unit tests cho handler methods sử dụng mocked usecases.
   - Viết test cho mapper round-trip.
   - Đảm bảo coverage >= 80%.
