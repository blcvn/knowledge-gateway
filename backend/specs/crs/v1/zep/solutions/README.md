# Solutions — Zep Feature Parity Change Requests

**Project:** VNP Memory  
**Domain:** Zep — End-to-End Context Engineering Platform  
**Path:** `specs/crs/v1/zep/solutions/`  
**Date:** 2026-06-17  
**Status:** Draft

> Các Solution documents này mô tả giải pháp kỹ thuật chi tiết để đáp ứng từng Change Request trong `specs/crs/v1/zep/`.  
> Differentiator cốt lõi của Zep: **Temporal Knowledge Graph** với `valid_at`/`invalid_at` — biết khi nào thông tin đúng, khi nào không còn đúng nữa. Sub-200ms context delivery.

---

## Danh sách Solutions

| Solution | CR | Tên | Loại | Wave | Ước tính |
|----------|-----|-----|------|------|---------|
| [SOL-ZEP-001](./SOL-ZEP-001-Thread-Session-Management.md) | CR-ZEP-001 | **Thread & Session Management** | New Service | 2 | 7 ngày |
| [SOL-ZEP-002](./SOL-ZEP-002-Memory-Message-Context-Assembly.md) | CR-ZEP-002 | **Memory: Message Ingestion & Context Assembly** | Upgrade Service | 3 | 8 ngày |
| [SOL-ZEP-003](./SOL-ZEP-003-Temporal-Knowledge-Graph.md) | CR-ZEP-003 | **Temporal Knowledge Graph** | New Service | 4 | 12.5 ngày |
| [SOL-ZEP-004](./SOL-ZEP-004-Semantic-Graph-Search-Reranking.md) | CR-ZEP-004 | **Semantic Graph Search & 5 Rerankers** | Upgrade Service | 4 | 12 ngày |
| [SOL-ZEP-005](./SOL-ZEP-005-MCP-Server-13-Tools.md) | CR-ZEP-005 | **MCP Server — 13 Read-Only Tools** | Upgrade Service | 5 | 7.5 ngày |
| [SOL-ZEP-006](./SOL-ZEP-006-Framework-Integrations.md) | CR-ZEP-006 | **Framework Integrations (AutoGen, CrewAI, ADK, LiveKit)** | New Packages | 5 | 12 ngày |
| [SOL-ZEP-007](./SOL-ZEP-007-Evaluation-Harness.md) | CR-ZEP-007 | **Evaluation Harness & Benchmarking** | New Tool | 6 | 10 ngày |
| [SOL-ZEP-008](./SOL-ZEP-008-Admin-Service-Multi-Tenant.md) | CR-ZEP-008 | **Admin Service & Multi-Tenant Management** | New Service | 2 | 8 ngày |
| [SOL-ZEP-009](./SOL-ZEP-009-Resilience-Observability.md) | CR-ZEP-009 | **Resilience & Observability Infrastructure** | Upgrade Shared | 1 | 8 ngày |

**Tổng ước tính:** ~85 ngày developer-days (6 Waves)

---

## Kiến trúc Tổng thể Giải pháp

```
External Clients
┌──────────────────────────────────────────────────────────────┐
│  Python/TS/Go SDK  │  MCP Server (stdio + HTTP Streamable)   │
│  AutoGen · CrewAI · Google ADK · LiveKit                     │
└──────────────────────────────┬───────────────────────────────┘
                               │ REST / MCP
               ┌───────────────▼────────────────────┐
               │     ZEP API GATEWAY (chi)           │
               │  10-Layer Middleware Stack (SOL-009) │
               │  Auth · RateLimit · Circuit Breaker  │
               └──┬──┬──┬──┬──┬──┬──────────────────┘
                  │  │  │  │  │  │  gRPC (in-process bufconn)
    ┌─────────────┘  │  │  │  │  └─────────────────────────┐
    │         ┌──────┘  │  │  └───────────────┐            │
    ▼         ▼         ▼  ▼                  ▼            ▼
┌────────┐ ┌────────┐ ┌──────┐ ┌────────┐ ┌──────┐ ┌────────┐
│ Thread │ │ Memory │ │Graph │ │ Search │ │ User │ │ Admin  │
│  :9042 │ │  :9043 │ │ :9044│ │  :9045 │ │ :9041│ │  :9046 │
│SOL-001 │ │SOL-002 │ │SOL-03│ │SOL-004 │ │      │ │SOL-008 │
└────────┘ └────────┘ └──────┘ └────────┘ └──────┘ └────────┘
                            │
                     NATS JetStream (async 10-20s)
                            │
                     ┌──────▼──────────────────────────────────┐
                     │  PostgreSQL 16 (pgvector)                │
                     │  Neo4j 5.22+ (temporal knowledge graph)  │
                     │  Redis 7 (cache TTL 30s)                 │
                     │  NATS JetStream (events + retry)         │
                     │  Graphiti service (Python, LLM extract)  │
                     └──────────────────────────────────────────┘
```

