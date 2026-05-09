---
id: TDD-zep-thread
title: Technical Design — zep-thread
service: zep-thread
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Zep
---

# Technical Design — zep-thread

> **Group**: Zep | **gRPC Port**: 9062 | **Health Port**: 12062

## 1. Service Overview

Thread/Session lifecycle management. Tracks `ended_at` for session boundary detection. Advisory lock-based concurrency control for metadata updates. Upsert pattern used by PutMemory flow.

## 2. Clean Architecture Layers

### Domain Layer (Layer 1)
- **Session** entity: UUID, SessionID, UserID (nullable), ProjectUUID, Metadata, EndedAt, CreatedAt, UpdatedAt, DeletedAt
- **AdvisoryLockKey**: SHA-256 hash of session_id → int64
- **Domain Events**: SessionCreated, SessionEnded, SessionUpdated, SessionDeleted
- **Domain Errors**: ErrSessionNotFound, ErrSessionEnded, ErrLockTimeout, ErrInvalidSessionID
- **Behaviors**: `IsEnded()`, `CanAcceptMessages()`

### Usecase Layer (Layer 2)
- **CreateSession**: Create session with optional user_id link
- **UpsertSession**: Create-or-update (critical for PutMemory flow)
- **UpdateSession**: Advisory lock → merge metadata → persist
- **EndSession**: Set ended_at → block future ingestion → publish event
- **Port interfaces**: ThreadService (input), SessionRepository + LockManager + EventPublisher (output)

### Adapter Layer (Layer 3)
- **gRPC handler**: ThreadServiceServer with mappers
- **PostgreSQL repos**: SessionRepository + LockManager (advisory locks)
- **NATS publisher**: Session lifecycle events

### Infrastructure Layer (Layer 4)
- Config (Viper), Server (gRPC), Wire (DI), Telemetry (OTel + Prometheus)

## 3. gRPC API

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

## 4. NATS Events

| Subject | Subscribers |
|---------|-------------|
| `zep.thread.session.created` | zep-graph |
| `zep.thread.session.ended` | zep-memory |
| `zep.thread.session.deleted` | zep-memory, zep-graph |

## 5. Storage

PostgreSQL: sessions table with JSONB metadata, advisory locks, composite indexes.

## 6. Multi-Tenancy

Project isolation via `project_uuid`. `UNIQUE (session_id, project_uuid)`.

---

> **Next Steps**: Decompose into FEAT specs in `specs/features/`.
