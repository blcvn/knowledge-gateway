# SOL-MB-005 — Solution: Admin Service (User, Project, Billing)

| Field | Value |
|---|---|
| **Solution ID** | SOL-MB-005 |
| **CR** | CR-MB-005 |
| **TDD ref** | [04-memobase-services.md](../../../tdd/architecture/04-memobase-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |
| **Component** | `services/memobase-admin` |

---

## 1. Giải pháp

Admin service: CRUD users, projects, profile configs, billing metrics.

### API endpoints

```
GET  /v1/admin/memobase/users/{id}/profile        → full profile
GET  /v1/admin/memobase/users/{id}/blobs?limit=20 → blob history
POST /v1/admin/memobase/users/{id}/reset-profile   → clear profile
GET  /v1/admin/memobase/billing?tenant_id=...      → token usage stats
```

## 2. Acceptance Criteria

- [ ] Admin can view any user's full profile
- [ ] Admin can reset profiles (GDPR support)
- [ ] Billing API returns token usage per tenant per day

