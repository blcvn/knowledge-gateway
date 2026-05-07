---
doc_id: KGS-SPEC-004
version: 1.0.0
status: Active
last_updated: 2026-04-24
scope: SERVICE-LEVEL
---

# KGS Platform — AI Integration Guide

> **Mục đích:** Hướng dẫn AI agent tạo code tích hợp với `kgs-platform`.  
> **Module:** `kgs-platform` (Go 1.24 / Kratos v2)  
> **Endpoints:** HTTP `:8000` · gRPC `:9000`

---

## 1. Kiến trúc tổng quan

```
Client (service tích hợp)
    │
    ├── gRPC :9000  →  GraphService, RulesService, PolicyService, OntologyService
    └── HTTP :8000  →  REST endpoints (OpenAPI spec: openapi.yaml)
                           │
                    ┌──────▼──────┐
                    │  Middleware │  Auth(API Key) → RateLimiter
                    └──────┬──────┘
                    ┌──────▼──────┐
                    │   Service   │  internal/service/* (proto ↔ domain)
                    └──────┬──────┘
                    ┌──────▼──────┐
                    │     Biz     │  OPA check → Neo4j write → Redis event
                    └──────┬──────┘
                    ┌──────▼──────┐
                    │    Data     │  Neo4j · PostgreSQL · Redis · Qdrant
                    └─────────────┘
```

**5 tầng bất biến — mọi thay đổi phải tuân thủ:**

| Tầng | Package | Trách nhiệm |
|------|---------|-------------|
| Transport | `internal/server/` | HTTP/gRPC decode, middleware |
| Service | `internal/service/` | Proto ↔ domain mapping |
| Biz | `internal/biz/` | Business logic, OPA, events |
| Data | `internal/data/` | Neo4j, PostgreSQL, Redis, Qdrant |
| Worker | `internal/server/worker.go` | Background jobs (cron, stream, kafka) |

---

## 2. Authentication

Mọi request phải gửi API Key qua một trong hai headers:

```
Authorization: <api-key>
X-API-Key: <api-key>
```

Với gRPC: gửi `app_id` qua metadata header `x-kgs-app-id`:

```go
ctx = metadata.AppendToOutgoingContext(ctx, "x-kgs-app-id", appID)
```

**Quan trọng — `app_id` injection pattern:**  
Service layer (`internal/service/graph.go`) đọc `app_id` theo thứ tự ưu tiên:
1. `x-kgs-app-id` gRPC metadata header
2. `app_id` field trong `properties_json` body (doc_to_kg pattern)
3. Fallback: `"system"`

```go
// internal/service/graph.go — extractAppID
func extractAppID(ctx context.Context) string {
    md, ok := metadata.FromIncomingContext(ctx)
    if ok {
        if vals := md.Get("x-kgs-app-id"); len(vals) > 0 && vals[0] != "" {
            return vals[0]
        }
    }
    return "system"
}
```

---

## 3. Tích hợp qua gRPC

### 3.1 Setup client

```go
import (
    pb "kgs-platform/api/graph/v1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/metadata"
)

func NewKGSClient(addr string) (pb.GraphClient, error) {
    conn, err := grpc.NewClient(addr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return nil, err
    }
    return pb.NewGraphClient(conn), nil
}

// Inject app_id vào mọi call
func ctxWithAppID(ctx context.Context, appID string) context.Context {
    return metadata.AppendToOutgoingContext(ctx, "x-kgs-app-id", appID)
}
```

### 3.2 CreateNode

```go
// Properties PHẢI bao gồm "id" (UUID) để Neo4j node có stable identifier
props := map[string]any{
    "id":         uuid.New().String(),
    "name":       "Login Feature",
    "priority":   "P0",
    "project_id": "proj-abc",  // dùng để query theo project
}
propsJSON, _ := json.Marshal(props)

reply, err := client.CreateNode(
    ctxWithAppID(ctx, "my-app-id"),
    &pb.CreateNodeRequest{
        Label:          "Feature",        // node label (EntityType name)
        PropertiesJson: string(propsJSON),
    },
)
// reply.NodeId = id lấy từ props["id"]
// reply.PropertiesJson = JSON của node sau khi lưu
```

