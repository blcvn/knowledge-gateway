# SOL-004 — Solution: Sessions Explorer API (CR-003)

| Field | Value |
|---|---|
| **Solution ID** | SOL-004 |
| **CR** | [CR-003 — Sessions Explorer](../CR-003-SESSIONS.md) |
| **Architecture ref** | §4.2 FEAT-014 · §5.2 Zep Domain · §5.2 Event Domain · §6.4 NATS Events |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Implemented** | 2026-06-17 |

---

## 1. Phân tích kiến trúc

Session data nằm rải rác ở 2 service:

| Data | Domain / Service | Storage |
|---|---|---|
| Session list, metadata, message_count | `storage-service/domain/session` | PostgreSQL |
| Messages content + working memory | `zep-thread` + `zep-memory` (Zep domain §5.2) | PostgreSQL (Zep DB) |
| Memory timeline per session | `vnp-event` (Event Domain §5.2) + NATS events |  `UserEvent` table |
| Memory diff after session | `graphiti-search` + `memobase-engine` | Neo4j + PostgreSQL |
| User summary | `memobase-context` | PostgreSQL |

Console route `FEAT-014` ↔ `/v1/console/sessions/*` (§4.2).

NATS subjects liên quan (§6.4):
- `zep.memory.messages.ingested` → trigger khi có message mới
- `memobase.profile.changed` → trigger khi profile update sau session

---

## 2. Giải pháp Backend

### 2.1 Handler (`console_sessions_handler.go`)

```go
type ConsoleSessionsHandler struct {
    sessionRepo  SessionRepository    // storage-service, PostgreSQL
    zepMemory    ZepMemoryClient      // gRPC → zep-memory service
    zepThread    ZepThreadClient      // gRPC → zep-thread service
    eventClient  VNPEventClient       // gRPC → vnp-event service
    searchHub    VNPSearchHubClient   // gRPC → cross-engine search
    memoCtx      MemobaseContextClient
}
```

### 2.2 Session List — PostgreSQL

Session table là entity của `storage-service` domain (§5 storage-service/domain/session):

```go
// GET /v1/console/sessions?status=active&page=1&page_size=20&search=...
func (h *ConsoleSessionsHandler) List(w http.ResponseWriter, r *http.Request) {
    tenantID := authctx.TenantID(r.Context())
    filter := SessionFilter{
        TenantID: tenantID,
        Status:   r.URL.Query().Get("status"),
        Search:   r.URL.Query().Get("search"),
        Page:     queryInt(r, "page", 1),
        PageSize: queryInt(r, "page_size", 20),
    }

    sessions, total, err := h.sessionRepo.List(r.Context(), filter)
    // ...
    httputil.JSON(w, 200, PaginatedResponse{
        Data:     sessions,
        Total:    total,
        Page:     filter.Page,
        PageSize: filter.PageSize,
        HasMore:  total > filter.Page*filter.PageSize,
    })
}
```

**PostgreSQL schema** (trong `storage-service` migrations):
```sql
CREATE TABLE sessions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    user_id       UUID NOT NULL,
    agent_id      TEXT,
    title         TEXT NOT NULL DEFAULT 'Untitled Session',
    status        TEXT DEFAULT 'active' CHECK (status IN ('active','completed','failed')),
    message_count INT  DEFAULT 0,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sessions_tenant_status ON sessions(tenant_id, status);
CREATE INDEX idx_sessions_user_id ON sessions(user_id, tenant_id);
CREATE INDEX idx_sessions_updated ON sessions(updated_at DESC);
-- FTS
CREATE INDEX idx_sessions_title_fts ON sessions USING gin(to_tsvector('english', title));
```

### 2.3 Session Detail — Zep messages

