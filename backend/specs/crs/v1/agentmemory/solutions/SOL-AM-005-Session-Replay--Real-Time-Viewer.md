# SOL-AM-005 — Solution: Session Replay & Real-Time Viewer

| Field | Value |
|---|---|
| **Solution ID** | SOL-AM-005 |
| **CR** | CR-AM-005 |
| **TDD ref** | [12-agentmemory-services.md](../../../tdd/architecture/12-agentmemory-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |
| **Component** | `services/observe-service` |

---

## 1. Giải pháp

See SOL-INTEL-005 (Session Replay) for full implementation.

Additional AgentMemory features:
- Real-time SSE viewer (already exists: `GET /v1/observe/stream`)
- WebSocket support for UI real-time update
- Replay speed control (1x, 2x, 5x)

```go
// GET /v1/observe/sessions/{id}/replay?speed=2
func (h *Handler) ReplaySession(w http.ResponseWriter, r *http.Request) {
    speed, _ := strconv.ParseFloat(r.URL.Query().Get("speed"), 64)
    if speed == 0 { speed = 1.0 }
    
    ch, _ := h.replayUC.ReplaySession(r.Context(), sessionID, 0)
    for obs := range ch {
        fmt.Fprintf(w, "data: %s\n\n", mustJSON(obs))
        w.(http.Flusher).Flush()
        time.Sleep(time.Duration(float64(10*time.Millisecond) / speed))
    }
}
```

## 2. Acceptance Criteria

- [ ] Replay at 1x, 2x, 5x speed
- [ ] SSE stream for real-time viewer
- [ ] JSONL export (see SOL-INTEL-005)