**Lưu ý quan trọng:**
- `Label` phải khớp với `EntityType.Name` đã đăng ký (hoặc `"system"` app types)
- `properties_json` PHẢI có field `"id"` (UUID string) — dùng làm stable node ID
- OPA sẽ evaluate `{app_id, action: "CREATE_NODE", resource: label}` trước khi write

### 3.3 CreateEdge

```go
edgeProps := map[string]any{
    "app_id": "my-app-id",
}
edgeJSON, _ := json.Marshal(edgeProps)

reply, err := client.CreateEdge(
    ctxWithAppID(ctx, "my-app-id"),
    &pb.CreateEdgeRequest{
        RelationType:   "IMPLEMENTS",   // edge type label
        SourceNodeId:   "node-uuid-1",
        TargetNodeId:   "node-uuid-2",
        PropertiesJson: string(edgeJSON),
    },
)
```

### 3.4 Graph Traversal

```go
// GetContext — lấy neighbors 1 hop
reply, _ := client.GetContext(ctx, &pb.GetContextRequest{
    NodeId:    "node-uuid",
    Depth:     2,           // max 10 (guardrail)
    Direction: "OUTGOING",  // INCOMING | OUTGOING | BOTH
})

// GetImpact — downstream traversal
reply, _ := client.GetImpact(ctx, &pb.GetImpactRequest{
    NodeId:   "node-uuid",
    MaxDepth: 3,
})

// GetSubgraph — batch fetch
reply, _ := client.GetSubgraph(ctx, &pb.GetSubgraphRequest{
    NodeIds: []string{"id1", "id2", "id3"}, // max 1000 (guardrail)
})
```

---

## 4. Tích hợp qua HTTP REST

### 4.1 Base URL và headers

```
Base URL: http://localhost:8000
Headers:
  Content-Type: application/json
  X-API-Key: <api-key>
```

### 4.2 CreateNode

```bash
POST /v1/graph/nodes
{
  "label": "Feature",
  "properties_json": "{\"id\":\"uuid-here\",\"name\":\"Login\",\"app_id\":\"my-app\"}"
}

# Response 200:
{
  "node_id": "uuid-here",
  "label": "Feature",
  "properties_json": "{...full node props...}"
}
```

### 4.3 GetProjectGraph (HTTP-only endpoint)

```bash
GET /v1/graph/project/{project_id}

# Response:
{
  "nodes": [...],
  "edges": [...]
}
```

Đây là endpoint HTTP-only (không có gRPC equivalent), dùng để lấy toàn bộ graph của một project:

```go
// internal/service/graph_http_handler.go
// Query: SELECT nodes WHERE project_id = $project_id
// Cypher: MATCH (n {project_id: $project_id}) RETURN n
```

---

## 5. Tích hợp qua Kafka (Document Ingestion)

Dành cho upstream services cần bulk-populate đồ thị từ parsed documents.

### 5.1 Topic và event schema

```
Topic:    document.ingested
Group ID: knowledge-service
Broker:   localhost:9092 (default)
```

```go
// DocumentIngestedEvent — internal/kafka/consumer.go
type DocumentIngestedEvent struct {
    DocID      string         `json:"docId"`
    AppID      string         `json:"appId"`
    DocType    string         `json:"docType"`    // PRD|SRS|UI|TESTCASE
    NodeType   string         `json:"nodeType"`   // KG node label
    Properties map[string]any `json:"properties"` // phải có "id"
    ParentID   string         `json:"parentId,omitempty"`
    EdgeType   string         `json:"edgeType,omitempty"`
}
```

### 5.2 Produce event

```go
import "github.com/segmentio/kafka-go"

writer := kafka.NewWriter(kafka.WriterConfig{
    Brokers: []string{"localhost:9092"},
    Topic:   "document.ingested",
})

event := DocumentIngestedEvent{
    DocID:    "prd-feature-001",
    AppID:    "my-app-id",
    DocType:  "PRD",
    NodeType: "Feature",
    Properties: map[string]any{
        "id":          "prd-feature-001",
        "name":        "User Authentication",
        "priority":    "P0",
        "project_id":  "proj-abc",
    },
    ParentID: "epic-001",   // nếu có → tự tạo edge
    EdgeType: "IMPLEMENTS", // loại edge kết nối ParentID → node mới
}

payload, _ := json.Marshal(event)
writer.WriteMessages(ctx, kafka.Message{Value: payload})
```