---

## Data Flow — Critical Path (sub-200ms)

```
Client → POST /api/v2/sessions/{id}/memory
   │
   ├─1. Gateway (middleware: 10 layers) ~5ms
   ├─2. Thread Service: UpsertSession (in-process bufconn) ~2ms
   ├─3. Session Lifecycle Guard (EndedAt check) ~0ms
   ├─4. Batch INSERT messages → PostgreSQL ~10ms
   └─5. Publish NATS "zep.memory.messages.ingested" (async, non-blocking) ~1ms
   └─ 200 OK ← returned to client (~18ms total)

           (async, 10-20 seconds separately)
           NATS consumer → Graph Service
               → Graphiti Python service (LLM extraction)
               → Neo4j upsert (nodes + edges + episodes)
               → NATS "zep.graph.extraction.completed"
                   → Search Service: invalidate Redis cache
```

---

## Key Design Decisions

| Quyết định | Lý do |
|-----------|-------|
| **Neo4j 5.22+** for temporal graph | Native vector index + temporal query support |
| **Graphiti** for entity extraction | Proven LLM extraction with temporal annotations |
| **NATS JetStream** async pipeline | Non-blocking PutMemory (sub-200ms) + at-least-once delivery |
| **Redis cache TTL 30s** for search | Temporal data needs freshness; NATS invalidation |
| **sony/gobreaker** circuit breaker | Battle-tested Go library; per-service configuration |
| **SHA-256 → int64** advisory lock | Deterministic; collision probability 1/(2^64) |
| **10-layer chi middleware** | Consistent across all Zep gateway routes |
| **vnp_ API key prefix** (CR-ZEP-008) | Distinguishable from other tokens in logs |
| **5 reranking strategies** | Different use cases: speed (RRF), diversity (MMR), accuracy (Cross-Encoder) |
| **13 MCP tools — read-only** | Safety by design; write via SDK/REST only |
| **tenacity retry** in eval harness | LLM rate limits; max 8 attempts, 4s→300s delay |
| **mypy strict** for Python integrations | Type safety for framework interfaces |

---

## Infrastructure Delta (Mới cần thêm)

> [!IMPORTANT]
> Zep yêu cầu **Neo4j 5.22+** và **Graphiti service** — đây là dependencies chưa có trong VNP Memory hiện tại.

| Component | Action | Mục đích |
|-----------|--------|----------|
| **Neo4j 5.22+** | UPGRADE từ version hiện tại | Temporal KG + native vector index |
| **Graphiti service** | NEW: Python container deploy | LLM entity extraction (async 10-20s) |
| **NATS JetStream** | Đã có embedded | Async graph extraction pipeline |

---

## Lộ trình Theo Wave

### Wave 1 — Foundation (SOL-ZEP-009)
- **Mục tiêu:** Circuit Breaker + Retry + 10-Layer Middleware + Advisory Lock Shared Package
- **Ước tính:** 8 ngày
- **Lý do:** Infrastructure foundation — tất cả services khác depend vào

### Wave 2 — Core CRUD (SOL-ZEP-001 + SOL-ZEP-008)
- **Mục tiêu:** Thread lifecycle management + Admin/Project management
- **Ước tính:** 7 + 8 = 15 ngày (có thể parallel)
- **Dependencies:** Wave 1 (middleware + advisory lock)

### Wave 3 — Memory Core (SOL-ZEP-002)
- **Mục tiêu:** Message ingestion (PutMemory) + Context Assembly (GetMemory) + GetUserContext
- **Ước tính:** 8 ngày
- **Dependencies:** Wave 2 (Thread Service UpsertSession + Session Lifecycle Guard)

### Wave 4 — Graph Intelligence (SOL-ZEP-003 + SOL-ZEP-004)
- **Mục tiêu:** Temporal KG extraction + Semantic Search với 5 rerankers
- **Ước tính:** 12.5 + 12 = 24.5 ngày (có thể parallel)
- **Dependencies:** Neo4j 5.22+ + Graphiti deploy + Wave 3 (NATS events from PutMemory)

### Wave 5 — Integration (SOL-ZEP-005 + SOL-ZEP-006)
- **Mục tiêu:** MCP 13 tools + AutoGen/CrewAI/ADK/LiveKit integrations
- **Ước tính:** 7.5 + 12 = 19.5 ngày (có thể parallel)
- **Dependencies:** Wave 3 (GetUserContext) + Wave 4 (search_graph)

### Wave 6 — Quality (SOL-ZEP-007)
- **Mục tiêu:** Evaluation Harness + LoCoMo + LongMemEval benchmarks
- **Ước tính:** 10 ngày
- **Dependencies:** Tất cả waves trước (cần full system để eval)
