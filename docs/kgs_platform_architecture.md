__KGS__

__Knowledge Graph Service__

*Platform Architecture & Design Specification*

────────────────────────────────────────────────────

*Multi\-tenant  ·  Namespace Isolation  ·  Custom Ontology per App  ·  Rule Engine  ·  Access Control*

Version 1\.0  |  Tháng 2 / 2026

# PHẦN I — Tầm Nhìn & Định Nghĩa

KGS \(Knowledge Graph Service\) là một nền tảng graph as\-a\-service cho phép nhiều ứng dụng độc lập \(tenants\) đăng ký và sử dụng hạ tầng graph chung, trong khi vẫn duy trì được sự cô lập hoàn toàn về dữ liệu, ontology, rules và access control\. BA Agent System là ứng dụng đầu tiên \(reference tenant\) được build trên nền tảng này\.

## 1\.1 Vấn đề cần giải quyết

__Vấn đề__

__Nếu KHÔNG có KGS__

__Với KGS__

Mỗi app tự dựng graph

Tốn công, code lặp lại, không nhất quán

Dùng chung hạ tầng, chỉ khai báo ontology

Ontology khác nhau

Hard\-code labels trong code, khó thay đổi

Ontology là config, thay đổi không cần redeploy

Isolation data

Phải tự filter application\_id ở mọi query

Platform filter tự động, app không thể lọt sang tenant khác

Rules per app

Logic rule trải rải khắp codebase

Rule Engine quản lý tập trung, khai báo dưới dạng config

Access control

Mỗi app tự implement, inconsistent

Policy engine tập trung, attribute\-based \(ABAC\)

Scaling

Mỗi app tự scale Neo4j riêng

Shared cluster, tối ưu chi phí

## 1\.2 Đối tượng sử dụng nền tảng

__Actor__

__Vai trò__

__Tương tác chính__

Platform Admin

Quản lý toàn bộ KGS platform

Tạo / xóa app, quota, monitoring

App Developer

Developer của tenant \(vd: BA Agent team\)

Đăng ký app, khai báo ontology, viết rules, gọi Graph API

App Service / Agent

Runtime service của tenant

CRUD nodes/edges qua Graph API, query context

End User

Người dùng cuối của ứng dụng

Không tương tác trực tiếp với KGS

# PHẦN II — Kiến Trúc Tổng Thể

## 2\.1 Layered Architecture

┌─────────────────────────────────────────────────────────────────┐
│                     Consumer Layer                              │
│   BA Agent System    ·    App B    ·    App C    ·    ...        │
└─────────────────────────┬───────────────────────────────────────┘
                          │  REST / gRPC  (App API Key)           
┌─────────────────────────▼───────────────────────────────────────┐
│                    KGS Service Layer (Kratos)                   │
│   Auth (API Key → App Context)  ·  Rate Limit  ·  Audit Log     │
└──────┬──────────┬──────────────┬──────────────┬─────────────────┘
        │          │              │              │                   
┌──────▼──────┐ ┌──▼───────────┐ ┌──▼────────┐ ┌─▼─────────────┐       
│ Registry    │ │ Ontology     │ │  Graph    │ │  Rule Engine  │       
│ (Biz/Data)  │ │ (Biz/Data)   │ │  (Biz/Data)│ │  (Biz/Data)   │       
└─────────────┘ └──────────────┘ └───────────┘ └───────────────┘       

## 2.2 Core Components (Go / Kratos)

__Component__ | __Trách nhiệm__ | __Technology__ | __Layer__
---|---|---|---
Service Layer | Interface gRPC/HTTP, Auth, Middleware | Go + Kratos | Service
App Registry | Quản lý app lifecycle, API key, quota | Go + GORM (PostgreSQL) | Biz/Data
Ontology Service | CRUD ontology, validation schema per app | Go + GORM (PostgreSQL) | Biz/Data
Graph API | CRUD nodes/edges, namespaced Cypher | Go + Neo4j Driver | Biz/Data
Rule Engine | Quản lý và chạy Cypher rules | Go + Redis Streams | Biz/Data
Policy Engine | Evaluate access policies (ABAC) | Go + OPA (Open Policy Agent) | Biz/Data
Query Planner | Translate generic query → namespaced Cypher | Go Internal | Biz

