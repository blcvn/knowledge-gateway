# Feature 21 — Sessions Explorer

> **Loại:** Console UI | **Priority:** P1 | **Status:** Implemented (UI)

## Mô tả

Sessions Explorer cho phép monitor và inspect agent sessions — xem working memory, session timeline, memory diff trước/sau session, và live sessions đang chạy.

---

## Business Logic

### Session List & Live View

- List all sessions với status (active/completed/abandoned)
- **Live sessions**: Highlight đang chạy, real-time observation count
- Filter: agent_id, status, time range

### Session Detail

Khi inspect một session:
- **Timeline**: Sequence of events trong session (tool calls, LLM calls, decisions)
- **Working Memory**: Structured document {title, state, goals, facts, errors}
- **Diff View**: So sánh memory state trước và sau session
- **User Summary**: AI-generated summary về session

### Session Timeline

Timeline view cho một session:
- Chronological list of events
- Event types: llm_call, tool_call, memory_read, memory_write, error, checkpoint
- Latency per event
- Click event → full detail (prompt, response, error message...)

### Memory Diff

Hiển thị what changed:
- Memories added during session
- Memories updated
- Memories deleted/forgotten
- Profile changes (Memobase)

---

## Dataflow

```
Console UI (Sessions Explorer)
        │
        ├── GET /v1/console/sessions
        │         └── All sessions (filterable)
        │
        ├── GET /v1/console/sessions/live
        │         └── Currently active sessions (real-time)
        │
        ├── GET /v1/console/sessions/{id}
        │         └── Session detail
        │
        ├── GET /v1/console/sessions/{id}/timeline
        │         └── Chronological event list
        │
        ├── GET /v1/console/sessions/{id}/diff
        │         └── Memory state diff (before/after)
        │
        ├── GET /v1/console/sessions/{id}/working-memory
        │         └── Current working memory document
        │
        └── GET /v1/console/sessions/{id}/user-summary
                  └── AI-generated session summary
```

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/v1/console/sessions` | List sessions |
| `GET` | `/v1/console/sessions/live` | Live sessions |
| `GET` | `/v1/console/sessions/{id}` | Session detail |
| `GET` | `/v1/console/sessions/{id}/timeline` | Event timeline |
| `GET` | `/v1/console/sessions/{id}/diff` | Memory diff |
| `GET` | `/v1/console/sessions/{id}/working-memory` | Working memory |
| `GET` | `/v1/console/sessions/{id}/user-summary` | User summary |
