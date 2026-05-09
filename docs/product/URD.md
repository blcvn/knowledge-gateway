# User Requirements Document (URD)

## VNP Memory — Unified Cognitive Infrastructure Layer for Enterprise AI

| Field | Value |
|---|---|
| **Product** | VNP Memory |
| **Version** | 1.0.0 |
| **Status** | Development |
| **Last Updated** | 2026-05-09 |
| **PRD Reference** | [PRD.md](PRD.md) |

---

## 1. Mục đích tài liệu

Tài liệu này mô tả yêu cầu từ phía người dùng đối với VNP Memory — nền tảng Unified Cognitive Infrastructure cho Enterprise AI. URD tập trung vào **ai dùng**, **dùng để làm gì**, **tương tác như thế nào**, và **trải nghiệm mong đợi**, là cầu nối giữa PRD và Technical Specifications.

---

## 2. User Personas

### 2.1 P1 — AI Agent Developer (Primary)

| Thuộc tính | Mô tả |
|---|---|
| **Vai trò** | Xây dựng AI Agent có persistent memory |
| **Kỹ năng** | Python/TypeScript/Go, async, LLM APIs, REST |
| **Mục tiêu** | Thêm bộ nhớ dài hạn vào agent mà không tự xây infrastructure |
| **Pain Points** | Agent mất ngữ cảnh giữa phiên; RAG không hiểu quan hệ; memory fragmented |
| **Use Cases** | Customer support bots, coding assistants, research agents |
| **Tần suất** | Hàng ngày |

**User Stories:**
- US-P1-01: Tôi muốn lưu memory cho agent bằng 1 API call (`memory.store()`) mà không cần biết engine nào xử lý.
- US-P1-02: Tôi muốn recall context liên quan từ tất cả memory layers qua 1 query thống nhất.
- US-P1-03: Tôi muốn agent tự cải thiện qua feedback loop mà không cần rebuild graph.
- US-P1-04: Tôi muốn truy vấn temporal timeline (ai nói gì, khi nào, còn đúng không).
- US-P1-05: Tôi muốn tổ chức context theo cấu trúc phân tầng (abstract → overview → detail).
- US-P1-06: Tôi muốn hệ thống tự động xây dựng user profile có cấu trúc từ conversations.
- US-P1-07: Tôi muốn AI tự động quên thông tin hết hạn hoặc bị thay thế.

### 2.2 P2 — Platform / DevOps Engineer

| Thuộc tính | Mô tả |
|---|---|
| **Vai trò** | Triển khai và vận hành VNP Memory infrastructure |
| **Kỹ năng** | Docker, Kubernetes, monitoring, CI/CD |
| **Mục tiêu** | Self-host, scale, HA, multi-tenant isolation |
| **Pain Points** | Complexity nhiều engines; monitoring fragmented; tenant isolation |
| **Use Cases** | Production deployment, scaling, security hardening |
| **Tần suất** | Hàng tuần |

**User Stories:**
- US-P2-01: Tôi muốn deploy toàn bộ stack bằng `docker compose up`.
- US-P2-02: Tôi muốn monitor tất cả engines qua Prometheus/Grafana thống nhất.
- US-P2-03: Tôi muốn cấu hình multi-tenant isolation để dữ liệu giữa tenants không bị trộn lẫn.
- US-P2-04: Tôi muốn quản lý API keys và quotas per tenant qua KGS Registry.
- US-P2-05: Tôi muốn health checks cho tất cả services từ một endpoint.

### 2.3 P3 — ML/AI Engineer

| Thuộc tính | Mô tả |
|---|---|
| **Vai trò** | Tối ưu context quality, ontology tuning |
| **Kỹ năng** | Python, graph theory, NLP, evaluation |
| **Mục tiêu** | Maximize retrieval accuracy, domain-specific optimization |
| **Pain Points** | Generic ontology miss domain entities; khó đo retrieval quality |
| **Use Cases** | Ontology design, search tuning, eval pipeline |
| **Tần suất** | Hàng tuần |

