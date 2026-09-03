# CR-003 — Sessions Explorer: Mock Sessions/Conversations → Real API

| Field | Value |
|---|---|
| **CR ID** | CR-003 |
| **Title** | Sessions Explorer: Kết nối danh sách sessions, messages, working memory với backend API |
| **Type** | Feature Implementation |
| **Priority** | P0 — Critical |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Module** | Sessions Explorer |
| **Files thay đổi** | `ui/src/mock/session.mock.ts`, `ui/src/hooks/useSessions.ts`, `ui/src/services/session.service.ts` |

---

## 1. Hiện trạng

### Mock data ([`session.mock.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/mock/session.mock.ts))

```typescript
export const sessionMock = {
  sessions: [
    { id: 'sess_1', user_id: 'user_123', title: 'Memory Architecture Discussion', agent_id: 'agent-001', status: 'active', message_count: 12, ... },
    { id: 'sess_2', user_id: 'user_456', title: 'Graphiti Query Optimization', status: 'completed', ... },
    // 4 sessions hardcoded
  ],

  conversation: {
    session_id: 'sess_1',
    messages: [
      { id: 'msg_1', role: 'user', content: 'How does Graphiti handle temporal episodic memory?', ... },
      { id: 'msg_2', role: 'assistant', content: 'Graphiti stores episodic memories as a temporal knowledge graph...', memory_sources: ['graphiti:ep_abc', 'cognee:sem_001'] },
      // 6 messages hardcoded
    ]
  }
};
```

### Hooks ([`useSessions.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/hooks/useSessions.ts))

```typescript
export function useSessionList() {
  return useQuery({
    queryFn: useMock
      ? () => Promise.resolve(sessionMock.sessions)  // ← fake
      : () => sessionService.getSessions(),
  });
}

export function useSessionDetail(id: string) {
  return useQuery({
    queryFn: useMock
      ? () => Promise.resolve(sessionMock.conversation)  // ← luôn trả về session 1
      : () => sessionService.getSessionDetail(id),
    enabled: !!id,
  });
}
```

### Service đã định nghĩa ([`session.service.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/services/session.service.ts))

Các endpoints đã có:
- `GET /v1/console/sessions` — danh sách sessions
- `GET /v1/console/sessions/live` — sessions đang active
- `GET /v1/console/sessions/{id}` — chi tiết session + messages
- `GET /v1/console/sessions/{id}/timeline` — timeline events
- `GET /v1/console/sessions/{id}/diff` — memory diff
- `GET /v1/console/sessions/{id}/working-memory` — working memory state

---

## 2. Backend API cần implement

Base path: `/v1/console/sessions`

### 2.1 GET /v1/console/sessions

Danh sách sessions với pagination và filtering.

**Query params**:
| Param | Type | Default | Mô tả |
|---|---|---|---|
| `page` | int | 1 | Số trang |
| `page_size` | int | 20 | Kích thước trang |
| `status` | string | all | Filter: `active`, `completed`, `failed` |
| `user_id` | string | - | Filter theo user |
| `agent_id` | string | - | Filter theo agent |
| `search` | string | - | Full-text search trong title |
| `sort` | string | `updated_at:desc` | Sort field và direction |

**Response schema**:
```json
{
  "data": [
    {
      "id": "sess_abc123",
      "user_id": "user_xyz",
      "title": "Memory Architecture Discussion",
      "agent_id": "agent-001",
      "status": "active",
      "message_count": 12,
      "created_at": "2026-06-01T09:00:00Z",
      "updated_at": "2026-06-16T10:45:00Z"
    }
  ],
  "total": 150,
  "page": 1,
  "page_size": 20,
  "has_more": true
}
```

