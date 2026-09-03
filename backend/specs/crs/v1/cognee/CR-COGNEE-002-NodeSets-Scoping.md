# Change Request: CR-COGNEE-002 — NodeSets Memory Scoping

**CR ID:** CR-COGNEE-002  
**Component:** `services/kg-service` | `gateway`  
**Priority:** High  
**Status:** Implemented  
**Reference:** Cognee PRD §4.2, SRS FR-ING-02, URD UR-ING-02  
**Spec:** `references/cognee/specs/services/02-cognee-ingestion.md`, `04-cognee-search.md`

---

## 1. Mô tả

Thêm hỗ trợ **NodeSets** vào hệ thống ingestion và search. NodeSets là cơ chế **tag-based memory partitioning** cho phép gắn nhãn dữ liệu theo nhiều chiều độc lập (tenant, user, project, topic, workflow) và sau đó tìm kiếm **chỉ trong phạm vi** các tag đó — nhanh hơn nhiều so với filter theo `dataset_id`.

---

## 2. Vấn đề hiện tại

`DataEntry` trong `services/cognee-ingestion/internal/domain/entity.go` hiện không có trường `NodeSets`. Dataset-level isolation là đủ cho multi-tenancy, nhưng không đủ cho các use case như:
- "Tìm kiếm chỉ trong context của user `u_123` và topic `contracts`".
- "Phân quyền đọc graph theo project, không theo dataset".

---

## 3. Thay đổi đề xuất

### 3.1. Service: `services/cognee-ingestion`

**[MODIFY]** `internal/domain/entity.go`

```go
type DataEntry struct {
    // ... existing fields ...
    NodeSets []string `json:"node_sets"` // [NEW] e.g. ["customer_123", "preferences"]
}
```

**[MODIFY]** `internal/usecase/add_data.go`

```go
// AddDataRequest — add node_sets to use case input
type AddDataRequest struct {
    DatasetID   uuid.UUID
    TenantID    string
    Items       []DataItem
    NodeSets    []string  // [NEW]
}
```

Khi tạo `DataEntry`, gắn `NodeSets` vào. Khi emit event `cognee.ingestion.data.ingested`, đính kèm NodeSets vào payload để Cognify service có thể gắn tag khi build graph.

**[MODIFY]** Proto:
```protobuf
// api/proto/cognee/ingestion/v1/ingestion.proto
message AddDataRequest {
  string         dataset_id = 1;
  string         tenant_id  = 2;
  repeated DataItem items   = 3;
  repeated string node_sets = 4; // [NEW]
}
```

### 3.2. Service: `services/cognee-cognify`

**[MODIFY]** `internal/usecase/add_datapoints.go`

Khi persist `GraphNode` vào Neo4j, đính kèm NodeSets dưới dạng labels:
```go
// Neo4j: node sẽ có thêm labels cho từng NodeSet
// Node(:Concept:customer_123:preferences)
for _, tag := range nodeSets {
    node.AddLabel(tag)
}
```

Khi index lên Qdrant, gắn NodeSets vào `payload` của mỗi point để hỗ trợ filtering:
```go
point.Payload["node_sets"] = nodeSets // for Qdrant filter
```

**[MODIFY]** `internal/adapter/event/subscriber.go`

Nhận `node_sets` từ event payload `cognee.ingestion.data.ingested`:
```go
type DataIngestedPayload struct {
    DatasetID string   `json:"dataset_id"`
    EntryIDs  []string `json:"entry_ids"`
    TenantID  string   `json:"tenant_id"`
    NodeSets  []string `json:"node_sets"` // [NEW]
}
```

### 3.3. Service: `services/cognee-search`

**[MODIFY]** `internal/usecase/search.go` + `internal/usecase/port/input.go`

```go
type SearchRequest struct {
    Query      string
    Strategies []SearchStrategy
    DatasetID  *uuid.UUID
    TenantID   string
    NodeSets   []string  // [NEW] — filter by node tags
    TopK       int
    SaveInteraction bool
}
```

