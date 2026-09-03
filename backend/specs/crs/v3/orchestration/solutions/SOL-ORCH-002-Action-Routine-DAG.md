# Solution: SOL-ORCH-002 — Action Task Graph & Routines (Workflow Templates)

**CR:** CR-ORCH-002
**TDD refs:** `architecture/12-agentmemory-services.md`, `models/obs-service.md`
**Version:** v3/orchestration

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** ActionService DAG executor + SketchService routine templates implemented
---

## 1. Architecture: State Machine Pattern

```
Action States (DAG nodes):
  pending → running → completed
                    → failed (with retry)
                    → cancelled

DAG Execution:
  - Topological sort: process nodes with no pending dependencies
  - Parallel: independent nodes run concurrently (goroutines)
  - Dependency tracking: action completes → notify dependents
```

---

## 2. Action Domain

```go
// services/orchestration-service/internal/domain/action.go [MODIFY/NEW]
type ActionStatus string
const (
    ActionPending   ActionStatus = "pending"
    ActionRunning   ActionStatus = "running"
    ActionCompleted ActionStatus = "completed"
    ActionFailed    ActionStatus = "failed"
    ActionCancelled ActionStatus = "cancelled"
)

type ActionType string
const (
    ActionLLMCall      ActionType = "llm_call"
    ActionMemoryStore  ActionType = "memory_store"
    ActionMemoryRecall ActionType = "memory_recall"
    ActionAPICall      ActionType = "api_call"
    ActionSignal       ActionType = "signal"
    ActionCheckpoint   ActionType = "checkpoint"
)

type Action struct {
    ID         string            `json:"id"`
    TenantID   string            `json:"tenant_id"`
    AgentID    string            `json:"agent_id"`
    SessionID  string            `json:"session_id"`
    Name       string            `json:"name"`
    Type       ActionType        `json:"type"`
    Payload    map[string]any    `json:"payload"`
    Status     ActionStatus      `json:"status"`
    DependsOn  []string          `json:"depends_on"`  // action IDs
    RetryCount int               `json:"retry_count"`
    MaxRetries int               `json:"max_retries"` // default: 3
    Result     map[string]any    `json:"result,omitempty"`
    Error      string            `json:"error,omitempty"`
    StartedAt  *time.Time        `json:"started_at,omitempty"`
    CompletedAt *time.Time       `json:"completed_at,omitempty"`
    CreatedAt  time.Time         `json:"created_at"`
}
```

---

## 3. DAG Executor Usecase

```go
// services/orchestration-service/internal/usecase/dag.go [NEW]
type DAGExecutor struct {
    actionRepo port.ActionRepository
    pub        port.EventPublisher
    registry   port.ServiceRegistry
    nats       *nats.Conn
}

// Execute one action (called by scheduler)
func (e *DAGExecutor) ExecuteAction(ctx context.Context, action *Action) {
    now := time.Now()
    action.Status = ActionRunning; action.StartedAt = &now
    e.actionRepo.Update(ctx, action)

    var result map[string]any
    var err error

    switch action.Type {
    case ActionMemoryStore:
        result, err = e.dispatchMemoryStore(ctx, action)
    case ActionMemoryRecall:
        result, err = e.dispatchMemoryRecall(ctx, action)
    case ActionSignal:
        result, err = e.dispatchSignal(ctx, action)
    case ActionCheckpoint:
        result, err = e.waitForCheckpoint(ctx, action)
    // ... other types
    }

    completedAt := time.Now()
    if err != nil {
        if action.RetryCount < action.MaxRetries {
            action.RetryCount++
            action.Status = ActionPending
            delay := time.Duration(1<<action.RetryCount) * time.Second
            time.AfterFunc(delay, func() { e.ExecuteAction(ctx, action) })
        } else {
            action.Status = ActionFailed; action.Error = err.Error()
            action.CompletedAt = &completedAt
        }
    } else {
        action.Status = ActionCompleted
        action.Result = result; action.CompletedAt = &completedAt
    }
    e.actionRepo.Update(ctx, action)

    // Notify dependents
    e.pub.Publish(ctx, fmt.Sprintf("action.completed.%s", action.ID), action)
    e.triggerDependents(ctx, action)
}

// triggerDependents — find actions waiting for this one, run if ready
func (e *DAGExecutor) triggerDependents(ctx context.Context, completed *Action) {
    dependents, _ := e.actionRepo.FindDependentsOf(ctx, completed.TenantID, completed.ID)
    for _, dep := range dependents {
        deps, _ := e.actionRepo.GetByIDs(ctx, dep.DependsOn)
        allCompleted := true
        for _, d := range deps {
            if d.Status != ActionCompleted { allCompleted = false; break }
        }
        if allCompleted {
            go e.ExecuteAction(ctx, dep)
        }
    }
}

// topologicalExecute — start execution of a DAG
func (e *DAGExecutor) topologicalExecute(ctx context.Context, actions []*Action) {
    // Find nodes with no dependencies
    for _, a := range actions {
        if len(a.DependsOn) == 0 {
            go e.ExecuteAction(ctx, a)
        }
    }
}
```

