# Product Requirements Document (PRD)

## VNP Memory — Unified Cognitive Infrastructure Layer for Enterprise AI

| Field | Value |
|---|---|
| **Product** | VNP Memory |
| **Version** | 2.0.0 |
| **Status** | Active Development |
| **Last Updated** | 2026-06-16 |
| **Category** | Enterprise AI Infrastructure |

---

## 1. Executive Summary

VNP Memory là một **Unified Cognitive Infrastructure Layer** cho Enterprise AI — nền tảng tích hợp toàn diện giải quyết bài toán "AI Memory" bằng cách hợp nhất 6 memory engines chuyên biệt (Cognee, Graphiti, Memobase, OpenViking, Zep, Supermemory) dưới một kiến trúc thống nhất.

### Tầm nhìn sản phẩm

> **"Enterprise Cognitive Infrastructure — Operating System cho AI Cognition"**

Thay vì xây dựng thêm "một vector DB nữa", VNP Memory tạo ra một **Persistent Context Platform** cho phép AI Agent:
- Nhớ đúng thứ, đúng lúc (Context Quality > Memory Quantity)
- Duy trì bộ nhớ dài hạn có cấu trúc, có quan hệ, có thời gian
- Tự động quên thông tin hết hạn và giải quyết mâu thuẫn
- Tuân thủ governance, audit trail, và multi-tenant isolation cấp enterprise

### Giá trị cốt lõi

| Pillar | Mô tả | Engine chính |
|---|---|---|
| **Episodic Memory** | Theo dõi sự kiện theo thời gian, temporal reasoning | Graphiti |
| **Semantic Memory** | Trích xuất tri thức, xây dựng knowledge graph | Cognee |
| **Conversational Memory** | Bộ nhớ hội thoại, context assembly < 200ms | Zep |
| **Profile Memory** | Structured user profiles từ conversations, event timeline | Memobase |
| **Adaptive Memory** | Living KG với auto-forgetting, external connectors | Supermemory |
| **Procedural Memory** | Context phân tầng L0/L1/L2, VikingFS | OpenViking |

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
| 5 | **Thiếu user profiling** | Không có structured profiles từ conversations |
| 6 | **Governance / Audit gap** | Không kiểm soát được AI nhớ gì, từ đâu, ai tạo |
| 7 | **Memory không tự evolve** | Knowledge cũ không bị replace khi có thông tin mới |

### 2.2 Hạn chế giải pháp hiện tại

| Hệ thống | Mạnh về | Thiếu |
|---|---|---|
| Zep | Conversational memory | User profiling, adaptive memory |
| Mem0 | Lightweight memory | Temporal reasoning, enterprise features |
| Graphiti | Temporal graph memory | Context assembly, user profiles |
| Cognee | Extraction pipeline | Session management, filesystem |
| Memobase | User profiling (YOLO engine) | Graph memory, temporal reasoning |
| Supermemory | Adaptive KG + connectors | Session management |

> **Không ai unify toàn bộ stack.** VNP Memory giải quyết bằng cách tích hợp 6 engines chuyên biệt dưới một API thống nhất.

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
| **Product Manager** | User profile analytics, usage metrics |
| **DevOps Team** | Container orchestration, health monitoring |
| **IDE Plugin User** | AI coding assistant với persistent memory |

---

## 4. Product Architecture

### 4.1 Deployment Models

VNP Memory hỗ trợ hai chế độ triển khai:

**Monolith Mode** (Development & Single-server)
```
apps/memory — Single Go binary
    Gateway (REST :8080, MCP :8082, Health :8083)
        └── InProcessRegistry (bufconn)
            └── 35 Engine Services (in-memory gRPC)
                └── Embedded NATS JetStream
```

