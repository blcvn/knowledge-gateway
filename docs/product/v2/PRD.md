# Product Requirements Document (PRD)

## VNP Memory — Unified Cognitive Infrastructure Layer for Enterprise AI

| Field | Value |
|---|---|
| **Product** | VNP Memory |
| **Version** | 2.3.0 |
| **Status** | Active Development |
| **Last Updated** | 2026-09-03 |
| **Category** | Enterprise AI Infrastructure |
| **Feature Catalog** | [docs/features/](../features/README.md) — 28 features |
| **Pain Points** | [docs/bussiness/painpoints/](../bussiness/painpoints/README.md) — 8 actors |
| **Solutions** | [docs/bussiness/solutions/](../bussiness/solutions/README.md) — 10 solutions |
| **Research** | [docs/bussiness/research/](../bussiness/research/README.md) — Neuroscience + Market |
| **Competitive** | [docs/bussiness/competitive/](../bussiness/competitive/README.md) — 5 competitors |

---

## 1. Executive Summary

VNP Memory là một **Unified Cognitive Infrastructure Layer** cho Enterprise AI — nền tảng tích hợp toàn diện giải quyết bài toán "AI Memory" bằng cách hợp nhất 6 memory engines chuyên biệt (Cognee, Graphiti, Memobase, OpenViking, Zep, Supermemory) dưới một kiến trúc thống nhất, kết hợp với **AgentMemory Layer** — lớp orchestration thông minh dành riêng cho AI Agent operations.

### Tầm nhìn sản phẩm

> **"Enterprise Cognitive Infrastructure — Operating System cho AI Cognition"**

Thay vì xây dựng thêm "một vector DB nữa", VNP Memory tạo ra một **Persistent Context Platform** cho phép AI Agent:
- Nhớ đúng thứ, đúng lúc (Context Quality > Memory Quantity)
- Duy trì bộ nhớ dài hạn có cấu trúc, có quan hệ, có thời gian
- Tự động quên thông tin hết hạn và giải quyết mâu thuẫn
- Quan sát và capture toàn bộ agent lifecycle (hook capture, session replay)
- Tuân thủ governance, audit trail, và multi-tenant isolation cấp enterprise

### Business Value — Before vs After

| Actor | Vấn đề trước đây | Với VNP Memory |
|---|---|---|
| AI Agent Developer | 6 tháng tự xây memory infra | **< 5 phút** (`make dev`) |
| AI Agent Developer | Context token cost $0.50/call | **$0.02/call** (−80%) |
| AI Agent Developer | Debug agent: 2-4 giờ/issue | **20 phút** (Session Replay) |
| Platform Engineer | 35+ services riêng lẻ | **1 binary**, 1 healthz endpoint |
| Enterprise Architect | GDPR forget: manual, bỏ sót | **1 API call**, cascading 6 engines |
| IDE Plugin User | 10 phút brief AI mỗi sáng | **0 phút** (persistent context) |
| AI Power User | AI không nhớ preferences | **Personalized từ session đầu** |

> Chi tiết: [Pain Points](../bussiness/painpoints/README.md) | [Solutions](../bussiness/solutions/README.md)

### Giá trị cốt lõi

| Pillar | Mô tả | Engine chính | Research backing |
|---|---|---|---|
| **Episodic Memory** | Theo dõi sự kiện theo thời gian, temporal reasoning | Graphiti | Temporal facts = brain’s event memory |
| **Semantic Memory** | Trích xuất tri thức, xây dựng knowledge graph | Cognee | Schema networks (personal-memory.md) |
| **Conversational Memory** | Bộ nhớ hội thoại, context assembly < 200ms | Zep | Hippocampus fast-write (sleep.md) |
| **Profile Memory** | Structured user profiles từ conversations | Memobase | World model update (predictive-processing.md) |
| **Adaptive Memory** | Living KG với auto-forgetting, external connectors | Supermemory | Synaptic pruning + forgetAfter (sleep.md) |
| **Procedural Memory** | Context phân tầng L0/L1/L2, VikingFS | OpenViking | Neocortex hierarchy (neocortex.md) |
| **AgentMemory Layer** | Hook capture, lifecycle, orchestration, consolidation | observe-service + memory-service | Sleep consolidation stages (sleep.md) |

---

## 2. Problem Statement

### 2.1 Vấn đề thị trường

Enterprise AI đang chuyển từ `RAG → Agentic RAG → Persistent Memory Systems`. Các vấn đề lớn nhất:

