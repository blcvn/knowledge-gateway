---
version: 1.1.0
last_updated: 2026-04-21
updated_by: ai-agent-architect
status: Approved
scope: REPO-LEVEL
doc_id: DOC-G02
---

# Skill Set Catalog
## Requirement-to-UI Automation Platform

> Tài liệu này liệt kê **toàn bộ các bộ kỹ năng (Skill Sets)** cần thiết để xây dựng sản phẩm ở mức chất lượng cao nhất. Mỗi bộ kỹ năng được phân loại theo nhóm chức năng, ghi rõ trạng thái hiện tại (đã có / cần tạo), và ánh xạ đến các agent sử dụng chúng.

---

## Tổng Quan

| Nhóm | Số Bộ Skills | Đã Có | Cần Tạo |
|---|---|---|---|
| 🏛️ Kiến trúc & Thiết kế | 3 | 1 | 2 |
| ⚙️ Backend Development | 2 | 1 | 1 |
| 🎨 Frontend Development | 2 | 2 | 0 |
| 🤖 AI & Data Engineering | 3 | 0 | 3 |
| 🧪 Quality Assurance | 2 | 2 | 0 |
| 🔒 Security Engineering | 1 | 0 | 1 |
| 🚀 DevOps & Infrastructure | 1 | 0 | 1 |
| 📋 Quản lý & Governance | 2 | 2 | 0 |
| **Tổng** | **16** | **8** | **8** |

---

## 🏛️ Nhóm 1: Kiến trúc & Thiết kế Phần Mềm

---

### SKILL-001 · Software Architecture & System Design
**Trạng thái:** 🔴 Cần tạo
**Đường dẫn đề xuất:** `.agent/skills/software-architect/`

**Mô tả:** Bộ kỹ năng thiết kế tổng thể hệ thống phần mềm — từ phân tích yêu cầu đến lựa chọn kiến trúc (microservices, event-driven, pipeline), thiết kế API contract, và đưa ra các Architecture Decision Records (ADR).

**Năng lực cốt lõi:**
- Microservices Architecture: Service boundary định nghĩa, inter-service communication (REST/gRPC/Kafka)
- Multi-stage Pipeline Design: Thiết kế luồng xử lý tuần tự có checkpoints và error recovery
- API Contract Design: OpenAPI/Swagger specification, versioning strategy
- Data Flow Architecture: Thiết kế luồng dữ liệu từ input (PRD text) → KG → Schema → UI
- Scalability Patterns: Horizontal scaling, stateless services, caching strategy
- Architecture Decision Records (ADR): Ghi lại mọi quyết định kiến trúc với context và trade-offs

**Agents sử dụng:** Tất cả agents (nền tảng thiết kế)
**Liên kết tài liệu:** `docs/adr/`, `docs/product/architecture.md`

---

### SKILL-002 · UI/UX Design
**Trạng thái:** ✅ Đã có
**Đường dẫn:** `.agent/skills/ui-ux-design-expert/`

**Mô tả:** Phân tích yêu cầu → thiết kế giao diện, đảm bảo consistency theo Design System, tối ưu trải nghiệm người dùng.

**Năng lực cốt lõi:** Requirement analysis, Design System (Atomic Design, Design Tokens), Visual hierarchy, Micro-interactions, Responsive design, Accessibility (WCAG 2.1)

**Agents sử dụng:** `ui-schema-generator-agent`, `ui-renderer-agent`

---

### SKILL-003 · API Design & Integration Patterns
**Trạng thái:** 🔴 Cần tạo
**Đường dẫn đề xuất:** `.agent/skills/api-design-expert/`

**Mô tả:** Thiết kế và tích hợp các API — REST, gRPC, GraphQL. Đảm bảo API contract nhất quán, an toàn, và có versioning rõ ràng giữa các services trong pipeline.

**Năng lực cốt lõi:**
- RESTful API Design: Resource naming, HTTP semantics, status codes, pagination
- gRPC / Protocol Buffers: Service definition, streaming, bidirectional communication
- API Versioning: URL versioning vs header versioning, backward compatibility
- Rate Limiting & Throttling: Token bucket, sliding window algorithms
- API Gateway Pattern: Routing, authentication delegation, request transformation
- OpenAPI Specification: Viết và validate spec, auto-generate client/server stubs

**Agents sử dụng:** `knowledge-graph-agent`, `ui-schema-generator-agent`, `qa-pipeline-agent`

---

## ⚙️ Nhóm 2: Backend Development

---

### SKILL-004 · Backend Development — Golang
**Trạng thái:** ✅ Đã có
**Đường dẫn:** `.agent/skills/golang-expert/`

**Mô tả:** Phát triển backend hiệu năng cao với Golang — concurrency patterns, memory management, design patterns cho stability và performance.

