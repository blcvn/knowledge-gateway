# Traceability Matrix — AgentMemory Solutions

**Project:** VNP Memory  
**Domain:** AgentMemory Feature Parity  
**Architecture ref:** `specs/architecture.md` v3.0  
**Date:** 2026-06-16

---

## Solution Map

| CR | Solution File | Integration Point | Wave |
|----|--------------|-------------------|------|
| CR-AM-001 | [SOL-001](./SOL-001-Observe-Service.md) | NEW `services/observe-service/` + Gateway routes | Wave 1 |
| CR-AM-002 | [SOL-002](./SOL-002-Memory-Lifecycle.md) | EXTEND `services/memory-service/` + PostgreSQL | Wave 1 |
| CR-AM-003 | [SOL-003](./SOL-003-Hybrid-Search-Engine.md) | NEW `services/observe-search/` + `pkg/search/` | Wave 1 |
| CR-AM-004 | [SOL-004](./SOL-004-Multi-Agent-Orchestration.md) | NEW `services/orchestration-service/` + PostgreSQL | Wave 3 |
| CR-AM-005 | [SOL-005](./SOL-005-Session-Replay.md) | EXTEND `services/observe-service/` + UI Tab | Wave 3 |
| CR-AM-006 | [SOL-006](./SOL-006-Consolidation-Pipeline.md) | EXTEND `services/memory-service/` + PostgreSQL | Wave 2 |
| CR-AM-007 | [SOL-007](./SOL-007-Governance-Audit-Diagnostics.md) | EXTEND `services/memory-service/` + `services/vnp-platform/` | Wave 4 |
| CR-AM-008 | [SOL-008](./SOL-008-Context-Injection-MCP.md) | EXTEND `gateway/` MCP adapter (16→53 tools) | Wave 2 |

---

## Architecture Fit

VNP Memory dùng **Monolith mode** (35 in-process services qua `bufconn` gRPC). Tất cả AgentMemory services mới sẽ được:

1. **Đăng ký vào `InProcessRegistry`** — gRPC in-memory transport, không có network hop.
2. **Khởi tạo trong `apps/memory/internal/bootstrap/`** — cùng pattern với `bootstrap/cognee.go`, `bootstrap/zep.go`, v.v.
3. **Subscribe NATS subjects** — dùng embedded NATS JetStream (stream `agentmemory.*`).
4. **Share PostgreSQL** — thêm tables mới vào existing DB connection pool.

### New Services Added to Monolith (35 → 38)

| Service ID | Name | Port (gRPC/bufconn) |
|------------|------|---------------------|
| 36 | `am-observe` | bufconn (internal) |
| 37 | `am-search` | bufconn (internal) |
| 38 | `am-orchestration` | bufconn (internal) |

---

## NATS Stream: `agentmemory`

New subjects added to VNP Memory NATS:

| Subject | Publisher | Consumers |
|---------|-----------|-----------|
| `agentmemory.session.started` | observe-service | consolidation, audit |
| `agentmemory.session.ended` | observe-service | consolidation (trigger summarize) |
| `agentmemory.observation.captured` | observe-service | search (index), consolidation |
| `agentmemory.memory.remembered` | memory-service | search (reindex) |
| `agentmemory.memory.superseded` | memory-service | audit |
| `agentmemory.memory.expired` | memory-service | search (remove), audit |
| `agentmemory.action.completed` | orchestration-service | sentinel checker |
| `agentmemory.signal.sent` | orchestration-service | target agent |
| `agentmemory.checkpoint.resolved` | orchestration-service | audit |

---

## Gateway Route Groups Added

| Prefix | Target Service | CR |
|--------|---------------|-----|
| `/v1/observe` | am-observe | CR-001 |
| `/v1/sessions` | am-observe | CR-001, CR-005 |
| `/v1/stream` | am-observe (SSE) | CR-005 |
| `/v1/observe/search/*` | am-search | CR-003 |
| `/v1/memory/agent/*` | memory-service (extended) | CR-002 |
| `/v1/memory/slots/*` | memory-service (extended) | CR-002 |
| `/v1/memory/compress` | memory-service (extended) | CR-006 |
| `/v1/memory/summarize` | memory-service (extended) | CR-006 |
| `/v1/memory/procedural` | memory-service (extended) | CR-006 |
| `/v1/memory/lessons` | memory-service (extended) | CR-006 |
| `/v1/memory/audit` | vnp-platform (extended) | CR-007 |
| `/v1/admin/doctor` | vnp-platform (extended) | CR-007 |
| `/v1/admin/snapshot` | vnp-platform (extended) | CR-007 |
| `/v1/admin/plugin/*` | vnp-platform (extended) | CR-008 |
| `/v1/orchestration/*` | am-orchestration | CR-004 |

---

## PostgreSQL Tables Added

| Table | CR | Service |
|-------|----|---------|
| `agent_sessions` | CR-001 | am-observe |
| `agent_raw_observations` | CR-001 | am-observe |
| `agent_compressed_observations` | CR-001 | am-observe |
| `agent_memories` | CR-002 | memory-service |
| `memory_slots` | CR-002 | memory-service |
| `agent_actions` | CR-004 | am-orchestration |
| `agent_leases` | CR-004 | am-orchestration |
| `agent_signals` | CR-004 | am-orchestration |
| `agent_routines` | CR-004 | am-orchestration |
| `agent_checkpoints` | CR-004 | am-orchestration |
| `agent_sentinels` | CR-004 | am-orchestration |
| `agent_sketches` | CR-004 | am-orchestration |
| `agent_crystals` | CR-004 | am-orchestration |
| `session_summaries` | CR-006 | memory-service |
| `procedural_memories` | CR-006 | memory-service |
| `lessons` | CR-006 | memory-service |
| `insights` | CR-006 | memory-service |
| `audit_entries` | CR-007 | vnp-platform |

---

## Implementation Status

| Solution | Status | Build | Key Files |
|----------|--------|-------|-----------|
| SOL-001 Observe Service | ✅ Implemented | ✅ | `services/observe-service/` |
| SOL-002 Memory Lifecycle | ✅ Implemented | ✅ | `services/memory-service/internal/usecase/agentmemory/` |
| SOL-003 Hybrid Search Engine | ✅ Implemented | ✅ | `services/observe-search/`, `pkg/search/` |
| SOL-004 Multi-Agent Orchestration | ✅ Implemented | ✅ | `services/orchestration-service/` |
| SOL-005 Session Replay | ✅ Implemented | ✅ | `services/observe-service/internal/replay/` |
| SOL-006 Consolidation Pipeline | ✅ Implemented | ✅ | `services/memory-service/internal/consolidation/` |
| SOL-007 Governance Audit & Diagnostics | ✅ Implemented | ✅ | `services/memory-service/internal/domain/agentmemory/audit.go`, `services/vnp-platform/` |
| SOL-008 Context Injection MCP | ✅ Implemented | ✅ | `gateway/internal/adapter/mcp/tools/agentmemory/` |

**Last updated:** 2026-06-17