| # | Vấn đề | Impact | Solution |
|---|---|---|---|
| 1 | **Context window đắt** | Chi phí token tăng tuyến tính | [S6](../bussiness/solutions/S6-context-efficiency.md) — 80% token reduction |
| 2 | **Agent không nhớ dài hạn** | Mất ngữ cảnh giữa các phiên | [S1](../bussiness/solutions/S1-persistent-memory.md) — 4-layer persistent memory |
| 3 | **Memory fragmented** | Thông tin rải rác, không thống nhất | [S2](../bussiness/solutions/S2-unified-api.md) — Unified Memory API |
| 4 | **Thiếu temporal reasoning** | Không theo dõi thay đổi theo thời gian | [S3](../bussiness/solutions/S3-temporal-reasoning.md) — Graphiti validity windows |
| 5 | **Thiếu user profiling** | Không có structured profiles | [S5](../bussiness/solutions/S5-user-profiling.md) — Memobase YOLO Engine |
| 6 | **Governance / Audit gap** | Không kiểm soát AI nhớ gì | [S9](../bussiness/solutions/S9-governance-compliance.md) — Enterprise Governance |
| 7 | **Memory không tự evolve** | Knowledge cũ không tự update | [S4](../bussiness/solutions/S4-knowledge-evolution.md) — Adaptive Knowledge Evolution |
| 8 | **Thiếu agent observability** | Không track được agent lifecycle | [S7](../bussiness/solutions/S7-agent-observability.md) — Hook Capture + Session Replay |
| 9 | **Multi-agent coordination** | Race conditions, không phối hợp | [S8](../bussiness/solutions/S8-multi-agent.md) — Distributed Leases + Signals |

> Phân tích chi tiết: [Pain Points](../bussiness/painpoints/README.md)

### 2.2 Hạn chế giải pháp hiện tại — Competitive Analysis

| Capability | Cognee | Graphiti | Memobase | Supermemory | Zep | **VNP Memory** |
|---|---|---|---|---|---|---|
| Knowledge graph | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ (5 engines) |
| Temporal reasoning | ⚠️ | ✅ | ❌ | ✅ | ✅ | ✅ |
| User profile | ❌ | ❌ | ✅ | ⚠️ | ❌ | ✅ |
| Filesystem memory | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ (OpenViking) |
| Auto-forget | ❌ | ❌ | ❌ | ✅ | ❌ | ✅ |
| Agent hooks (12 types) | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| Session replay | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| Multi-agent coord. | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ (leases) |
| Memory consolidation | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ (4-tier) |
| GDPR cascading | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| MCP server | ❌ | ❌ | ❌ | ✅ | ❌ | ✅ (37+ tools) |
| External connectors | ❌ | ❌ | ❌ | ✅ | ❌ | ✅ |
| Custom ontology | ❌ | ✅ | ❌ | ❌ | ✅ | ✅ |
| Multi-modal ingestion | ✅ | ⚠️ | ❌ | ✅ | ✅ | ✅ |
| Context < 100-200ms | ⚠️ | ✅ | ✅ | ⚠️ | ✅ | ✅ |
| Enterprise governance | ⚠️ | ❌ | ❌ | ❌ | ❌ | ✅ |
| Unified API | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |

**Positioning:** VNP Memory không cạnh tranh trực tiếp với 5 engines trên — VNP Memory **orchestrates** chúng.

> **VNP Memory = Cognee + Graphiti + Memobase + Supermemory + Zep + AgentMemory + Enterprise Governance**

> Chi tiết: [Competitive Analysis](../bussiness/competitive/README.md) | [Research Insights](../bussiness/research/README.md)

---

## 3. Target Users

### 3.1 Primary Users

| Persona | Nhu cầu | Pain Points | Key Solutions |
|---|---|---|---|
| **AI Agent Developer** | Memory SDK cho agent, hook capture | [P1 pain points](../bussiness/painpoints/P1-ai-agent-developer.md) | S1, S2, S6, S7 |
| **Platform Engineer** | Self-host & scale memory infra | [P2 pain points](../bussiness/painpoints/P2-platform-engineer.md) | S9, S10 |
| **ML/AI Engineer** | Tối ưu context quality | [P3 pain points](../bussiness/painpoints/P3-ml-ai-engineer.md) | S3, S6, S7 |
| **Enterprise Architect** | Governance & compliance | [P4 pain points](../bussiness/painpoints/P4-enterprise-architect.md) | S9 |

### 3.2 Secondary Users

| Persona | Nhu cầu | Pain Points | Key Solutions |
|---|---|---|---|
| **AI Framework Integrator** | Standard API, context injection | [P6 pain points](../bussiness/painpoints/P6-framework-integrator.md) | S2, S6 |
| **Product Manager** | User profile analytics | [P8 pain points](../bussiness/painpoints/P8-product-manager.md) | S5 |
| **IDE Plugin User** | AI coding assistant với persistent memory | [P5 pain points](../bussiness/painpoints/P5-ide-plugin-user.md) | S1, S6 |
| **AI Power User** | Personalization, transparency | [P7 pain points](../bussiness/painpoints/P7-ai-power-user.md) | S1, S4, S5 |

---

## 3.3 Solution Architecture Overview

Mỗi pain point của actor được giải quyết bởi một Solution cụ thể:

| Solution | Giải quyết | Features |
|---|---|---|
| [S1 — Persistent Memory](../bussiness/solutions/S1-persistent-memory.md) | Agent mất context sau session | F01, F04, F05, F06, F07 |
| [S2 — Unified API](../bussiness/solutions/S2-unified-api.md) | Memory fragmented, no standard | F01, F10, F13 |
| [S3 — Temporal Reasoning](../bussiness/solutions/S3-temporal-reasoning.md) | RAG không hiểu thời gian | F02, F04, F09 |
| [S4 — Knowledge Evolution](../bussiness/solutions/S4-knowledge-evolution.md) | Knowledge không tự update | F07, F09, F19 |
| [S5 — User Profiling](../bussiness/solutions/S5-user-profiling.md) | Không có user profile | F05, F18 |
| [S6 — Context Efficiency](../bussiness/solutions/S6-context-efficiency.md) | Context tốn token/chậm | F05, F06, F12, F13 |
| [S7 — Agent Observability](../bussiness/solutions/S7-agent-observability.md) | Không debug được agent | F08, F20, F21, F26 |
| [S8 — Multi-Agent Coordination](../bussiness/solutions/S8-multi-agent.md) | Race conditions | F11 |
| [S9 — Enterprise Governance](../bussiness/solutions/S9-governance-compliance.md) | GDPR, audit, policy gap | F14, F16, F22, F27 |
| [S10 — Infrastructure](../bussiness/solutions/S10-infrastructure-simplicity.md) | 35+ services phức tạp | F01, F15, F23, F24, F25 |

---

## 4. Product Architecture

### 4.1 Deployment Models

**Monolith Mode** (Development & Single-server)
```
backend/apps/memory — Single Go binary
    Gateway (REST :8080, MCP :8082, Health :8083)
        └── InProcessRegistry (bufconn)
            └── 35+ Engine Services (in-memory gRPC)
                └── Embedded NATS JetStream
```

**Distributed Mode** (Production)
```
backend/gateway/ — API Gateway
    └── gRPC → Distributed Engine Services
               → Shared Infrastructure (PostgreSQL, Neo4j, Redis, NATS)
```

### 4.2 Memory Engine Roles

| Engine | Memory Type | Vai trò | Services |
|---|---|---|---|
| **Graphiti** | Episodic | Temporal context graph, fact validity, provenance | graphiti-ingestion, graphiti-search, graphiti-knowledge, graphiti-store |
| **Cognee** | Semantic | Knowledge extraction pipeline, multi-modal ingestion | cognee-ingestion, cognee-cognify, cognee-search |
| **Zep** | Conversational | Session memory, context assembly, Graph RAG | zep-user, zep-thread, zep-memory, zep-graph, zep-search, zep-admin |
| **Memobase** | Profile | Structured user profiles (YOLO engine), event timeline | memobase-ingestion, memobase-engine, memobase-context |
| **OpenViking** | Procedural | Virtual filesystem (VikingFS), tiered context L0/L1/L2 | ov-fs, ov-search, ov-session, ov-resource, ov-crypto, ov-admin |
| **Supermemory** | Adaptive | Living KG, auto-forgetting, external connectors, RAG | sm-document, sm-memory, sm-search, sm-profile, sm-connector, sm-mcp, sm-auth, sm-analytics, sm-project |

### 4.3 AgentMemory Layer

Lớp orchestration thứ 2 dành riêng cho AI Agent operations:

| Service | Vai trò | Feature |
|---|---|---|
| **observe-service** | Hook capture, session management, SSE streaming | F08, F26 |
| **memory-service** | Typed memory lifecycle, versioning, eviction | F09 |
| **search-service** | Hybrid search: BM25 + Vector + RRF | F10 |
| **orchestration-service** | Distributed leases, signals, sentinels | F11 |
| **pipeline-service** | 4-tier consolidation pipeline | F12 |
| **observe-search** | AgentMemory search across raw observations | F10 |

### 4.4 Platform Services

| Service | Vai trò |
|---|---|
| **vnp-admin** | Tenant lifecycle, API key management, quotas |
| **vnp-event** | Cross-engine event timeline, UserEvent log |
| **vnp-search-hub** | Cross-engine recall (parallel gRPC fan-out + merge) |
| **vnp-platform** | Admin APIs, auth, analytics |

---

## 5. Core Features (6 Memory Engines)

> Tham khảo chi tiết tại [Feature Catalog](../features/README.md) — 28 features với dataflow đầy đủ.

### 5.1 Unified Memory API — F01

Gateway routes tự động tới engine phù hợp:

```
POST /v1/memory/store    → Auto-route by type
POST /v1/memory/recall   → Cross-engine (vnp-search-hub)
POST /v1/memory/forget   → Cascading delete
GET  /v1/memory/timeline → Temporal events (vnp-event)
```

