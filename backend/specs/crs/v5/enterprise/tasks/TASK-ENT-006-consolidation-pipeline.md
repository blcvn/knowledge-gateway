# TASK-ENT-006 — 4-Tier Memory Consolidation Pipeline

| Field | Value |
|---|---|
| **Task ID** | TASK-ENT-006 |
| **Wave** | 2 |
| **Solution** | [SOL-ENT-002](../solutions/SOL-ENT-002-Consolidation-Pipeline.md) §1.1 |
| **Component** | `services/pipeline-service/internal/usecase/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 6h |

---

## Mục tiêu

Full 4-tier consolidation: Compression → Session Summary → Procedure Extraction → Cross-Session Insights.

---

## Công việc cụ thể

### `services/pipeline-service/internal/usecase/pipeline.go` [MODIFY]

```go
func (p *ConsolidationPipeline) RunForSession(ctx context.Context, sessionID string) error {
    observations, err := p.obsRepo.GetSessionObservations(ctx, sessionID, 0)
    if err != nil || len(observations) == 0 { return err }

    tenantID := observations[0].TenantID
    userID   := observations[0].UserID

    // ─── TIER 1: LLM Compression (group by 5min window) ─────────────────────
    windows := groupByTimeWindow(observations, 5*time.Minute)
    var blobs []CompressedBlob

    for _, window := range windows {
        blob, err := p.llm.CompressBatch(ctx, &port.CompressRequest{
            Observations: window,
            TargetReduction: 0.8, // 80% reduction target
        })
        if err != nil {
            p.circuitBreaker.RecordFailure()
            if p.circuitBreaker.IsOpen() {
                return fmt.Errorf("pipeline aborted: LLM circuit breaker open")
            }
            continue
        }
        p.circuitBreaker.RecordSuccess()
        blobs = append(blobs, *blob)
        p.memRepo.StoreBlob(ctx, tenantID, blob)
    }

    // ─── TIER 2: Session Summary ─────────────────────────────────────────────
    summary, err := p.llm.SummarizeSession(ctx, &port.SummarizeRequest{
        Blobs: blobs, SessionID: sessionID,
    })
    if err == nil {
        p.memRepo.StoreSessionSummary(ctx, sessionID, summary)
    }

    // ─── TIER 3: Procedure Extraction (every 5 sessions) ─────────────────────
    sessionCount, _ := p.obsRepo.CountUserSessions(ctx, tenantID, userID)
    if sessionCount%5 == 0 {
        recentSessions, _ := p.obsRepo.GetRecentSessionIDs(ctx, tenantID, userID, 5)
        allBlobs, _ := p.memRepo.GetBlobsForSessions(ctx, tenantID, recentSessions)
        procedures, err := p.llm.ExtractProcedures(ctx, allBlobs)
        if err == nil {
            for _, proc := range procedures {
                p.procRepo.Store(ctx, tenantID, proc) // → OpenViking L1
            }
        }
    }

    // ─── TIER 4: Cross-Session Insights (every 20 sessions) ──────────────────
    if sessionCount%20 == 0 {
        recentSessions, _ := p.obsRepo.GetRecentSessionIDs(ctx, tenantID, userID, 20)
        allBlobs, _ := p.memRepo.GetBlobsForSessions(ctx, tenantID, recentSessions)
        insights, err := p.llm.ExtractCrossSessionInsights(ctx, allBlobs)
        if err == nil {
            p.memRepo.StoreInsights(ctx, tenantID, userID, insights)
        }
    }

    p.pub.Publish(ctx, "memory.consolidation.completed", map[string]any{
        "session_id": sessionID, "blobs": len(blobs),
        "tier2": summary != nil,
        "tier3": sessionCount%5 == 0,
        "tier4": sessionCount%20 == 0,
    })
    return nil
}

// NATS subscriber: auto-trigger on session complete
func (p *PipelineService) StartSubscriber(ctx context.Context) {
    p.nats.Subscribe("agent.session.complete", func(msg *nats.Msg) {
        var event struct{ SessionID string `json:"session_id"` }
        json.Unmarshal(msg.Data, &event)
        go func() {
            if err := p.pipeline.RunForSession(context.Background(), event.SessionID); err != nil {
                slog.Error("consolidation failed", "session_id", event.SessionID, "error", err)
            }
        }()
    })
}
```

---

## Acceptance Criteria

- [ ] Tier 1: 70-90% compression of raw observations
- [ ] Circuit breaker: abort after 3 consecutive LLM failures
- [ ] Tier 2: session summary with attempted/succeeded/failed
- [ ] Tier 3: triggered every 5 sessions, procedures → OpenViking
- [ ] Tier 4: triggered every 20 sessions, insights stored
- [ ] NATS event published after completion
- [ ] Triggered within 1s of "agent.session.complete" event

## Files

```
services/pipeline-service/internal/usecase/pipeline.go         [MODIFY]
services/pipeline-service/internal/port/llm.go                 [MODIFY — compress, summarize, extract]
deployment/dev/migrations/0044_session_blobs.sql                [NEW]
```