**Năng lực cốt lõi:** Goroutines/Channels, sync primitives, Worker Pool, Circuit Breaker, `sync.Pool`, pprof profiling, database connection pooling

**Agents sử dụng:** `requirement-parser-agent`, `semantic-extractor-agent`, `knowledge-graph-agent`, `ui-schema-generator-agent`

---

### SKILL-005 · Graph Database Engineering (Neo4j / ArangoDB)
**Trạng thái:** 🔴 Cần tạo
**Đường dẫn đề xuất:** `.agent/skills/graph-db-expert/`

**Mô tả:** Thiết kế ontology, xây dựng và tối ưu hóa Knowledge Graph. Chuyên sâu về Cypher query language, graph schema design, và tích hợp graph DB vào pipeline.

**Năng lực cốt lõi:**
- Ontology Design: Node types, relationship types, property schemas cho domain cụ thể
- Cypher Query Language (Neo4j): MATCH, MERGE, CREATE, WITH, aggregation, path patterns
- Graph Data Modeling: Biết khi nào dùng node vs property vs relationship
- Idempotent Upsert: `MERGE` patterns để tránh duplicate khi chạy lại pipeline
- Index & Constraint Design: `UNIQUE`, `INDEX` để tối ưu query performance
- Graph Traversal Algorithms: Shortest path, connected components, subgraph extraction
- ArangoDB / AQL: Tương đương cho ArangoDB nếu được chọn

**Agents sử dụng:** `knowledge-graph-agent`, `traceability-validator-agent`
**Liên kết tài liệu:** `services/knowledge-graph-service/docs/`

---

## 🎨 Nhóm 3: Frontend Development

---

### SKILL-006 · Frontend Development — React + TypeScript
**Trạng thái:** ✅ Đã có
**Đường dẫn:** `.agent/skills/react-ts-expert/`

**Mô tả:** Phát triển giao diện React + TypeScript với kiến trúc tổ chức code chuẩn, performance tối ưu, trải nghiệm người dùng tốt nhất và an toàn bảo mật.

**Năng lực cốt lõi:** Feature-Sliced Design, React Query/SWR, Zustand, TypeScript strict mode, Code splitting, Core Web Vitals, XSS/CSRF prevention

**Agents sử dụng:** `ui-renderer-agent`, `ui-schema-generator-agent`

---

### SKILL-007 · Dynamic UI Schema Rendering (JSON Schema → React)
**Trạng thái:** ✅ Đã có (trong `react-ts-expert`)
**Đường dẫn:** `.agent/skills/react-ts-expert/`

**Mô tả:** Kỹ năng chuyên biệt render React components động từ JSON Schema — Component Factory pattern, dynamic form generation, wizard/stepper rendering.

**Năng lực cốt lõi:**
- Component Factory Pattern: `switch/map` theo schema `type` để render đúng component
- JSON Schema Form: `react-jsonschema-form`, `react-hook-form` + Zod validation
- Dynamic Navigation: Render workflow transitions từ KG thành clickable navigation
- Tailwind CSS + Shadcn/MUI: Implement design tokens nhất quán
- Code Export: Generate clean, production-ready React code từ rendered schema

**Agents sử dụng:** `ui-renderer-agent`

---

## 🤖 Nhóm 4: AI & Data Engineering

---

### SKILL-008 · LLM Engineering & Prompt Design
**Trạng thái:** 🔴 Cần tạo
**Đường dẫn đề xuất:** `.agent/skills/llm-engineer/`

**Mô tả:** Thiết kế và tối ưu hóa prompts cho LLM để thực hiện các tác vụ phân tích văn bản phức tạp — trích xuất entities, phân loại đoạn văn, suy luận business rules — với độ chính xác và tính ổn định cao.

**Năng lực cốt lõi:**
- Prompt Engineering: Zero-shot, few-shot, chain-of-thought, structured output (JSON mode)
- LLM Selection: Biết khi nào dùng GPT-4o vs Claude vs Gemini vs local model (Llama) cho từng task
- Output Parsing & Validation: Parse JSON từ LLM output, xử lý malformed response
- Hallucination Mitigation: Grounding techniques, confidence scoring, fallback strategies
- Context Window Management: Chunking large documents, RAG (Retrieval-Augmented Generation)
- LLM Cost Optimization: Token counting, caching identical prompts, batching requests
- Evaluation & Benchmarking: Đánh giá độ chính xác extraction qua test set chuẩn

**Agents sử dụng:** `requirement-parser-agent`, `semantic-extractor-agent`

---

### SKILL-009 · Natural Language Processing (NLP)
**Trạng thái:** 🔴 Cần tạo
**Đường dẫn đề xuất:** `.agent/skills/nlp-expert/`

