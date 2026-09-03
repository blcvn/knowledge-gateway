# Change Request: CR-CONSOLE-003 — Sessions Explorer Backend APIs

**CR ID:** CR-CONSOLE-003
**Component:** `backend/gateway`, `backend/services/observe-service`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Console
**Feature:** [F21](../../../features/21-sessions-explorer/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-07 | Agent Developer | Không track được agent session state |

---

## 2. APIs

### `GET /v1/console/sessions?status=live&limit=20`

```json
{
  "sessions": [
    {
      "id": "sess-abc",
      "agent_id": "agent-xyz",
      "status": "live",
      "hook_count": 47,
      "started_at": "2026-09-03T10:00:00Z",
      "last_event_at": "2026-09-03T10:05:23Z"
    }
  ],
  "total": 142
}
```

### `GET /v1/console/sessions/{id}`

Full session detail: agent info, status, hook count, memory summary.

### `GET /v1/console/sessions/{id}/timeline`

```json
{
  "session_id": "sess-abc",
  "events": [
    {
      "index": 0, "hook_type": "agent_start",
      "timestamp": "...", "duration_ms": 0
    },
    {
      "index": 1, "hook_type": "llm_call",
      "timestamp": "...", "duration_ms": 423,
      "payload_summary": "Prompt: 1200 tokens, Response: 342 tokens"
    }
  ]
}
```

### `GET /v1/console/sessions/{id}/diff`

```json
{
  "memories_added": [
    {"id": "mem-new", "type": "episodic", "content": "..."}
  ],
  "memories_updated": [...],
  "memories_removed": []
}
```

### `GET /v1/console/sessions/{id}/working-memory`

Current L0 working memory (active sketches) for the session.

### `GET /v1/console/sessions/{id}/user-summary`

Compressed user context for this session.

---

## 3. Acceptance Criteria

- [ ] List sessions with live/completed/failed filter
- [ ] Timeline events ordered by index
- [ ] Diff shows before/after memory state
- [ ] Working memory returns current sketches
- [ ] `/sessions/live` SSE stream for real-time session list updates
