# SOL-ENT-002 — Solution: Memory Consolidation Pipeline (4-Tier)

| Field | Value |
|---|---|
| **Solution ID** | SOL-ENT-002 |
| **CR** | [CR-ENT-002](../../../../docs/crs/v5/enterprise/CR-ENT-002-Consolidation-Pipeline.md) |
| **TDD ref** | [12-agentmemory-services.md](../../../tdd/architecture/12-agentmemory-services.md) §pipeline-service |
| **Status** | ✅ Implemented |
| **Priority** | 🟡 High |

---

## 1. Giải pháp

`pipeline-service` 4-tier consolidation triggered khi session complete.

### 1.1 `services/pipeline-service/internal/usecase/pipeline.go` [MODIFY]

```go
type ConsolidationPipeline struct {
    obsRepo    port.ObservationRepository
    memRepo    port.MemoryRepository
    procRepo   port.ProceduralRepository   // OpenViking L1
    llm        port.LLMClient
    nats       port.EventPublisher
}

func (p *ConsolidationPipeline) RunForSession(ctx context.Context, sessionID string) error {
    observations, _ := p.obsRepo.GetSessionObservations(ctx, sessionID, 0)
    if len(observations) == 0 { return nil }

    // TIER 1: LLM Compression (group by 5min window)
    windows := groupByTimeWindow(observations, 5*time.Minute)
    var blobs []CompressedBlob
    for _, window := range windows {
        blob, _ := p.llm.CompressBatch(ctx, window)  // 70-90% reduction
        blobs = append(blobs, blob)
        p.memRepo.StoreBlob(ctx, blob)
    }

    // TIER 2: Session Summary
    summary, _ := p.llm.SummarizeSession(ctx, blobs)
    // Fields: attempted, succeeded, failed, decisions, entities
    p.memRepo.StoreSessionSummary(ctx, sessionID, summary)

    // TIER 3: Procedure Extraction (trigger: N sessions threshold)
    sessionCount, _ := p.obsRepo.CountUserSessions(ctx, summary.TenantID, summary.UserID)
    if sessionCount % 5 == 0 {
        procedures, _ := p.llm.ExtractProcedures(ctx, p.getRecentSessions(ctx, summary, 5))
        for _, proc := range procedures {
            p.procRepo.Store(ctx, proc)  // → OpenViking L1
        }
    }

    // TIER 4: Cross-Session Insights (trigger: 20 sessions or weekly)
    if sessionCount % 20 == 0 {
        insights, _ := p.llm.ExtractCrossSessionInsights(ctx,
            p.getRecentSessions(ctx, summary, 20))
        p.memRepo.StoreInsights(ctx, insights)
    }

    p.nats.Publish(ctx, "memory.consolidation.completed", map[string]string{
        "session_id": sessionID, "blobs": fmt.Sprintf("%d", len(blobs)),
    })
    return nil
}
```

### 1.2 NATS trigger

```go
// Subscribe to "agent.session.complete"
func (p *PipelineService) StartSubscriber(ctx context.Context) {
    p.nats.Subscribe("agent.session.complete", func(msg *nats.Msg) {
        var event SessionCompleteEvent
        json.Unmarshal(msg.Data, &event)
        go p.pipeline.RunForSession(context.Background(), event.SessionID)
    })
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/pipeline-service/internal/usecase/pipeline.go` | MODIFY — full 4-tier |
| `services/pipeline-service/internal/port/llm.go` | MODIFY — add CompressBatch, SummarizeSession |
| `deployment/dev/migrations/0XX_session_blobs.sql` | NEW |

---

## 3. Acceptance Criteria

- [ ] Tier 1: 70-90% compression of raw hooks
- [ ] Tier 2: Session summary với attempted/succeeded/failed/decisions/entities
- [ ] Tier 3: Procedures extracted every 5 sessions
- [ ] Tier 4: Cross-session insights every 20 sessions
- [ ] Circuit breaker: stop pipeline if LLM fails 3× consecutive
- [ ] NATS event published after completion

---

**Ghi chú audit:** pipeline-service: PipelineUseCase + 4 templates + Redis consumer worker + ConsolidationPipeline 4-tier in memory-service