**User Stories:**
- US-P3-01: Tôi muốn định nghĩa custom ontology (entity/edge types) cho domain cụ thể.
- US-P3-02: Tôi muốn chạy evaluation pipeline để đo context completeness và answer accuracy.
- US-P3-03: Tôi muốn so sánh search strategies (Graph RAG vs RAG vs Temporal) trên cùng dataset.
- US-P3-04: Tôi muốn phân tích knowledge graph để phát hiện patterns và insights.

### 2.4 P4 — Enterprise Architect

| Thuộc tính | Mô tả |
|---|---|
| **Vai trò** | Governance, compliance, security architecture |
| **Kỹ năng** | Security policies, ABAC, audit |
| **Mục tiêu** | Đảm bảo AI memory tuân thủ compliance (GDPR, audit trail) |
| **Pain Points** | Không kiểm soát được AI nhớ gì, từ đâu, ai tạo |
| **Use Cases** | Tenant isolation audit, GDPR forget, memory governance |
| **Tần suất** | Hàng tháng |

**User Stories:**
- US-P4-01: Tôi muốn audit trail cho mọi memory operation (ai tạo, khi nào, từ đâu).
- US-P4-02: Tôi muốn GDPR forget cascading xóa memory across tất cả engines.
- US-P4-03: Tôi muốn ABAC policies kiểm soát ai được đọc/ghi entity type nào.
- US-P4-04: Tôi muốn memory TTL và retention policies tự động expire data.

### 2.5 P5 — IDE Plugin User (Secondary)

| Thuộc tính | Mô tả |
|---|---|
| **Vai trò** | Developer dùng AI coding assistants (Claude Code, Codex) |
| **Mục tiêu** | AI assistant nhớ project context giữa các phiên |
| **Pain Points** | AI quên context sau mỗi session |
| **Tần suất** | Hàng ngày |

**User Stories:**
- US-P5-01: Tôi muốn AI assistant nhớ coding preferences và project structure.
- US-P5-02: Tôi muốn AI tìm kiếm trong codebase đã index qua MCP tools.
- US-P5-03: Tôi muốn nói "remember this" và AI lưu vào long-term memory.

### 2.6 P6 — AI Framework Integrator (Secondary)

| Thuộc tính | Mô tả |
|---|---|
| **Vai trò** | Xây integration cho AI frameworks (AutoGen, CrewAI, LangChain) |
| **Mục tiêu** | Implement framework-native memory interface backed by VNP Memory |
| **Tần suất** | Theo dự án |

**User Stories:**
- US-P6-01: Tôi muốn VNP Memory implement standardized memory interface của framework.
- US-P6-02: Tôi muốn SDK với type hints đầy đủ và async support.

### 2.7 P7 — AI Power User (Secondary)

| Thuộc tính | Mô tả |
|---|---|
| **Vai trò** | Người dùng cuối tương tác qua AI assistant hàng ngày |
| **Mục tiêu** | AI cá nhân hóa, nhớ sở thích và context giữa các phiên |
| **Pain Points** | AI quên mọi thứ sau mỗi session; không cá nhân hóa |
| **Tần suất** | Hàng ngày |

**User Stories:**
- US-P7-01: Tôi muốn AI nhớ sở thích, dự án, và thảo luận trước đó giữa các phiên.
- US-P7-02: Tôi muốn nói "quên đi" và AI thực sự quên thông tin đó.
- US-P7-03: Tôi muốn xem và quản lý bộ nhớ AI đã lưu qua dashboard.
- US-P7-04: Tôi muốn memory graph visualization để hiểu AI nhớ gì.

### 2.8 P8 — Product Manager (Secondary)

| Thuộc tính | Mô tả |
|---|---|
| **Vai trò** | Quản lý sản phẩm AI có tương tác người dùng |
| **Mục tiêu** | Phân tích hành vi người dùng qua structured profiles |
| **Tần suất** | Hàng tuần |

