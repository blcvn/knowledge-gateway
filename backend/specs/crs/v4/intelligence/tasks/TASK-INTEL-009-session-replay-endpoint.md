# TASK-INTEL-009 — Session Replay HTTP endpoints: SSE replay + JSONL export

| Field | Value |
|---|---|
| **Task ID** | TASK-INTEL-009 |
| **Wave** | 3 |
| **Solution** | [SOL-INTEL-005](../solutions/SOL-INTEL-005-Session-Replay.md) §1.2 |
| **Component** | `gateway/adapter/handler/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-INTEL-008 |
| **Estimated** | 2h |

---

## Mục tiêu

Session Replay HTTP endpoints: SSE replay + JSONL export

---

## Công việc cụ thể

### `gateway/adapter/handler/agentmemory_handler.go` [MODIFY]

```go
// GET /v1/observe/sessions/{id}/replay?speed=1.0&from=0
func (h *AgentMemoryHandler) ReplaySession(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "id")
    speed, _   := strconv.ParseFloat(r.URL.Query().Get("speed"), 64)
    fromIndex, _ := strconv.Atoi(r.URL.Query().Get("from"))
    if speed == 0 { speed = 1.0 }

    // Verify tenant owns session
    if !h.sessionUC.OwnedByTenant(r.Context(), sessionID) {
        writeError(w, 403, "forbidden", "session not found")
        return
    }

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("X-Accel-Buffering", "no")

    ch, err := h.replayUC.ReplayWithSpeed(r.Context(), sessionID, speed)
    if err != nil { writeError(w, 500, "replay_failed", err.Error()); return }

    for obs := range ch {
        data, _ := json.Marshal(obs)
        fmt.Fprintf(w, "event: observation\ndata: %s\n\n", data)
        w.(http.Flusher).Flush()
    }
    fmt.Fprintf(w, "event: done\ndata: {}\n\n")
    w.(http.Flusher).Flush()
}

// GET /v1/observe/sessions/{id}/export
func (h *AgentMemoryHandler) ExportSession(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "id")
    w.Header().Set("Content-Type", "application/x-ndjson")
    w.Header().Set("Content-Disposition",
        fmt.Sprintf(`attachment; filename="%s.jsonl"`, sessionID))
    h.replayUC.ExportJSONL(r.Context(), sessionID, w)
}
```

### Router

```go
r.Get("/v1/observe/sessions/{id}/replay", handler.ReplaySession)
r.Get("/v1/observe/sessions/{id}/export", handler.ExportSession)
```

---

## Acceptance Criteria

- [ ] SSE replay streams with `event: observation` format
- [ ] Final `event: done` sent when replay complete
- [ ] JSONL export sets correct Content-Disposition header
- [ ] Tenant check: cannot replay another tenant's session
- [ ] Speed parameter works (1x, 2x, 5x)

## Files

```
gateway/adapter/handler/agentmemory_handler.go  [MODIFY — add replay+export]
gateway/adapter/handler/router.go               [MODIFY — register routes]
```

**Trạng thái:** ✅ Implemented

---

**Ghi chú audit:** gateway: GET /v1/observe/replay/sessions, /replay/{id}/timeline, /replay/{id}/export (JSONL) routes registered; ExportSessionJSONL handler added
