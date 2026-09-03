# TASK-ENT-008 — OPA Policy Enforcement

| Field | Value |
|---|---|
| **Task ID** | TASK-ENT-008 |
| **Wave** | 3 |
| **Solution** | [SOL-ENT-003](../solutions/SOL-ENT-003-Governance-Center.md) §1.2 |
| **Component** | `shared/pkg/privacy/opa.go` |
| **Priority** | 🔴 Critical |
| **Depends On** | — |
| **Estimated** | 4h |

---

## Mục tiêu

OPA (Open Policy Agent) enforcement: reject memory stores that violate policy.

---

## Công việc cụ thể

### `shared/pkg/privacy/opa.go` [NEW]

```go
package privacy

// OPAEnforcer checks OPA policies before memory operations
type OPAEnforcer struct {
    rego *rego.Rego
}

const defaultPolicy = `
package vnp.memory

default allow = true
default violation = ""

# Policy 1: No PII in semantic memory
deny_pii {
    input.type == "semantic"
    contains_pii(input.content)
}

violation = "pii_in_semantic_memory" {
    deny_pii
}

allow = false { violation != "" }

# Helper: simple PII detection (extend with ML model)
contains_pii(content) {
    regex.match("[0-9]{3}-[0-9]{2}-[0-9]{4}", content)  # SSN
}
contains_pii(content) {
    regex.match("[0-9]{16}", content)  # Credit card
}
`

func NewOPAEnforcer(policyRego string) (*OPAEnforcer, error) {
    if policyRego == "" { policyRego = defaultPolicy }
    r := rego.New(
        rego.Query("data.vnp.memory.allow, data.vnp.memory.violation"),
        rego.Module("policy.rego", policyRego),
    )
    return &OPAEnforcer{rego: r}, nil
}

func (o *OPAEnforcer) AllowStore(ctx context.Context, req *StoreCheckRequest) error {
    pq, err := o.rego.PrepareForEval(ctx)
    if err != nil { return err }

    results, err := pq.Eval(ctx, rego.EvalInput(map[string]any{
        "type":      req.Type,
        "content":   req.Content,
        "tenant_id": req.TenantID,
    }))
    if err != nil { return err }

    allow := results[0].Expressions[0].Value.(bool)
    violation := results[0].Expressions[1].Value.(string)

    if !allow {
        return fmt.Errorf("OPA policy violation: %s", violation)
    }
    return nil
}
```

### Integration in memory store flow

```go
// gateway/adapter/handler/memory_handler.go [MODIFY]
// Before dispatching to engine:
if err := h.opa.AllowStore(r.Context(), &privacy.StoreCheckRequest{
    Type: req.Type, Content: req.Content, TenantID: tenantID,
}); err != nil {
    writeError(w, 422, "policy_violation", err.Error())
    return
}
```

---

## Acceptance Criteria

- [ ] Default policy rejects SSN in semantic memory
- [ ] Default policy rejects credit card numbers
- [ ] Policy violations return `422 Unprocessable Entity`
- [ ] Custom policy loadable from config file
- [ ] `go test ./shared/pkg/privacy/...` passes with mock OPA

## Files

```
shared/pkg/privacy/opa.go         [NEW]
shared/pkg/privacy/opa_test.go    [NEW]
gateway/adapter/handler/memory_handler.go  [MODIFY — OPA check]
```
