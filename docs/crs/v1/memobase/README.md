# Change Requests — Memobase Feature Parity

**Project:** VNP Memory  
**Domain:** Memobase — User Profile-Based Long-Term Memory System  
**Path:** `specs/crs/v1/memobase/`  
**Date:** 2026-06-16  
**Reference:** memobase v0.0.40 (memodb-io/memobase)  
**Status:** Proposed

> Các Change Requests này được tạo từ phân tích đối chiếu giữa VNP Memory hiện tại và tài liệu tham chiếu:
> `references/memobase/docs/PRD.md`, `SRS.md`, `URD.md`, `specs/services/*.md`.

---

## Tổng quan Change Requests

| CR ID | Tên | Loại | Priority | Status |
|---|---|---|---|---|
| [CR-MB-001](./CR-MB-001-Blob-Ingestion-Buffer-Zone.md) | **Blob Ingestion & Buffer Zone** (ChatBlob/DocBlob/SummaryBlob, FSM, auto-flush) | 🆕 New Service | Critical | Proposed |
| [CR-MB-002](./CR-MB-002-Memory-Engine-Profile-YOLO.md) | **Memory Engine: Profile & YOLO Merge** (3-call pipeline, event gist, tagging) | 🆕 New Service | Critical | Proposed |
| [CR-MB-003](./CR-MB-003-Context-Service.md) | **Context Service** (profile read, Redis cache, context assembly, token budget) | 🆕 New Service | Critical | Proposed |
| [CR-MB-004](./CR-MB-004-Event-Timeline-Semantic-Search.md) | **Event Timeline & Semantic Search** (pgvector HNSW, gist search, tag filter) | 🆕 New Service | High | Proposed |
| [CR-MB-005](./CR-MB-005-Admin-Service.md) | **Admin Service** (user CRUD, project config, billing, multi-tenant) | 🆕 New Service | High | Proposed |
| [CR-MB-006](./CR-MB-006-Gateway-MCP-Server.md) | **Gateway & MCP Server** (3 MCP tools, memobase REST, CORS, rate limit) | Extend | High | Proposed |
| [CR-MB-007](./CR-MB-007-Shared-Infrastructure.md) | **Shared Infrastructure** (LLM adapters, tiktoken, prompts, OTel, circuit breaker) | 🆕 New Packages | High | Proposed |

---

## Feature Gap Matrix

