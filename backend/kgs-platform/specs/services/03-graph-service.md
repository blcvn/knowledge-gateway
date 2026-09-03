# graph-service — Graph Core Service

> **Role:** CRUD cho nodes và edges trong Knowledge Graph với namespace isolation hoàn toàn. Đây là "tim" của KGS Platform.

---

## 1. Trách Nhiệm (Single Responsibility)

`graph-service` chịu trách nhiệm **duy nhất** cho:
- **Node CRUD**: Tạo, đọc, cập nhật, xóa nodes với namespace isolation
- **Edge CRUD**: Tạo, đọc, xóa edges với validation whitelist
- **Namespace Injection**: Tự động prefix `{app_id}__{entity_type}` cho mọi operation
- **Schema Validation**: Gọi ontology-service để validate trước khi write
- **Access Check**: Gọi policy-service để enforce ABAC policies
- **Outbox Emit**: Sau mỗi write, emit event vào KGSyncOutbox để async sync sang Neo4j/Qdrant

---

## 2. Kiến Trúc Nội Tại

```
┌─────────────────────────────────────────────────────────────────────┐
│                         graph-service                                │
│                                                                     │
│  gRPC Server (port 9003)                                            │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                  GraphServiceServer                           │   │
│  │                                                              │   │
│  │  CreateNode()   GetNode()    UpdateNode()   DeleteNode()     │   │
│  │  CreateEdge()   GetEdge()    DeleteEdge()                    │   │
│  │  BatchCreateNodes()   BatchCreateEdges()                     │   │
│  │  GetNodesByType()                                            │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                       │
│  ┌───────────────────────────▼─────────────────────────────────┐   │
│  │                  GraphUsecase (Write Pipeline)                │   │
│  │                                                              │   │
│  │  1. Namespace computation                                    │   │
│  │     namespace = "graph/{app_id}/{tenant_id}"                 │   │
│  │     neo4j_label = "{app_id}__{entity_type}"                  │   │
│  │  2. Ontology validation (gRPC → ontology-service)           │   │
│  │  3. Policy check (gRPC → policy-service)                    │   │
│  │  4. Write to PostgreSQL (source-of-truth)                   │   │
│  │  5. Write KGSyncOutbox record                               │   │
│  │  6. Publish NATS event (for rule-engine triggers)           │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                       │
│  ┌───────────────────────────▼─────────────────────────────────┐   │
│  │                  Data Layer                                   │   │
│  │  PostgreSQL: kg_entities, kg_edges, kg_sync_outbox           │   │
│  │  NATS:       event publish (graph.node.created, etc.)        │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
         │                          │
         ▼ gRPC call                ▼ gRPC call
 ontology-service             policy-service
```

---

## 3. Data Models

### 3.1 KGEntity (PostgreSQL — Source of Truth)

```go
type KGEntity struct {
    ID             string         `gorm:"primaryKey;type:varchar(100)"`
    // Format: "{app_id}__{EntityType}__{id_value}"
    // Example: "ba_agent__Requirement__REQ-AUTH-001"
    AppID          string         `gorm:"type:varchar(50);not null;index:idx_app_type"`
    TenantID       string         `gorm:"type:varchar(100);not null;index:idx_app_type"`
    Namespace      string         `gorm:"type:varchar(200);not null;index"`
    // Format: "graph/{app_id}/{tenant_id}"
    EntityType     string         `gorm:"type:varchar(100);not null;index:idx_app_type"`
    // Raw type name, e.g. "Requirement"
    Neo4jLabel     string         `gorm:"type:varchar(200);not null"`
    // Namespaced label for Neo4j, e.g. "ba_agent__Requirement"
    PropertiesJSON datatypes.JSON `gorm:"type:jsonb;not null"`
    // All node properties
    Version        int            `gorm:"default:1"`
    CreatedAt      time.Time
    UpdatedAt      time.Time
    DeletedAt      gorm.DeletedAt `gorm:"index"`
}
```

### 3.2 KGEdge (PostgreSQL — Source of Truth)

```go
type KGEdge struct {
    ID             string         `gorm:"primaryKey;type:varchar(100)"`
    // Format: UUID
    AppID          string         `gorm:"type:varchar(50);not null;index"`
    TenantID       string         `gorm:"type:varchar(100);not null"`
    Namespace      string         `gorm:"type:varchar(200);not null;index"`
    RelationType   string         `gorm:"type:varchar(100);not null"`
    // e.g. "HAS_USECASE"
    SourceEntityID string         `gorm:"type:varchar(100);not null;index"`
    TargetEntityID string         `gorm:"type:varchar(100);not null;index"`
    PropertiesJSON datatypes.JSON `gorm:"type:jsonb"`
    Version        int            `gorm:"default:1"`
    CreatedAt      time.Time
    UpdatedAt      time.Time
    DeletedAt      gorm.DeletedAt `gorm:"index"`
}
```

