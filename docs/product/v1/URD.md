# User Requirements Document (URD)

## VNP Memory — Unified Cognitive Infrastructure Layer for Enterprise AI

| Field | Value |
|---|---|
| **Product** | VNP Memory |
| **Version** | 2.1.0 |
| **Status** | Active Development |
| **Last Updated** | 2026-09-03 |
| **PRD Reference** | [PRD.md](PRD.md) |

---

## 1. Mục đích tài liệu

Tài liệu này mô tả yêu cầu từ phía người dùng đối với VNP Memory — nền tảng Unified Cognitive Infrastructure cho Enterprise AI. URD tập trung vào **ai dùng**, **dùng để làm gì**, **tương tác như thế nào**, và **trải nghiệm mong đợi**.

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
- US-P1-01: Tôi muốn lưu memory cho agent bằng 1 API call (`POST /v1/memory/store`) mà không cần biết engine nào xử lý.
- US-P1-02: Tôi muốn recall context liên quan từ tất cả memory layers qua 1 query thống nhất (`POST /v1/memory/recall`).
- US-P1-03: Tôi muốn agent tự cải thiện qua feedback loop mà không cần rebuild graph (Supermemory versioning).
- US-P1-04: Tôi muốn truy vấn temporal timeline (`GET /v1/memory/timeline`) — ai nói gì, khi nào, còn đúng không.
- US-P1-05: Tôi muốn tổ chức context theo cấu trúc phân tầng L0/L1/L2 (OpenViking procedural memory).
- US-P1-06: Tôi muốn hệ thống tự động xây dựng user profile có cấu trúc từ conversations (Memobase YOLO engine).
- US-P1-07: Tôi muốn AI tự động quên thông tin hết hạn hoặc bị thay thế (Supermemory forgetAfter).

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
- US-P2-01: Tôi muốn start toàn bộ infrastructure bằng `make infra-up` rồi chạy monolith bằng `make dev`.
- US-P2-02: Tôi muốn monitor tất cả 35 services qua Prometheus/Grafana thống nhất (`/v1/console/observability/*`).
- US-P2-03: Tôi muốn cấu hình NATS embedded hoặc external qua `VNP_MEMORY_NATS_MODE`.
- US-P2-04: Tôi muốn quản lý tenants và API keys qua admin API (`POST /v1/admin/tenants`, `/v1/admin/tenants/{id}/keys`).
- US-P2-05: Tôi muốn health checks aggregated cho tất cả 35 services từ `GET /healthz` (port :8083).

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
- US-P3-01: Tôi muốn định nghĩa custom ontology cho Zep (`POST /v1/zep/graph/ontology`) và facts (`POST /v1/zep/graph/facts`).
- US-P3-02: Tôi muốn so sánh search strategies across engines (cognee-search vs graphiti-search vs sm-search).
- US-P3-03: Tôi muốn trace context assembly pipeline qua Agent Debugger (`POST /v1/console/debugger/trace`).
- US-P3-04: Tôi muốn monitor pipeline status per engine (`GET /v1/console/pipelines/status`).

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
- US-P4-01: Tôi muốn audit trail cho mọi memory operation (`GET /v1/console/governance/audit`).
- US-P4-02: Tôi muốn GDPR forget cascading xóa memory across tất cả engines (`POST /v1/console/governance/gdpr/forget`).
- US-P4-03: Tôi muốn preview impact trước khi xóa (`POST /v1/console/governance/gdpr/forget/preview`).
- US-P4-04: Tôi muốn OPA policies per entity type (`POST/PUT /v1/console/governance/policies`).

### 2.5 P5 — IDE Plugin User (Secondary)

| Thuộc tính | Mô tả |
|---|---|
| **Vai trò** | Developer dùng AI coding assistants (Claude Code, Codex) |
| **Mục tiêu** | AI assistant nhớ project context giữa các phiên |
| **Pain Points** | AI quên context sau mỗi session |
| **Tần suất** | Hàng ngày |

**User Stories:**
- US-P5-01: Tôi muốn AI assistant nhớ coding preferences qua MCP tools (`memory_store`).
- US-P5-02: Tôi muốn AI tìm kiếm trong codebase đã index (`ov_search`, `ov_grep`).
- US-P5-03: Tôi muốn nói "remember this" và AI lưu vào long-term memory (`memory_store` với type=procedural).
- US-P5-04: Tôi muốn AI đọc/ghi files trong context DB (`ov_read_file`, `ov_write_file`).

