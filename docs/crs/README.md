# VNP Memory — Change Request Catalog

> Toàn bộ Change Requests của VNP Memory, phân theo version và tính năng.
> **Format:** Mỗi CR = 1 file, mô tả vấn đề, API contract, code changes, acceptance criteria.

---

## Versioning Strategy

```
v0/  — UI Console (Dashboard, Auth, Sessions, Profiles...)       [Existing]
v1/  — Engine Implementation (Cognee, Graphiti, Memobase, ...)   [Existing]
v2/  — API Alignment (Auth API, Org SDK, Sessions, Schemas)      [Existing]
v3/  — Core Memory & Integration (Unified API, Recall, GDPR...) [New — MVP]
v4/  — Intelligence Layer (Profiling, Context, Knowledge...)     [New — Diff]
v5/  — Enterprise & Operations (Multi-agent, Governance...)      [New — Scale]
```

---

## v0 — UI Console

> Console UI change requests — Dashboard, Auth, Sessions, Memory Explorer, Governance

| CR | Title |
|---|---|
| [CR-000-OVERVIEW](./v0/ui/CR-000-OVERVIEW.md) | Console UI Overview |
| [CR-001-AUTH](./v0/ui/CR-001-AUTH.md) | Authentication UI |
| [CR-002-DASHBOARD](./v0/ui/CR-002-DASHBOARD.md) | Dashboard |
| [CR-003-SESSIONS](./v0/ui/CR-003-SESSIONS.md) | Sessions Explorer |
| [CR-004-MEMORY](./v0/ui/CR-004-MEMORY.md) | Memory Management |
| [CR-005-ADAPTIVE](./v0/ui/CR-005-ADAPTIVE.md) | Adaptive Memory UI |
| [CR-006-PROFILES](./v0/ui/CR-006-PROFILES.md) | User Profiles |
| [CR-007-GOVERNANCE](./v0/ui/CR-007-GOVERNANCE.md) | Governance Center |
| [CR-008-OBSERVABILITY](./v0/ui/CR-008-OBSERVABILITY.md) | Observability |
| [CR-009-PIPELINES](./v0/ui/CR-009-PIPELINES.md) | Pipelines |
| [CR-010-INFRASTRUCTURE](./v0/ui/CR-010-INFRASTRUCTURE.md) | Infrastructure |
| [CR-011-ORG-SDK](./v0/ui/CR-011-ORG-SDK.md) | Org & SDK Settings |

---

## v1 — Engine Implementation

> Engine-level CRs — mỗi engine có folder riêng với implementation specs.

| Folder | Engine | CRs |
|---|---|---|
| [v1/agentmemory/](./v1/agentmemory/README.md) | AgentMemory Layer | 8 CRs (Observe, Lifecycle, Search, Orchestration...) |
| [v1/cognee/](./v1/cognee/README.md) | Cognee (Semantic) | 7 CRs |
| [v1/graphiti/](./v1/graphiti/README.md) | Graphiti (Episodic) | 7 CRs |
| [v1/memobase/](./v1/memobase/README.md) | Memobase (Profile) | 7 CRs |
| [v1/openviking/](./v1/openviking/README.md) | OpenViking (Procedural) | 7 CRs |
| [v1/supermemory/](./v1/supermemory/README.md) | Supermemory (Adaptive) | 10 CRs |
| [v1/zep/](./v1/zep/README.md) | Zep (Conversational) | 9 CRs |

---

## v2 — API Alignment

> API contract alignment giữa backend và frontend/external clients.

| CR | Title |
|---|---|
| [CR-000-index](./v2/api-update/CR-000-index.md) | API Gap Index |
| [CR-001-auth-api](./v2/api-update/CR-001-auth-api.md) | Auth API |
| [CR-002-org-sdk-api](./v2/api-update/CR-002-org-sdk-api.md) | Org & SDK API |
| [CR-003-session-query-params](./v2/api-update/CR-003-session-query-params.md) | Session Query Params |
| [CR-004-response-schema-contracts](./v2/api-update/CR-004-response-schema-contracts.md) | Response Schema |

