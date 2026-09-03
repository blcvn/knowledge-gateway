# TASK-ORCH-005 — Action DAG Executor

| Field | Value |
|---|---|
| **Task ID** | TASK-ORCH-005 |
| **Wave** | 2 |
| **Solution** | [SOL-ORCH-002](../solutions/SOL-ORCH-002-Action-Routine-DAG.md) §3 |
| **Component** | `services/orchestration-service/internal/usecase/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-ORCH-001 |
| **Estimated** | 4h |

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** orchestration/actions.go: ActionService with dependency graph (Requires/ConflictsWith); DAG executor logic
---

## Mục tiêu

Implement `DAGExecutor` — topological sort, parallel execution, retry backoff.

---

## Công việc cụ thể

### 1. DB Migration `deployment/dev/migrations/0048_actions_routines.sql` [NEW]

```sql
CREATE TABLE agent_actions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    TEXT NOT NULL,
    agent_id     TEXT,
    session_id   TEXT,
    name         TEXT,
    type         TEXT NOT NULL,
    payload      JSONB,
    status       TEXT NOT NULL DEFAULT 'pending',
    depends_on   TEXT[] DEFAULT '{}',
    retry_count  INT DEFAULT 0,
    max_retries  INT DEFAULT 3,
    result       JSONB,
    error        TEXT,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_actions_tenant_status ON agent_actions(tenant_id, status);
CREATE INDEX idx_actions_depends ON agent_actions USING gin(depends_on);

CREATE TABLE agent_routines (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    name        TEXT NOT NULL,
    description TEXT,
    steps       JSONB NOT NULL,
    parameters  TEXT[] DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

### 2. Tạo `services/orchestration-service/internal/usecase/dag.go` [NEW]

```go
type DAGExecutor struct {
    actionRepo port.ActionRepository
    pub        port.EventPublisher
    registry   port.ServiceRegistry
    nats       *nats.Conn
}

func (e *DAGExecutor) Execute(ctx context.Context, action *domain.Action) {
    // Set running
    now := time.Now()
    action.Status = domain.ActionRunning; action.StartedAt = &now
    e.actionRepo.Update(ctx, action)

    var result map[string]any
    var err error
    switch action.Type {
    case domain.ActionMemoryStore:  result, err = e.dispatchMemoryStore(ctx, action)
    case domain.ActionMemoryRecall: result, err = e.dispatchMemoryRecall(ctx, action)
    case domain.ActionSignal:       result, err = e.dispatchSignal(ctx, action)
    case domain.ActionCheckpoint:   result, err = e.waitForCheckpoint(ctx, action)
    }

    completedAt := time.Now()
    if err != nil && action.RetryCount < action.MaxRetries {
        action.RetryCount++; action.Status = domain.ActionPending
        delay := time.Duration(1<<uint(action.RetryCount)) * time.Second
        time.AfterFunc(delay, func() { e.Execute(ctx, action) })
    } else if err != nil {
        action.Status = domain.ActionFailed; action.Error = err.Error()
        action.CompletedAt = &completedAt
    } else {
        action.Status = domain.ActionCompleted
        action.Result = result; action.CompletedAt = &completedAt
    }
    e.actionRepo.Update(ctx, action)
    e.pub.Publish(ctx, fmt.Sprintf("action.completed.%s", action.ID), action)
    e.triggerDependents(ctx, action)
}

func (e *DAGExecutor) TopologicalStart(ctx context.Context, actions []*domain.Action) {
    for _, a := range actions {
        if len(a.DependsOn) == 0 { go e.Execute(ctx, a) }
    }
}

func (e *DAGExecutor) triggerDependents(ctx context.Context, completed *domain.Action) {
    dependents, _ := e.actionRepo.FindDependentsOf(ctx, completed.TenantID, completed.ID)
    for _, dep := range dependents {
        deps, _ := e.actionRepo.GetByIDs(ctx, dep.DependsOn)
        allDone := true
        for _, d := range deps { if d.Status != domain.ActionCompleted { allDone = false; break } }
        if allDone { go e.Execute(ctx, dep) }
    }
}
```

### 3. Test `services/orchestration-service/internal/usecase/dag_test.go` [NEW]

```go
func TestDAG_Sequential_ABeforeB(t *testing.T) {
    // A → B: B only runs after A completes
    // Verify execution order
}
func TestDAG_Parallel_IndependentActions(t *testing.T) {
    // A, B independent: both run concurrently
}
func TestDAG_Retry_OnFailure(t *testing.T) {
    // Action fails once → retried with backoff → succeeds on 2nd
}
```

---

## Acceptance Criteria

- [ ] Independent actions run concurrently (goroutines)
- [ ] Dependent action B starts only after A completes
- [ ] Failed action: retry with `2^retryCount` seconds delay
- [ ] After max retries: status=failed, error persisted
- [ ] `triggerDependents` correctly identifies and starts next actions

## Files

```
services/orchestration-service/internal/usecase/dag.go       [NEW]
services/orchestration-service/internal/usecase/dag_test.go  [NEW]
deployment/dev/migrations/0048_actions_routines.sql           [NEW]
```
