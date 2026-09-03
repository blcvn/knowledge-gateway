# v4/intelligence Task Planning — README

**Domain:** Intelligence & Adaptive Features
**Solutions ref:** `backend/specs/crs/v4/intelligence/solutions/`
**TDD ref:** `backend/specs/tdd/architecture/`
**Date:** 2026-09-03

---

## Wave Map

| Wave | Scope | Tasks |
|---|---|---|
| **Wave 1** (Profile & Context) | User profile assembly, tiered context injection | TASK-INTEL-001 → TASK-INTEL-004 |
| **Wave 2** (Evolution & Decay) | Knowledge evolution, memory decay/eviction | TASK-INTEL-005 → TASK-INTEL-007 |
| **Wave 3** (Replay & Debug) | Session replay, context debugger | TASK-INTEL-008 → TASK-INTEL-010 |

---

## Task List

| Task | Wave | Solution | Component | Est |
|---|---|---|---|---|
| [TASK-INTEL-001](./TASK-INTEL-001-user-profile-migration.md) | 1 | SOL-INTEL-001 | `deployment/migrations/` | 1h |
| [TASK-INTEL-002](./TASK-INTEL-002-profile-assembly-usecase.md) | 1 | SOL-INTEL-001 | `services/memobase-engine/` | 5h |
| [TASK-INTEL-003](./TASK-INTEL-003-profile-api-endpoint.md) | 1 | SOL-INTEL-001 | `gateway/adapter/handler/` | 2h |
| [TASK-INTEL-004](./TASK-INTEL-004-tiered-context-injection.md) | 1 | SOL-INTEL-002 | `services/ov-fs/` | 5h |
| [TASK-INTEL-005](./TASK-INTEL-005-knowledge-evolution.md) | 2 | SOL-INTEL-003 | `services/sm-memory/` | 6h |
| [TASK-INTEL-006](./TASK-INTEL-006-memory-decay-salience.md) | 2 | SOL-INTEL-004 | `services/memory-service/` | 5h |
| [TASK-INTEL-007](./TASK-INTEL-007-memory-decay-migration.md) | 2 | SOL-INTEL-004 | `deployment/migrations/` | 1h |
| [TASK-INTEL-008](./TASK-INTEL-008-session-replay-export.md) | 3 | SOL-INTEL-005 | `services/observe-service/` | 4h |
| [TASK-INTEL-009](./TASK-INTEL-009-session-replay-endpoint.md) | 3 | SOL-INTEL-005 | `gateway/adapter/handler/` | 2h |
| [TASK-INTEL-010](./TASK-INTEL-010-agent-context-debugger.md) | 3 | SOL-INTEL-006 | `gateway/adapter/handler/` | 3h |

**Total estimate:** ~34h
