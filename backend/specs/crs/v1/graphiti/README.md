# Change Requests — Graphiti Feature Parity

**Project:** VNP Memory  
**Domain:** Graphiti — Temporal Context Graph Engine  
**Path:** `specs/crs/v1/graphiti/`  
**Date:** 2026-06-16  
**Reference:** graphiti-core v0.28.2 (Zep Software)  
**Status:** Proposed

> Các Change Requests này được tạo từ phân tích đối chiếu giữa VNP Memory hiện tại và tài liệu tham chiếu:
> `references/graphiti/docs/PRD.md`, `SRS.md`, `URD.md`, `specs/services/*.md`.

---

## Tổng quan Change Requests

| CR ID | Tên | Loại | Priority | Status |
|---|---|---|---|---|
| [CR-GR-001](./CR-GR-001-Episode-Ingestion-Pipeline.md) | **Episode Ingestion Pipeline** (9-step orchestrator, Saga, Chunking) | 🆕 New Service | Critical | Proposed |
| [CR-GR-002](./CR-GR-002-Temporal-Knowledge-Graph-Store.md) | **Temporal Knowledge Graph Store** (bi-temporal model, 4 backends) | 🆕 New Service | Critical | Proposed |
| [CR-GR-003](./CR-GR-003-Knowledge-Processing-Service.md) | **Knowledge Processing Service** (LLM extraction, resolution, community) | 🆕 New Service | Critical | Proposed |
| [CR-GR-004](./CR-GR-004-Hybrid-Search-Engine.md) | **Hybrid Search Engine** (cosine + BM25 + BFS, 5 rerankers, temporal filters) | 🆕 New Service | Critical | Proposed |
| [CR-GR-005](./CR-GR-005-Custom-Ontology.md) | **Custom Ontology** (prescribed entity/edge types, domain presets) | Extend | High | Proposed |
| [CR-GR-006](./CR-GR-006-Gateway-MCP-Server.md) | **Gateway & MCP Server** (9 MCP tools, graphiti REST, tenant routing) | Extend | High | Proposed |
| [CR-GR-007](./CR-GR-007-Admin-Observability.md) | **Admin Service & Observability** (OTel tracing, token tracking, tenant mgmt) | 🆕 New Service | High | Proposed |

---

## Feature Gap Matrix