**Flow xử lý của KGS khi nhận event:**
1. `CreateNode(appID, nodeType, properties)` → OPA check → Neo4j write → Redis event
2. Nếu `parentId + edgeType` có giá trị → `CreateEdge(appID, edgeType, parentId, nodeId, nil)`

---

## 6. Rules Management API

Rules cho phép trigger Cypher queries tự động:

```bash
POST /v1/rules
{
  "name": "FlagHighPriority",
  "trigger_type": "ON_WRITE",    # ON_WRITE | SCHEDULED
  "cypher_query": "MATCH (n {app_id: $app_id}) WHERE n.priority='P0' RETURN n",
  "action": "webhook",
  "cron": ""                     # dùng khi trigger_type=SCHEDULED, e.g. "0 * * * *"
}
```

**Trigger types:**
- `ON_WRITE`: Chạy tự động khi có event `node.created/updated/deleted` trên Redis Stream
- `SCHEDULED`: Chạy theo cron expression (gocron)

---

## 7. Policy Management API

Quản lý OPA Rego policies — kiểm soát quyền write vào graph:

```bash
POST /v1/policies
{
  "name": "allow-my-app",
  "rego_content": "package kgs\ndefault allow := false\nallow if { input.app_id == \"my-app-id\" }"
}
```

**Lifecycle:**
1. Policy được INSERT vào PostgreSQL
2. `PolicySyncRunner` (mỗi 30s) upload Rego lên OPA: `PUT /v1/policies/policy_{id}`
3. Mọi CreateNode/UpdateNode/DeleteNode đều qua OPA check trước khi ghi

---

## 8. Thêm Feature mới — Quy trình chuẩn

### 8.1 Thêm API endpoint mới (ví dụ: `BatchCreateNodes`)

**Bước 1: Định nghĩa Proto**

```protobuf
// api/graph/v1/graph.proto
message BatchCreateNodesRequest {
  repeated CreateNodeRequest nodes = 1;
}
message BatchCreateNodesReply {
  repeated CreateNodeReply nodes = 1;
}

service Graph {
  // ...existing...
  rpc BatchCreateNodes(BatchCreateNodesRequest) returns (BatchCreateNodesReply);
}
```

**Bước 2: Implement Service Layer**

```go
// internal/service/graph.go
func (s *GraphService) BatchCreateNodes(ctx context.Context, req *pb.BatchCreateNodesRequest) (*pb.BatchCreateNodesReply, error) {
    appID := extractAppID(ctx)
    var results []*pb.CreateNodeReply

    for _, nodeReq := range req.Nodes {
        var props map[string]any
        json.Unmarshal([]byte(nodeReq.PropertiesJson), &props)

        res, err := s.uc.CreateNode(ctx, appID, nodeReq.Label, props)
        if err != nil {
            return nil, err  // fail fast
        }

        resJSON, _ := json.Marshal(res)
        nodeId, _ := res["id"].(string)
        results = append(results, &pb.CreateNodeReply{
            NodeId:         nodeId,
            Label:          nodeReq.Label,
            PropertiesJson: string(resJSON),
        })
    }
    return &pb.BatchCreateNodesReply{Nodes: results}, nil
}
```

**Quy tắc Service Layer:**
- KHÔNG chứa business logic
- CHỈ map proto types ↔ domain types
- Gọi `uc.Method()` duy nhất
- Dùng `extractAppID(ctx)` để lấy app_id

**Bước 3: Thêm Biz method (nếu cần logic mới)**

```go
// internal/biz/graph.go
// Chỉ thêm nếu có business logic khác với CreateNode đơn lẻ
// Nếu chỉ là loop qua CreateNode → KHÔNG cần thêm biz method
```

**Bước 4: Đăng ký trong server**

```go
// internal/server/grpc.go
func NewGRPCServer(..., gs *service.GraphService) *grpc.Server {
    // GraphService đã registered — không cần thay đổi nếu extend cùng service
}
```

### 8.2 Thêm Worker mới

```go
// internal/biz/my_worker.go
type MyWorker struct {
    repo   GraphRepo
    ticker *time.Ticker
    log    *log.Helper
}

func NewMyWorker(repo GraphRepo, logger log.Logger) *MyWorker {
    return &MyWorker{repo: repo, log: log.NewHelper(logger)}
}

func (w *MyWorker) Start(ctx context.Context) {
    w.ticker = time.NewTicker(5 * time.Minute)
    go func() {
        for {
            select {
            case <-w.ticker.C:
                w.run(ctx)
            case <-ctx.Done():
                return
            }
        }
    }()
}

func (w *MyWorker) Stop(ctx context.Context) error {
    w.ticker.Stop()
    return nil
}

func (w *MyWorker) run(ctx context.Context) {
    // business logic
}
```

