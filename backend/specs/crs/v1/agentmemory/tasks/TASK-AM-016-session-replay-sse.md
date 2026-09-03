# TASK-AM-016 — Session Replay + SSE (CR-AM-005)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-016 |
| **Wave** | 3 (Orchestration) |
| **Component** | `services/observe-service/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-005 (Session Replay) |
| **Priority** | Medium |
| **Depends On** | TASK-AM-003 |
| **Estimated** | 4h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** observe-service SSE streaming implemented  
---

## Context

Session Replay cho phép playback lại timeline của observations trong một session. Bao gồm filtering theo hook_type, time range, và simulated playback với speed multiplier.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/observe-service/internal/replay/replay.go` |
| CREATE | `services/observe-service/internal/replay/timeline.go` |
| CREATE | `services/observe-service/internal/usecase/replay.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

### `internal/replay/timeline.go`

```go
package replay

import (
    "sort"
    "time"

    "github.com/vnp-memory/services/observe-service/internal/domain"
)

type TimelineEntry struct {
    Index      int
    Timestamp  time.Time
    HookType   string
    ToolName   string
    ObsType    string
    Title      string
    Facts      []string
    Duration   time.Duration  // time since previous event
    AgentID    string
}

type Timeline struct {
    SessionID   string
    Project     string
    TotalEvents int
    Duration    time.Duration  // total session duration
    Entries     []TimelineEntry
}

// BuildTimeline creates a sorted, annotated timeline from observations
func BuildTimeline(sessionID, project string, obs []domain.CompressedObservation) Timeline {
    if len(obs) == 0 { return Timeline{SessionID: sessionID, Project: project} }

    sort.Slice(obs, func(i, j int) bool { return obs[i].Timestamp.Before(obs[j].Timestamp) })

    entries := make([]TimelineEntry, len(obs))
    var prevTime time.Time
    for i, o := range obs {
        var dur time.Duration
        if i > 0 { dur = o.Timestamp.Sub(prevTime) }
        entries[i] = TimelineEntry{
            Index:    i,
            Timestamp: o.Timestamp,
            HookType: o.ObsType,
            ObsType:  o.ObsType,
            Title:    o.Title,
            Facts:    o.Facts,
            Duration: dur,
            AgentID:  o.AgentID,
        }
        prevTime = o.Timestamp
    }

    totalDur := obs[len(obs)-1].Timestamp.Sub(obs[0].Timestamp)
    return Timeline{
        SessionID:   sessionID,
        Project:     project,
        TotalEvents: len(obs),
        Duration:    totalDur,
        Entries:     entries,
    }
}

// Filter returns subset matching filters
func (t Timeline) Filter(hookTypes []string, fromIdx, toIdx int) Timeline {
    if len(hookTypes) == 0 && fromIdx == 0 && toIdx == 0 { return t }
    hookSet := make(map[string]bool, len(hookTypes))
    for _, h := range hookTypes { hookSet[h] = true }

    var filtered []TimelineEntry
    for _, e := range t.Entries {
        if fromIdx > 0 && e.Index < fromIdx { continue }
        if toIdx > 0 && e.Index > toIdx { continue }
        if len(hookSet) > 0 && !hookSet[e.HookType] { continue }
        filtered = append(filtered, e)
    }
    t.Entries = filtered
    t.TotalEvents = len(filtered)
    return t
}
```

### `internal/replay/replay.go`

```go
package replay

import (
    "context"
    "time"
)

type PlaybackConfig struct {
    Speed      float64  // 1.0 = realtime, 2.0 = 2x speed
    StartIndex int
}

type PlaybackEvent struct {
    Entry     TimelineEntry
    Progress  float64  // 0.0 - 1.0
    Remaining int      // events remaining
}

// Playback streams timeline events with time delays
// Returns a channel of PlaybackEvents; caller manages context cancellation
func Playback(ctx context.Context, timeline Timeline, cfg PlaybackConfig) <-chan PlaybackEvent {
    ch := make(chan PlaybackEvent, 10)

    go func() {
        defer close(ch)
        entries := timeline.Entries
        if cfg.StartIndex > 0 {
            entries = entries[cfg.StartIndex:]
        }
        speed := cfg.Speed
        if speed <= 0 { speed = 1.0 }

        for i, entry := range entries {
            select {
            case <-ctx.Done(): return
            default:
            }

            // Simulate time delay between events
            if i > 0 && entry.Duration > 0 {
                delay := time.Duration(float64(entry.Duration) / speed)
                if delay > 30*time.Second { delay = 30*time.Second }  // cap delay
                timer := time.NewTimer(delay)
                select {
                case <-timer.C:
                case <-ctx.Done(): timer.Stop(); return
                }
            }

            ch <- PlaybackEvent{
                Entry:    entry,
                Progress: float64(i+1) / float64(len(entries)),
                Remaining: len(entries) - i - 1,
            }
        }
    }()

    return ch
}
```

### `internal/usecase/replay.go`

```go
package usecase