### 3.3 KGSyncOutbox

```go
type KGSyncOutbox struct {
    ID         uint      `gorm:"primaryKey"`
    AppID      string    `gorm:"type:varchar(50);not null;index"`
    EntityID   string    `gorm:"type:varchar(100);index"` // null for edges
    EdgeID     string    `gorm:"type:varchar(100);index"` // null for nodes
    Operation  string    `gorm:"type:varchar(20);not null"`
    // UPSERT_ENTITY | UPSERT_EDGE | DELETE_ENTITY | DELETE_EDGE
    Payload    datatypes.JSON `gorm:"type:jsonb;not null"`
    Attempts   int       `gorm:"default:0"`
    Status     string    `gorm:"type:varchar(20);default:'PENDING'"`
    // PENDING | PROCESSING | DONE | FAILED
    Error      string    `gorm:"type:text"`
    CreatedAt  time.Time `gorm:"index"`
    ProcessedAt *time.Time
}
```

---

## 4. Write Pipeline (CreateNode)

```
Request: CreateNode(app_id="ba_agent", entity_type="Requirement", properties={...})

Step 1: Namespace Injection
  namespace  = "graph/ba_agent/tenant_001"
  neo4j_label = "ba_agent__Requirement"
  entity_id  = "ba_agent__Requirement__REQ-AUTH-001"

Step 2: Ontology Validation (gRPC)
  → ontology-service.ValidateNodeProperties(app_id, "Requirement", properties)
  ← ValidationResult{valid: true}
  [If invalid → return HTTP 422]

Step 3: Policy Check (gRPC)
  → policy-service.Evaluate(app_id, action="graph:write", resource="Requirement", context={...})
  ← PolicyResult{allow: true}
  [If denied → return HTTP 403]

Step 4: PostgreSQL Write (transaction)
  BEGIN TRANSACTION
    INSERT INTO kg_entities (id, app_id, tenant_id, namespace, entity_type, ...)
    INSERT INTO kg_sync_outbox (app_id, entity_id, operation="UPSERT_ENTITY", payload={...})
  COMMIT

Step 5: NATS Publish (async, after transaction)
  Publish("graph.node.created", {app_id, entity_id, entity_type, ...})
  [Rule Engine listens for ON_WRITE triggers]

Step 6: Return Response
  {
    "node_id": "ba_agent__Requirement__REQ-AUTH-001",
    "entity_type": "Requirement",
    "app_id": "ba_agent",
    "properties": {...},
    "created_at": "2026-06-11T...",
    "meta": {
      "namespace": "graph/ba_agent/tenant_001",
      "neo4j_label": "ba_agent__Requirement",
      "validation_passed": true
    }
  }
```

---

## 5. gRPC API

```protobuf
service GraphService {
  // Node operations
  rpc CreateNode(CreateNodeRequest) returns (Node);
  rpc GetNode(GetNodeRequest) returns (Node);
  rpc UpdateNode(UpdateNodeRequest) returns (Node);
  rpc DeleteNode(DeleteNodeRequest) returns (google.protobuf.Empty);
  rpc GetNodesByType(GetNodesByTypeRequest) returns (GetNodesByTypeResponse);
  rpc BatchCreateNodes(BatchCreateNodesRequest) returns (BatchCreateNodesResponse);

  // Edge operations
  rpc CreateEdge(CreateEdgeRequest) returns (Edge);
  rpc GetEdge(GetEdgeRequest) returns (Edge);
  rpc DeleteEdge(DeleteEdgeRequest) returns (google.protobuf.Empty);
  rpc GetEdgesByRelationType(GetEdgesByRelationTypeRequest) returns (GetEdgesByRelationTypeResponse);
  rpc BatchCreateEdges(BatchCreateEdgesRequest) returns (BatchCreateEdgesResponse);

  // Internal read (called by query-intel-service)
  rpc GetEntityMetadata(GetEntityMetadataRequest) returns (EntityMetadata);
}

message CreateNodeRequest {
  string entity_type = 1;          // "Requirement" (no namespace prefix)
  bytes properties_json = 2;
  // app_id, tenant_id từ gRPC metadata (injected by gateway)
}

message Node {
  string node_id = 1;              // "ba_agent__Requirement__REQ-001"
  string entity_type = 2;
  string app_id = 3;
  string namespace = 4;
  bytes properties_json = 5;
  string neo4j_label = 6;
  bool validation_passed = 7;
  google.protobuf.Timestamp created_at = 8;
  google.protobuf.Timestamp updated_at = 9;
}

message CreateEdgeRequest {
  string source_node_id = 1;
  string target_node_id = 2;
  string relation_type = 3;        // "HAS_USECASE"
  bytes properties_json = 4;
}
```

