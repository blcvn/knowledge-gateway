---
id: TASK-065
title: "[SOL-003 T11] vnp-event — GDPR Cascading Forget + Event Timeline Handlers"
service: vnp-event
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
Expose HTTP handlers trong `vnp-event` service cho:
- GDPR cascading forget (xóa user data across 11 services)
- Event timeline query (lịch sử events của user/session)

## Bối Cảnh Nghiệp Vụ
Gateway đã implement `console_gdpr_usecase.go` (SOL-002 T13) với 11-service cascade + audit trail logic. `vnp-event` cần expose handlers để nhận forget requests và orchestrate deletion tới tất cả engine services.

## Phạm Vi Công Việc (Scope)

### In Scope
1. **GDPR Forget Handler**:
   - `POST /api/v1/gdpr/forget` — Cascade delete user data across all engines
   - Request: `{ user_id, reason, requested_by }`
   - Response: `{ forget_id, status, affected_services[], audit_trail_id }`
2. **GDPR Forget Preview**:
   - `POST /api/v1/gdpr/forget/preview` — Preview what would be deleted (dry-run)
   - Response: `{ user_id, data_summary: { engine: count }[], total_records }`
3. **Event Timeline Handler**:
   - `GET /api/v1/events/timeline?user_id=X&from=Y&to=Z` — Query user event history
   - Response: `{ events: Event[], total }`
4. **Cascade Orchestrator**: Fan-out delete to 11 services with rollback on partial failure
5. **Audit Trail**: Record every forget action with before/after snapshot IDs

### Out of Scope
- Data anonymization (alternative to deletion)
- Automated retention policy triggers (separate FEAT)

## Thiết Kế Kỹ Thuật

### GDPR Cascade Order (Dependency-safe)
```
Phase 1 (parallel): cognee, graphiti, openviking, supermemory
Phase 2 (after Phase 1): zep (depends on graphiti edges)
Phase 3 (after Phase 2): memobase (final profile cleanup)
Phase 4 (after all): vnp-admin (audit record), vnp-event (self-cleanup)
```

### Internal Architecture
```
handler/gdpr_handler.go
  → usecase/gdpr_usecase.go
    → orchestrator/cascade_orchestrator.go
      → clients/{engine}_client.go × 11
    → store/audit_store.go (audit trail)

handler/timeline_handler.go
  → usecase/timeline_usecase.go
    → store/event_store.go
```

### Failure Handling
- Each service delete is idempotent
- Partial failure: record which services succeeded/failed in audit trail
- Retry failed services with exponential backoff (max 3 attempts)
- Final status: `completed` | `partial_failure` | `failed`

## Acceptance Criteria
- [ ] AC-1: `POST /api/v1/gdpr/forget/preview` returns data summary without deleting
- [ ] AC-2: `POST /api/v1/gdpr/forget` cascades delete across all 11 services
- [ ] AC-3: Audit trail records forget action with service-level status
- [ ] AC-4: Partial failure results in `partial_failure` status (not crash)
- [ ] AC-5: `GET /api/v1/events/timeline` returns paginated event history
- [ ] AC-6: Unit tests ≥ 80% coverage

## Test Requirements
- Unit tests: Cascade orchestrator, failure handling, audit trail
- Integration tests: Mock gRPC services for cascade delete flow
- Minimum coverage: 80%
