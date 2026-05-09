# Product Requirements Document (PRD)

## VNP Memory — Unified Cognitive Infrastructure Layer for Enterprise AI

| Field | Value |
|---|---|
| **Product** | VNP Memory |
| **Version** | 1.0.0 |
| **Status** | Development |
| **Last Updated** | 2026-05-09 |
| **Category** | Enterprise AI Infrastructure |

---

## 1. Executive Summary

VNP Memory là một **Unified Cognitive Infrastructure Layer** cho Enterprise AI — nền tảng tích hợp toàn diện giải quyết bài toán "AI Memory" bằng cách thống nhất bốn engine chuyên biệt (Cognee, Graphiti, Zep, OpenViking) dưới một kiến trúc phân tầng thống nhất, được quản trị bởi KGS Platform (Knowledge Graph Service).

### Tầm nhìn sản phẩm

> **"Enterprise Cognitive Infrastructure — Operating System cho AI Cognition"**

Thay vì xây dựng thêm "một vector DB nữa", VNP Memory tạo ra một **Persistent Context Platform** cho phép AI Agent:
- Nhớ đúng thứ, đúng lúc (Context Quality > Memory Quantity)
- Duy trì bộ nhớ dài hạn có cấu trúc, có quan hệ, có thời gian
- Tuân thủ governance, audit trail, và multi-tenant isolation cấp enterprise

### Giá trị cốt lõi

| Pillar | Mô tả | Engine chính |
|---|---|---|
| **Episodic Memory** | Theo dõi sự kiện theo thời gian, temporal reasoning | Graphiti |
| **Semantic Memory** | Trích xuất tri thức, xây dựng knowledge graph | Cognee |
| **Conversational Memory** | Bộ nhớ hội thoại, context assembly < 200ms | Zep |
| **Procedural Memory** | Tổ chức context phân tầng, skills, workflows | OpenViking |
| **Knowledge Governance** | Multi-tenant graph, ontology, ABAC policies | KGS Platform |

---

## 2. Problem Statement

### 2.1 Vấn đề thị trường

Enterprise AI đang chuyển từ `RAG → Agentic RAG → Persistent Memory Systems`. Các vấn đề lớn nhất:

| # | Vấn đề | Impact |
|---|---|---|
| 1 | **Context window đắt** | Chi phí token tăng tuyến tính với context size |
| 2 | **Agent không nhớ dài hạn** | Mất ngữ cảnh giữa các phiên, không tự cải thiện |
| 3 | **Memory fragmented** | Thông tin rải rác ở nhiều hệ thống, không thống nhất |
| 4 | **Thiếu temporal reasoning** | Không theo dõi được sự thay đổi thông tin theo thời gian |
| 5 | **Thiếu organizational memory** | Không có bộ nhớ cấp tổ chức, multi-agent |
| 6 | **Governance / Audit gap** | Không kiểm soát được AI nhớ gì, từ đâu, ai tạo |
| 7 | **Schema enforcement thiếu** | Knowledge graph không có validation, ontology |

### 2.2 Hạn chế giải pháp hiện tại

| Hệ thống | Mạnh về | Thiếu |
|---|---|---|
| Zep | Conversational memory | Graph governance, procedural memory |
| Mem0 | Lightweight memory | Temporal reasoning, enterprise features |
| Graphiti | Temporal graph memory | Context assembly, multi-tenant governance |
| Cognee | Extraction pipeline | Session management, filesystem paradigm |
| Neo4j stack | Graph reasoning | Memory orchestration, agent integration |
| Weaviate/Pinecone | Retrieval | Relationship awareness, governance |
| LangGraph | Orchestration | Persistent memory, knowledge graph |

> **Không ai unify toàn bộ stack.** VNP Memory giải quyết bằng cách tích hợp các engine chuyên biệt dưới một governance layer thống nhất.

---

## 3. Target Users

### 3.1 Primary Users

