# UI Solution: UI-SOL-INTEL-005 — Session Replay (Intelligence Layer)

**Solution ID:** UI-SOL-INTEL-005  
**CR References:** [CR-INTEL-005](../../../../docs/crs/v4/intelligence/CR-INTEL-005-Session-Replay.md)  
**Backend Solution:** [SOL-INTEL-005](../../../../backend/specs/crs/v4/intelligence/solutions/SOL-INTEL-005-Session-Replay.md)  
**Feature:** Session Replay — Intelligence-Level Timeline, JSONL Import  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/sessions/replay/`

---

## 1. Mục Đích

Session Replay UI tập trung vào góc nhìn **Intelligence Layer**:
- Timeline hiển thị hook events theo chronological order
- SSE streaming replay với speed control
- JSONL import từ Claude Code
- Thống kê session: duration, tokens, tool calls, LLM calls
- Search trong hooks

---

## 2. Backend API Contract

```http
GET /v1/observe/sessions?user_id=u_123&limit=10  → ObserveSession[]
GET /v1/observe/sessions/{id}                    → SessionDetail + hooks[]
GET /v1/observe/sessions/{id}/replay              → SSE stream
POST /v1/observe/sessions/import                  → ImportResult
```

### TypeScript Types

```typescript
interface HookEvent {
  hook_type: 'session_start' | 'session_end' | 'llm_prompt' |
             'llm_response' | 'tool_call' | 'tool_result' | 'observe_hook';
  timestamp:  string;
  sequence:   number;
  payload:    Record<string, unknown>;
  latency_ms?: number;
}

interface SessionSummary {
  session_id:   string;
  agent_id:     string;
  status:       'active' | 'completed' | 'abandoned';
  hook_count:   number;
  total_tokens: number;
  duration_ms:  number;
  llm_calls:    number;
  tool_calls:   number;
}
```

---

## 3. Components

### 3.1 Intelligence-focused Session List

```
SessionReplayListPage
├── SessionFilters          ← agent_id, date range, status
├── SessionCards (grid)
│   └── SessionCard
│       ├── AgentBadge
│       ├── StatusBadge
│       ├── Stats
│       │   ├── Hooks: 47
│       │   ├── LLM Calls: 12
│       │   └── Tool Calls: 23
│       ├── Duration        ← "1m 23s"
│       └── ReplayButton    ← opens replay viewer
└── ImportJSONLButton       ← drag-drop import
```

### 3.2 Replay Viewer (Intelligence Mode)

```
SessionReplayViewer
├── SummaryHeader (sticky)
│   ├── SessionId + Agent
│   ├── Duration + Tokens
│   ├── LLM Calls count
│   └── Tool Calls count
├── SplitLayout
│   ├── Timeline (left 40%)
│   │   ├── ScrubBar        ← seek to any hook
│   │   ├── PlayControls    ← ▶ | ⏸ | ⟳ | Speed (1x/2x/4x)
│   │   ├── SearchBar       ← "search hooks" (filter by type/content)
│   │   └── HookList        ← scrollable
│   │       └── HookItem    ← type badge + timestamp
│   └── HookDetail (right 60%)
│       ├── HookTypeBadge   ← large colored badge
│       ├── TimestampPrecise
│       └── PayloadViewer   ← JSON syntax-highlighted
├── LLMCallSummary (collapsible)
│   └── LLMCallRow          ← prompt tokens | response tokens | latency
└── ToolCallSummary (collapsible)
    └── ToolCallRow         ← tool name | args preview | success/fail
```

---

## 4. Hook Type Display

```typescript
const HOOK_CONFIG: Record<string, { icon: string; color: string; label: string }> = {
  session_start:   { icon: '🟢', color: 'bg-blue-100',   label: 'Session Start' },
  llm_prompt:      { icon: '💬', color: 'bg-purple-100', label: 'LLM Prompt' },
  llm_response:    { icon: '🤖', color: 'bg-indigo-100', label: 'LLM Response' },
  tool_call:       { icon: '🔧', color: 'bg-orange-100', label: 'Tool Call' },
  tool_result:     { icon: '✅', color: 'bg-green-100',  label: 'Tool Result' },
  observe_hook:    { icon: '📍', color: 'bg-gray-100',   label: 'Observe Hook' },
  session_end:     { icon: '🔴', color: 'bg-red-50',     label: 'Session End' },
};
```

---

## 5. Acceptance Criteria (Frontend)

- [ ] Session list shows LLM calls and tool calls counts
- [ ] Hook search: filter by hook_type or content keyword
- [ ] Replay scrub bar: seek to specific hook by index
- [ ] Speed control: 1x/2x/4x
- [ ] LLM Call Summary: tokens input/output per call
- [ ] Tool Call Summary: name, success/fail result
- [ ] JSONL import: drag-drop `.jsonl` file
- [ ] Import progress indicator (hook count)
