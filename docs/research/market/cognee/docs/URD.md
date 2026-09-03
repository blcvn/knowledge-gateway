# User Requirements Document (URD)

**Product:** Cognee — AI Knowledge Engine  
**Version:** 1.0.3  
**Status:** Production (Beta)  
**Last Updated:** 2026-05-07  
**Source:** Derived from codebase analysis of `topoteretes/cognee`  
**License:** Apache-2.0

---

## 1. Giới Thiệu

### 1.1 Mục đích tài liệu

Tài liệu này mô tả các yêu cầu từ phía người dùng (User Requirements) đối với hệ thống Cognee — một AI Knowledge Engine mã nguồn mở. URD tập trung vào **ai dùng**, **dùng để làm gì**, và **trải nghiệm mong đợi** thay vì chi tiết kỹ thuật triển khai.

### 1.2 Phạm vi sản phẩm

Cognee cung cấp khả năng:
- Nhập dữ liệu đa định dạng và chuyển đổi thành knowledge graph có cấu trúc
- Tìm kiếm đa chiến lược kết hợp vector search, graph traversal, và LLM completion
- Quản lý bộ nhớ dài hạn cho AI Agent với session-to-graph bridging
- Hỗ trợ multi-tenant với per-user dataset isolation

### 1.3 Định nghĩa & Viết tắt

| Thuật ngữ | Định nghĩa |
|-----------|------------|
| **ECL** | Extract-Cognify-Load — Pipeline xử lý dữ liệu cốt lõi |
| **Knowledge Graph** | Đồ thị tri thức biểu diễn entities và relationships |
| **RAG** | Retrieval-Augmented Generation |
| **NodeSet** | Cơ chế phân vùng và gắn tag cho memory |
| **DataPoint** | Đơn vị tri thức atomic với schema tùy chỉnh |
| **MCP** | Model Context Protocol — giao thức kết nối AI với tools |

---

## 2. User Personas

### 2.1 Primary Personas

#### P1 — AI/ML Engineer

| Thuộc tính | Mô tả |
|-----------|--------|
| **Vai trò** | Xây dựng AI Agent có khả năng nhớ và học |
| **Kỹ năng** | Python async, LLM APIs, vector databases |
| **Mục tiêu** | Thêm persistent memory vào agent mà không cần tự xây infrastructure |
| **Pain point** | Agent mất ngữ cảnh giữa các phiên; RAG truyền thống không hiểu quan hệ |
| **Use cases** | Customer support bots, coding assistants, research agents |

#### P2 — Backend Developer

| Thuộc tính | Mô tả |
|-----------|--------|
| **Vai trò** | Tích hợp knowledge base vào ứng dụng backend |
| **Kỹ năng** | REST API, Docker, database management |
| **Mục tiêu** | Triển khai hệ thống knowledge management có thể mở rộng |
| **Pain point** | Dữ liệu phân tán, thiếu cách truy vấn thống nhất |
| **Use cases** | Enterprise search, document Q&A, knowledge management |

#### P3 — Data Scientist

| Thuộc tính | Mô tả |
|-----------|--------|
| **Vai trò** | Phân tích và khai thác knowledge graph |
| **Kỹ năng** | Python, data analysis, graph theory |
| **Mục tiêu** | Khám phá patterns và insights từ dữ liệu phi cấu trúc |
| **Pain point** | Chuyển đổi dữ liệu thô thành cấu trúc có thể truy vấn tốn nhiều effort |
| **Use cases** | Research analytics, pattern discovery, trend analysis |

#### P4 — DevOps / Platform Engineer

| Thuộc tính | Mô tả |
|-----------|--------|
| **Vai trò** | Triển khai và vận hành hệ thống memory ở production |
| **Kỹ năng** | Docker, Kubernetes, monitoring, CI/CD |
| **Mục tiêu** | Self-host và scale hệ thống với high availability |
| **Pain point** | Complexity trong việc cấu hình nhiều database backends |
| **Use cases** | Infrastructure management, monitoring, scaling |

### 2.2 Secondary Personas

#### P5 — Product Team (Cognee Cloud)
- Sử dụng managed service, không cần quản lý infrastructure
- Quan tâm đến dashboard, usage metrics, team collaboration

#### P6 — Researcher
- Nghiên cứu Graph RAG, knowledge representation, cognitive science
- Sử dụng evaluation framework và notebooks

#### P7 — IDE/Agent User (Claude Code, Hermes, Cursor)
- Tương tác gián tiếp qua MCP server hoặc plugin
- Cần memory tự động, zero-config

