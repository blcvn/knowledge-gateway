---
id: DOC-S01
service: zep-thread
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Zep Team
---

# zep-thread

> **Group**: Zep (Context Engineering) | **gRPC Port**: 9062 | **Health Port**: 12062 | **Origin**: Zep

## Purpose

Quản lý lifecycle của Session/Thread entities — đơn vị tổ chức conversation giữa user và AI agent. Cung cấp Thread CRUD với metadata management (JSONB merge-patch), session state management (`ended_at` — blocks future message ingestion), advisory lock-based concurrency control, và user-thread association.

### Business Capability

- **Thread CRUD**: Create/Read/Update/Delete conversation threads
- **Session State**: `ended_at` field marks session closed — blocks future message ingestion
- **Advisory Locks**: PostgreSQL advisory locks (SHA-256 hash of session_id) cho concurrent metadata updates
- **User Association**: Optional `user_id` FK linking threads to users
- **Upsert**: Create-or-update pattern used by PutMemory flow

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server + NATS JetStream publisher
- **Database**: PostgreSQL (sessions table, advisory locks, JSONB metadata)
- **Architecture**: 4-layer Clean Architecture
- **DI**: Google Wire
- **Concurrency**: Advisory locks with retry policy (200ms→30s, max 15 retries, 2x multiplier)

## Quick Start

```bash
make build-zep-thread
make run-zep-thread
docker compose up zep-thread
```

## API Surface

### gRPC Service

```protobuf
service ThreadService {
  rpc CreateSession(CreateSessionRequest) returns (SessionResponse);
  rpc GetSession(GetSessionRequest) returns (SessionResponse);
  rpc UpdateSession(UpdateSessionRequest) returns (SessionResponse);
  rpc UpsertSession(UpsertSessionRequest) returns (SessionResponse);
  rpc EndSession(EndSessionRequest) returns (google.protobuf.Empty);
  rpc ListSessions(ListSessionsRequest) returns (SessionListResponse);
  rpc ListOrderedSessions(ListSessionsRequest) returns (SessionListResponse);
  rpc ListUserSessions(ListUserSessionsRequest) returns (SessionListResponse);
}
```

### REST (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v2/sessions` | Create session |
| `GET` | `/api/v2/sessions/{id}` | Get session |
| `PATCH` | `/api/v2/sessions/{id}` | Update session metadata |
| `GET` | `/api/v2/sessions` | List sessions |

## NATS Events

| Subject | Payload | Subscribers |
|---------|---------|-------------|
| `zep.thread.session.created` | `{session_id, user_id, project_uuid}` | zep-graph (init session context) |
| `zep.thread.session.ended` | `{session_id, project_uuid, ended_at}` | zep-memory (cleanup pending ops) |
| `zep.thread.session.deleted` | `{session_id, project_uuid}` | zep-memory (cascade messages), zep-graph (cascade graph data) |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL | SQL | Session persistence, advisory locks |
| NATS JetStream | Pub | Publish session lifecycle events |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)
- [Reference Design](../../../references/zep/specs/services/03-thread-service.md)

## Owner

- **Team**: VNP Memory — Zep