```go
// internal/server/worker.go — thêm vào WorkerServer
type WorkerServer struct {
    // ...existing...
    myWorker *biz.MyWorker  // thêm mới
}

func (s *WorkerServer) Start(ctx context.Context) error {
    // ...existing...
    s.myWorker.Start(ctx)
    return nil
}
```

```go
// internal/biz/biz.go — thêm vào ProviderSet
var ProviderSet = wire.NewSet(
    // ...existing...
    NewMyWorker,  // thêm mới
)
```

### 8.3 Thêm Data Repository mới

```go
// internal/data/my_repo.go
type myRepo struct {
    data *Data
    log  *log.Helper
}

func NewMyRepo(data *Data, logger log.Logger) biz.MyRepo {
    return &myRepo{data: data, log: log.NewHelper(logger)}
}

func (r *myRepo) FindBySomething(ctx context.Context, param string) ([]*biz.MyEntity, error) {
    var results []*biz.MyEntity
    err := r.data.db.WithContext(ctx).
        Where("app_id = ? AND something = ?", "system", param).
        Find(&results).Error
    return results, err
}
```

```go
// internal/data/data.go — thêm vào ProviderSet
var ProviderSet = wire.NewSet(
    NewData, NewGreeterRepo, NewRegistryRepo, NewOntologyRepo,
    NewGraphRepo, NewRulesRepo, NewPolicyRepo, NewRedisClient,
    NewMyRepo,  // thêm mới
)
```

---

## 9. Patterns & Conventions bắt buộc

### 9.1 app_id namespace isolation

**MỌI** Neo4j operation đều phải scope theo `app_id`:

```go
// ✅ ĐÚNG — có app_id trong mọi query
MATCH (n {app_id: $app_id, id: $node_id}) RETURN n

// ❌ SAI — không scope theo app_id (cross-tenant leak)
MATCH (n {id: $node_id}) RETURN n
```

### 9.2 Properties PHẢI có "id"

```go
// ✅ ĐÚNG
props := map[string]any{
    "id":   uuid.New().String(),  // stable UUID
    "name": "My Node",
}

// ❌ SAI — node sẽ không có stable ID, edge không thể tạo
props := map[string]any{
    "name": "My Node",
}
```

### 9.3 OPA fail-closed

Mọi write operation (CreateNode/UpdateNode/DeleteNode) đều **PHẢI** qua OPA check. Nếu OPA unreachable → **phải reject** (không bypass):

```go
// internal/biz/graph.go — pattern bắt buộc
allowed, err := uc.opa.EvaluatePolicy(ctx, appID, "CREATE_NODE", label)
if err != nil {
    return nil, err  // fail closed — không tiếp tục
}
if !allowed {
    return nil, errors.New("access denied by OPA policy")
}
```

### 9.4 Redis event sau write thành công

Sau mỗi write operation thành công, PHẢI publish Redis stream event (fire-and-forget):

```go
// Stream: "kgs:events:nodes"
// Event types: "node.created" | "node.updated" | "node.deleted"
uc.redisCli.XAdd(ctx, &redis.XAddArgs{
    Stream: "kgs:events:nodes",
    Values: map[string]interface{}{
        "event_type": "node.created",
        "app_id":     appID,
        "label":      label,
    },
})
// Lỗi XAdd KHÔNG return cho client (fire-and-forget)
```

### 9.5 Depth guardrails

```go
// internal/biz/graph_guardrails.go
const MaxAllowedDepth = 10
const MaxAllowedNodes = 1000

// GetContext/GetImpact/GetCoverage phải validate depth trước khi query
if err := ValidateDepth(depth); err != nil {
    return nil, err  // 400 Bad Request
}
```

### 9.6 Wire dependency injection

Mọi dependency phải wired qua Google Wire — không khởi tạo trực tiếp trong constructor:

```go
// ✅ ĐÚNG — inject qua constructor param
func NewMyUsecase(repo biz.MyRepo, logger log.Logger) *MyUsecase {
    return &MyUsecase{repo: repo}
}

// ❌ SAI — khởi tạo trực tiếp
func NewMyUsecase() *MyUsecase {
    repo := NewMyRepo(...)  // KHÔNG làm thế này
}
```