---

## v3 — Core Memory & Integration 🆕

> **Phase:** MVP Completion — Unified API, Cross-engine recall, GDPR, Temporal reasoning, MCP
> **Pain Points giải quyết:** P1-01, P1-02, P1-03, P4-02, P5-01, P6-01, P6-03
> [→ v3 README](./v3/core-memory/README.md)

| CR | Title | Priority | Solution |
|---|---|---|---|
| [CR-CORE-001](./v3/core-memory/CR-CORE-001-Unified-Memory-Router.md) | Unified Memory Router — Auto-type & routing | 🔴 Critical | S2 |
| [CR-CORE-002](./v3/core-memory/CR-CORE-002-Cross-Engine-Recall.md) | Cross-Engine Recall — RRF Hybrid Search | 🔴 Critical | S2 |
| [CR-CORE-003](./v3/core-memory/CR-CORE-003-Cascading-Forget.md) | Cascading Forget — GDPR delete 6 engines | 🔴 Critical | S9 |
| [CR-CORE-004](./v3/core-memory/CR-CORE-004-Persistent-Session-Context.md) | Persistent Session Context — cross-session | 🟡 High | S1 |
| [CR-CORE-005](./v3/core-memory/CR-CORE-005-Temporal-Reasoning.md) | Temporal Reasoning — isLatest versioning | 🟡 High | S3 |
| [CR-CORE-006](./v3/core-memory/CR-CORE-006-MCP-Server.md) | MCP Server — 37+ tools, dual transport | 🟡 High | S2 |

---

## v4 — Intelligence Layer 🆕

> **Phase:** Differentiation — User profiling, context efficiency, knowledge evolution, observability
> **Pain Points giải quyết:** P1-04, P1-05, P1-06, P1-07, P3-03, P7-01, P7-02, P7-03
> [→ v4 README](./v4/intelligence/README.md)

| CR | Title | Priority | Solution |
|---|---|---|---|
| [CR-INTEL-001](./v4/intelligence/CR-INTEL-001-User-Profile-Assembly.md) | User Profile Assembly — context < 100ms | 🔴 Critical | S5 |
| [CR-INTEL-002](./v4/intelligence/CR-INTEL-002-Tiered-Context-Injection.md) | Tiered Context Injection — L0/L1/L2 | 🔴 Critical | S6 |
| [CR-INTEL-003](./v4/intelligence/CR-INTEL-003-Knowledge-Evolution.md) | Knowledge Evolution — contradiction resolution | 🟡 High | S4 |
| [CR-INTEL-004](./v4/intelligence/CR-INTEL-004-Memory-Decay-Eviction.md) | Memory Decay & Salience Eviction | 🟡 High | S4 |
| [CR-INTEL-005](./v4/intelligence/CR-INTEL-005-Session-Replay.md) | Session Replay — JSONL import, timeline | 🟡 High | S7 |
| [CR-INTEL-006](./v4/intelligence/CR-INTEL-006-Agent-Context-Debugger.md) | Agent Context Debugger — trace per call | 🟠 Medium | S7 |

---

## v5 — Enterprise & Operations 🆕

> **Phase:** Scale & Compliance — Multi-agent, governance, observability, infrastructure
> **Pain Points giải quyết:** P1-08, P1-09, P2-01, P2-02, P2-03, P4-01, P4-02, P4-03, P8-02
> [→ v5 README](./v5/enterprise/README.md)

| CR | Title | Priority | Solution |
|---|---|---|---|
| [CR-ENT-001](./v5/enterprise/CR-ENT-001-Distributed-Leases.md) | Distributed Lease System — multi-agent | 🟡 High | S8 |
| [CR-ENT-002](./v5/enterprise/CR-ENT-002-Consolidation-Pipeline.md) | Consolidation Pipeline — 4-tier sleep model | 🟡 High | S6 |
| [CR-ENT-003](./v5/enterprise/CR-ENT-003-Governance-Center.md) | Governance Center — GDPR, OPA, Audit | 🔴 Critical | S9 |
| [CR-ENT-004](./v5/enterprise/CR-ENT-004-MultiTenant-Isolation.md) | Multi-Tenant Isolation — TenantID zero-leak | 🔴 Critical | S9 |
| [CR-ENT-005](./v5/enterprise/CR-ENT-005-Unified-Observability.md) | Unified Observability — metrics, LLM cost | 🟡 High | S10 |
| [CR-ENT-006](./v5/enterprise/CR-ENT-006-Infrastructure-Health.md) | Infrastructure Health — aggregated /healthz | 🟠 Medium | S10 |

