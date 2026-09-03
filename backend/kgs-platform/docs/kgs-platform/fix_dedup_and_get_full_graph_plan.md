# Kế hoạch Fix Duplication Bug & Thêm API GetFullGraph

## Mục lục

1. [Tổng quan vấn đề](#1-tổng-quan-vấn-đề)
2. [Fix #1: Node/Edge Duplication Bug](#2-fix-1-nodeedge-duplication-bug)
3. [Fix #2: Thêm API GetFullGraph](#3-fix-2-thêm-api-getfullgraph)
4. [Thứ tự triển khai](#4-thứ-tự-triển-khai)
5. [Test plan](#5-test-plan)

---

## 1. Tổng quan vấn đề

### 1.1 Duplication Bug

**Triệu chứng:** Mỗi lần gọi `SaveGraph` (BatchUpsertEntities), số lượng node trong Neo4j bị nhân đôi (ví dụ 231 → 462 nodes), các edge giữa cùng source/target cũng bị duplicate.

**Nguyên nhân gốc:** Tất cả Cypher write queries sử dụng `CREATE` thay vì `MERGE`. Khi entity có cùng `(app_id, tenant_id, id)` được gửi lại, Neo4j tạo node/edge mới thay vì cập nhật node/edge hiện có.

**Các file bị ảnh hưởng:**

| File | Dòng | Mô tả |
|------|------|-------|
| `internal/batch/neo4j_writer.go` | 65 | `writeChunk` — bulk create nodes |
| `internal/data/graph_node.go` | 50 | `CreateNode` — single node create |
| `internal/data/graph_edge.go` | 33 | `CreateEdge` — single edge create |

### 1.2 Thiếu API GetFullGraph

kgs-platform hiện **không có RPC** nào cho phép lấy toàn bộ graph (nodes + edges) theo `app_id` + `tenant_id`. `HybridSearch` không thể dùng làm workaround vì:
- `topK` bị hard cap ở `maxSearchTopK = 100` (`internal/search/search.go:16`)
- `query="*"` không hợp lệ trong Lucene full-text search

---

## 2. Fix #1: Node/Edge Duplication Bug

### 2.1 Thay đổi Cypher: CREATE → MERGE

#### 2.1.1 File: `internal/batch/neo4j_writer.go` — `writeChunk`

**Hiện tại (dòng 63-68):**
```cypher
UNWIND $entities AS e
CREATE (n:%s {app_id: $app_id, tenant_id: $tenant_id, id: e.id})
SET n += e
RETURN count(n) AS created
```

**Sửa thành:**
```cypher
UNWIND $entities AS e
MERGE (n:%s {app_id: $app_id, tenant_id: $tenant_id, id: e.id})
SET n += e
RETURN count(n) AS created
```

**Lưu ý:**
- `MERGE` sẽ match node có cùng `(app_id, tenant_id, id)` nếu đã tồn tại, chỉ tạo mới nếu không tìm thấy
- `SET n += e` vẫn hoạt động đúng — cập nhật properties nếu node đã tồn tại
- Rename return alias từ `created` thành `upserted` để phản ánh đúng ngữ nghĩa
- Cân nhắc dùng `ON CREATE SET` và `ON MATCH SET` nếu cần phân biệt logic tạo mới vs cập nhật (ví dụ: `created_at` chỉ set khi tạo mới, `updated_at` set khi cập nhật)

**Code change:**
```go
// neo4j_writer.go — writeChunk method
query := fmt.Sprintf(`
    UNWIND $entities AS e
    MERGE (n:%s {app_id: $app_id, tenant_id: $tenant_id, id: e.id})
    ON CREATE SET n += e, n.created_at = datetime()
    ON MATCH SET n += e, n.updated_at = datetime()
    RETURN count(n) AS upserted
`, label)
```

#### 2.1.2 File: `internal/data/graph_node.go` — `CreateNode`

**Hiện tại (dòng 49-53):**
```cypher
CREATE (n:%s {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})
SET n += $props
RETURN n
```

**Sửa thành:**
```go
// graph_node.go — CreateNode method
query := fmt.Sprintf(`
    MERGE (n:%s {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})
    ON CREATE SET n += $props, n.created_at = datetime()
    ON MATCH SET n += $props, n.updated_at = datetime()
    RETURN n
`, cleanLabel)
```

#### 2.1.3 File: `internal/data/graph_edge.go` — `CreateEdge`

**Hiện tại (dòng 30-35):**
```cypher
MATCH (a {app_id: $app_id, tenant_id: $tenant_id, id: $source_node_id})
MATCH (b {app_id: $app_id, tenant_id: $tenant_id, id: $target_node_id})
CREATE (a)-[rel:%s {app_id: $app_id, tenant_id: $tenant_id, id: $edge_id}]->(b)
SET rel += $props
RETURN rel
```

**Sửa thành:**
```go
// graph_edge.go — CreateEdge method
query := fmt.Sprintf(`
    MATCH (a {app_id: $app_id, tenant_id: $tenant_id, id: $source_node_id})
    MATCH (b {app_id: $app_id, tenant_id: $tenant_id, id: $target_node_id})
    MERGE (a)-[rel:%s {app_id: $app_id, tenant_id: $tenant_id, id: $edge_id}]->(b)
    ON CREATE SET rel += $props, rel.created_at = datetime()
    ON MATCH SET rel += $props, rel.updated_at = datetime()
    RETURN rel
`, cleanRelationType)
```

### 2.2 Thêm UNIQUE Constraint trong Neo4j

Thêm constraint để đảm bảo tính toàn vẹn dữ liệu ở tầng database, ngăn duplicate ngay cả khi code bị lỗi.

#### 2.2.1 Tạo migration hoặc init script

**File mới:** `internal/data/neo4j_constraints.go`

```go
package data

import (
    "context"
    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// EnsureConstraints creates unique constraints for node identity.
// Neo4j Community Edition does not support unique constraints on relationships,
// so we only constrain nodes. MERGE on edges handles dedup at query level.
func EnsureConstraints(ctx context.Context, driver neo4j.DriverWithContext) error {
    session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
    defer session.Close(ctx)

    // Constraint cho từng label phổ biến
    // Lưu ý: Neo4j yêu cầu chỉ định label cụ thể cho constraint
    constraints := []string{
        // Generic constraint — áp dụng cho tất cả nodes có property combo này
        // Neo4j 5.x hỗ trợ node key constraint
        `CREATE CONSTRAINT IF NOT EXISTS FOR (n:Entity)
         REQUIRE (n.app_id, n.tenant_id, n.id) IS NODE KEY`,
    }

    for _, cypher := range constraints {
        _, err := session.Run(ctx, cypher, nil)
        if err != nil {
            return fmt.Errorf("failed to create constraint: %w", err)
        }
    }
    return nil
}
```

**Vấn đề với label-based constraint:**
- Neo4j constraint yêu cầu chỉ định label cụ thể
- kgs-platform sử dụng dynamic labels (entity type trở thành label)
- **Giải pháp:** Thêm label chung `Entity` cho tất cả node khi MERGE, sau đó tạo constraint trên label `Entity`

**Cập nhật Cypher trong `writeChunk`:**
```cypher
UNWIND $entities AS e
MERGE (n:%s {app_id: $app_id, tenant_id: $tenant_id, id: e.id})
ON CREATE SET n += e, n.created_at = datetime(), n:Entity
ON MATCH SET n += e, n.updated_at = datetime()
RETURN count(n) AS upserted
```

> **Lưu ý:** `n:Entity` trong `ON CREATE SET` sẽ thêm label `Entity` cho node mới. Node đã tồn tại sẽ đã có label này từ lần tạo đầu tiên.

#### 2.2.2 Gọi EnsureConstraints khi khởi động

Thêm vào `internal/data/data.go` (hoặc file init tương đương):

```go
// Trong hàm NewData hoặc init
if err := EnsureConstraints(ctx, neo4jDriver); err != nil {
    log.Fatalf("Failed to ensure Neo4j constraints: %v", err)
}
```

### 2.3 Xử lý dữ liệu duplicate hiện có

Sau khi deploy fix, cần chạy script cleanup để gộp các node bị duplicate:

```cypher
// 1. Tìm các node bị duplicate (cùng app_id, tenant_id, id nhưng khác internal Neo4j ID)
MATCH (n)
WITH n.app_id AS app_id, n.tenant_id AS tenant_id, n.id AS id, collect(n) AS nodes
WHERE size(nodes) > 1
RETURN app_id, tenant_id, id, size(nodes) AS count
ORDER BY count DESC
LIMIT 100

// 2. Merge duplicate nodes — giữ node đầu tiên, chuyển relationships, xóa phần còn lại
MATCH (n)
WITH n.app_id AS app_id, n.tenant_id AS tenant_id, n.id AS id, collect(n) AS nodes
WHERE size(nodes) > 1
WITH nodes[0] AS keep, nodes[1..] AS duplicates
UNWIND duplicates AS dup
// Chuyển incoming relationships
CALL {
    WITH keep, dup
    MATCH (dup)<-[r]->(other)
    WHERE other <> keep
    WITH keep, r, other, type(r) AS relType, properties(r) AS relProps
    // Tạo relationship mới từ keep
    CALL apoc.create.relationship(keep, relType, relProps, other) YIELD rel
    DELETE r
    RETURN count(*) AS moved
}
// Xóa duplicate node
DETACH DELETE dup
RETURN count(*) AS removed

// 3. Xóa duplicate edges (cùng source, target, type, id)
MATCH (a)-[r]->(b)
WITH a, b, type(r) AS relType, r.id AS relId, collect(r) AS rels
WHERE size(rels) > 1
WITH rels[0] AS keep, rels[1..] AS duplicates
UNWIND duplicates AS dup
DELETE dup
RETURN count(*) AS removed_edges
```

> **Lưu ý:** Script cleanup cần APOC plugin. Nếu không có APOC, phải xử lý thủ công per relationship type.

---

## 3. Fix #2: Thêm API GetFullGraph

### 3.1 Proto Definition

**File:** `proto/kgs/v1/graph.proto` (thêm vào service definition hiện có)

```protobuf
// Thêm message definitions
message GetFullGraphRequest {
  string app_id = 1;
  string tenant_id = 2;
  // Optional: pagination
  int32 node_limit = 3;   // 0 = no limit (default: 10000)
  int32 node_offset = 4;  // offset for pagination
}

message GraphNode {
  string id = 1;
  string label = 2;
  map<string, string> properties = 3;
}

message GraphEdge {
  string id = 1;
  string relation_type = 2;
  string source_node_id = 3;
  string target_node_id = 4;
  map<string, string> properties = 5;
}

message GetFullGraphResponse {
  repeated GraphNode nodes = 1;
  repeated GraphEdge edges = 2;
  int32 total_nodes = 3;
  int32 total_edges = 4;
}

// Thêm RPC vào service
service GraphService {
  // ... existing RPCs ...
  rpc GetFullGraph(GetFullGraphRequest) returns (GetFullGraphResponse);
}
```

### 3.2 Cypher Queries

#### 3.2.1 Lấy tất cả nodes

```cypher
MATCH (n {app_id: $app_id, tenant_id: $tenant_id})
RETURN n, labels(n) AS labels
ORDER BY n.id
SKIP $offset
LIMIT $limit
```

#### 3.2.2 Lấy tất cả edges

```cypher
MATCH (a {app_id: $app_id, tenant_id: $tenant_id})-[r]->(b {app_id: $app_id, tenant_id: $tenant_id})
RETURN r, type(r) AS rel_type, a.id AS source_id, b.id AS target_id
ORDER BY r.id
```

#### 3.2.3 Query tối ưu — lấy cả nodes và edges trong 1 query

```cypher
MATCH (n {app_id: $app_id, tenant_id: $tenant_id})
WITH collect(n) AS allNodes
UNWIND allNodes AS n
OPTIONAL MATCH (n)-[r]->(m)
WHERE m IN allNodes
RETURN
  collect(DISTINCT {id: n.id, labels: labels(n), props: properties(n)}) AS nodes,
  collect(DISTINCT {id: r.id, type: type(r), source: startNode(r).id, target: endNode(r).id, props: properties(r)}) AS edges
```

> **Lưu ý performance:** Với graph lớn (>10k nodes), nên dùng 2 query riêng biệt + pagination cho nodes. Edges lấy theo batch nodes.

### 3.3 Data Layer Implementation

**File:** `internal/data/graph_query.go` (file mới)

```go
package data

import (
    "context"
    "fmt"

    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type FullGraphResult struct {
    Nodes      []NodeResult
    Edges      []EdgeResult
    TotalNodes int
    TotalEdges int
}

type NodeResult struct {
    ID         string
    Labels     []string
    Properties map[string]any
}

type EdgeResult struct {
    ID           string
    RelationType string
    SourceNodeID string
    TargetNodeID string
    Properties   map[string]any
}

func (r *graphRepo) GetFullGraph(ctx context.Context, appID, tenantID string, limit, offset int) (*FullGraphResult, error) {
    session := r.data.neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
    defer session.Close(ctx)

    if limit <= 0 {
        limit = 10000
    }

    result := &FullGraphResult{}

    // 1. Count totals
    countResult, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
        res, err := tx.Run(ctx, `
            MATCH (n {app_id: $app_id, tenant_id: $tenant_id})
            RETURN count(n) AS total
        `, map[string]any{"app_id": appID, "tenant_id": tenantID})
        if err != nil {
            return nil, err
        }
        if res.Next(ctx) {
            return res.Record().Values[0], nil
        }
        return int64(0), nil
    })
    if err != nil {
        return nil, fmt.Errorf("count nodes: %w", err)
    }
    result.TotalNodes = int(countResult.(int64))

    // 2. Fetch nodes with pagination
    nodesResult, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
        res, err := tx.Run(ctx, `
            MATCH (n {app_id: $app_id, tenant_id: $tenant_id})
            RETURN n, labels(n) AS labels
            ORDER BY n.id
            SKIP $offset
            LIMIT $limit
        `, map[string]any{
            "app_id":    appID,
            "tenant_id": tenantID,
            "offset":    offset,
            "limit":     limit,
        })
        if err != nil {
            return nil, err
        }

        var nodes []NodeResult
        for res.Next(ctx) {
            record := res.Record()
            node := record.Values[0].(neo4j.Node)
            labels := record.Values[1].([]any)

            labelStrings := make([]string, len(labels))
            for i, l := range labels {
                labelStrings[i] = l.(string)
            }

            nodes = append(nodes, NodeResult{
                ID:         node.Props["id"].(string),
                Labels:     labelStrings,
                Properties: node.Props,
            })
        }
        return nodes, res.Err()
    })
    if err != nil {
        return nil, fmt.Errorf("fetch nodes: %w", err)
    }
    result.Nodes = nodesResult.([]NodeResult)

    // 3. Fetch edges for the retrieved nodes
    if len(result.Nodes) > 0 {
        nodeIDs := make([]string, len(result.Nodes))
        for i, n := range result.Nodes {
            nodeIDs[i] = n.ID
        }

        edgesResult, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
            res, err := tx.Run(ctx, `
                MATCH (a {app_id: $app_id, tenant_id: $tenant_id})-[r]->(b {app_id: $app_id, tenant_id: $tenant_id})
                WHERE a.id IN $node_ids AND b.id IN $node_ids
                RETURN r, type(r) AS rel_type, a.id AS source_id, b.id AS target_id
            `, map[string]any{
                "app_id":    appID,
                "tenant_id": tenantID,
                "node_ids":  nodeIDs,
            })
            if err != nil {
                return nil, err
            }

            var edges []EdgeResult
            for res.Next(ctx) {
                record := res.Record()
                rel := record.Values[0].(neo4j.Relationship)
                relType := record.Values[1].(string)
                sourceID := record.Values[2].(string)
                targetID := record.Values[3].(string)

                edges = append(edges, EdgeResult{
                    ID:           rel.Props["id"].(string),
                    RelationType: relType,
                    SourceNodeID: sourceID,
                    TargetNodeID: targetID,
                    Properties:   rel.Props,
                })
            }
            return edges, res.Err()
        })
        if err != nil {
            return nil, fmt.Errorf("fetch edges: %w", err)
        }
        result.Edges = edgesResult.([]EdgeResult)
        result.TotalEdges = len(result.Edges)
    }

    return result, nil
}
```

### 3.4 Service Layer Implementation

**File:** `internal/service/graph.go` (thêm method)

```go
func (s *GraphService) GetFullGraph(ctx context.Context, req *pb.GetFullGraphRequest) (*pb.GetFullGraphResponse, error) {
    started := time.Now()
    s.log.Infof("GetFullGraph app_id=%s tenant_id=%s limit=%d offset=%d",
        req.AppId, req.TenantId, req.NodeLimit, req.NodeOffset)

    result, err := s.graphRepo.GetFullGraph(ctx, req.AppId, req.TenantId,
        int(req.NodeLimit), int(req.NodeOffset))
    if err != nil {
        s.log.Errorf("GetFullGraph failed app_id=%s tenant_id=%s err=%v", req.AppId, req.TenantId, err)
        return nil, status.Errorf(codes.Internal, "get full graph: %v", err)
    }

    // Convert to proto
    resp := &pb.GetFullGraphResponse{
        TotalNodes: int32(result.TotalNodes),
        TotalEdges: int32(result.TotalEdges),
    }

    for _, n := range result.Nodes {
        propsMap := make(map[string]string)
        for k, v := range n.Properties {
            propsMap[k] = fmt.Sprintf("%v", v)
        }
        resp.Nodes = append(resp.Nodes, &pb.GraphNode{
            Id:         n.ID,
            Label:      n.Labels[0], // primary label
            Properties: propsMap,
        })
    }

    for _, e := range result.Edges {
        propsMap := make(map[string]string)
        for k, v := range e.Properties {
            propsMap[k] = fmt.Sprintf("%v", v)
        }
        resp.Edges = append(resp.Edges, &pb.GraphEdge{
            Id:           e.ID,
            RelationType: e.RelationType,
            SourceNodeId: e.SourceNodeID,
            TargetNodeId: e.TargetNodeID,
            Properties:   propsMap,
        })
    }

    s.log.Infof("GetFullGraph succeeded app_id=%s tenant_id=%s nodes=%d edges=%d duration=%s",
        req.AppId, req.TenantId, len(resp.Nodes), len(resp.Edges), time.Since(started))
    return resp, nil
}
```

### 3.5 Tích hợp với ba-agent-service

Sau khi GetFullGraph RPC sẵn sàng, cập nhật `KGServiceRepository` trong ba-agent-service:

```go
// Thay thế getGraphByScope hiện tại (dùng HybridSearch workaround)
func (r *KGServiceRepository) getGraphByScope(ctx context.Context, scope v32.GraphScope) (*v32.Graph, error) {
    resp, err := r.client.GetFullGraph(ctx, &pb.GetFullGraphRequest{
        AppId:    scope.AppID,
        TenantId: scope.TenantID,
    })
    if err != nil {
        return nil, fmt.Errorf("get full graph: %w", err)
    }
    return r.converter.FullGraphResponseToGraph(resp), nil
}
```

---

## 4. Thứ tự triển khai

### Phase 1: Fix Duplication Bug (Ưu tiên cao)

| # | Task | File | Ước lượng |
|---|------|------|-----------|
| 1.1 | Đổi `CREATE` → `MERGE` trong `writeChunk` | `internal/batch/neo4j_writer.go` | S |
| 1.2 | Đổi `CREATE` → `MERGE` trong `CreateNode` | `internal/data/graph_node.go` | S |
| 1.3 | Đổi `CREATE` → `MERGE` trong `CreateEdge` | `internal/data/graph_edge.go` | S |
| 1.4 | Thêm `ON CREATE SET` / `ON MATCH SET` với timestamps | 3 files trên | S |
| 1.5 | Tạo `neo4j_constraints.go` — EnsureConstraints | `internal/data/` | M |
| 1.6 | Gọi EnsureConstraints khi khởi động | `internal/data/data.go` | S |
| 1.7 | Viết unit tests cho MERGE behavior | `internal/batch/`, `internal/data/` | M |
| 1.8 | Chạy cleanup script cho dữ liệu duplicate hiện có | Neo4j console | M |

### Phase 2: Thêm API GetFullGraph (Ưu tiên cao)

| # | Task | File | Ước lượng |
|---|------|------|-----------|
| 2.1 | Thêm proto definitions (messages + RPC) | `proto/kgs/v1/graph.proto` | M |
| 2.2 | Generate Go code từ proto | `make proto` | S |
| 2.3 | Implement `GetFullGraph` trong data layer | `internal/data/graph_query.go` | L |
| 2.4 | Implement `GetFullGraph` trong service layer | `internal/service/graph.go` | M |
| 2.5 | Register RPC trong gRPC server | `cmd/server/main.go` | S |
| 2.6 | Viết unit/integration tests | `internal/data/`, `internal/service/` | L |
| 2.7 | Publish proto mới lên registry | CI/CD | S |

### Phase 3: Tích hợp ba-agent-service (Sau Phase 2)

| # | Task | File | Ước lượng |
|---|------|------|-----------|
| 3.1 | Cập nhật proto dependency trong ba-agent-service | `go.mod` | S |
| 3.2 | Thêm converter cho GetFullGraphResponse | `DomainConverter` | M |
| 3.3 | Thay thế `getGraphByScope` workaround | `KGServiceRepository` | M |
| 3.4 | Cập nhật tests | Tests | M |

**Ước lượng kích cỡ:** S = nhỏ (< 30 dòng code), M = trung bình (30-100 dòng), L = lớn (> 100 dòng)

---

## 5. Test plan

### 5.1 Unit Tests — Duplication Fix

```
- [ ] Test MERGE tạo node mới khi chưa tồn tại
- [ ] Test MERGE cập nhật node khi đã tồn tại (cùng app_id, tenant_id, id)
- [ ] Test MERGE không tạo duplicate node
- [ ] Test MERGE edge giữa 2 nodes không tạo duplicate relationship
- [ ] Test ON CREATE SET — created_at được set khi tạo mới
- [ ] Test ON MATCH SET — updated_at được set khi cập nhật
- [ ] Test batch MERGE (writeChunk) với entities đã tồn tại trong DB
- [ ] Test constraint violation khi cố tạo duplicate (nếu dùng constraint)
```

### 5.2 Integration Tests — GetFullGraph

```
- [ ] Test GetFullGraph trả đúng tất cả nodes theo app_id + tenant_id
- [ ] Test GetFullGraph trả đúng tất cả edges giữa các nodes
- [ ] Test GetFullGraph với pagination (limit + offset)
- [ ] Test GetFullGraph với tenant không có data → trả empty response
- [ ] Test GetFullGraph không trả nodes/edges của tenant khác
- [ ] Test GetFullGraph performance với graph lớn (>1000 nodes)
```

### 5.3 End-to-End Test

```
- [ ] SaveGraph 1 lần → verify node count = expected
- [ ] SaveGraph lần 2 (cùng data) → verify node count không đổi
- [ ] SaveGraph lần 2 (data thay đổi) → verify properties được cập nhật
- [ ] GetFullGraph → verify trả đúng toàn bộ graph đã save
- [ ] ba-agent-service gọi GetFullGraph qua gRPC → verify response mapping
```

---

## Appendix: Risk & Considerations

### MERGE Performance
- `MERGE` chậm hơn `CREATE` vì phải kiểm tra node/edge tồn tại trước khi tạo
- Với UNIQUE constraint/index trên `(app_id, tenant_id, id)`, MERGE sẽ sử dụng index lookup → performance tốt
- Benchmark cần thiết: so sánh throughput MERGE vs CREATE với batch 200 entities

### Neo4j Edition
- **Community Edition:** Không hỗ trợ NODE KEY constraint, chỉ hỗ trợ UNIQUE constraint trên single property
  - Workaround: Tạo composite property `_unique_key = app_id + "/" + tenant_id + "/" + id` và đặt UNIQUE constraint trên đó
- **Enterprise Edition:** Hỗ trợ NODE KEY constraint trên multiple properties

### Backward Compatibility
- Đổi `CREATE` → `MERGE` không breaking change — API signatures không thay đổi
- Thêm `GetFullGraph` RPC là additive change — không ảnh hưởng RPCs hiện có
- Cleanup script cần chạy trong maintenance window nếu data lớn