**Database schema** (PostgreSQL):
```sql
CREATE TABLE sessions (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID NOT NULL,
  agent_id      TEXT,
  title         TEXT NOT NULL DEFAULT 'Untitled Session',
  status        TEXT DEFAULT 'active' CHECK (status IN ('active', 'completed', 'failed')),
  message_count INT DEFAULT 0,
  tenant_id     UUID NOT NULL,
  created_at    TIMESTAMPTZ DEFAULT NOW(),
  updated_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sessions_tenant_status ON sessions(tenant_id, status);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_updated_at ON sessions(updated_at DESC);
```

### 2.2 GET /v1/console/sessions/live

Danh sách sessions đang active trong thời gian thực.

**Response**: Array của `Session` với `status='active'`, sort by `updated_at DESC`, limit 50.

Có thể dùng cùng query như `/sessions` với filter `status=active&page_size=50`.

### 2.3 GET /v1/console/sessions/{id}

Chi tiết một session bao gồm toàn bộ messages.

**Response schema** (khớp với [`Conversation`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/types/session.ts)):
```json
{
  "session_id": "sess_abc123",
  "messages": [
    {
      "id": "msg_001",
      "role": "user",
      "content": "How does Graphiti handle temporal episodic memory?",
      "timestamp": "2026-06-01T09:00:00Z",
      "memory_sources": null
    },
    {
      "id": "msg_002",
      "role": "assistant",
      "content": "Graphiti stores episodic memories as a temporal knowledge graph...",
      "timestamp": "2026-06-01T09:00:05Z",
      "memory_sources": ["graphiti:ep_abc", "cognee:sem_001"]
    }
  ]
}
```

**Database schema** (PostgreSQL):
```sql
CREATE TABLE messages (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id    UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  role          TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
  content       TEXT NOT NULL,
  memory_sources TEXT[],                        -- Array of engine:id references
  timestamp     TIMESTAMPTZ DEFAULT NOW(),
  metadata      JSONB DEFAULT '{}'
);

CREATE INDEX idx_messages_session_id ON messages(session_id, timestamp);
```

### 2.4 GET /v1/console/sessions/{id}/timeline

Timeline của memory operations trong session.

**Response schema**:
```json
[
  {
    "event_type": "memory_recalled",
    "engine": "graphiti",
    "memory_id": "ep_abc",
    "timestamp": "2026-06-01T09:00:04Z",
    "latency_ms": 45,
    "details": { "query": "temporal episodic", "score": 0.92 }
  },
  {
    "event_type": "memory_stored",
    "engine": "memobase",
    "memory_id": "prof_xyz",
    "timestamp": "2026-06-01T09:00:10Z",
    "latency_ms": 120,
    "details": { "topic": "Memory Architecture" }
  }
]
```

**Nguồn**: NATS JetStream event log hoặc bảng `session_events` trong PostgreSQL.

### 2.5 GET /v1/console/sessions/{id}/working-memory

Trạng thái working memory hiện tại của session (Zep).

**Response schema** (khớp với [`WorkingMemory`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/types/session.ts)):
```json
{
  "session_id": "sess_abc123",
  "summary": "User is exploring Graphiti's temporal memory capabilities. Comparing with Memobase for a combined agent architecture.",
  "entities": ["Graphiti", "Memobase", "temporal memory", "AI agent", "knowledge graph"]
}
```

**Nguồn**: Zep service (`zep-memory`) — `GET /v1/zep/sessions/{id}/memory`

### 2.6 GET /v1/console/sessions/{id}/diff

Memory diff — những gì thay đổi trong memory sau session này.

**Response schema**:
```json
{
  "session_id": "sess_abc123",
  "added": [
    { "engine": "graphiti", "memory_id": "ep_new_001", "content": "User prefers Graphiti over Memobase for temporal queries" }
  ],
  "updated": [
    { "engine": "memobase", "memory_id": "prof_123", "field": "expertise.graphiti", "before": "beginner", "after": "intermediate" }
  ],
  "deleted": []
}
```

---

## 3. Frontend thay đổi

### 3.1 Xóa mock dependency trong `useSessions.ts`