| Persona | Nhu cầu | Use Case |
|---|---|---|
| **AI Agent Developer** | Memory SDK cho agent | Chatbot, coding assistant, support bot |
| **Platform Engineer** | Self-host & scale memory infra | Multi-tenant deployment, monitoring |
| **ML/AI Engineer** | Tối ưu context quality | Ontology tuning, retrieval evaluation |
| **Enterprise Architect** | Governance & compliance | Audit trail, GDPR, tenant isolation |

### 3.2 Secondary Users

| Persona | Nhu cầu |
|---|---|
| **AI Framework Author** | Integration SDK (LangChain, CrewAI, AutoGen) |
| **Data Scientist** | Knowledge graph analysis, pattern discovery |
| **DevOps Team** | Container orchestration, health monitoring |

---

## 4. Product Architecture

### 4.1 Kiến trúc tổng thể — 4-Layer

```
                    Applications / AI Agents
                            │
                     Memory API Gateway
                            │
    ┌──────────┬──────────┬──────────┬──────────┬──────────┐
    │          │          │          │          │          │
 Episodic  Semantic  Conversa-  Procedural  Profile  Adaptive
 (time)    (facts)   tional     (workflows) (user)   (KG+forget)
    │          │     (sessions)     │          │          │
 Graphiti   Cognee    Zep      OpenViking  Memobase  Supermemory
    │          │          │          │          │          │
    └──────────┴──────────┴──────────┴──────────┴──────────┘
                            │
  ╔═════════════════════════╧═══════════════════════════╗
  ║        KGS Platform (5-Layer Architecture)          ║
  ╠═════════════════════════════════════════════════════╣
  ║  L5 — Transport    gRPC/HTTP · Middleware · Workers ║
  ║  L4 — Governance   Registry · Ontology · Rules      ║
  ║  L3 — Query & Intelligence   Planner · Search       ║
  ║  L2 — Sync & Processing     Outbox · Batch · Overlay║
  ║  L1 — Storage       PG · Neo4j · Qdrant · Redis     ║
  ╚═════════════════════════════════════════════════════╝
```

### 4.2 Memory Engine Roles