---

## 10. Error Handling

```
Storage Error:
  Neo4j/Postgres lỗi → data layer → biz layer → service layer → Kratos encode
  → HTTP 500 / gRPC Internal

OPA Unreachable:
  OPAClient.Do() lỗi → return (false, err)
  → biz trả "access denied" → HTTP 403 / gRPC PermissionDenied

OPA Denied:
  opaResp.Result == false
  → errors.New("access denied by OPA policy") → 403

Guardrail:
  ValidateDepth(11) → ErrDepthExceeded → 400 Bad Request

Redis XAdd fail:
  Không propagate lên caller — node đã saved thành công
  Event có thể bị mất (acceptable trade-off)
```

---

## 11. Storage Write Matrix

| Operation | Neo4j | PostgreSQL | Redis Stream | OPA |
|-----------|:-----:|:----------:|:------------:|:---:|
| CreateNode | ✅ WRITE | — | ✅ XADD | ✅ CHECK |
| UpdateNode | ✅ WRITE | — | ✅ XADD | ✅ CHECK |
| DeleteNode | ✅ WRITE | — | ✅ XADD | ✅ CHECK |
| CreateEdge | ✅ WRITE | — | — | ❌ TODO |
| GetContext/Impact/Coverage/Subgraph | ✅ READ | — | — | — |
| CreateRule | — | ✅ INSERT | — | — |
| CreatePolicy | — | ✅ INSERT | — | — |
| Kafka Ingest | ✅ WRITE | — | ✅ XADD | ✅ CHECK |

---

## 12. Config Reference

```yaml
# configs/config.yaml — cấu trúc data section
data:
  database:
    source: "host=localhost port=5432 dbname=kgs_platform user=postgres password=..."
  neo4j:
    uri: "bolt://localhost:7687"
    user: "neo4j"
    password: "password"
  redis:
    addr: "localhost:6379"
    password: ""
    read_timeout: "0.2s"
    write_timeout: "0.2s"
  qdrant:
    url: "http://localhost:6333"
    collection: "knowledge_chunks"
  opa:
    url: "http://localhost:8181"
  kafka:
    brokers: ["localhost:9092"]
    topic_document_ingested: "document.ingested"
```

---

## 13. Startup Flow (tham khảo khi debug)

```
main() → wire.Build()
  → NewData()
      ├── PostgreSQL AutoMigrate (App, APIKey, Quota, Rule, Policy, EntityType...)
      ├── SeedOntology() — insert 19 node types + 16 edge types nếu chưa có
      └── Connect Neo4j + Redis + Qdrant
  → NewApp() → app.Run()
      ├── HTTP Server :8000
      ├── gRPC Server :9000
      └── WorkerServer.Start()
          ├── PolicySyncRunner → sync policies to OPA ngay lập tức
          ├── RuleRunner → schedule all SCHEDULED rules
          ├── EventRunner → XGROUP CREATE kgs:events:nodes kgs-worker-group
          └── KafkaConsumer.Start() → listen "document.ingested"
```

---

## 14. Checklist cho mọi PR tích hợp

- [ ] API Key được gửi qua header `X-API-Key` hoặc `Authorization`
- [ ] `app_id` được inject qua `x-kgs-app-id` metadata (gRPC) hoặc `properties_json.app_id` (HTTP)
- [ ] Mọi node properties có field `"id"` (UUID string)
- [ ] Mọi Cypher query scope theo `app_id` parameter
- [ ] Write operations đi qua OPA check (không bypass)
- [ ] Redis XAdd sau write — fire-and-forget, không block
- [ ] Depth guardrail check trước traversal query
- [ ] Dependencies wired qua Google Wire (không khởi tạo trực tiếp)
- [ ] Không có logic trong Service Layer (chỉ mapping)
- [ ] Không truy cập Data Layer trực tiếp từ Service Layer (phải qua Biz)

---

*Tài liệu liên kết:*
- `specs/dataflow.md` — Chi tiết flow từng API
- `specs/service-inventory.md` — Toàn bộ components và dependencies  
- `specs/technical-design.md` — Thiết kế kỹ thuật chi tiết
- `openapi.yaml` — OpenAPI spec cho HTTP endpoints
