---
id: DOC-S01
service: zep-memory
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Zep Team
---

# zep-memory

> **Group**: Zep (Context Engineering) | **gRPC Port**: 9063 | **Health Port**: 12063 | **Origin**: Zep

## Purpose

Core orchestrator của Zep — quản lý message ingestion và context assembly. Đây là service phức tạp nhất, orchestrate giữa zep-thread (session state) + zep-graph (async extraction) + zep-search (fact retrieval).

### Business Capability

- **PutMemory**: Ingest messages → PostgreSQL → trigger async graph extraction
- **GetMemory**: Assemble context = last N messages + relevant facts from Knowledge Graph
- **Message CRUD**: Store, retrieve, update messages per session
- **User Context**: Pre-formatted context blocks optimized for LLMs
- **Graceful Degradation**: GetMemory returns messages without facts if zep-search unavailable

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server + NATS JetStream publisher
- **Database**: PostgreSQL (messages table with role_type enum)
- **Architecture**: 4-layer Clean Architecture
- **Inter-service**: gRPC client → zep-thread, zep-search; NATS pub → zep-graph

## Quick Start

```bash
make build-zep-memory
make run-zep-memory
docker compose up zep-memory
```

## API Surface

### gRPC Service

```protobuf
service MemoryService {
  rpc PutMemory(PutMemoryRequest) returns (google.protobuf.Empty);
  rpc GetMemory(GetMemoryRequest) returns (MemoryResponse);
  rpc DeleteMemory(DeleteMemoryRequest) returns (google.protobuf.Empty);
  rpc GetMessagesForSession(GetMessagesRequest) returns (MessageListResponse);
  rpc GetMessage(GetMessageRequest) returns (MessageResponse);
  rpc UpdateMessageMetadata(UpdateMessageMetadataRequest) returns (MessageResponse);
  rpc GetUserContext(GetUserContextRequest) returns (UserContextResponse);
}
```

## Data Flow

### PutMemory Critical Path

```
Client → PutMemory(session_id, messages[])
  1. Upsert session via zep-thread (gRPC)
  2. Check session.ended_at → reject if ended
  3. Build message entities with role_type
  4. Batch INSERT → PostgreSQL
  5. Publish zep.memory.messages.ingested → NATS → zep-graph (async 10-20s)
```

### GetMemory Context Assembly

```
Client → GetMemory(session_id, last_n)
  1. Fetch last max(N, 4) messages from PostgreSQL
  2. Get session info from zep-thread
  3. Determine groupID = user_id ?? session_id
  4. Get relevant facts from zep-search (last 4 messages as search context)
  5. Assemble Memory response = messages + facts
```

## NATS Events

| Subject | Payload | Subscribers |
|---------|---------|-------------|
| `zep.memory.messages.ingested` | `{session_id, user_id, project_uuid, messages[], add_prefix}` | zep-graph (async entity extraction, 10-20s) |
| `zep.memory.deleted` | `{session_id, project_uuid}` | zep-graph (cleanup related episodes) |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| zep-thread | gRPC | UpsertSession, GetSession |
| zep-search | gRPC | GetRelevantFacts (for GetMemory context assembly) |
| zep-graph | NATS | PublishMessagesIngested (async graph extraction) |
| PostgreSQL | SQL | Message persistence |
| NATS JetStream | Pub | Async event publishing |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)
- [Reference Design](../../../references/zep/specs/services/04-memory-service.md)

## Owner

- **Team**: VNP Memory — Zep
