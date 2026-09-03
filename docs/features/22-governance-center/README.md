# Feature 22 — Governance Center

> **Loại:** Compliance | **Priority:** High | **Status:** Implemented (CR-AM-007)

## Mô tả

Governance Center là hub quản lý compliance, audit, và security policies của VNP Memory. Bao gồm GDPR forget (cascading xóa across all engines), audit trail với 40+ operation types, OPA policy management, và tenant lifecycle management.

---

## Business Logic

### GDPR Forget

GDPR Forget là quy trình xóa dữ liệu của user theo yêu cầu compliance:

1. **Preview (Dry-run)**: Trước khi xóa thật, có thể preview xem những gì sẽ bị xóa.
2. **Cascading Delete**: Thực thi xóa đồng thời trên tất cả 6 engines + agent memories + session data.
3. **Audit Log**: Mọi deletion được ghi vào audit trail với actor, timestamp, và scope.
4. **Verification**: Sau khi xóa, re-query để verify data = empty.

### Audit Trail

Ghi lại mọi operation quan trọng:
- **Operation types**: 40+ types bao gồm memory_store, memory_delete, profile_update, key_create, key_revoke, tenant_create, policy_update, gdpr_forget...
- **Per record**: actor (user/API key), action, entity_type, entity_id, tenant_id, engine, timestamp, metadata
- **Searchable**: Filter theo actor, action, entity, tenant, engine, time range

### Policy Management (OPA)

OPA (Open Policy Agent) policies cho fine-grained control:
- Define policies per entity type (e.g., "Only admin can delete memories older than 7 days")
- Policies apply ở gateway layer — enforce trước khi request tới engine

### Tenant Management

- Create/update tenants (name, slug, tier, status)
- Suspend/delete tenant (cascading disable)
- Engine aliases per tenant (custom engine routing keys)

### Health Monitor & Diagnostics (CR-AM-007)

- `HealthSnapshot`: Detailed struct thay thế simple `{status: "ok"}` — gồm per-service status, memory usage, queue depth, error rate.
- **Doctor Command**: Chạy diagnostic checks → identify problems + suggest fixes.
- **Git Snapshots**: Snapshot repository state tại điểm quan trọng (cho audit trail).
- **Circuit Breaker Monitor**: Xem trạng thái circuit breakers trên tất cả LLM providers.

---

## Dataflow

### GDPR Forget Flow

```
POST /v1/console/governance/gdpr/forget/preview
        │
        ├── Input: {user_id, tenant_id}
        │
        ▼
GovernanceHandler
        │
        ├── Fan-out PREVIEW queries to all 6 engines:
        │         ├── Cognee: "What data exists for user X?"
        │         ├── Graphiti: "What episodes belong to user X?"
        │         ├── Zep: "What sessions belong to user X?"
        │         ├── Memobase: "What blobs/profiles belong to user X?"
        │         ├── OpenViking: "What files belong to user X?"
        │         └── Supermemory: "What memories belong to user X?"
        │
        └── Return: {total_records: N, breakdown_per_engine: {...}, warnings: [...]}


POST /v1/console/governance/gdpr/forget
        │
        ▼
GovernanceHandler (cascade delete)
        │
        ├── Fan-out DELETE to all 6 engines + agent memories (parallel)
        │
        ├── Write audit entry:
        │         {action: "gdpr_forget", actor: admin_id, user_id, engines_affected}
        │
        └── Return: {deleted_count: N, audit_id: "..."}
```

### Audit Trail Query

```
GET /v1/console/governance/audit
        │
        ├── Query params: actor, action, entity_type, tenant_id, engine, from, to
        │
        ▼
GovernanceHandler
        │
        └── Query audit_entries table (PostgreSQL)
                  └── Return: [{id, actor, action, entity, timestamp, metadata}]
```

### Policy Management

```
POST /v1/console/governance/policies
        │
        ├── Input: {name, entity_type, policy_rego: "...OPA rule..."}
        │
        ▼
GovernanceHandler
        │
        ├── Validate OPA policy syntax
        ├── Store policy in PostgreSQL
        └── Hot-reload OPA engine with new policy

(On every request)
        └── Gateway → OPA check: "Is this action allowed for this actor?"
                  ├── Allow → proceed
                  └── Deny  → 403 Forbidden
```

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/v1/console/governance/gdpr/forget/preview` | GDPR forget preview (dry-run) |
| `POST` | `/v1/console/governance/gdpr/forget` | Execute GDPR forget |
| `GET` | `/v1/console/governance/audit` | Search audit log |
| `GET` | `/v1/console/governance/policies` | List policies |
| `POST` | `/v1/console/governance/policies` | Create policy |
| `PUT` | `/v1/console/governance/policies/{id}` | Update policy |
| `GET` | `/v1/console/governance/tenants` | List tenants |
| `POST` | `/v1/console/governance/tenants` | Create tenant |
| `PUT` | `/v1/console/governance/tenants/{id}` | Update tenant |

---

## Database Tables

| Table | Nội dung |
|-------|---------|
| `audit_entries` | Audit trail records (40+ operation types) |
| Tenant tables | In `vnp-platform` service |