Memory types và routing:
```
type=semantic       → cognee-ingestion
type=episodic       → graphiti-ingestion
type=conversational → zep-memory
type=profile        → memobase-ingestion (Blob → Buffer)
type=procedural     → ov-fs
type=adaptive       → sm-memory
type=auto           → LLM content classification
```

### 5.2 Episodic Memory (Graphiti) — F02

| Feature | Mô tả | Priority |
|---|---|---|
| Temporal Fact Management | Mỗi fact có validity window (valid_at/invalid_at) | P0 |
| Episode Ingestion | POST /v1/graphiti/episodes | P0 |
| Entity/Edge Extraction | LLM-powered extraction với deduplication | P0 |
| Hybrid Search | Semantic + BM25 + Graph traversal + reranking | P0 |
| Graph Nodes/Edges | GET /v1/graphiti/nodes/{id}, edges/{id} | P0 |
| Knowledge Graph | graphiti-knowledge service | P1 |

### 5.3 Semantic Memory (Cognee) — F03

| Feature | Mô tả | Priority |
|---|---|---|
| Multi-modal Ingestion | PDF, text, audio, image, CSV, URL | P0 |
| Dataset Management | POST /v1/cognee/datasets | P0 |
| Cognify Pipeline | POST /v1/cognee/datasets/{id}/cognify | P0 |
| Multi-strategy Search | POST /v1/cognee/search (15+ strategies) | P0 |

### 5.4 Conversational Memory (Zep) — F04

| Feature | Mô tả | Priority |
|---|---|---|
| User Management | POST/GET/PATCH /v1/zep/users | P0 |
| Session Memory | PUT/GET /v1/zep/sessions/{id}/memory | P0 |
| Graph Search | POST /v1/zep/graph/search | P0 |
| Facts Management | POST /v1/zep/graph/facts | P0 |
| Ontology | POST /v1/zep/graph/ontology | P1 |

### 5.5 Profile Memory (Memobase) — F05

| Feature | Mô tả | Priority |
|---|---|---|
| Blob Ingestion | POST /v1/memobase/users/{uid}/blobs | P0 |
| Buffer Auto-flush | Flush khi >= 20 blobs (configurable) | P0 |
| Context Assembly | GET /v1/memobase/users/{uid}/context | P0 |
| Structured Profiles | GET /v1/memobase/users/{uid}/profiles | P0 |
| Event Timeline | GET /v1/memobase/users/{uid}/events | P1 |

**YOLO Engine**: Fixed 3 LLM calls per flush (extract → merge → events). Profile categories: preference/fact/goal/habit.

### 5.6 Procedural Memory (OpenViking) — F06

| Feature | Mô tả | Priority |
|---|---|---|
| VikingFS | GET/PUT/DELETE /v1/ov/files/{path} | P0 |
| Directory Tree | GET /v1/ov/tree/{path} | P0 |
| Grep + Semantic Search | POST /v1/ov/grep, /v1/ov/search | P0 |
| Session Management | POST /v1/ov/sessions | P0 |
| Resource Ingest | POST /v1/ov/resources/ingest (Git, HTTP, local) | P1 |

**Tiered Context**: L0 (~100 tok) → L1 (~2K tok) → L2 (full detail), load on demand.

### 5.7 Adaptive Memory (Supermemory) — F07

| Feature | Mô tả | Priority |
|---|---|---|
| Document Management | POST/GET /v1/sm/documents | P0 |
| Memory Store | POST /v1/sm/memories (adaptive KG) | P0 |
| Hybrid Search + RAG | POST /v1/sm/search, /v1/sm/rag | P0 |
| User Profile | GET /v1/sm/profiles/{uid} | P0 |
| External Connectors | POST /v1/sm/connections + sync | P1 |
| Project Spaces | POST /v1/sm/projects/spaces | P1 |

**Auto-forgetting**: Memory versioning (parent → root chain), `isLatest` flag, `forgetAfter` duration.

---

## 6. AgentMemory Layer

### 6.1 Agent Observe & Hook Capture — F08

Observe Service là **hook capture pipeline** cho AI Agent — thu thập tất cả hoạt động agent và xây dựng session timeline có cấu trúc.

**12 Hook Types**: session_start/end, llm_prompt/response, tool_call/response, memory_read/write, error, decision, observation, checkpoint.

**14-Step Observe Pipeline**:
1. Receive → 2. Validate → 3. Authenticate → 4. Deduplicate (30s TTL DedupMap)
5. Privacy Redact (API keys, JWT, PII) → 6. Parse Hook Type → 7. Enrich
8. Classify → 9. Store Raw → 10. Index (BM25) → 11. Embed (vector)
12. Publish NATS → 13. Update Session State → 14. Stream SSE

Session states: `active` / `completed` / `abandoned`.

### 6.2 Agent Memory Lifecycle — F09

