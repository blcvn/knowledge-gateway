# UI Solution: UI-SOL-AM-001 — Observe Service Dashboard

**Solution ID:** UI-SOL-AM-001  
**CR References:** [CR-AM-001](../../../../docs/crs/v1/agentmemory/CR-AM-001-Observe-Service.md)  
**Backend Solution:** [SOL-001-Observe-Service.md](../../../../backend/specs/crs/v1/agentmemory/solutions/SOL-001-Observe-Service.md)  
**Feature:** Observe Service — Hook Capture Pipeline & Session Viewer  
**Priority:** 🔴 Critical  
**Frontend Component:** `ui/src/pages/sessions/` + `ui/src/pages/observability/`

---

## 1. Mục Đích

Xây dựng UI cho Observe Service cho phép admin/developer:
- Xem danh sách tất cả agent sessions với real-time status
- Xem chi tiết hooks timeline của từng session
- Replay session theo trình tự thời gian (SSE streaming)
- Import JSONL từ Claude Code để phân tích

---

## 2. Backend API Alignment

### API Endpoints Sử Dụng

| Method | Path | Mô tả |
|--------|------|--------|
| `GET` | `/v1/observe/sessions` | Danh sách sessions với filters |
| `GET` | `/v1/observe/sessions/{id}` | Session detail + hooks array |
| `GET` | `/v1/observe/sessions/{id}/replay` | SSE stream replay |
| `POST` | `/v1/observe/sessions/import` | Import JSONL file |

### TypeScript Types

```typescript
// ui/src/types/observe.ts

interface ObserveSession {
  session_id:   string;
  agent_id:     string;
  user_id?:     string;
  started_at:   string;       // ISO 8601
  ended_at?:    string;
  status:       'active' | 'completed' | 'abandoned';
  hook_count:   number;
  total_tokens: number;
  duration_ms:  number;
}

interface HookEvent {
  hook_type:  'session_start' | 'session_end' | 'llm_prompt' |
              'llm_response' | 'tool_call' | 'tool_result' | 'observe_hook';
  timestamp:  string;          // ISO 8601
  sequence:   number;
  payload:    Record<string, unknown>;
  latency_ms?: number;
}

interface SessionDetail extends ObserveSession {
  hooks: HookEvent[];
}

interface ImportResult {
  session_id:     string;
  hooks_imported: number;
}
```

---

## 3. Components Architecture

### 3.1 Sessions List Page (`/sessions`)

```
SessionsPage
├── SessionFilters          ← status, agent_id, date range
├── SessionsTable           ← sortable, paginated
│   ├── SessionRow          ← status badge, hook count, duration
│   └── LiveBadge           ← pulsing dot if status='active'
└── SessionImportButton     ← trigger JSONL import modal
```

**React Query Hook:**
```typescript
// ui/src/api/hooks/useObserveSessions.ts
export function useObserveSessions(filters: SessionFilters) {
  return useQuery({
    queryKey: ['observe', 'sessions', filters],
    queryFn: () => observeApi.listSessions(filters),
    refetchInterval: 5000,   // auto-refresh mỗi 5s cho live sessions
  });
}
```

### 3.2 Session Detail Page (`/sessions/{id}`)

```
SessionDetailPage
├── SessionHeader           ← agent_id, status, duration, token summary
├── HooksTimeline           ← vertical timeline, scrollable
│   ├── HookCard            ← hook_type badge, timestamp, payload preview
│   │   ├── LLMPromptCard   ← collapsible prompt content, token count
│   │   ├── ToolCallCard    ← tool name, args, result
│   │   └── DefaultCard     ← raw JSON payload
│   └── TimelineConnector   ← vertical line between hooks
├── ReplayControls          ← Play/Pause/Seek slider
└── ContextPanel            ← hook detail JSON viewer (right panel)
```

**Replay với SSE:**
```typescript
// ui/src/api/hooks/useSessionReplay.ts
export function useSessionReplay(sessionId: string) {
  const [hooks, setHooks] = useState<HookEvent[]>([]);
  const [playing, setPlaying] = useState(false);

  useEffect(() => {
    if (!playing) return;
    const sse = new EventSource(`/v1/observe/sessions/${sessionId}/replay`);
    sse.onmessage = (e) => {
      const hook = JSON.parse(e.data) as HookEvent;
      setHooks(prev => [...prev, hook]);
    };
    sse.onerror = () => setPlaying(false);
    return () => sse.close();
  }, [playing, sessionId]);

  return { hooks, playing, setPlaying };
}
```

### 3.3 JSONL Import Modal

```typescript
// ui/src/components/sessions/JSONLImportModal.tsx
// Drag-and-drop .jsonl file → POST /v1/observe/sessions/import (multipart)
// Show progress: "Importing 47 hooks..."
// Redirect to new session detail on success
```

---

## 4. UI Layout & Design

### Sessions Table Columns
| Column | Format |
|--------|--------|
| Agent | `claude-code` badge với icon |
| Status | `active` (green pulse) / `completed` (gray) / `abandoned` (red) |
| Hooks | Number với icon |
| Duration | `2m 30s` format |
| Tokens | `1,234` |
| Started | Relative time `2 hours ago` |
| Actions | `View` / `Replay` buttons |

### Hook Cards — Color Coding
| Hook Type | Color | Icon |
|-----------|-------|------|
| `session_start` | Blue | 🟢 |
| `llm_prompt` | Purple | 💬 |
| `llm_response` | Indigo | 🤖 |
| `tool_call` | Orange | 🔧 |
| `tool_result` | Green/Red | ✅/❌ |
| `session_end` | Gray | 🔴 |

---

## 5. State Management

```typescript
// TanStack Query keys
const sessionKeys = {
  list: (f: SessionFilters) => ['observe', 'sessions', f],
  detail: (id: string)      => ['observe', 'sessions', id],
};

// Realtime: WebSocket channel 'pipeline.progress' cho active sessions
// → queryClient.setQueryData(['observe','sessions', filters], updater)
```

---

## 6. Acceptance Criteria (Frontend)

- [ ] Sessions list tải trong `< 2s` với pagination 20 items/page
- [ ] Live sessions hiển thị pulsing badge và auto-refresh mỗi 5s
- [ ] HooksTimeline hiển thị đúng thứ tự theo `sequence` field
- [ ] Replay SSE streaming — hooks xuất hiện theo thời gian thực
- [ ] JSONL import: drag-and-drop + progress indicator
- [ ] Hook payload có thể expand/collapse (collapsible JSON)
- [ ] Mobile-responsive: table scrolls horizontally trên nhỏ hơn 768px