---

## 6. Namespace Isolation Guarantees

| Threat | Cơ chế bảo vệ |
|--------|--------------|
| App A đọc node của App B | `WHERE app_id = {injected_app_id}` trong mọi query |
| App A tạo node với label của App B | `neo4j_label` được inject bởi platform, không do client set |
| App A traverse sang namespace khác | Outbox sync chỉ dùng `{app_id}__` prefix trong Cypher |
| API Key của App A giả mạo App B | `app_id` lấy từ validated token context, không từ request body |

---

## 7. HTTP REST Endpoints (Exposed qua Gateway)

| Method | Path | Scope | Mô tả |
|--------|------|-------|-------|
| POST | `/v1/graph/nodes` | `graph:write` | Tạo node mới |
| GET | `/v1/graph/nodes/:id` | `graph:read` | Lấy node theo ID |
| PUT | `/v1/graph/nodes/:id` | `graph:write` | Update node properties |
| DELETE | `/v1/graph/nodes/:id` | `graph:write` | Xóa node (soft delete) |
| GET | `/v1/graph/nodes` | `graph:read` | List nodes theo entity type + filter |
| POST | `/v1/graph/nodes/batch` | `graph:write` | Batch create nodes |
| POST | `/v1/graph/edges` | `graph:write` | Tạo edge mới |
| GET | `/v1/graph/edges/:id` | `graph:read` | Lấy edge theo ID |
| DELETE | `/v1/graph/edges/:id` | `graph:write` | Xóa edge |

---

## 8. NATS Events Published

| Event | Topic | Consumers |
|-------|-------|-----------|
| NodeCreated | `graph.node.created` | rule-engine-service (ON_WRITE trigger) |
| NodeUpdated | `graph.node.updated` | rule-engine-service |
| NodeDeleted | `graph.node.deleted` | sync-worker-service |
| EdgeCreated | `graph.edge.created` | rule-engine-service |
| EdgeDeleted | `graph.edge.deleted` | sync-worker-service |

---

## 9. Batch Operations

```json
// POST /v1/graph/nodes/batch
{
  "nodes": [
    {
      "entity_type": "Requirement",
      "properties": { "req_id": "REQ-001", "title": "...", ... }
    },
    {
      "entity_type": "UseCase",
      "properties": { "uc_id": "UC-001", "title": "...", ... }
    }
  ],
  "options": {
    "on_conflict": "UPSERT",  // UPSERT | SKIP | FAIL
    "validate_all_first": true // Validate toàn bộ trước khi write bất kỳ record nào
  }
}

// Response:
{
  "created": 2,
  "updated": 0,
  "skipped": 0,
  "failed": 0,
  "results": [...]
}
```

---

## 10. Configuration

```yaml
# configs/graph.yaml
graph_service:
  grpc_port: 9003

  database:
    dsn: "postgres://kgs:password@postgres:5432/kgs_graph"
    max_open_conns: 50

  nats:
    addr: nats:4222
    subjects:
      node_created: graph.node.created
      edge_created: graph.edge.created

  dependencies:
    ontology_service: ontology-service:9002
    policy_service: policy-service:9006

  batch:
    max_batch_size: 1000
    validate_all_first: true

  observability:
    metrics_port: 9093
```

---

## 11. Observability

| Metric | Mô tả |
|--------|-------|
| `graph_nodes_created_total{app_id, entity_type}` | Số nodes tạo mới |
| `graph_edges_created_total{app_id, relation_type}` | Số edges tạo mới |
| `graph_write_validation_failed_total{app_id}` | Số lần validation fail |
| `graph_write_policy_denied_total{app_id}` | Số lần policy deny |
| `graph_write_duration_seconds` | Latency của write pipeline |
| `graph_outbox_pending_total{app_id}` | Số outbox records đang chờ sync |