**User Stories:**
- US-P8-01: Tôi muốn xem user profiles có cấu trúc (topic/sub_topic/content) từ conversations.
- US-P8-02: Tôi muốn theo dõi usage analytics và token consumption.

---

## 3. User Requirements

### 3.1 Unified Memory Operations (UR-MEM)

#### UR-MEM-01: Store — Lưu memory thống nhất
**Persona:** P1, P5 | **Priority:** Critical

**Mô tả:** Người dùng lưu memory qua 1 API thống nhất, hệ thống tự route tới engine phù hợp.

**Acceptance Criteria:**
- API: `memory.store(data, type="episodic|semantic|conversational|procedural|profile|adaptive")`
- Auto-routing: text messages → Zep, documents → Cognee, temporal events → Graphiti, structured context → OpenViking, chat conversations → Memobase (profile), general content → Supermemory (adaptive KG)
- Background processing (non-blocking)
- Hỗ trợ session-scoped và permanent storage

#### UR-MEM-02: Recall — Truy xuất cross-engine
**Persona:** P1, P5 | **Priority:** Critical

**Mô tả:** Truy xuất context liên quan từ tất cả memory engines qua 1 query.

**Acceptance Criteria:**
- API: `memory.recall(query, scope="user|org|global")`
- Hybrid retrieval: semantic + graph + temporal + hierarchical + profile + adaptive KG
- Kết quả merged và ranked từ tất cả engines (bao gồm Memobase profiles và Supermemory memories)
- Latency < 500ms (p95), profile recall < 100ms

#### UR-MEM-03: Timeline — Temporal queries
**Persona:** P1, P3 | **Priority:** High

**Mô tả:** Truy vấn timeline sự kiện với temporal validity.

**Acceptance Criteria:**
- Query facts tại một thời điểm cụ thể (`valid_at`)
- Xem lịch sử thay đổi của entity
- Auto-invalidation khi facts mâu thuẫn
- Provenance tracking (nguồn gốc mỗi fact)

#### UR-MEM-04: Forget — Xóa memory
**Persona:** P1, P4 | **Priority:** High

**Mô tả:** Xóa memory với cascading across tất cả engines.

**Acceptance Criteria:**
- Xóa theo dataset, user, tenant, hoặc specific entity
- Cascade: graph nodes + vector indices + session data + filesystem
- GDPR-compliant: complete data erasure
- Audit log cho deletion events

#### UR-MEM-05: Evolve — Self-improvement
**Persona:** P1 | **Priority:** Medium

**Mô tả:** Hệ thống tự cải thiện knowledge quality theo thời gian.

**Acceptance Criteria:**
- Graph enrichment không phá hủy data hiện tại
- Feedback loop: interaction → weight adjustment → better ranking
- Supermemory version control: updates/extends/derives memory relations
- Memobase profile merge (YOLO): auto-merge profiles với fixed 3 LLM calls
- Background processing

### 3.2b User Profile & Personalization (UR-PROFILE)

#### UR-PROFILE-01: Structured user profile extraction
**Persona:** P1, P7, P8 | **Priority:** High

**Mô tả:** Tự động xây dựng và duy trì user profile có cấu trúc từ conversations.

**Acceptance Criteria:**
- Profile format: topic/sub_topic/content (controllable schema)
- Buffer zone: batch processing, auto-flush khi > 1024 tokens hoặc idle > 1h
- Fixed 3 LLM calls per flush (extract → merge → events)
- Profile strict mode: chỉ collect theo schema đã định nghĩa
- Context API: prompt-ready string < 100ms, configurable token budget

#### UR-PROFILE-02: Adaptive memory with auto-forgetting
**Persona:** P1, P7 | **Priority:** High

**Mô tả:** Living knowledge graph tự động quên thông tin hết hạn, giải quyết mâu thuẫn.

**Acceptance Criteria:**
- Memory versioning: parent → root chain, isLatest flag
- Relation types: updates, extends, derives
- Time-based forgetting (forgetAfter)
- Contradiction resolution (auto-update isLatest)
- Noise filtering: meaningless content không thành permanent memory
- Static vs Dynamic memory classification