---

## 4. Routine Template

```go
// services/orchestration-service/internal/domain/routine.go [NEW]
type RoutineStep struct {
    Name       string         `json:"name"`
    Type       ActionType     `json:"type"`
    Payload    map[string]any `json:"payload"`  // may contain {{param}} placeholders
    DependsOn  []string       `json:"depends_on"` // step names (not IDs)
    MaxRetries int            `json:"max_retries"`
}

type Routine struct {
    ID          string        `json:"id"`
    TenantID    string        `json:"tenant_id"`
    Name        string        `json:"name"`
    Description string        `json:"description"`
    Steps       []RoutineStep `json:"steps"`
    Parameters  []string      `json:"parameters"` // param names: ["topic", "user_id"]
    CreatedAt   time.Time     `json:"created_at"`
}

// services/orchestration-service/internal/usecase/routine.go [NEW]
func (u *RoutineUseCase) Instantiate(ctx context.Context, routineID, tenantID, agentID, sessionID string, params map[string]string) ([]*Action, error) {
    routine, err := u.routineRepo.Get(ctx, routineID, tenantID)
    if err != nil { return nil, err }

    // Map step names to action IDs for dependency resolution
    stepToActionID := map[string]string{}
    var actions []*Action

    for _, step := range routine.Steps {
        payload := interpolate(step.Payload, params) // replace {{param}} with values
        depIDs := []string{}
        for _, depName := range step.DependsOn {
            if id, ok := stepToActionID[depName]; ok {
                depIDs = append(depIDs, id)
            }
        }

        action := &Action{
            ID: uuid.NewString(), TenantID: tenantID,
            AgentID: agentID, SessionID: sessionID,
            Name: step.Name, Type: step.Type,
            Payload: payload, DependsOn: depIDs,
            MaxRetries: step.MaxRetries, Status: ActionPending,
        }
        actions = append(actions, action)
        stepToActionID[step.Name] = action.ID
    }

    // Persist all actions
    for _, a := range actions {
        u.actionRepo.Create(ctx, a)
    }

    // Start execution
    go u.dag.topologicalExecute(ctx, actions)
    return actions, nil
}

func interpolate(payload map[string]any, params map[string]string) map[string]any {
    result := map[string]any{}
    for k, v := range payload {
        if s, ok := v.(string); ok {
            for param, val := range params {
                s = strings.ReplaceAll(s, "{{"+param+"}}", val)
            }
            result[k] = s
        } else {
            result[k] = v
        }
    }
    return result
}
```

---

## 5. DB Migration

```sql
-- deployment/dev/migrations/0048_actions_routines.sql
CREATE TABLE agent_actions (
    id          UUID PRIMARY KEY,
    tenant_id   TEXT NOT NULL, agent_id TEXT, session_id TEXT,
    name TEXT, type TEXT, payload JSONB,
    status TEXT DEFAULT 'pending',
    depends_on  TEXT[] DEFAULT '{}',
    retry_count INT DEFAULT 0, max_retries INT DEFAULT 3,
    result JSONB, error TEXT,
    started_at TIMESTAMPTZ, completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_actions_tenant_status ON agent_actions(tenant_id, status);

CREATE TABLE agent_routines (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL, name TEXT, description TEXT,
    steps JSONB NOT NULL, parameters TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 6. File Changes

| File | Action |
|---|---|
| `services/orchestration-service/internal/domain/action.go` | **[NEW/MODIFY]** |
| `services/orchestration-service/internal/domain/routine.go` | **[NEW]** |
| `services/orchestration-service/internal/usecase/dag.go` | **[NEW]** DAG executor |
| `services/orchestration-service/internal/usecase/routine.go` | **[NEW]** |
| `services/orchestration-service/internal/usecase/dag_test.go` | **[NEW]** |
| `gateway/adapter/handler/orchestration_handler.go` | **[MODIFY]** action/routine endpoints |
| `deployment/dev/migrations/0048_actions_routines.sql` | **[NEW]** |
