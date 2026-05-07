# Software Requirements Specification (SRS)
## KGS Platform — Knowledge Graph Service

| Trường | Giá trị |
|---|---|
| **Tài liệu** | SRS — Software Requirements Specification |
| **Dự án** | VNP Design Platform |
| **Service** | `kgs-platform` |
| **Module Go** | `kgs-platform` |
| **Phiên bản** | 1.0.0 |
| **Ngày tạo** | 2026-04-08 |
| **Trạng thái** | Draft — Derived from source code |

---

## Mục lục

1. [Giới thiệu](#1-giới-thiệu)
2. [Phạm vi hệ thống](#2-phạm-vi-hệ-thống)
3. [Các bên liên quan (Stakeholders)](#3-các-bên-liên-quan-stakeholders)
4. [Kiến trúc tổng quan](#4-kiến-trúc-tổng-quan)
5. [Yêu cầu chức năng](#5-yêu-cầu-chức-năng)
   - 5.1 [FR-REG — Registry & App Management](#51-fr-reg--registry--app-management)
   - 5.2 [FR-ONT — Ontology Management](#52-fr-ont--ontology-management)
   - 5.3 [FR-GRAPH — Knowledge Graph CRUD](#53-fr-graph--knowledge-graph-crud)
   - 5.4 [FR-QUERY — Graph Query & Traversal](#54-fr-query--graph-query--traversal)
   - 5.5 [FR-RULES — Business Rules Engine](#55-fr-rules--business-rules-engine)
   - 5.6 [FR-POLICY — Access Control & Policy Management](#56-fr-policy--access-control--policy-management)
   - 5.7 [FR-EVENT — Event-Driven Processing](#57-fr-event--event-driven-processing)
   - 5.8 [FR-KAFKA — Document Ingestion via Kafka](#58-fr-kafka--document-ingestion-via-kafka)
   - 5.9 [FR-VECTOR — Vector Semantic Search](#59-fr-vector--vector-semantic-search)
6. [Yêu cầu phi chức năng](#6-yêu-cầu-phi-chức-năng)
7. [Mô hình dữ liệu](#7-mô-hình-dữ-liệu)
   - 7.1 [PostgreSQL — State Store](#71-postgresql--state-store)
   - 7.2 [Neo4j — Graph Store](#72-neo4j--graph-store)
   - 7.3 [Redis — Stream & Cache](#73-redis--stream--cache)
8. [API Specification](#8-api-specification)
9. [Ràng buộc hệ thống (Guardrails)](#9-ràng-buộc-hệ-thống-guardrails)
10. [Luồng tích hợp với hệ thống ngoài](#10-luồng-tích-hợp-với-hệ-thống-ngoài)
11. [Yêu cầu bảo mật](#11-yêu-cầu-bảo-mật)
12. [Yêu cầu vận hành](#12-yêu-cầu-vận-hành)
13. [Giả định và phụ thuộc](#13-giả-định-và-phụ-thuộc)

---

## 1. Giới thiệu

### 1.1 Mục đích tài liệu

Tài liệu này mô tả đầy đủ các yêu cầu phần mềm (cả chức năng và phi chức năng) của service **kgs-platform** — Knowledge Graph Service thuộc VNP Design Platform. Tài liệu được sinh ra từ phân tích mã nguồn thực tế, phục vụ cho:

- Các nhà phát triển triển khai và bảo trì service
- Các team upstream cần tích hợp với kgs-platform
- QA/Testing để xây dựng test cases
- DevOps để cấu hình vận hành

### 1.2 Định nghĩa thuật ngữ

| Thuật ngữ | Định nghĩa |
|---|---|
| **KGS** | Knowledge Graph Service — tên ngắn của kgs-platform |
| **App / AppID** | Client application đăng ký sử dụng KGS; mỗi app có namespace riêng trong graph |
| **Ontology** | Bộ định nghĩa kiểu node (EntityType) và kiểu quan hệ (RelationType) |
| **Node** | Đỉnh trong Knowledge Graph, có label và properties |
| **Edge** | Cạnh (relationship) trong Knowledge Graph, có type và properties |
| **OPA** | Open Policy Agent — policy engine dạng sidecar |
| **Rego** | Ngôn ngữ khai báo dùng để viết policy cho OPA |
| **Cypher** | Ngôn ngữ truy vấn của Neo4j |
| **Redis Stream** | Cấu trúc append-only log trong Redis, dùng như message queue nội bộ |
| **app_id** | Namespace định danh app, được gắn vào mọi node/edge trong Neo4j |

### 1.3 Phạm vi tài liệu

Tài liệu này **bao gồm**:
- Tất cả API endpoints (HTTP + gRPC)
- Background workers (RuleRunner, EventRunner, PolicySyncRunner, KafkaConsumer)
- Mô hình dữ liệu PostgreSQL, Neo4j, Redis
- Yêu cầu tích hợp với OPA, Qdrant, Kafka

Tài liệu này **không bao gồm**:
- Chi tiết implement của các service upstream (pipeline-shim, ai-orchestrator)
- Cấu hình deployment/infrastructure (xem technical-design.md)

---

## 2. Phạm vi hệ thống

**kgs-platform** có vai trò là **Knowledge Graph Gateway** trung tâm của VNP Design Platform:

```
                   ┌──────────────────────────────────────────────────┐
                   │                 kgs-platform                     │
                   │                                                  │
  upstream         │   ┌──────────┐   ┌──────────┐   ┌────────────┐ │
  services  ──────▶│   │ HTTP API │   │ gRPC API │   │  Workers   │ │
                   │   │  :8000   │   │  :9000   │   │(background)│ │
  ai-orchestrator  │   └────┬─────┘   └────┬─────┘   └─────┬──────┘ │
  ─────────Kafka──▶│        └──────┬────────┘               │        │
                   │         ┌─────▼──────────────────────  │        │
                   │         │        Business Logic         │        │
                   │         │  (GraphUsecase, RulesUsecase  │        │
                   │         │   PolicyUsecase, OPAClient)   │        │
                   │         └───────────────┬───────────────┘        │
                   │                   ┌─────▼─────────────────────┐  │
                   │                   │         Data Layer         │  │
                   │                   │  PostgreSQL │ Neo4j        │  │
                   │                   │  Redis     │ Qdrant        │  │
                   │                   └────────────────────────────┘  │
                   │                                    │               │
                   │                              ┌─────▼──────┐        │
                   │                              │    OPA     │        │
                   │                              │ (sidecar)  │        │
                   │                              └────────────┘        │
                   └──────────────────────────────────────────────────┘
```

**Nhiệm vụ cốt lõi:**
1. Lưu trữ và quản lý Knowledge Graph đa tenant (app_id-namespaced)
2. Cung cấp API truy vấn đồ thị (context, impact, coverage, subgraph)
3. Quản lý ontology (schema cho node/edge types)
4. Tự động hóa business rules trên đồ thị (SCHEDULED + ON_WRITE)
5. Kiểm soát truy cập bằng OPA Rego policies
6. Tiếp nhận document events từ Kafka để populate graph

---

## 3. Các bên liên quan (Stakeholders)

| Stakeholder | Vai trò | Tương tác với KGS |
|---|---|---|
| **pipeline-shim** | AI pipeline service | Gọi GraphService qua gRPC/HTTP để lấy context cho LLM |
| **ai-orchestrator** | Document processing service | Publish events `document.ingested` lên Kafka |
| **service-kg-to-preview** | UI preview generator | Query graph để render UI components |
| **Frontend / API Gateway** | VNP Design Platform UI | Gọi RegistryService (CRUD App/APIKey), GraphService (query) |
| **Platform Admin** | Quản trị viên hệ thống | Quản lý Ontology, Rules, Policies |
| **DevOps / Infra** | Vận hành hệ thống | Deploy, monitor, configure |

---

## 4. Kiến trúc tổng quan

### 4.1 Stack công nghệ

| Thành phần | Công nghệ | Phiên bản |
|---|---|---|
| Language | Go | 1.24.0 |
| Framework | go-kratos/kratos | v2.9.2 |
| HTTP Transport | Kratos HTTP | :8000 |
| gRPC Transport | Kratos gRPC | :9000 |
| DI / Wiring | Google Wire | v0.7.0 |
| Graph DB | Neo4j | bolt:// + neo4j-go-driver v5.28.4 |
| Relational DB | PostgreSQL | pgx v5 + GORM v1.31.1 |
| Cache / Stream | Redis | go-redis v9.18.0 |
| Vector DB | Qdrant | HTTP REST client tự implement |
| Message Bus | Apache Kafka | kafka-go v0.4.50 |
| Policy Engine | Open Policy Agent | HTTP sidecar |
| JSON Validation | gojsonschema | v1.2.0 |
| Scheduler | gocron | v2.19.1 |
| Observability | OpenTelemetry | v1.39.0 |

### 4.2 Layered Architecture

```
cmd/server/
├── main.go              # Entrypoint — wire inject, app bootstrap
│
internal/
├── conf/                # Protobuf config schema
├── server/
│   ├── http.go          # HTTP server — đăng ký tất cả services
│   ├── grpc.go          # gRPC server — đăng ký tất cả services
│   ├── worker.go        # WorkerServer — background jobs
│   └── middleware/      # Auth, RateLimiter middleware
├── service/             # Transport layer — map proto ↔ biz
│   ├── graph.go
│   ├── registry.go
│   ├── ontology.go
│   ├── rules.go
│   ├── policy.go
│   └── greeter.go
├── biz/                 # Business logic layer
│   ├── graph.go         # GraphUsecase
│   ├── rules.go         # RulesUsecase, Rule, Policy models
│   ├── ontology.go      # EntityType, RelationType models
│   ├── registry.go      # App, APIKey, Quota, AuditLog models
│   ├── opa_client.go    # OPA HTTP client
│   ├── query_planner.go # Cypher query builder
│   ├── graph_guardrails.go # Depth/node count limits
│   ├── validator.go     # JSON Schema validator
│   ├── rule_runner.go   # Scheduled rule executor
│   ├── event_runner.go  # Redis Stream consumer
│   └── policy_sync.go   # OPA policy sync runner
├── data/                # Data access layer (Repository impl)
│   ├── data.go          # Data struct — DB connections
│   ├── graph_node.go    # Neo4j node CRUD
│   ├── graph_edge.go    # Neo4j edge CRUD
│   ├── graph_query.go   # Neo4j query executor
│   ├── ontology.go      # Postgres ontology repo
│   ├── registry.go      # Postgres registry repo
│   ├── rules.go         # Postgres rules repo
│   ├── policy.go        # Postgres policy repo
│   ├── qdrant.go        # Qdrant HTTP client
│   └── seed.go          # Ontology seed data (19 node types, 16 edge types)
└── kafka/
    └── consumer.go      # Kafka consumer — document.ingested topic
```

---

## 5. Yêu cầu chức năng

### 5.1 FR-REG — Registry & App Management

#### FR-REG-01: Đăng ký ứng dụng (App)

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-REG-01 |
| **Tên** | CreateApp |
| **Mức ưu tiên** | Cao |
| **Endpoint** | `POST /v1/apps` |

**Mô tả:** Hệ thống phải cho phép đăng ký một client application mới vào KGS. Mỗi App có một `app_id` duy nhất, đóng vai trò namespace để cô lập data trong graph.

**Input:**
```json
{
  "appName": "string (required)",
  "description": "string (optional)",
  "owner": "string (required)"
}
```

**Output:**
```json
{
  "appId": "string",
  "status": "ACTIVE"
}
```

**Logic:**
- Tạo bản ghi `App` trong PostgreSQL
- Status mặc định là `ACTIVE`
- Soft-delete có hỗ trợ (gorm.DeletedAt)

#### FR-REG-02: Xem danh sách & chi tiết App

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-REG-02 |
| **Endpoints** | `GET /v1/apps`, `GET /v1/apps/{appId}` |

**Mô tả:** Hệ thống phải cho phép liệt kê tất cả Apps và lấy chi tiết từng App.

#### FR-REG-03: Phát hành API Key

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-REG-03 |
| **Endpoint** | `POST /v1/apps/{appId}/keys` |
| **Mức ưu tiên** | Cao |

**Mô tả:** Hệ thống phải cho phép phát hành API Key cho một App để xác thực các requests sau này.

**Input:**
```json
{
  "appId": "string",
  "name": "string",
  "scopes": "string (comma-separated, e.g. 'read,write')",
  "ttlSeconds": "int64 (optional, 0 = no expiry)"
}
```

**Output:**
```json
{
  "apiKey": "string (raw key — hiển thị một lần duy nhất)",
  "keyHash": "string (SHA-256 hash)",
  "keyPrefix": "string (vài ký tự đầu để nhận dạng)"
}
```

**Logic:**
- Lưu `keyHash` (SHA-256) vào PostgreSQL, không lưu raw key
- `keyPrefix` để người dùng nhận dạng key
- `scopes` quy định quyền: `read`, `write`, v.v.

#### FR-REG-04: Thu hồi API Key

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-REG-04 |
| **Endpoint** | `DELETE /v1/keys/{keyHash}` |

**Mô tả:** Huỷ bỏ một API Key theo hash. Key bị thu hồi sẽ không còn được xác thực.

---

### 5.2 FR-ONT — Ontology Management

#### FR-ONT-01: Tạo EntityType (Node Schema)

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-ONT-01 |
| **Endpoint** | `POST /v1/ontology/entities` |
| **Mức ưu tiên** | Cao |

**Mô tả:** Hệ thống phải cho phép định nghĩa các loại node (EntityType) cho một App. EntityType xác định label và JSON Schema cho properties của node.

**Input:**
```json
{
  "name": "string (e.g. 'Customer', 'Transaction')",
  "description": "string",
  "schema": "string (JSON Schema definition)"
}
```

**Logic:**
- Validate `schema` phải là JSON Schema hợp lệ (dùng `gojsonschema`)
- Lưu vào PostgreSQL, unique constraint trên `(app_id, name)`
- Tự động seed 18 EntityTypes thuộc `system` app khi khởi động

**EntityTypes được seed sẵn (app_id = "system"):**

| Layer | EntityType | Mô tả |
|---|---|---|
| PRD/URD | `Feature` | Product feature từ PRD |
| PRD/URD | `UserStory` | User story |
| PRD/URD | `BusinessRule` | Business rule |
| PRD/URD | `Actor` | System actor/user role |
| PRD/URD | `UseCase` | Use case definition |
| PRD/URD | `DataEntity` | Domain data entity |
| PRD/URD | `Constraint` | System constraint |
| SRS | `SRSRequirement` | Yêu cầu phần mềm có cấu trúc |
| SRS | `SystemInterface` | Đặc tả interface hệ thống |
| UI | `UIScreen` | Màn hình UI |
| UI | `UIComponent` | Thành phần UI |
| UI | `UIFlow` | Luồng điều hướng UI |
| UI | `UIValidationRule` | Quy tắc validation UI |
| Test | `TestRequirement` | Yêu cầu kiểm thử |
| Test | `TestDesign` | Tài liệu thiết kế test |
| Test | `TestCase` | Test case đơn lẻ |
| Test | `TestSuite` | Nhóm test cases |
| Test | `TestScript` | Script kiểm thử tự động |

#### FR-ONT-02: Tạo RelationType (Edge Schema)

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-ONT-02 |
| **Endpoint** | `POST /v1/ontology/relations` |

**Input:**
```json
{
  "name": "string (e.g. 'REFINES', 'TESTS')",
  "description": "string",
  "propertiesSchema": "string (JSON Schema, optional)",
  "sourceTypes": ["string"],
  "targetTypes": ["string"]
}
```

**RelationTypes được seed sẵn (app_id = "system"):**

| RelationType | Mô tả |
|---|---|
| `REFINES` | SRSRequirement refines Feature/UserStory |
| `DERIVES_FROM` | TestRequirement derives from Feature |
| `TESTS` | TestCase tests a Requirement |
| `IMPLEMENTS` | TestScript implements TestCase |
| `GROUPS` | TestSuite groups TestCases |
| `SPECIFIES_INTERFACE` | SRSRequirement specifies SystemInterface |
| `RENDERED_ON` | UIComponent rendered on UIScreen |
| `CONTAINS_COMPONENT` | UIScreen contains UIComponent |
| `NAVIGATES_TO` | UIFlow navigates to UIScreen |
| `VALIDATES_FIELD` | UIValidationRule validates UIComponent field |
| `TESTED_ON_SCREEN` | TestCase tested on UIScreen |
| `HAS_CHILD` | Parent-child relationship |
| `DEPENDS_ON` | Dependency relationship |
| `RELATED_TO` | Generic association |
| `AUTOMATES` | TestScript automates TestCase |
| `PART_OF` | Component is part of system |

#### FR-ONT-03: Liệt kê Ontology

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-ONT-03 |
| **Endpoints** | `GET /v1/ontology/entities`, `GET /v1/ontology/relations` |

**Mô tả:** Hệ thống phải cho phép listing toàn bộ EntityTypes và RelationTypes đang được đăng ký.

---

### 5.3 FR-GRAPH — Knowledge Graph CRUD

#### FR-GRAPH-01: Tạo Node

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-GRAPH-01 |
| **Endpoint** | `POST /v1/graph/nodes` |
| **Mức ưu tiên** | Critical |

**Mô tả:** Hệ thống phải cho phép tạo node mới trong Neo4j. Mọi node đều được gắn `app_id` để namespace hóa trong đồ thị đa tenant.

**Input:**
```json
{
  "label": "string (e.g. 'Feature', 'UIScreen')",
  "propertiesJson": "string (JSON object)"
}
```

**Output:**
```json
{
  "nodeId": "string",
  "label": "string",
  "propertiesJson": "string"
}
```

**Luồng xử lý (bắt buộc theo thứ tự):**
1. **OPA Policy Check** — gọi `POST /v1/data/kgs/allow` với payload `{app_id, action: "CREATE_NODE", resource: label}`; từ chối nếu OPA trả về `false`
2. **Neo4j Write** — thực thi Cypher `CREATE (n:Label {app_id: $app_id}) SET n += $props`
3. **Event Publish** — ghi event `node.created` lên Redis Stream `kgs:events:nodes`

**Cypher template:**
```cypher
CREATE (n:<label> {app_id: $app_id})
SET n += $props
RETURN n
```

#### FR-GRAPH-02: Cập nhật Node

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-GRAPH-02 |
| **Endpoint** | `PUT /v1/graph/nodes/{nodeId}` |

**Input:**
```json
{
  "appId": "string",
  "nodeId": "string",
  "properties": {"key": "value"}
}
```

**Luồng xử lý:**
1. OPA Policy Check (`action: "UPDATE_NODE"`)
2. Neo4j Write: `MATCH (n {app_id, id}) SET n += $props`
3. Publish `node.updated` lên Redis Stream

#### FR-GRAPH-03: Xoá Node

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-GRAPH-03 |
| **Endpoint** | `DELETE /v1/graph/nodes/{nodeId}` |

**Mô tả:** Xoá node cùng tất cả relationship liên quan (`DETACH DELETE`).

**Luồng xử lý:**
1. OPA Policy Check (`action: "DELETE_NODE"`)
2. Neo4j Write: `MATCH (n {app_id, id}) DETACH DELETE n`
3. Publish `node.deleted` lên Redis Stream

#### FR-GRAPH-04: Tạo Edge (Relationship)

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-GRAPH-04 |
| **Endpoint** | `POST /v1/graph/edges` |

**Input:**
```json
{
  "sourceNodeId": "string",
  "targetNodeId": "string",
  "relationType": "string (e.g. 'REFINES', 'TESTS')",
  "propertiesJson": "string"
}
```

**Cypher template:**
```cypher
MATCH (a {app_id: $app_id, id: $source_node_id})
MATCH (b {app_id: $app_id, id: $target_node_id})
CREATE (a)-[rel:<relationType> {app_id: $app_id}]->(b)
SET rel += $props
RETURN rel
```

---

### 5.4 FR-QUERY — Graph Query & Traversal

#### FR-QUERY-01: Get Context (Neighbors)

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-QUERY-01 |
| **Endpoint** | `GET /v1/graph/nodes/{nodeId}/context?depth=N&direction=INCOMING\|OUTGOING\|BOTH` |

**Mô tả:** Lấy các node lân cận xung quanh một node trung tâm, hỗ trợ lọc theo hướng relationship.

**Parameters:**
- `depth` (int) — số bước traversal, mặc định 1, tối đa 10 (guardrail)  
- `direction` — `INCOMING`, `OUTGOING`, hoặc `BOTH`

**Cypher được sinh bởi QueryPlanner:**
```cypher
-- direction = BOTH
MATCH (n {app_id: $app_id, id: $node_id})-[r]-(m {app_id: $app_id})
RETURN n, r, m

-- direction = OUTGOING
MATCH (n {app_id: $app_id, id: $node_id})-[r]->(m {app_id: $app_id})
RETURN n, r, m

-- direction = INCOMING
MATCH (n {app_id: $app_id, id: $node_id})<-[r]-(m {app_id: $app_id})
RETURN n, r, m
```

#### FR-QUERY-02: Get Impact (Downstream Traversal)

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-QUERY-02 |
| **Endpoint** | `GET /v1/graph/nodes/{nodeId}/impact?maxDepth=N` |

**Mô tả:** Tìm tất cả node phía downstream bị ảnh hưởng bởi một node (traversal xuôi chiều theo relationship).

**Cypher:**
```cypher
MATCH p=(n {app_id: $app_id, id: $node_id})-[*1..{maxDepth}]->(m {app_id: $app_id})
RETURN nodes(p) AS nodes, relationships(p) AS rels
```

#### FR-QUERY-03: Get Coverage (Upstream Traversal)

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-QUERY-03 |
| **Endpoint** | `GET /v1/graph/nodes/{nodeId}/coverage?maxDepth=N` |

**Mô tả:** Tìm tất cả node phía upstream (nguồn gốc) của một node (traversal ngược chiều).

**Cypher:**
```cypher
MATCH p=(n {app_id: $app_id, id: $node_id})<-[*1..{maxDepth}]-(m {app_id: $app_id})
RETURN nodes(p) AS nodes, relationships(p) AS rels
```

#### FR-QUERY-04: Get Subgraph

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-QUERY-04 |
| **Endpoint** | `POST /v1/graph/subgraph` |

**Mô tả:** Lấy subgraph bao gồm nodes và relationships giữa một tập hợp node IDs cho trước. Dùng cho rendering UI hoặc context assembly cho LLM.

**Input:**
```json
{
  "nodeIds": ["id1", "id2", "id3"]
}
```

**Guardrail:** Số nodeIds tối đa là 1000.

**Cypher:**
```cypher
MATCH (n {app_id: $app_id})-[r]->(m {app_id: $app_id})
WHERE n.id IN $node_ids AND m.id IN $node_ids
RETURN n, r, m
```

---

### 5.5 FR-RULES — Business Rules Engine

#### FR-RULES-01: Tạo Business Rule

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-RULES-01 |
| **Endpoint** | `POST /v1/rules` |
| **Mức ưu tiên** | Cao |

**Mô tả:** Hệ thống phải cho phép định nghĩa các business rules được thực thi tự động trên đồ thị. Rule có hai loại trigger:

| TriggerType | Mô tả |
|---|---|
| `SCHEDULED` | Thực thi theo lịch cron, ví dụ `"0 0 * * *"` (mỗi ngày lúc nửa đêm) |
| `ON_WRITE` | Phản ứng ngay khi có node được tạo/cập nhật/xoá (qua Redis Stream) |

**Input:**
```json
{
  "name": "string",
  "description": "string",
  "triggerType": "SCHEDULED | ON_WRITE",
  "cron": "string (cron expression, chỉ cho SCHEDULED)",
  "cypherQuery": "string (Cypher query để thực thi)",
  "action": "string (webhook | push_notification)",
  "payloadJson": "string (payload cho action)"
}
```

#### FR-RULES-02: Liệt kê & Xem Rule

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-RULES-02 |
| **Endpoints** | `GET /v1/rules`, `GET /v1/rules/{id}` |

#### FR-RULES-03: Thực thi Rule theo Cron (RuleRunner)

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-RULES-03 |
| **Component** | `biz.RuleRunner` (WorkerServer) |

**Mô tả:** Khi khởi động, `RuleRunner` phải:
1. Load tất cả Rules có `trigger_type = SCHEDULED` và `is_active = true` từ PostgreSQL
2. Đăng ký mỗi rule với scheduler (gocron) theo `cron` expression
3. Khi đến lịch: thực thi `cypher_query` trên Neo4j với params `{app_id: rule.AppID}`
4. Log kết quả; trong production: ghi vào bảng `rule_executions`

#### FR-RULES-04: Thực thi Rule theo Event (EventRunner)

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-RULES-04 |
| **Component** | `biz.EventRunner` (WorkerServer) |

**Mô tả:** `EventRunner` phải liên tục đọc Redis Stream `kgs:events:nodes` (consumer group `kgs-worker-group`) và:
1. Với mỗi event (`node.created`, `node.updated`, `node.deleted`):
2. Load tất cả Rules có `trigger_type = ON_WRITE` và `is_active = true`
3. Thực thi `cypher_query` của rule với event payload làm params
4. Trigger action của rule (webhook, v.v.)

---

### 5.6 FR-POLICY — Access Control & Policy Management

#### FR-POLICY-01: Tạo OPA Policy

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-POLICY-01 |
| **Endpoint** | `POST /v1/policies` |
| **Mức ưu tiên** | Cao |

**Mô tả:** Hệ thống phải cho phép nhập nội dung Rego policy để kiểm soát truy cập vào graph operations cho từng App.

**Input:**
```json
{
  "name": "string",
  "description": "string",
  "regoContent": "string (Rego policy code)"
}
```

**Default policy (kgs.rego):**
```rego
package kgs

import rego.v1

default allow := false

allow if {
    input.app_id == "demo-app"
}
```

**OPA evaluation input schema:**
```json
{
  "input": {
    "app_id": "string",
    "action": "CREATE_NODE | UPDATE_NODE | DELETE_NODE",
    "resource": "string (e.g. label name)"
  }
}
```

#### FR-POLICY-02: Policy Sync đến OPA (PolicySyncRunner)

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-POLICY-02 |
| **Component** | `biz.PolicySyncRunner` (WorkerServer) |

**Mô tả:** `PolicySyncRunner` phải tự động đồng bộ policies từ PostgreSQL lên OPA theo chu kỳ:
1. Mỗi 30 giây: load tất cả Policy records có `is_active = true`
2. Với mỗi policy: `PUT /v1/policies/policy_{id}` lên OPA với nội dung Rego
3. Log lỗi nếu upload thất bại, tiếp tục với policy tiếp theo

#### FR-POLICY-03: OPA Access Control Gate

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-POLICY-03 |

**Mô tả:** Tất cả write operations lên graph (CreateNode, UpdateNode, DeleteNode) **phải** qua OPA evaluation trước khi thực thi. Nếu OPA không khả dụng (unreachable), hệ thống phải **fail closed** (từ chối request).

---

### 5.7 FR-EVENT — Event-Driven Processing

#### FR-EVENT-01: Redis Stream Event Publishing

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-EVENT-01 |
| **Stream** | `kgs:events:nodes` |

**Mô tả:** Sau mỗi write operation thành công, hệ thống phải publish event lên Redis Stream.

**Event schema:**
```
Stream: kgs:events:nodes
Fields:
  event_type: "node.created" | "node.updated" | "node.deleted"
  app_id: string
  label: string (for node.created)
  node_id: string (for node.updated, node.deleted)
```

#### FR-EVENT-02: Redis Stream Consumer

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-EVENT-02 |
| **Consumer Group** | `kgs-worker-group` |
| **Consumer** | `worker-1` |

**Mô tả:** EventRunner consume events từ stream và trigger ON_WRITE rules. Stream sử dụng consumer group để đảm bảo at-least-once delivery. ACK message sau khi xử lý xong.

---

### 5.8 FR-KAFKA — Document Ingestion via Kafka

#### FR-KAFKA-01: Consume Document Ingested Events

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-KAFKA-01 |
| **Topic** | `document.ingested` |
| **Consumer Group** | `knowledge-service` |
| **Component** | `kafka.Consumer` (WorkerServer) |

**Mô tả:** KGS phải lắng nghe Kafka topic `document.ingested` và tự động tạo nodes/edges tương ứng trong graph.

**Event schema (DocumentIngestedEvent):**
```json
{
  "docId": "string",
  "appId": "string",
  "docType": "PRD | SRS | UI | TESTCASE",
  "nodeType": "string (EntityType label, e.g. 'Feature')",
  "properties": {"key": "value"},
  "parentId": "string (optional)",
  "edgeType": "string (optional, e.g. 'REFINES')"
}
```

**Luồng xử lý:**
1. Parse Kafka message thành `DocumentIngestedEvent`
2. Gọi `GraphUsecase.CreateNode(appId, nodeType, properties)` → tạo node
3. Nếu `parentId != ""` và `edgeType != ""`: gọi `GraphUsecase.CreateEdge(appId, edgeType, parentId, nodeId, nil)` → tạo edge

---

### 5.9 FR-VECTOR — Vector Semantic Search

#### FR-VECTOR-01: Upsert Knowledge Chunks vào Qdrant

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-VECTOR-01 |
| **Component** | `data.QdrantClient` |
| **Collection** | `knowledge_chunks` |

**Mô tả:** Hệ thống phải có khả năng lưu vector embeddings của knowledge chunks vào Qdrant để phục vụ semantic search.

**Qdrant Point schema:**
```json
{
  "id": "string",
  "vector": [float32, ...],
  "payload": {"key": "value"}
}
```

#### FR-VECTOR-02: Semantic Search trên Knowledge Chunks

| Thuộc tính | Mô tả |
|---|---|
| **ID** | FR-VECTOR-02 |

**Mô tả:** Hệ thống phải cho phép tìm kiếm semantic trên Qdrant theo vector query, threshold điểm, và số lượng kết quả (topK).

---

## 6. Yêu cầu phi chức năng

### 6.1 NFR-PERF — Hiệu năng

| ID | Yêu cầu |
|---|---|
| NFR-PERF-01 | HTTP API response time < 200ms cho 95th percentile (không tính traversal query phức tạp) |
| NFR-PERF-02 | gRPC API response time < 100ms cho CRUD đơn lẻ |
| NFR-PERF-03 | Graph traversal query (context/impact/coverage) < 1s cho depth ≤ 5 trong graph 100K nodes |
| NFR-PERF-04 | Kafka consumer lag < 5 giây so với producer |
| NFR-PERF-05 | Redis Stream event latency < 500ms từ lúc write đến lúc EventRunner xử lý |
| NFR-PERF-06 | Sử dụng `go.uber.org/automaxprocs` để tự điều chỉnh GOMAXPROCS theo CPU available |

### 6.2 NFR-SEC — Bảo mật

| ID | Yêu cầu |
|---|---|
| NFR-SEC-01 | Tất cả ghi vào graph phải qua OPA evaluation; nếu OPA không khả dụng → fail closed |
| NFR-SEC-02 | API Keys phải được lưu dưới dạng SHA-256 hash, không lưu raw key |
| NFR-SEC-03 | Mọi node và edge trong Neo4j phải có `app_id` property để đảm bảo isolation |
| NFR-SEC-04 | Cypher queries phải dùng parameters để ngăn Cypher Injection (không interpolate giá trị runtime vào query string) |
| NFR-SEC-05 | Rego policy được quản lý qua DB và sync sang OPA; không hard-code logic authorization |

### 6.3 NFR-REL — Độ tin cậy

| ID | Yêu cầu |
|---|---|
| NFR-REL-01 | Service phải có panic recovery middleware trên mọi request (kratos recovery middleware) |
| NFR-REL-02 | Kafka consumer phải retry khi gặp lỗi đọc message, không exit process |
| NFR-REL-03 | EventRunner phải retry đọc Redis Stream sau 2 giây khi lỗi |
| NFR-REL-04 | Data connections (Postgres, Neo4j, Redis) phải fail-fast khi khởi động nếu không kết nối được |
| NFR-REL-05 | PolicySyncRunner phải tiếp tục sync các policy còn lại dù một policy upload thất bại |

### 6.4 NFR-SCALE — Khả năng mở rộng

| ID | Yêu cầu |
|---|---|
| NFR-SCALE-01 | Multi-tenant: mỗi app_id độc lập về data, không thể truy cập data của app khác |
| NFR-SCALE-02 | App, APIKey, Quota, Rule, Policy có thể scale theo số lượng App đăng ký |
| NFR-SCALE-03 | Neo4j graph được namespace hóa bằng `app_id` property trên mọi node/edge |

### 6.5 NFR-OBS — Observability

| ID | Yêu cầu |
|---|---|
| NFR-OBS-01 | Tích hợp OpenTelemetry tracing (go.opentelemetry.io/otel v1.39.0) |
| NFR-OBS-02 | Structured logging qua Kratos log.Logger |
| NFR-OBS-03 | Metric exposure cho monitoring (OTel SDK metric) |
| NFR-OBS-04 | Log tất cả OPA evaluation failures ở level Error |

---

## 7. Mô hình dữ liệu

### 7.1 PostgreSQL — State Store

Database name: `kgs_platform` (configurable)

#### Table: apps

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `app_id` | varchar(50) | PRIMARY KEY | Unique identifier |
| `app_name` | varchar(200) | NOT NULL | Tên hiển thị |
| `description` | text | | Mô tả chi tiết |
| `owner` | varchar(100) | NOT NULL | Người sở hữu |
| `status` | varchar(20) | DEFAULT 'ACTIVE' | ACTIVE, INACTIVE, SUSPENDED |
| `created_at` | timestamp | | |
| `updated_at` | timestamp | | |
| `deleted_at` | timestamp | INDEX | Soft delete |

#### Table: api_keys

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `key_hash` | varchar(64) | PRIMARY KEY | SHA-256 của raw key |
| `app_id` | varchar(50) | NOT NULL, INDEX | FK → apps |
| `key_prefix` | varchar(10) | NOT NULL | Vài ký tự đầu key |
| `name` | varchar(100) | | Tên định danh |
| `scopes` | varchar(500) | | Comma-separated permissions |
| `expires_at` | timestamp | NULLABLE | null = không hết hạn |
| `created_at` | timestamp | | |
| `deleted_at` | timestamp | INDEX | Soft delete / revoke |

#### Table: quotas

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `id` | uint | PRIMARY KEY | |
| `app_id` | varchar(50) | NOT NULL, UNIQUE INDEX (app_id, quota_type) | |
| `quota_type` | varchar(50) | NOT NULL | e.g. "requests_per_minute", "max_nodes" |
| `limit` | int64 | NOT NULL | Giá trị giới hạn |

#### Table: audit_logs

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `id` | uint | PRIMARY KEY | |
| `app_id` | varchar(50) | INDEX | |
| `action` | varchar(100) | NOT NULL | |
| `actor` | varchar(100) | NOT NULL | |
| `details` | text | | |
| `created_at` | timestamp | INDEX | |

#### Table: entity_types

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `id` | uint | PRIMARY KEY | |
| `app_id` | varchar(50) | UNIQUE INDEX (app_id, name) | |
| `name` | varchar(100) | NOT NULL, UNIQUE INDEX (app_id, name) | Label name |
| `description` | text | | |
| `schema` | jsonb | NOT NULL | JSON Schema cho properties |
| `created_at`, `updated_at`, `deleted_at` | timestamp | | |

#### Table: relation_types

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `id` | uint | PRIMARY KEY | |
| `app_id` | varchar(50) | UNIQUE INDEX (app_id, name) | |
| `name` | varchar(100) | NOT NULL | Relationship type name |
| `description` | text | | |
| `properties` | jsonb | | JSON Schema cho edge properties |
| `source_types` | jsonb | | List of valid source EntityType names |
| `target_types` | jsonb | | List of valid target EntityType names |

#### Table: rules

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `id` | uint | PRIMARY KEY | |
| `app_id` | varchar(50) | NOT NULL, INDEX | |
| `name` | varchar(100) | NOT NULL | |
| `trigger_type` | varchar(20) | NOT NULL | SCHEDULED \| ON_WRITE |
| `cron` | varchar(50) | | Cron expression (SCHEDULED only) |
| `cypher_query` | text | NOT NULL | Query thực thi trên Neo4j |
| `action` | varchar(50) | | webhook \| push_notification |
| `payload` | jsonb | | Action payload |
| `is_active` | bool | DEFAULT true | |

#### Table: rule_executions

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `id` | uint | PRIMARY KEY | |
| `app_id` | varchar(50) | NOT NULL, INDEX | |
| `rule_id` | uint | NOT NULL | FK → rules |
| `status` | varchar(20) | NOT NULL | SUCCESS \| FAILED |
| `message` | text | | Error message nếu FAILED |
| `started_at` | timestamp | INDEX | |
| `ended_at` | timestamp | | |

#### Table: policies

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `id` | uint | PRIMARY KEY | |
| `app_id` | varchar(50) | NOT NULL, INDEX | |
| `name` | varchar(100) | NOT NULL | |
| `description` | text | | |
| `rego_content` | text | NOT NULL | Nội dung Rego policy |
| `is_active` | bool | DEFAULT true | |

### 7.2 Neo4j — Graph Store

**Database:** `kgs` (configurable)  
**Connection:** bolt://localhost:7687

**Node property schema (tất cả nodes):**
```
Node properties:
  app_id: string (REQUIRED — namespace)
  id: string (OPTIONAL — application-level ID)
  + bất kỳ properties nào trong propertiesJson
```

**Edge property schema (tất cả edges):**
```
Relationship properties:
  app_id: string (REQUIRED — namespace)
  + bất kỳ properties nào trong propertiesJson
```

**Access pattern — tất cả queries phải filter theo app_id:**
```cypher
MATCH (n {app_id: $app_id, id: $node_id}) ...
```

### 7.3 Redis — Stream & Cache

**Connection:** `localhost:6379`

**Streams:**

| Key | Type | Producer | Consumer | Mô tả |
|---|---|---|---|---|
| `kgs:events:nodes` | Stream | GraphUsecase | EventRunner (group: kgs-worker-group) | Node write events |

**Event message fields:**
```
event_type: "node.created" | "node.updated" | "node.deleted"
app_id: string
label: string (node.created only)
node_id: string (node.updated, node.deleted)
```

---

## 8. API Specification

### 8.1 Transport

| Protocol | Port | Mô tả |
|---|---|---|
| HTTP/1.1 (REST+JSON) | 8000 | Dành cho external clients, gateways |
| gRPC (HTTP/2 + Protobuf) | 9000 | Dành cho internal service-to-service |

### 8.2 Middleware (áp dụng cho tất cả routes)

| Middleware | Mô tả |
|---|---|
| **Recovery** | Catch panics, trả về 500 thay vì crash |
| **Auth** | Kiểm tra API Key header |
| **RateLimiter** | Giới hạn số request theo Quota config |

### 8.3 REST API Endpoints Summary

| Method | Path | Service | Mô tả |
|---|---|---|---|
| `GET` | `/helloworld/{name}` | Greeter | Health check |
| `GET` | `/v1/apps` | Registry | List applications |
| `POST` | `/v1/apps` | Registry | Create application |
| `GET` | `/v1/apps/{appId}` | Registry | Get application |
| `POST` | `/v1/apps/{appId}/keys` | Registry | Issue API key |
| `DELETE` | `/v1/keys/{keyHash}` | Registry | Revoke API key |
| `POST` | `/v1/ontology/entities` | Ontology | Create EntityType |
| `GET` | `/v1/ontology/entities` | Ontology | List EntityTypes |
| `POST` | `/v1/ontology/relations` | Ontology | Create RelationType |
| `GET` | `/v1/ontology/relations` | Ontology | List RelationTypes |
| `POST` | `/v1/graph/nodes` | Graph | Create node |
| `GET` | `/v1/graph/nodes/{nodeId}` | Graph | Get node |
| `PUT` | `/v1/graph/nodes/{nodeId}` | Graph | Update node |
| `DELETE` | `/v1/graph/nodes/{nodeId}` | Graph | Delete node |
| `POST` | `/v1/graph/edges` | Graph | Create edge |
| `GET` | `/v1/graph/nodes/{nodeId}/context` | Graph | Get context neighbors |
| `GET` | `/v1/graph/nodes/{nodeId}/impact` | Graph | Get downstream impact |
| `GET` | `/v1/graph/nodes/{nodeId}/coverage` | Graph | Get upstream coverage |
| `POST` | `/v1/graph/subgraph` | Graph | Get subgraph by node IDs |
| `POST` | `/v1/rules` | Rules | Create rule |
| `GET` | `/v1/rules` | Rules | List rules |
| `GET` | `/v1/rules/{id}` | Rules | Get rule |
| `POST` | `/v1/policies` | AccessControl | Create policy |
| `GET` | `/v1/policies` | AccessControl | List policies |
| `GET` | `/v1/policies/{id}` | AccessControl | Get policy |

### 8.4 gRPC Services

Tất cả REST endpoints đều có gRPC counterpart:
- `helloworld.v1.Greeter`
- `registry.v1.Registry`
- `ontology.v1.Ontology`
- `graph.v1.Graph`
- `rules.v1.Rules`
- `accesscontrol.v1.AccessControl`

---

## 9. Ràng buộc hệ thống (Guardrails)

| Constant | Giá trị | Áp dụng cho | Lý do |
|---|---|---|---|
| `MaxAllowedDepth` | 10 | GetContext, GetImpact, GetCoverage | Ngăn traversal đệ quy vô hạn trong Neo4j |
| `MaxAllowedNodes` | 1000 | GetSubgraph | Giới hạn kích thước subgraph response |

**Lỗi trả về khi vi phạm:**
- `ErrDepthExceeded`: "requested query depth exceeds the maximum allowed limit"
- `ErrNodesExceeded`: "requested query node count exceeds the maximum allowed limit"

---

## 10. Luồng tích hợp với hệ thống ngoài

### 10.1 Luồng Document Ingestion (ai-orchestrator → KGS)

```
ai-orchestrator
    │
    │ Kafka: document.ingested
    │ {docId, appId, nodeType, properties, parentId?, edgeType?}
    ▼
KGS KafkaConsumer
    │
    ├──▶ GraphUsecase.CreateNode(appId, nodeType, properties)
    │        │
    │        ├──▶ OPA.EvaluatePolicy(appId, "CREATE_NODE", nodeType)
    │        │
    │        └──▶ Neo4j.CreateNode(label=nodeType, app_id=appId, props)
    │                 │
    │                 └──▶ Redis.XAdd("kgs:events:nodes", "node.created")
    │
    └──▶ [if parentId exists] GraphUsecase.CreateEdge(appId, edgeType, parentId, nodeId)
              └──▶ Neo4j.CreateEdge(relationType=edgeType, source=parentId, target=nodeId)
```

### 10.2 Luồng Query từ pipeline-shim / service-kg-to-preview

```
pipeline-shim / service-kg-to-preview
    │
    │ gRPC: GetContext({nodeId, depth, direction})
    │    OR GetSubgraph({nodeIds})
    ▼
KGS GraphService
    │
    ├──▶ GraphUsecase.GetContext / GetSubgraph
    │        │
    │        ├──▶ QueryPlanner.Build*Query() → Cypher string
    │        │
    │        └──▶ Neo4jRepo.ExecuteQuery(cypher, params{app_id, node_id,...})
    │                 └──▶ Return: {nodes, edges}
    │
    └──▶ Return: GraphReply{nodes[], edges[]}
```

### 10.3 Luồng Policy Evaluation (OPA gate)

```
[Any write request: CreateNode | UpdateNode | DeleteNode]
    │
    ▼
OPAClient.EvaluatePolicy(app_id, action, resource)
    │
    │ POST http://opa:8181/v1/data/kgs/allow
    │ Body: {"input": {"app_id": "...", "action": "...", "resource": "..."}}
    │
    ├── OPA returns {result: true}  → Tiếp tục xử lý
    └── OPA returns {result: false} → Return error "access denied by OPA policy"
        OPA unreachable            → Return error (fail closed)
```

### 10.4 Luồng ON_WRITE Rule Execution

```
GraphUsecase.CreateNode/UpdateNode/DeleteNode
    │
    └──▶ Redis.XAdd("kgs:events:nodes", event_payload)

EventRunner (background goroutine)
    │
    └──▶ Redis.XReadGroup("kgs:events:nodes", "kgs-worker-group", ">")
              │
              ├──▶ RulesRepo.ListRules(app_id) → filter ON_WRITE + is_active
              │
              ├──▶ For each rule: GraphRepo.ExecuteQuery(rule.CypherQuery, {app_id, event})
              │
              ├──▶ Trigger rule.Action (webhook, notification)
              │
              └──▶ Redis.XAck(message.ID)
```

---

## 11. Yêu cầu bảo mật

### 11.1 Authentication

| Yêu cầu | Mô tả |
|---|---|
| API Key based | Clients phải gửi API Key trong header, middleware `Auth` kiểm tra |
| Key storage | Chỉ lưu SHA-256 hash; không thể recover raw key |
| Key revocation | Xoá soft-delete bản ghi trong `api_keys` |
| Scopes | API Keys có danh sách scopes; middleware kiểm tra permission tương ứng với action |

### 11.2 Authorization

| Yêu cầu | Mô tả |
|---|---|
| OPA Gate | Tất cả write operations bắt buộc qua OPA evaluation |
| Fail Closed | OPA unreachable → reject request |
| Policy-as-Code | Policies viết bằng Rego, lưu trong DB, sync lên OPA mỗi 30s |
| App Isolation | Mọi Neo4j operation đều filter theo `app_id`; không thể cross-tenant |

### 11.3 Input Validation

| Yêu cầu | Mô tả |
|---|---|
| JSON Schema | Properties của node được validate theo EntityType schema (gojsonschema) |
| Cypher Injection | Dùng parameterized Cypher; label/relationType được interpolate an toàn |
| Depth Guardrail | Reject query có depth > 10 (DoS prevention) |
| Node Count Guardrail | Reject subgraph query với > 1000 nodeIDs |

---

## 12. Yêu cầu vận hành

### 12.1 Configuration (config.yaml)

```yaml
server:
  http:
    addr: 0.0.0.0:8000
    timeout: 1s
  grpc:
    addr: 0.0.0.0:9000
    timeout: 1s
data:
  database:
    driver: postgres
    source: "host=... user=... password=... dbname=kgs_platform port=5432 sslmode=disable"
  redis:
    network: tcp
    addr: "127.0.0.1:6379"
    password: ""
    read_timeout: 0.2s
    write_timeout: 0.2s
  neo4j:
    uri: bolt://localhost:7687
    user: neo4j
    password: "..."
    database: kgs
  opa:
    url: http://localhost:8181
  qdrant:
    url: http://localhost:6333
    collection: knowledge_chunks
  kafka:
    brokers:
      - localhost:9092
    topic_document_ingested: document.ingested
```

### 12.2 Startup Sequence

Khi khởi động, service thực hiện theo thứ tự:

1. Load config từ `configs/config.yaml`
2. Kết nối PostgreSQL → fail-fast nếu không kết nối được
3. Auto-migrate schemas (9 tables)
4. Seed Ontology (18 EntityTypes + 16 RelationTypes cho app_id = "system")
5. Kết nối Neo4j → fail-fast
6. Kết nối Redis + Ping → fail-fast
7. Khởi tạo Qdrant client (lazy connection)
8. Khởi tạo OPA client
9. Khởi tạo Kafka consumer
10. Start HTTP server (:8000)
11. Start gRPC server (:9000)
12. Start WorkerServer → RuleRunner + EventRunner + PolicySyncRunner + KafkaConsumer

### 12.3 Build & Deploy

```bash
# Build binary
make build
# Output: ./bin/server

# Run via Docker
docker build -t vnp/kgs-platform:latest .
docker run -v configs/:/configs/ kgs-platform
```

### 12.4 Infrastructure Dependencies

| Service | Min Version | Role |
|---|---|---|
| PostgreSQL | 14+ | Primary state store |
| Neo4j | 5.x (bolt://) | Graph database |
| Redis | 6.2+ (Stream support) | Event stream + cache |
| Qdrant | 1.x | Vector store (optional) |
| Apache Kafka | 2.8+ | Event bus (optional — consumer only) |
| Open Policy Agent | 0.60+ | Authorization sidecar |

---

## 13. Giả định và phụ thuộc

### 13.1 Giả định

| # | Giả định |
|---|---|
| A1 | `app_id` được extract từ API Key tại middleware lớp Auth (hiện tại đang hardcode "demo-app" trong biz layer — cần hoàn thiện) |
| A2 | OPA sidecar luôn chạy cùng container/pod với kgs-platform |
| A3 | Neo4j database `kgs` đã được tạo trước khi service khởi động |
| A4 | Kafka broker sẵn sàng trước khi service start (consumer sẽ retry nếu mất kết nối) |
| A5 | Qdrant collection `knowledge_chunks` đã được tạo trước (hoặc sẽ tạo runtime) |
| A6 | JSON Schema trong EntityType phải là valid JSON Schema Draft 4/7 |

### 13.2 Phụ thuộc kỹ thuật

| Dependency | Version | Mức độ bắt buộc |
|---|---|---|
| PostgreSQL | Required | Critical |
| Neo4j | Required | Critical |
| Redis | Required | Critical |
| Open Policy Agent | Required | Critical (fail-closed) |
| Apache Kafka | Optional | Non-critical (consumer chỉ start nếu config) |
| Qdrant | Optional | Non-critical (lazy init) |

### 13.3 Hạn chế đã biết (Known Limitations)

| # | Hạn chế | Impact |
|---|---|---|
| L1 | `app_id` đang được hardcode "demo-app" hoặc "system" trong nhiều biz methods — chưa extract từ Auth context | Multi-tenant chưa hoàn chỉnh |
| L2 | `RegistryService` và `OntologyService` chưa implement biz logic (đang trả về empty response) | Registry/Ontology API chưa functional |
| L3 | `GetNode` endpoint trả về empty response (chưa implement) | Read single node chưa hoạt động |
| L4 | `GraphReply` mapping từ Neo4j result chưa implement trong nhiều query endpoints | Context/Impact/Coverage/Subgraph trả về empty |
| L5 | `PolicySyncRunner` đang hardcode appID = "demo-app" thay vì iterate qua tất cả apps | Chỉ sync policies cho 1 app |
| L6 | Rule execution không ghi vào `rule_executions` table (TODO trong code) | Không có audit trail cho rule execution |
| L7 | `OPAClient.PutPolicy` hardcode `localhost:8181` thay vì dùng config URL | Sẽ fail nếu OPA không trên localhost |

---

*Tài liệu này được tạo tự động từ phân tích mã nguồn `kgs-platform` tại commit hiện tại. Vui lòng cập nhật khi có thay đổi significant trong codebase.*