| Feature | Mô tả |
|---|---|
| 6 Memory Types | working / episodic / semantic / procedural / adaptive / profile |
| Jaccard Versioning | Similarity-based deduplication, chain parent → root |
| Memory Decay | eviction_score = importance × recency × frequency |
| TTL Auto-forget | Per-memory configurable TTL |
| Eviction Policy | LRU-based với importance scoring |

### 6.3 Hybrid Search Engine — F10

| Strategy | Implementation |
|---|---|
| BM25 | In-memory inverted index, TF-IDF scoring |
| Vector | pgvector cosine similarity |
| RRF Fusion | Reciprocal Rank Fusion để merge kết quả |

Search targets: raw observations, memories, sessions, agent state.

### 6.4 Multi-Agent Orchestration — F11

| Feature | Mô tả |
|---|---|
| Distributed Leases | Agent request lease trước khi truy cập shared resource |
| Inter-agent Signals | Async signal passing giữa agents |
| Sentinel Guards | Boundary enforcement cho agent scopes |
| Action Queue | Serialized action execution với ordering |

### 6.5 Memory Consolidation Pipeline — F12

4-tier compression pipeline:

| Tier | Trigger | Output |
|---|---|---|
| L1 — Session Summary | Session end | Compressed summary ~500 tokens |
| L2 — Daily Digest | Daily cron | Cross-session patterns |
| L3 — Weekly Synthesis | Weekly cron | Long-term knowledge |
| L4 — Core Knowledge | Manual/threshold | Permanent memory |

### 6.6 Session Replay — F26

| Feature | Mô tả |
|---|---|
| Timeline Scrubbing | Jump đến bất kỳ timestamp trong session |
| Event Filtering | Filter theo hook type |
| Playback Speed | 1x, 2x, 4x, slow motion |
| JSONL Import | Import external transcripts (Claude Code format) |
| SSE Streaming | Real-time event streaming khi replay |

---

## 7. Memory API Gateway

### 7.1 Gateway Ports

| Port | Protocol | Description |
|---|---|---|
| :8080 | REST HTTP | Primary API (50+ routes) |
| :8081 | gRPC | Internal gRPC (optional) |
| :8082 | MCP SSE + HTTP | Model Context Protocol (37+ tools) |
| :8083 | HTTP | Health check + Prometheus metrics |

### 7.2 Authentication & Multi-tenancy — F14

| Mechanism | Mô tả |
|---|---|
| API Key | Per-tenant keys, SHA-256 hash, KeyPrefix cho identification |
| JWT RS256 | Bearer token, RSA-256 signed |
| Dev Mode | AUTH_DEV_MODE=true — skip auth, localhost only |
| Rate Tiers | free / pro / enterprise, per-tenant quotas |
| Namespace | TenantID injected vào mọi query, isolation guaranteed |

### 7.3 MCP Server & Context Injection — F13

Protocol: `2024-11-05`. Methods: initialize, tools/list, tools/call, ping.

**37+ MCP Tools** (tăng từ 16 ở v1):

| Domain | Tools |
|---|---|
| Memory Core | memory_store, memory_recall, memory_search, memory_timeline, memory_profile, memory_forget |
| Graph | graph_query |
| OpenViking FS | ov_read_file, ov_write_file, ov_search, ov_list_dir, ov_grep, ov_tree, ov_session_commit, ov_ingest, ov_delete |
| AgentMemory | observe_*, agent_remember, agent_recall, agent_slots, orchestrate_lease, orchestrate_signal, và nhiều tools khác |

**Context Injection**: Pre-call hook tự động inject relevant memory vào LLM prompt với configurable token budget. Sources: Memobase profile, OpenViking tiered context, Supermemory KG.

**Agent Scoping**: `isolated` / `shared` / `project` namespace.

### 7.4 WebSocket Real-time Events — F28

```
Console UI → WebSocket /v1/console/ws → Real-time event stream
```

Event categories: memory operations, pipeline progress, agent hooks, system alerts.

---

## 8. Console (Admin UI) — 12 Sections

