# 05 — Cognee Memory Service

> **gRPC**: 9014 | **Health**: 9094

---

## 1. Purpose

Session-based agent memory (V2 API): remember/recall/forget, session lifecycle management, và session-to-graph bridging (persist session → KG via Cognify).

---

## 2. Clean Architecture

```
services/cognee-memory/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # Session, MemoryMessage, MemoryFact
│   │   ├── value_object.go     # MessageRole, MemoryType, SessionState
│   │   ├── event.go            # SessionPersistedEvent
│   │   └── errors.go
│   ├── usecase/
│   │   ├── remember.go         # Add messages to session memory
│   │   ├── recall.go           # Retrieve relevant memories (search-backed)
│   │   ├── forget.go           # Delete session/memories
│   │   ├── list_sessions.go
│   │   ├── persist_session.go  # Bridge: session → graph (emit to Cognify)
│   │   ├── port/
│   │   │   ├── input.go        # RememberUseCase, RecallUseCase, ForgetUseCase
│   │   │   └── output.go       # SessionRepository, MemorySearcher, EventPublisher
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/handler.go     # cognee.memory.v1.MemoryService impl
│   │   ├── repository/
│   │   │   ├── postgres/       # Session table, message history
│   │   │   └── redis/          # Session cache (hot sessions)
│   │   ├── client/
│   │   │   └── search_client.go # gRPC call to cognee-search for recall
│   │   └── event/
│   │       └── publisher.go    # NATS: cognee.memory.session.persisted
│   └── infra/
│       ├── config/config.go
│       └── wire/wire.go
```

---

## 3. API Operations

### Remember
```
Remember(session_id, messages[]) → ack
  1. Get/create session
  2. Append messages to session store
  3. Update session cache (Redis)
  4. (Optional) Auto-persist if threshold reached
```

### Recall
```
Recall(session_id, query, top_k) → MemoryResult[]
  1. Search session messages (in-session)
  2. Call cognee-search for KG-backed memories
  3. Merge + rank by relevance
  4. Return with provenance
```

### Forget
```
Forget(session_id | memory_ids[]) → ack
  1. Delete from session store
  2. Invalidate cache
  3. (Optional) Soft-delete for audit
```

### PersistSession
```
PersistSession(session_id) → ack
  1. Collect all session messages
  2. Emit cognee.memory.session.persisted
  3. Cognify subscriber processes → KG
  4. Mark session as "persisted"
```

---

## 4. Domain Entities

```go
type Session struct {
    ID         uuid.UUID
    TenantID   string
    UserID     string
    State      SessionState    // ACTIVE, PERSISTED, ARCHIVED
    CreatedAt  time.Time
    LastActive time.Time
    Metadata   map[string]string
}

type MemoryMessage struct {
    ID        uuid.UUID
    SessionID uuid.UUID
    Role      MessageRole     // USER, ASSISTANT, SYSTEM
    Content   string
    Timestamp time.Time
    Metadata  map[string]string
}

type MemoryFact struct {
    ID         uuid.UUID
    SessionID  uuid.UUID
    Fact       string          // Extracted fact from conversation
    Confidence float64
    Source     string          // Which message(s) produced this
}
```

---

## 5. NATS Events

| Subject | Direction | Payload |
|---------|-----------|---------|
| `cognee.memory.session.persisted` | Publish | `{session_id, messages[], tenant_id}` |
