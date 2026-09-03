# SOL-SM-002 — Solution: Memory Engine & Knowledge Graph

| Field | Value |
|---|---|
| **Solution ID** | SOL-SM-002 |
| **CR** | CR-SM-002 |
| **TDD ref** | [07-supermemory-services.md](../../../tdd/architecture/07-supermemory-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |
| **Component** | `services/sm-engine` |

---

## 1. Giải pháp

Adaptive memory engine: detect contradictions, evolve knowledge over time.

See SOL-INTEL-003 (Knowledge Evolution) for contradiction resolution implementation.
Supermemory adds: memory space management, memory slots, memory versioning.

```go
// Memory space: named collection of memories
type MemorySpace struct {
    ID       string
    Name     string    // "work", "personal", "project_alpha"
    TenantID string
    UserID   string
}
```

## 2. Acceptance Criteria

- [ ] Memory spaces allow scoping queries
- [ ] Version history tracked per memory
- [ ] Contradiction detection integrated with SOL-INTEL-003