#### UR-PROFILE-03: External data connectors
**Persona:** P1 | **Priority:** Medium

**Mô tả:** Tự động sync dữ liệu từ external sources vào memory.

**Acceptance Criteria:**
- Google Drive, Gmail, Notion, OneDrive, GitHub connectors
- Real-time webhooks + cron sync mỗi 4h
- Custom OAuth keys per provider
- Document limit configurable per connection

### 3.2 Data Ingestion (UR-ING)

#### UR-ING-01: Multi-modal ingestion
**Persona:** P1, P3 | **Priority:** Critical

**Mô tả:** Nhập dữ liệu từ nhiều nguồn và định dạng.

**Acceptance Criteria:**
- Text, PDF, DOCX, PPTX, CSV, audio, image, video (transcription)
- URLs (web scraping), Git repositories
- Structured data (JSON, fact triples)
- Chat messages (role-typed: user/assistant/system/tool)
- ChatBlob, DocBlob, SummaryBlob (Memobase format)
- Code files (AST-aware chunking via Supermemory)
- Tổ chức theo dataset/namespace/containerTag

#### UR-ING-02: Knowledge graph construction
**Persona:** P1, P3 | **Priority:** Critical

**Mô tả:** Tự động chuyển đổi dữ liệu thô thành knowledge graph.

**Acceptance Criteria:**
- Pipeline: classify → chunk → extract entities → build graph → embed
- Entity deduplication tự động
- Relationship/edge extraction với temporal validity
- Custom prompt cho domain-specific extraction
- Configurable chunk_size

#### UR-ING-03: Tiered context generation
**Persona:** P1 | **Priority:** High

**Mô tả:** Tự động tạo context phân tầng cho mỗi resource.

**Acceptance Criteria:**
- L0 Abstract (~100 tokens): one-sentence summary
- L1 Overview (~2K tokens): core info + usage scenarios
- L2 Detail (full content): deep reading khi cần
- Load on demand: L0 → L1 → L2

### 3.3 Search & Retrieval (UR-SEARCH)

#### UR-SEARCH-01: Multi-strategy search
**Persona:** P1, P3 | **Priority:** Critical

**Mô tả:** Nhiều chiến lược tìm kiếm cho các use cases khác nhau.

**Acceptance Criteria:**
- **Semantic search**: Vector similarity (dense + sparse)
- **Graph completion**: Q&A kết hợp graph + LLM
- **RAG completion**: Traditional RAG over chunks
- **Temporal search**: Time-aware graph queries
- **Lexical search**: Keyword/exact-term matching (BM25)
- **Natural language**: NL → structured graph query
- **Hierarchical**: Directory recursive search with score propagation

#### UR-SEARCH-02: Context assembly
**Persona:** P1 | **Priority:** Critical

**Mô tả:** Assemble context từ nhiều nguồn thành format sẵn sàng cho LLM.

**Acceptance Criteria:**
- Pre-formatted context blocks (không cần format thủ công)
- Latency < 200ms (conversational), < 500ms (full hybrid)
- Token-optimized output
- Template support cho custom formatting

#### UR-SEARCH-03: Reranking & filtering
**Persona:** P1, P3 | **Priority:** High

**Acceptance Criteria:**
- Multi-strategy reranking: RRF, MMR, Cross-encoder, Node Distance
- Temporal filters: valid_at, invalid_at, created_at
- Label/type filtering cho precision control
- Hotness scoring và convergence detection

### 3.4 Session Management (UR-SESSION)

#### UR-SESSION-01: Session lifecycle
**Persona:** P1 | **Priority:** Critical

**Acceptance Criteria:**
- Create/end sessions với user association
- Message ingestion với role-typed messages
- Configurable lastN message history retrieval
- Session-scoped memory isolation

#### UR-SESSION-02: Working Memory
**Persona:** P1 | **Priority:** High

**Acceptance Criteria:**
- Auto-compress long conversations
- Extract long-term memory từ sessions (background)
- Working Memory v2: structured document (title, state, goals, facts, errors)
- 2-phase commit: archive → extract