### 2.6 P6 — AI Framework Integrator (Secondary)

| Thuộc tính | Mô tả |
|---|---|
| **Vai trò** | Xây integration cho AI frameworks (AutoGen, CrewAI, LangChain) |
| **Mục tiêu** | Implement framework-native memory interface backed by VNP Memory |
| **Tần suất** | Theo dự án |

**User Stories:**
- US-P6-01: Tôi muốn integrate qua standard REST API với clear OpenAPI schema.
- US-P6-02: Tôi muốn SDK với async support và type hints đầy đủ.

### 2.7 P7 — AI Power User (Secondary)

| Thuộc tính | Mô tả |
|---|---|
| **Vai trò** | Người dùng cuối tương tác qua AI assistant hàng ngày |
| **Mục tiêu** | AI cá nhân hóa, nhớ sở thích và context giữa các phiên |
| **Pain Points** | AI quên mọi thứ sau mỗi session; không cá nhân hóa |
| **Tần suất** | Hàng ngày |

**User Stories:**
- US-P7-01: Tôi muốn AI nhớ sở thích và dự án (Memobase profile: preference/fact/goal/habit).
- US-P7-02: Tôi muốn nói "quên đi" và AI thực sự quên (`POST /v1/memory/forget`).
- US-P7-03: Tôi muốn xem memory profile của mình qua console (`GET /v1/console/profiles/{user_id}`).
- US-P7-04: Tôi muốn xem event timeline của tương tác với AI (`GET /v1/console/profiles/{user_id}/events`).

### 2.8 P8 — Product Manager (Secondary)

| Thuộc tính | Mô tả |
|---|---|
| **Vai trò** | Quản lý sản phẩm AI có tương tác người dùng |
| **Mục tiêu** | Phân tích hành vi người dùng qua structured profiles |
| **Tần suất** | Hàng tuần |

**User Stories:**
- US-P8-01: Tôi muốn xem structured user profiles (key/value/category/score) từ conversations.
- US-P8-02: Tôi muốn theo dõi usage analytics và cost (`GET /v1/console/observability/costs`).

---

## 3. User Requirements

### 3.1 Unified Memory Operations (UR-MEM)

#### UR-MEM-01: Store — Lưu memory thống nhất
**Persona:** P1, P5 | **Priority:** Critical | **Endpoint:** `POST /v1/memory/store`

**Mô tả:** Người dùng lưu memory qua 1 API thống nhất, hệ thống tự route tới engine phù hợp.

**Acceptance Criteria:**
- `type=semantic` → cognee-ingestion
- `type=episodic` → graphiti-ingestion
- `type=conversational` → zep-memory
- `type=profile` → memobase-ingestion (Blob → Buffer, auto-flush at 20 blobs)
- `type=procedural` → ov-fs
- `type=adaptive` → sm-memory
- `type=auto` → LLM content classification + routing
- Background processing (non-blocking)
- NATS event published: `memory.blob.inserted`

#### UR-MEM-02: Recall — Truy xuất cross-engine
**Persona:** P1, P5 | **Priority:** Critical | **Endpoint:** `POST /v1/memory/recall`

**Mô tả:** Truy xuất context liên quan từ tất cả memory engines qua 1 query.

**Acceptance Criteria:**
- Parallel gRPC fan-out qua `vnp-search-hub`
- Kết quả từ: cognee-search + graphiti-search + memobase-context + ov-search + zep-search + sm-search
- Merge + rerank kết quả
- Latency < 500ms (p95)

#### UR-MEM-03: Timeline — Temporal queries
**Persona:** P1, P3 | **Priority:** High | **Endpoint:** `GET /v1/memory/timeline`

**Mô tả:** Truy vấn timeline sự kiện từ `vnp-event`.

**Acceptance Criteria:**
- Query UserEvent per user, per tenant, per engine
- EventType filter: ingestion/search/memory/profile/admin
- GistText (LLM-generated summary) per event
- Timeline sorted by CreatedAt

#### UR-MEM-04: Forget — Xóa memory
**Persona:** P1, P4 | **Priority:** High | **Endpoint:** `POST /v1/memory/forget`

**Mô tả:** Xóa memory với cascading across tất cả engines.

**Acceptance Criteria:**
- Cascade: Cognee + Graphiti + Zep + Memobase + Supermemory + OpenViking
- GDPR-compliant endpoint: `POST /v1/console/governance/gdpr/forget`
- Dry-run preview: `POST /v1/console/governance/gdpr/forget/preview`
- Audit log cho deletion events

