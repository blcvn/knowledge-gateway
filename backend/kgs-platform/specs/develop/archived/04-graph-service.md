# graph-service — Graph Core Service (REFACTOR)

> **Strategy:** 🔨 REFACTOR  
> **Source:** `kgs-platform/internal/biz/graph.go` + `internal/data/` (full reuse)  
> **Target:** `kgs-platform/cmd/graph/`  
> **Priority:** P0 — Core service, cần deploy sớm nhất

---

## 1. Phân Tích Codebase Hiện Tại

### 1.1 Code Có Thể Tái Sử Dụng TRỰC TIẾP

`kgs-platform/internal/biz/graph.go` (1074 lines) đã implement **đầy đủ** business logic:

| Method | Lines | Trạng thái |
|--------|-------|-----------|
| `CreateNode()` | ~200 | ✅ Hoàn chỉnh (OPA + validation + write + outbox + event) |
| `CreateEdge()` | ~120 | ✅ Hoàn chỉnh (validation + write + outbox + event) |
| `DeleteNode()` | ~90 | ✅ Hoàn chỉnh (soft delete + outbox) |
| `DeleteEdge()` | ~80 | ✅ Hoàn chỉnh (soft delete + outbox) |
| `BatchDeleteNodes()` | ~70 | ✅ Hoàn chỉnh |
| `GetContext()` | ~30 | ✅ Hoàn chỉnh (k-hop traversal) |
| `GetImpact()` | ~20 | ✅ Hoàn chỉnh |
| `GetCoverage()` | ~20 | ✅ Hoàn chỉnh |
| `GetSubgraph()` | ~20 | ✅ Hoàn chỉnh |
| `GetFullGraph()` | ~15 | ✅ Hoàn chỉnh |

**Data layer** (`internal/data/`) cũng đã hoàn chỉnh:

| File | Function |
|------|----------|
| `graph_write_pg.go` | PostgreSQL write (UpsertEntity, UpsertEdge, SoftDelete) |
| `graph_node.go` | Node read operations |
| `graph_edge.go` | Edge read operations |
| `graph_query.go` | Cypher query execution via Neo4j |
| `outbox.go` | Outbox pattern (EnqueueOutbox, PollOutbox) |
| `entity_pg.go` | Entity CRUD (PostgreSQL) |
| `edge_pg.go` | Edge CRUD (PostgreSQL) |
| `models_kg.go` | KGEntity, KGEdge, KGSyncOutbox models |
| `qdrant.go` | Qdrant client (search, upsert, delete) |

### 1.2 Những Gì Cần Thêm

| Tính năng | Trạng thái | Ghi chú |
|-----------|-----------|---------|
| gRPC Server (port 9003) | ❌ Thiếu | Hiện chỉ có HTTP server |
| `GetNodesByType()` RPC | ❌ Thiếu | Cần thêm |
| `BatchCreateNodes()` RPC | ❌ Thiếu | Cần thêm |
| `BatchCreateEdges()` RPC | ❌ Thiếu | Cần thêm |
| App Context từ gRPC metadata | ❌ Thiếu | Cần extract `x-app-id` từ metadata |
| Gọi ontology-service (gRPC) | ⚠️ Partial | Hiện dùng local OntologyValidator |
| Gọi policy-service (gRPC) | ⚠️ Partial | Hiện dùng local OPAClient |

---

## 2. Chiến Lược Refactor

### 2.1 Giữ Nguyên Toàn Bộ Business Logic

**KHÔNG** viết lại `biz/graph.go` — copy trực tiếp sang package mới:

```
kgs-platform/internal/biz/graph.go  →  kgs-platform/internal/graph/biz/graph.go
kgs-platform/internal/data/*.go     →  kgs-platform/internal/graph/data/*.go
```

### 2.2 Thêm gRPC Adapter

Chỉ cần viết thêm:
1. Proto definition
2. gRPC server (thin adapter layer)
3. gRPC clients cho ontology-service và policy-service

### 2.3 Thay Thế Local Calls

| Hiện tại | Mới |
|----------|-----|
| `biz.OntologyValidator.ValidateEntity()` — local | `ontology-service gRPC ValidateNodeProperties()` |
| `biz.OPAClient.EvaluatePolicy()` — local OPA | `policy-service gRPC Evaluate()` |
| Redis Streams for events | NATS publish |

---

## 3. Cấu Trúc Service Mới

