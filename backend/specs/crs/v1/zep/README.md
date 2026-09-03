# Change Requests — Zep Feature Parity

**Project:** VNP Memory  
**Domain:** Zep — End-to-End Context Engineering Platform  
**Path:** `specs/crs/v1/zep/`  
**Date:** 2026-06-16  
**Status:** Proposed

> Các Change Requests này được tạo từ phân tích đối chiếu giữa VNP Memory hiện tại và tài liệu Zep (`PRD v1.0`, `SRS v1.0`, `URD v1.0`, `specs/services`).
>
> **Zep's Core Value:** Biến conversation history và business data thành **pre-formatted, relationship-aware context blocks** với temporal reasoning, được deliver trong **sub-200ms**.
>
> **Differentiator so với các hệ thống khác:** Temporal Knowledge Graph với `valid_at`/`invalid_at` — biết khi nào thông tin đúng, khi nào không còn đúng nữa.

---

## Danh sách Change Requests

| CR ID | Tên | Loại | Priority | Status |
|---|---|---|---|---|
| [CR-ZEP-001](./CR-ZEP-001-Thread-Session-Management.md) | **Thread & Session Management** (lifecycle, ended_at, user association, advisory locks) | 🆕 New Service | Critical | Proposed |
| [CR-ZEP-002](./CR-ZEP-002-Memory-Message-Context-Assembly.md) | **Memory — Message Ingestion & Context Assembly** (PutMemory, GetMemory, 6 role types, GetUserContext) | ⬆️ Upgrade | Critical | Proposed |
| [CR-ZEP-003](./CR-ZEP-003-Temporal-Knowledge-Graph.md) | **Temporal Knowledge Graph** (9-node ontology, valid_at/invalid_at, Graphiti, Neo4j, custom ontology) | 🆕 New Service | Critical | Proposed |
| [CR-ZEP-004](./CR-ZEP-004-Semantic-Graph-Search-Reranking.md) | **Semantic Graph Search** (multi-scope, 5 rerankers: RRF/MMR/Cross-Encoder/NodeDist/EpisodeMentions) | ⬆️ Upgrade | Critical | Proposed |
| [CR-ZEP-005](./CR-ZEP-005-MCP-Server-13-Tools.md) | **MCP Server 13 Tools** (search_graph, get_user_context, graph exploration, stdio+HTTP) | ⬆️ Upgrade | High | Proposed |
| [CR-ZEP-006](./CR-ZEP-006-Framework-Integrations.md) | **Framework Integrations** (AutoGen, CrewAI dual storage, ADK, LiveKit, tool factories) | 🆕 New Packages | High | Proposed |
| [CR-ZEP-007](./CR-ZEP-007-Evaluation-Harness-Benchmarking.md) | **Evaluation Harness** (LoCoMo/LongMemEval benchmarks, completeness + accuracy metrics) | 🆕 New Tool | Medium | Proposed |
| [CR-ZEP-008](./CR-ZEP-008-Admin-Service-Multi-Tenant.md) | **Admin Service & Multi-Tenant** (project management, health aggregation, API keys, telemetry) | 🆕 New Service | Medium | Proposed |
| [CR-ZEP-009](./CR-ZEP-009-Resilience-Observability.md) | **Resilience & Observability** (circuit breaker, advisory locks, 10-layer middleware, telemetry) | ⬆️ Upgrade Shared | Medium | Proposed |

---

## Kiến trúc mục tiêu (Zep Context Engine trên VNP Memory)

```
External Clients: Python/TS/Go SDK · REST API · MCP Server (stdio+HTTP)
                  AutoGen · CrewAI · Google ADK · LiveKit
                                        │
                    ┌───────────────────▼───────────────────┐
                    │         ZEP API GATEWAY (Go)            │
                    │  Auth · RateLimit · 10-Layer Middleware  │
                    │  MCP Server (13 read-only tools)         │
                    └──┬────┬────┬────┬────┬────┬─────────────┘
                       │    │    │    │    │    │  gRPC (internal)
          ┌────────────┘    │    │    │    │    └────────────────┐
          ▼                 ▼    ▼    ▼    ▼                     ▼
    ┌──────────┐  ┌───────────┐ ┌──────┐ ┌──────┐  ┌──────┐ ┌───────┐
    │  Thread  │  │  Memory   │ │Graph │ │Search│  │User  │ │Admin  │
    │  Service │  │  Service  │ │ Svc  │ │ Svc  │  │ Svc  │ │ Svc   │
    │  :9042   │  │  :9043    │ │:9044 │ │:9045 │  │:9041 │ │:9046  │
    └──────────┘  └───────────┘ └──────┘ └──────┘  └──────┘ └───────┘
         │               │        │         │
         └───────────────┼────────┼─────────┘
                         │  NATS JetStream (async graph extraction: 10-20s)
    ┌────────────────────▼────────▼──────────────────────────────────────┐
    │  Infrastructure:                                                    │
    │  PostgreSQL 16 (pgvector) | Neo4j 5.22+ | Redis 7 | NATS JetStream │
    │  Graphiti Service (Python, LLM entity extraction)                   │
    └────────────────────────────────────────────────────────────────────┘
```

---

## Tính năng cốt lõi độc đáo của Zep

| Feature | Mô tả | CR liên quan |
|---------|-------|-------------|
| **Temporal Facts** | `valid_at`/`invalid_at` — track khi nào fact đúng/sai | CR-ZEP-003 |
| **Session Lifecycle Guard** | `ended_at` — ngăn add messages vào session đã kết thúc | CR-ZEP-001 |
| **Graceful Degradation** | GetMemory trả về messages kể cả khi Graph service down | CR-ZEP-002 |
| **Context Assembly** | Pre-formatted context blocks ready for LLM injection | CR-ZEP-002 |
| **5 Rerankers** | RRF, MMR, Cross-Encoder, NodeDistance, EpisodeMentions | CR-ZEP-004 |
| **13 MCP Tools** | Đầy đủ graph exploration read-only tools | CR-ZEP-005 |
| **Dual CrewAI Storage** | Per-user memory + shared knowledge graph | CR-ZEP-006 |
| **Advisory Locks** | SHA-256 hash-based PostgreSQL locks cho concurrent updates | CR-ZEP-009 |

---

## Infrastructure mới cần thêm

> [!IMPORTANT]
> Zep yêu cầu **Neo4j 5.22+** và **Graphiti service** — đây là dependencies mới không có trong VNP Memory hiện tại.

| Component | Loại | Mục đích |
|-----------|------|---------|
| **Neo4j 5.22+** | Graph DB | Lưu temporal knowledge graph (nodes, edges, episodes) |
| **Graphiti service** | Python service | LLM-powered entity extraction (async 10-20s) |
| **NATS JetStream** | Message Queue | Async graph extraction pipeline |

---

## Lộ trình triển khai

| Wave | CR | Mô tả |
|------|-----|-------|
| **Wave 1 — Foundation** | CR-009 | Resilience & Observability infrastructure |
| **Wave 2 — Core CRUD** | CR-001, CR-008 | Thread management + Admin/Projects |
| **Wave 3 — Memory Core** | CR-002 | Message ingestion & context assembly |
| **Wave 4 — Graph Intelligence** | CR-003, CR-004 | Temporal KG + Semantic Search |
| **Wave 5 — Integration** | CR-005, CR-006 | MCP 13 tools + Framework integrations |
| **Wave 6 — Quality** | CR-007 | Eval harness + LoCoMo/LongMemEval benchmarks |
