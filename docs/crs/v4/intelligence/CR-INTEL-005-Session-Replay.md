# Change Request: CR-INTEL-005 — Session Replay & JSONL Import

**CR ID:** CR-INTEL-005
**Component:** `backend/services/observe-service`, `backend/services/obs-service`
**Priority:** 🟡 High
**Status:** Open
**Version:** v4 / Intelligence Layer
**Solution:** [S7 — Agent Observability](../../../bussiness/solutions/S7-agent-observability.md)
**Features:** [F08](../../../features/08-agent-observe-hooks/README.md), [F26](../../../features/26-session-replay/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-07 | AI Agent Developer | Debug agent issue mất 2-4 giờ vì không có session history |
| PP-P4-04 | Enterprise Architect | Không audit được agent decisions |

**Before:** Debug = grep logs + reconstruct mental model.
**After:** Session Replay với timeline scrubbing → debug trong 20 phút.

---

## 2. Replay Features

```
Timeline view: chronological list of hooks
  10:00:01 session_start     {agent: "claude-code"}
  10:00:05 llm_prompt        {prompt: "Implement auth...", tokens: 2048}
  10:00:08 tool_call         {tool: "write_file", path: "auth.go"}
  10:00:09 tool_result       {success: true}
  10:00:15 llm_response      {content: "Done. I wrote..."}
  10:01:00 session_end       {total_hooks: 47, duration: 59s}

Scrubbing: jump to any timestamp in session
JSONL import: paste Claude Code .jsonl → auto-import as session
```

---

## 3. API Contract

```http
# List sessions
GET /v1/observe/sessions?user_id=u_123&limit=10

# Get session detail + hooks
GET /v1/observe/sessions/{session_id}
→ {
    "session_id": "s_456",
    "agent_id": "claude-code",
    "started_at": "...",
    "status": "completed",
    "hooks": [ ... 47 hook objects ... ]
  }

# Replay (stream hooks in order via SSE)
GET /v1/observe/sessions/{session_id}/replay
→ SSE stream: data: {hook_type: "llm_prompt", timestamp: "...", payload: {...}}

# Import JSONL (Claude Code format)
POST /v1/observe/sessions/import
Content-Type: multipart/form-data
file=@claude_session.jsonl
→ { "session_id": "s_new", "hooks_imported": 47 }
```

---

## 4. Acceptance Criteria

- [ ] Timeline view: chronological hooks với timestamps
- [ ] SSE replay stream (real-time scrubbing)
- [ ] JSONL import: Claude Code format supported
- [ ] Session status: active / completed / abandoned
- [ ] Search trong session hooks (`GET /v1/observe/sessions/{id}?search=tool_call`)
- [ ] Session duration, hook count, token usage summary