| Feature | Memobase Spec | VNP Memory hiện tại | CR |
|---|---|---|---|
| **Blob Ingestion** | | | |
| ChatBlob (OpenAI-compatible messages format) | ✅ SRS §3.3 | ⚠️ Partial | CR-001 |
| DocBlob (document text) | ✅ PRD §5.2 | ❌ Không có | CR-001 |
| SummaryBlob (pre-made summary) | ✅ PRD §5.2 | ❌ Không có | CR-001 |
| Buffer Zone FSM (idle→processing→done/failed) | ✅ SRS §3.4 | ❌ Không có | CR-001 |
| Token-size counting via tiktoken | ✅ SRS §3.3 | ❌ Không có | CR-001 |
| Auto-flush on token threshold (1024 tokens) | ✅ PRD §5.2, URD US-012 | ❌ Không có | CR-001 |
| Auto-flush on idle timeout (1 hour) | ✅ PRD §5.2, URD US-012 | ❌ Không có | CR-001 |
| Manual flush API (POST /buffer/{user_id}) | ✅ SRS §3.4 | ❌ Không có | CR-001 |
| Status-based concurrency lock (prevent duplicate flush) | ✅ SRS §3.4 | ❌ Không có | CR-001 |
| persistent_chat_blobs mode (delete after processing) | ✅ PRD §9.1 | ❌ Không có | CR-001 |
| Buffer capacity API | ✅ SRS §3.4 | ❌ Không có | CR-001 |
| NATS memobase.buffer.ready event | ✅ specs/00 §7.2 | ❌ Không có | CR-001 |
| **Memory Engine (Profile)** | | | |
| Fixed 3-call LLM pipeline (entry_summary + extract + YOLO merge) | ✅ PRD §5.3, SRS §3.5 | ❌ Không có | CR-002 |
| YOLO Merge (add/update/delete in 1 LLM call) | ✅ PRD §5.3, SRS §3.5 | ❌ Không có | CR-002 |
| Parallel processing (errgroup: profile + event goroutines) | ✅ specs/03 §4 | ❌ Không có | CR-002 |
| Profile topic/sub_topic/content structure | ✅ SRS §3.5 | ⚠️ Partial | CR-002 |
| Profile strict mode (only collect defined schema) | ✅ PRD §5.3 | ❌ Không có | CR-002 |
| Profile validate mode (remove meaningless slots) | ✅ PRD §5.3 | ❌ Không có | CR-002 |
| Profile organize (auto-reorganize when subtopics > max) | ✅ specs/03 §4 | ❌ Không có | CR-002 |
| Profile re-summary (shrink oversized slots) | ✅ specs/03 §4 | ❌ Không có | CR-002 |
| Custom profile schema (additional_user_profiles per project) | ✅ PRD §5.3, URD US-021 | ❌ Không có | CR-002 |
| max_profile_subtopics enforcement (default 15) | ✅ PRD §9.1 | ❌ Không có | CR-002 |
| max_slot_token_size enforcement (default 128) | ✅ PRD §9.1 | ❌ Không có | CR-002 |
| Event gist generation (split event_tip into fine-grained lines) | ✅ PRD §5.4 | ❌ Không có | CR-002 |
| Event tagging (custom tags: emotion, goal, etc.) | ✅ PRD §5.4 | ❌ Không có | CR-002 |
| Multilingual prompts (EN/ZH template registry) | ✅ PRD §3.2 G-6 | ❌ Không có | CR-002 |
| max_process_token_size truncation (16384 tokens) | ✅ specs/03 §4 | ❌ Không có | CR-002 |
| **Context Service** | | | |
| Redis profile caching (TTL 20min) | ✅ PRD §5.5, SRS §3.6 | ❌ Không có | CR-003 |
| NATS profile.changed → cache invalidation | ✅ specs/00 §7.2 | ❌ Không có | CR-003 |
| Context Assembly (prompt-ready string) | ✅ PRD §5.5, SRS §3.8 | ❌ Không có | CR-003 |
| max_token_size budget enforcement (tiktoken) | ✅ SRS §3.8 | ❌ Không có | CR-003 |
| profile_event_ratio (default 0.7) | ✅ SRS §3.8 | ❌ Không có | CR-003 |
| prefer_topics priority ordering | ✅ SRS §3.6, URD US-041 | ❌ Không có | CR-003 |
| only_topics filtering | ✅ SRS §3.6, URD US-041 | ❌ Không có | CR-003 |
| topic_limits (per-topic max profiles) | ✅ SRS §3.6 | ❌ Không có | CR-003 |
| Custom context template (customize_context_prompt) | ✅ PRD §5.5, URD US-042 | ❌ Không có | CR-003 |
| Parallel fetch (profiles + events concurrently) | ✅ specs/04 §4 | ❌ Không có | CR-003 |
| Context API < 100ms P99 latency target | ✅ PRD §3.2 G-3 | ❌ Not measured | CR-003 |
| Manual profile CRUD (add/update/delete individual slots) | ✅ SRS §3.6, URD US-022 | ❌ Không có | CR-003 |
| **Event Timeline** | | | |
| pgvector cosine similarity event search | ✅ SRS §3.7, URD US-031 | ❌ Không có | CR-004 |
| Event gist semantic search (fine-grained) | ✅ PRD §5.4, SRS §3.7 | ❌ Không có | CR-004 |
| Event tag filtering (JSONB @> containment) | ✅ SRS §3.7, URD US-032 | ❌ Không có | CR-004 |
| time_range_in_days filter (default 21 days) | ✅ SRS §3.7, URD US-034 | ❌ Không có | CR-004 |
| Similarity threshold filter (default 0.2) | ✅ SRS §3.7 | ❌ Không có | CR-004 |
| HNSW vector index (m=16, ef_construction=200) | ✅ specs/05 §7 | ❌ Không có | CR-004 |
| Embedding dimension validation at startup | ✅ SRS §4.2 NFR-R04 | ❌ Không có | CR-004 |
| event_data.event_tags JSONB structure | ✅ specs/05 §3 | ❌ Không có | CR-004 |
| Event update/delete API | ✅ SRS §3.7 | ⚠️ Partial | CR-004 |
| **Admin Service** | | | |
| User CRUD with composite PK (id, project_id) | ✅ SRS §3.2, §5.1 | ⚠️ Partial | CR-005 |
| User cascade delete (blobs, profiles, events, etc.) | ✅ SRS §3.2 | ❌ Không có | CR-005 |
| Multi-project isolation (project_id partitioning) | ✅ SRS §3.1 FR-AUTH-002 | ❌ Không có | CR-005 |
| Project token auth (sk-proj-* format) | ✅ PRD §7.2 | ❌ Không có | CR-005 |
| Project profile config API (YAML per project) | ✅ SRS §3.9, URD US-051 | ❌ Không có | CR-005 |
| Profile config overrideable fields (lang, strict, tags, additional) | ✅ SRS §3.9 | ❌ Không có | CR-005 |
| Billing tracking (token_left, next_refill_at) | ✅ PRD §5.6, URD US-052 | ❌ Không có | CR-005 |
| Usage statistics (daily: insert_count, input_tokens, output_tokens) | ✅ SRS §3.9, URD US-052 | ❌ Không có | CR-005 |
| Project status (active/suspended → 403) | ✅ SRS §3.1 | ❌ Không có | CR-005 |
| User listing with pagination | ✅ SRS §3.9, URD US-003 | ❌ Không có | CR-005 |
| NATS admin.user.deleted broadcast | ✅ specs/00 §7.2 | ❌ Không có | CR-005 |
| **Gateway & MCP** | | | |
| Full memobase REST API (30+ endpoints) | ✅ PRD §7.1 | ❌ Partial | CR-006 |
| MCP tool: save_memory (insert + flush) | ✅ PRD §5.7, URD US-061 | ❌ Không có | CR-006 |
| MCP tool: get_user_profiles | ✅ PRD §5.7 | ❌ Không có | CR-006 |
| MCP tool: search_memories | ✅ PRD §5.7 | ❌ Không có | CR-006 |
| MCP SSE transport | ✅ PRD §5.7 | ❌ Không có | CR-006 |
| sk-proj-* project token auth | ✅ PRD §7.2 | ❌ Không có | CR-006 |
| X-Process-Time response header | ✅ SRS §4.5 NFR-O06 | ❌ Không có | CR-006 |
| Custom error format {data, errno, errmsg} | ✅ SRS §6.2 | ❌ Không có | CR-006 |
| USE_CORS + API_HOSTS CORS config | ✅ SRS §4.3 NFR-S04 | ❌ Không có | CR-006 |
| Rate limiting per-project per-endpoint | ✅ specs/00 §8 | ❌ Không có | CR-006 |
| X-Request-ID tracking | ✅ SRS §4.5 NFR-O02 | ❌ Không có | CR-006 |
| **Shared Infrastructure** | | | |
| Bifrost LLM gateway adapter | ✅ specs/00 §4 | ❌ Không có | CR-007 |
| Doubao LLM adapter (ByteDance) | ✅ SRS §4.6 NFR-C04 | ❌ Không có | CR-007 |
| Ollama LLM adapter | ✅ SRS §4.6 NFR-C04 | ❌ Không có | CR-007 |
| Jina embedding adapter | ✅ PRD §6.1, SRS §4.6 | ❌ Không có | CR-007 |
| Ollama embedding adapter | ✅ PRD §6.1, SRS §4.6 | ❌ Không có | CR-007 |
| tiktoken-go token counting (gpt-4o encoder) | ✅ SRS §4.1 NFR-P07 | ❌ Không có | CR-007 |
| Prompt template registry (EN/ZH per prompt type) | ✅ specs/03 §2 | ❌ Không có | CR-007 |
| Circuit breaker (sony/gobreaker per downstream) | ✅ specs/00 §8 | ❌ Không có | CR-007 |
| Bulkhead (semaphore for LLM calls, max=10) | ✅ specs/03 §8 | ❌ Không có | CR-007 |
| Retry with exponential backoff | ✅ specs/00 §8 | ⚠️ Partial | CR-007 |
| DB pool monitoring (warning >80%) | ✅ SRS §4.5 NFR-O05 | ❌ Không có | CR-007 |
| MEMOBASE_* env var prefix for config overrides | ✅ SRS §7.2 | ❌ Không có | CR-007 |
| OTel traces per LLM call (provider, model, tokens) | ✅ SRS §4.5 | ❌ Partial | CR-007 |

