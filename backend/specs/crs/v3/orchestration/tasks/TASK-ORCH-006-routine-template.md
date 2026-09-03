# TASK-ORCH-006 — Routine Template Instantiation

| Field | Value |
|---|---|
| **Task ID** | TASK-ORCH-006 |
| **Wave** | 3 |
| **Solution** | [SOL-ORCH-002](../solutions/SOL-ORCH-002-Action-Routine-DAG.md) §4 |
| **Component** | `services/orchestration-service/internal/usecase/` |
| **Priority** | 🟠 Medium |
| **Depends On** | TASK-ORCH-005 |
| **Estimated** | 3h |

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** orchestration/sketches.go: SketchService with LLM-based routine template generation
---

## Mục tiêu

Implement `RoutineUseCase` — parameterized template → concrete Action DAG.

---

## Công việc cụ thể

### 1. Tạo `services/orchestration-service/internal/usecase/routine.go` [NEW]

```go
func (u *RoutineUseCase) Instantiate(ctx context.Context, routineID, tenantID, agentID, sessionID string, params map[string]string) ([]*domain.Action, error) {
    routine, err := u.routineRepo.Get(ctx, routineID, tenantID)
    if err != nil { return nil, fmt.Errorf("routine not found: %w", err) }

    stepToActionID := map[string]string{}
    var actions []*domain.Action

    for _, step := range routine.Steps {
        payload := interpolate(step.Payload, params)
        depIDs := []string{}
        for _, depName := range step.DependsOn {
            if id, ok := stepToActionID[depName]; ok { depIDs = append(depIDs, id) }
        }
        action := &domain.Action{
            ID: uuid.NewString(), TenantID: tenantID, AgentID: agentID,
            SessionID: sessionID, Name: step.Name, Type: step.Type,
            Payload: payload, DependsOn: depIDs,
            MaxRetries: step.MaxRetries, Status: domain.ActionPending,
        }
        if err := u.actionRepo.Create(ctx, action); err != nil { return nil, err }
        actions = append(actions, action)
        stepToActionID[step.Name] = action.ID
    }

    go u.dag.TopologicalStart(ctx, actions)
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

### 2. Tạo `services/orchestration-service/internal/usecase/routine_test.go` [NEW]

```go
func TestRoutine_Instantiate_InterpolatesParams(t *testing.T) {
    // Routine with step payload: {"query": "{{topic}} latest"}
    // Instantiate with params: {"topic": "AI"}
    // Expected action payload: {"query": "AI latest"}
}

func TestRoutine_Instantiate_DependencyResolved(t *testing.T) {
    // Step B depends_on: ["step_a"]
    // After instantiation: B.DependsOn = [action_a.ID]
}
```

### 3. Handler endpoints

```go
// POST /v1/orchestration/routines
// POST /v1/orchestration/routines/{id}/run
// GET  /v1/orchestration/routines
// GET  /v1/orchestration/actions
// GET  /v1/orchestration/actions/{id}
```

---

## Acceptance Criteria

- [ ] `{{param}}` placeholders replaced in payload
- [ ] Step dependencies mapped to action IDs
- [ ] `TopologicalStart` called: first steps (no deps) begin immediately
- [ ] Routine create + run endpoint works end-to-end

## Files

```
services/orchestration-service/internal/usecase/routine.go       [NEW]
services/orchestration-service/internal/usecase/routine_test.go  [NEW]
```
