# KGS Platform — User Requirements Document (URD)

**Version:** 1.0  
**Date:** 2026-05-07  
**Module:** `github.com/blcvn/knowledge-gateway/kgs-platform`  
**Status:** Living Document

---

## 1. Giới thiệu

### 1.1 Mục đích tài liệu

Tài liệu này mô tả các yêu cầu từ phía người dùng (User Requirements) đối với hệ thống KGS Platform — nền tảng Knowledge Graph as a Service. Tài liệu tập trung vào các use cases, kịch bản sử dụng, và kỳ vọng của từng nhóm người dùng khi tương tác với hệ thống.

### 1.2 Phạm vi hệ thống

KGS Platform cung cấp dịch vụ quản lý đồ thị tri thức (Knowledge Graph) cho nhiều ứng dụng và tenant đồng thời, bao gồm:

- **Quản lý ứng dụng và xác thực** (App Registry, API Key)
- **Quản lý schema tri thức** (Ontology — EntityType, RelationType)
- **Thao tác đồ thị** (CRUD nodes, edges, traversal, subgraph)
- **Tìm kiếm tri thức** (Hybrid Search — vector + text + graph centrality)
- **Phiên làm việc tạm thời** (Overlay Graph — create, commit, discard)
- **Quy tắc nghiệp vụ** (Business Rules — scheduled + event-driven)
- **Kiểm soát truy cập** (OPA Policy-based access control)
- **Phân tích đồ thị** (Analytics — coverage, traceability, clustering)

### 1.3 Định nghĩa thuật ngữ

| Thuật ngữ | Định nghĩa |
|-----------|-------------|
| **Knowledge Graph** | Đồ thị tri thức gồm các node (entity) và edge (relation) biểu diễn mối quan hệ giữa các khái niệm |
| **Namespace** | Không gian tên dạng `graph/{appId}/{tenantId}` dùng để cách ly dữ liệu giữa các ứng dụng/tenant |
| **Entity (Node)** | Một thực thể trong đồ thị, có label (entity type), properties, và metadata |
| **Edge (Relation)** | Quan hệ có hướng giữa hai entity, có relation type và properties |
| **Ontology** | Lược đồ (schema) định nghĩa các loại entity và relation hợp lệ trong đồ thị |
| **Overlay** | Lớp đồ thị tạm thời liên kết với session, cho phép chỉnh sửa trước khi commit vào đồ thị chính |
| **OPA** | Open Policy Agent — engine đánh giá chính sách truy cập dựa trên ngôn ngữ Rego |
| **Outbox Pattern** | Pattern đảm bảo eventual consistency bằng cách ghi sự kiện vào bảng outbox trong cùng transaction |

---

## 2. Nhóm người dùng (User Personas)

### 2.1 Platform Administrator (Admin)

**Mô tả:** Người quản trị hệ thống KGS Platform, chịu trách nhiệm đăng ký ứng dụng, quản lý API keys, cấu hình policies, và giám sát hệ thống.

**Đặc điểm:**
- Hiểu biết về kiến trúc hệ thống và bảo mật
- Truy cập trực tiếp qua API hoặc admin tools
- Cần audit trail cho mọi thao tác quản trị

**Kỳ vọng:**
- Quản lý ứng dụng đơn giản, có khả năng tạm dừng hoặc kích hoạt lại
- Phát hành API keys với phạm vi truy cập và thời hạn tùy chỉnh
- Thu hồi API keys ngay lập tức khi phát hiện vi phạm bảo mật
- Cấu hình rate limits theo từng ứng dụng
- Định nghĩa và triển khai OPA policies cho access control

---

### 2.2 Application Developer (Developer)

**Mô tả:** Lập trình viên tích hợp KGS Platform vào ứng dụng của mình thông qua REST API hoặc gRPC.

**Đặc điểm:**
- Sử dụng API documentation (OpenAPI spec) để tích hợp
- Cần SDK-friendly API design
- Quan tâm đến performance và error handling

**Kỳ vọng:**
- API rõ ràng, nhất quán, có documentation đầy đủ
- Tạo và quản lý ontology phù hợp domain của ứng dụng
- CRUD operations trên entities và edges với response time thấp
- Graph traversal queries linh hoạt (context, impact, coverage, subgraph)
- Hybrid search với khả năng cấu hình blend ratio và filters
- Error messages rõ ràng, actionable