### 3.2 Profile Memory (UR-PROFILE)

#### UR-PROFILE-01: Structured user profile extraction
**Persona:** P1, P7, P8 | **Priority:** High
**Endpoints:** `/v1/memobase/users/{uid}/blobs`, `/v1/memobase/users/{uid}/profiles`

**Mô tả:** Tự động xây dựng và duy trì user profile có cấu trúc từ conversations qua Memobase YOLO Engine.

**Acceptance Criteria:**
- Profile format: `key/value/category(preference|fact|goal|habit)/score`
- Blob types: conversation/fact/document/image
- Buffer auto-flush khi ≥ 20 blobs (configurable `FlushThreshold`)
- Fixed 3 LLM calls per flush: extract → merge → events
- Context API: prompt-ready string < 100ms via `GET /v1/memobase/users/{uid}/context`
- Profile score per attribute (float64)

#### UR-PROFILE-02: Adaptive memory with auto-forgetting
**Persona:** P1, P7 | **Priority:** High
**Endpoints:** `/v1/sm/memories`, `/v1/console/adaptive/*`

**Mô tả:** Living knowledge graph tự động quên thông tin hết hạn, giải quyết mâu thuẫn.

**Acceptance Criteria:**
- Memory versioning: parent → root chain, `isLatest` flag
- Relation types: updates/extends/derives
- `forgetAfter` duration configurable per memory type
- Contradiction resolution (auto-update `isLatest`)
- Static vs Dynamic memory classification
- Version Explorer via `GET /v1/console/adaptive/memories/{id}/versions`

#### UR-PROFILE-03: External data connectors
**Persona:** P1 | **Priority:** Medium
**Endpoints:** `POST /v1/sm/connections`, `POST /v1/sm/connections/{id}/sync`

**Mô tả:** Tự động sync dữ liệu từ external sources vào Supermemory.

**Acceptance Criteria:**
- Google Drive, Gmail, Notion, OneDrive, GitHub connectors
- Manual sync trigger via API
- Connection management via `GET /v1/console/adaptive/connectors`

### 3.3 Data Ingestion (UR-ING)

#### UR-ING-01: Multi-modal ingestion
**Persona:** P1, P3 | **Priority:** Critical

**Acceptance Criteria:**
- Cognee: PDF, text, DOCX, CSV, URL, audio, image (`POST /v1/cognee/datasets/{id}/data`)
- Zep: Role-typed messages (user/assistant) (`PUT /v1/zep/sessions/{id}/memory`)
- Memobase: ChatBlob, DocBlob, SummaryBlob (`POST /v1/memobase/users/{uid}/blobs`)
- OpenViking: Files, Git repos, HTTP resources (`POST /v1/ov/resources/ingest`)
- Supermemory: Documents (markdown/pdf/html), memories (`POST /v1/sm/documents`, `POST /v1/sm/memories`)
- Graphiti: Episodes (text/JSON/fact_triple) (`POST /v1/graphiti/episodes`)

#### UR-ING-02: Knowledge graph construction
**Persona:** P1, P3 | **Priority:** Critical

**Acceptance Criteria:**
- Cognee Cognify pipeline: `POST /v1/cognee/datasets/{id}/cognify`
- Entity deduplication tự động
- Relationship/edge extraction với temporal validity (Graphiti)
- Graph facts for Zep: `POST /v1/zep/graph/facts`
- Supermemory memory versioning during construction

#### UR-ING-03: Tiered context generation
**Persona:** P1 | **Priority:** High

**Acceptance Criteria:**
- L0 Abstract (~100 tokens): one-sentence summary
- L1 Overview (~2K tokens): core info + usage scenarios
- L2 Detail (full content): deep reading khi cần
- Load on demand via ov-search hierarchical retrieval

### 3.4 Search & Retrieval (UR-SEARCH)

#### UR-SEARCH-01: Multi-strategy search
**Persona:** P1, P3 | **Priority:** Critical

**Acceptance Criteria:**
- **Semantic search**: Vector similarity via cognee-search, sm-search
- **Graph completion**: Cognee (15+ strategies) via `POST /v1/cognee/search`
- **Temporal search**: Graphiti via `POST /v1/graphiti/search`
- **Session search**: Zep via `POST /v1/zep/sessions/{id}/search`
- **Graph search**: Zep KG via `POST /v1/zep/graph/search`
- **Hierarchical**: OpenViking directory recursive via `POST /v1/ov/search`
- **RAG**: Supermemory via `POST /v1/sm/rag`
- **Hybrid**: Cross-engine via `POST /v1/memory/recall`

