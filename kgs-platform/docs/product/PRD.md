# KGS Platform — Product Requirements Document (PRD)

**Version:** 1.0  
**Date:** 2026-05-07  
**Module:** `github.com/blcvn/knowledge-gateway/kgs-platform`  
**Status:** Living Document

---

## 1. Executive Summary

KGS Platform (Knowledge Graph Service Platform) là một hệ thống **multi-tenant Knowledge Graph as a Service** được xây dựng trên nền tảng Go (Kratos Framework), cung cấp khả năng quản lý đồ thị tri thức (Knowledge Graph) với các tính năng enterprise-grade bao gồm: multi-tenant isolation, ontology-driven schema validation, hybrid search (vector + text + graph centrality), overlay graph cho session-scoped writes, CQRS với transactional outbox pattern, và policy-based access control thông qua OPA (Open Policy Agent).

Hệ thống được thiết kế theo kiến trúc **layered Clean Architecture** với dependency injection (Google Wire), hỗ trợ cả giao thức HTTP và gRPC, và sử dụng polyglot persistence stack bao gồm PostgreSQL (source of truth), Neo4j (graph traversal), Qdrant (vector search), Redis (caching + locking + event streaming), và NATS (event bus).

---

## 2. Vision & Objectives

### 2.1 Product Vision

Cung cấp một nền tảng quản lý tri thức dạng đồ thị có khả năng mở rộng, bảo mật, và đa nhà thuê (multi-tenant), phục vụ làm **lớp tri thức trung tâm** (central knowledge layer) cho hệ sinh thái VNP Memory — nơi các dịch vụ AI agent, business analyst tools, và workflow automation có thể lưu trữ, truy vấn, và phân tích tri thức có cấu trúc.

### 2.2 Core Objectives

| # | Objective | Description |
|---|-----------|-------------|
| O-1 | **Multi-Tenant Isolation** | Cách ly hoàn toàn dữ liệu đồ thị giữa các ứng dụng và tenant thông qua namespace `graph/{appId}/{tenantId}` |
| O-2 | **Schema-Driven Graph** | Hỗ trợ ontology (EntityType, RelationType) với JSON Schema validation và edge constraint checking |
| O-3 | **Hybrid Search** | Kết hợp vector search (semantic), text search (BM25), và graph centrality scoring để tìm kiếm tri thức |
| O-4 | **Overlay Graph** | Session-scoped temporary graph layer cho phép tạo, chỉnh sửa, commit hoặc discard thay đổi |
| O-5 | **Transactional Consistency** | CQRS + Outbox pattern đảm bảo eventual consistency giữa PostgreSQL ↔ Neo4j ↔ Qdrant |
| O-6 | **Policy-Based Access Control** | Tích hợp OPA sidecar để evaluate Rego policies trước mỗi graph mutation |
| O-7 | **Observability** | Prometheus metrics, OpenTelemetry tracing, structured logging |
| O-8 | **API-First** | Protobuf-first API design với auto-generated HTTP/gRPC stubs và OpenAPI spec |

---

## 3. Target Users & Personas

### 3.1 Platform Administrator
- Đăng ký và quản lý ứng dụng (App Registry)
- Phát hành và thu hồi API keys
- Cấu hình quotas và rate limits
- Quản lý OPA policies

### 3.2 Application Developer
- Tích hợp KGS qua REST API hoặc gRPC
- Tạo và quản lý ontology (EntityType, RelationType)
- CRUD entities và edges trong Knowledge Graph
- Thực hiện graph traversal queries (context, impact, coverage)
- Sử dụng hybrid search API

### 3.3 AI Agent / Automated System
- Batch import entities và edges từ document processing pipelines
- Sử dụng overlay graph cho session-scoped knowledge extraction
- Trigger business rules dựa trên graph events
- Truy vấn tri thức để hỗ trợ reasoning và generation

### 3.4 Business Analyst
- Truy vấn coverage reports theo domain
- Phân tích traceability matrix giữa các entity types
- Xem cluster analysis
- Sử dụng role-based projection để xem dữ liệu phù hợp vai trò

---

## 4. Feature Set