```
kgs-platform/
├── cmd/
│   └── graph/
│       └── main.go                 ← Entry point
└── internal/
    └── graph/
        ├── biz/
        │   ├── graph.go            ← TÁI SỬ DỤNG từ internal/biz/graph.go
        │   ├── graph_write.go      ← TÁI SỬ DỤNG
        │   ├── graph_guardrails.go ← TÁI SỬ DỤNG
        │   ├── namespace.go        ← TÁI SỬ DỤNG
        │   ├── validator.go        ← THAY THẾ: gRPC call to ontology-service
        │   └── policy.go           ← THAY THẾ: gRPC call to policy-service
        ├── data/
        │   ├── entity_pg.go        ← TÁI SỬ DỤNG
        │   ├── edge_pg.go          ← TÁI SỬ DỤNG
        │   ├── graph_node.go       ← TÁI SỬ DỤNG
        │   ├── graph_edge.go       ← TÁI SỬ DỤNG
        │   ├── graph_write_pg.go   ← TÁI SỬ DỤNG
        │   ├── graph_query.go      ← TÁI SỬ DỤNG
        │   ├── outbox.go           ← TÁI SỬ DỤNG
        │   ├── nats.go             ← TÁI SỬ DỤNG (với topic updates)
        │   └── models_kg.go        ← TÁI SỬ DỤNG
        └── server/
            └── grpc.go             ← MỚI: gRPC server adapter
```

---

## 4. gRPC Server Adapter (Cần Viết Mới)

### 4.1 Proto Definition

```protobuf
// api/graph/v1/graph.proto
syntax = "proto3";
package graph.v1;

service GraphService {
  // Node operations
  rpc CreateNode(CreateNodeRequest) returns (Node);
  rpc GetNode(GetNodeRequest) returns (Node);
  rpc UpdateNode(UpdateNodeRequest) returns (Node);
  rpc DeleteNode(DeleteNodeRequest) returns (DeleteNodeResponse);
  rpc GetNodesByType(GetNodesByTypeRequest) returns (GetNodesByTypeResponse);
  rpc BatchCreateNodes(BatchCreateNodesRequest) returns (BatchCreateNodesResponse);

  // Edge operations
  rpc CreateEdge(CreateEdgeRequest) returns (Edge);
  rpc GetEdge(GetEdgeRequest) returns (Edge);
  rpc DeleteEdge(DeleteEdgeRequest) returns (google.protobuf.Empty);
  rpc GetEdgesByRelationType(GetEdgesByRelationTypeRequest) returns (GetEdgesByRelationTypeResponse);
  rpc BatchCreateEdges(BatchCreateEdgesRequest) returns (BatchCreateEdgesResponse);

  // Internal read (called by query-intel-service, overlay-service)
  rpc GetEntityMetadata(GetEntityMetadataRequest) returns (EntityMetadata);
  rpc GetFullGraph(GetFullGraphRequest) returns (GetFullGraphResponse);
  
  // Traversal (di chuyển sang query-intel, nhưng giữ backward compat)
  rpc GetContext(GetContextRequest) returns (SubgraphResponse);
  rpc GetImpact(GetImpactRequest) returns (SubgraphResponse);
  rpc GetCoverage(GetCoverageRequest) returns (SubgraphResponse);
  rpc GetSubgraph(GetSubgraphRequest) returns (SubgraphResponse);
}

message CreateNodeRequest {
  string entity_type = 1;     // "Requirement" (no namespace prefix)
  bytes properties_json = 2;
  // app_id, tenant_id từ gRPC metadata (x-app-id, x-tenant-id)
}

message Node {
  string node_id = 1;         // "ba_agent__Requirement__REQ-001"
  string entity_type = 2;
  string app_id = 3;
  string namespace = 4;
  bytes properties_json = 5;
  string neo4j_label = 6;
  bool validation_passed = 7;
  google.protobuf.Timestamp created_at = 8;
  google.protobuf.Timestamp updated_at = 9;
}

message DeleteNodeResponse {
  int32 edges_removed = 1;
}
```

### 4.2 gRPC Server Implementation