**[MODIFY]** Mỗi Retriever trong `internal/adapter/retriever/`:
- **Vector (Qdrant)**: Thêm `filter: {"must": [{"key": "node_sets", "match": {"any": nodeSets}}]}` vào query.
- **Graph (Neo4j)**: Thêm label filter vào Cypher: `MATCH (n) WHERE all(tag IN $node_sets WHERE tag IN labels(n))`.

**[MODIFY]** Proto:
```protobuf
// api/proto/cognee/search/v1/search.proto
message SearchRequest {
  string            query       = 1;
  repeated Strategy strategies  = 2;
  string            dataset_id  = 3;
  string            tenant_id   = 4;
  repeated string   node_sets   = 5; // [NEW]
  int32             top_k       = 6;
  bool              save_interaction = 7;
}
```

### 3.4. Service: `services/vnp-gateway`

**[MODIFY]** `internal/adapter/http/cognee_routes.go`

`POST /api/v1/cognee/add`:
```json
{
  "dataset_name": "crm",
  "items": [...],
  "node_sets": ["customer_123", "preferences"]
}
```

`POST /api/v1/cognee/search`:
```json
{
  "query": "What does this customer prefer?",
  "strategies": ["GRAPH_COMPLETION"],
  "node_sets": ["customer_123", "preferences"]
}
```

---

## 4. Traceability

| Item | Ref |
|---|---|
| Ingestion gRPC port | `cognee-ingestion:9011` |
| Cognify gRPC port | `cognee-cognify:9012` |
| Search gRPC port | `cognee-search:9013` |
| Neo4j label strategy | Multi-label per node |
| Qdrant filter strategy | Payload field `node_sets` |
| NATS payload change | `cognee.ingestion.data.ingested` + `node_sets[]` |

---

## 5. Acceptance Criteria

- [x] `POST /api/v1/cognee/add` nhận được `node_sets` array và lưu vào `DataEntry.NodeSets`.
- [x] Sau cognify, node trên Neo4j có thêm labels tương ứng với từng NodeSet tag.
- [x] Sau cognify, Qdrant points có `payload.node_sets` chứa các tags.
- [x] `POST /api/v1/cognee/search` với `node_sets: ["customer_123"]` chỉ trả về results từ những documents được tag `"customer_123"`.
- [x] Search với NodeSet filter có performance cao hơn full scan (kiểm tra bằng Qdrant filter benchmark).
- [x] NodeSet filter hoạt động đúng với tất cả retrievers: SIMILARITY, GRAPH_COMPLETION, KEYWORD, CHUNKS.

---

## 6. Implementation Notes

**Implemented in:** `services/kg-service` + `gateway` (MERGE-P2-T2)

| File | Change |
|------|--------|
| `services/kg-service/internal/domain/cognee/entity.go` | `[MODIFY]` `DataItem.NodeSets []string` + `[NEW]` `SearchRequest` entity với `NodeSets []string` |
| `services/kg-service/internal/adapter/cognee/noop.go` | `[MODIFY]` `Interface` + `NoopClient` thêm `SearchWithNodeSets()` |
| `services/kg-service/internal/adapter/cognee/client.go` | `[MODIFY]` `HTTPClient.SearchWithNodeSets()` → gửi `node_sets` payload đến Cognee Python `/api/v1/search` |
| `services/kg-service/internal/usecase/cognee/service.go` | `[NEW]` `NodeSetsSearchUseCase.SearchWithNodeSets()` |
| `services/kg-service/internal/adapter/grpc/router.go` | `[MODIFY]` `CogneeSearch` handler kiểm tra `node_sets` → route đến `nsSearch` nếu có |
| `services/kg-service/cmd/server/main.go` | `[MODIFY]` Wire up `NodeSetsSearchUseCase` |

**Note:** NodeSets được truyền qua payload đến Cognee Python service — việc filter trên Neo4j labels và Qdrant payload.node_sets là trách nhiệm của Cognee Python, không phải kg-service.
