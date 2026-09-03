# v3/core-memory Task Planning — README

**Domain:** Core Memory & Integration  
**Solutions ref:** `backend/specs/crs/v3/core-memory/solutions/`  
**TDD ref:** `backend/specs/tdd/architecture/`  
**Date:** 2026-09-03

---

## Wave Map

| Wave | Scope | Tasks |
|---|---|---|
| **Wave 1** (Gateway Router) | Memory routing layer, type classifier | TASK-CORE-001 → TASK-CORE-003 |
| **Wave 2** (Cross-Engine) | RRF search, GDPR forget, session context | TASK-CORE-004 → TASK-CORE-007 |
| **Wave 3** (Intelligence) | Temporal reasoning, MCP 37+ tools | TASK-CORE-008 → TASK-CORE-011 |

---

## Task List

| Task | Wave | Solution | Component | Est |
|---|---|---|---|---|
| [TASK-CORE-001](./TASK-CORE-001-memory-domain-types.md) | 1 | SOL-CORE-001 | `gateway/domain/` | 2h |
| [TASK-CORE-002](./TASK-CORE-002-memory-handler-routing.md) | 1 | SOL-CORE-001 | `gateway/adapter/handler/` | 4h |
| [TASK-CORE-003](./TASK-CORE-003-llm-classifier.md) | 1 | SOL-CORE-001 | `gateway/internal/usecase/` | 3h |
| [TASK-CORE-004](./TASK-CORE-004-rrf-search-hub.md) | 2 | SOL-CORE-002 | `services/vnp-search-hub/` | 5h |
| [TASK-CORE-005](./TASK-CORE-005-cascading-forget.md) | 2 | SOL-CORE-003 | `services/vnp-admin/`, `gateway/` | 5h |
| [TASK-CORE-006](./TASK-CORE-006-session-context-assembly.md) | 2 | SOL-CORE-004 | `services/zep-memory/`, `services/memobase-context/` | 4h |
| [TASK-CORE-007](./TASK-CORE-007-audit-log-migration.md) | 2 | SOL-CORE-003 | `deployment/dev/migrations/` | 1h |
| [TASK-CORE-008](./TASK-CORE-008-temporal-search.md) | 3 | SOL-CORE-005 | `services/graphiti-search/` | 4h |
| [TASK-CORE-009](./TASK-CORE-009-mcp-agent-tools.md) | 3 | SOL-CORE-006 | `gateway/adapter/mcp/` | 5h |
| [TASK-CORE-010](./TASK-CORE-010-mcp-graph-tools.md) | 3 | SOL-CORE-006 | `gateway/adapter/mcp/` | 4h |
| [TASK-CORE-011](./TASK-CORE-011-mcp-admin-tools.md) | 3 | SOL-CORE-006 | `gateway/adapter/mcp/` | 3h |

**Total estimate:** ~40h (5 working days)