#### UR-SEARCH-02: Context assembly
**Persona:** P1 | **Priority:** Critical

**Acceptance Criteria:**
- Memobase context: prompt-ready string < 100ms via `GET /v1/memobase/users/{uid}/context`
- Includes: Summary + Profiles + Events + TokenCount
- Profile context preview: `GET /v1/console/profiles/{user_id}/context`
- Configurable token budget

#### UR-SEARCH-03: Reranking & filtering
**Persona:** P1, P3 | **Priority:** High

**Acceptance Criteria:**
- `vnp-search-hub` cross-engine merge + rerank
- Memory type filtering: semantic/episodic/conversational/profile/procedural/adaptive
- Time range filters
- Engine-specific filters

### 3.5 Session Management (UR-SESSION)

#### UR-SESSION-01: Session lifecycle
**Persona:** P1 | **Priority:** Critical

**Acceptance Criteria:**
- Create session: `POST /v1/ov/sessions` (OpenViking)
- Add messages: `POST /v1/ov/sessions/{id}/messages`
- 2-phase commit: `POST /v1/ov/sessions/{id}/commit`
- Zep user sessions: `PUT/GET /v1/zep/sessions/{id}/memory`
- Live session monitoring: `GET /v1/console/sessions/live`

#### UR-SESSION-02: Working Memory
**Persona:** P1 | **Priority:** High

**Acceptance Criteria:**
- Working Memory via OpenViking ov-session
- Structured document: title, state, goals, facts, errors
- Session diff: `GET /v1/console/sessions/{id}/diff`
- Working memory inspector: `GET /v1/console/sessions/{id}/working-memory`
- User summary: `GET /v1/console/sessions/{id}/user-summary`

#### UR-SESSION-03: Session Timeline
**Persona:** P1, P3 | **Priority:** Medium

**Acceptance Criteria:**
- Full session timeline: `GET /v1/console/sessions/{id}/timeline`
- Session replay in Console
- Memory before/after diff view

### 3.6 Authentication & Multi-tenancy (UR-AUTH)

#### UR-AUTH-01: Authentication
**Persona:** P2, P4 | **Priority:** Critical

**Acceptance Criteria:**
- API Key authentication: SHA-256 hashed, `KeyPrefix` (8 chars) for identification
- JWT RS256 bearer token
- Dev mode: `AUTH_DEV_MODE=true` — skip auth (dev only)
- API key lifecycle: create, revoke, expire (`ExpiresAt`, `RevokedAt`)
- Key management: `POST /v1/admin/tenants/{id}/keys`

#### UR-AUTH-02: Multi-tenant isolation
**Persona:** P2, P4 | **Priority:** Critical

**Acceptance Criteria:**
- `TenantID` required on all domain entities
- Subscription tiers: free/pro/enterprise
- Tenant status: active/suspended/deleted
- User roles per tenant: admin/editor/viewer
- Engine aliases per tenant (custom engine keys)
- Tenant management: `GET/POST/PUT /v1/console/governance/tenants`

#### UR-AUTH-03: Rate Limiting
**Persona:** P2 | **Priority:** High

**Acceptance Criteria:**
- Per-tenant rate limiting (Redis-backed)
- Rate tiers: free/pro/enterprise
- NATS event on rate limit exceeded: `gateway.ratelimit.exceeded`
- Rate limit metrics via console

### 3.7 Agent & Framework Integration (UR-AGENT)

#### UR-AGENT-01: MCP Server
**Persona:** P5 | **Priority:** High

**Acceptance Criteria:**
- Transport: SSE (`GET /mcp/sse`), HTTP Streamable (`POST /mcp/message`)
- JSON-RPC 2.0 protocol, version `2024-11-05`
- 16 tools: memory_store, memory_recall, memory_search, memory_timeline, memory_profile, memory_forget, graph_query, ov_read_file, ov_write_file, ov_search, ov_list_dir, ov_grep, ov_tree, ov_session_commit, ov_ingest, ov_delete
- Port `:8082` (configurable via `VNP_MEMORY_SERVER_MCP_PORT`)

#### UR-AGENT-02: REST API Integration
**Persona:** P1, P6 | **Priority:** High