```go
// internal/graph/server/grpc.go
package server

import (
    "context"
    "encoding/json"

    graphpb "kgs-platform/api/graph/v1"
    "kgs-platform/internal/graph/biz"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"
)

type GraphServer struct {
    graphpb.UnimplementedGraphServiceServer
    uc  *biz.GraphUsecase
    log *slog.Logger
}

// extractAppContext extracts app_id and tenant_id from gRPC metadata
// injected by kgs-gateway
func extractAppContext(ctx context.Context) (appID, tenantID string, err error) {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return "", "", status.Error(codes.Unauthenticated, "missing metadata")
    }
    
    appIDs := md.Get("x-app-id")
    tenantIDs := md.Get("x-tenant-id")
    
    if len(appIDs) == 0 || len(tenantIDs) == 0 {
        return "", "", status.Error(codes.Unauthenticated, "missing app context")
    }
    
    return appIDs[0], tenantIDs[0], nil
}

func (s *GraphServer) CreateNode(ctx context.Context, req *graphpb.CreateNodeRequest) (*graphpb.Node, error) {
    appID, tenantID, err := extractAppContext(ctx)
    if err != nil {
        return nil, err
    }
    
    var properties map[string]any
    if err := json.Unmarshal(req.PropertiesJson, &properties); err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid properties JSON: %v", err)
    }
    
    // Gọi GraphUsecase (tái sử dụng toàn bộ)
    result, err := s.uc.CreateNode(ctx, appID, tenantID, req.EntityType, properties)
    if err != nil {
        return nil, toGRPCStatus(err)
    }
    
    return toProtoNode(result), nil
}

func (s *GraphServer) GetNode(ctx context.Context, req *graphpb.GetNodeRequest) (*graphpb.Node, error) {
    appID, tenantID, err := extractAppContext(ctx)
    if err != nil {
        return nil, err
    }
    
    result, err := s.uc.GetNode(ctx, appID, tenantID, req.NodeId)
    if err != nil {
        return nil, toGRPCStatus(err)
    }
    
    return toProtoNode(result), nil
}

func (s *GraphServer) DeleteNode(ctx context.Context, req *graphpb.DeleteNodeRequest) (*graphpb.DeleteNodeResponse, error) {
    appID, tenantID, err := extractAppContext(ctx)
    if err != nil {
        return nil, err
    }
    
    edgesRemoved, err := s.uc.DeleteNode(ctx, appID, tenantID, req.NodeId)
    if err != nil {
        return nil, toGRPCStatus(err)
    }
    
    return &graphpb.DeleteNodeResponse{EdgesRemoved: int32(edgesRemoved)}, nil
}

// BatchCreateNodes — Thêm mới
func (s *GraphServer) BatchCreateNodes(ctx context.Context, req *graphpb.BatchCreateNodesRequest) (*graphpb.BatchCreateNodesResponse, error) {
    appID, tenantID, err := extractAppContext(ctx)
    if err != nil {
        return nil, err
    }
    
    var created, failed int32
    var results []*graphpb.Node
    
    for _, nodeReq := range req.Nodes {
        var props map[string]any
        json.Unmarshal(nodeReq.PropertiesJson, &props)
        
        result, err := s.uc.CreateNode(ctx, appID, tenantID, nodeReq.EntityType, props)
        if err != nil {
            failed++
            continue
        }
        created++
        results = append(results, toProtoNode(result))
    }
    
    return &graphpb.BatchCreateNodesResponse{
        Created: created,
        Failed:  failed,
        Results: results,
    }, nil
}

// toGRPCStatus converts domain errors to gRPC status codes
func toGRPCStatus(err error) error {
    // Map domain errors to gRPC codes
    switch {
    case errors.Is(err, biz.ErrForbidden):
        return status.Error(codes.PermissionDenied, err.Error())
    case errors.Is(err, biz.ErrNotFound):
        return status.Error(codes.NotFound, err.Error())
    case errors.Is(err, biz.ErrValidation):
        return status.Error(codes.InvalidArgument, err.Error())
    default:
        return status.Error(codes.Internal, err.Error())
    }
}
```

---

## 5. Thay Thế Local Calls Bằng gRPC

### 5.1 OntologyValidator → ontology-service gRPC

```go
// internal/graph/biz/validator.go — THAY THẾ local validator

type RemoteOntologyValidator struct {
    client ontologypb.OntologyServiceClient
    log    *slog.Logger
}

func (v *RemoteOntologyValidator) ValidateEntity(ctx context.Context, appID, entityType string, properties map[string]any) error {
    propsJSON, _ := json.Marshal(properties)
    
    result, err := v.client.ValidateNodeProperties(ctx, &ontologypb.ValidateNodePropertiesRequest{
        AppId:            appID,
        EntityTypeName:   entityType,
        PropertiesJson:   propsJSON,
    })
    if err != nil {
        // Nếu ontology-service không available, log warning và cho phép (graceful degradation)
        v.log.Warn("ontology validation failed, allowing", "error", err)
        return nil
    }
    
    if !result.Valid {
        return fmt.Errorf("schema validation failed: %s", strings.Join(result.Errors, "; "))
    }
    return nil
}
```

### 5.2 OPAClient → policy-service gRPC

