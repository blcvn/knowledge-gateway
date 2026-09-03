# SOL-ENT-003 — Solution: Governance Center (GDPR + OPA + Audit)

| Field | Value |
|---|---|
| **Solution ID** | SOL-ENT-003 |
| **CR** | [CR-ENT-003](../../../../docs/crs/v5/enterprise/CR-ENT-003-Governance-Center.md) |
| **TDD ref** | [08-platform-services.md](../../../tdd/architecture/08-platform-services.md) |
| **Status** | 🔄 Partial |
| **Priority** | 🔴 Critical |

---

## 1. Giải pháp

3 thành phần: Memory Visibility, OPA Policy Enforcement, Immutable Audit Trail.

### 1.1 Memory Visibility — `services/vnp-admin/internal/usecase/visibility.go` [NEW]

```go
// GET /v1/admin/users/{id}/memories
func (u *VisibilityUseCase) GetUserMemories(ctx context.Context, req *VisibilityRequest) (*MemoryInventory, error) {
    // Fan-out tới tất cả engines
    engines := []string{"cognee-search", "graphiti-search", "zep-search",
                         "memobase-engine", "ov-search", "sm-search", "observe-service"}
    
    var wg sync.WaitGroup
    results := map[string][]MemoryUnit{}
    mu := sync.Mutex{}
    
    for _, eng := range engines {
        wg.Add(1)
        go func(e string) {
            defer wg.Done()
            units, _ := u.queryEngine(ctx, e, req.TenantID, req.UserID, req.Filter)
            mu.Lock()
            results[e] = units
            mu.Unlock()
        }(eng)
    }
    wg.Wait()

    return &MemoryInventory{
        UserID: req.UserID, Engines: results,
        TotalCount: countAll(results),
    }, nil
}
```

### 1.2 OPA Policy Enforcement — `shared/pkg/privacy/opa.go` [NEW]

```go
type OPAEnforcer struct {
    rego *rego.Rego
}

// Check before storing memory
func (o *OPAEnforcer) AllowStore(ctx context.Context, req *StoreRequest) error {
    input := map[string]any{
        "tenant_id": req.TenantID, "type": req.Type,
        "content": req.Content, "metadata": req.Metadata,
    }
    result, _ := o.rego.Eval(ctx, rego.EvalInput(input))
    if !result[0].Bindings["allow"].(bool) {
        violation := result[0].Bindings["violation"].(string)
        return fmt.Errorf("OPA policy violation: %s", violation)
    }
    return nil
}
```

### 1.3 Audit Trail — `deployment/dev/migrations/0XX_audit_log.sql`

```sql
CREATE TABLE audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    user_id     TEXT,
    operation   TEXT NOT NULL,    -- 'store', 'recall', 'forget', 'policy_violation'
    resource_id TEXT,
    actor_id    TEXT NOT NULL,    -- who performed the operation
    metadata    JSONB,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
-- Immutable: no UPDATE or DELETE allowed (row-level security)
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY audit_insert_only ON audit_log FOR INSERT WITH CHECK (true);
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/vnp-admin/internal/usecase/visibility.go` | NEW |
| `shared/pkg/privacy/opa.go` | NEW — OPA enforcer |
| `gateway/adapter/handler/admin_handler.go` | MODIFY — add visibility + audit endpoints |
| `deployment/dev/migrations/0XX_audit_log.sql` | NEW |

---

## 3. Acceptance Criteria

- [ ] Memory inventory: returns all memories across 6 engines + observe in < 2s
- [ ] OPA policy blocks PII storage in semantic memory
- [ ] Audit log immutable (INSERT-only via RLS)
- [ ] GDPR forget generates completion certificate (see SOL-CORE-003)
- [ ] `GET /v1/admin/audit` searchable by user, operation, time range

---

**Ghi chú audit:** MemoryVisibilityService + AuditService + PolicyService (Rego CRUD) in vnp-admin; OPA enforcement middleware pending full OPA integration