**Acceptance Criteria:**
- JSON REST API on `:8080`
- CORS support cho frontend clients
- RequestID header cho tracing
- Structured error responses

#### UR-AGENT-03: WebSocket Streaming
**Persona:** P1, P5 | **Priority:** Medium

**Acceptance Criteria:**
- WebSocket endpoint: `GET /v1/console/ws`
- Real-time event streaming cho console
- Reconnection support

### 3.8 Deployment & Operations (UR-OPS)

#### UR-OPS-01: Quick start
**Persona:** P1, P2 | **Priority:** Critical

**Acceptance Criteria:**
- `make infra-up` → start 5 infrastructure containers
- `make dev` → start monolith (all 35 services)
- `curl http://localhost:8083/healthz` → verify all services
- Working in < 5 minutes from clone

#### UR-OPS-02: Production deployment
**Persona:** P2 | **Priority:** High

**Acceptance Criteria:**
- `VNP_MEMORY_NATS_MODE=external` cho external NATS cluster
- Docker multi-stage build via `make docker`
- Full stack via `make docker-up`
- Graceful shutdown: HTTP drain → NATS drain → gRPC stop → DB close

#### UR-OPS-03: Observability
**Persona:** P2 | **Priority:** High

**Acceptance Criteria:**
- OpenTelemetry tracing qua tất cả engines
- Prometheus metrics trên port `:8083`
- Structured JSON logging (slog)
- Console observability: `GET /v1/console/observability/{metrics|traces|errors|costs}`
- Secret redaction in logs/traces

### 3.9 Governance (UR-GOV)

#### UR-GOV-01: GDPR Compliance
**Persona:** P4 | **Priority:** High

**Acceptance Criteria:**
- GDPR forget: `POST /v1/console/governance/gdpr/forget`
- Preview: `POST /v1/console/governance/gdpr/forget/preview` (dry-run)
- Cascading across all 6 engines
- Audit trail cho deletion events

#### UR-GOV-02: Policy Management
**Persona:** P4 | **Priority:** High

**Acceptance Criteria:**
- Policy CRUD: `GET/POST/PUT /v1/console/governance/policies`
- Audit log search: `GET /v1/console/governance/audit` (actor, action, entity, tenant, engine)

#### UR-GOV-03: Pipeline Monitoring
**Persona:** P2, P3 | **Priority:** Medium

**Acceptance Criteria:**
- Pipeline status per engine: `GET /v1/console/pipelines/{engine}`
- Job list/detail: `GET /v1/console/pipelines/{engine}/jobs`
- Queue depth monitoring: `GET /v1/console/pipelines/queues`
- Worker status: `GET /v1/console/pipelines/workers`
- Reusable templates: `GET /v1/console/pipelines/templates`

---

## 4. Interaction Models

### 4.1 REST API Direct

```
Application → POST /v1/memory/store     → Auto-route to engine
           → POST /v1/memory/recall    → Cross-engine (vnp-search-hub)
           → GET  /v1/memory/timeline  → Event timeline (vnp-event)
           → POST /v1/memory/forget    → Cascading delete

Engine-specific:
           → POST /v1/cognee/datasets/{id}/cognify  → Semantic pipeline
           → POST /v1/graphiti/episodes              → Temporal episodes
           → PUT  /v1/zep/sessions/{id}/memory      → Conversational
           → POST /v1/memobase/users/{uid}/blobs    → Profile blobs
           → POST /v1/sm/memories                   → Adaptive memory
           → GET  /v1/ov/files/{path}               → Procedural context
```

### 4.2 MCP Protocol (AI Assistants)

```
AI Assistant → MCP SSE/HTTP (:8082) → vnp-gateway
(Claude Code)    JSON-RPC 2.0          ├── memory_store   → cognee-ingestion
                                        ├── memory_recall  → vnp-search-hub
                                        ├── memory_search  → cognee-search
                                        ├── memory_timeline → vnp-event
                                        ├── memory_profile → memobase-context
                                        ├── memory_forget  → vnp-event
                                        ├── graph_query    → graphiti-store
                                        ├── ov_read_file   → ov-fs
                                        ├── ov_write_file  → ov-fs
                                        ├── ov_search      → ov-search
                                        ├── ov_list_dir    → ov-fs
                                        ├── ov_grep        → ov-fs
                                        ├── ov_tree        → ov-fs
                                        ├── ov_session_commit → ov-session
                                        ├── ov_ingest      → ov-resource
                                        └── ov_delete      → ov-fs
```