## 2.3 Storage Layer
- **Neo4j**: Graph database (Namespaced labels: `{APP_ID}__{Type}`).
- **PostgreSQL**: Relational database for Registry, Ontology, Rules, and Policies metadata.
- **Redis**: Streaming for rules and caching.

## 2\.3 Multi\-tenancy: Shared Graph \+ Namespace Isolation

Mỗi tenant không có database riêng\. Thay vào đó, tất cả nodes và edges trong Neo4j đều mang nhãn namespace chứa app\_id:

// Convention đặt tên label trong Neo4j:

// Format:  \{APP\_ID\}\_\_\{EntityType\}

// Ví dụ BA Agent System \(app\_id = 'ba\_agent'\):

\(:ba\_agent\_\_Requirement \{ req\_id: 'REQ\-001', \.\.\. \}\)

\(:ba\_agent\_\_UseCase     \{ uc\_id:  'UC\-001',  \.\.\. \}\)

// Ví dụ một app khác \(app\_id = 'crm\_app'\):

\(:crm\_app\_\_Contact  \{ contact\_id: 'C\-001', \.\.\. \}\)

\(:crm\_app\_\_Deal     \{ deal\_id:    'D\-001', \.\.\. \}\)

*ℹ  Platform tự động inject namespace prefix vào mọi Cypher query\. App developer không bao giờ thấy hoặc cần tự gõ prefix này\.*

__Aspect__

__Shared Graph \+ Namespace__

__Separate DB per App__

Cost

Thấp — dùng chung Neo4j cluster

Cao — mỗi app 1 database instance

Isolation

Logical — platform đảm bảo

Physical — Neo4j đảm bảo

Cross\-app query

Có thể enable cho trusted apps

Cần federation layer phức tạp

Backup granularity

Backup theo app phức tạp hơn

Backup từng app độc lập

Phù hợp khi

< 50 apps, mỗi app < 5M nodes

> 50 apps hoặc cần compliance mạnh

# PHẦN III — App Registry & Application Lifecycle

App Registry là nơi quản lý toàn bộ lifecycle của một tenant: từ đăng ký, cấu hình, đến suspend / delete\.

## 3.1 Business Models (GORM) — App Registry

```go
// App represents a client application registered in the KGS platform.
type App struct {
	AppID       string `gorm:"primaryKey;type:varchar(50)"`
	AppName     string `gorm:"type:varchar(200);not null"`
	Description string `gorm:"type:text"`
	Owner       string `gorm:"type:varchar(100);not null"`
	Status      string `gorm:"type:varchar(20);default:'ACTIVE'"` // ACTIVE, INACTIVE, SUSPENDED
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	APIKeys []APIKey `gorm:"foreignKey:AppID"`
	Quotas  []Quota  `gorm:"foreignKey:AppID"`
}

// APIKey represents an authentication key for an App.
type APIKey struct {
	KeyHash   string `gorm:"primaryKey;type:varchar(64)"` // SHA-256 hash of the key
	AppID     string `gorm:"type:varchar(50);not null;index"`
	KeyPrefix string `gorm:"type:varchar(10);not null"` // First few chars for identification
	Name      string `gorm:"type:varchar(100)"`
	Scopes    string `gorm:"type:varchar(500)"` // Comma-separated scopes (e.g., "read,write")
	ExpiresAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// Quota defines rate limits and resource limits for an App.
type Quota struct {
	ID        uint   `gorm:"primaryKey"`
	AppID     string `gorm:"type:varchar(50);not null;uniqueIndex:idx_app_quota_type"`
	QuotaType string `gorm:"type:varchar(50);not null;uniqueIndex:idx_app_quota_type"` // e.g., "requests_per_minute", "max_nodes"
	Limit     int64  `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

## 3.2 App Lifecycle Flow

__Bước__ | __Action__ | __Actor__ | __Result__
---|---|---|---
1. Register | `rpc CreateApp` (POST `/v1/apps`) | Platform Admin | App tạo, AppID được sinh ra
2. Issue API Key | `rpc IssueApiKey` (POST `/v1/apps/{app_id}/keys`) | Platform Admin | API Key trả về một lần (plain), lưu hash
3. Define Ontology | `rpc CreateEntityType` | App Developer | Entity types được register
4. Define Relations | `rpc CreateRelationType`| App Developer | Relation types được register
5. Define Rules | `rpc CreateRule` | App Developer | Rules active, chạy theo schedule/event
6. Define Policies | `rpc CreatePolicy` | App Developer | Access policies active
7. Use Graph API | Graph CRUD operations | App Service/Agent | Nodes/edges được tạo trong namespace
8. Suspend/Delete | Admin update status | Platform Admin | API keys revoked