| # | Feature | Routes | Description |
|---|---|---|---|
| F15 | Dashboard | /v1/console/dashboard/* | Health, metrics, throughput, memory heatmap |
| F16 | Memory Explorer | /v1/console/memory/* | Search, inspect, neighbors, versions |
| F17 | Graph Studio | /v1/console/graph/* | Subgraph, entity, timeline, ontology, query |
| F18 | User Profiles | /v1/console/profiles/* | Profile explorer, config, events, context, buffers |
| F19 | Adaptive Memory | /v1/console/adaptive/* | Memory versions, connectors, analytics, forget-rules |
| F20 | Agent Context Debugger | /v1/console/debugger/* | Context assembly traces |
| F21 | Sessions Explorer | /v1/console/sessions/* | Live sessions, timeline, diff, working-memory |
| F22 | Governance Center | /v1/console/governance/* | Tenants, policies, audit, GDPR forget |
| F23 | Pipeline Monitor | /v1/console/pipelines/* | Job status, queues, workers, templates |
| F24 | Infrastructure Health | /v1/console/infra/* | Service topology, databases, resources |
| F25 | Observability | /v1/console/observability/* | Metrics, traces, errors, costs |
| F27 | Organization & SDK | /v1/console/org/* | API keys, billing, SSO, engine aliases |

---

## 9. Technical Architecture

### 9.1 Technology Stack

| Layer | Technology |
|---|---|
| Language | Go 1.23+ |
| HTTP Framework | stdlib net/http (Go 1.22+ pattern matching) |
| In-process gRPC | bufconn (google.golang.org/grpc) |
| Message Broker | NATS JetStream (embedded or external) |
| Graph Database | Neo4j 5+ |
| Relational Database | PostgreSQL 17 + pgvector |
| Vector Database | Qdrant (external, optional) |
| Cache | Redis 7+ |
| Object Storage | MinIO (S3 compatible) |
| Custom Storage | VikingFS (Go-native) |
| AI Gateway | Bifrost (multi-provider LLM routing) |
| Observability | OpenTelemetry + Prometheus |
| Logging | slog (structured JSON, stdlib) |

### 9.2 Data Flow — Event-Driven

```
Agent Request → Memory API Gateway → Engine Selection
    │
    ├─→ Memobase: Blob → Buffer (auto-flush at 20) → YOLO Engine (3 LLM calls)
    ├─→ Cognee: Add → Cognify (7 stages) → Knowledge Graph
    ├─→ Graphiti: Episode → Extract → Deduplicate → Temporal Graph
    ├─→ Zep: Message → Graph ingestion → Context assembly
    ├─→ Supermemory: Content → Memory (adaptive KG) → Version chain
    ├─→ OpenViking: Data → VikingFS → L0/L1/L2 tiered context
    └─→ AgentMemory: Observation → 14-step Pipeline → SSE Stream
         │
         ▼
    NATS JetStream → Event fan-out → Subscribed services
         └─→ Consolidation Pipeline → 4-tier compression → Summary store
```

### 9.3 Middleware Stack

```
Recovery → RequestID → Logger → CORS → Auth → RateLimit → Metrics → Timeout
```

---

## 10. Deployment

### 10.1 Service Ports (Monolith)

| Service | Port | Health Check |
|---|---|---|
| Memory REST API | 8080 | /v1/admin/health |
| MCP Server | 8082 | /sse |
| Health + Metrics | 8083 | /healthz |
| PostgreSQL | 5432 | pg_isready |
| Neo4j | 7687 | bolt |
| Qdrant | 6333 | /healthz |
| Redis | 6379 | PING |
| MinIO | 9000 | — |

### 10.2 Quick Start

```bash
make infra-up   # Start infrastructure
make dev        # Run monolith (35+ services in one process)

# Verify
curl http://localhost:8083/healthz | jq

# REST API
curl -X POST http://localhost:8080/v1/memory/store \
  -H "Content-Type: application/json" \
  -d '{"content":"hello world","type":"fact"}'

# MCP tools (37+)
curl -X POST http://localhost:8082/message \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

# AgentMemory observe session
curl -X POST http://localhost:8080/v1/observe/sessions \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"agent-001","session_name":"coding-session"}'
```

---

## 11. Security & Compliance

### 11.1 Security

| Requirement | Implementation |
|---|---|
| Authentication | API Key (SHA-256) + JWT RS256 |
| Tenant Isolation | TenantID injected vào mọi domain query |
| Encryption | TLS in transit; ov-crypto cho at-rest |
| Secret Redaction | Auto-redact trong Observe pipeline (step 5) |
| Dev Mode Guard | AUTH_DEV_MODE chỉ accept localhost |

### 11.2 Governance — F22

| Feature | Mô tả |
|---|---|
| GDPR Forget | POST /v1/console/governance/gdpr/forget — cascading |
| Audit Trail | GET /v1/console/governance/audit |
| Retention Policies | forgetAfter per memory type, TTL config |
| OPA Policies | POST/PUT /v1/console/governance/policies |
| Tenant Management | GET/POST/PUT /v1/console/governance/tenants |

---

## 12. Non-Functional Requirements

### 12.1 Performance

| Metric | Target |
|---|---|
| Memobase context retrieval (p95) | < 100ms |
| Conversational context assembly (p95) | < 200ms (Zep) |
| AgentMemory observe ingest (p95) | < 50ms |
| Hierarchical search (p95) | < 500ms |
| Graph search (p95) | < 1000ms |
| Concurrent sessions | >= 1,000 per instance |
| Token cost reduction vs naive RAG | >= 80% |

### 12.2 Reliability

| Requirement | Implementation |
|---|---|
| Availability | >= 99.9% API uptime |
| Circuit Breaker | Downstream protection, NATS event on open |
| Fallback LLM | Bifrost multi-provider routing |
| Observe Deduplication | DedupMap với 30s TTL |
| Graceful Shutdown | HTTP drain → NATS drain → gRPC stop → DB close |

---

## 13. Integration & Interoperability

### 13.1 Agent Framework Support

| Framework | Integration Method |
|---|---|
| LangChain / LangGraph | Python SDK |
| CrewAI / AutoGen | REST API |
| OpenAI Agents SDK | REST API / MCP |
| Claude Code | MCP Server (SSE) + JSONL Session Import |
| Vercel AI SDK | REST API (Supermemory) |
| Mastra / n8n | REST API |

### 13.2 LLM Provider Support (Bifrost)

OpenAI, Azure OpenAI, Anthropic, Google Gemini, Groq, Mistral, Ollama, HuggingFace, llama.cpp, OpenRouter.

---

## 14. Roadmap

### Phase 1 — Foundation ✅ (Complete)
- ✅ 35+ service monolith architecture
- ✅ VNP Gateway với 50+ REST routes
- ✅ MCP Server với 37+ tools (F13)
- ✅ WebSocket real-time events (F28)
- ✅ 6 Memory Engines (F01-F07)
- ✅ Console API routes — 12 sections (F15-F25, F27)
- ✅ Embedded NATS JetStream + InProcessRegistry (bufconn)
- ✅ AgentMemory Layer: Observe + Lifecycle + Search + Orchestration + Consolidation (F08-F12)
- ✅ Session Replay với JSONL import (F26)
- ✅ Organization & SDK Manager (F27)

### Phase 2 — Console UI 🔲
- 🔲 SPA embedded UI (backend/apps/memory/internal/ui)
- 🔲 Dashboard với real-time metrics (F15)
- 🔲 Memory Explorer với graph visualization (F16)
- 🔲 Graph Studio (F17)
- 🔲 Agent Context Debugger (F20)
- 🔲 Session Replay UI với timeline scrubbing (F26)
- 🔲 Pipeline Monitor + Governance Center UI (F22, F23)

### Phase 3 — Advanced Memory 🔲
- 🔲 Cross-engine memory deduplication
- 🔲 Memory decay & salience scoring tự động
- 🔲 Supermemory connector auto-sync (Google Drive, GitHub, Notion)
- 🔲 Graphiti + Memobase integration (temporal profiles)
- 🔲 Consolidation L3/L4 automation

### Phase 4 — Enterprise 🔲
- 🔲 OPA policy engine integration
- 🔲 Kubernetes Helm charts + Multi-region deployment
- 🔲 SDK (Python, TypeScript, Go)
- 🔲 SurrealDB unified storage evaluation

---

## 15. Success Metrics

| Metric | Target | Measurement |
|---|---|---|
| Memobase context retrieval (p95) | < 100ms | Prometheus |
| Conversational context retrieval (p95) | < 200ms | Prometheus |
| Cross-engine recall (p95) | < 500ms | Prometheus |
| AgentMemory observe ingest (p95) | < 50ms | Prometheus |
| Token cost reduction | >= 80% | Usage monitoring |
| Tenant isolation | 100% (zero leaks) | Integration tests |
| API uptime | >= 99.9% | Health monitoring |
| MCP tool success rate | >= 99% | Traces |
| Profile extraction accuracy | >= 85% relevance | Eval framework |

---

## 16. Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Integration complexity | 6 engines + AgentMemory + in-process | Monolith simplifies dev, distributed for scale |
| Memobase token budget | LLM flush cost scales with users | Fixed 3-call YOLO engine, async processing |
| Memory explosion | Storage grows unbounded | forgetAfter, Consolidation pipeline L4 |
| Context assembly latency | Agent cần < 1s response | Memobase < 100ms, precomputed profiles |
| Multi-tenant isolation | Cross-tenant data leak | TenantID on every query, integration tests |
| LLM provider dependency | Single provider failure | Bifrost multi-provider failover |
| Observe pipeline bottleneck | High-frequency hook events | DedupMap (30s TTL), async NATS, SSE backpressure |

---

## 17. Component References

| Component | Location |
|---|---|
| **Gateway** | backend/gateway/ — REST, MCP, WebDAV |
| **Monolith** | backend/apps/memory/ — 35+ services, embedded NATS |
| **Memory Service** | backend/services/memory-service/ |
| **Observe Service** | backend/services/observe-service/ |
| **Observe Search** | backend/services/observe-search/ |
| **Orchestration Service** | backend/services/orchestration-service/ |
| **Pipeline Service** | backend/services/pipeline-service/ |
| **Search Service** | backend/services/search-service/ |
| **Storage Service** | backend/services/storage-service/ |
| **VNP Platform** | backend/services/vnp-platform/ |
| **Obs Service** | backend/services/obs-service/ |
| **Feature Catalog** | docs/features/ — 28 features với dataflow chi tiết |
| **Previous PRD** | docs/product/v1/PRD.md |

---

## 18. Design Decisions & Trade-offs

| Decision | Choice | Rationale | Trade-off |
|---|---|---|---|
| Monolith vs microservices | Monolith (bufconn) | Zero-latency in-process, dev simplicity | Single process failure |
| NATS embedded | Embedded by default | Zero infra overhead for dev | External NATS cho production |
| HTTP router | stdlib net/http | No dependency, Go 1.22+ patterns | Less middleware ecosystem |
| Vector storage | pgvector (PostgreSQL) | Consolidated infra, ACID | Qdrant optional cho scale |
| Auth | SHA-256 API key + JWT RS256 | Standard, auditable | No ABAC yet (OPA planned) |
| Memobase flush | Fixed 3 LLM calls (YOLO) | Predictable cost, fast | Less granular control |
| Supermemory versioning | parent → root chain | Full audit trail, contradiction resolution | Storage overhead |
| UI embedding | SPA embedded in binary | Single binary deployment | UI rebuild = binary rebuild |
| AgentMemory observe | 14-step pipeline | Comprehensive processing, dedup, privacy | Pipeline latency (~50ms) |
| Consolidation | 4-tier compression | Progressive summarization, storage efficiency | LLM cost per tier |
| MCP tools | 37+ tools | Full memory operations via AI assistants | Toolset discovery complexity |

---

*Tham khảo chi tiết từng feature: [docs/features/](../features/README.md)*

---

## Appendix A — Research Foundations

### A.1 Neuroscience-Inspired Design Principles

VNP Memory được thiết kế dựa trên các nguyên lý từ neuroscience — lý giải "tại sao" đằng sau kiến trúc:

| Design Principle | Neuroscience Source | Implementation trong VNP Memory |
|---|---|---|
| **Capture everything, store smart** | Hippocampus (RAM) vs Neocortex (HDD) | F08 capture all → F12 consolidate |
| **Offline consolidation** | Sleep stages: NREM → REM → insight | F12 4-tier pipeline (khi agent idle) |
| **Memory = relationships** | Schema theory, Hebbian learning | Knowledge Graphs: F02, F03, F04 |
| **Surprise drives learning** | Predictive Processing — prediction error | Contradiction detection: F07, F09 |
| **Forgetting is a feature** | Synaptic pruning during sleep | `forgetAfter` + eviction: F07, F09 |
| **Tiered abstraction** | Neocortex 6 layers | L0/L1/L2 context: F06 |
| **Context = reconstruction** | Memory as active reconstruction | Context assembly: F05, F13 |
| **Temporal reasoning essential** | Memory encodes time stamps | `valid_at`/`invalid_at`: F02, F09 |

> Đọc thêm: [Research Insights](../bussiness/research/README.md)

### A.2 Competitive Research Sources

| Competitor | Documents nghiên cứu |
|---|---|
| **Cognee** | [`docs/research/market/cognee/`](../../research/market/cognee/) — PRD, SRS, URD, TDD, Architecture, 13 service specs |
| **Graphiti** | [`docs/research/market/graphiti/`](../../research/market/graphiti/) — PRD, SRS, URD, Architecture, 8 service specs |
| **Memobase** | [`docs/research/market/memobase/`](../../research/market/memobase/) — PRD, SRS, URD, TDD, Architecture, 9 service specs |
| **Supermemory** | [`docs/research/market/supermemory/`](../../research/market/supermemory/) — PRD, SRS, URD, TDD, Architecture, 10 service specs |
| **Zep** | [`docs/research/market/zep/`](../../research/market/zep/) — PRD, SRS, URD, Architecture, 10 service specs |

> Phân tích đầy đủ: [Competitive Landscape](../bussiness/competitive/README.md)

### A.3 Neuroscience Research Sources

| Topic | File | Key Insight áp dụng |
|---|---|---|
| Memory consolidation | [`sleep.md`](../../research/sleep.md) | F12 Consolidation 4-tier (mirrors sleep stages) |
| Schema & learning | [`personal-memory.md`](../../research/personal-memory.md) | F09 Jaccard-based versioning (schema assimilation) |
| Predictive Processing | [`predictive-processing.md`](../../research/predictive-processing.md) | F07/F09 contradiction detection (prediction error) |
| Synapse strength | [`synapse.md`](../../research/synapse.md) | F09 salience scoring (synaptic weight) |
| Perception pipeline | [`sensor.md`](../../research/sensor.md) | F08 14-step observe pipeline |
| Hierarchical cortex | [`neocortex.md`](../../research/neocortex.md) | F06 L0/L1/L2 tiered context |
| Morphological computation | [`morphonomic.md`](../../research/morphonomic.md) | Architecture: structure as computation |
| Neuromorphic computing | [`neumorphic-computing.md`](../../research/neumorphic-computing.md) | Future: hardware-efficient memory ops |
| Symbol grounding | [`writing.md`](../../research/writing.md) | Context injection: symbol → concept activation |
