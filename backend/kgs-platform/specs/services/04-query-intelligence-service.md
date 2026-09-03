# query-intel-service — Query & Intelligence Service

> **Role:** Xử lý toàn bộ truy vấn thông minh trên graph: Query Planner, Context/Impact/Coverage queries, Analytics, và Projection Engine. Đây là read-path chuyên biệt của KGS Platform.

---

## 1. Trách Nhiệm (Single Responsibility)

`query-intel-service` chịu trách nhiệm **duy nhất** cho:
- **Query Planning**: Translate generic queries → namespaced Cypher queries
- **Graph Traversal Queries**: Context, Impact, Coverage, Subgraph queries (Neo4j)
- **Safe Query Interface**: Whitelist-based query builder (không cho phép raw Cypher)
- **Analytics Engine**: Coverage report, Traceability matrix, Cluster analysis
- **Projection Engine**: Role-based field filtering, PII masking, View definitions
- **Guardrails**: Enforce max_depth=10, max_nodes=10000 per query

---

## 2. Kiến Trúc Nội Tại

```
┌─────────────────────────────────────────────────────────────────────────┐
│                       query-intel-service                                │
│                                                                         │
│  gRPC Server (port 9004)                                                │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                  QueryIntelServiceServer                          │   │
│  │                                                                  │   │
│  │  GetContext()         GetImpact()          GetCoverage()         │   │
│  │  GetSubgraph()        ExecuteQuery()                             │   │
│  │  GetCoverageReport()  GetTraceabilityMatrix()  ClusterAnalysis() │   │
│  │  CreateView()         GetView()            ListViews()           │   │
│  │  DeleteView()         ResolveView()                              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                               │                                          │
│  ┌────────────────────────────▼────────────────────────────────────┐   │
│  │                   Intelligence Pipeline                           │   │
│  │                                                                  │   │
│  │  ┌──────────────────┐  ┌────────────────┐  ┌─────────────────┐  │   │
│  │  │  Query Planner   │  │ Analytics Eng  │  │ Projection Eng  │  │   │
│  │  │                  │  │                │  │                  │  │   │
│  │  │ BuildContext()   │  │ CoverageReport │  │ ViewResolver    │  │   │
│  │  │ BuildImpact()    │  │ Traceability   │  │ FieldFilter     │  │   │
│  │  │ BuildCoverage()  │  │ ClusterAnalysis│  │ PIIMasking      │  │   │
│  │  │ BuildSubgraph()  │  │                │  │                  │  │   │
│  │  │ Guardrails       │  │                │  │                  │  │   │
│  │  └──────────────────┘  └────────────────┘  └─────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                               │                                          │
│  ┌────────────────────────────▼────────────────────────────────────┐   │
│  │                   Data Access                                     │   │
│  │  Neo4j:      Graph traversal (context, impact, coverage, cluster)│   │
│  │  PostgreSQL: Full-text node lookup, view definitions             │   │
│  │  (calls ontology-service for schema info)                        │   │
│  │  (calls policy-service for field-level access check)             │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Query Planner

### 3.1 Namespace Injection

```go
// Mọi query đều được inject namespace tự động
func (qp *QueryPlanner) Namespace(entityType string) string {
    return fmt.Sprintf("%s__%s", qp.appID, entityType)
}

// Ví dụ: app_id="ba_agent", entity_type="Requirement"
// → Neo4j label: "ba_agent__Requirement"
```

### 3.2 Context Query (k-hop neighborhood)

```go
// GetContext: Lấy subgraph xung quanh một node (k hops)
// Input: node_id, direction (INCOMING | OUTGOING | BOTH), depth

// Cypher được generate:
MATCH path = (start:`{app_id}__Requirement` {id: $node_id})
  -[*1..{depth}]->
  (neighbor)
WHERE all(n in nodes(path) WHERE any(l in labels(n) WHERE l STARTS WITH '{app_id}__'))
RETURN path
LIMIT {max_nodes}
```

### 3.3 Impact Query (downstream traversal)

```go
// GetImpact: Lấy tất cả nodes bị ảnh hưởng downstream từ một node
// Use case: "Nếu REQ-001 thay đổi, những UseCase nào bị ảnh hưởng?"

MATCH (start:`{app_id}__Requirement` {id: $node_id})
  -[rel:HAS_USECASE|IMPLEMENTS|VALIDATES*1..{depth}]->
  (downstream)