```go
// GET /v1/console/sessions/{id}
func (h *ConsoleSessionsHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")
    tenantID := authctx.TenantID(r.Context())

    // Verify session belongs to tenant
    if _, err := h.sessionRepo.Get(r.Context(), sessionID, tenantID); err != nil {
        httputil.NotFound(w); return
    }

    // Fetch messages từ zep-thread via gRPC bufconn
    thread, err := h.zepThread.GetThread(r.Context(), &zep.GetThreadRequest{
        SessionId: sessionID,
    })
    // ...

    // Map ZepMessage → frontend Message type
    msgs := make([]Message, len(thread.Messages))
    for i, m := range thread.Messages {
        msgs[i] = Message{
            ID:            m.Id,
            Role:          m.Role,
            Content:       m.Content,
            Timestamp:     m.CreatedAt,
            MemorySources: m.Metadata["memory_sources"],
        }
    }

    httputil.JSON(w, 200, Conversation{
        SessionID: sessionID,
        Messages:  msgs,
    })
}
```

### 2.4 Working Memory — zep-memory

```go
// GET /v1/console/sessions/{id}/working-memory
func (h *ConsoleSessionsHandler) GetWorkingMemory(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")

    // Gọi zep-memory service qua bufconn
    mem, err := h.zepMemory.GetMemory(r.Context(), &zep.GetMemoryRequest{
        SessionId: sessionID,
    })
    // ...

    // Zep memory có: Summary + Facts (mapped to entities)
    entities := make([]string, len(mem.Facts))
    for i, f := range mem.Facts {
        entities[i] = f.Name
    }

    httputil.JSON(w, 200, WorkingMemory{
        SessionID: sessionID,
        Summary:   mem.Summary,
        Entities:  entities,
    })
}
```

### 2.5 Session Timeline — vnp-event

Dùng `vnp-event` service (§5.2 Event Domain), query UserEvent với filter `session_id`:

```go
// GET /v1/console/sessions/{id}/timeline
func (h *ConsoleSessionsHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")

    // vnp-event gRPC call — filter by session metadata
    events, err := h.eventClient.GetTimeline(r.Context(), &event.TimelineRequest{
        TenantID:  authctx.TenantID(r.Context()),
        SessionID: sessionID,  // filter trên metadata JSONB
    })

    httputil.JSON(w, 200, events)
}
```

Trong `vnp-event/domain/event`:
```go
UserEvent — TenantID, UserID, Engine, EventType, Action, GistText
// Thêm field SessionID để filter
```

### 2.6 Session Diff — Cross-engine

```go
// GET /v1/console/sessions/{id}/diff
// So sánh memory state trước và sau session bằng cách query
// memories created/modified trong time window của session
func (h *ConsoleSessionsHandler) GetDiff(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")
    session, _ := h.sessionRepo.Get(r.Context(), sessionID, ...)

    // Query memories created/updated between session.created_at and session.updated_at
    // từ event log của vnp-event
    added, _ := h.eventClient.GetMemoriesCreatedInSession(r.Context(), session)
    updated, _ := h.eventClient.GetMemoriesUpdatedInSession(r.Context(), session)

    httputil.JSON(w, 200, SessionDiff{
        SessionID: sessionID,
        Added:     added,
        Updated:   updated,
        Deleted:   []MemoryRef{},
    })
}
```

---

## 3. Luồng dữ liệu

```
GET /v1/console/sessions/{id}
    → Auth middleware → TenantID
    → SessionRepo.Get(id, tenantID) [PostgreSQL] ← verify ownership
    → ZepThread.GetThread(sessionID) [gRPC bufconn → zep-thread]
    → Map ZepMessage → Message
    → Response: Conversation{session_id, messages[]}
```

---

## 4. Frontend — `useSessions.ts`

```typescript
// Bỏ toàn bộ mock, add pagination support
export function useSessionList(params?: {
    status?: string; page?: number; pageSize?: number; search?: string;
}) {
    return useQuery({
        queryKey: ['sessions', params],
        queryFn: () => sessionService.getSessions(params),
        placeholderData: keepPreviousData,  // Giữ data cũ khi đổi page
    });
}

export function useLiveSessions() {
    return useQuery({
        queryKey: ['sessions', 'live'],
        queryFn: () => sessionService.getLiveSessions(),
        refetchInterval: 10_000,
    });
}
```
