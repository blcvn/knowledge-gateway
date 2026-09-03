# Solution: SOL-ZEP-002 — Memory: Message Ingestion & Context Assembly

**CR ID:** CR-ZEP-002  
**Solution ID:** SOL-ZEP-002  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Nâng cấp `services/memory-service/` (zep domain) để implement đầy đủ Zep's memory model: **PutMemory** (batch message ingest < 200ms với NATS publish) + **GetMemory** (context assembly với graceful degradation khi graph down) + **GetUserContext** (pre-formatted block cho LLM). Tận dụng NATS JetStream embedded đã có sẵn.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| `ZepMemory` domain | `services/memory-service/internal/domain/zep/` | Có: ZepMemory, ZepMessage, ZepSession |
| `ZepMessage` entity | | Role(user\|assistant), Content — thiếu RoleType enum, TokenCount, Metadata |
| `MemoryService` usecase | `services/memory-service/internal/usecase/zep/` | Có: cơ bản |
| `zep-memory` gRPC service | `apps/memory/internal/bootstrap/` | Có: đăng ký |
| NATS JetStream | `apps/memory/internal/bus/` | Sẵn sàng: `zep.memory.messages.ingested` |

### Gap phân tích

- `ZepMessage.Role` chỉ là string tự do, thiếu `RoleType` enum với 6 values
- Thiếu `TokenCount` và `Metadata` trên Message
- Không có Session Lifecycle Guard (phải gọi Thread Service trước)
- Thiếu `GetUserContext` endpoint trả về formatted string
- Thiếu graceful degradation khi graph service down
- Chưa có `lastN` parameter cho GetMemory

---

## 3. Thiết kế Giải pháp

### 3.1. Domain Model (Nâng cấp)

```go
// services/memory-service/internal/domain/zep/memory.go
// [MODIFY] Nâng cấp ZepMessage và ZepMemory

package zep

import "time"

type RoleType string

const (
    RoleTypeNone      RoleType = "norole"
    RoleTypeSystem    RoleType = "system"
    RoleTypeAssistant RoleType = "assistant"
    RoleTypeUser      RoleType = "user"
    RoleTypeFunction  RoleType = "function"
    RoleTypeTool      RoleType = "tool"
)

// ValidRoleTypes: validate incoming role_type
var ValidRoleTypes = map[RoleType]bool{
    RoleTypeNone: true, RoleTypeSystem: true, RoleTypeAssistant: true,
    RoleTypeUser: true, RoleTypeFunction: true, RoleTypeTool: true,
}

type Message struct {
    UUID        string
    SessionID   string
    ProjectUUID string
    Role        string      // free text role label (e.g. "gpt-4o", "human")
    RoleType    RoleType    // typed enum — used for routing and filtering
    Content     string
    TokenCount  int         // token count of Content
    Metadata    map[string]any  // JSONB
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   *time.Time  // soft delete
}

// Fact = temporal edge from Knowledge Graph (read-only)
type Fact struct {
    UUID      string
    Name      string      // relationship label (e.g. "WORKS_AT")
    Fact      string      // human-readable: "Alice works at Beta Inc"
    CreatedAt time.Time
    ValidAt   *time.Time  // when fact became true (from Graphiti)
    InvalidAt *time.Time  // when fact ceased to be true
    ExpiredAt *time.Time  // when superseded by newer fact
}

// Memory = composite API response (NOT persisted, assembled on read)
type Memory struct {
    Messages      []Message
    RelevantFacts []Fact
    Metadata      map[string]any
}

// UserContextResponse = pre-formatted LLM-injectable context block
type UserContextResponse struct {
    Context string  // formatted, ready to inject into system prompt
    Facts   []Fact  // raw facts for programmatic access
}
```

### 3.2. PutMemory Flow (Critical Path — sub-200ms)

