# TASK-ORCH-002 — Checkpoint Usecase (Create / Resolve / Poll)

| Field | Value |
|---|---|
| **Task ID** | TASK-ORCH-002 |
| **Wave** | 2 |
| **Solution** | [SOL-ORCH-001](../solutions/SOL-ORCH-001-Checkpoints-Sentinels.md) §3 |
| **Component** | `services/orchestration-service/internal/usecase/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-ORCH-001 |
| **Estimated** | 3h |

---

## Mục tiêu

Implement `CheckpointUseCase` — create, poll, resolve (approve/reject), auto-expire.

---

## Công việc cụ thể

### 1. Tạo `services/orchestration-service/internal/usecase/checkpoint.go` [NEW]

Xem code đầy đủ tại [SOL-ORCH-001 §3](../solutions/SOL-ORCH-001-Checkpoints-Sentinels.md).

Key behaviors:
- `Create()`: insert to DB → publish `agent.checkpoint.created` NATS event → schedule goroutine for auto-expiry
- `GetStatus()`: simple DB lookup (agent polls this)
- `Resolve()`: validate pending → update status → publish `agent.checkpoint.resolved.{id}` NATS
- `expireCheckpoint()` (goroutine): after `timeout_minutes` → set status=expired + publish resolved event

### 2. Port interface `services/orchestration-service/internal/port/checkpoint_repository.go` [NEW]

```go
type CheckpointRepository interface {
    Create(ctx context.Context, cp *domain.Checkpoint) error
    Get(ctx context.Context, id, tenantID string) (*domain.Checkpoint, error)
    Update(ctx context.Context, cp *domain.Checkpoint) error
    ListPending(ctx context.Context, tenantID string) ([]*domain.Checkpoint, error)
}
```

### 3. Tạo `services/orchestration-service/internal/usecase/checkpoint_test.go` [NEW]

```go
func TestCheckpoint_Create_PublishesEvent(t *testing.T) { ... }
func TestCheckpoint_Resolve_Approve_UnblocksAgent(t *testing.T) { ... }
func TestCheckpoint_Resolve_AlreadyResolved_Error(t *testing.T) { ... }
func TestCheckpoint_AutoExpire_RejectsAfterTimeout(t *testing.T) { ... }
```

---

## Acceptance Criteria

- [ ] `Create()` inserts checkpoint + publishes NATS event
- [ ] `Resolve(approved=true)` → status=approved, NATS signal sent
- [ ] `Resolve` on non-pending → error
- [ ] Auto-expiry fires after timeout (test with 1s timeout)

## Files

```
services/orchestration-service/internal/usecase/checkpoint.go       [NEW]
services/orchestration-service/internal/usecase/checkpoint_test.go  [NEW]
services/orchestration-service/internal/port/checkpoint_repository.go [NEW]
```