# PHẦN IV — Ontology Service

Ontology Service cho phép mỗi app tự định nghĩa schema graph của mình\. Platform sẽ dùng ontology này để validate mọi write operation và tự động tạo Neo4j constraints\.

## 4.1 Business Models (GORM) — Ontology

```go
// EntityType defines the schema and constraints for a specific node label in Neo4j.
type EntityType struct {
	ID          uint           `gorm:"primaryKey"`
	AppID       string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_app_entity"`
	Name        string         `gorm:"type:varchar(100);not null;uniqueIndex:idx_app_entity"` // e.g. "Customer", "Transaction"
	Description string         `gorm:"type:text"`
	Schema      datatypes.JSON `gorm:"type:jsonb;not null"` // JSON Schema definition for properties
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// RelationType defines the schema and constraints for a specific edge type in Neo4j.
type RelationType struct {
	ID          uint           `gorm:"primaryKey"`
	AppID       string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_app_relation"`
	Name        string         `gorm:"type:varchar(100);not null;uniqueIndex:idx_app_relation"` // e.g. "PURCHASED", "TRANSFER_TO"
	Description string         `gorm:"type:text"`
	Properties  datatypes.JSON `gorm:"type:jsonb"` // JSON Schema for edge properties (optional)
	SourceTypes datatypes.JSON `gorm:"type:jsonb"` // List of valid source EntityType names
	TargetTypes datatypes.JSON `gorm:"type:jsonb"` // List of valid target EntityType names
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}
```

## 4.2 Ontology API Endpoints

__Method__ | __Endpoint__ | __Mô tả__
---|---|---
POST | `/v1/ontology/entity-types` | Tạo entity type mới
GET | `/v1/ontology/entity-types/{name}` | Lấy chi tiết entity type
POST | `/v1/ontology/relation-types` | Tạo relation type mới
GET | `/v1/ontology/relation-types/{name}` | Lấy chi tiết relation type
GET | `/v1/ontology` | Lấy toàn bộ ontology của app

## 4\.3 Ví dụ: BA Agent System đăng ký Ontology

// POST /apps/ba\_agent/ontology/entity\-types

\{

  "type\_name":      "Requirement",

  "display\_name":   "Software Requirement",

  "id\_property":    "req\_id",

  "searchable\_props": \["title", "description"\],

  "required\_props": \["req\_id", "title", "type", "priority", "status", "version"\],

  "properties": \{

    "req\_id":   \{ "type": "string", "pattern": "^REQ\-\[A\-Z\]\+\-\[0\-9\]\{3\}$" \},

    "title":    \{ "type": "string", "maxLength": 200 \},

    "type":     \{ "type": "string", "enum": \["FUNCTIONAL","NON\_FUNCTIONAL","CONSTRAINT"\] \},

    "priority": \{ "type": "string", "enum": \["MUST","SHOULD","COULD","WONT"\] \},

    "status":   \{ "type": "string", "enum": \["DRAFT","APPROVED","DEPRECATED"\] \},

    "version":  \{ "type": "string" \}

  \}

\}

// POST /apps/ba\_agent/ontology/relation\-types

\{

  "type\_name":    "HAS\_USECASE",

  "from\_types":   \["Requirement"\],

  "to\_types":     \["UseCase"\],

  "cardinality":  "ONE\_TO\_MANY",

  "properties": \{

    "confidence":    \{ "type": "number", "minimum": 0, "maximum": 1 \},

    "impact\_weight": \{ "type": "number", "minimum": 0, "maximum": 1 \},

    "source":        \{ "type": "string", "enum": \["manual","agent","rule\_engine"\] \}

  \}

\}

## 4\.4 Ontology Validation Flow

Mỗi khi App Service gọi Graph API để tạo/cập nhật node hoặc edge, platform thực hiện validation pipeline sau:

Incoming Request

     │

     ▼

1\. Auth: API Key → App Context \(app\_id, scopes\)

     │

     ▼

2\. Namespace Injection: label = '\{app\_id\}\_\_\{type\_name\}'

     │

     ▼

3\. Ontology Lookup: load entity/relation type từ cache \(Redis, TTL=5min\)

     │

     ▼

4\. JSON Schema Validation: validate payload theo properties schema

     │                          └─ 422 nếu fail

     ▼

5\. Relation Whitelist Check: from\_type \+ to\_type trong allowed pairs?

     │                          └─ 403 nếu fail

     ▼

6\. Access Policy Check \(OPA\): role có quyền write entity type này không?

     │                          └─ 403 nếu fail

     ▼

7\. Execute Cypher \(namespaced\)

     │

     ▼

8\. Emit Event → Outbox → Downstream \(Vector DB, Rule Engine\)

# PHẦN V — Graph API

Graph API là interface chính để App Service tương tác với graph\. Tất cả operations đều là namespace\-aware — app chỉ thấy data của chính mình\.

## 5.1 Graph Operations (v1)

__Method__ | __Endpoint__ | __Mô tả__ | __Auth Scope__
---|---|---|---
POST | `/v1/graph/nodes` | Tạo node mới theo ontology | `graph:write`
GET | `/v1/graph/nodes/{node_id}` | Lấy node theo ID | `graph:read`
POST | `/v1/graph/edges` | Tạo edge giữa 2 nodes | `graph:write`
GET | `/v1/graph/nodes/{node_id}/context` | Lấy context subgraph xung quanh node | `graph:read`
GET | `/v1/graph/nodes/{node_id}/impact` | Phân tích tác động (downstream) | `graph:read`
GET | `/v1/graph/nodes/{node_id}/coverage` | Phân tích bao phủ (upstream) | `graph:read`
POST | `/v1/graph/subgraph` | Lấy subgraph cho danh sách node IDs | `graph:read`

## 5.2 Node & Edge Payloads

```json
// CreateNodeRequest
{
  "label": "Requirement",
  "properties_json": "{\"req_id\": \"REQ-001\", \"title\": \"User Login\"}"
}