### 3.5 Authentication & Multi-tenancy (UR-AUTH)

#### UR-AUTH-01: Authentication
**Persona:** P2, P4 | **Priority:** Critical

**Acceptance Criteria:**
- API Key authentication (per-tenant keys)
- JWT bearer token
- Dev mode: no auth, localhost only
- API key management: create, revoke, rotate

#### UR-AUTH-02: Multi-tenant isolation
**Persona:** P2, P4 | **Priority:** Critical

**Acceptance Criteria:**
- Namespace labels prevent cross-tenant data access
- API Key → tenant binding tại auth layer
- Query Planner auto-inject namespace vào mọi query
- Raw Cypher bị cấm — chỉ whitelist operations
- Per-tenant quotas (max_nodes, max_requests)

#### UR-AUTH-03: ABAC policies
**Persona:** P4 | **Priority:** High

**Acceptance Criteria:**
- OPA-based attribute-based access control
- Policies per entity type, per tenant, per role
- Rego language cho fine-grained rules
- Audit log cho policy evaluations

### 3.6 Agent & Framework Integration (UR-AGENT)

#### UR-AGENT-01: MCP Server
**Persona:** P5 | **Priority:** High

**Acceptance Criteria:**
- Transport: stdio (Claude Desktop), SSE, HTTP Streamable
- Tools: store, recall, search, timeline, context, graph_query
- Read & write operations
- Docker deployment support

#### UR-AGENT-02: Framework SDKs
**Persona:** P1, P6 | **Priority:** High

**Acceptance Criteria:**
- Python SDK (async-first), TypeScript SDK, Go SDK
- LangChain, CrewAI, AutoGen, Google ADK integrations
- Vercel AI SDK, OpenAI Agents SDK, Mastra, n8n (via Supermemory)
- Type hints đầy đủ, comprehensive examples
- Framework-native memory interface compliance

#### UR-AGENT-03: IDE plugins
**Persona:** P5 | **Priority:** Medium

**Acceptance Criteria:**
- Claude Code Memory plugin (MCP/CLI)
- OpenCode Memory plugin (MCP/CLI)
- Codex Memory plugin (MCP)
- Zero-config preferred

### 3.7 Deployment & Operations (UR-OPS)

#### UR-OPS-01: Quick start
**Persona:** P1, P2 | **Priority:** Critical

**Acceptance Criteria:**
- `docker compose up` cho full stack
- Minimal env vars: LLM_API_KEY + API keys
- Health checks tích hợp cho tất cả services
- Documentation và getting started guide

#### UR-OPS-02: Production deployment
**Persona:** P2 | **Priority:** High

**Acceptance Criteria:**
- Kubernetes deployment với Helm charts
- Resource limits configurable
- Horizontal scaling cho stateless services
- Backup & restore procedures

#### UR-OPS-03: Observability
**Persona:** P2 | **Priority:** High

**Acceptance Criteria:**
- OpenTelemetry tracing across tất cả engines
- Prometheus metrics (latency, throughput, errors)
- Structured logging (JSON format)
- Grafana dashboards cho memory operations
- Secret auto-redaction trong traces/logs

### 3.8 Governance (UR-GOV)

#### UR-GOV-01: Ontology management
**Persona:** P3, P4 | **Priority:** High

**Acceptance Criteria:**
- Define entity/edge types qua Pydantic models hoặc JSON Schema
- Ontology-as-config: thay đổi không cần redeploy
- Schema validation trên mọi write operation
- Relation whitelist check

#### UR-GOV-02: Memory lifecycle
**Persona:** P4 | **Priority:** High

**Acceptance Criteria:**
- Memory TTL configurable per type
- Retention policies (auto-archive/delete)
- Memory lifecycle: ACTIVE → SUSPENDED → DELETED
- Rule-driven memory decay

#### UR-GOV-03: Audit & provenance
**Persona:** P4 | **Priority:** High