| Feature | Graphiti Spec | VNP Memory hiện tại | CR |
|---|---|---|---|
| **Episode Ingestion Pipeline** | | | |
| 9-step ingestion pipeline (chunk→extract→resolve→persist→community) | ✅ SRS §3.1 | ❌ Không có orchestration | CR-001 |
| Per-group-id sequential processing (prevent dedup races) | ✅ specs/02 | ❌ Không có | CR-001 |
| Content chunking (density-based, 3000 token, 200 overlap) | ✅ specs/02 §5 | ❌ Không có | CR-001 |
| Saga management (HAS_EPISODE + NEXT_EPISODE + incremental LLM summary) | ✅ PRD §5.1, SRS §3.1 | ❌ Không có | CR-001 |
| Direct triplet insertion (add_triplet API) | ✅ SRS §3.1 | ❌ Không có | CR-001 |
| Bulk episode processing (parallel extraction + in-memory dedup) | ✅ PRD §5.1 | ❌ Không có | CR-001 |
| Streaming gRPC for bulk (progress + partial failure) | ✅ specs/02 | ❌ Không có | CR-001 |
| Backpressure / queue full → 429 | ✅ specs/02 §4.3 | ❌ Không có | CR-001 |
| Episode cascade delete | ✅ SRS §3.1 | ❌ Không có | CR-001 |
| **Temporal Knowledge Graph** | | | |
| Bi-temporal model (valid_at / invalid_at / expired_at) | ✅ SRS §4.3 | ❌ Không đầy đủ | CR-002 |
| EntityNode (uuid, name, labels[], summary, attributes{}, name_embedding) | ✅ SRS §4.1 | ⚠️ Partial | CR-002 |
| EpisodicNode (content, source, valid_at, entity_edges, provenance) | ✅ SRS §4.1 | ⚠️ Partial | CR-002 |
| CommunityNode (name, summary, name_embedding) | ✅ SRS §4.1 | ❌ Không có | CR-002 |
| SagaNode (first/last_episode, last_summarized_at) | ✅ SRS §4.1 | ❌ Không có | CR-002 |
| EntityEdge (fact, fact_embedding, episodes provenance, valid_at, invalid_at, expired_at) | ✅ SRS §4.2 | ⚠️ Partial | CR-002 |
| 5-type edge model (Entity/Episodic/Community/HasEpisode/NextEpisode) | ✅ SRS §4.2 | ❌ Không đầy đủ | CR-002 |
| InvalidateEntityEdge (temporal invalidation, NOT delete) | ✅ SRS §3.6, PRD §3 | ❌ Không có | CR-002 |
| Atomic SaveBulk (episode+nodes+edges, single transaction) | ✅ specs/05 §3 | ❌ Không có | CR-002 |
| FalkorDB driver | ✅ PRD §5.3 | ❌ Không có | CR-002 |
| Kuzu driver | ✅ PRD §5.3 | ❌ Không có | CR-002 |
| Neptune driver | ✅ PRD §5.3 | ❌ Không có | CR-002 |
| BFS graph traversal search | ✅ PRD §5.2 | ❌ Không có | CR-002 |
| Node distance reranker (graph proximity) | ✅ SRS §3.6 | ❌ Không có | CR-002 |
| Episode mentions reranker | ✅ SRS §3.6 | ❌ Không có | CR-002 |
| Community cluster queries | ✅ specs/05 | ❌ Không có | CR-002 |
| **Knowledge Processing** | | | |
| Entity extraction (LLM, per source type: text/json/message) | ✅ SRS §3.3 | ⚠️ Partial | CR-003 |
| Entity resolution two-phase (deterministic + LLM) | ✅ specs/04 §4.2 | ❌ Không có | CR-003 |
| Edge extraction (fact triples, temporal validity) | ✅ SRS §3.3 | ⚠️ Partial | CR-003 |
| Edge resolution (DUPLICATE/NEW/CONTRADICTION/UPDATE) | ✅ specs/04 §4.4 | ❌ Không có | CR-003 |
| Attribute extraction (entity summary updates) | ✅ specs/04 | ❌ Không có | CR-003 |
| Community detection (Label Propagation + LLM summarization) | ✅ PRD §5.1 | ❌ Không có | CR-003 |
| Bulk extraction (cross-episode parallel) | ✅ specs/04 | ❌ Không có | CR-003 |
| Cross-encoder reranking (neural scoring) | ✅ PRD §5.2 | ❌ Không có | CR-003 |
| Anthropic, Gemini, Groq, Ollama adapters | ✅ PRD §5.4 | ⚠️ Partial | CR-003 |
| Bifrost gateway adapter (production) | ✅ specs/04 | ❌ Không có | CR-003 |
| LLM response caching (Redis) | ✅ SRS §7.2 | ❌ Không có | CR-003 |
| Prompt registry (versioned templates per use case) | ✅ specs/04 §6 | ❌ Không có | CR-003 |
| Multilingual extraction support | ✅ specs/04 §6.2 | ❌ Không có | CR-003 |
| Token usage tracking per prompt type | ✅ SRS §8.2 | ❌ Không có | CR-003 |
| Input sanitization (Unicode, control chars) | ✅ SRS §3.3 | ❌ Không có | CR-003 |
| Retry with exponential backoff (max 4, 5s-120s) | ✅ SRS §7.3 | ⚠️ Partial | CR-003 |
| **Hybrid Search** | | | |
| Cosine similarity (vector search) | ✅ PRD §5.2 | ✅ Có (Qdrant) | — |
| BM25 fulltext search | ✅ PRD §5.2 | ❌ Không có (graph-native) | CR-004 |
| BFS graph traversal | ✅ PRD §5.2 | ❌ Không có | CR-004 |
| RRF reranking | ✅ SRS §3.6 | ❌ Không có | CR-004 |
| MMR reranking | ✅ SRS §3.6 | ❌ Không có | CR-004 |
| Cross-encoder reranking | ✅ SRS §3.6 | ❌ Không có | CR-004 |
| Node distance reranking | ✅ SRS §3.6 | ❌ Không có | CR-004 |
| Episode mentions reranking | ✅ SRS §3.6 | ❌ Không có | CR-004 |
| Temporal filters (valid_at, invalid_at, created_at range) | ✅ PRD §5.2 | ❌ Không có | CR-004 |
| Entity label filters | ✅ SRS §3.6 | ❌ Không có | CR-004 |
| Search result caching (Redis + NATS invalidation) | ✅ specs/03 | ❌ Không có | CR-004 |
| Pre-built search recipes | ✅ URD §SOP-003 | ❌ Không có | CR-004 |
| Multi-type search (edges+nodes+episodes+communities simultaneously) | ✅ SRS §3.6 | ❌ Không có | CR-004 |
| Search latency < 1000ms target | ✅ PRD §8 | ❌ Không đo | CR-004 |
| **Custom Ontology** | | | |
| Prescribed entity types (EntityTypeSchema) | ✅ PRD §5.1 | ❌ Không có | CR-005 |
| Prescribed edge types (EdgeTypeSchema) | ✅ PRD §5.1 | ❌ Không có | CR-005 |
| Ontology constraint enforcement in LLM prompt | ✅ URD §SOP-005 | ❌ Không có | CR-005 |
| Per-tenant ontology registry | ✅ SRS §9 | ❌ Không có | CR-005 |
| Domain presets (HR, CRM, Software Project) | ✅ PRD | ❌ Không có | CR-005 |
| **Gateway & MCP** | | | |
| Full graphiti REST API (/episodes, /search, /entities, /edges, /sagas) | ✅ SRS §5.1 | ❌ Partial | CR-006 |
| MCP tools: add_memory, search_memory, get_episodes | ✅ SRS §5.2 | ⚠️ Partial | CR-006 |
| MCP tools: delete_episode, delete_entity_node, delete_entity_edge | ✅ SRS §5.2 | ❌ Không có | CR-006 |
| MCP tools: get_entity_edge, clear_graph, get_status | ✅ SRS §5.2 | ❌ Không có | CR-006 |
| /healthz aggregate health check | ✅ PRD §5.5 | ❌ Only {status: "ok"} | CR-006 |
| Multi-tenant X-Group-ID propagation to gRPC | ✅ specs/00 §8.4 | ⚠️ Partial | CR-006 |
| Rate limiting per-tenant per-endpoint | ✅ specs/00 §8.2 | ❌ Không có | CR-006 |
| **Admin & Observability** | | | |
| Tenant create/delete/list API | ✅ specs/06 | ❌ Không có | CR-007 |
| Community rebuild trigger (post-bulk-import) | ✅ specs/02 §4.2 | ❌ Không có | CR-007 |
| OpenTelemetry tracing (distributed, per pipeline step) | ✅ PRD §5.6 | ❌ Partial | CR-007 |
| Token usage reporting per prompt type | ✅ SRS §8.2 | ❌ Không có | CR-007 |
| Prometheus metrics (all services, standardized) | ✅ specs/00 §8.3 | ❌ Partial | CR-007 |
| Anonymous telemetry (opt-out via env var) | ✅ PRD §5.6 | ❌ Không có | CR-007 |
| NATS events (tenant.created, community.rebuilt, health.degraded) | ✅ specs/00 §3.3 | ❌ Không có | CR-007 |

