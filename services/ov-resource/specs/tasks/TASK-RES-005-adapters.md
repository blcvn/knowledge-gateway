---
id: TASK-RES-005
service: ov-resource
status: Done
---

# TASK-RES-005: Implement External Adapters (gRPC, NATS, FS)

## Objective
Implement external communication boundaries (`internal/adapter/`) defined in `api.md` and `architecture.md`.

## Requirements
1. **gRPC Server Handlers (`grpc/handler.go`)**:
   - Map `OvResourceService` Protobuf.
   - Implement `Ingest`, `Parse`, `Watch` (stream `WatchEvent` responses), and `Refresh`.
   - Enforce standard gRPC error mapping:
     - `INVALID_ARGUMENT` (400) for unsupported format.
     - `NOT_FOUND` (404) for watch source missing.
     - `RESOURCE_EXHAUSTED` (429) for full queue.
     - `INTERNAL` (500) for parse/write failures.
2. **ov-fs Client (`client/fs_client.go`)**:
   - Implement `FileWriterPort` calling `ov-fs` via gRPC at `OV_FS_ADDR`.
3. **NATS JetStream Publisher (`event/publisher.go`)**:
   - Connect to `NATS_URL`.
   - Publish `ov.resource.ingested` payload `{path, account_id, chunks, parser_type}` to the `openviking` stream.
4. **Bifrost Client (Optional/Fallback)**:
   - Provide interface structure if embedding generation via `Bifrost (LLM)` is delegated here per `README.md`.

## Dependencies
- gRPC, NATS JetStream, Protobuf definitions.
