---
id: FEAT-011
title: Governance API — Tenant, Policy, Audit, GDPR
service: vnp-gateway
version: 1.0.0
status: Draft
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
linked_ux: "ux_spec.md §6.8 Governance Center"
---

## Mục Tiêu

REST APIs cho Governance Center: tenant CRUD, OPA policy management, audit log query, GDPR cascading forget.

## Scope

### In Scope
- `GET /v1/console/governance/tenants` — List tenants
- `POST /v1/console/governance/tenants` — Create tenant
- `PUT /v1/console/governance/tenants/{id}` — Update tenant (quotas, namespaces)
- `GET /v1/console/governance/policies` — List OPA policies
- `POST /v1/console/governance/policies` — Create policy
- `PUT /v1/console/governance/policies/{id}` — Update policy
- `GET /v1/console/governance/audit` — Search audit logs
- `POST /v1/console/governance/gdpr/forget` — GDPR cascading forget
- `POST /v1/console/governance/gdpr/forget/preview` — Dry-run forget preview

### Out of Scope
- Retention scheduling (background worker)
- OPA policy evaluation engine (vnp-admin responsibility)

## Thiết Kế Kỹ Thuật

### API Contract

#### POST `/v1/console/governance/gdpr/forget`
```json
{
  "user_id": "user_123",
  "engines": ["all"],
  "cascade": true,
  "dry_run": false
}
```

**Response (200):**
```json
{
  "job_id": "forget_abc123",
  "status": "in_progress",
  "affected": {
    "cognee": { "datasets": 2, "documents": 15 },
    "graphiti": { "episodes": 8, "nodes": 42 },
    "zep": { "sessions": 5, "messages": 120 },
    "openviking": { "files": 3 },
    "memobase": { "profiles": 1, "events": 45 },
    "supermemory": { "memories": 12, "documents": 8 }
  }
}
```

#### GET `/v1/console/governance/audit`
**Query params:** `?actor=xxx&action=xxx&entity=xxx&engine=xxx&from=xxx&to=xxx&cursor=xxx&limit=50`

### Internal Architecture
- **Handler:** `adapter/http/governance_handler.go`
- **Proxy to:** `vnp-admin` (tenants, policies, audit), `vnp-event` (GDPR forget fan-out)
- GDPR forget: fan-out delete requests to all 6 engine admin services

## Acceptance Criteria
- [ ] AC-1: Tenant CRUD with quota management (max_nodes, max_requests)
- [ ] AC-2: Policy CRUD with Rego syntax validation
- [ ] AC-3: Audit log search with filters (actor, action, engine, time range)
- [ ] AC-4: GDPR forget cascades across all 6 engines
- [ ] AC-5: GDPR dry-run returns affected count without deleting
- [ ] AC-6: All endpoints require `super_admin` role
- [ ] AC-7: Audit log records all governance actions

## Test Requirements
- Unit tests: Policy validation, GDPR cascade planning
- Integration tests: Multi-engine forget with mocks
- Minimum coverage: 80%