---

## New Services to Build

| Service | Port | Maps to Python |
|---|---|---|
| `graphiti-ingestion` | 9001 (gRPC) | `graphiti.add_episode()` + `add_episode_bulk()` + `add_triplet()` |
| `graphiti-knowledge` | 9003 (gRPC) | `llm_client/`, `embedder/`, `cross_encoder/`, `prompts/` |
| `graphiti-search` | 9002 (gRPC) | `search/search.py`, `search_config_recipes.py` |
| `graphiti-store` | 9004 (gRPC) | `driver/` (Neo4j/FalkorDB/Kuzu/Neptune) |
| `graphiti-admin` | 9005 (gRPC) | `utils/maintenance/`, `telemetry.py` |

## Services to Extend

| Service | Changes |
|---|---|
| `gateway` | Graphiti REST routes, 9 MCP tools, healthz aggregate, rate limiting |

---

## Dependency Graph

```
CR-002 (Store)          ← Foundation (no dependencies)
    ↑
CR-003 (Knowledge)      ← Depends on Store (entity resolution search)
    ↑
CR-001 (Ingestion)      ← Depends on Knowledge + Store
    ↑
CR-004 (Search)         ← Depends on Knowledge (embedding) + Store (queries)
    ↑
CR-005 (Ontology)       ← Extends Knowledge + Ingestion
CR-006 (Gateway/MCP)    ← Routes to Ingestion + Search + Store + Admin
CR-007 (Admin)          ← Depends on Knowledge + Store + Search
```

