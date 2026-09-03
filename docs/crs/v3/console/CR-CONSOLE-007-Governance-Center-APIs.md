# Change Request: CR-CONSOLE-007 — Governance Center Backend APIs

**CR ID:** CR-CONSOLE-007
**Component:** `backend/gateway`, `backend/services/vnp-admin`
**Priority:** 🔴 Critical
**Status:** Open
**Version:** v3 / Console
**Feature:** [F22](../../../features/22-governance-center/README.md)
**Depends On:** CR-ENT-003, CR-ENT-004

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P4-02 | Enterprise Architect | Không có GDPR compliance tool |
| PP-P4-03 | Enterprise Architect | Không có policy management |

---

## 2. APIs

### GDPR Forget

| Method | Path | Description |
|---|---|---|
| `POST` | `/v1/admin/forget` | Cascading delete user data |
| `GET` | `/v1/admin/forget/{audit_id}` | Check forget status |

### Audit Trail

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/admin/audit` | Query audit log |
| `GET` | `/v1/admin/audit/export` | Export as CSV |
| `GET` | `/v1/console/governance/audit` | Console audit view |

```
GET /v1/admin/audit?user_id=xxx&operation=forget&from=2026-09-01&limit=50
Response:
{
  "entries": [
    {
      "id": "...", "tenant_id": "...", "actor_id": "admin-1",
      "operation": "forget.completed", "user_id": "user-123",
      "metadata": {"deleted_from": ["cognee", "graphiti"]},
      "created_at": "..."
    }
  ],
  "total": 142
}
```

### Policy Management (OPA)

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/admin/policies` | List active policies |
| `POST` | `/v1/admin/policies` | Create/update policy |
| `DELETE` | `/v1/admin/policies/{id}` | Remove policy |
| `POST` | `/v1/admin/policies/validate` | Test policy against sample |

### Tenant Management (Admin)

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/admin/tenants` | List all tenants |
| `GET` | `/v1/admin/tenants/{id}` | Tenant detail |
| `POST` | `/v1/admin/tenants/{id}/suspend` | Suspend tenant |
| `POST` | `/v1/admin/tenants/{id}/activate` | Activate tenant |
| `GET` | `/v1/admin/users/{id}/memories` | View user's memories (cross-engine) |

---

## 3. Acceptance Criteria

- [ ] Audit log: immutable (no UPDATE/DELETE via API)
- [ ] Audit export: GDPR-compliant CSV download
- [ ] Policy validate: test rule against sample input
- [ ] User memories: admin can view any user's memories across engines
- [ ] Forget: cascading delete with audit record before+after