---

## Pain Point → CR Traceability Matrix

| Pain Point | Actor | Giải quyết bởi CR(s) |
|---|---|---|
| PP-P1-01 Memory mất sau session | AI Agent Dev | CR-CORE-004 |
| PP-P1-02 Memory fragmented 6 APIs | AI Agent Dev | CR-CORE-001, CR-CORE-002 |
| PP-P1-03 RAG không hiểu thời gian | AI Agent Dev | CR-CORE-005 |
| PP-P1-04 Knowledge không tự update | AI Agent Dev | CR-INTEL-003 |
| PP-P1-05 Không biết user preferences | AI Agent Dev | CR-INTEL-001 |
| PP-P1-06 Context cost $0.50/call | AI Agent Dev | CR-INTEL-002 |
| PP-P1-07 Không debug agent decisions | AI Agent Dev | CR-INTEL-005, CR-INTEL-006 |
| PP-P1-08 Multi-agent race conditions | AI Agent Dev | CR-ENT-001 |
| PP-P1-09 Memory bùng nổ | AI Agent Dev | CR-INTEL-004, CR-ENT-002 |
| PP-P2-01 35+ health endpoints | Platform Eng | CR-ENT-006 |
| PP-P2-02 Không monitor latency, cost | Platform Eng | CR-ENT-005 |
| PP-P2-03 Cross-tenant leak risk | Platform Eng | CR-ENT-004 |
| PP-P4-01 Không biết AI nhớ gì | Enterprise Arch | CR-ENT-003 |
| PP-P4-02 GDPR forget manual | Enterprise Arch | CR-CORE-003, CR-ENT-003 |
| PP-P4-03 Không có audit trail | Enterprise Arch | CR-ENT-003, CR-ENT-004 |
| PP-P5-01 IDE brief AI mỗi sáng | IDE Plugin User | CR-CORE-004 |
| PP-P5-02 IDE verbose MCP | IDE Plugin User | CR-CORE-006 |
| PP-P5-04 AI re-read codebase | IDE Plugin User | CR-INTEL-002 |
| PP-P6-01 No standard API | Framework Int | CR-CORE-001 |
| PP-P6-03 Manual context injection | Framework Int | CR-CORE-006 |
| PP-P7-01 AI không nhớ preferences | AI Power User | CR-INTEL-001 |
| PP-P7-02 AI nhớ thông tin cũ | AI Power User | CR-INTEL-003 |
| PP-P7-03 Memory không tự quên | AI Power User | CR-INTEL-004 |
| PP-P8-02 Không track LLM cost | Product Manager | CR-ENT-005 |

---

## Solution → CR Mapping

| Solution | CRs implement |
|---|---|
| S1 Persistent Memory | CR-CORE-004 |
| S2 Unified API | CR-CORE-001, CR-CORE-002, CR-CORE-006 |
| S3 Temporal Reasoning | CR-CORE-005 |
| S4 Knowledge Evolution | CR-INTEL-003, CR-INTEL-004 |
| S5 User Profiling | CR-INTEL-001 |
| S6 Context Efficiency | CR-INTEL-002, CR-ENT-002 |
| S7 Agent Observability | CR-INTEL-005, CR-INTEL-006 |
| S8 Multi-Agent Coord. | CR-ENT-001 |
| S9 Enterprise Governance | CR-CORE-003, CR-ENT-003, CR-ENT-004 |
| S10 Infrastructure | CR-ENT-005, CR-ENT-006 |