---

## 3. User Requirements

### 3.1 Data Ingestion (UR-ING)

#### UR-ING-01: Nhập dữ liệu đa định dạng
**Persona:** P1, P2, P3  
**Priority:** Critical  
**Mô tả:** Người dùng cần nhập dữ liệu từ nhiều nguồn và định dạng khác nhau vào hệ thống một cách thống nhất.

**Acceptance Criteria:**
- Hỗ trợ text strings, file paths, URLs, và mixed lists
- Hỗ trợ định dạng: PDF, TXT, MD, CSV, DOCX, PPTX, audio, image
- Hỗ trợ web scraping từ URL
- Tổ chức dữ liệu theo dataset với tên do người dùng đặt
- API đơn giản: `await cognee.add(data, dataset_name="research")`

#### UR-ING-02: Gắn tag và phân vùng dữ liệu (NodeSets)
**Persona:** P1, P2  
**Priority:** High  
**Mô tả:** Người dùng cần phân loại và gắn tag dữ liệu theo nhiều chiều (customer, workflow, topic).

**Acceptance Criteria:**
- Gắn nhiều tags cho mỗi data item: `node_set=["customer_123", "preferences"]`
- Tìm kiếm trong phạm vi tag cụ thể
- Hỗ trợ patterns: per-customer, per-workflow, per-topic, per-environment

#### UR-ING-03: Custom data schema (DataPoints)
**Persona:** P1, P3  
**Priority:** Medium  
**Mô tả:** Người dùng nâng cao cần định nghĩa schema riêng cho dữ liệu.

**Acceptance Criteria:**
- Extend `DataPoint` class với custom fields
- Chỉ định index fields cho search
- Tự động tạo embeddings cho các fields được đánh dấu

---

### 3.2 Knowledge Processing (UR-PROC)

#### UR-PROC-01: Tự động xây dựng Knowledge Graph
**Persona:** P1, P2, P3  
**Priority:** Critical  
**Mô tả:** Hệ thống tự động chuyển đổi dữ liệu thô thành knowledge graph có cấu trúc.

**Acceptance Criteria:**
- Pipeline tự động: classify → chunk → extract entities → build graph → embed
- Cấu hình được chunk_size
- Hỗ trợ custom prompt cho entity extraction
- Hỗ trợ ontology grounding (OWL) để chuẩn hóa entities
- API đơn giản: `await cognee.cognify(datasets="research")`

#### UR-PROC-02: Temporal knowledge extraction
**Persona:** P1, P3  
**Priority:** Medium  
**Mô tả:** Trích xuất sự kiện và dòng thời gian từ dữ liệu.

**Acceptance Criteria:**
- Mode temporal_cognify trích xuất events và timestamps
- Xây dựng temporal knowledge graph
- Hỗ trợ truy vấn theo timeline

#### UR-PROC-03: Graph enrichment (Memify)
**Persona:** P1, P3  
**Priority:** Medium  
**Mô tả:** Làm giàu graph hiện có với derived facts, rules, patterns mà không cần rebuild.

**Acceptance Criteria:**
- Enrichment không phá hủy graph hiện tại
- Thêm triplet embeddings và index
- API: `await cognee.memify(dataset="research")`

---

### 3.3 Search & Retrieval (UR-SEARCH)

#### UR-SEARCH-01: Tìm kiếm đa chiến lược
**Persona:** P1, P2, P3  
**Priority:** Critical  
**Mô tả:** Người dùng cần nhiều chiến lược tìm kiếm phù hợp với các use cases khác nhau.

**Acceptance Criteria:**
- **GRAPH_COMPLETION**: Q&A phức tạp kết hợp graph + LLM
- **RAG_COMPLETION**: Tìm kiếm truyền thống over chunks
- **CHUNKS**: Semantic vector search nhanh
- **CHUNKS_LEXICAL**: Tìm kiếm từ khóa chính xác (Jaccard)
- **SUMMARIES**: Tóm tắt có sẵn
- **CYPHER**: Raw graph queries cho advanced users
- **NATURAL_LANGUAGE**: NL → structured query
- **TEMPORAL**: Time-aware search
- **FEELING_LUCKY**: Auto-select strategy tốt nhất

#### UR-SEARCH-02: Chain-of-thought và query decomposition
**Persona:** P1, P3  
**Priority:** High  
**Mô tả:** Hỗ trợ reasoning phức tạp qua nhiều bước.

