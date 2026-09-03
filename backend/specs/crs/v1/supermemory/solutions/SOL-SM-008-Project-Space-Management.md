# SOL-SM-008 — Solution: Project & Space Management

| Field | Value |
|---|---|
| **Solution ID** | SOL-SM-008 |
| **CR** | CR-SM-008 |
| **TDD ref** | [07-supermemory-services.md](../../../tdd/architecture/07-supermemory-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |
| **Component** | `services/sm-memory` |

---

## 1. Giải pháp

Memory spaces = project-scoped buckets. CRUD spaces, add/remove members, set permissions.

```go
// POST /v1/sm/spaces
type Space struct {
    ID       string; Name string; TenantID string
    Members  []string; Private bool; Tags []string
}

// POST /v1/sm/spaces/{id}/members
// DELETE /v1/sm/spaces/{id}/members/{user_id}
```

## 2. Acceptance Criteria

- [ ] Space CRUD functional
- [ ] Private spaces: only members can query
- [ ] Space-level stats: memory count, token usage