```go
// internal/graph/biz/policy.go — THAY THẾ local OPA client

type RemotePolicyClient struct {
    client policypb.PolicyServiceClient
    log    *slog.Logger
}

func (c *RemotePolicyClient) EvaluatePolicy(ctx context.Context, appID, action, resource string) (bool, error) {
    result, err := c.client.Evaluate(ctx, &policypb.EvaluateRequest{
        AppId: appID,
        Input: &policypb.EvaluateInput{
            Action:   action,
            Resource: resource,
        },
    })
    if err != nil {
        // Graceful degradation: nếu policy-service down, allow by default
        c.log.Warn("policy evaluation failed, allowing", "app_id", appID, "action", action, "error", err)
        return true, nil
    }
    return result.Allow, nil
}
```

---

## 6. NATS Events (Cập Nhật)

Hiện tại graph.go dùng Redis Streams (`kgs:events:nodes`). Cần chuyển sang NATS:

```go
// internal/graph/data/nats.go
// TÁI SỬ DỤNG từ kgs-platform/internal/data/nats.go

// Chỉ cần update topics:
const (
    TopicNodeCreated = "graph.node.created"  // Đã có
    TopicNodeUpdated = "graph.node.updated"  // Thêm mới
    TopicNodeDeleted = "graph.node.deleted"  // Đã có
    TopicEdgeCreated = "graph.edge.created"  // Đã có
    TopicEdgeDeleted = "graph.edge.deleted"  // Đã có
)
```

---

## 7. Entry Point

```go
// cmd/graph/main.go
package main

import (
    "github.com/go-kratos/kratos/v2"
    "github.com/go-kratos/kratos/v2/transport/grpc"
    
    graphpb "kgs-platform/api/graph/v1"
    "kgs-platform/internal/graph/biz"
    "kgs-platform/internal/graph/data"
    "kgs-platform/internal/graph/server"
)

func main() {
    db := data.NewDB(conf.Database.DSN)
    neo4j := data.NewNeo4j(conf.Neo4j)
    redis := data.NewRedis(conf.Redis)
    nats := data.NewNATS(conf.NATS.Addr)
    
    // gRPC clients for dependencies
    ontologyConn, _ := grpc.Dial(conf.OntologyService, grpc.WithInsecure())
    policyConn, _ := grpc.Dial(conf.PolicyService, grpc.WithInsecure())
    
    ontologyClient := ontologypb.NewOntologyServiceClient(ontologyConn)
    policyClient := policypb.NewPolicyServiceClient(policyConn)
    
    // Data repos (tái sử dụng)
    writeRepo := data.NewGraphWriteRepo(db)
    entityReader := data.NewEntityReader(db, neo4j)
    
    // Business logic (tái sử dụng)
    validator := biz.NewRemoteOntologyValidator(ontologyClient)
    policyEval := biz.NewRemotePolicyClient(policyClient)
    planner := biz.NewQueryPlanner()
    lockMgr := data.NewRedisLockManager(redis)
    
    graphUC := biz.NewGraphUsecaseWithStorage(
        nil, // legacy repo
        writeRepo,
        entityReader,
        planner,
        policyEval,
        validator,
        redis,
        lockMgr,
        nil, // overlay (not needed in graph-service)
        logger,
    )
    
    // gRPC server (mới)
    grpcSrv := grpc.NewServer(grpc.Address(":9003"))
    graphpb.RegisterGraphServiceServer(grpcSrv, server.NewGraphServer(graphUC))
    
    app := kratos.New(kratos.Server(grpcSrv))
    app.Run()
}
```

---

## 8. Namespace Injection

Giữ nguyên logic namespace injection từ `biz/namespace.go`:

```go
// Tái sử dụng TRỰC TIẾP
func ComputeNamespace(appID, tenantID string) string {
    return fmt.Sprintf("graph/%s/%s", appID, tenantID)
}

func ComputeNeo4jLabel(appID, entityType string) string {
    return fmt.Sprintf("%s__%s", appID, entityType)
}
```

---

## 9. Backward Compatibility với Monolith

Trong giai đoạn migration, graph-service cần forward requests từ monolith:

```go
// Legacy HTTP endpoints (forward từ gateway)
// Tất cả /v1/graphiti/**, /v1/console/graph/** vẫn hoạt động
// Gateway route chúng vào graph-service thay vì kg-service monolith
```

---

## 10. Ước Tính Effort

| Task | Effort |
|------|--------|
| Copy + adapt biz/graph.go | 0.5 ngày |
| Copy + adapt data/ layer | 0.5 ngày |
| Proto definition + code gen | 0.5 ngày |
| gRPC server adapter | 1.5 ngày |
| Replace OPA → policy-service gRPC | 0.5 ngày |
| Replace OntologyValidator → ontology-service gRPC | 0.5 ngày |
| NATS events update | 0.5 ngày |
| Unit + integration tests | 1.5 ngày |
| **Total** | **6 ngày** |

**Tiết kiệm so với viết mới:** ~70% (business logic đã có)
