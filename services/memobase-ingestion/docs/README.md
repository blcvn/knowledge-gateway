---
id: DOC-S01
service: memobase-ingestion
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
owner: VNP Memory — Memobase Team
---

# memobase-ingestion

> **Group**: Memobase | **gRPC Port**: 9031 | **Health Port**: 9098 | **Origin**: Memobase

## Purpose

Blob ingestion service for the Memobase user-profile memory system. Accepts conversational, document, and summary blobs, manages a **token-aware Buffer Zone FSM**, and triggers flush to `memobase-engine` when the buffer threshold is reached.

### Business Capability

- **Blob Ingestion**: Accept `ChatBlob`, `DocBlob`, `SummaryBlob` with JSONB storage
- **Buffer Zone FSM**: Token-aware batching with 4-state machine (`IDLE → PROCESSING → DONE / FAILED`)
- **Token Counting**: tiktoken-go (gpt-4o encoder) for accurate token budgeting
- **Flush Triggering**: Automatic flush when `token_sum ≥ 1024` or idle timeout `> 1h`
- **Blob Lifecycle**: Raw blobs optionally deleted after successful processing (non-persistent mode)

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server + NATS JetStream publisher
- **Database**: PostgreSQL (blobs, buffer_zones tables)
- **Tokenizer**: tiktoken-go (gpt-4o encoder) via `pkg/tokenizer/`
- **Architecture**: 4-layer Clean Architecture (Domain → Usecase → Adapter → Infra)
- **DI**: Google Wire

## Quick Start

```bash
# From monorepo root
make build-memobase-ingestion
make run-memobase-ingestion

# Or with Docker
docker compose up memobase-ingestion postgresql nats
```

## API Surface

### gRPC Service

```protobuf
service MemobaseIngestionService {
  rpc InsertBlob(InsertBlobRequest) returns (InsertBlobResponse);
  rpc GetBufferStatus(BufferStatusRequest) returns (BufferStatus);
  rpc FlushBuffer(FlushBufferRequest) returns (FlushResponse);
  rpc DeleteBlob(DeleteBlobRequest) returns (google.protobuf.Empty);
}
```

### REST (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/memobase/users/{uid}/blobs` | Insert blob (chat/doc/summary) |
| POST | `/v1/memobase/users/{uid}/flush` | Force flush buffer |

### Buffer Zone FSM

```
IDLE → (token_sum ≥ 1024 OR idle > 1h) → PROCESSING → DONE → IDLE
PROCESSING → (fail) → FAILED → (retry) → PROCESSING
```

## NATS Events Published

| Subject | Payload | Subscriber |
|---------|---------|------------|
| `memobase.buffer.ready` | `{user_id, project_id, buffer_ids[]}` | memobase-engine |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL | SQL | Blob and buffer zone persistence |
| NATS JetStream | Publisher | Emit `memobase.buffer.ready` on flush trigger |
| memobase-engine | gRPC (indirect via NATS) | Receives buffer ready events |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)
- [Architecture Spec](../../../specs/architecture/04-memobase-services.md)
- [Memobase Reference](../../../references/memobase/)

## Owner

- **Team**: VNP Memory — Memobase
