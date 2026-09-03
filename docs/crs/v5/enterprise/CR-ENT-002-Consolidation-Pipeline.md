# Change Request: CR-ENT-002 — Memory Consolidation Pipeline (4-Tier)

**CR ID:** CR-ENT-002
**Component:** `backend/services/pipeline-service`
**Priority:** 🟡 High
**Status:** Open
**Version:** v5 / Enterprise & Operations
**Solution:** [S6 — Context Efficiency](../../../bussiness/solutions/S6-context-efficiency.md)
**Features:** [F12](../../../features/12-consolidation-pipeline/README.md)
**ADR:** [ADR-008](../../../adr/ADR-008-consolidation-4tier.md)
**Research:** [sleep.md](../../../research/sleep.md) — Neuroscience: sleep consolidation stages

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-09 | AI Agent Developer | Hook volume bùng nổ — 1000 hooks/session → storage cost |
| PP-P2-05 | Platform Engineer | Storage không scale — không có compression strategy |

**Storage Impact:**
- Before: 145 raw hooks per session = ~145KB
- After: 4-tier → 12 compressed summaries → 1 session summary = ~8KB (−95%)

---

## 2. 4-Tier Pipeline

```
TIER 1 — LLM Compression (mirrors NREM Stage 1-2)
  Trigger: session complete OR every 50 hooks
  Group hooks by 5-minute window
  LLM: compress batch → remove redundancy, preserve key info
  Target: 70-90% size reduction
  Circuit breaker: stop if LLM fails 3× consecutive

TIER 2 — Session Summary (mirrors NREM Stage 3-4)
  Input: all Tier 1 outputs for session
  LLM: "Trong session này agent đã làm gì?"
  Output: { attempted, succeeded, failed, decisions, entities }

TIER 3 — Procedure Extraction (mirrors REM)
  Trigger: N sessions threshold (default: 5 sessions)
  LLM: extract reusable procedures from multi-session patterns
  Output: procedural memory → OpenViking L1

TIER 4 — Cross-Session Insights (mirrors deep sleep integration)
  Trigger: weekly batch OR 20 sessions
  LLM: cross-agent patterns, lessons learned
  Output: adaptive memory → Supermemory
```

---

## 3. API Contract

```http
# Manual trigger consolidation
POST /v1/consolidate
{
  "session_id": "s_456",
  "tier": 1  // 1 | 2 | 3 | 4 | "all"
}
→ {
    "job_id": "job_789",
    "status": "queued",
    "estimated_duration_ms": 3500
  }

# Get consolidation status
GET /v1/consolidate/jobs/{job_id}
→ {
    "status": "completed",
    "tier": 1,
    "hooks_input": 145,
    "summaries_output": 12,
    "reduction_pct": 87,
    "duration_ms": 2847
  }
```

---

## 4. Thay đổi đề xuất

### 4.1 NATS-triggered pipeline

```go
// backend/services/pipeline-service/internal/usecase/consolidate.go [NEW]

// Subscribe to session.complete event
nats.Subscribe("agent.session.complete", func(event SessionCompleteEvent) {
    job := ConsolidationJob{
        SessionID: event.SessionID,
        AgentID:   event.AgentID,
        TenantID:  event.TenantID,
    }
    // Tier 1 immediately
    go u.runTier1(context.Background(), job)
})

func (u *ConsolidationUseCase) runTier1(ctx context.Context, job ConsolidationJob) {
    hooks := u.hookRepo.GetBySession(ctx, job.SessionID)
    windows := groupBy5MinWindow(hooks)
    
    for _, window := range windows {
        compressed := u.llm.Complete(ctx, compressPrompt(window))
        u.blobRepo.Save(ctx, &CompressedBlob{
            SessionID: job.SessionID,
            Content:   compressed,
            Tier:      1,
        })
    }
    
    nats.Publish("consolidation.tier1.done", job)
}
```

---

## 5. Acceptance Criteria

- [ ] Tier 1: ≥ 70% storage reduction
- [ ] Tier 2: session_summary có đầy đủ: attempted, succeeded, failed, decisions, entities
- [ ] Tier 3: procedures saved vào OpenViking L1
- [ ] Tier 4: insights saved vào Supermemory
- [ ] Circuit breaker: skip tier nếu LLM fails 3× (không block next tier)
- [ ] Job status API cho monitoring
