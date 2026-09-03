# Change Request: CR-INTEL-004 — Memory Decay & Salience Eviction

**CR ID:** CR-INTEL-004
**Component:** `backend/services/memory-service`
**Priority:** 🟡 High
**Status:** Open
**Version:** v4 / Intelligence Layer
**Solution:** [S4 — Knowledge Evolution](../../../bussiness/solutions/S4-knowledge-evolution.md)
**Features:** [F09](../../../features/09-agent-memory-lifecycle/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-09 | AI Agent Developer | Memory không tự quên — stale info, storage bùng nổ |
| PP-P7-03 | AI Power User | AI nhớ thứ không quan trọng từ 6 tháng trước |

**Neuroscience insight (sleep.md):** Synaptic pruning trong sleep — não tự quên các kết nối yếu, giữ lại kết nối quan trọng.

---

## 2. Salience Score Algorithm

```
salience = importance × recency × frequency

importance = manual_score (1.0 default) × access_boost
recency    = exp(-λ × days_since_last_access)  # exponential decay
frequency  = log(1 + access_count) / log(max_access_count)

Eviction threshold: salience < 0.1 → candidate for deletion
```

---

## 3. API Contract

```http
# Get memory with salience score
GET /v1/memory/{id}
→ {
    "id": "m_123",
    "content": "...",
    "salience": 0.73,
    "last_accessed": "2026-09-01T10:00:00Z",
    "access_count": 15
  }

# Set forgetAfter on specific memory
PATCH /v1/memory/{id}
{ "forget_after": "2026-12-31T00:00:00Z" }

# Run eviction (manual trigger)
POST /v1/memory/evict
{ "user_id": "u_123", "threshold": 0.1 }
→ { "evicted": 23, "kept": 481 }
```

---

## 4. Scheduled Eviction

```go
// backend/services/memory-service/internal/usecase/eviction.go [NEW]
func (u *EvictionUseCase) RunScheduled(ctx context.Context) {
    // Run daily at 3am (NATS trigger or cron)
    memories := u.memRepo.GetLowSalience(ctx, threshold=0.1)
    for _, m := range memories {
        u.memRepo.Delete(ctx, m.ID)
        u.nats.Publish("memory.evicted", EvictedEvent{MemoryID: m.ID})
    }
}

// Salience update on access
func (u *MemoryService) OnAccess(ctx context.Context, memID string) {
    u.memRepo.IncrementAccessCount(ctx, memID)
    u.memRepo.UpdateLastAccessed(ctx, memID, time.Now())
    // Recalculate salience
}
```

---

## 5. Acceptance Criteria

- [ ] Salience score: composite of importance × recency × frequency
- [ ] Exponential decay: salience giảm theo thời gian kể từ last access
- [ ] `forgetAfter` TTL: hard evict sau deadline
- [ ] Scheduled eviction: auto-run mỗi ngày
- [ ] Eviction threshold configurable per tenant
- [ ] Evicted memories publish NATS event (audit trail)
