---
id: FEAT-014
title: Sessions & Conversations API
service: vnp-gateway
version: 1.0.0
status: Draft
priority: P1
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
linked_ux: "ux_spec.md §6.7 Sessions & Conversations"
---

## Mục Tiêu

REST APIs cho Sessions & Conversations screen — session timeline, live conversation viewer, user memory summary, session diff, working memory inspector.

## Bối Cảnh Nghiệp Vụ

Sessions (UX §6.7) cần:
1. **Session Timeline** — replay full AI interaction
2. **Live Conversation Viewer** — realtime message streaming
3. **User Memory Summary** — Memobase-powered profile summary per session
4. **Session Diff** — compare memory before/after conversation
5. **Working Memory Inspector** — structured document view, 2-phase commit status

## Scope

### In Scope
- `GET /v1/console/sessions` — List sessions (paginated, filterable)
- `GET /v1/console/sessions/{id}` — Session detail with messages
- `GET /v1/console/sessions/{id}/timeline` — Session replay timeline
- `GET /v1/console/sessions/{id}/diff` — Memory diff (before vs after)
- `GET /v1/console/sessions/{id}/working-memory` — Working memory state
- `GET /v1/console/sessions/{id}/user-summary` — User memory profile summary
- `GET /v1/console/sessions/live` — List active live sessions

### Out of Scope
- Session creation/modification (engine responsibility)
- Live message streaming (covered by FEAT-012 WebSocket)

## Thiết Kế Kỹ Thuật

### API Contract

#### GET `/v1/console/sessions`

**Query params:** `?user_id=xxx&engine=zep|ov&status=active|completed&from=xxx&to=xxx&cursor=xxx&limit=50`

**Response (200):**
```json
{
  "sessions": [
    {
      "id": "sess_abc123",
      "user_id": "user_123",
      "engine": "zep",
      "status": "completed",
      "message_count": 24,
      "started_at": "2026-05-13T10:00:00Z",
      "ended_at": "2026-05-13T10:30:00Z",
      "summary": "Discussed knowledge graph architecture..."
    }
  ],
  "next_cursor": "...",
  "total": 342
}
```

#### GET `/v1/console/sessions/{id}/diff`

**Response (200):**
```json
{
  "before": {
    "total_memories": 120,
    "by_engine": { "cognee": 40, "graphiti": 30, "zep": 50 }
  },
  "after": {
    "total_memories": 128,
    "by_engine": { "cognee": 42, "graphiti": 33, "zep": 53 }
  },
  "delta": {
    "memories_added": 10,
    "memories_updated": 3,
    "memories_forgotten": 2,
    "new_entities": ["VNP Memory", "Knowledge Graph"]
  }
}
```

#### GET `/v1/console/sessions/{id}/working-memory`

**Response (200):**
```json
{
  "session_id": "sess_abc123",
  "engine": "openviking",
  "document": {
    "title": "Session Working Document",
    "state": "active",
    "goals": ["Understand KG architecture"],
    "facts": ["User prefers TypeScript", "Building vnp-memory platform"],
    "errors": [],
    "token_count": 1200
  },
  "commit_status": {
    "phase": "extract",
    "archived": true,
    "extracted": false,
    "long_term_progress": 0.6
  }
}
```

### Internal Architecture
- **Handler:** `adapter/http/session_handler.go`
- **Proxy to:** `zep-core` (Zep sessions), `ov-session` (OV sessions), `memobase-context` (user summary)
- **Usecase:** `usecase/session.go` — merge sessions from Zep + OpenViking
- Session diff: compare event snapshots from `vnp-event` service

## Acceptance Criteria
- [ ] AC-1: Session list returns paginated, filterable sessions across Zep + OpenViking
- [ ] AC-2: Session detail shows full message timeline with engine badges
- [ ] AC-3: Session diff shows memory delta (added, updated, forgotten)
- [ ] AC-4: Working memory inspector shows document state, commit status
- [ ] AC-5: User summary returns Memobase profile with freshness indicator
- [ ] AC-6: Live sessions endpoint returns currently active sessions
- [ ] AC-7: All endpoints require auth; results scoped to tenant

## Test Requirements
- Unit tests: Session merge logic, diff computation
- Integration tests: Multi-engine session aggregation with mocks
- Minimum coverage: 80%
