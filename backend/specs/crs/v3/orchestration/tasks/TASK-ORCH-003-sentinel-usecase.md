# TASK-ORCH-003 — Sentinel Usecase & Sketch/Crystal

| Field | Value |
|---|---|
| **Task ID** | TASK-ORCH-003 |
| **Wave** | 2 |
| **Solution** | [SOL-ORCH-001](../solutions/SOL-ORCH-001-Checkpoints-Sentinels.md) §4,§5 |
| **Component** | `services/orchestration-service/internal/usecase/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-ORCH-001 |
| **Estimated** | 3h |

---

## Mục tiêu

Implement `SentinelUseCase` (event watchers) và `SketchUseCase` (ephemeral memory).

---

## Công việc cụ thể

### 1. Tạo `services/orchestration-service/internal/usecase/sentinel.go` [NEW]

Key behaviors:
- `Create()`: insert to DB → if trigger_type=event, subscribe to NATS subject immediately
- `trigger()`: update last_triggered_at → dispatch action (notify|signal|webhook|pause_agent)
- `Remove()`: delete DB record + unsubscribe NATS

```go
func (u *SentinelUseCase) Create(ctx context.Context, s *domain.Sentinel) (*domain.Sentinel, error) {
    s.ID = uuid.NewString(); s.Status = "active"
    if err := u.repo.Create(ctx, s); err != nil { return nil, err }
    if s.TriggerType == "event" && s.TriggerSubject != "" {
        sub, err := u.nats.Subscribe(s.TriggerSubject, func(msg *nats.Msg) {
            go u.trigger(context.Background(), s, msg.Data)
        })
        if err == nil {
            u.mu.Lock(); u.subs[s.ID] = sub; u.mu.Unlock()
        }
    }
    return s, nil
}
```

### 2. Tạo `services/orchestration-service/internal/usecase/sketch.go` [NEW]

```go
func (u *SketchUseCase) Create(ctx context.Context, tenantID, agentID, content string) (*domain.Sketch, error) {
    sk := &domain.Sketch{
        ID: uuid.NewString(), TenantID: tenantID, AgentID: agentID,
        Content: content, ExpiresAt: time.Now().Add(24 * time.Hour),
    }
    return sk, u.repo.Create(ctx, sk)
}

func (u *SketchUseCase) Crystallize(ctx context.Context, sketchID, tenantID, memType string) error {
    sk, err := u.repo.Get(ctx, sketchID, tenantID)
    if err != nil { return err }
    // Dispatch to appropriate engine via memory store
    conn := u.registry.Get(domain.EngineForType(memType))
    // ... call ingest gRPC
    return u.repo.Delete(ctx, sketchID)
}
```

### 3. Background sketch cleanup (goroutine in service startup)

```go
// Every hour: DELETE FROM agent_sketches WHERE expires_at < NOW()
go func() {
    ticker := time.NewTicker(time.Hour)
    for range ticker.C {
        db.Exec(ctx, "DELETE FROM agent_sketches WHERE expires_at < NOW()")
    }
}()
```

---

## Acceptance Criteria

- [ ] Sentinel created → NATS subscription active (for event type)
- [ ] Trigger fires within 100ms of NATS message
- [ ] Sketch created with 24h expiry
- [ ] Crystallize → memory stored + sketch deleted
- [ ] Background cleanup runs hourly

## Files

```
services/orchestration-service/internal/usecase/sentinel.go  [NEW]
services/orchestration-service/internal/usecase/sketch.go    [NEW]
```