**Distributed Mode** (Production)
```
gateway/ — API Gateway
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

### 4.3 Platform Services

| Service | Vai trò |
|---|---|
| **vnp-admin** | Tenant lifecycle, API key management, quotas |
| **vnp-event** | Cross-engine event timeline, UserEvent log |
| **vnp-search-hub** | Cross-engine recall (parallel gRPC fan-out + merge) |
| **vnp-platform** | Admin APIs, auth, analytics |

---

## 5. Core Features

### 5.1 Unified Memory API (Implemented)

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

### 5.2 Episodic Memory (Graphiti)

| Feature | Mô tả | Priority |
|---|---|---|
| **Temporal Fact Management** | Mỗi fact có validity window (valid_at/invalid_at) | P0 |
| **Episode Ingestion** | `POST /v1/graphiti/episodes` | P0 |
| **Entity/Edge Extraction** | LLM-powered extraction với deduplication | P0 |
| **Hybrid Search** | Semantic + BM25 + Graph traversal + reranking | P0 |
| **Graph Nodes/Edges** | `GET /v1/graphiti/nodes/{id}`, `GET /v1/graphiti/edges/{id}` | P0 |
| **Knowledge Graph** | graphiti-knowledge service | P1 |

### 5.3 Semantic Memory (Cognee)

| Feature | Mô tả | Priority |
|---|---|---|
| **Multi-modal Ingestion** | PDF, text, audio, image, CSV, URL | P0 |
| **Dataset Management** | `POST /v1/cognee/datasets` | P0 |
| **Cognify Pipeline** | `POST /v1/cognee/datasets/{id}/cognify` | P0 |
| **Multi-strategy Search** | `POST /v1/cognee/search` (15+ strategies) | P0 |
| **Custom Pipelines** | Extensible task pipeline architecture | P1 |

### 5.4 Conversational Memory (Zep)

| Feature | Mô tả | Priority |
|---|---|---|
| **User Management** | `POST/GET/PATCH /v1/zep/users` | P0 |
| **Session Memory** | `PUT/GET /v1/zep/sessions/{id}/memory` | P0 |
| **Graph Search** | `POST /v1/zep/graph/search` | P0 |
| **Facts Management** | `POST /v1/zep/graph/facts` | P0 |
| **Ontology** | `POST /v1/zep/graph/ontology` | P1 |
| **Session Search** | `POST /v1/zep/sessions/{id}/search` | P1 |

### 5.5 Profile Memory (Memobase)

| Feature | Mô tả | Priority |
|---|---|---|
| **Blob Ingestion** | `POST /v1/memobase/users/{uid}/blobs` | P0 |
| **Buffer Auto-flush** | Flush khi ≥ 20 blobs (configurable) | P0 |
| **Manual Flush** | `POST /v1/memobase/users/{uid}/flush` | P0 |
| **Context Assembly** | `GET /v1/memobase/users/{uid}/context` | P0 |
| **Structured Profiles** | `GET /v1/memobase/users/{uid}/profiles` — key/value/category/score | P0 |
| **Event Timeline** | `GET /v1/memobase/users/{uid}/events` | P1 |

**Memobase YOLO Engine**: Fixed 3 LLM calls per flush (extract → merge → events), profile categories: preference/fact/goal/habit.

### 5.6 Procedural Memory (OpenViking)

| Feature | Mô tả | Priority |
|---|---|---|
| **VikingFS** | `GET/PUT/DELETE /v1/ov/files/{path}` | P0 |
| **Directory Tree** | `GET /v1/ov/tree/{path}` | P0 |
| **Grep Search** | `POST /v1/ov/grep` | P0 |
| **Semantic Search** | `POST /v1/ov/search` | P0 |
| **Session Management** | `POST /v1/ov/sessions`, add messages, commit | P0 |
| **Resource Ingest** | `POST /v1/ov/resources/ingest` (Git, HTTP, local) | P1 |

**Tiered Context**: L0 (~100 tok) → L1 (~2K tok) → L2 (full detail), load on demand.

### 5.7 Adaptive Memory (Supermemory)

| Feature | Mô tả | Priority |
|---|---|---|
| **Document Management** | `POST/GET /v1/sm/documents` | P0 |
| **Memory Store** | `POST /v1/sm/memories` (adaptive KG) | P0 |
| **Hybrid Search** | `POST /v1/sm/search` | P0 |
| **RAG** | `POST /v1/sm/rag` | P0 |
| **User Profile** | `GET /v1/sm/profiles/{uid}` | P0 |
| **External Connectors** | `POST /v1/sm/connections` + sync | P1 |
| **Project Spaces** | `POST /v1/sm/projects/spaces` | P1 |

**Auto-forgetting**: Memory versioning (parent → root chain), `isLatest` flag, relation types: updates/extends/derives, `forgetAfter` duration.

---

## 6. Memory API Gateway

### 6.1 Gateway Ports

| Port | Protocol | Description |
|---|---|---|
| `:8080` | REST HTTP | Primary API (50+ routes) |
| `:8081` | gRPC | Internal gRPC (optional) |
| `:8082` | MCP SSE + HTTP | Model Context Protocol (16 tools) |
| `:8083` | HTTP | Health check + Prometheus metrics |

### 6.2 Authentication & Multi-tenancy

| Mechanism | Mô tả |
|---|---|
| **API Key** | Per-tenant keys, SHA-256 hash, `KeyPrefix` cho identification |
| **JWT RS256** | Bearer token, RSA-256 signed |
| **Dev Mode** | `AUTH_DEV_MODE=true` — skip auth, localhost only |
| **Rate Tiers** | free / pro / enterprise, per-tenant quotas |
| **Namespace** | TenantID injected vào mọi query, isolation guaranteed |

### 6.3 Circuit Breaker

Gateway có circuit breaker tích hợp trên mọi downstream gRPC calls. Events published qua NATS `gateway.circuit.opened` khi downstream failure.

### 6.4 MCP Server (Model Context Protocol)

```
AI Assistant → MCP Protocol (SSE/HTTP) → vnp-gateway :8082
                                            ├── GET  /sse         — SSE transport
                                            ├── GET  /mcp/sse
                                            ├── POST /message     — JSON-RPC 2.0
                                            └── POST /mcp/message