```go
// services/memory-service/internal/usecase/zep/put_memory.go

type PutMemoryUseCase struct {
    threadClient ThreadServiceClient    // gRPC → zep-thread (SOL-ZEP-001)
    msgRepo      MessageRepository
    publisher    EventPublisher         // NATS
    tokenizer    TokenizerPort          // count tokens per message
}

type PutMemoryRequest struct {
    SessionID   string
    ProjectUUID string
    Messages    []MessageInput
}

type MessageInput struct {
    Role     string
    RoleType RoleType
    Content  string
    Metadata map[string]any
}

// Execute MUST complete in sub-200ms (synchronous path)
// Async graph extraction happens via NATS (10-20s separate)
func (uc *PutMemoryUseCase) Execute(ctx context.Context, req PutMemoryRequest) error {
    // Step 1: Upsert session via Thread Service (< 10ms — in-process bufconn)
    session, err := uc.threadClient.UpsertSession(ctx, &threadpb.UpsertSessionRequest{
        SessionId:   req.SessionID,
        ProjectUuid: req.ProjectUUID,
    })
    if err != nil { return fmt.Errorf("upsert session: %w", err) }

    // Step 2: Session lifecycle guard
    if session.EndedAt != nil {
        return ErrSessionEnded  // 400: "session has been ended"
    }

    // Step 3: Build Message entities
    messages := make([]Message, 0, len(req.Messages))
    for _, m := range req.Messages {
        // Validate role_type
        if !ValidRoleTypes[m.RoleType] {
            m.RoleType = RoleTypeNone  // default for unknown types
        }

        messages = append(messages, Message{
            UUID:        newUUID(),
            SessionID:   req.SessionID,
            ProjectUUID: req.ProjectUUID,
            Role:        m.Role,
            RoleType:    m.RoleType,
            Content:     m.Content,
            TokenCount:  uc.tokenizer.Count(m.Content),
            Metadata:    m.Metadata,
            CreatedAt:   time.Now(),
            UpdatedAt:   time.Now(),
        })
    }

    // Step 4: Batch INSERT messages → PostgreSQL (single transaction)
    if err := uc.msgRepo.BatchInsert(ctx, messages); err != nil {
        return fmt.Errorf("batch insert messages: %w", err)
    }

    // Step 5: Publish NATS event (async, non-blocking, fire-and-forget)
    // Graph extraction will consume this in 10-20s (CR-ZEP-003)
    go func() {
        _ = uc.publisher.Publish(context.Background(), "zep.memory.messages.ingested",
            MessagesIngestedEvent{
                SessionID:   req.SessionID,
                ProjectUUID: req.ProjectUUID,
                UserID:      session.UserId, // optional
                Messages:    messages,
            })
    }()

    // Step 6: Return immediately (sub-200ms achieved)
    return nil
}
```

### 3.3. GetMemory Flow (Context Assembly with Graceful Degradation)

```go
// services/memory-service/internal/usecase/zep/get_memory.go

type GetMemoryUseCase struct {
    threadClient  ThreadServiceClient
    msgRepo       MessageRepository
    searchClient  SearchServiceClient    // gRPC → zep-search (SOL-ZEP-004)
}

type GetMemoryRequest struct {
    SessionID   string
    ProjectUUID string
    LastN       int      // number of recent messages to return (default 10, min 4)
}

func (uc *GetMemoryUseCase) Execute(ctx context.Context, req GetMemoryRequest) (*Memory, error) {
    // Enforce minimum: always fetch at least 4 messages for fact retrieval
    lastN := req.LastN
    if lastN < 4 { lastN = 4 }
    if lastN == 0 { lastN = 10 } // default

    // Step 1: Fetch last N messages from PostgreSQL (always succeeds)
    messages, err := uc.msgRepo.GetLastN(ctx, req.SessionID, req.ProjectUUID, lastN)
    if err != nil { return nil, err }

    // Step 2: Determine groupID for fact retrieval
    // groupID = userID if linked, else sessionID
    groupID := req.SessionID
    session, err := uc.threadClient.GetSession(ctx, req.SessionID, req.ProjectUUID)
    if err == nil && session.UserId != nil {
        groupID = *session.UserId
    }

    // Step 3: Get relevant facts — GRACEFUL DEGRADATION
    // If Search Service is down, return messages without facts (no 500 error)
    var facts []Fact
    factsCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
    defer cancel()

    searchResp, err := uc.searchClient.GetRelevantFacts(factsCtx, &searchpb.GetRelevantFactsRequest{
        GroupId:  groupID,
        Messages: last4Messages(messages),  // use last 4 for relevance query
        MaxFacts: 5,
    })
    if err != nil {
        // Graceful degradation: log warning, continue without facts
        slog.Warn("graph search unavailable, returning messages only",
            "session_id", req.SessionID,
            "error", err,
        )
        facts = []Fact{}  // empty, not nil → JSON: []
    } else {
        facts = convertFacts(searchResp.Facts)
    }

    // Step 4: Assemble Memory response
    return &Memory{
        Messages:      messages,
        RelevantFacts: facts,
        Metadata:      map[string]any{"session_id": req.SessionID},
    }, nil
}
```

### 3.4. GetUserContext (Pre-formatted LLM Block)

```go
// services/memory-service/internal/usecase/zep/get_user_context.go

type GetUserContextUseCase struct {
    searchClient SearchServiceClient
    msgRepo      MessageRepository
    threadClient ThreadServiceClient
}

func (uc *GetUserContextUseCase) Execute(ctx context.Context, req GetUserContextRequest) (*UserContextResponse, error) {
    // 1. Get user's facts from graph (edges scope)
    facts, err := uc.searchClient.GetUserFacts(ctx, &searchpb.GetUserFactsRequest{
        UserID:      req.UserID,
        ProjectUUID: req.ProjectUUID,
        MaxFacts:    20,
    })
    if err != nil {
        facts = &searchpb.GetUserFactsResponse{Facts: []*searchpb.Fact{}}
    }

    // 2. Get recent messages (last 5)
    var recentMessages []Message
    if req.ThreadID != "" {
        recentMessages, _ = uc.msgRepo.GetLastN(ctx, req.ThreadID, req.ProjectUUID, 5)
    }

    // 3. Format context block
    context := buildContextBlock(facts.Facts, recentMessages)

    return &UserContextResponse{
        Context: context,
        Facts:   convertFacts(facts.Facts),
    }, nil
}

// buildContextBlock formats facts and messages for LLM injection
func buildContextBlock(facts []*searchpb.Fact, messages []Message) string {
    var sb strings.Builder

    if len(facts) > 0 {
        sb.WriteString("FACTS about this user:\n")
        for _, f := range facts {
            line := fmt.Sprintf(" - %s", f.Fact)
            if f.ValidAt != nil {
                line += fmt.Sprintf(" (valid from: %s", f.ValidAt.AsTime().Format("2006-01"))
            }
            if f.InvalidAt != nil {
                line += fmt.Sprintf(", until: %s", f.InvalidAt.AsTime().Format("2006-01"))
            }
            if f.ValidAt != nil { line += ")" }
            sb.WriteString(line + "\n")
        }
        sb.WriteString("\n")
    }

    if len(messages) > 0 {
        sb.WriteString("RECENT MESSAGES:\n")
        for _, m := range messages {
            sb.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, m.Content))
        }
    }

    return sb.String()
}
```