// CreateEdgeRequest
{
  "source_node_id": "req-uuid-1",
  "target_node_id": "uc-uuid-1",
  "relation_type": "HAS_USECASE",
  "properties_json": "{\"confidence\": 0.9}"
}
```

GET

/apps/\{app\_id\}/coverage

Coverage report: nodes không có target relation

graph:read

POST

/apps/\{app\_id\}/query/explain

Dry\-run query, trả về execution plan \(không write\)

graph:read

## 5\.4 Request / Response Format Chuẩn

// POST /apps/ba\_agent/nodes

// Request:

\{

  "entity\_type": "Requirement",

  "properties": \{

    "req\_id":   "REQ\-AUTH\-001",

    "title":    "Đăng nhập bằng email/password",

    "type":     "FUNCTIONAL",

    "priority": "MUST",

    "status":   "APPROVED",

    "version":  "1\.0\.0"

  \}

\}

// Response 201:

\{

  "node\_id":      "ba\_agent\_\_Requirement\_\_REQ\-AUTH\-001",

  "entity\_type":  "Requirement",

  "app\_id":       "ba\_agent",

  "properties":   \{ \.\.\. \},

  "created\_at":   "2026\-02\-23T10:00:00Z",

  "meta": \{

    "namespace":        "ba\_agent",

    "neo4j\_label":      "ba\_agent\_\_Requirement",

    "validation\_passed": true

  \}

\}

// POST /apps/ba\_agent/edges

// Request:

\{

  "relation\_type": "HAS\_USECASE",

  "from\_node\_id":  "ba\_agent\_\_Requirement\_\_REQ\-AUTH\-001",

  "to\_node\_id":    "ba\_agent\_\_UseCase\_\_UC\-001",

  "properties": \{

    "confidence":    1\.0,

    "impact\_weight": 0\.9,

    "source":        "manual"

  \}

\}

## 5\.5 Safe Query Interface

App có thể gửi query linh hoạt hơn qua POST /query nhưng bị giới hạn bởi whitelist functions để đảm bảo an toàn:

// POST /apps/ba\_agent/query

\{

  "query\_type": "FIND\_NODES",         // whitelist: FIND\_NODES | FIND\_PATH | AGGREGATE

  "filters": \{

    "entity\_type": "Requirement",

    "properties": \{ "status": "APPROVED", "priority": "MUST" \}

  \},

  "order\_by":    "created\_at",

  "limit":       50,

  "offset":      0

\}

*⚠  App không được phép gửi raw Cypher\. Mọi query đều phải qua Query Planner để inject namespace và enforce guardrails \(max\_depth=4, max\_nodes=500\)\.*

# PHẦN VI — Rule Engine Service

Rule Engine là async service chạy các Cypher rules theo schedule hoặc event-driven qua Redis Streams.

## 6.1 Business Models (GORM) — Rules

```go
// Rule represents a business rule that runs either on a schedule or triggered by events.
type Rule struct {
	ID          uint           `gorm:"primaryKey"`
	AppID       string         `gorm:"type:varchar(50);not null;index:idx_app_rule"`
	Name        string         `gorm:"type:varchar(100);not null"`
	Description string         `gorm:"type:text"`
	TriggerType string         `gorm:"type:varchar(20);not null"` // e.g., "SCHEDULED", "ON_WRITE"
	Cron        string         `gorm:"type:varchar(50)"`          // e.g., "0 0 * * *"
	CypherQuery string         `gorm:"type:text;not null"`        // Cypher query to execute
	Action      string         `gorm:"type:varchar(50)"`          // webhook, push_notification
	Payload     datatypes.JSON `gorm:"type:jsonb"`                // Action payload
	IsActive    bool           `gorm:"default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// RuleExecution tracks the history of rule executions.
type RuleExecution struct {
	ID        uint      `gorm:"primaryKey"`
	AppID     string    `gorm:"type:varchar(50);not null;index:idx_app_ex"`
	RuleID    uint      `gorm:"not null"`
	Status    string    `gorm:"type:varchar(20);not null"` // SUCCESS, FAILED
	Message   string    `gorm:"type:text"`
	StartedAt time.Time `gorm:"index"`
	EndedAt   time.Time
}
```

## 6.2 Rule API Endpoints

__Method__ | __Endpoint__ | __Mô tả__
---|---|---
POST | `/v1/rules` | Tạo rule mới
GET | `/v1/rules/{id}` | Lấy chi tiết rule
GET | `/v1/rules` | List rules theo app_id

# PHẦN VII — Access Control (ABAC)

KGS dùng Policy-Based Access Control với OPA. Mỗi app quản lý tập hợp policies bằng Rego.

## 7.1 Business Models (GORM) — Policy

```go
// Policy defines OPA Rego policies managed via the database.
type Policy struct {
	ID          uint   `gorm:"primaryKey"`
	AppID       string `gorm:"type:varchar(50);not null;index:idx_app_policy"`
	Name        string `gorm:"type:varchar(100);not null"`
	Description string `gorm:"type:text"`
	RegoContent string `gorm:"type:text;not null"`
	IsActive    bool   `gorm:"default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}
```

## 7.2 Policy API Endpoints

__Method__ | __Endpoint__ | __Mô tả__
---|---|---
POST | `/v1/policies` | Tạo policy mới (Rego)
GET | `/v1/policies` | List policies
GET | `/v1/policies/{id}` | Lấy chi tiết policy

# PHẦN VIII — Custom API Schema & Response Mapping (Planned)

> [!NOTE]
> Tính năng "Views" giúp các ứng dụng tùy biến format dữ liệu trả về từ Graph API. Hiện đang trong giai đoạn thiết kế và chưa được implement trong core platform.

## 8.1 Concept: View Definition
Mục tiêu là cho phép app định nghĩa một "view" (whitelist fields, renaming, embedding relations) để giảm tải việc transform dữ liệu ở phía client.

## 8.2 Proposed View Schema
Dự kiến view definition sẽ bao gồm:
- **Field Mapping**: Map giữa Neo4j property và output JSON key.
- **Relation Embedding**: Tự động lồng các nodes liên quan vào JSON (vd: lồng danh sách UseCases vào Requirement detail).
- **Computed Fields**: Các trường tính toán đơn giản từ dữ liệu có sẵn.

# PHẦN IX — Namespace Isolation: Cơ Chế Nội Tại

Phần này mô tả chi tiết cách platform đảm bảo isolation — quan trọng cho cả security lẫn correctness.

## 9.1 Query Planner: Namespace Injection (Go)

KGS tự động inject namespace vào mọi label trong Neo4j. App developer chỉ cần làm việc với raw labels (vd: `Requirement`), system sẽ tự động map sang `{APP_ID}__{Requirement}`.

```go
// query_planner.go (logic conceptual)
func (qp *QueryPlanner) Namespace(entityType string) string {
    return fmt.Sprintf("%s__%s", qp.appID, entityType)
}

// Khi tạo node:
// Cypher: CREATE (n:ba_agent__Requirement {req_id: $id, ...})
```

## 9.2 Isolation Guarantees

__Threat__ | __Mechanism Bảo Vệ__ | __Layer__
---|---|---
App A đọc data App B | Namespace label filter trong mọi query | Query Planner
App A tạo node với label của App B | Label được inject bởi platform, client không set | Graph Service
App A gửi raw Cypher chứa label khác | Raw Cypher không được phép, chỉ whitelist operations | KGS Service
API Key của App A dùng app_id của App B | API Key hash → app_id lookup trong Registry | Auth Middleware
App A traverse edge sang App B | labelFilter trong traversal query (APOC) giới hạn prefix | Query Planner
Ontology trùng tên type | Type name được prefixed duy nhất trong Neo4j | Ontology Service

# PHẦN X — Onboarding Flow: BA Agent System

Đây là walkthrough đầy đủ để onboarding một tenant (vd: BA Agent System) lên KGS Platform.

## Step 1 — Đăng ký Application
POST `/v1/apps`
```json
{
  "app_name": "BA Agent System",
  "description": "Multi-agent system cho IEEE traceability",
  "owner": "ba-team@example.com"
}
```

## Step 2 — Tạo API Key
POST `/v1/apps/{app_id}/keys`
```json
{ "name": "production", "scopes": "read,write" }
```
*Lưu plain API key ngay lập tức.*

## Step 3 — Khai báo Ontology
POST `/v1/ontology/entity-types` (Requirement, UseCase, etc.)
POST `/v1/ontology/relation-types` (HAS_USECASE, etc.)

## Step 4 — Đăng ký Rules & Policies
POST `/v1/rules` (Custom Cypher rules)
POST `/v1/policies` (Rego policies)

## Step 5 — Verify & Sử dụng
- Kiểm tra ontology: GET `/v1/ontology`
- Tạo node: POST `/v1/graph/nodes`
- Lấy context: GET `/v1/graph/nodes/{id}/context`

# PHẦN XI — Implementation Roadmap

Dự án hiện đang ở giai đoạn hoàn thiện core Graph Service và Rule Engine.

- [x] **Phase 1: Foundation (Kratos + PostgreSQL)**
- [x] **Phase 2: Graph Core (Neo4j + Namespace logic)**
- [x] **Phase 3: Rule Runner (Async execution via Redis)**
- [/] **Phase 4: Policy Sync (OPA integration)**
- [ ] **Phase 5: Response Views & Analytics**

# PHẦN XII — Quyết Định Thiết Kế & Trade-offs

__Quyết định__ | __Lựa chọn__ | __Lý do__ | __Trade-off__
---|---|---|---
Multi-tenancy | Shared graph + namespace | Tối ưu chi phí Neo4j; dễ migrate sang separate DB sau | Isolation là logical, không physical
Raw Cypher cho app | KHÔNG cho phép | Bảo mật namespace; tránh app bypass guardrails | App mất flexibility, phải dùng query builder
Ontology storage | PostgreSQL (không dùng Neo4j) | Ontology là config, cần ACID; Neo4j cho graph data | Thêm 1 store cần sync
Rule execution | Async (queue-based) | Không block Graph API; dễ retry/scale | Delay giữa event và rule execution
Access Control engine | OPA (Open Policy Agent) | Mature, auditable, Rego language linh hoạt | Cần maintain OPA bundle sync
API Key auth | Hash stored, prefix visible | Không lưu plain key; prefix giúp user identify key | Không thể recover key nếu mất
Response Views | Server-side mapping | App không cần transform data phía client | Thêm latency nhỏ cho view resolution

─────────────────────────────────────────────────

**END OF DOCUMENT — KGS Platform Architecture v1.1 (Updated Feb 2026)**

*BA Agent System là reference tenant đầu tiên được build trên nền tảng này.*

