---
id: FEAT-002
title: OPA Policy CRUD Service
service: vnp-admin
version: 1.0.0
status: Draft
priority: P1
created: 2026-05-13
updated: 2026-05-13
linked_sol: gateway/SOL-002 (T09)
linked_ux: "ux_spec.md §6.8 Governance Center — OPA Policy Editor"
---

## Mục Tiêu

CRUD API cho OPA policies quản lý access control và data governance trên VNP Memory platform.

## Scope

### In Scope
- gRPC `PolicyService.CreatePolicy(Policy)` — tạo policy mới
- gRPC `PolicyService.UpdatePolicy(Policy)` — cập nhật policy
- gRPC `PolicyService.ListPolicies(TenantID)` — liệt kê policies
- gRPC `PolicyService.DeletePolicy(PolicyID)` — xóa policy
- gRPC `PolicyService.ValidateRego(RegoCode)` — validate Rego syntax
- PostgreSQL table `policies`

### Out of Scope
- OPA runtime evaluation (separate OPA sidecar)
- Policy versioning (future)

## Thiết Kế Kỹ Thuật

### Data Model

```sql
CREATE TABLE policies (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    name        TEXT NOT NULL,
    description TEXT,
    rego_code   TEXT NOT NULL,
    scope       TEXT NOT NULL,          -- "memory.read", "memory.write", "admin.*"
    enabled     BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by  TEXT NOT NULL,
    UNIQUE(tenant_id, name)
);
```

## Acceptance Criteria
- [ ] AC-1: Create policy with valid Rego syntax
- [ ] AC-2: Reject policy with invalid Rego syntax (400 error)
- [ ] AC-3: List policies filtered by tenant
- [ ] AC-4: Update policy triggers audit event
- [ ] AC-5: Delete policy requires confirmation (soft-delete first)

## Test Requirements
- Unit tests: Rego validation, CRUD logic
- Integration tests: PostgreSQL + audit event publishing
- Minimum coverage: 80%
