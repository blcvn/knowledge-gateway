---
id: TASK-062
title: "[SOL-003 T08] vnp-admin — Audit Log Query + Policy CRUD HTTP Handlers"
service: vnp-admin
type: FEAT
priority: P1
status: Done
created: 2026-05-14
updated: 2026-05-14
linked_specs:
  - ui/specs/solutions/SOL-003-ui-gateway-hardening.md
  - gateway/specs/solutions/SOL-002-ux-console-api-upgrade.md
---

## Mục Tiêu
Expose HTTP handlers trong `vnp-admin` service để Gateway gRPC client (SOL-002 T08–T09) có thể proxy calls cho:
- Audit log search/query
- OPA Policy CRUD (list, create, update)

## Bối Cảnh Nghiệp Vụ
Gateway đã implement `console_audit_usecase.go` và `console_policy_usecase.go` (SOL-002 T08–T09) với PG store + migration. Các usecases này cần downstream `vnp-admin` service để persist actual audit events và evaluate OPA policies.

## Phạm Vi Công Việc (Scope)

### In Scope
1. **Audit Log Handler**:
   - `GET /api/v1/audit/logs` — Search audit logs with filters (user_id, action, date_range)
   - `POST /api/v1/audit/logs` — Record new audit event
2. **Policy Handler**:
   - `GET /api/v1/policies` — List all OPA policies
   - `POST /api/v1/policies` — Create policy
   - `PUT /api/v1/policies/{id}` — Update policy
   - `DELETE /api/v1/policies/{id}` — Delete policy
3. **Usecase layer**: Business logic for audit filtering, policy validation
4. **Repository layer**: PostgreSQL store queries

### Out of Scope
- OPA runtime evaluation engine (separate service)
- Real-time audit streaming (FEAT-012 scope)

## Thiết Kế Kỹ Thuật

### API Contract
```
GET  /api/v1/audit/logs?user_id=X&action=Y&from=Z&to=W&limit=50
  → 200: { data: AuditLogEntry[], total: number }

POST /api/v1/audit/logs
  Body: { user_id, action, resource_type, resource_id, metadata }
  → 201: { id: string }

GET  /api/v1/policies
  → 200: { data: Policy[] }

POST /api/v1/policies
  Body: { name, description, rego_code, scope }
  → 201: Policy

PUT  /api/v1/policies/{id}
  Body: Partial<Policy>
  → 200: Policy
```

### Internal Architecture
```
handler/audit_handler.go   → usecase/audit_usecase.go   → store/audit_store.go
handler/policy_handler.go  → usecase/policy_usecase.go  → store/policy_store.go
```

## Acceptance Criteria
- [ ] AC-1: `GET /api/v1/audit/logs` returns paginated audit entries
- [ ] AC-2: `POST /api/v1/audit/logs` creates new audit event → returns ID
- [ ] AC-3: Policy CRUD endpoints return correct status codes (201 create, 200 update, 204 delete)
- [ ] AC-4: Unit tests for handler + usecase ≥ 80% coverage
- [ ] AC-5: Gateway gRPC client can successfully call vnp-admin audit/policy endpoints

## Test Requirements
- Unit tests: Handler response codes, usecase filtering logic
- Integration tests: PG store CRUD operations
- Minimum coverage: 80%