**Acceptance Criteria:**
- `GRAPH_COMPLETION_COT`: Multi-step reasoning
- `GRAPH_COMPLETION_DECOMPOSITION`: Tách câu hỏi phức hợp
- `GRAPH_COMPLETION_CONTEXT_EXTENSION`: Mở rộng ngữ cảnh

#### UR-SEARCH-03: Feedback loop
**Persona:** P1  
**Priority:** Medium  
**Mô tả:** Agent cần khả năng tự cải thiện từ phản hồi.

**Acceptance Criteria:**
- Lưu interactions với `save_interaction=True`
- Gửi feedback qua `SearchType.FEEDBACK`
- Feedback weights ảnh hưởng đến ranking tương lai

---

### 3.4 Memory API — V2 (UR-MEM)

#### UR-MEM-01: Remember — Lưu trữ memory đơn giản
**Persona:** P1, P7  
**Priority:** Critical  
**Mô tả:** Agent cần lưu thông tin nhanh chóng mà không cần hiểu pipeline nội bộ.

**Acceptance Criteria:**
- Permanent store: `await cognee.remember("fact")`
- Session store: `await cognee.remember("preference", session_id="chat_1")`
- Hỗ trợ structured entries: QAEntry, TraceEntry, FeedbackEntry
- Background processing (non-blocking)
- `RememberResult` là promise-like object (printable, awaitable, inspectable)

#### UR-MEM-02: Recall — Truy xuất memory thông minh
**Persona:** P1, P7  
**Priority:** Critical  
**Mô tả:** Agent cần truy xuất thông tin liên quan từ memory.

**Acceptance Criteria:**
- Auto-routing: `await cognee.recall("What does the user prefer?")`
- Session-scoped: `await cognee.recall(query, session_id="chat_1")`
- Multi-scope: graph, session, trace, graph_context, all
- Session cache first, fall-through to graph search

#### UR-MEM-03: Forget — Xóa memory
**Persona:** P1, P2  
**Priority:** High  
**Mô tả:** Người dùng cần xóa dữ liệu và memory khi không còn cần thiết.

**Acceptance Criteria:**
- Xóa theo dataset: `await cognee.forget(dataset="main_dataset")`
- Xóa toàn bộ: `cognee-cli forget --all`
- Cascade delete: dữ liệu, graph, vector indices

#### UR-MEM-04: Improve — Tự cải thiện
**Persona:** P1  
**Priority:** Medium  
**Mô tả:** Hệ thống tự cải thiện chất lượng retrieval theo thời gian.

**Acceptance Criteria:**
- Triplet embedding + index refresh
- Background execution
- API: `await cognee.improve(dataset="research")`

---

### 3.5 Authentication & Multi-tenancy (UR-AUTH)

#### UR-AUTH-01: Xác thực người dùng
**Persona:** P2, P4  
**Priority:** High  
**Mô tả:** Hệ thống cần hỗ trợ xác thực an toàn.

**Acceptance Criteria:**
- JWT-based bearer token authentication
- API key authentication (`X-Api-Key` header)
- Cookie-based session
- Password reset và email verification flows
- Chế độ unauthenticated cho development (`REQUIRE_AUTHENTICATION=false`)

#### UR-AUTH-02: Multi-tenant isolation
**Persona:** P2, P4  
**Priority:** High  
**Mô tả:** Nhiều users/tenants cần dữ liệu hoàn toàn tách biệt.

**Acceptance Criteria:**
- Per-user dataset scoping
- Per-tenant knowledge separation
- Isolated graph + vector DB instances khi `ENABLE_BACKEND_ACCESS_CONTROL=True`
- Permission types: read, write, delete, share
- Role-based access control (RBAC)

---

### 3.6 Agent Integration (UR-AGENT)

#### UR-AGENT-01: Agent memory decorator
**Persona:** P1  
**Priority:** High  
**Mô tả:** AI developers cần inject memory vào agent function dễ dàng.

**Acceptance Criteria:**
- Decorator pattern: `@agent_memory(with_memory=True)`
- Tự động prepend relevant memory context vào LLM prompts
- Configurable: top_k, session_memory, trace persistence
- Không yêu cầu thay đổi agent logic hiện có

#### UR-AGENT-02: MCP Server cho IDE integration
**Persona:** P7  
**Priority:** High  
**Mô tả:** AI coding assistants cần truy cập Cognee memory qua MCP protocol.

**Acceptance Criteria:**
- Transport: stdio (Claude Desktop), SSE, HTTP
- MCP Tools: cognify, search, save_interaction, list_data, delete_dataset, cognify_status
- Direct mode (local) và API mode (remote)
- Dockerized deployment