---

### 2.3 AI Agent / Automated System (Agent)

**Mô tả:** Hệ thống AI tự động (như BA Agent, document extraction pipelines) tương tác với KGS Platform để lưu trữ và truy vấn tri thức.

**Đặc điểm:**
- Batch operations với volume lớn
- Cần overlay graph cho session-scoped knowledge building
- Trigger-based rule execution
- Yêu cầu consistency giữa extracted và generated knowledge

**Kỳ vọng:**
- Import hàng loạt entities/edges từ pipelines xử lý tài liệu
- Sử dụng overlay graph để tạo tri thức tạm thời, sau đó commit
- Tự động trigger business rules khi có thay đổi trên graph
- Phân biệt provenance type (EXTRACTED, GENERATED, MANUAL)
- Version tracking cho mỗi batch import

---

### 2.4 Business Analyst (BA)

**Mô tả:** Nhà phân tích nghiệp vụ sử dụng KGS để xem và phân tích tri thức theo góc nhìn nghiệp vụ.

**Đặc điểm:**
- Không cần kỹ năng lập trình sâu
- Quan tâm đến data quality và coverage
- Cần role-based view phù hợp

**Kỳ vọng:**
- Xem coverage reports theo domain để biết mức độ đầy đủ của tri thức
- Phân tích traceability matrix giữa requirements và user stories
- Cluster analysis để phát hiện nhóm tri thức liên quan
- Role-based projection chỉ hiện thông tin phù hợp với vai trò BA

---

## 3. Use Cases

### UC-01: Đăng ký ứng dụng và phát hành API Key

**Actor:** Platform Administrator  
**Precondition:** Admin có quyền truy cập quản trị  
**Main Flow:**

1. Admin gọi `POST /v1/apps` với thông tin ứng dụng (appName, description, owner)
2. Hệ thống tạo AppID (UUID), lưu vào database, trả về AppID và status
3. Admin gọi `POST /v1/apps/{appId}/keys` với name, scopes, TTL
4. Hệ thống tạo API key (48-byte random + prefix "kgs_ak_")
5. Hệ thống hash API key (SHA-256), lưu hash vào database
6. Hệ thống trả về raw API key (chỉ hiển thị 1 lần), key hash, và key prefix

**Postcondition:** Ứng dụng được đăng ký và có API key để xác thực

**Exception Flows:**
- E1: AppName trùng lặp → trả về lỗi conflict
- E2: AppID không tồn tại khi issue key → trả về lỗi not found

---

### UC-02: Định nghĩa Ontology

**Actor:** Application Developer  
**Precondition:** Ứng dụng đã đăng ký, Developer có API key hợp lệ  
**Main Flow:**

1. Developer gọi `POST /v1/ontology/entities` với EntityType definition (name, description, JSON Schema)
2. Hệ thống validate JSON Schema syntax
3. Hệ thống lưu EntityType vào database (kgs_entity_types)
4. Developer gọi `POST /v1/ontology/relations` với RelationType definition (name, description, sourceTypes, targetTypes, propertiesSchema)
5. Hệ thống lưu RelationType vào database (kgs_relation_types)
6. Hệ thống sync ontology đến projection engine (nếu sync_projection enabled)

**Postcondition:** Ontology schema được cập nhật, các entity/edge writes sau đó sẽ được validate theo schema

---

### UC-03: Tạo Entity và Edge

**Actor:** Application Developer / AI Agent  
**Precondition:** Namespace hợp lệ, Ontology đã định nghĩa (optional)  
**Main Flow:**

1. Client gửi `POST /v1/graph/nodes` với label và properties (JSON)
2. Middleware stack xử lý: Auth → Namespace extraction → Rate limiting
3. GraphUsecase thực hiện:
   a. Acquire distributed lock cho node ID (Redis)
   b. Evaluate OPA policy (appID, action="CREATE_NODE", resource=label)
   c. Validate entity theo ontology (nếu enabled)
   d. Write vào PostgreSQL (entity + outbox record trong cùng transaction)
   e. Publish event lên Redis Stream
4. Outbox worker async sync đến Neo4j và Qdrant
5. Client nhận response với nodeId, label, properties