---

## 4. Database Schema

```sql
-- messages table (nâng cấp từ zep_messages)
CREATE TABLE messages (
    uuid         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   VARCHAR NOT NULL REFERENCES sessions(session_id) ON DELETE CASCADE,
    project_uuid UUID NOT NULL,
    role         VARCHAR NOT NULL,            -- free text (e.g. "gpt-4o", "human")
    role_type    VARCHAR NOT NULL DEFAULT 'norole',  -- enum: norole|system|assistant|user|function|tool
    content      TEXT NOT NULL,
    token_count  INT NOT NULL DEFAULT 0,
    metadata     JSONB DEFAULT '{}',
    created_at   TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at   TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    deleted_at   TIMESTAMPTZ                        -- soft delete
);

-- role_type constraint
ALTER TABLE messages ADD CONSTRAINT messages_role_type_check
    CHECK (role_type IN ('norole','system','assistant','user','function','tool'));

-- Indexes
CREATE INDEX msg_session_idx ON messages(session_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX msg_project_idx ON messages(project_uuid) WHERE deleted_at IS NULL;
CREATE INDEX msg_role_type_idx ON messages(role_type) WHERE deleted_at IS NULL;
CREATE INDEX msg_metadata_gin ON messages USING GIN(metadata);

-- Efficient last-N query:
-- SELECT * FROM messages WHERE session_id = $1 ORDER BY created_at DESC LIMIT $2
```

---

## 5. NATS Event Contract

```go
// Event published after successful PutMemory
type MessagesIngestedEvent struct {
    SessionID   string    `json:"session_id"`
    ProjectUUID string    `json:"project_uuid"`
    UserID      *string   `json:"user_id,omitempty"`
    Messages    []Message `json:"messages"`
    IngestedAt  time.Time `json:"ingested_at"`
}

// Published to: "zep.memory.messages.ingested"
// Consumed by: Graph Service (SOL-ZEP-003) for entity extraction (10-20s async)
```

---

## 6. API Endpoints

```
POST   /api/v2/sessions/{id}/memory    → PutMemory (batch ingest, sub-200ms)
GET    /api/v2/sessions/{id}/memory    → GetMemory (messages + facts, graceful degradation)
DELETE /api/v2/sessions/{id}/memory    → DeleteSessionMemory
GET    /api/v2/sessions/{id}/messages  → ListMessages (paginated)
GET    /api/v2/sessions/{id}/messages/{uuid} → GetMessage
PATCH  /api/v2/sessions/{id}/messages/{uuid} → UpdateMessageMetadata (advisory lock)
GET    /api/v2/users/{id}/context      → GetUserContext (pre-formatted LLM block)
```

**GetMemory query params:**
```
GET /api/v2/sessions/{id}/memory?lastN=10&minRating=0.7
```

---

## 7. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | Domain model upgrade (Message, RoleType, Fact, Memory) | 1 ngày |
| **P2** | DB schema migration (thêm role_type, token_count, metadata) | 0.5 ngày |
| **P3** | PutMemory (batch insert + NATS publish + lifecycle guard) | 2 ngày |
| **P4** | GetMemory (lastN + graceful degradation) | 1.5 ngày |
| **P5** | GetUserContext (format builder) | 1 ngày |
| **P6** | UpdateMessageMetadata (advisory lock) | 1 ngày |
| **P7** | Gateway handlers + tests | 1 ngày |

**Tổng:** ~8 ngày (Wave 3)

---

## 8. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| role_type "tool" → đúng enum | ValidRoleTypes map + DB CHECK constraint |
| POST vào ended session → 400 | EndedAt != nil → ErrSessionEnded |
| GET memory → messages + facts | GetMemoryUseCase: parallel fetch |
| Graph down → messages bình thường | Graceful degradation với timeout 100ms |
| lastN=10 → 10 messages gần nhất | `ORDER BY created_at DESC LIMIT $1` |
| GetUserContext → inject sẵn vào prompt | buildContextBlock() với FACTS + RECENT MESSAGES |
| PutMemory p95 < 200ms | Batch INSERT + non-blocking NATS publish |