```typescript
// SAU — không còn mock
import { useQuery } from '@tanstack/react-query';
import { sessionService } from '../services/session.service';

export function useSessionList(filters?: {
  status?: string;
  page?: number;
  pageSize?: number;
  search?: string;
}) {
  return useQuery({
    queryKey: ['sessions', filters],
    queryFn: () => sessionService.getSessions(filters),
    staleTime: 30 * 1000,
  });
}

export function useLiveSessions() {
  return useQuery({
    queryKey: ['sessions', 'live'],
    queryFn: () => sessionService.getLiveSessions(),
    refetchInterval: 10 * 1000,  // Poll every 10s for live sessions
  });
}

export function useSessionDetail(id: string) {
  return useQuery({
    queryKey: ['sessions', id, 'detail'],
    queryFn: () => sessionService.getSessionDetail(id),
    enabled: !!id,
  });
}

export function useSessionTimeline(id: string) {
  return useQuery({
    queryKey: ['sessions', id, 'timeline'],
    queryFn: () => sessionService.getTimeline(id),
    enabled: !!id,
  });
}

export function useWorkingMemory(id: string) {
  return useQuery({
    queryKey: ['sessions', id, 'working-memory'],
    queryFn: () => sessionService.getWorkingMemory(id),
    enabled: !!id,
    refetchInterval: 5 * 1000,  // Poll mỗi 5s khi session active
  });
}

export function useSessionDiff(id: string) {
  return useQuery({
    queryKey: ['sessions', id, 'diff'],
    queryFn: () => sessionService.getDiff(id),
    enabled: !!id,
  });
}
```

### 3.2 Cập nhật `session.service.ts`

Thêm filter params vào `getSessions`:

```typescript
getSessions: (filters?: { status?: string; page?: number; pageSize?: number; search?: string }) => {
  const qs = new URLSearchParams();
  if (filters?.status) qs.set('status', filters.status);
  if (filters?.page) qs.set('page', String(filters.page));
  if (filters?.pageSize) qs.set('page_size', String(filters.pageSize));
  if (filters?.search) qs.set('search', filters.search);
  return apiClient.get<PaginatedResponse<Session>>(`${BASE}?${qs.toString()}`);
},
```

### 3.3 Cập nhật Type

```typescript
// Thêm vào session.ts
export interface SessionTimeline {
  event_type: string;
  engine: string;
  memory_id: string;
  timestamp: string;
  latency_ms: number;
  details: Record<string, unknown>;
}

export interface SessionDiff {
  session_id: string;
  added: Array<{ engine: string; memory_id: string; content: string }>;
  updated: Array<{ engine: string; memory_id: string; field: string; before: unknown; after: unknown }>;
  deleted: Array<{ engine: string; memory_id: string }>;
}
```

---

## 4. Điều kiện hoàn thành

- [ ] `GET /v1/console/sessions` trả về sessions từ PostgreSQL với pagination
- [ ] `GET /v1/console/sessions/live` trả về active sessions real-time
- [ ] `GET /v1/console/sessions/{id}` trả về messages từ database
- [ ] `GET /v1/console/sessions/{id}/working-memory` lấy từ Zep service
- [ ] `GET /v1/console/sessions/{id}/timeline` trả về event log
- [ ] Sessions Explorer không còn import từ `session.mock.ts`
- [ ] Pagination hoạt động đúng trên frontend
- [ ] Filter theo status (`active`/`completed`) hoạt động
- [ ] Message `memory_sources` hiển thị đúng engine references

---

## 5. Notes

> **Zep Integration**: `working-memory` endpoint cần forward đến `zep-memory` service qua gRPC. Nếu Zep service chưa ready, endpoint có thể trả về HTTP 503 — frontend cần handle gracefully.

> **Session Title**: Title hiện tại là `string`. Có thể implement auto-title generation (LLM summarize first 3 messages) trong background.