### 4.1 App Registry & Authentication

| Feature | Description | Status |
|---------|-------------|--------|
| FR-REG-01 | Đăng ký ứng dụng mới (AppID, Name, Description, Owner) | ✅ Implemented |
| FR-REG-02 | Liệt kê và xem chi tiết ứng dụng | ✅ Implemented |
| FR-REG-03 | Phát hành API Key với scopes và TTL | ✅ Implemented |
| FR-REG-04 | Thu hồi API Key (revoke) | ✅ Implemented |
| FR-REG-05 | API Key validation (SHA-256 hash, expiry check) | ✅ Implemented |
| FR-REG-06 | Quota management (requests_per_minute) | ✅ Implemented |
| FR-REG-07 | Audit logging cho admin actions | ✅ Defined |

### 4.2 Ontology Management

| Feature | Description | Status |
|---------|-------------|--------|
| FR-ONT-01 | Tạo EntityType với JSON Schema definition | ✅ Implemented |
| FR-ONT-02 | Tạo RelationType với source/target type constraints | ✅ Implemented |
| FR-ONT-03 | Liệt kê EntityTypes và RelationTypes | ✅ Implemented |
| FR-ONT-04 | Ontology validation (entity type check, schema validation) | ✅ Implemented |
| FR-ONT-05 | Edge constraint validation (source/target type enforcement) | ✅ Implemented |
| FR-ONT-06 | Configurable strict/soft validation mode | ✅ Implemented |
| FR-ONT-07 | Ontology sync to projection engine | ✅ Implemented |

### 4.3 Graph Operations (CRUD)

| Feature | Description | Status |
|---------|-------------|--------|
| FR-GRP-01 | Create Node (entity) với auto UUID generation | ✅ Implemented |
| FR-GRP-02 | Get Node by ID | ✅ Implemented |
| FR-GRP-03 | Create Edge giữa hai nodes | ✅ Implemented |
| FR-GRP-04 | Delete Node (soft delete) với cascade edge removal | ✅ Implemented |
| FR-GRP-05 | Delete Edge (soft delete) | ✅ Implemented |
| FR-GRP-06 | Batch Delete Nodes | ✅ Implemented |
| FR-GRP-07 | Get Full Graph (paginated, max 10,000 nodes) | ✅ Implemented |
| FR-GRP-08 | Overlay-aware writes (route to overlay nếu overlay_id present) | ✅ Implemented |
| FR-GRP-09 | Distributed node-level locking (Redis) | ✅ Implemented |
| FR-GRP-10 | OPA policy check trước mỗi mutation | ✅ Implemented |

### 4.4 Graph Traversal

| Feature | Description | Status |
|---------|-------------|--------|
| FR-TRV-01 | GetContext — neighborhood traversal (depth, direction) | ✅ Implemented |
| FR-TRV-02 | GetImpact — downstream impact analysis | ✅ Implemented |
| FR-TRV-03 | GetCoverage — upstream coverage analysis | ✅ Implemented |
| FR-TRV-04 | GetSubgraph — subgraph extraction by node IDs | ✅ Implemented |
| FR-TRV-05 | Batched traversal for depth > 3 (windowed Cypher queries) | ✅ Implemented |
| FR-TRV-06 | Depth guardrail (max 10) | ✅ Implemented |
| FR-TRV-07 | Node count guardrail (max 10,000) | ✅ Implemented |

### 4.5 Hybrid Search

| Feature | Description | Status |
|---------|-------------|--------|
| FR-SRC-01 | Vector search (semantic similarity via embeddings) | ✅ Implemented |
| FR-SRC-02 | Text search (BM25/keyword matching) | ✅ Implemented |
| FR-SRC-03 | Score blending (alpha-weighted semantic + text) | ✅ Implemented |
| FR-SRC-04 | Centrality-based reranking (beta-weighted) | ✅ Implemented |
| FR-SRC-05 | Structural filtering (entity types, domains, confidence, provenance) | ✅ Implemented |
| FR-SRC-06 | Multiple embedding providers (OpenAI, AI Proxy, VNP Air, deterministic) | ✅ Implemented |
| FR-SRC-07 | Namespace-scoped search | ✅ Implemented |