**Alternative Flow (Overlay Mode):**
- A1: Nếu properties chứa `overlay_id`, entity được ghi vào overlay thay vì base graph
- A2: Không cần distributed lock, không cần OPA check (overlay là session-scoped)

---

### UC-04: Graph Traversal — Impact Analysis

**Actor:** Application Developer / Business Analyst  
**Precondition:** Graph có dữ liệu, entities đã được index trong Neo4j  
**Main Flow:**

1. Client gửi `GET /v1/graph/nodes/{nodeId}/impact?maxDepth=3`
2. Hệ thống validate depth (max 10)
3. QueryPlanner sinh Cypher query:
   ```cypher
   MATCH p=(n {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})-[*1..3]->(m {app_id: $app_id, tenant_id: $tenant_id})
   RETURN nodes(p) AS nodes, relationships(p) AS rels
   ```
4. Hệ thống execute query trên Neo4j
5. Client nhận response với danh sách nodes và edges downstream

**Alternative Flow (Deep Traversal):**
- A1: Nếu maxDepth > 3, hệ thống chia thành nhiều batched queries (window size = 3)
- A2: Kết quả được merge từ nhiều depth windows

---

### UC-05: Hybrid Search

**Actor:** Application Developer / AI Agent  
**Precondition:** Entities đã được index trong Qdrant (vector) và text index  
**Main Flow:**

1. Client gọi hybrid search API với query text và options (topK, alpha, entityTypes, domains)
2. Search Engine thực hiện song song:
   a. Vector search (Qdrant) → semantic results với cosine similarity scores
   b. Text search (BM25) → text matching results
3. Score blending: `finalScore = alpha * semanticScore + (1-alpha) * textScore`
4. Centrality reranking: `finalScore = (1-beta) * blendedScore + beta * centralityScore`
5. Apply filters (entity types, domains, min confidence, provenance types)
6. Sort by final score, truncate to topK
7. Client nhận ranked results

---

### UC-06: Overlay Graph Lifecycle

**Actor:** AI Agent  
**Precondition:** Session ID hợp lệ, namespace tồn tại  
**Main Flow:**

1. Agent gọi API tạo overlay với namespace, sessionID, baseVersionID
2. Hệ thống tạo OverlayGraph trong Redis (TTL = 1h), bind session
3. Agent ghi entities/edges vào overlay (multiple writes)
4. Agent gọi commit API với conflict policy
5. Hệ thống:
   a. Detect conflicts giữa baseVersion và currentVersion
   b. Resolve conflicts theo policy (KEEP_OVERLAY, KEEP_BASE, MERGE, REQUIRE_MANUAL)
   c. Deduplicate entity/edge deltas
   d. Write vào PostgreSQL + outbox (single transaction)
   e. Create new version delta
   f. Cleanup overlay từ Redis
   g. Publish commit event qua NATS

**Alternative Flows:**
- A1: Agent gọi discard → overlay bị xóa khỏi Redis, publish discard event
- A2: Session timeout → overlay tự auto-expire theo TTL
- A3: Commit partial → overlay status = PARTIAL, cho phép tiếp tục writes

---

### UC-07: Business Rules Execution

**Actor:** Platform Administrator (config) / System (execution)  
**Precondition:** Rules đã được tạo và activated  
**Main Flow (Scheduled):**

1. Admin tạo rule với triggerType="SCHEDULED", cron expression, và Cypher query
2. gocron scheduler tự động trigger rule theo cron schedule
3. RuleRunner execute Cypher query trên Neo4j
4. Kết quả được log, action (webhook/notification) được dispatch

**Main Flow (Event-Driven):**

1. Admin tạo rule với triggerType="ON_WRITE"
2. EventRunner lắng nghe Redis Stream `kgs:events:nodes`
3. Khi có event (node.created), EventRunner:
   a. List active ON_WRITE rules cho appID
   b. Execute mỗi rule's Cypher query với event payload
   c. Dispatch action theo rule configuration

---

### UC-08: Analytics — Coverage Report

**Actor:** Business Analyst  
**Precondition:** Graph có dữ liệu trong namespace  
**Main Flow:**

