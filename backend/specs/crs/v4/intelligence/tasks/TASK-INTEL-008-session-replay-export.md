# TASK-INTEL-008 — Session Replay UseCase: ordered SSE stream + JSONL export

| Field | Value |
|---|---|
| **Task ID** | TASK-INTEL-008 |
| **Wave** | 3 |
| **Solution** | [SOL-INTEL-005](../solutions/SOL-INTEL-005-Session-Replay.md) §1.1 |
| **Component** | `services/observe-service/internal/usecase/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 4h |

---

## Mục tiêu

Session Replay UseCase: ordered SSE stream + JSONL export

---

## Công việc cụ thể

### `services/observe-service/internal/usecase/replay.go` [NEW]

```go
type ReplayUseCase struct {
    obsRepo port.ObservationRepository
}

// ReplaySession — stream observations in order as SSE channel
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
                time.Sleep(10 * time.Millisecond) // playback pacing
            }
        }
    }()
    return ch, nil
}

// ExportJSONL — export all observations as newline-delimited JSON
func (u *ReplayUseCase) ExportJSONL(ctx context.Context, sessionID string, w io.Writer) error {
    observations, err := u.obsRepo.GetSessionObservations(ctx, sessionID, 0)
    if err != nil { return err }

    enc := json.NewEncoder(w)
    for _, obs := range observations {
        if err := enc.Encode(obs); err != nil { return err }
    }
    return nil
}

// ReplayWithSpeed — speed multiplier: 1.0=realtime, 2.0=2x faster
func (u *ReplayUseCase) ReplayWithSpeed(ctx context.Context, sessionID string, speed float64) (<-chan *Observation, error) {
    if speed <= 0 { speed = 1.0 }
    // Same as ReplaySession but delay = 10ms/speed
    ...
}
```

---

## Acceptance Criteria

- [ ] ReplaySession streams in order (fromIndex respected)
- [ ] ExportJSONL produces valid NDJSON
- [ ] Context cancellation stops stream
- [ ] ReplayWithSpeed adjusts pacing correctly
- [ ] Unit tests: replay order, JSONL format, cancellation

## Files

```
services/observe-service/internal/usecase/replay.go       [NEW]
services/observe-service/internal/usecase/replay_test.go  [NEW]
```

**Trạng thái:** ✅ Implemented

---

**Ghi chú audit:** observe-service/internal/usecase/replay.go: ReplayUseCase.ListSessions+LoadTimeline; replay/replay.go: Playback() + timeline.go: BuildTimeline+Filter