### 4.6 Overlay Graph

| Feature | Description | Status |
|---------|-------------|--------|
| FR-OVL-01 | Create overlay with session binding và base version tracking | ✅ Implemented |
| FR-OVL-02 | Add entity/edge deltas to overlay | ✅ Implemented |
| FR-OVL-03 | Delete entity/edge deltas in overlay | ✅ Implemented |
| FR-OVL-04 | Commit overlay (full/partial) with conflict detection | ✅ Implemented |
| FR-OVL-05 | Discard overlay (by ID hoặc session) | ✅ Implemented |
| FR-OVL-06 | Conflict resolution policies (KEEP_OVERLAY, KEEP_BASE, MERGE, REQUIRE_MANUAL) | ✅ Implemented |
| FR-OVL-07 | NATS event publishing on commit/discard | ✅ Implemented |
| FR-OVL-08 | TTL-based auto-expiry (default 1h) | ✅ Implemented |
| FR-OVL-09 | Delta deduplication trước commit | ✅ Implemented |

### 4.7 Business Rules Engine

| Feature | Description | Status |
|---------|-------------|--------|
| FR-RUL-01 | Create/List/Get business rules | ✅ Implemented |
| FR-RUL-02 | Scheduled rules (CRON trigger via gocron) | ✅ Implemented |
| FR-RUL-03 | Event-driven rules (ON_WRITE trigger via Redis Streams) | ✅ Implemented |
| FR-RUL-04 | Cypher query execution cho rule logic | ✅ Implemented |
| FR-RUL-05 | Action dispatch (webhook, notification) | ✅ Partial |
| FR-RUL-06 | Rule execution tracking | ✅ Defined |

### 4.8 Access Control (OPA)

| Feature | Description | Status |
|---------|-------------|--------|
| FR-ACL-01 | Create/List/Get OPA policies | ✅ Implemented |
| FR-ACL-02 | Rego policy evaluation via OPA sidecar | ✅ Implemented |
| FR-ACL-03 | Policy sync (DB → OPA via PUT /v1/policies) | ✅ Implemented |
| FR-ACL-04 | Fail-closed security model (OPA unreachable → deny) | ✅ Implemented |

### 4.9 Analytics

| Feature | Description | Status |
|---------|-------------|--------|
| FR-ANL-01 | Coverage Report (domain-level entity coverage analysis) | ✅ Implemented |
| FR-ANL-02 | Traceability Matrix (source → target path analysis) | ✅ Implemented |
| FR-ANL-03 | Cluster Analysis (community detection) | ✅ Implemented |
| FR-ANL-04 | Analytics caching (TTL-based) | ✅ Implemented |

### 4.10 Projection & View Resolution

| Feature | Description | Status |
|---------|-------------|--------|
| FR-PRJ-01 | Role-based entity/edge filtering | ✅ Implemented |
| FR-PRJ-02 | PII property masking | ✅ Implemented |
| FR-PRJ-03 | Confidence threshold filtering | ✅ Implemented |
| FR-PRJ-04 | Ontology sync to projection rules | ✅ Implemented |

---

## 5. Architecture Overview

