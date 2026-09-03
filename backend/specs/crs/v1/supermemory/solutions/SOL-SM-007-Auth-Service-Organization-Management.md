# SOL-SM-007 — Solution: Auth Service & Organization Management

| Field | Value |
|---|---|
| **Solution ID** | SOL-SM-007 |
| **CR** | CR-SM-007 |
| **TDD ref** | [07-supermemory-services.md](../../../tdd/architecture/07-supermemory-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |
| **Component** | `services/vnp-platform` |

---

## 1. Giải pháp

Supermemory auth delegated to vnp-platform (Auth service). Organization management via RBAC.

Supermemory-specific RBAC roles:
- `sm:admin` — full access to all memory spaces
- `sm:editor` — read/write own space
- `sm:viewer` — read-only

No separate auth service needed — use gateway JWT middleware.

## 2. Acceptance Criteria

- [ ] SM operations gated by sm:* RBAC roles
- [ ] Org admin can view all member memory stats
- [ ] API key scoped to specific memory spaces

