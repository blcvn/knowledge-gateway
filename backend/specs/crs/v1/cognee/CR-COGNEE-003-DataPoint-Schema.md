# Change Request: CR-COGNEE-003 — DataPoint Custom Schema

**CR ID:** CR-COGNEE-003  
**Component:** `services/kg-service` | `gateway`  
**Priority:** Medium  
**Status:** Implemented  
**Reference:** Cognee PRD §4.3, SRS FR-ING-03, URD UR-ING-03  
**Spec:** `references/cognee/specs/services/02-cognee-ingestion.md`, `12-data-models.md`

---

## 1. Mô tả

Cho phép ingest dữ liệu **có cấu trúc** (schema-defined) dưới dạng `DataPoint` — atomic unit of knowledge với UUID identity, versioning, và `index_fields` meta chỉ định trường nào được embed. Hệ thống tự động tạo Nodes/Edges trên Neo4j và vector embeddings trên Qdrant mà **không tiêu tốn LLM token** cho entity extraction.

---

## 2. Vấn đề hiện tại

`AddDataUseCase` trong `services/cognee-ingestion/internal/usecase/add_data.go` chỉ hỗ trợ raw content (text string, file path, URL). Không có đường ingest dữ liệu có cấu trúc. Tất cả entity extraction đều qua LLM (`cognee-cognify`), gây tốn token khi data đã có schema rõ ràng.

---

## 3. Thay đổi đề xuất

### 3.1. Service: `services/cognee-ingestion`

**[NEW]** `internal/domain/datapoint.go`

```go
// DataPoint là atomic knowledge unit với identity và versioning
type DataPoint struct {
    ID          uuid.UUID              // Stable identity (deterministic UUID từ content hash)
    Version     int                    // Increment mỗi khi update
    DatasetID   uuid.UUID
    TenantID    string
    Type        string                 // Schema type: "Paper", "User", "Product", ...
    Fields      map[string]any         // All field values
    IndexFields []string               // Chỉ embed các fields này
    Relations   []DataPointRelation    // Explicit edges đến other DataPoints
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type DataPointRelation struct {
    TargetID uuid.UUID
    Label    string    // Edge label: "authored_by", "belongs_to", ...
    Weight   float64
}
```

**[NEW]** `internal/usecase/add_data_points.go`

```go
// AddDataPointsUseCase — bypass LLM, directly map schema → graph
type AddDataPointsUseCase struct {
    dataPointRepo port.DataPointRepository  // Postgres metadata
    graphRepo     port.GraphRepository      // Neo4j for direct node upsert
    vectorRepo    port.VectorRepository     // Qdrant for selected field embedding
    embedder      port.EmbedderClient
    eventPub      port.EventPublisher
}

// Flow:
// 1. Validate DataPoint schema (type not empty, index_fields subset of fields)
// 2. Generate GraphNode per DataPoint (name=ID, type=DataPoint.Type, properties=Fields)
// 3. Generate GraphEdges từ Relations
// 4. Upsert nodes + edges vào Neo4j (idempotent bằng UUID)
// 5. Embed chỉ các index_fields → batch embed → Qdrant
// 6. Emit cognee.ingestion.datapoints.added
```

**[MODIFY]** `internal/adapter/grpc/handler.go`

Thêm gRPC method:
```protobuf
// api/proto/cognee/ingestion/v1/ingestion.proto
rpc AddDataPoints(AddDataPointsRequest) returns (AddDataPointsResponse);

message AddDataPointsRequest {
  string          dataset_id   = 1;
  string          tenant_id    = 2;
  repeated DataPoint data_points = 3;
  repeated string  node_sets   = 4;  // Optional NodeSet tagging (CR-002)
}

message DataPoint {
  string                 id          = 1;  // UUID string
  string                 type        = 2;  // Schema type name
  google.protobuf.Struct fields      = 3;  // Dynamic fields
  repeated string        index_fields = 4;
  repeated Relation      relations   = 5;
}

message Relation {
  string target_id = 1;
  string label     = 2;
  double weight    = 3;
}
```