WHERE any(l in labels(downstream) WHERE l STARTS WITH '{app_id}__')
RETURN downstream, rel
ORDER BY rel.impact_weight DESC
LIMIT {max_nodes}
```

### 3.4 Coverage Query (upstream traceability)

```go
// GetCoverage: Lấy tất cả nodes upstream (nguồn gốc) của một node
// Use case: "TestCase TC-001 test những Requirement nào?"

MATCH (start:`{app_id}__TestCase` {id: $node_id})
  <-[rel:VALIDATES|TESTS_REQUIREMENT*1..{depth}]-
  (upstream)
WHERE any(l in labels(upstream) WHERE l STARTS WITH '{app_id}__')
RETURN upstream, rel
```

### 3.5 Guardrails

```go
const (
    MaxAllowedDepth = 10    // Tối đa 10 hops
    MaxAllowedNodes = 10000 // Tối đa 10,000 nodes trả về
    MaxSubgraphIDs  = 100   // Tối đa 100 node IDs cho subgraph query
)

// Nếu vượt guardrails → 400 Bad Request với thông báo rõ ràng
```

---

## 4. Safe Query Interface

App có thể gửi query linh hoạt nhưng phải dùng whitelist operations:

```json
// POST /v1/query
{
  "query_type": "FIND_NODES",        // Whitelist: FIND_NODES | FIND_PATH | AGGREGATE
  "filters": {
    "entity_type": "Requirement",
    "properties": {
      "status": "APPROVED",
      "priority": "MUST"
    }
  },
  "order_by":    "created_at",
  "limit":       50,
  "offset":      0
}

// Supported query_types:
// FIND_NODES:  Tìm nodes theo filter
// FIND_PATH:   Tìm path giữa 2 nodes
// AGGREGATE:   Count/sum theo entity type
```

> ⚠ App **không được** gửi raw Cypher. Mọi query phải qua Query Planner để inject namespace và enforce guardrails.

---

## 5. Analytics Engine

### 5.1 Coverage Report

```
GET /v1/analytics/coverage?entity_type=Requirement

Response:
{
  "entity_type": "Requirement",
  "total_count": 150,
  "covered_count": 120,       // Có ít nhất 1 edge đến TestCase
  "uncovered_count": 30,      // Không có edge nào
  "coverage_rate": 0.80,
  "uncovered_nodes": [
    { "node_id": "ba_agent__Requirement__REQ-045", "title": "..." },
    ...
  ]
}
```

**Cypher:**
```cypher
MATCH (r:`ba_agent__Requirement`)
OPTIONAL MATCH (r)-[:VALIDATES]-(t:`ba_agent__TestCase`)
RETURN r.req_id, count(t) as test_count
```

### 5.2 Traceability Matrix

```
GET /v1/analytics/traceability?source_type=Requirement&target_type=TestCase

Response:
{
  "source_type": "Requirement",
  "target_type": "TestCase",
  "matrix": [
    {
      "source_id": "REQ-001",
      "source_title": "User Login",
      "targets": [
        { "target_id": "TC-001", "relation": "VALIDATES", "confidence": 0.95 }
      ]
    },
    ...
  ]
}
```

### 5.3 Cluster Analysis

```
POST /v1/analytics/clusters

{
  "algorithm": "LOUVAIN",     // LOUVAIN | LABEL_PROPAGATION | WEAKLY_CONNECTED
  "min_cluster_size": 3
}