1. BA gọi coverage report API với namespace và domain
2. Analytics Engine execute Cypher query đếm entities theo type trong domain
3. So sánh với required entity types trong ontology
4. Tính coverage percentage = (found types / required types) × 100%
5. Client nhận report gồm: coverage %, by-type breakdown, uncovered types

---

### UC-09: Access Control Policy Management

**Actor:** Platform Administrator  
**Precondition:** OPA sidecar đang chạy  
**Main Flow:**

1. Admin gọi `POST /v1/policies` với Rego policy content
2. Hệ thống lưu policy vào database (kgs_policies)
3. PolicySync tự động upload Rego content lên OPA via `PUT /v1/policies/{id}`
4. Khi có graph mutation request:
   a. OPAClient gửi input (app_id, action, resource) đến OPA
   b. OPA evaluate Rego policy, trả về allow/deny
   c. Nếu deny → request bị reject với 403 Forbidden

---

## 4. Yêu cầu về giao diện (Interface Requirements)

### 4.1 API Interfaces

| Giao thức | Port | Base Path | Mô tả |
|-----------|------|-----------|-------|
| HTTP/REST | 8000 | `/v1/` | API chính cho tất cả operations |
| gRPC | 9000 | — | Protobuf-based RPC cho high-performance clients |
| HTTP/REST | 8000 | `/kg/` | KG Namespace HTTP endpoints (extended graph operations) |
| HTTP | 8000 | `/metrics` | Prometheus metrics endpoint |
| HTTP | 8000 | `/healthz` | Liveness probe |
| HTTP | 8000 | `/readyz` | Readiness probe |

### 4.2 Authentication Flow

```
Client Request
    ↓
[X-API-Key header] hoặc [Bearer token]
    ↓
Auth Middleware
    ├── Hash API key (SHA-256)
    ├── Lookup in Redis cache (TTL cache)
    ├── Fallback to PostgreSQL lookup
    ├── Validate: not revoked, not expired
    └── Extract AppID → inject vào context
    ↓
Namespace Middleware
    ├── Extract X-KG-Namespace hoặc từ URL path
    └── Validate namespace matches authenticated AppID
    ↓
RateLimiter Middleware
    ├── Lookup quota cho AppID
    └── Redis-based sliding window counter
```

### 4.3 Error Response Format

```json
{
  "code": 403,
  "reason": "FORBIDDEN",
  "message": "access denied by OPA policy",
  "metadata": {
    "action": "CREATE_NODE",
    "label": "Customer"
  }
}
```

---

## 5. Yêu cầu về dữ liệu (Data Requirements)

### 5.1 Data Models

| Model | Storage | Primary Key | Tenant Isolation |
|-------|---------|-------------|------------------|
| App | PostgreSQL (kgs_apps) | AppID (UUID) | Global |
| APIKey | PostgreSQL (kgs_api_keys) | KeyHash (SHA-256) | Per-App |
| Quota | PostgreSQL (kgs_quotas) | ID (auto) | Per-App |
| AuditLog | PostgreSQL (kgs_audit_logs) | ID (auto) | Per-App |
| EntityType | PostgreSQL (kgs_entity_types) | ID (auto) | Per-App + Per-Tenant |
| RelationType | PostgreSQL (kgs_relation_types) | ID (auto) | Per-App + Per-Tenant |
| Entity (KGEntity) | PostgreSQL (kgs_entities) | EntityID (UUID) | Per-App + Per-Tenant |
| Edge (KGEdge) | PostgreSQL (kgs_edges) | EdgeID (UUID) | Per-App + Per-Tenant |
| Entity (Graph) | Neo4j | id property | Per-Namespace |
| Entity (Vector) | Qdrant | Point ID | Per-Namespace (collection) |
| Rule | PostgreSQL (kgs_rules) | ID (auto) | Per-App + Per-Tenant |
| RuleExecution | PostgreSQL (kgs_rule_executions) | ID (auto) | Per-App + Per-Tenant |
| Policy | PostgreSQL (kgs_policies) | ID (auto) | Per-App + Per-Tenant |
| SyncOutbox | PostgreSQL (kgs_sync_outbox) | ID (auto) | Per-App + Per-Tenant |
| Overlay | Redis (TTL) | OverlayID (UUID) | Per-Namespace |

### 5.2 Data Retention

