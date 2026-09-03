# CR-007 — Governance Center: Mock → Real API

| Field | Value |
|---|---|
| **CR ID** | CR-007 |
| **Title** | Governance Center: Quản lý Tenant, Policy, Audit Logs, GDPR |
| **Type** | Feature Implementation |
| **Priority** | P1 — High |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Module** | Governance |
| **Files thay đổi** | `ui/src/mock/governance.mock.ts`, `ui/src/hooks/useGovernance.ts`, `ui/src/services/governance.service.ts` |

---

## 1. Hiện trạng

Mock data ([`governance.mock.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/mock/governance.mock.ts)):
Cung cấp tenant default, OPA policy "Admin Only", và 1 audit log fake.

---

## 2. Backend API cần implement

Base path: `/v1/console/governance`
Data source: `vnp-admin` database.

### 2.1 Tenants Management

- `GET /v1/console/governance/tenants`
- `POST /v1/console/governance/tenants`
- `PUT /v1/console/governance/tenants/{id}`

**Response schema** (`Tenant`):
```json
{
  "id": "tenant_123",
  "name": "Finance Dept",
  "created_at": "2026-01-01T00:00:00Z",
  "status": "Active"
}
```

### 2.2 OPA Policies

- `GET /v1/console/governance/policies`
- `POST /v1/console/governance/policies`
- `PUT /v1/console/governance/policies/{id}`

Quản lý Open Policy Agent rules cho RBAC/ABAC.

**Response schema** (`Policy`):
```json
{
  "id": "pol_abc",
  "name": "Read Only Memory",
  "rego_code": "package authz\nallow { input.action == \"read\" }",
  "scope": "memory:*",
  "enabled": true
}
```

### 2.3 Audit Logs

- `GET /v1/console/governance/audit` (với query params filtering)

**Response schema** (`AuditLogEntry`):
```json
{
  "id": "log_789",
  "tenant_id": "tenant_123",
  "actor_id": "usr_456",
  "action": "DELETE",
  "entity_type": "MemoryItem",
  "created_at": "2026-06-16T12:00:00Z",
  "result": "success"
}
```

### 2.4 GDPR / Forget Requests

- `POST /v1/console/governance/gdpr/forget`
- `POST /v1/console/governance/gdpr/forget/preview` (Dry run)

Gửi request xóa toàn bộ data liên quan đến một UserID trên mọi engine (Cascading delete).

---

## 3. Frontend thay đổi

### 3.1 Xóa mock dependency trong `useGovernance.ts`

```typescript
// SAU
import { useQuery } from '@tanstack/react-query';
import { governanceService } from '../services/governance.service';

export function useTenants() {
  return useQuery({
    queryKey: ['governance', 'tenants'],
    queryFn: () => governanceService.getTenants(),
  });
}

export function usePolicies() {
  return useQuery({
    queryKey: ['governance', 'policies'],
    queryFn: () => governanceService.getPolicies(),
  });
}

export function useAuditLogs(filters: Record<string, string>) {
  return useQuery({
    queryKey: ['governance', 'auditLogs', filters],
    queryFn: () => governanceService.getAuditLogs(filters),
  });
}

// Bổ sung các useMutation cho create/update tenant, policy, và trigger GDPR
```

---

## 4. Điều kiện hoàn thành

- [ ] Danh sách Tenants và Policies load thành công từ db.
- [ ] Audit logs tìm kiếm và lọc đúng theo API params.
- [ ] Thực hiện preview GDPR request trả về summary những gì sẽ bị xóa.
- [ ] Không còn import mock data.