**Acceptance Criteria:**
- Provenance tracking per node/edge (source, creator, timestamp)
- Rule execution history
- Event outbox cho audit trail
- Confidence scores per fact

---

## 4. Interaction Models

### 4.1 Unified Python SDK

```python
from vnp_memory import Memory

memory = Memory(api_key="...", endpoint="http://localhost:8080")

# Store (auto-routed)
await memory.store("User prefers dark mode", type="conversational")
await memory.store(document, type="semantic", dataset="docs")

# Recall (cross-engine)
results = await memory.recall("What does the user prefer?")

# Timeline (Graphiti)
events = await memory.timeline("user_123", time_range="last_30d")

# Forget (cascading)
await memory.forget(user_id="user_123", scope="all")
```

### 4.2 REST API Gateway

```
Application ────► Memory API Gateway (:8080)
                  │
                  ├── /v1/memory/store       → Auto-route to engine
                  ├── /v1/memory/recall      → Cross-engine retrieval
                  ├── /v1/memory/timeline    → Temporal queries
                  ├── /v1/memory/sessions    → Session management
                  ├── /v1/memory/search      → Multi-strategy search
                  ├── /v1/memory/context     → Tiered context (L0/L1/L2)
                  ├── /v1/graph/*            → KGS Graph operations
                  └── /healthz               → Unified health check
```

### 4.3 MCP Protocol (AI Assistants)

```
AI Assistant ─── MCP Protocol ───► VNP Memory MCP Server
(Claude/Codex)   (stdio/HTTP)      │
                                   ├── memory_store   (auto-route)
                                   ├── memory_recall  (cross-engine)
                                   ├── search         (semantic)
                                   ├── timeline       (temporal)
                                   ├── context        (tiered L0/L1/L2)
                                   └── graph_query    (KGS)
```

### 4.4 Engine-Specific Interaction (Advanced)

Khi cần truy cập trực tiếp engine cụ thể:

| Engine | SDK | Port | Use Case |
|---|---|---|---|
| Cognee | `import cognee` | 8000 | Knowledge extraction pipeline |
| Graphiti | `from graphiti_core import Graphiti` | 8001 | Temporal graph operations |
| Zep | `from zep_cloud import AsyncZep` | 8002 | Session/user management |
| OpenViking | `from openviking import OpenViking` | 1933 | Virtual filesystem, tiered context |
| Memobase | `from memobase import MemoBaseClient` | 8019 | User profiles, event timeline |
| Supermemory | `import supermemory` (TS/Python) | 8020 | Adaptive KG, hybrid search, connectors |

---

## 5. Standard Operating Procedures (SOPs)

### SOP-01: Quick Start — First-time Setup

| Step | Action | Chi tiết |
|---|---|---|
| 1 | Clone repository | `git clone <repo>` |
| 2 | Configure environment | Copy `.env.template` → `.env`, set API keys |
| 3 | Start stack | `docker compose up` |
| 4 | Verify health | `curl http://localhost:8080/healthz` |
| 5 | Test memory | Store → Recall via SDK or REST API |

### SOP-02: Agent Memory Integration

| Step | Action | Chi tiết |
|---|---|---|
| 1 | Install SDK | `pip install vnp-memory` |
| 2 | Initialize client | `memory = Memory(api_key="...", endpoint="...")` |
| 3 | Store conversation | `await memory.store(message, type="conversational")` |
| 4 | Build knowledge | `await memory.store(document, type="semantic")` |
| 5 | Recall context | `results = await memory.recall(query)` |
| 6 | Inject into prompt | Use results as LLM context |

### SOP-03: Multi-tenant Setup

| Step | Action | Chi tiết |
|---|---|---|
| 1 | Register tenant | `POST /v1/apps` (KGS Registry) |
| 2 | Issue API key | `POST /v1/apps/{id}/keys` |
| 3 | Define ontology | `POST /v1/ontology/entity-types` |
| 4 | Setup relations | `POST /v1/ontology/relation-types` |
| 5 | Configure policies | `POST /v1/policies` (OPA ABAC) |
| 6 | Set quotas | Configure max_nodes, rate limits |
| 7 | Use API | Tenant uses Memory API with their key |