// Sử dụng Neo4j Graph Data Science algorithms:
CALL gds.louvain.stream('ba_agent_graph')
YIELD nodeId, communityId
```

---

## 6. Projection Engine

### 6.1 View Definition

```go
type ViewDefinition struct {
    ID           uint           `gorm:"primaryKey"`
    AppID        string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_app_view"`
    Name         string         `gorm:"type:varchar(100);not null;uniqueIndex:idx_app_view"`
    EntityType   string         `gorm:"type:varchar(100);not null"`
    AllowedRoles datatypes.JSON `gorm:"type:jsonb"` // ["developer", "manager"]
    Fields       datatypes.JSON `gorm:"type:jsonb"`
    // {
    //   "req_id": { "alias": "id" },
    //   "title": {},
    //   // "internal_notes": OMITTED (not in list = hidden)
    // }
    PIIFields    datatypes.JSON `gorm:"type:jsonb"` // ["email", "phone"] → auto-mask
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### 6.2 View Resolution Pipeline

```
Request: GET /v1/graph/nodes/ba_agent__Requirement__REQ-001
         + Headers: x-view: developer_view

QueryIntelService:
  1. Load ViewDefinition "developer_view" from cache/DB
  2. Check: request role in view.AllowedRoles?
  3. Filter fields: only return fields in view.Fields
  4. Apply alias: rename fields per view.Fields config
  5. PII Masking: mask fields in view.PIIFields
     - email "user@example.com" → "u***@e***.com"
     - phone "0901234567" → "090*****67"
  6. Return projected response
```

---

## 7. gRPC API

```protobuf
service QueryIntelService {
  // Graph Traversal Queries
  rpc GetContext(GetContextRequest) returns (SubgraphResponse);
  rpc GetImpact(GetImpactRequest) returns (SubgraphResponse);
  rpc GetCoverage(GetCoverageRequest) returns (SubgraphResponse);
  rpc GetSubgraph(GetSubgraphRequest) returns (SubgraphResponse);

  // Safe Query
  rpc ExecuteQuery(ExecuteQueryRequest) returns (QueryResponse);

  // Analytics
  rpc GetCoverageReport(CoverageReportRequest) returns (CoverageReportResponse);
  rpc GetTraceabilityMatrix(TraceabilityRequest) returns (TraceabilityResponse);
  rpc ClusterAnalysis(ClusterRequest) returns (ClusterResponse);

  // View Management
  rpc CreateView(CreateViewRequest) returns (ViewDefinition);
  rpc GetView(GetViewRequest) returns (ViewDefinition);
  rpc ListViews(ListViewsRequest) returns (ListViewsResponse);
  rpc DeleteView(DeleteViewRequest) returns (google.protobuf.Empty);
  rpc ResolveView(ResolveViewRequest) returns (ResolveViewResponse);
}

message GetContextRequest {
  string node_id = 1;
  string direction = 2;          // INCOMING | OUTGOING | BOTH
  int32 depth = 3;               // Max depth (guardrail: max 10)
  repeated string relation_types = 4; // Optional filter
}

message SubgraphResponse {
  repeated Node nodes = 1;
  repeated Edge edges = 2;
  int32 total_nodes = 3;
  bool truncated = 4;            // True if hit guardrail limit
}
```

---

## 8. HTTP REST Endpoints (Exposed qua Gateway)

| Method | Path | Scope | Mô tả |
|--------|------|-------|-------|
| GET | `/v1/graph/nodes/:id/context` | `graph:read` | Context subgraph |
| GET | `/v1/graph/nodes/:id/impact` | `graph:read` | Downstream impact |
| GET | `/v1/graph/nodes/:id/coverage` | `graph:read` | Upstream coverage |
| POST | `/v1/graph/subgraph` | `graph:read` | Subgraph từ list node IDs |
| POST | `/v1/query` | `graph:read` | Safe query interface |
| GET | `/v1/analytics/coverage` | `analytics:read` | Coverage report |
| GET | `/v1/analytics/traceability` | `analytics:read` | Traceability matrix |
| POST | `/v1/analytics/clusters` | `analytics:read` | Cluster analysis |
| POST | `/v1/views` | `ontology:write` | Tạo view definition |
| GET | `/v1/views` | `ontology:read` | List views |
| GET | `/v1/views/:name` | `ontology:read` | Lấy view definition |
| DELETE | `/v1/views/:name` | `ontology:write` | Xóa view |

---

## 9. Configuration

```yaml
# configs/query-intel.yaml
query_intel_service:
  grpc_port: 9004

  neo4j:
    uri: bolt://neo4j:7687
    username: neo4j
    password: secret
    max_connection_pool_size: 50

  database:
    dsn: "postgres://kgs:password@postgres:5432/kgs_graph"

  guardrails:
    max_depth: 10
    max_nodes: 10000
    max_subgraph_ids: 100
    query_timeout: 30s

  dependencies:
    ontology_service: ontology-service:9002
    policy_service: policy-service:9006

  observability:
    metrics_port: 9094
```

---

## 10. Observability

| Metric | Mô tả |
|--------|-------|
| `query_intel_requests_total{query_type, app_id}` | Số queries theo loại |
| `query_intel_duration_seconds{query_type}` | Latency histogram |
| `query_intel_guardrail_hit_total{rule}` | Số lần bị guardrail chặn |
| `query_intel_nodes_returned_total{app_id}` | Số nodes trả về |
| `query_intel_analytics_duration_seconds{report_type}` | Analytics latency |