| Data Category | Retention Policy |
|---------------|-----------------|
| Entities & Edges | Soft delete (is_deleted flag), permanent delete via GC |
| Outbox Records | Synced records purged after processing |
| Overlay Graphs | TTL-based auto-expiry (default 1h) |
| Audit Logs | Retain indefinitely |
| Rule Executions | Configurable retention |
| API Keys | Retain until explicit revocation |

---

## 6. Yêu cầu về hiệu năng (Performance Requirements)

| Metric | Yêu cầu | Ghi chú |
|--------|---------|---------|
| Entity write latency | < 100ms (P95) | Single entity upsert, including lock acquire |
| Graph traversal (depth ≤ 3) | < 200ms | Direct Cypher query trên Neo4j |
| Graph traversal (depth > 3) | < 1s | Batched traversal queries |
| Hybrid search | < 500ms (P95) | Vector + text + reranking |
| Overlay commit | < 2s | Depends on delta size |
| API throughput | ≥ 1000 req/min/app | Configurable per-app quota |
| Outbox sync latency | < 5s | PG → Neo4j/Qdrant propagation |

---

## 7. Yêu cầu về bảo mật (Security Requirements)

| # | Yêu cầu | Mô tả |
|---|---------|-------|
| SEC-01 | **API Key Authentication** | Mọi API request phải có API key hợp lệ (not revoked, not expired) |
| SEC-02 | **Namespace Isolation** | Mỗi app chỉ truy cập được namespace của mình |
| SEC-03 | **OPA Policy Enforcement** | Graph mutations được kiểm tra bởi OPA policies |
| SEC-04 | **Fail-Closed Model** | Nếu OPA không khả dụng, request bị deny |
| SEC-05 | **Key Hashing** | API keys được hash bằng SHA-256 trước khi lưu trữ |
| SEC-06 | **Rate Limiting** | Redis-based sliding window rate limiter per-app |
| SEC-07 | **PII Masking** | Projection engine mask các properties nhạy cảm theo role |
| SEC-08 | **Audit Trail** | Mọi admin actions được ghi audit log |
| SEC-09 | **Soft Delete** | Entities/edges bị soft delete, không xóa vĩnh viễn ngay lập tức |

---

## 8. Yêu cầu về triển khai (Deployment Requirements)

### 8.1 Container Requirements

```yaml
# Docker image: Debian Slim base
# Exposed ports: 8000 (HTTP), 9000 (gRPC)
# Configuration volume: /data/conf/config.yaml
# Entry point: ./server -conf /data/conf/config.yaml
```

### 8.2 Infrastructure Dependencies

| Service | Purpose | Min Version |
|---------|---------|-------------|
| PostgreSQL | Primary data store | 14+ |
| Neo4j | Graph traversal engine | 5.x |
| Qdrant | Vector search engine | 1.x |
| Redis | Cache, lock, events, overlay store | 7+ |
| NATS | Event bus (JetStream) | 2.12+ |
| OPA | Policy evaluation sidecar | 0.60+ |

### 8.3 Health Monitoring

| Endpoint | Type | Response |
|----------|------|----------|
| `/healthz` | Liveness | `{"status":"ok"}` if server is running |
| `/readyz` | Readiness | `{"status":"ok"}` if all dependencies connected |
| `/metrics` | Prometheus | Full metrics scrape endpoint |

---

## 9. Constraints & Assumptions

### 9.1 Constraints

- Hệ thống yêu cầu tất cả infrastructure dependencies (PG, Neo4j, Qdrant, Redis, NATS, OPA) phải available
- Cross-namespace queries bị cấm hoàn toàn
- Maximum traversal depth = 10 (hard limit)
- Maximum nodes per subgraph query = 10,000
- Outbox worker chạy single-instance per deployment (avoid duplicate processing)
- Overlay TTL default 1 hour, không persist qua restart

### 9.2 Assumptions

- Mỗi entity có unique ID (UUID) trong scope namespace
- API keys được bảo mật phía client (không lưu plaintext)
- OPA sidecar chạy cùng pod với KGS server (localhost:8181)
- NATS JetStream đã được configure với stream "kgs-events"
- Neo4j database "kgs" đã được tạo sẵn
- Qdrant collection đã được tạo sẵn với vector size phù hợp embedding model
