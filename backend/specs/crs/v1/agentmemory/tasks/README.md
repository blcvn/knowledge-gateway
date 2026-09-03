# AgentMemory Task Planning — README

**Project:** VNP Memory  
**Domain:** AgentMemory Feature Parity  
**Solutions ref:** `specs/crs/v1/agentmemory/solutions/`  
**Architecture ref:** `specs/architecture.md` v3.0  
**Date:** 2026-06-17  

---

## Wave Map

| Wave | Scope | Tasks |
|------|-------|-------|
| **Wave 1** (Foundation) | Observe Service + Memory Lifecycle + Hybrid Search | TASK-AM-001 → TASK-AM-008 |
| **Wave 2** (Integration) | Consolidation Pipeline + MCP Tools (53) | TASK-AM-009 → TASK-AM-013 |
| **Wave 3** (Orchestration) | Multi-Agent + Session Replay | TASK-AM-014 → TASK-AM-018 |
| **Wave 4** (Governance) | Audit + Diagnostics + DB Migrations | TASK-AM-019 → TASK-AM-022 |

---

## Task List (22 tasks)

| Task | Wave | Solution | Component | Est |
|------|------|----------|-----------|-----|
| [TASK-AM-001](./TASK-AM-001-protobuf-contracts.md) | 1 | SOL-001/002/003/004 | `api/proto/` | 3h |
| [TASK-AM-002](./TASK-AM-002-observe-service-core.md) | 1 | SOL-001 | `services/observe-service/` | 8h |
| [TASK-AM-003](./TASK-AM-003-observe-service-pipeline.md) | 1 | SOL-001 | `services/observe-service/` | 6h |
| [TASK-AM-004](./TASK-AM-004-privacy-pkg.md) | 1 | SOL-001 | `pkg/privacy/` | 2h |
| [TASK-AM-005](./TASK-AM-005-memory-lifecycle-domain.md) | 1 | SOL-002 | `services/memory-service/` | 6h |
| [TASK-AM-006](./TASK-AM-006-memory-lifecycle-repos.md) | 1 | SOL-002 | `services/memory-service/` | 4h |
| [TASK-AM-007](./TASK-AM-007-search-pkg.md) | 1 | SOL-003 | `pkg/search/` | 6h |
| [TASK-AM-008](./TASK-AM-008-observe-search-service.md) | 1 | SOL-003 | `services/observe-search/` | 6h |
| [TASK-AM-009](./TASK-AM-009-consolidation-pipeline.md) | 2 | SOL-006 | `services/memory-service/` | 6h |
| [TASK-AM-010](./TASK-AM-010-consolidation-compressor.md) | 2 | SOL-006 | `services/memory-service/` | 4h |
| [TASK-AM-011](./TASK-AM-011-mcp-tool-registry.md) | 2 | SOL-008 | `gateway/` | 5h |
| [TASK-AM-012](./TASK-AM-012-mcp-tool-handlers.md) | 2 | SOL-008 | `gateway/` | 6h |
| [TASK-AM-013](./TASK-AM-013-context-injection.md) | 2 | SOL-008 | `services/observe-service/` + `gateway/` | 3h |
| [TASK-AM-014](./TASK-AM-014-orchestration-service-core.md) | 3 | SOL-004 | `services/orchestration-service/` | 8h |
| [TASK-AM-015](./TASK-AM-015-orchestration-background.md) | 3 | SOL-004 | `services/orchestration-service/` | 4h |
| [TASK-AM-016](./TASK-AM-016-session-replay-sse.md) | 3 | SOL-005 | `services/observe-service/` | 4h |
| [TASK-AM-017](./TASK-AM-017-gateway-routes.md) | 3 | All | `gateway/` | 4h |
| [TASK-AM-018](./TASK-AM-018-bootstrap-integration.md) | 3 | All | `apps/memory/` | 3h |
| [TASK-AM-019](./TASK-AM-019-governance-audit.md) | 4 | SOL-007 | `services/memory-service/` | 5h |
| [TASK-AM-020](./TASK-AM-020-health-doctor-snapshot.md) | 4 | SOL-007 | `services/vnp-platform/` | 4h |
| [TASK-AM-021](./TASK-AM-021-database-migrations.md) | 4 | All | `deploy/dev/migrations/` | 3h |
| [TASK-AM-022](./TASK-AM-022-integration-tests.md) | 4 | All | `tests/agentmemory/` | 5h |

---

## Architecture Context

```
Monolith (InProcessRegistry bufconn) — Services 35 → 38:
  am-observe    (service #36) → sessions, observations, SSE
  am-search     (service #37) → BM25+Vector hybrid search
  am-orchestration (service #38) → actions, leases, signals, checkpoints

New pkg/:
  pkg/privacy/          — HMAC redaction (from SOL-001)
  pkg/search/           — BM25 + VectorIndex + RRF (from SOL-003)
  pkg/resilience/       — Circuit breaker (from SOL-006/007)

Data Stores:
  PostgreSQL — all persistent state
  NATS JetStream — events (stream: agentmemory.*)
  BM25 gob file — search index persistence
  Bifrost — LLM gateway (compression, summarization)
```

## New NATS Subjects

| Subject | Publisher | Consumer |
|---------|-----------|----------|
| `agentmemory.session.started` | am-observe | consolidation, audit |
| `agentmemory.session.ended` | am-observe | consolidation (summarize) |
| `agentmemory.observation.captured` | am-observe | am-search (index) |
| `agentmemory.memory.remembered` | memory-service | am-search (reindex) |
| `agentmemory.memory.superseded` | memory-service | audit |
| `agentmemory.memory.expired` | memory-service | am-search (remove), audit |
| `agentmemory.action.completed` | am-orchestration | sentinel checker |
| `agentmemory.signal.sent` | am-orchestration | target agent |
| `agentmemory.checkpoint.resolved` | am-orchestration | audit |

## Execution Order

```
Wave 1 (parallel start):
  AM-004 (privacy pkg) → needed by AM-002
  AM-007 (search pkg) → needed by AM-008
  AM-001 (proto) → needed by all services
  
  Then: AM-002 + AM-003 (observe service)
  Then: AM-005 + AM-006 (memory lifecycle)
  Then: AM-008 (observe-search)

Wave 2:
  AM-009 + AM-010 (consolidation) — depends on AM-006
  AM-011 + AM-012 + AM-013 (MCP tools) — depends on AM-001

Wave 3:
  AM-014 + AM-015 (orchestration)
  AM-016 (session replay SSE)
  AM-017 (gateway routes) — depends on all services done
  AM-018 (bootstrap) — depends on AM-017

Wave 4:
  AM-019 (governance/audit)
  AM-020 (health/doctor)
  AM-021 (resilience pkg)
  AM-022 (DB migrations) — can run early
```
