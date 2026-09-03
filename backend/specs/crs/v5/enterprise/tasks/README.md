# v5/enterprise Task Planning — README

**Domain:** Enterprise & Operations
**Solutions ref:** `backend/specs/crs/v5/enterprise/solutions/`
**TDD ref:** `backend/specs/tdd/architecture/`
**Date:** 2026-09-03

---

## Wave Map

| Wave | Scope | Tasks |
|---|---|---|
| **Wave 1** (Security & Isolation) | Multi-tenant isolation, audit log | TASK-ENT-001 → TASK-ENT-003 |
| **Wave 2** (Orchestration) | Distributed leases, consolidation pipeline | TASK-ENT-004 → TASK-ENT-006 |
| **Wave 3** (Governance & Ops) | Governance center, observability, health | TASK-ENT-007 → TASK-ENT-011 |

---

## Task List

| Task | Wave | Solution | Component | Est |
|---|---|---|---|---|
| [TASK-ENT-001](./TASK-ENT-001-tenant-middleware.md) | 1 | SOL-ENT-004 | `shared/pkg/tenant/` | 3h |
| [TASK-ENT-002](./TASK-ENT-002-tenant-guard.md) | 1 | SOL-ENT-004 | `shared/pkg/tenant/` | 2h |
| [TASK-ENT-003](./TASK-ENT-003-tenant-isolation-test.md) | 1 | SOL-ENT-004 | `tests/integration/` | 3h |
| [TASK-ENT-004](./TASK-ENT-004-distributed-lease.md) | 2 | SOL-ENT-001 | `services/orchestration-service/` | 5h |
| [TASK-ENT-005](./TASK-ENT-005-signal-router.md) | 2 | SOL-ENT-001 | `services/orchestration-service/` | 3h |
| [TASK-ENT-006](./TASK-ENT-006-consolidation-pipeline.md) | 2 | SOL-ENT-002 | `services/pipeline-service/` | 6h |
| [TASK-ENT-007](./TASK-ENT-007-governance-visibility.md) | 3 | SOL-ENT-003 | `services/vnp-admin/` | 5h |
| [TASK-ENT-008](./TASK-ENT-008-opa-policy.md) | 3 | SOL-ENT-003 | `shared/pkg/privacy/` | 4h |
| [TASK-ENT-009](./TASK-ENT-009-observability-metrics.md) | 3 | SOL-ENT-005 | `shared/pkg/telemetry/` | 4h |
| [TASK-ENT-010](./TASK-ENT-010-llm-cost-tracking.md) | 3 | SOL-ENT-005 | `shared/pkg/telemetry/` | 3h |
| [TASK-ENT-011](./TASK-ENT-011-health-aggregation.md) | 3 | SOL-ENT-006 | `gateway/adapter/handler/` | 4h |

**Total estimate:** ~42h