import (
    "context"

    "github.com/vnp-memory/services/observe-service/internal/domain"
    "github.com/vnp-memory/services/observe-service/internal/replay"
    "github.com/vnp-memory/services/observe-service/internal/usecase/port"
)

type ReplayUseCase struct {
    sessionRepo port.ISessionRepo
    obsRepo     port.IObservationRepo
}

type ListReplaySessionsRequest struct {
    TenantID string
    Project  string
    Limit    int
    Offset   int
}

type ListReplaySessionsResponse struct {
    Sessions []domain.Session
    Total    int
}

func (uc *ReplayUseCase) ListSessions(ctx context.Context, req ListReplaySessionsRequest) (*ListReplaySessionsResponse, error) {
    sessions, err := uc.sessionRepo.List(ctx, req.TenantID, "completed", req.Project, req.Limit, req.Offset)
    if err != nil { return nil, err }
    return &ListReplaySessionsResponse{Sessions: sessions, Total: len(sessions)}, nil
}

type LoadReplayRequest struct {
    SessionID  string
    TenantID   string
    HookTypes  []string
    FromIndex  int
    ToIndex    int
}

type LoadReplayResponse struct {
    Timeline replay.Timeline
}

func (uc *ReplayUseCase) LoadTimeline(ctx context.Context, req LoadReplayRequest) (*LoadReplayResponse, error) {
    session, err := uc.sessionRepo.GetByID(ctx, req.SessionID)
    if err != nil { return nil, err }

    obs, err := uc.obsRepo.ListCompressed(ctx, req.SessionID, 500, 0)
    if err != nil { return nil, err }

    // Cast to domain.CompressedObservation
    compObs := make([]domain.CompressedObservation, len(obs))
    for i, o := range obs { compObs[i] = o.(domain.CompressedObservation) }

    timeline := replay.BuildTimeline(req.SessionID, session.Project, compObs)
    if len(req.HookTypes) > 0 || req.FromIndex > 0 || req.ToIndex > 0 {
        timeline = timeline.Filter(req.HookTypes, req.FromIndex, req.ToIndex)
    }

    return &LoadReplayResponse{Timeline: timeline}, nil
}
```

### MODIFY `gateway/router.go` — Replay routes

```go
// Session Replay
r.Get("/v1/observe/sessions",            h.ForwardTo("am-observe", "ObserveService/ListSessions"))
r.Get("/v1/observe/sessions/{id}",       h.ForwardTo("am-observe", "ObserveService/GetSession"))
r.Delete("/v1/observe/sessions/{id}",    h.ForwardTo("am-observe", "ObserveService/DeleteSession"))
r.Get("/v1/observe/sessions/{id}/observations", h.ForwardTo("am-observe", "ObserveService/GetObservations"))
r.Get("/v1/observe/replay/sessions",     h.ForwardTo("am-observe", "ObserveService/ListReplaySessions"))
r.Get("/v1/observe/replay/{id}/timeline", h.ForwardTo("am-observe", "ObserveService/LoadTimeline"))
r.Get("/v1/observe/replay/{id}/playback", h.ForwardTo("am-observe", "ObserveService/StartPlayback"))  // SSE stream
```

---

## Verification

```bash
cd services/observe-service
go test ./internal/replay/... -v

# Manual verification:
# 1. Create session → add 10 observations
# 2. GET /v1/observe/replay/sessions → session listed
# 3. GET /v1/observe/replay/{id}/timeline → N entries with timestamps
# 4. GET /v1/observe/replay/{id}/timeline?hook_types=post_tool_use → filtered
# 5. GET /v1/observe/replay/{id}/playback → SSE stream of events
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| `GET /v1/observe/replay/sessions` → completed sessions list | ✅ |
| `GET /v1/observe/replay/{id}/timeline` → sorted entries with durations | ✅ |
| Filter by `hook_types` → only matching events | ✅ |
| Filter by `from_index` / `to_index` → subset of timeline | ✅ |
| Playback SSE → events delivered with simulated time delay | ✅ |
| Speed=2.0 → delays halved | ✅ |
| Timeline total_duration = last_timestamp - first_timestamp | ✅ |