---

## Architecture Diagram (Target State)

```
External Clients (REST / MCP / AI Agents)
               │
         ┌─────▼──────────────────────────────────────────────────┐
         │  graphiti-gateway (REST :8080, MCP :8082, gRPC :8081)  │
         │  JWT auth │ Rate limiting │ group_id propagation       │
         └─────┬──────────────────────────────────────────────────┘
               │ gRPC
       ┌───────┼────────────────────────────────────┐
       │       │                                    │
   ┌───▼────┐ ┌▼────────┐ ┌──────────┐ ┌─────────┐ │
   │ingest  │ │search   │ │knowledge │ │ admin   │ │
   │:9001   │ │:9002    │ │:9003     │ │:9005    │ │
   └───┬────┘ └────┬────┘ └────┬─────┘ └────┬────┘ │
       │           │           │             │      │
       └───────────┴───────────┴─────────────┘      │
                               │                    │
                        ┌──────▼──────┐             │
                        │   store     │◄────────────┘
                        │   :9004     │
                        └──────┬──────┘
                               │
              ┌────────────────┼──────────────────┐
              │                │                  │
           Neo4j          FalkorDB             Kuzu
           :7687           :6379             (embedded)
                               +
                     ┌─────────┴──────────┐
                     │  Shared Infra      │
                     │  Redis :6379       │
                     │  NATS  :4222       │
                     │  OTel  :4317       │
                     └────────────────────┘
```

---

## Recommended Implementation Order

| Wave | CRs | Rationale |
|---|---|---|
| **Wave 1** (Foundation) | CR-002 (Store), CR-003 (Knowledge) | All other services depend on these two |
| **Wave 2** (Core Pipeline) | CR-001 (Ingestion), CR-004 (Search) | Core functionality: ingest + query |
| **Wave 3** (Developer UX) | CR-006 (Gateway/MCP), CR-005 (Ontology) | External API surface + domain modeling |
| **Wave 4** (Operations) | CR-007 (Admin/Observability) | Production readiness: tracing, metrics, tenant mgmt |

---

## Performance Targets (from graphiti PRD §8)

| Metric | Target |
|---|---|
| Search latency (hybrid BM25+cosine+RRF) | < 1000ms |
| Ingestion throughput | Configurable via semaphore (10-50 concurrent LLM calls) |
| Embedding dimensions | 1536 (OpenAI default), configurable per provider |
| Max concurrent group workers | 100 (per ingestion service) |
| LLM retry: max attempts | 4 |
| LLM retry: backoff | 5s initial, 120s max |
