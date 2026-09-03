---
id: FEAT-001
title: GDPR Cascading Forget
service: vnp-event
version: 1.0.0
status: Draft
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_sol: gateway/SOL-002 (T13)
linked_ux: "ux_spec.md §6.8 Governance Center — GDPR Forget Center"
---

## Mục Tiêu

Implement GDPR "right to be forgotten" với cascading delete across all 6 memory engines. Supports dry-run preview.

## Scope

### In Scope
- gRPC `ForgetService.ForgetUser(user_id, engines, cascade, dry_run)` — execute forget
- gRPC `ForgetService.GetForgetJobStatus(job_id)` — check progress
- Fan-out delete to all 6 engines
- Audit trail of all deletions
- Dry-run mode returns affected count without deleting

### Out of Scope
- Scheduled retention cleanup (background worker)
- User consent management

## Thiết Kế Kỹ Thuật

### Business Logic

1. Receive forget request with user_id and engine list
2. If dry_run: query each engine for affected entities count, return preview
3. If not dry_run:
   a. Create forget job record (status: in_progress)
   b. Fan-out delete to each engine concurrently:
      - Cognee: `cognee-ingestion.DeleteByUser(user_id)`
      - Graphiti: `graphiti-store.DeleteByUser(user_id)`
      - Zep: `zep-user.DeleteUser(user_id)` (cascades to sessions/memory)
      - OpenViking: `ov-admin.DeleteUserData(user_id)`
      - Memobase: `memobase-ingestion.DeleteUser(user_id)` (cascades to profiles/events)
      - Supermemory: `sm-engine.DeleteUserData(user_id)` (cascades to versions)
   c. Collect results, update job status
   d. Record audit event
4. Publish NATS event `gdpr.forget.completed`

### Data Model

```sql
CREATE TABLE forget_jobs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    user_id     TEXT NOT NULL,
    engines     TEXT[] NOT NULL,
    dry_run     BOOLEAN NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',  -- pending, in_progress, completed, failed
    results     JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);
```

## Acceptance Criteria
- [ ] AC-1: Dry-run returns affected entity count per engine without deleting
- [ ] AC-2: Full forget cascades to all 6 engines
- [ ] AC-3: Partial failure (1 engine fails) still deletes from other engines
- [ ] AC-4: Job status trackable via GetForgetJobStatus
- [ ] AC-5: Audit event recorded for every forget operation
- [ ] AC-6: Supermemory version chains fully cleaned (not just latest)
- [ ] AC-7: Memobase profiles + events + blobs all deleted

## Test Requirements
- Unit tests: Fan-out logic, partial failure handling
- Integration tests: Mock 6 engine delete endpoints
- Minimum coverage: 80%