#### UR-AGENT-03: Framework integration
**Persona:** P1  
**Priority:** Medium  
**Mô tả:** Tích hợp với các AI agent framework phổ biến.

**Acceptance Criteria:**
- LangChain integration
- LlamaIndex integration
- Claude Code plugin (hooks: SessionStart, PostToolUse, SessionEnd)
- Hermes Agent integration

---

### 3.7 Deployment & Operations (UR-OPS)

#### UR-OPS-01: Zero-config local development
**Persona:** P1, P2, P3  
**Priority:** Critical  
**Mô tả:** Khởi động nhanh với minimal configuration.

**Acceptance Criteria:**
- Chỉ cần 1 env var: `LLM_API_KEY`
- Default stack zero-setup: SQLite + LanceDB + Kuzu (file-based)
- `pip install cognee` → sử dụng ngay
- Hỗ trợ Python 3.10 – 3.13

#### UR-OPS-02: Docker deployment
**Persona:** P4  
**Priority:** High  
**Mô tả:** Triển khai production với Docker.

**Acceptance Criteria:**
- `docker compose up` cho basic deployment
- Docker profiles cho optional services: neo4j, chromadb, postgres, redis, mcp, ui
- Health checks tích hợp
- Resource limits configurable (CPU, memory)

#### UR-OPS-03: Cloud deployment
**Persona:** P4  
**Priority:** Medium  
**Mô tả:** Triển khai trên cloud platforms.

**Acceptance Criteria:**
- Cognee Cloud: `await cognee.serve(url=..., api_key=...)`
- Modal (serverless), Railway, Fly.io, Render, Daytona
- Distributed workers via Modal cho horizontal scaling

#### UR-OPS-04: Observability
**Persona:** P4  
**Priority:** High  
**Mô tả:** Monitoring và tracing cho production operations.

**Acceptance Criteria:**
- OpenTelemetry tracing cho pipeline, LLM calls, DB queries
- Structured logging (structlog) với configurable log level
- Tích hợp Langfuse, Sentry, PostHog
- Activity tracking endpoint (`/api/v1/activity`)
- Secret auto-redaction trong traces/logs
- OTLP export tương thích Grafana, Datadog, Jaeger

---

### 3.8 Developer Experience (UR-DX)

#### UR-DX-01: Python SDK (async-first)
**Persona:** P1, P2, P3  
**Priority:** Critical  
**Mô tả:** SDK Python dễ sử dụng, async-native.

**Acceptance Criteria:**
- Import đơn giản: `import cognee`
- Tất cả operations đều async
- Type hints đầy đủ
- Documentation và examples

#### UR-DX-02: CLI
**Persona:** P1, P2  
**Priority:** Medium  
**Mô tả:** Command-line interface cho quick operations.

**Acceptance Criteria:**
- `cognee-cli remember "..."` / `recall "..."` / `forget --all`
- `cognee-cli -ui` — launch full UI stack

#### UR-DX-03: REST API
**Persona:** P2  
**Priority:** High  
**Mô tả:** REST API cho cross-language integration.

**Acceptance Criteria:**
- FastAPI server tại `http://localhost:8000`
- OpenAPI documentation tự động
- Hỗ trợ cả Bearer token và API key auth
- CORS configurable

#### UR-DX-04: Custom pipelines
**Persona:** P1, P3  
**Priority:** Medium  
**Mô tả:** Người dùng nâng cao cần tạo pipeline tùy chỉnh.

**Acceptance Criteria:**
- Wrap custom functions trong `Task()`
- Composable, reorderable pipeline steps
- `await cognee.run_custom_pipeline(tasks=[...], data=..., dataset=...)`

#### UR-DX-05: Frontend UI
**Persona:** P1, P3  
**Priority:** Low  
**Mô tả:** UI trực quan cho development và demos.

**Acceptance Criteria:**
- Next.js visualization UI
- Graph visualization
- Khởi chạy: `cd cognee-frontend && npm install && npm run dev`

---

## 4. User Workflows (SOPs)

### 4.1 SOP-01: Quick Start — First-time Setup

```
1. Cài đặt: pip install cognee (hoặc uv pip install cognee)
2. Thiết lập: export LLM_API_KEY="sk-..."
3. Sử dụng:
   import cognee
   await cognee.add("data.pdf", dataset_name="docs")
   await cognee.cognify(datasets="docs")
   results = await cognee.search("What is in the document?")
```