| Engine | Layer | Vai trò | Technology |
|---|---|---|---|
| **Graphiti** | Episodic | Temporal context graph, fact validity tracking, provenance | Python, Neo4j/FalkorDB/Kuzu |
| **Cognee** | Semantic | Knowledge extraction pipeline, multi-modal ingestion, Graph RAG | Python, Neo4j/Kuzu, LanceDB |
| **Zep** | Conversational | Session memory, context assembly < 200ms, user/session CRUD | Go, PostgreSQL, Neo4j |
| **OpenViking** | Procedural | Virtual filesystem (viking://), tiered context (L0/L1/L2), session management | Python/Rust, RAGFS |
| **KGS Platform** | Governance | Multi-tenant graph service, ontology, ABAC, rule engine | Go, PostgreSQL, Neo4j, Qdrant |

---

## 5. Core Features

### 5.1 Unified Memory API

API thống nhất cho tất cả memory operations:

```python
# Store — route tới engine phù hợp
memory.store(data, type="episodic|semantic|conversational|procedural")

# Recall — hybrid retrieval across engines
memory.recall(query, scope="user|org|global")

# Evolve — tự động enrichment qua KGS Rule Engine
memory.evolve(dataset)

# Invalidate — temporal fact management
memory.invalidate(fact_id, reason="superseded")

# Timeline — Graphiti temporal query
memory.timeline(entity, time_range)
```

### 5.2 Episodic Memory (Graphiti)

| Feature | Mô tả | Priority |
|---|---|---|
| **Temporal Fact Management** | Mỗi fact có validity window (valid_at/invalid_at) | P0 |
| **Episode Ingestion** | Ingest từ text, JSON, message, fact_triple | P0 |
| **Entity/Edge Extraction** | LLM-powered extraction với deduplication | P0 |
| **Hybrid Search** | Semantic + BM25 + Graph traversal + Multi-layer reranking | P0 |
| **Prescribed Ontology** | Định nghĩa entity/edge types qua Pydantic models | P1 |
| **Community Detection** | Clustering algorithm cho entity communities | P1 |
| **Saga Management** | Nhóm episodes liên quan thành sequences | P1 |

### 5.3 Semantic Memory (Cognee)

| Feature | Mô tả | Priority |
|---|---|---|
| **Multi-modal Ingestion** | PDF, text, audio, image, CSV, URL | P0 |
| **Knowledge Graph Construction** | Document → Entity extraction → Graph | P0 |
| **15+ Search Types** | GRAPH_COMPLETION, RAG, TEMPORAL, CYPHER, etc. | P0 |
| **V2 Memory API** | remember(), recall(), forget(), improve() | P0 |
| **NodeSets** | Memory scoping per customer/workflow/topic | P1 |
| **Custom Pipelines** | Extensible task pipeline architecture | P1 |
| **Feedback Loop** | Self-improvement qua interaction feedback | P1 |
| **Ontology Grounding** | RDF/XML ontology support | P2 |

### 5.4 Conversational Memory (Zep)

| Feature | Mô tả | Priority |
|---|---|---|
| **User/Session Management** | CRUD với metadata, soft-delete, multi-tenant | P0 |
| **Message Ingestion** | Role-typed messages (user/assistant/system/tool) | P0 |
| **Graph RAG** | Automatic relationship extraction, temporal KG | P0 |
| **Context Assembly** | Pre-formatted blocks < 200ms latency | P0 |
| **Multi-SDK** | Python, TypeScript, Go SDKs | P1 |
| **Framework Integrations** | AutoGen, CrewAI, ADK, LiveKit | P1 |
| **Eval Harness** | Reproducible evaluation pipeline | P2 |

### 5.5 Procedural Memory (OpenViking)

| Feature | Mô tả | Priority |
|---|---|---|
| **Virtual Filesystem** | viking:// URI, unified namespace | P0 |
| **Three-Tier Context** | L0 Abstract (~100 tok) → L1 Overview (~2K) → L2 Detail (full) | P0 |
| **Hierarchical Retrieval** | 5-step pipeline: Intent → Global → Recursive → Rerank → Aggregate | P0 |
| **Session Management** | 2-phase commit, Working Memory v2 | P0 |
| **Resource Ingestion** | Git repos, HTTP/HTTPS, local files, documents | P1 |
| **Multi-channel Bot** | Telegram, Slack, Discord, Feishu | P2 |

### 5.6 Knowledge Governance (KGS Platform)

| Feature | Mô tả | Priority |
|---|---|---|
| **Multi-tenant Isolation** | Namespace labels `{APP_ID}__{Type}` trên Neo4j | P0 |
| **Ontology Service** | Schema validation, JSON Schema, constraint sync | P0 |
| **Policy Engine (OPA)** | ABAC policies per entity type, per tenant | P0 |
| **App Registry** | Tenant lifecycle, API keys, quotas | P0 |
| **Rule Engine** | Cron + event-driven rules, auto-enrichment | P1 |
| **Query Planner** | Cypher generation, namespace injection, guardrails | P1 |
| **Hybrid Search** | Vector + text + centrality reranking | P1 |
| **Overlay Graphs** | Commit/discard/conflict resolution | P2 |

---

## 6. Memory API Gateway

### 6.1 API Design

Gateway thống nhất routing requests tới engine phù hợp:

| Method | Endpoint | Engine Target | Mô tả |
|---|---|---|---|
| POST | `/v1/memory/store` | Router → Engine | Store memory (auto-route by type) |
| GET | `/v1/memory/recall` | All engines | Hybrid recall across engines |
| POST | `/v1/memory/episodes` | Graphiti | Ingest temporal episodes |
| GET | `/v1/memory/timeline` | Graphiti | Temporal query |
| POST | `/v1/memory/cognify` | Cognee | Build knowledge graph |
| GET | `/v1/memory/search` | Cognee/Zep | Multi-mode search |
| POST | `/v1/memory/sessions` | Zep/OpenViking | Session management |
| GET | `/v1/memory/context` | OpenViking | Tiered context retrieval |
| POST | `/v1/graph/nodes` | KGS | Create graph node |
| GET | `/v1/graph/nodes/{id}/context` | KGS | Context subgraph |

### 6.2 Authentication & Multi-tenancy

| Mechanism | Mô tả |
|---|---|
| **API Key** | Per-tenant keys, hash → app_id lookup |
| **JWT** | Session-based authentication |
| **Namespace Injection** | Auto-inject tenant prefix vào mọi query |
| **ABAC (OPA)** | Attribute-based access control per entity type |

### 6.3 MCP Server (Model Context Protocol)

Expose memory operations cho AI assistants:

| Tool | Engine | Mô tả |
|---|---|---|
| `memory_store` | Router | Store memory with auto-routing |
| `memory_recall` | All | Hybrid recall |
| `search` | Cognee/OpenViking | Semantic search |
| `timeline` | Graphiti | Temporal query |
| `context` | OpenViking | Tiered context |
| `graph_query` | KGS | Graph operations |

**Transport**: stdio, SSE, HTTP Streamable

---

## 7. Technical Architecture

### 7.1 Technology Stack

| Layer | Technology |
|---|---|
| **Memory API Gateway** | Go / Python (FastAPI) |
| **Graphiti Engine** | Python 3.10+, graphiti-core |
| **Cognee Engine** | Python 3.10+, cognee SDK |
| **Zep Engine** | Go 1.21+, go-chi |
| **OpenViking Engine** | Python 3.10+, Rust (CLI/RAGFS) |
| **KGS Platform** | Go, Kratos framework |
| **Graph Database** | Neo4j 5.22+ (primary), FalkorDB, Kuzu |
| **Relational Database** | PostgreSQL (source-of-truth) |
| **Vector Database** | Qdrant, LanceDB, pgvector |
| **Cache/Streaming** | Redis, NATS |
| **Policy Engine** | OPA (Open Policy Agent) |
| **Observability** | OpenTelemetry, Prometheus |
| **Container** | Docker, Kubernetes (Helm) |
| **AI Gateway** | Bifrost (multi-provider LLM routing) |

### 7.2 Storage Strategy — Dual Backend

| Stack | Components | Use Case |
|---|---|---|
| **Specialized** (Production) | Neo4j + PostgreSQL + Qdrant + Redis | Battle-tested, scale independently |
| **Unified** (Planned) | SurrealDB | Simplified infra, multi-model |

> Query Planner (KGS L2) abstract hóa storage backend. Cả hai stack có thể chạy song song.

### 7.3 Data Flow — Event-Driven CQRS

```
Agent Request → Memory API Gateway → Engine Selection
    │
    ├─→ Graphiti: Episode → Extract → Deduplicate → Graph
    ├─→ Cognee: Add → Cognify → Knowledge Graph
    ├─→ Zep: Message → Graphiti ingestion → Context assembly
    └─→ OpenViking: Data → Parse → Chunk → Embed → VikingFS
         │
         ▼
    KGS Platform
    ├─ L4: Ontology validation + Policy check
    ├─ L3: Namespace injection + Query planning
    ├─ L2: Outbox → Fan-out to Neo4j + Qdrant
    └─ L1: PostgreSQL (write) → Neo4j/Qdrant (read replicas)
```

---

## 8. Deployment

### 8.1 Deployment Models

| Model | Mô tả |
|---|---|
| **Development** | Docker Compose, all services local |
| **Staging** | Kubernetes, separate namespaces |
| **Production** | Kubernetes + Helm, HA configuration |

### 8.2 Service Topology

| Service | Port | Health Check |
|---|---|---|
| Memory API Gateway | 8080 | `/healthz` |
| Cognee | 8000 | `/health` |
| Graphiti | 8001 | `/healthz` |
| Zep | 8002 | `/healthz` |
| OpenViking | 1933 | `/health` |
| KGS Platform | 9000 | `/healthz` |
| Bifrost (AI Gateway) | 8443 | `/health` |
| Neo4j | 7687 | bolt |
| PostgreSQL | 5432 | pg_isready |
| Qdrant | 6333 | `/healthz` |
| Redis | 6379 | PING |

---

## 9. Security & Compliance

### 9.1 Security Requirements

| Requirement | Implementation |
|---|---|
| **Authentication** | API Key + JWT + Cookie-based |
| **Authorization** | OPA ABAC policies per tenant/entity |
| **Tenant Isolation** | Namespace labels, query injection |
| **Encryption** | TLS in transit, AES-256-GCM at rest (OpenViking) |
| **Secret Redaction** | Auto-redact API keys, tokens in logs/traces |
| **Raw Cypher Prevention** | Whitelist operations only, no raw queries |

### 9.2 Governance Features

| Feature | Mô tả |
|---|---|
| **Memory TTL** | Configurable expiration per memory type |
| **Retention Policy** | Auto-archive/delete based on rules |
| **GDPR Forget** | Cascading deletion across all engines |
| **Role-scoped Memory** | ABAC per entity type per tenant |
| **Audit Trail** | Rule execution history + event outbox |
| **Provenance Tracking** | Namespace metadata per node/edge |
| **Memory Lifecycle** | ACTIVE → SUSPENDED → DELETED |

---

## 10. Observability

| Signal | Implementation |
|---|---|
| **Tracing** | OpenTelemetry distributed tracing across all engines |
| **Metrics** | Prometheus (latency, throughput, error rates) |
| **Logging** | Structured logging (structlog/slog), JSON format |
| **Health** | Per-service health endpoints |
| **Dashboard** | Grafana dashboards for memory operations |

---

## 11. Non-Functional Requirements

### 11.1 Performance

| Metric | Target |
|---|---|
| Context retrieval latency (p95) | < 200ms (Zep), < 500ms (OpenViking), < 1000ms (Graphiti) |
| Knowledge graph construction | Background async, < 30s per document |
| Concurrent sessions | ≥ 1,000 per instance |
| Token cost reduction | ≥ 80% vs naive context stuffing |

### 11.2 Scalability

| Dimension | Approach |
|---|---|
| **Horizontal** | Stateless API gateway, engine replicas |
| **Storage** | Each engine scales independently |
| **Multi-tenant** | KGS namespace isolation, quota per app |
| **Distributed** | Queue-based processing (Cognee Modal workers) |

### 11.3 Reliability

| Requirement | Implementation |
|---|---|
| **Availability** | ≥ 99.9% API uptime |
| **Fallback LLM** | Bifrost multi-provider routing |
| **Retry Logic** | Tenacity-based retry with backoff |
| **Data Durability** | PostgreSQL ACID + async replication |

---

## 12. Integration & Interoperability

### 12.1 Agent Framework Support

| Framework | Integration Method |
|---|---|
| LangChain | Python SDK |
| LangGraph | Python SDK |
| CrewAI | ZepUserStorage + ZepGraphStorage |
| AutoGen | autogen_core.memory.Memory interface |
| OpenAI Agents SDK | REST API / MCP |
| Claude Tools | MCP Server |
| Google ADK | zep-adk integration |

### 12.2 LLM Provider Support

Qua **Bifrost AI Gateway**:
- OpenAI, Azure OpenAI, Anthropic, Google Gemini
- Groq, Mistral, Ollama (local)
- HuggingFace, llama.cpp
- OpenRouter (multi-model)

### 12.3 IDE Plugins

| Plugin | Protocol |
|---|---|
| Claude Code Memory | MCP / CLI |
| OpenCode Memory | MCP / CLI |
| Codex Memory | MCP |

---

## 13. Competitive Moat

### 13.1 Memory Orchestration Engine
- Merge memories across engines
- Resolve conflicts (temporal, semantic)
- Salience ranking & adaptive retrieval

### 13.2 Context Compiler
```
Input:  task + user + org + time + policies
Output: optimized context package (minimal tokens, max relevance)
```

### 13.3 Cognitive Policies
- Memory TTL, retention, GDPR forget
- Role-scoped memory, confidential classes
- OPA policies + Rule Engine = production-ready

### 13.4 Knowledge Graph Governance (Unique)
- **Ontology-as-config** — schema changes without redeploy
- **Namespace isolation** — multi-tenant on shared infra
- **Rule-driven enrichment** — auto-create edges, validate consistency
- **Policy-driven access** — ABAC per entity type, per tenant

---

## 14. Roadmap

### Phase 1 — Conversational Memory (Current)
- ✅ Zep deployment với session/user management
- ✅ Cognee deployment với knowledge graph pipeline
- ✅ Graphiti deployment với temporal context graph
- ✅ OpenViking deployment với virtual filesystem
- ✅ Bifrost AI Gateway cho multi-provider LLM
- 🔲 KGS Platform namespace isolation integration
- 🔲 Unified Memory API Gateway

### Phase 2 — Graph Memory
- 🔲 Graphiti + KGS ontology integration
- 🔲 KGS Rule Engine cho auto-enrichment
- 🔲 Cross-engine memory deduplication
- 🔲 Unified search across all engines

### Phase 3 — Organizational Memory
- 🔲 KGS multi-tenant full deployment
- 🔲 OPA policies per organization
- 🔲 Context Compiler (task + user + org → optimized context)
- 🔲 Memory observability dashboard

### Phase 4 — Autonomous Memory Optimization
- 🔲 Memory decay & salience scoring
- 🔲 Auto-summarization hierarchy
- 🔲 SurrealDB unified storage evaluation
- 🔲 Streaming ingestion & real-time subscriptions

---

## 15. Success Metrics

| Metric | Target | Measurement |
|---|---|---|
| Context retrieval latency (p95) | < 500ms | Prometheus/Grafana |
| Task completion improvement | ≥ 40% vs vanilla RAG | Eval harness |
| Token cost reduction | ≥ 80% | Usage monitoring |
| Memory extraction accuracy | ≥ 85% relevance | Eval framework |
| Tenant isolation verification | 100% (zero cross-tenant leaks) | Security audit |
| API uptime | ≥ 99.9% | Health monitoring |
| Agent framework integrations | ≥ 5 frameworks | SDK availability |

---

## 16. Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| **Integration complexity** | 4 engines + KGS = complex orchestration | Phased rollout, engine-by-engine |
| **Semantic consistency** | Contradicting facts across engines | Graphiti temporal invalidation + KGS Rule Engine |
| **Memory explosion** | Token cost & retrieval noise tăng | Salience scoring, memory decay, quota (KGS) |
| **Context assembly latency** | Agent cần < 1s response | Precomputed neighborhoods, hot cache, tiered loading |
| **Multi-tenant isolation** | Cross-tenant data leak = deal-breaker | KGS namespace labels + OPA policies + query injection |
| **LLM provider dependency** | Single provider failure | Bifrost multi-provider failover |

---

## 17. Appendix — Component PRD References

| Component | PRD Location | Version |
|---|---|---|
| Cognee | `services/cognee/docs/PRD.md` | 1.0.2 |
| Graphiti | `services/graphiti/docs/PRD.md` | 0.28.2 |
| Zep | `services/zep/docs/PRD.md` | 1.0 |
| OpenViking | `services/OpenViking/docs/PRD.md` | 0.1.x |
| KGS Platform | `docs/requirements.md` → Architecture refs | — |

---

## 18. Design Decisions & Trade-offs

| Decision | Choice | Rationale | Trade-off |
|---|---|---|---|
| Multi-engine vs monolith | Multi-engine | Mỗi engine chuyên biệt 1 memory type | Integration complexity cao |
| Multi-tenancy | Shared graph + namespace (KGS) | Tối ưu chi phí, dễ migrate | Logical isolation, không physical |
| Raw Cypher | KHÔNG cho phép | Bảo mật namespace, guardrails | App mất flexibility |
| Ontology storage | PostgreSQL | ACID, config data | Thêm store cần sync |
| Rule execution | Async (Redis Streams) | Không block API, dễ retry | Delay giữa event và execution |
| Access Control | OPA | Mature, auditable, Rego | Cần maintain OPA bundle |
| Storage strategy | Dual-backend (Specialized + SurrealDB planned) | Proven stack now, simplified later | Higher infra complexity hiện tại |