### 5.1 Layered Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                     API Layer (Protobuf)                     │
│  HTTP (8000) + gRPC (9000) + KG Namespace HTTP (/kg/*)      │
├──────────────────────────────────────────────────────────────┤
│                    Middleware Stack                           │
│  Tracing → AccessLog → Metrics → Recovery → Auth → Namespace │
│                     → RateLimiter                            │
├──────────────────────────────────────────────────────────────┤
│                     Service Layer                            │
│  GraphService · OntologyService · RegistryService            │
│  RulesService · PolicyService · HealthService                │
├──────────────────────────────────────────────────────────────┤
│                    Business Logic (biz)                       │
│  GraphUsecase · OntologyValidator · QueryPlanner             │
│  RegistryUsecase · RulesUsecase · PolicyUsecase              │
│  OPAClient · ViewResolver · EventRunner                      │
├──────────────────────────────────────────────────────────────┤
│                  Cross-Cutting Modules                        │
│  Search Engine · Analytics Engine · Overlay Manager           │
│  Outbox Worker · Projection Engine · Lock Manager             │
│  Batch Handler · Observability                               │
├──────────────────────────────────────────────────────────────┤
│                     Data Layer                               │
│  PostgreSQL (GORM) · Neo4j Driver · Qdrant Client            │
│  Redis Client · NATS Client                                  │
└──────────────────────────────────────────────────────────────┘
```

### 5.2 Data Flow — Write Path (CQRS + Outbox)

```
Client Request
    ↓
API Layer (Auth + Namespace middleware)
    ↓
OPA Policy Check (allow/deny)
    ↓
Ontology Validation (entity type + schema + edge constraints)
    ↓
Distributed Lock Acquisition (Redis)
    ↓
PostgreSQL Write (Entity/Edge + Outbox record — single TX)
    ↓
Redis Event (kgs:events:nodes stream)
    ↓
Outbox Worker (polls pending records)
    ├── Neo4j Sync (upsert entity/edge for graph traversal)
    └── Qdrant Sync (upsert vector for semantic search)
```

---

## 6. Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Language | Go 1.25 | Core implementation |
| Framework | Kratos v2 | HTTP/gRPC server, middleware, DI |
| DI | Google Wire | Compile-time dependency injection |
| ORM | GORM (PostgreSQL) | Relational data persistence |
| Graph DB | Neo4j v5 | Graph traversal (Cypher queries) |
| Vector DB | Qdrant | Semantic vector search |
| Cache/Lock | Redis | Distributed locking, rate limiting, event streams, overlay store |
| Event Bus | NATS JetStream | Overlay events, async messaging |
| Policy Engine | OPA (Open Policy Agent) | Access control policy evaluation |
| Scheduler | gocron v2 | Scheduled rule execution |
| Observability | Prometheus + OpenTelemetry | Metrics + distributed tracing |
| API Definition | Protobuf → gRPC + HTTP + OpenAPI | Code generation |
| Container | Docker (Debian Slim) | Production deployment |

---

## 7. Non-Functional Requirements

| Category | Requirement |
|----------|-------------|
| **Performance** | Write latency < 100ms (P95) cho single entity upsert |
| **Throughput** | Hỗ trợ 1000+ concurrent requests per minute per app (configurable) |
| **Availability** | Health checks (/healthz, /readyz) cho liveness/readiness probes |
| **Scalability** | Horizontal scaling qua stateless app servers + external state stores |
| **Security** | API Key auth, namespace isolation, OPA policy enforcement, fail-closed |
| **Consistency** | Eventual consistency (PG → Neo4j/Qdrant) via transactional outbox |
| **Observability** | Prometheus metrics, OpenTelemetry traces, structured access logs |
| **Resilience** | Outbox retry (max 10 attempts), lock TTL, graceful degradation |
| **Data Isolation** | Strict per-namespace (app+tenant) data isolation |
| **Guardrails** | Max traversal depth = 10, max nodes = 10,000, batch size = 100 |

---

## 8. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Graph write success rate | > 99.9% | `kgs_entity_write_total{status="success"}` |
| Outbox sync lag | < 5s (P95) | `kgs_outbox_lag_seconds` |
| Outbox pending queue | < 100 records | `kgs_outbox_pending` |
| Search latency | < 500ms (P95) | `kgs_search_duration_seconds` |
| API error rate | < 0.1% | HTTP 5xx responses / total |
| Overlay commit success | > 99% | Overlay operations metrics |

---

## 9. Roadmap & Future Enhancements

| Phase | Feature | Priority |
|-------|---------|----------|
| Next | Full JSON Schema validation (Phase 5 placeholder) | High |
| Next | Version GC + Compaction (CronJob) | High |
| Next | SurrealDB unified adapter (dual-stack mode) | Medium |
| Future | GraphQL API endpoint | Medium |
| Future | Real-time WebSocket subscriptions for graph changes | Low |
| Future | Multi-region deployment support | Low |
| Future | Webhook action execution for business rules | Medium |