### 4.3 Memobase Buffer Flow

```
User Interaction → Blob Insert → Buffer (in-memory)
                                    ↓ (≥ 20 blobs OR manual flush)
                               YOLO Engine (fixed 3 LLM calls)
                                    ├── extract → Blob content → Profile candidates
                                    ├── merge   → Merge with existing profiles
                                    └── events  → Generate event gist texts
                                    ↓
                               ProfileRepository → Score updated profiles
                               EventRepository → Log events
                               → Context available < 100ms
```

### 4.4 Supermemory Adaptive Memory Flow

```
Content Insert → SMMemory (adaptive KG)
                    → Memory versioning: parent → root chain
                    → isLatest flag management
                    → Contradiction detection → auto-update
                    → forgetAfter scheduling
                    → Version available in console:
                      GET /v1/console/adaptive/memories/{id}/versions
```

### 4.5 Engine-Specific Direct Access

| Engine | REST Prefix | Services |
|---|---|---|
| Cognee | `/v1/cognee/*` | cognee-ingestion, cognee-cognify, cognee-search |
| Graphiti | `/v1/graphiti/*` | graphiti-ingestion, graphiti-search, graphiti-store |
| Memobase | `/v1/memobase/*` | memobase-ingestion, memobase-context |
| OpenViking | `/v1/ov/*` | ov-fs, ov-search, ov-session, ov-resource |
| Zep | `/v1/zep/*` | zep-user, zep-memory, zep-graph, zep-search |
| Supermemory | `/v1/sm/*` | sm-document, sm-memory, sm-search, sm-profile, sm-connector |

---

## 5. Standard Operating Procedures (SOPs)

### SOP-01: Quick Start — First-time Setup

| Step | Action | Chi tiết |
|---|---|---|
| 1 | Clone repository | `git clone <repo>` |
| 2 | Start infrastructure | `make infra-up` — PostgreSQL, Neo4j, Qdrant, Redis, MinIO |
| 3 | Configure environment | Copy `configs/config.yaml`, set API keys |
| 4 | Start monolith | `make dev` |
| 5 | Verify health | `curl http://localhost:8083/healthz \| jq` |
| 6 | Test memory store | `curl -X POST http://localhost:8080/v1/memory/store -d '{"content":"test","type":"fact"}'` |

### SOP-02: Agent Memory Integration via MCP

| Step | Action | Chi tiết |
|---|---|---|
| 1 | Start VNP Memory | `make dev` |
| 2 | Configure MCP client | Point to `http://localhost:8082` |
| 3 | Initialize | `{"jsonrpc":"2.0","id":1,"method":"initialize"}` |
| 4 | List tools | `{"jsonrpc":"2.0","id":2,"method":"tools/list"}` |
| 5 | Store memory | `{"method":"tools/call","params":{"name":"memory_store","arguments":{"content":"...","type":"fact"}}}` |
| 6 | Recall memory | `{"method":"tools/call","params":{"name":"memory_recall","arguments":{"query":"..."}}}` |

### SOP-03: Memobase Profile Setup

| Step | Action | Chi tiết |
|---|---|---|
| 1 | Insert blobs | `POST /v1/memobase/users/{uid}/blobs` với type=conversation |
| 2 | Accumulate buffer | Tiếp tục insert (auto-flush tại 20 blobs) |
| 3 | Manual flush | `POST /v1/memobase/users/{uid}/flush` |
| 4 | Get profiles | `GET /v1/memobase/users/{uid}/profiles` |
| 5 | Get context | `GET /v1/memobase/users/{uid}/context` — prompt-ready |
| 6 | Monitor | `GET /v1/console/profiles/{user_id}/buffers` |

### SOP-04: Tenant Management

| Step | Action | Chi tiết |
|---|---|---|
| 1 | Create tenant | `POST /v1/admin/tenants` (name, slug, tier) |
| 2 | Issue API key | `POST /v1/admin/tenants/{id}/keys` |
| 3 | Configure engine aliases | `engine_aliases` map in Tenant metadata |
| 4 | Monitor usage | `GET /v1/console/governance/tenants` |
| 5 | Set policies | `POST /v1/console/governance/policies` |

### SOP-05: Production Deployment

