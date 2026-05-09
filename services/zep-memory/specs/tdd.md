---
id: TDD-zep-memory
title: Technical Design — zep-memory
service: zep-memory
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Zep
---

# Technical Design — zep-memory

> **Group**: Zep | **gRPC Port**: 9063 | **Health Port**: 12063

## 1. Service Overview

Core orchestrator. PutMemory ingests messages → triggers async graph extraction. GetMemory assembles context = messages + relevant facts from KG. Most complex service — orchestrates zep-thread + zep-graph + zep-search.

## 2. Domain Model

- **Memory**: Composite (Messages + RelevantFacts + Metadata)
- **Message**: UUID, SessionID, ProjectUUID, Role, RoleType (enum), Content, TokenCount, Metadata
- **RoleType**: `norole|system|assistant|user|function|tool`
- **Fact**: UUID, Name, Fact, CreatedAt, ValidAt, InvalidAt, ExpiredAt
- **Domain Events**: MessagesIngested, MemoryDeleted
- **Domain Errors**: ErrSessionEnded, ErrEmptyMessages, ErrMessageNotFound, ErrInvalidRole

## 3. Critical Data Flows

### PutMemory (< 200ms target)
1. UpsertSession via zep-thread (gRPC)
2. Check session.ended_at → reject if ended
3. Build message entities with role_type
4. Batch INSERT → PostgreSQL
5. Publish MessagesIngested → NATS → zep-graph (async 10-20s)

### GetMemory (Context Assembly)
1. Fetch last max(N, 4) messages
2. Get session info from zep-thread
3. Determine groupID = user_id ?? session_id
4. Get relevant facts from zep-search (last 4 messages as context)
5. Assemble: messages[:N] + facts → MemoryResponse

## 4. Port Interfaces

```go
type MemoryService interface { // Input
  PutMemory, GetMemory, DeleteMemory,
  GetMessagesForSession, GetMessage, UpdateMessageMetadata, GetUserContext
}
type MessageRepository interface { // Output
  CreateMany, GetLastN, GetByUUID, UpdateMetadata, ListBySession, SoftDeleteBySession
}
type ThreadClient interface { UpsertSession, GetSession }
type SearchClient interface { GetRelevantFacts }
type GraphEventPublisher interface { PublishMessagesIngested }
```

## 5. NATS Events

| Subject | Subscribers |
|---------|-------------|
| `zep.memory.messages.ingested` | zep-graph (async extraction) |
| `zep.memory.deleted` | zep-graph (cleanup) |

## 6. Multi-Tenancy

Project isolation via `project_uuid` on every query.

---

> **Next Steps**: Decompose into FEAT specs in `specs/features/`.