### 4.2 SOP-02: Agent Memory Integration

```
1. Cài đặt cognee trong agent environment
2. Thêm decorator @agent_memory(with_memory=True) vào agent function
3. LLMGateway tự động prepend memory context
4. Agent traces được lưu và học từ interactions
```

### 4.3 SOP-03: V2 Memory API cho Chatbot

```
1. User gửi message → agent xử lý
2. Lưu interaction: await cognee.remember(QAEntry(question=..., answer=...), session_id="chat_1")
3. Recall context: results = await cognee.recall("relevant query", session_id="chat_1")
4. Gửi feedback: await cognee.remember(FeedbackEntry(qa_id=..., feedback_score=5))
5. Improve: await cognee.improve(dataset="chat_memory")
```

### 4.4 SOP-04: Production Deployment

```
1. Clone repo và cấu hình .env (copy từ .env.template)
2. Chọn database stack:
   - Dev: SQLite + LanceDB + Kuzu (default)
   - Production: PostgreSQL + pgvector + Neo4j
3. docker compose up (+ profiles nếu cần)
4. Verify: curl http://localhost:8000/health
5. Bật monitoring: COGNEE_TRACING_ENABLED=true + OTEL endpoint
6. Cấu hình auth: REQUIRE_AUTHENTICATION=true
```

### 4.5 SOP-05: MCP Server Setup cho IDE

```
1. Cấu hình MCP trong IDE (Claude Desktop / Cursor / VS Code)
2. Direct mode: uv run python src/server.py --transport sse
3. API mode: uv run python src/server.py --transport sse --api-url http://localhost:8000 --api-token TOKEN
4. Docker: docker compose --profile mcp up
5. Sử dụng MCP tools: cognify, search, save_interaction
```

---

## 5. Non-Functional User Requirements

### 5.1 Performance (UR-NFR-PERF)
- **UR-NFR-PERF-01**: Search response < 5 giây cho typical queries
- **UR-NFR-PERF-02**: Background cognify không block UI/API
- **UR-NFR-PERF-03**: Rate limiting tích hợp cho LLM calls

### 5.2 Reliability (UR-NFR-REL)
- **UR-NFR-REL-01**: Fallback LLM khi primary provider lỗi
- **UR-NFR-REL-02**: Retry logic tự động (via Tenacity)
- **UR-NFR-REL-03**: Database migrations backward compatible (Alembic)

### 5.3 Usability (UR-NFR-USE)
- **UR-NFR-USE-01**: Zero-config startup (chỉ cần LLM_API_KEY)
- **UR-NFR-USE-02**: Comprehensive examples và tutorials
- **UR-NFR-USE-03**: Jupyter notebook support

### 5.4 Security (UR-NFR-SEC)
- **UR-NFR-SEC-01**: API keys hashed (SHA-256) khi lưu trữ
- **UR-NFR-SEC-02**: Auto-redaction secrets trong logs/traces
- **UR-NFR-SEC-03**: DNS rebinding protection cho MCP server
- **UR-NFR-SEC-04**: CORS configurable

### 5.5 Compatibility (UR-NFR-COMP)
- **UR-NFR-COMP-01**: Python 3.10 – 3.13
- **UR-NFR-COMP-02**: macOS, Linux, Windows
- **UR-NFR-COMP-03**: ARM64 và x86_64
- **UR-NFR-COMP-04**: Pluggable backends (LLM, graph, vector, relational, storage)

---

## 6. Traceability Matrix

| User Requirement | PRD Section | Spec Document |
|-----------------|-------------|---------------|
| UR-ING-01 | §4.1.1 | L2-core-operations-layer.md |
| UR-ING-02 | §4.2 | L5-domain-modules-layer.md |
| UR-ING-03 | §4.3 | L6-infrastructure-adapters-layer.md |
| UR-PROC-01 | §4.1.2 | L3-pipeline-orchestration-layer.md |
| UR-PROC-02 | §4.1.2 | L4-task-execution-layer.md |
| UR-SEARCH-01 | §4.1.3 | L5-domain-modules-layer.md |
| UR-MEM-01..04 | §4.1.4 | L2-core-operations-layer.md |
| UR-AUTH-01..02 | §12 | L5-domain-modules-layer.md §6 |
| UR-AGENT-01..03 | §10 | L5-domain-modules-layer.md §7 |
| UR-OPS-01..04 | §9, §7 | L7-external-services-layer.md |
| UR-DX-01..05 | §13 | L1-public-api-layer.md |
