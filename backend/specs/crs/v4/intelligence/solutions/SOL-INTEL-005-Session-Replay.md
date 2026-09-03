# SOL-INTEL-005 — Solution: Session Replay & JSONL Import

| Field | Value |
|---|---|
| **Solution ID** | SOL-INTEL-005 |
| **CR** | [CR-INTEL-005](../../../../docs/crs/v4/intelligence/CR-INTEL-005-Session-Replay.md) |
| **TDD ref** | [12-agentmemory-services.md](../../../tdd/architecture/12-agentmemory-services.md) §observe-service |
| **Status** | Open |
| **Priority** | 🟡 High |

---

## 1. Giải pháp

Session Replay = stream session observations back in order (SSE), allowing:
1. Live replay in Console UI
2. JSONL export for batch import
3. Retrospective analysis

### 1.1 `services/observe-service/internal/usecase/replay.go` [NEW]

```go
type ReplayUseCase struct {
    obsRepo port.ObservationRepository
}

func (u *ReplayUseCase) ReplaySession(ctx context.Context, sessionID string, fromIndex int) (<-chan *Observation, error) {
    observations, err := u.obsRepo.GetSessionObservations(ctx, sessionID, fromIndex)
    if err != nil { return nil, err }

    ch := make(chan *Observation, 100)
    go func() {
        defer close(ch)
        for _, obs := range observations {
            select {
            case <-ctx.Done(): return
            case ch <- obs:
                // Add playback delay (simulate original timing if requested)
                time.Sleep(10 * time.Millisecond)
            }
        }
    }()
    return ch, nil
}

// ExportJSONL — export session as JSONL for offline analysis
func (u *ReplayUseCase) ExportJSONL(ctx context.Context, sessionID string, w io.Writer) error {
    observations, _ := u.obsRepo.GetSessionObservations(ctx, sessionID, 0)
    enc := json.NewEncoder(w)
    for _, obs := range observations {
        if err := enc.Encode(obs); err != nil { return err }
    }
    return nil
}
```

### 1.2 HTTP SSE endpoint

```go
// GET /v1/observe/sessions/{id}/replay
func (h *AgentMemoryHandler) ReplaySession(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "id")
    fromIndex, _ := strconv.Atoi(r.URL.Query().Get("from"))

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")

    ch, _ := h.replayUC.ReplaySession(r.Context(), sessionID, fromIndex)
    for obs := range ch {
        data, _ := json.Marshal(obs)
        fmt.Fprintf(w, "data: %s\n\n", data)
        w.(http.Flusher).Flush()
    }
}

// GET /v1/observe/sessions/{id}/export
func (h *AgentMemoryHandler) ExportSession(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "id")
    w.Header().Set("Content-Type", "application/x-ndjson")
    w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.jsonl"`, sessionID))
    h.replayUC.ExportJSONL(r.Context(), sessionID, w)
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/observe-service/internal/usecase/replay.go` | NEW |
| `gateway/adapter/handler/agentmemory_handler.go` | MODIFY — add replay + export |
| `gateway/adapter/handler/router.go` | MODIFY — register replay route |

---

## 3. Acceptance Criteria

- [ ] SSE replay streams observations in order with correct hook_type
- [ ] JSONL export contains all observations (no truncation)
- [ ] `from_index` parameter allows partial replay
- [ ] Replay respects tenant isolation