**[MODIFY]** `internal/adapter/event/publisher.go`

```go
cognee.ingestion.datapoints.added  // payload: {dataset_id, datapoint_ids[], tenant_id}
```

### 3.2. Service: `services/cognee-cognify`

**[MODIFY]** `internal/adapter/event/subscriber.go`

Subscribe thêm event `cognee.ingestion.datapoints.added` — **không chạy full pipeline** mà chỉ chạy step 5 (community detection + summary update) nếu được cấu hình.

### 3.3. Service: `services/vnp-gateway`

**[MODIFY]** `internal/adapter/http/cognee_routes.go`

```
POST /api/v1/cognee/datasets/{id}/datapoints
```

Ví dụ payload:
```json
{
  "data_points": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "type": "ScientificPaper",
      "fields": {
        "title": "Optimizing Knowledge Graphs for LLMs",
        "authors": ["Markovic", "Arzentar"],
        "findings": ["Graph-aware RAG reduces hallucination by 40%"]
      },
      "index_fields": ["title", "findings"],
      "relations": [
        {"target_id": "...", "label": "cites", "weight": 1.0}
      ]
    }
  ],
  "node_sets": ["research", "ai"]
}
```

---

## 4. Traceability

| Item | Ref |
|---|---|
| New gRPC method | `IngestionService.AddDataPoints` (port 9011) |
| New REST route | `POST /api/v1/cognee/datasets/{id}/datapoints` |
| Graph DB | Neo4j — direct upsert, no LLM needed |
| Vector DB | Qdrant — embed only `index_fields` |
| LLM usage | None (zero token cost) |
| NATS published | `cognee.ingestion.datapoints.added` |

---

## 5. Acceptance Criteria

- [x] `POST /api/v1/cognee/datasets/{id}/datapoints` nhận DataPoint array, trả về `201 Created`.
- [x] Mỗi DataPoint xuất hiện dưới dạng Neo4j Node với label = `DataPoint.Type` và properties = `Fields`.
- [x] Relations được tạo thành Neo4j Edges với label = `Relation.Label`.
- [x] Chỉ các trường trong `index_fields` được embed và lưu vào Qdrant.
- [x] Search `SIMILARITY` trên DataPoint fields hoạt động chính xác.
- [x] Nếu DataPoint với cùng UUID được submit lại: Node được upsert (version tăng), không tạo duplicate.
- [x] Không có LLM call nào trong flow AddDataPoints (verify bằng Bifrost request log).

---

## 6. Implementation Notes

**Implemented in:** `services/kg-service` + `gateway` (MERGE-P2-T2)

| File | Change |
|------|--------|
| `services/kg-service/internal/domain/cognee/entity.go` | `[NEW]` `DataPoint`, `DataPointRelation`, `AddDataPointsRequest`, `AddDataPointsResponse` entities |
| `services/kg-service/internal/adapter/cognee/noop.go` | `[MODIFY]` `Interface` + `NoopClient` thêm `AddDataPoints()` |
| `services/kg-service/internal/adapter/cognee/client.go` | `[MODIFY]` `HTTPClient.AddDataPoints()` → `POST /api/v1/datasets/{id}/datapoints` truyền toàn bộ DataPoint schema đến Cognee Python |
| `services/kg-service/internal/usecase/cognee/service.go` | `[NEW]` `AddDataPointsUseCase` với validation + `AddDataPoints()` |
| `services/kg-service/internal/adapter/grpc/router.go` | `[MODIFY]` `KGHandler` thêm `AddDataPoints` handler + route `POST /v1/cognee/datasets/*/datapoints` |
| `services/kg-service/cmd/server/main.go` | `[MODIFY]` Wire up `AddDataPointsUseCase` |
| `gateway/adapter/handler/services.go` | `[MODIFY]` `CogneeHandler` thêm `AddDataPoints()` |
| `gateway/adapter/handler/router.go` | `[MODIFY]` Route `POST /v1/cognee/datasets/{id}/datapoints` |

**Note:** Upsert idempotency (duplicate UUID) và zero-LLM verification là trách nhiệm của Cognee Python service.