```

Protocol version: `2024-11-05`. Supports: initialize, tools/list, tools/call, ping.

---

## 7. Console (Admin UI)

### 7.1 Embedded UI

UI console được embed vào binary. Khi `spaFS` được inject, gateway serve SPA tại `/`. Khi không có SPA, gateway chạy standalone API mode.

### 7.2 Console Features

| Feature | Routes | Description |
|---|---|---|
| **Dashboard** | `/v1/console/dashboard/*` | Health, metrics, throughput, memory heatmap |
| **Memory Explorer** | `/v1/console/memory/*` | Search, inspect, neighbors, versions |
| **Graph Studio** | `/v1/console/graph/*` | Subgraph, entity, timeline, ontology, query |
| **User Profiles** | `/v1/console/profiles/*` | Profile explorer, config, events, context, buffers |
| **Adaptive Memory** | `/v1/console/adaptive/*` | Memory versions, connectors, analytics, forget-rules |
| **Agent Debugger** | `/v1/console/debugger/*` | Context assembly traces |
| **Sessions** | `/v1/console/sessions/*` | Live sessions, timeline, diff, working-memory |
| **Governance** | `/v1/console/governance/*` | Tenants, policies, audit, GDPR forget |
| **Pipelines** | `/v1/console/pipelines/*` | Job status, queues, workers, templates |
| **Infrastructure** | `/v1/console/infra/*` | Service topology, databases, resources |
| **Observability** | `/v1/console/observability/*` | Metrics, traces, errors, costs |
| **WebSocket** | `/v1/console/ws` | Real-time event streaming |

---

## 8. Technical Architecture

### 8.1 Technology Stack

| Layer | Technology |
|---|---|
| **Language** | Go 1.23+ |
| **HTTP Framework** | stdlib net/http (Go 1.22+ pattern matching) |
| **In-process gRPC** | bufconn (google.golang.org/grpc) |
| **Message Broker** | NATS JetStream (embedded or external) |
| **Graph Database** | Neo4j 5+ |
| **Relational Database** | PostgreSQL 17 + pgvector |
| **Vector Database** | Qdrant (external, optional) |
| **Cache** | Redis 7+ |
| **Object Storage** | MinIO (S3 compatible) |
| **Custom Storage** | VikingFS (Go-native) |
| **AI Gateway** | Bifrost (multi-provider LLM routing) |
| **Observability** | OpenTelemetry + Prometheus |
| **Logging** | slog (structured JSON, stdlib) |

### 8.2 Data Flow — Event-Driven

```
Agent Request → Memory API Gateway → Engine Selection
    │
    ├─→ Memobase: Blob → Buffer (auto-flush at 20) → YOLO Engine (3 LLM calls)
    │                → Profile extracted → Event logged
    ├─→ Cognee: Add → Cognify (7 stages) → Knowledge Graph
    ├─→ Graphiti: Episode → Extract → Deduplicate → Temporal Graph
    ├─→ Zep: Message → Graph ingestion → Context assembly
    ├─→ Supermemory: Content → Memory (adaptive KG) → Version chain
    └─→ OpenViking: Data → VikingFS → L0/L1/L2 tiered context
         │
         ▼
    NATS JetStream → Event fan-out → Subscribed services
```

### 8.3 Middleware Stack

```
Recovery → RequestID → Logger → CORS → Auth → RateLimit → Metrics → Timeout
```

---

## 9. Deployment

### 9.1 Deployment Models

| Model | Mô tả |
|---|---|
| **Development** | Single binary (monolith) + Docker Compose infra |
| **Production** | Gateway + distributed engine services, Kubernetes + Helm |

### 9.2 Service Ports (Monolith)

| Service | Port | Health Check |
|---|---|---|
| Memory REST API | 8080 | `/v1/admin/health` |
| MCP Server | 8082 | `/sse` (SSE test) |
| Health + Metrics | 8083 | `/healthz` |
| PostgreSQL | 5432 | pg_isready |
| Neo4j | 7687 | bolt |
| Qdrant | 6333 | `/healthz` |
| Redis | 6379 | PING |
| MinIO | 9000 | — |

### 9.3 Quick Start

```bash
# Start infrastructure
make infra-up

# Run monolith (all 35 services in one process)
make dev

# Verify (lists all 35 services)
curl http://localhost:8083/healthz | jq

# REST API test
curl -X POST http://localhost:8080/v1/memory/store \
  -H "Content-Type: application/json" \
  -d '{"content":"hello world","type":"fact"}'

# MCP tools
curl -X POST http://localhost:8082/message \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

---

## 10. Security & Compliance

### 10.1 Security Requirements

| Requirement | Implementation |
|---|---|
| **Authentication** | API Key (SHA-256 hash) + JWT RS256 |
| **Authorization** | Rate tier per tenant (free/pro/enterprise) |
| **Tenant Isolation** | TenantID injected vào mọi domain query |
| **Encryption** | TLS in transit; VikingFS ov-crypto cho at-rest |
| **Secret Redaction** | Auto-redact API keys, tokens in logs/traces |
| **Dev Mode Guard** | AUTH_DEV_MODE chỉ accept localhost traffic |

### 10.2 Governance Features

| Feature | Mô tả |
|---|---|
| **GDPR Forget** | `POST /v1/console/governance/gdpr/forget` — cascading across engines |
| **GDPR Preview** | Dry-run mode trước khi xóa |
| **Audit Trail** | `GET /v1/console/governance/audit` — searchable audit logs |
| **Retention Policies** | `forgetAfter` per memory type (Supermemory), TTL config |
| **OPA Policies** | `POST/PUT /v1/console/governance/policies` |
| **Tenant Management** | `GET/POST/PUT /v1/console/governance/tenants` |
| **API Key Lifecycle** | Create → Active → Revoked/Expired |

---

## 11. Observability

| Signal | Implementation |
|---|---|
| **Tracing** | OpenTelemetry distributed tracing across all engines |
| **Metrics** | Prometheus — latency, throughput, error rates, LLM costs |
| **Logging** | slog structured JSON, secret redaction |
| **Health** | Per-service health endpoints + aggregated `/healthz` |
| **Console** | `/v1/console/observability/*` — metrics, traces, errors, costs |

---

## 12. Non-Functional Requirements

### 12.1 Performance

| Metric | Target |
|---|---|
| Memobase context retrieval (p95) | < 100ms |
| Conversational context assembly (p95) | < 200ms (Zep) |
| Hierarchical search (p95) | < 500ms |
| Graph search (p95) | < 1000ms |
| Knowledge graph construction | Background async, non-blocking |
| Concurrent sessions | ≥ 1,000 per instance |
| Token cost reduction vs naive RAG | ≥ 80% |
| LLM calls per Memobase flush | Fixed 3 calls |

### 12.2 Scalability

| Dimension | Approach |
|---|---|
| **Horizontal** | Stateless gateway, engine replicas |
| **In-process** | Monolith — 35 services share process, bufconn zero-copy |
| **NATS** | Embedded in monolith, external for distributed |
| **Multi-tenant** | TenantID isolation, per-tenant quotas |

### 12.3 Reliability

| Requirement | Implementation |
|---|---|
| **Availability** | ≥ 99.9% API uptime |
| **Circuit Breaker** | Downstream protection, NATS event on open |
| **Fallback LLM** | Bifrost multi-provider routing |
| **Data Durability** | PostgreSQL ACID + WAL |
| **Graceful Shutdown** | HTTP drain → NATS drain → gRPC stop → DB close |

---

## 13. Integration & Interoperability

### 13.1 Agent Framework Support

| Framework | Integration Method |
|---|---|
| LangChain | Python SDK |
| LangGraph | Python SDK |
| CrewAI | REST API |
| AutoGen | REST API / MCP |
| OpenAI Agents SDK | REST API / MCP |
| Claude Code | MCP Server (SSE) |
| Vercel AI SDK | REST API (Supermemory) |
| Mastra | REST API |
| n8n | REST API |

### 13.2 MCP Protocol

MCP Server expose 16 tools qua SSE và HTTP Streamable transport. Hỗ trợ:
- Claude Desktop (stdio)
- Claude Code (MCP)
- Any MCP-compatible client

### 13.3 LLM Provider Support

Qua **Bifrost AI Gateway**:
- OpenAI, Azure OpenAI, Anthropic, Google Gemini
- Groq, Mistral, Ollama (local)
- HuggingFace, llama.cpp
- OpenRouter (multi-model)

---

## 14. Roadmap

### Phase 1 — Foundation ✅ (Complete)
- ✅ 35-service monolith architecture
- ✅ VNP Gateway với 50+ REST routes
- ✅ MCP Server với 16 tools
- ✅ WebSocket real-time events
- ✅ Memobase profile memory (YOLO engine)
- ✅ Supermemory adaptive memory
- ✅ Console API routes (12 feature areas)
- ✅ Embedded NATS JetStream
- ✅ InProcessRegistry (bufconn)

### Phase 2 — Console UI 🔲
- 🔲 SPA embedded UI (apps/memory/internal/ui)
- 🔲 Dashboard với real-time metrics
- 🔲 Memory Explorer với graph visualization
- 🔲 Profile management UI
- 🔲 Agent Context Debugger

### Phase 3 — Advanced Memory 🔲
- 🔲 Cross-engine memory deduplication
- 🔲 Memory decay & salience scoring
- 🔲 Supermemory connector auto-sync (Google Drive, GitHub, Notion)
- 🔲 Graphiti + Memobase integration (temporal profiles)

### Phase 4 — Enterprise 🔲
- 🔲 OPA policy engine integration
- 🔲 Kubernetes Helm charts
- 🔲 Multi-region deployment
- 🔲 SurrealDB unified storage evaluation
- 🔲 SDK (Python, TypeScript, Go)

---

## 15. Success Metrics

| Metric | Target | Measurement |
|---|---|---|
| Memobase context retrieval (p95) | < 100ms | Prometheus |
| Conversational context retrieval (p95) | < 200ms | Prometheus |
| Cross-engine recall (p95) | < 500ms | Prometheus |
| Token cost reduction | ≥ 80% | Usage monitoring |
| Tenant isolation verification | 100% (zero leaks) | Integration tests |
| API uptime | ≥ 99.9% | Health monitoring |
| MCP tool success rate | ≥ 99% | Traces |
| Profile extraction accuracy | ≥ 85% relevance | Eval framework |

---

## 16. Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| **Integration complexity** | 6 engines + in-process = complex orchestration | Monolith simplifies dev, distributed for scale |
| **Memobase token budget** | LLM flush cost scales with users | Fixed 3-call YOLO engine, async processing |
| **Memory explosion** | Storage grows unbounded | Supermemory forgetAfter, Memobase TTL |
| **Context assembly latency** | Agent cần < 1s response | Memobase < 100ms, precomputed profiles |
| **Multi-tenant isolation** | Cross-tenant data leak | TenantID on every query, integration tests |
| **LLM provider dependency** | Single provider failure | Bifrost multi-provider failover |
| **NATS embedded single point** | Monolith NATS failure | External NATS option (VNP_MEMORY_NATS_MODE=external) |

---

## 17. Component References

| Component | Location |
|---|---|
| **Gateway** | `gateway/` — REST, MCP, WebDAV |
| **Monolith** | `apps/memory/` — 35 services, embedded NATS |
| **Memory Service** | `services/memory-service/` — Memobase, Zep, SM domains |
| **Pipeline Service** | `services/pipeline-service/` — Job management |
| **Search Service** | `services/search-service/` — Cross-engine search |
| **Storage Service** | `services/storage-service/` — VikingFS, sessions |
| **VNP Platform** | `services/vnp-platform/` — Admin, Auth, Events |
| **Obs Service** | `services/obs-service/` — Observability infra |

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