### SOP-04: Production Deployment

| Step | Action | Chi tiết |
|---|---|---|
| 1 | Configure env | Set production database URLs, API keys |
| 2 | Deploy infra | PostgreSQL, Neo4j, Qdrant, Redis |
| 3 | Deploy engines | Kubernetes + Helm charts |
| 4 | Configure auth | Enable API key auth, disable dev mode |
| 5 | Setup monitoring | OpenTelemetry → Grafana dashboards |
| 6 | Verify isolation | Test cross-tenant access = blocked |
| 7 | Load test | Verify latency SLAs under load |

### SOP-05: MCP Server Setup cho IDE

| Step | Action | Chi tiết |
|---|---|---|
| 1 | Start VNP Memory | `docker compose up` |
| 2 | Configure MCP | Point IDE to `http://localhost:8080/mcp` |
| 3 | Set identity | API key in MCP config |
| 4 | Test | "search for X" → triggers MCP search tool |

### SOP-06: Ontology Configuration

| Step | Action | Chi tiết |
|---|---|---|
| 1 | Design schema | Define entity types & relations for domain |
| 2 | Register via API | `POST /v1/ontology/entity-types` |
| 3 | Test ingestion | Ingest sample data, verify graph structure |
| 4 | Tune extraction | Adjust custom prompts, chunk_size |
| 5 | Evaluate | Run eval harness, measure completeness |
| 6 | Iterate | Refine ontology based on metrics |

---

## 6. Non-Functional User Requirements

### 6.1 Performance (UR-NFR-PERF)

| ID | Requirement | Target |
|---|---|---|
| NFR-PERF-01 | Conversational context retrieval | < 200ms (p95) |
| NFR-PERF-02 | Hierarchical search latency | < 500ms (p95) |
| NFR-PERF-03 | Graph search latency | < 1000ms (p95) |
| NFR-PERF-04 | Knowledge graph construction | Background async, non-blocking |
| NFR-PERF-05 | Concurrent sessions | ≥ 1,000 per instance |
| NFR-PERF-06 | Token cost reduction vs naive RAG | ≥ 80% |
| NFR-PERF-07 | User profile retrieval (Memobase) | < 100ms (p95) |
| NFR-PERF-08 | Supermemory profile retrieval | < 50ms (p95) |
| NFR-PERF-09 | LLM calls per profile flush (Memobase) | Fixed 3 calls |

### 6.2 Reliability (UR-NFR-REL)

| ID | Requirement | Target |
|---|---|---|
| NFR-REL-01 | API uptime | ≥ 99.9% |
| NFR-REL-02 | Fallback LLM | Bifrost multi-provider failover |
| NFR-REL-03 | Retry logic | Automatic with exponential backoff |
| NFR-REL-04 | Data durability | PostgreSQL ACID + WAL |

### 6.3 Security (UR-NFR-SEC)

| ID | Requirement | Target |
|---|---|---|
| NFR-SEC-01 | Tenant isolation | Zero cross-tenant data leaks |
| NFR-SEC-02 | Encryption at-rest | AES-256-GCM (OpenViking) |
| NFR-SEC-03 | Secret redaction | Auto-redact in logs/traces |
| NFR-SEC-04 | API key storage | Hashed (SHA-256 or Argon2id) |
| NFR-SEC-05 | Network security | TLS in transit, localhost-only dev mode |

### 6.4 Usability (UR-NFR-USE)

| ID | Requirement | Target |
|---|---|---|
| NFR-USE-01 | Quick start | Working demo in < 10 minutes |
| NFR-USE-02 | SDK ergonomics | Async-first, type hints, Pythonic |
| NFR-USE-03 | Error messages | Structured, actionable, with guidance |
| NFR-USE-04 | Documentation | Getting started, API reference, examples |

---

## 7. Acceptance Criteria Summary

