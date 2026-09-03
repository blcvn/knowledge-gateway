# Change Request: CR-ZEP-002 — Message Ingestion & Context Assembly (Memory Service)

**CR ID:** CR-ZEP-002  
**Component:** `services/memory-service` [UPGRADE SERVICE]  
**Priority:** Critical  
**Status:** In Progress
**Reference:** Zep PRD §6.1 F2, SRS §5.1, specs/services/04-memory-service.md  
**Target:** sub-200ms PutMemory + GetMemory with graceful degradation

---

## 1. Mô tả

Nâng cấp **Memory Service** của VNP Memory để hỗ trợ đầy đủ Zep's message-centric memory model:

1. **PutMemory**: Ingest batch messages → PostgreSQL → publish NATS event cho async graph extraction.
2. **GetMemory**: Assemble context = last N messages + relevant facts từ Knowledge Graph.
3. **Role-typed Messages**: 6 role types: `norole`, `system`, `assistant`, `user`, `function`, `tool`.
4. **Session Lifecycle Guard**: Từ chối ingest nếu session đã `ended_at`.
5. **User Context (pre-formatted)**: Endpoint trả về context string đã format sẵn cho LLM prompt injection.
6. **Graceful Degradation**: Nếu graph search fails → trả về messages without facts (không crash).

---

## 2. Vấn đề hiện tại

- VNP Memory chưa hỗ trợ đầy đủ 6 role types (chỉ có user/assistant).
- Chưa có `GetUserContext` endpoint trả về formatted context block cho LLM.
- Thiếu session lifecycle guard — có thể add messages vào session đã kết thúc.
- Chưa có graceful degradation khi graph service không khả dụng.

---

## 3. Thay đổi đề xuất

### 3.1. [UPGRADE] `services/memory-service/` (Port gRPC: 9043)

### 3.2. Domain Model

```go
type Message struct {
    UUID        string
    SessionID   string
    ProjectUUID string
    Role        string      // free text: "user", "assistant", "system"
    RoleType    RoleType    // enum (typed)
    Content     string
    TokenCount  int
    Metadata    map[string]any  // JSONB
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   *time.Time  // soft delete
}

type RoleType string
const (
    RoleTypeNone      RoleType = "norole"
    RoleTypeSystem    RoleType = "system"
    RoleTypeAssistant RoleType = "assistant"
    RoleTypeUser      RoleType = "user"
    RoleTypeFunction  RoleType = "function"
    RoleTypeTool      RoleType = "tool"
)

// Fact = temporal edge từ Knowledge Graph
type Fact struct {
    UUID      string
    Name      string      // relationship label
    Fact      string      // human-readable statement
    CreatedAt time.Time
    ValidAt   *time.Time  // when fact became true
    InvalidAt *time.Time  // when fact ceased to be true
    ExpiredAt *time.Time  // when fact was superseded
}

// Memory = composite response (API overlay, not persisted)
type Memory struct {
    Messages      []Message
    RelevantFacts []Fact
    Metadata      map[string]any
}
```

### 3.3. PutMemory Flow (Critical Path — sub-200ms)

```
Client POST /api/v2/sessions/{id}/memory
  │
  ▼ (synchronous — sub-200ms)
1. Call Thread Service: UpsertSession(sessionID)
2. Check session.EndedAt → reject nếu ended (ErrSessionEnded)
3. Build Message entities với UUID mới
4. Batch INSERT messages → PostgreSQL
5. Publish NATS event "memory.messages.ingested" (async, non-blocking)
6. Return 200 OK ngay lập tức
  │
  └── (async 10-20s) → Graph Service extract entities → Neo4j/KG
```

### 3.4. GetMemory Flow (Context Assembly)

```
1. Fetch last max(N, 4) messages từ PostgreSQL
2. Get session.UserID (nếu có) → set groupID = userID ?? sessionID
3. Call Search Service: GetRelevantFacts(groupID, last4Messages, maxFacts=5)
   - Nếu Search Service fails → graceful degradation: trả về messages mà không có facts
4. Assemble Memory{messages[N], relevantFacts}
```

### 3.5. GetUserContext (Pre-formatted Block)

```go
// Trả về formatted context string cho injection vào system prompt
type UserContextResponse struct {
    Context string      // pre-formatted, ready to inject
    Facts   []Fact      // raw facts for programmatic access
}

// Format mẫu:
// "FACTS about this user:
//  - Alice worked at Acme Corp until June 2023 (fact: WORKED_AT, valid_at: 2020-01, invalid_at: 2023-06)
//  - Alice currently works at Beta Inc (fact: WORKS_AT, valid_at: 2023-07)
//  RECENT MESSAGES: ..."
```

### 3.6. Message Update

`PATCH /sessions/:id/messages/:uuid` — cập nhật metadata của message (JSONB merge-patch với advisory lock).

### 3.7. API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/api/v2/sessions/:id/memory` | Add messages (PutMemory) |
| `GET` | `/api/v2/sessions/:id/memory` | Get memory (messages + facts) |
| `DELETE` | `/api/v2/sessions/:id/memory` | Delete session memory |
| `GET` | `/api/v2/sessions/:id/messages` | List messages (paginated) |
| `GET` | `/api/v2/sessions/:id/messages/:uuid` | Get specific message |
| `PATCH` | `/api/v2/sessions/:id/messages/:uuid` | Update message metadata |

---

## 4. Acceptance Criteria

- [ ] POST messages với `role_type: "tool"` → được lưu đúng enum value.
- [ ] POST messages vào session đã ended → trả về 400 với message `session has been ended`.
- [ ] GET memory → response có cả `messages` array lẫn `relevant_facts` array.
- [ ] Nếu Graph Service down → GET memory trả về messages bình thường (facts array rỗng, không lỗi 500).
- [ ] `GET /sessions/:id/memory?lastN=10` → trả về đúng 10 messages gần nhất.
- [ ] GetUserContext trả về formatted string có thể inject thẳng vào system prompt.
- [ ] PutMemory latency p95 < 200ms (không tính async graph extraction).