| Step | Action | Chi tiết |
|---|---|---|
| 1 | Configure env | Set production DSN, API keys, NATS URL |
| 2 | External NATS | `VNP_MEMORY_NATS_MODE=external`, `VNP_MEMORY_NATS_URL=nats://...` |
| 3 | Deploy infra | PostgreSQL cluster, Neo4j, Redis, NATS |
| 4 | Build & deploy | `make docker` → Kubernetes deployment |
| 5 | Disable dev mode | `VNP_MEMORY_AUTH_DEV_MODE=false` |
| 6 | Setup monitoring | OpenTelemetry → Prometheus → Grafana |
| 7 | Verify isolation | Test cross-tenant access = blocked |

### SOP-06: GDPR Forget Procedure

| Step | Action | Chi tiết |
|---|---|---|
| 1 | Preview impact | `POST /v1/console/governance/gdpr/forget/preview` |
| 2 | Review results | Check cascade: all 6 engines impacted |
| 3 | Execute forget | `POST /v1/console/governance/gdpr/forget` |
| 4 | Verify audit | `GET /v1/console/governance/audit` — deletion logged |
| 5 | Confirm erasure | Re-query user data = empty |

---

## 6. Non-Functional User Requirements

### 6.1 Performance (UR-NFR-PERF)

| ID | Requirement | Target |
|---|---|---|
| NFR-PERF-01 | Memobase context retrieval | < 100ms (p95) |
| NFR-PERF-02 | Conversational context assembly (Zep) | < 200ms (p95) |
| NFR-PERF-03 | Cross-engine recall | < 500ms (p95) |
| NFR-PERF-04 | Graph search latency | < 1000ms (p95) |
| NFR-PERF-05 | Knowledge graph construction | Background async, non-blocking |
| NFR-PERF-06 | Concurrent sessions | ≥ 1,000 per instance |
| NFR-PERF-07 | Token cost reduction vs naive RAG | ≥ 80% |
| NFR-PERF-08 | LLM calls per Memobase flush | Fixed 3 calls (YOLO engine) |

### 6.2 Reliability (UR-NFR-REL)

| ID | Requirement | Target |
|---|---|---|
| NFR-REL-01 | API uptime | ≥ 99.9% |
| NFR-REL-02 | Circuit breaker | Auto-open on downstream failure |
| NFR-REL-03 | Graceful shutdown | HTTP → NATS → gRPC → DB (30s timeout) |
| NFR-REL-04 | Data durability | PostgreSQL ACID + WAL |
| NFR-REL-05 | Fallback LLM | Bifrost multi-provider routing |

### 6.3 Security (UR-NFR-SEC)

| ID | Requirement | Target |
|---|---|---|
| NFR-SEC-01 | Tenant isolation | TenantID on every entity, zero cross-tenant leaks |
| NFR-SEC-02 | API key storage | SHA-256 hash, prefix only in response |
| NFR-SEC-03 | Secret redaction | Auto-redact in logs/traces |
| NFR-SEC-04 | Dev mode guard | localhost-only when AUTH_DEV_MODE=true |
| NFR-SEC-05 | Network security | TLS in transit; ov-crypto for at-rest |

### 6.4 Usability (UR-NFR-USE)

| ID | Requirement | Target |
|---|---|---|
| NFR-USE-01 | Quick start | Working demo in < 5 minutes (`make infra-up && make dev`) |
| NFR-USE-02 | Health check | All 35 services visible in `/healthz` |
| NFR-USE-03 | Error messages | Structured JSON errors với code + message |
| NFR-USE-04 | Configuration | Single YAML file + ENV overrides |

---

## 7. Acceptance Criteria Summary

| Category | Criteria | Metric |
|---|---|---|
| **Unified API** | Store/Recall works across 6 engines | All engines receive routed data |
| **Latency** | Memobase context < 100ms | Prometheus p95 |
| **Latency** | Cross-engine recall < 500ms | Prometheus p95 |
| **MCP Tools** | All 16 tools respond correctly | Integration tests |
| **Memobase** | Profiles extracted from conversations | key/value/category/score format |
| **Supermemory** | Version chain managed | parent→root chain, isLatest correct |
| **Sessions** | 2-phase commit works (OpenViking) | archive → extract verified |
| **GDPR** | Cascading forget across 6 engines | Data erasure verified |
| **Governance** | Audit trail complete | All operations logged |
| **Deployment** | Full stack: `make infra-up && make dev` | `/healthz` shows 35 services |
| **Observability** | Traces visible | End-to-end tracing works |
| **Isolation** | No cross-tenant data | 0 leak incidents |

---

## 8. Traceability Matrix