| Category | Criteria | Metric |
|---|---|---|
| **Unified API** | Store/Recall works across all 6 engines | All engines receive routed data |
| **Latency** | Context retrieval within SLA | < 500ms (p95) |
| **Completeness** | Retrieved context contains relevant facts | > 90% completeness |
| **Accuracy** | Agent answers match expectations | > 80% accuracy |
| **Isolation** | No cross-tenant data leakage | 0 leak incidents |
| **Integration** | MCP tools functional from IDE | All tools respond correctly |
| **Temporal** | Fact invalidation on contradiction | Auto-invalidation verified |
| **Profile** | Structured profiles extracted from conversations | topic/sub_topic/content format |
| **Auto-forget** | Expired/contradicted memories forgotten | Supermemory version chain updated |
| **Benchmark** | Memory quality on standard benchmarks | > 80% on LongMemEval |
| **Governance** | GDPR forget cascades across engines | Complete data erasure |
| **Deployment** | Full stack starts with docker compose | All health checks pass |
| **Observability** | Traces visible in Grafana | End-to-end tracing works |

---

## 8. Traceability Matrix

| User Requirement | PRD Section | Engine(s) |
|---|---|---|
| UR-MEM-01 (Store) | §5.1, §6.1 | Gateway → All |
| UR-MEM-02 (Recall) | §5.1, §6.1 | Gateway → All |
| UR-MEM-03 (Timeline) | §5.2 | Graphiti |
| UR-MEM-04 (Forget) | §9.2 | All engines + Supermemory auto-forget |
| UR-MEM-05 (Evolve) | §5.3, §5.7 | Cognee + Supermemory + KGS |
| UR-PROFILE-01 (User profiles) | §5.6 | Memobase |
| UR-PROFILE-02 (Adaptive memory) | §5.7 | Supermemory |
| UR-PROFILE-03 (Connectors) | §5.7 | Supermemory |
| UR-ING-01 (Multi-modal) | §5.3, §5.5, §5.7 | Cognee + OpenViking + Supermemory |
| UR-ING-02 (KG construction) | §5.2, §5.3, §5.7 | Graphiti + Cognee + Supermemory |
| UR-ING-03 (Tiered context) | §5.5 | OpenViking |
| UR-SEARCH-01 (Multi-strategy) | §5.2–5.7 | All engines |
| UR-SEARCH-02 (Assembly) | §5.4, §5.6 | Zep + Memobase |
| UR-SEARCH-03 (Reranking) | §5.2, §5.5, §5.7 | Graphiti + OpenViking + Supermemory |
| UR-SESSION-01 (Lifecycle) | §5.4, §5.5 | Zep + OpenViking |
| UR-SESSION-02 (Working Memory) | §5.5 | OpenViking |
| UR-AUTH-01 (Authentication) | §6.2, §9.1 | Gateway + KGS |
| UR-AUTH-02 (Multi-tenant) | §5.8, §9.1 | KGS Platform |
| UR-AUTH-03 (ABAC) | §5.8, §9.1 | KGS (OPA) |
| UR-AGENT-01 (MCP) | §6.3 | Gateway + Memobase + Supermemory |
| UR-AGENT-02 (SDKs) | §12 | All engines |
| UR-OPS-01 (Quick start) | §8 | All |
| UR-OPS-03 (Observability) | §10 | All engines |
| UR-GOV-01 (Ontology) | §5.8 | KGS Platform |
| UR-GOV-02 (Lifecycle) | §9.2 | KGS Platform + Supermemory |
| UR-GOV-03 (Audit) | §9.2 | KGS Platform |

---

## 9. Appendix — Component URD References

| Component | URD Location |
|---|---|
| Cognee | `services/cognee/docs/URD.md` |
| Graphiti | `services/graphiti/docs/URD.md` |
| Zep | `services/zep/docs/URD.md` |
| OpenViking | `services/OpenViking/docs/URD.md` |
| Memobase | `services/memobase/docs/URD.md` |
| Supermemory | `services/supermemory/docs/URD.md` |