---

## New Services to Build

| Service | gRPC Port | Health Port | Maps to Python |
|---|---|---|---|
| `memobase-ingestion` | 9041 | 9091 | `api_layer/blob.py` + `controllers/buffer.py` |
| `memobase-engine` | 9042 | 9092 | `controllers/modal/chat/` + `llms/` + `prompts/` |
| `memobase-context` | 9043 | 9093 | `controllers/context.py` + `controllers/profile.py` |
| `memobase-event` | 9044 | 9094 | `controllers/event.py` + `controllers/event_gist.py` |
| `memobase-admin` | 9045 | 9095 | `controllers/user.py` + `controllers/project.py` + `controllers/billing.py` |

## Shared Packages to Build

| Package | Purpose |
|---|---|
| `pkg/adapters/llm/` | Bifrost, OpenAI, Doubao, Ollama adapters |
| `pkg/adapters/embedder/` | OpenAI, Jina, Ollama embedding adapters |
| `pkg/tokenizer/` | tiktoken-go wrapper |
| `pkg/prompt/` | EN/ZH template registry |
| `pkg/resilience/` | Circuit breaker, retry, bulkhead |
| `pkg/middleware/` | Auth, CORS, rate limiting, tracing |
| `pkg/observability/` | OTel traces, Prometheus metrics |

