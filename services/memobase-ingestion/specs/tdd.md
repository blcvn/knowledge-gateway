---
id: TDD-memobase-ingestion
title: Technical Design — memobase-ingestion
service: memobase-ingestion
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-09
group: Memobase
---

# Technical Design — memobase-ingestion

> **Group**: Memobase | **gRPC Port**: 9031 | **Health Port**: 9098

## 1. Service Overview

Blob ingestion service with token-aware Buffer Zone FSM. Accepts ChatBlob, DocBlob, SummaryBlob, tracks token counts via tiktoken, and triggers flush to memobase-engine via NATS when threshold reached.

## 2. Clean Architecture Layers

### Domain Layer (Layer 1)
- **BlobType**: enum (chat, doc, summary)
- **GeneralBlob**: id, user_id, project_id, blob_type, blob_data (JSONB), add_fields
- **BufferZone**: id, user_id, project_id, blob_id, blob_type, token_size, status (FSM)
- **BufferStatus**: FSM states — `idle`, `processing`, `done`, `failed`
- **BufferReadyEvent**: domain event {user_id, project_id, buffer_ids[]}

### Usecase Layer (Layer 2)
- **InsertBlobUseCase**: Store blob → create buffer entry (token_size = tiktoken.encode(blob)) → check threshold → trigger flush
- **FlushBufferUseCase**: Query idle entries → update status to processing → publish NATS event
- **GetBufferStatusUseCase**: Aggregate buffer metrics per user

### Adapter Layer (Layer 3)
- **gRPC handler**: InsertBlob, GetBufferStatus, FlushBuffer, DeleteBlob
- **PostgreSQL repos**: BlobRepository (CRUD), BufferZoneRepository (FSM transitions)
- **NATS publisher**: `memobase.buffer.ready` with {user_id, project_id, buffer_ids[]}

### Infrastructure Layer (Layer 4)
- Config (Viper), Server (gRPC), Wire (DI), Telemetry (OTel + Prometheus)

## 3. gRPC API

```protobuf
service MemobaseIngestionService {
  rpc InsertBlob(InsertBlobRequest) returns (InsertBlobResponse);
  rpc GetBufferStatus(BufferStatusRequest) returns (BufferStatus);
  rpc FlushBuffer(FlushBufferRequest) returns (FlushResponse);
  rpc DeleteBlob(DeleteBlobRequest) returns (google.protobuf.Empty);
}
```

## 4. Buffer Zone FSM

```
IDLE → (token_sum ≥ 1024 OR idle > 1h) → PROCESSING → DONE → IDLE
PROCESSING → (fail) → FAILED → (retry) → PROCESSING
```

**Concurrency**: Status-based optimistic locking (`WHERE status=idle` in SQL query prevents re-processing).

## 5. NATS Events

| Direction | Subject | Payload |
|-----------|---------|---------|
| Publish | `memobase.buffer.ready` | `{user_id, project_id, buffer_ids[], blob_type}` |

## 6. Data Model

### Tables
- `general_blobs`: id (PK), user_id, project_id, blob_type, blob_data (JSONB)
- `buffer_zones`: id (PK), user_id, project_id, blob_id (FK), token_size, status (FSM)

### Key Indexes
- `idx_buffer_user_status`: (user_id, project_id, blob_type, status) — buffer queries
- `idx_blobs_user_type`: (user_id, project_id, blob_type) — blob lookups

## 7. Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL | SQL | Blob + buffer persistence |
| NATS JetStream | Publisher | memobase.buffer.ready events |

## 8. Observability

- **Metrics**: insert_blob_total, buffer_flush_total, buffer_token_sum (gauge), flush_latency_ms
- **Traces**: OTel spans for InsertBlob, FlushBuffer RPCs
- **Logs**: Structured JSON via slog with request_id, tenant_id, user_id
- **Health**: gRPC health check + HTTP /healthz on port 9098

## 9. Multi-Tenancy

Composite PK `(id, project_id)` on all tables. `project_id` propagated via gRPC metadata `x-tenant-id`.

---

> **Next Steps**: Decompose into FEAT-001 (InsertBlob API), FEAT-002 (Buffer Zone FSM), FEAT-003 (NATS flush trigger) in `specs/features/`.