**Mô tả:** Xử lý và phân tích văn bản phi cấu trúc — tokenization, entity recognition, semantic similarity, text classification — để chuẩn bị input cho LLM và post-process output.

**Năng lực cốt lõi:**
- Text Preprocessing: Normalization, cleaning, language detection, sentence segmentation
- Named Entity Recognition (NER): Domain-specific NER cho business entities (Actor, Action, Constraint)
- Text Classification: Phân loại đoạn văn thành categories (Overview, Functional, API, etc.)
- Semantic Similarity: Embedding-based deduplication của entities (e.g., "Order" vs "Đơn hàng")
- Dependency Parsing: Trích xuất subject-verb-object cho relationship inference
- Entity Deduplication & Normalization: Chuẩn hóa naming, merge duplicate concepts

**Agents sử dụng:** `requirement-parser-agent`, `semantic-extractor-agent`

---

### SKILL-010 · Data Pipeline Engineering
**Trạng thái:** 🔴 Cần tạo
**Đường dẫn đề xuất:** `.agent/skills/data-pipeline-expert/`

**Mô tả:** Thiết kế và vận hành các pipeline xử lý dữ liệu multi-stage — đảm bảo data integrity, idempotency, error recovery, observability cho toàn bộ luồng từ PRD input → KG → UI Schema.

**Năng lực cốt lõi:**
- Pipeline Architecture: DAG-based execution, checkpoint/restart, dead-letter queues
- Idempotency: Đảm bảo re-run không tạo ra duplicate data hoặc side effects
- Data Validation: Schema validation tại mỗi stage boundary (input/output contracts)
- Error Recovery: Retry strategies, circuit breaker, partial failure handling
- Observability: Distributed tracing (OpenTelemetry), pipeline metrics (throughput, latency per stage)
- Message Queue Integration: Kafka / RabbitMQ cho async stage communication
- Data Lineage: Tracing dữ liệu từ source (PRD line) → target (UI element)

**Agents sử dụng:** `requirement-parser-agent` → `semantic-extractor-agent` → `knowledge-graph-agent` → `ui-schema-generator-agent`

---

## 🧪 Nhóm 5: Quality Assurance

---

### SKILL-011 · UI Testing & Automation
**Trạng thái:** ✅ Đã có
**Đường dẫn:** `.agent/skills/ui-testing-expert/`

**Mô tả:** Viết test cases, test scripts Playwright/Cypress, tìm root cause lỗi UI qua 5-layer diagnosis.

**Năng lực cốt lõi:** BDD test cases, Playwright POM, selector strategy, flakiness prevention, root cause analysis

**Agents sử dụng:** `qa-pipeline-agent`

---

### SKILL-012 · Backend Testing — Golang Security & Quality
**Trạng thái:** ✅ Đã có
**Đường dẫn:** `.agent/skills/golang-testing-expert/`

**Mô tả:** Đọc code tìm lỗ hổng, viết table-driven tests, integration tests với testcontainers, fuzz testing, security vulnerability analysis.

**Năng lực cốt lõi:** Code audit checklist, security scanning (gosec, govulncheck), race detection, fuzz testing

**Agents sử dụng:** `qa-pipeline-agent`, `traceability-validator-agent`

---

## 🔒 Nhóm 6: Security Engineering

---

### SKILL-013 · Security Engineering & Hardening
**Trạng thái:** 🔴 Cần tạo
**Đường dẫn đề xuất:** `.agent/skills/security-engineer/`

**Mô tả:** Thiết kế và thực thi bảo mật toàn diện cho hệ thống — authentication, authorization, data protection, API security, và threat modeling.

**Năng lực cốt lõi:**
- Authentication & Authorization: JWT/OAuth2/OIDC flows, RBAC, API key management
- Input Validation & Sanitization: Chống SQL injection, Command injection, XSS ở mọi entry point
- Secrets Management: Vault, AWS Secrets Manager — không bao giờ hardcode secrets
- Transport Security: TLS configuration, certificate management, HSTS
- Dependency Vulnerability Scanning: `govulncheck`, `npm audit`, Dependabot integration
- Threat Modeling: STRIDE analysis cho từng service và data flow
- Security Headers: CSP, X-Frame-Options, CORS policy cho web endpoints
- Data Privacy: PII handling, data minimization, encryption at rest

**Agents sử dụng:** `qa-pipeline-agent`, `doc-consistency-agent`
**Liên kết tài liệu:** `docs/standards/security-policy.md`

---

## 🚀 Nhóm 7: DevOps & Infrastructure

---

### SKILL-014 · DevOps, CI/CD & Infrastructure
**Trạng thái:** 🔴 Cần tạo
**Đường dẫn đề xuất:** `.agent/skills/devops-expert/`