## Services to Extend

| Service | Changes |
|---|---|
| `gateway` | Memobase REST endpoints (30+), 3 MCP tools (save_memory, get_user_profiles, search_memories), MCP SSE transport, sk-proj-* auth, X-Process-Time header, custom error format |

---

## NATS JetStream Event Map

| Subject | Publisher | Subscribers | Purpose |
|---|---|---|---|
| `memobase.buffer.ready` | ingestion | engine | Trigger LLM processing |
| `memobase.engine.completed` | engine | ingestion, context | Mark buffer done, invalidate cache |
| `memobase.engine.failed` | engine | ingestion | Mark buffer failed |
| `memobase.profile.changed` | engine | context | Invalidate Redis profile cache |
| `memobase.event.created` | engine | event | Index embeddings |
| `memobase.admin.project.updated` | admin | engine, all | Reload project profile config |
| `memobase.admin.user.deleted` | admin | ingestion, context, event | Cascade cleanup |

---

## Architecture Diagram (Target State)

```
External Clients (REST / MCP SSE / Stdio / SDK)
               │
         ┌─────▼────────────────────────────────────────────────────────┐
         │  memobase-gateway (:8080 REST, :8082 MCP SSE, :8081 gRPC)   │
         │  Auth (Bearer / sk-proj-*) │ Rate Limit │ CORS               │
         │  MCP: save_memory, get_user_profiles, search_memories        │
         └─────┬─────────────────────────────────────────────────────────┘
               │ gRPC (per route)
       ┌───────┼──────────────────────────────────────────────┐
       │       │                │             │               │
   ┌───▼────┐ ┌▼──────────┐ ┌──▼──────┐ ┌───▼────┐ ┌───────▼─┐
   │ingest  │ │  engine   │ │ context │ │ event  │ │  admin  │
   │:9041   │ │  :9042    │ │  :9043  │ │ :9044  │ │  :9045  │
   └───┬────┘ └─────┬─────┘ └────┬────┘ └───┬────┘ └─────────┘
       │             │            │          │
       └─────────────┼────────────┼──────────┘
                     │ NATS JetStream (async)
              ┌──────▼──────────────────────┐
              │     Shared Infrastructure    │
              │  PostgreSQL + pgvector       │
              │  Redis (profile cache)       │
              │  NATS JetStream             │
              │  OTel Collector             │
              └────────────────────────────┘
```

---

## Key Differentiators của Memobase so với Graphiti/AgentMemory

| Aspect | Memobase | Graphiti | AgentMemory |
|---|---|---|---|
| **Memory model** | User Profile (topic/sub_topic/content) | Temporal Knowledge Graph | Declarative + Episodic |
| **Core algorithm** | YOLO Merge (3 fixed LLM calls) | Edge invalidation (bi-temporal) | Planning-based retrieval |
| **Search** | Vector similarity (pgvector) | Hybrid (cosine + BM25 + BFS) | Semantic + episodic retrieval |
| **Data structure** | Relational (PG) + JSONB + pgvector | Graph (Neo4j/FalkorDB) | Hybrid |
| **Latency target** | < 100ms context API | < 1000ms search | Not specified |
| **Cost optimization** | 40-50% LLM cost reduction via YOLO | Community detection batching | Background processing |
| **Personalization** | High (user-specific profile topics) | Moderate (entity graph) | Moderate |

---

## Recommended Implementation Order

| Wave | CRs | Rationale |
|---|---|---|
| **Wave 1** (Data Layer) | CR-005 (Admin), CR-001 (Ingestion) | User + blob management needed first |
| **Wave 2** (Processing) | CR-002 (Engine) | Core LLM pipeline, depends on ingestion |
| **Wave 3** (Read Path) | CR-003 (Context), CR-004 (Event) | Read APIs and search after data exists |
| **Wave 4** (Access Layer) | CR-006 (Gateway/MCP), CR-007 (Shared Infra) | API surface + infrastructure hardening |

---

## Performance Targets (từ memobase PRD §13)

| Metric | Target |
|---|---|
| Context API latency | < 100ms P99 (excluding embedding API) |
| LLM calls per flush | Fixed 3 (entry_summary + extract + YOLO merge) |
| Profile cache TTL | 1200s (20 minutes) |
| Profile cache hit rate | > 80% |
| LLM cost reduction | 40-50% vs naive per-message processing |
| DB pool size | pool_size=75, max_overflow=50 |
| Buffer flush threshold | 1024 tokens (default) |
| Buffer idle timeout | 3600s (1 hour) |