| User Requirement | PRD Section | Implementation |
|---|---|---|
| UR-MEM-01 (Store) | §5.1 | `gateway/adapter/handler/handler.go` (MemoryHandler.Store) |
| UR-MEM-02 (Recall) | §5.1 | `gateway/adapter/handler/handler.go` (MemoryHandler.Recall) → vnp-search-hub |
| UR-MEM-03 (Timeline) | §5.1 | `gateway/adapter/handler/handler.go` (MemoryHandler.Timeline) → vnp-event |
| UR-MEM-04 (Forget) | §10.2 | `gateway/adapter/handler/handler.go` (MemoryHandler.Forget) |
| UR-PROFILE-01 (Memobase) | §5.5 | `services/memory-service/domain/memobase/`, usecase/memobase/service.go |
| UR-PROFILE-02 (Adaptive) | §5.7 | `services/memory-service/domain/sm/`, `gateway/handler` sm.* |
| UR-PROFILE-03 (Connectors) | §5.7 | `POST /v1/sm/connections`, `POST /v1/sm/connections/{id}/sync` |
| UR-ING-01 (Multi-modal) | §5.2-5.7 | All engine ingestion endpoints |
| UR-ING-02 (KG) | §5.2, §5.3 | Cognee cognify, Graphiti episodes, Zep graph facts |
| UR-ING-03 (Tiered) | §5.6 | OpenViking L0/L1/L2 via ov-search |
| UR-SEARCH-01 (Multi-strategy) | §5.2-5.7 | Per-engine search endpoints |
| UR-SEARCH-02 (Assembly) | §5.5 | `GET /v1/memobase/users/{uid}/context` |
| UR-SEARCH-03 (Reranking) | §5.1 | vnp-search-hub merge + rerank |
| UR-SESSION-01 (Lifecycle) | §5.6 | `POST /v1/ov/sessions`, `POST /v1/zep/sessions` |
| UR-SESSION-02 (Working) | §5.6 | `GET /v1/console/sessions/{id}/working-memory` |
| UR-AUTH-01 (Authentication) | §6.2 | `gateway/domain/entity.go` (AuthContext, APIKey) |
| UR-AUTH-02 (Multi-tenant) | §6.2 | `services/vnp-platform/domain/admin/` (Tenant, SubscriptionTier) |
| UR-AUTH-03 (Rate Limiting) | §6.2 | `gateway/domain/entity.go` (RateTier), NATS RateLimitExceeded |
| UR-AGENT-01 (MCP) | §6.3 | `gateway/adapter/mcp/server.go` (16 tools, SSE + HTTP) |
| UR-AGENT-03 (WebSocket) | §7.2 | `gateway/adapter/handler/ws.go` |
| UR-OPS-01 (Quick start) | §9.3 | `apps/memory/Makefile` (infra-up, dev) |
| UR-OPS-03 (Observability) | §11 | `services/obs-service/`, `/v1/console/observability/*` |
| UR-GOV-01 (GDPR) | §10.2 | `POST /v1/console/governance/gdpr/forget{,/preview}` |
| UR-GOV-02 (Policies) | §10.2 | `GET/POST/PUT /v1/console/governance/policies` |
| UR-GOV-03 (Pipeline) | §7.2 | `/v1/console/pipelines/*` → `services/pipeline-service/` |

---

## 9. Appendix — Service to Domain Mapping

| Service Code | Go Package | Domain Entities |
|---|---|---|
| memobase-ingestion | `memory-service/usecase/memobase.IngestUseCase` | Blob, Buffer |
| memobase-context | `memory-service/usecase/memobase.ContextUseCase` | UserContext, Profile, Event |
| zep-user | `memory-service/usecase/zep.UserService` | ZepUser |
| zep-memory | `memory-service/usecase/zep.MemoryService` | ZepSession, ZepMemory, ZepMessage |
| sm-memory | `memory-service/usecase/sm.MemoryService` | SMMemory |
| vnp-admin | `vnp-platform/domain/admin` | Tenant, User, APIKey, HealthStatus |
| vnp-event | `vnp-platform/domain/event` | UserEvent, Timeline, EventType |
| pipeline-* | `pipeline-service/domain/pipeline` | Pipeline, Job, Queue, Worker, Template |
| ov-fs | `storage-service/domain/fs` | FileEntity (VikingFS) |
| gateway domain | `gateway/domain` | AuthContext, StoreRequest, RouteTarget |