**Mô tả:** Thiết lập và vận hành CI/CD pipeline, containerization, orchestration, monitoring để đảm bảo sản phẩm được deploy nhanh, ổn định, và có thể quan sát được (observable).

**Năng lực cốt lõi:**
- Docker & Container Design: Multi-stage builds, minimal base images, health checks
- Kubernetes: Deployment, Service, ConfigMap, HPA, resource limits
- CI/CD Pipeline: GitHub Actions / GitLab CI — build, test, lint, security scan, deploy stages
- Observability Stack: Prometheus + Grafana (metrics), Loki (logs), Jaeger (traces)
- Infrastructure as Code (IaC): Terraform / Helm charts cho reproducible environments
- Environment Strategy: dev / staging / production với promotion gates
- Database Migration: Automated schema migration (golang-migrate) trong CI/CD pipeline

**Agents sử dụng:** `doc-consistency-agent` (CI trigger), `qa-pipeline-agent` (CI test stage)
**Liên kết tài liệu:** `services/*/docs/runbook.md`

---

## 📋 Nhóm 8: Quản lý & Governance

---

### SKILL-015 · Documentation Management & Governance
**Trạng thái:** ✅ Đã có
**Đường dẫn:** `.agent/skills/doc-management-expert/`

**Mô tả:** Định nghĩa và phân loại tài liệu (Product-level vs Service-level), quản lý version, đảm bảo code ↔ doc consistency, enforce governance rules.

**Năng lực cốt lõi:** Document taxonomy, ADR management, version metadata, drift detection, audit workflow

**Agents sử dụng:** `doc-consistency-agent`

---

### SKILL-016 · AI Agent Architecture & Orchestration
**Trạng thái:** ✅ Đã có
**Đường dẫn:** `.agent/skills/ai-agent-architect/`

**Mô tả:** Phân tích yêu cầu để xác định agent cần có, viết skill files chuẩn, thiết kế orchestration patterns, điều khiển agent qua tài liệu.

**Năng lực cốt lõi:** Agent decomposition, skill file authoring, Sequential/Fan-Out/Critic/Router patterns, documentation-driven control

**Agents sử dụng:** Meta-skill — dùng để quản lý toàn bộ hệ thống agent

---

## Skill Set × Agent Matrix

| Skill Set | Parser | Extractor | KG | Schema Gen | Renderer | Traceability | Doc | QA |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| SKILL-001 Software Architecture | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| SKILL-002 UI/UX Design | | | | ✓ | ✓ | | | |
| SKILL-003 API Design | | | ✓ | ✓ | | | | ✓ |
| SKILL-004 Golang Backend | ✓ | ✓ | ✓ | ✓ | | | | |
| SKILL-005 Graph DB | | | ✓ | | | ✓ | | |
| SKILL-006 React + TypeScript | | | | ✓ | ✓ | | | |
| SKILL-007 JSON Schema Rendering | | | | ✓ | ✓ | | | |
| SKILL-008 LLM Engineering | ✓ | ✓ | | | | | | |
| SKILL-009 NLP | ✓ | ✓ | | | | | | |
| SKILL-010 Data Pipeline | ✓ | ✓ | ✓ | ✓ | | | | |
| SKILL-011 UI Testing | | | | | | | | ✓ |
| SKILL-012 Backend Testing | | | | | | ✓ | | ✓ |
| SKILL-013 Security Engineering | | | | | | | ✓ | ✓ |
| SKILL-014 DevOps & CI/CD | | | | | | | ✓ | ✓ |
| SKILL-015 Doc Management | | | | | | | ✓ | |
| SKILL-016 AI Agent Architecture | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

---

## Ưu Tiên Tạo Skills Còn Thiếu

| Ưu tiên | Skill | Lý do |
|---|---|---|
| 🔴 P0 | SKILL-008 LLM Engineering | Cốt lõi của pipeline — không có LLM thì không có sản phẩm |
| 🔴 P0 | SKILL-005 Graph DB | Trái tim của Knowledge Graph layer |
| 🔴 P0 | SKILL-009 NLP | Tiền xử lý input cho LLM |
| 🟠 P1 | SKILL-001 Software Architecture | Thiết kế tổng thể — cần sớm để định hướng mọi service |
| 🟠 P1 | SKILL-010 Data Pipeline | Gắn kết các stages với nhau |
| 🟡 P2 | SKILL-003 API Design | Cần khi bắt đầu xây dựng inter-service communication |
| 🟡 P2 | SKILL-013 Security Engineering | Cần trước khi production deployment |
| 🟢 P3 | SKILL-014 DevOps & CI/CD | Cần khi chuẩn bị release |
