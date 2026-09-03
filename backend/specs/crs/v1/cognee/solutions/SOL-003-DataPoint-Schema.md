# Solution: SOL-003 — DataPoint Custom Schema

**CR ID:** CR-COGNEE-003  
**Solution ID:** SOL-003  
**Priority:** Medium (Wave 2)  
**Architecture:** EXTEND `services/cognee-ingestion/` + Neo4j + Qdrant, Zero LLM

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md`:
- `cognee-ingestion/internal/usecase/add_data.go` — chỉ xử lý raw content (text, file, URL).
- Entity extraction đi qua `cognee-cognify` → LLM (Bifrost) → tốn token cho dữ liệu có schema rõ ràng.
- **Neo4j** upsert nodes bằng UUID — idempotency đã có.
- **Qdrant** upsert points bằng ID — update dễ dàng.
- PostgreSQL lưu `DataEntry` metadata — thêm bảng `cognee_datapoints` cho metadata versioning.

**Chiến lược:** DataPoints bypass LLM hoàn toàn. Ingestion service nhận structured data, tự mapping trực tiếp sang Neo4j nodes + Qdrant points mà không qua cognify pipeline.

---

## 2. Giải pháp chi tiết

### 2.1. [NEW] Domain — `services/cognee-ingestion/internal/domain/datapoint.go`

```go
// services/cognee-ingestion/internal/domain/datapoint.go

package domain

import (
    "time"
    "github.com/google/uuid"
)

// DataPoint — atomic knowledge unit với stable UUID identity
type DataPoint struct {
    ID          uuid.UUID              // Stable identity (deterministic UUID từ content hash)
    Version     int                    // Increment on update, start at 1
    DatasetID   uuid.UUID
    TenantID    string
    Type        string                 // Schema type: "Paper", "User", "Product", "Employee", ...
    Fields      map[string]any         // All field values (dynamic schema)
    IndexFields []string               // Only embed these fields into Qdrant
    Relations   []DataPointRelation    // Explicit edges to other DataPoints
    NodeSets    []string               // NodeSet tags (CR-002 integration)
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// DataPointRelation — explicit FK edge
type DataPointRelation struct {
    TargetID uuid.UUID
    Label    string    // Edge label: "authored_by", "belongs_to", "cites", "works_in"
    Weight   float64   // Default 1.0
}

// DeterministicUUID generates stable UUID from content hash
// Ensures idempotent ingestion: same content → same UUID
func DeterministicUUID(namespace, key string) uuid.UUID {
    return uuid.NewSHA1(uuid.NameSpaceURL, []byte(namespace+":"+key))
}
```

### 2.2. [NEW] Use Case — `services/cognee-ingestion/internal/usecase/add_data_points.go`

```go
// services/cognee-ingestion/internal/usecase/add_data_points.go

package usecase

import (
    "context"
    "fmt"
    "strings"

    "github.com/vnp-memory/services/cognee-ingestion/internal/domain"
    "github.com/vnp-memory/services/cognee-ingestion/internal/usecase/port"
)

type AddDataPointsRequest struct {
    DatasetID  domain.UUID
    TenantID   string
    DataPoints []domain.DataPoint
    NodeSets   []string  // Optional NodeSet tags (CR-002)
}

type AddDataPointsResult struct {
    Upserted int
    Created  int
    Updated  int
}

// AddDataPointsUseCase — bypasses LLM entirely
// Direct mapping: DataPoint schema → Neo4j nodes + Qdrant embeddings
type AddDataPointsUseCase struct {
    dataPointRepo port.DataPointRepository   // Postgres metadata
    graphRepo     port.GraphRepository        // Neo4j direct upsert
    vectorRepo    port.VectorRepository       // Qdrant upsert
    embedder      port.EmbedderClient         // Bifrost (only for embedding, no LLM)
    eventPub      port.EventPublisher
}

func NewAddDataPointsUseCase(
    dataPointRepo port.DataPointRepository,
    graphRepo port.GraphRepository,
    vectorRepo port.VectorRepository,
    embedder port.EmbedderClient,
    eventPub port.EventPublisher,
) *AddDataPointsUseCase {
    return &AddDataPointsUseCase{dataPointRepo, graphRepo, vectorRepo, embedder, eventPub}
}

func (uc *AddDataPointsUseCase) Execute(ctx context.Context, req AddDataPointsRequest) (*AddDataPointsResult, error) {
    result := &AddDataPointsResult{}

    for i := range req.DataPoints {
        dp := &req.DataPoints[i]

        // Attach DatasetID + TenantID + NodeSets
        dp.DatasetID = req.DatasetID
        dp.TenantID = req.TenantID
        if len(req.NodeSets) > 0 && len(dp.NodeSets) == 0 {
            dp.NodeSets = req.NodeSets
        }

        // Step 1: Validate
        if err := validateDataPoint(*dp); err != nil {
            return nil, fmt.Errorf("datapoint %s validation: %w", dp.ID, err)
        }

        // Step 2: Check existing version in Postgres
        existing, _ := uc.dataPointRepo.GetByID(ctx, dp.ID)
        if existing != nil {
            dp.Version = existing.Version + 1
            result.Updated++
        } else {
            dp.Version = 1
            result.Created++
        }

        // Step 3: Upsert Neo4j node with dynamic labels
        node := buildGraphNode(*dp)
        if err := uc.graphRepo.UpsertNode(ctx, node); err != nil {
            return nil, fmt.Errorf("upsert neo4j node %s: %w", dp.ID, err)
        }

        // Step 4: Create Neo4j edges from Relations
        for _, rel := range dp.Relations {
            if err := uc.graphRepo.UpsertEdge(ctx, domain.GraphEdge{
                ID:        buildEdgeID(dp.ID, rel.TargetID, rel.Label),
                Subject:   dp.ID.String(),
                Object:    rel.TargetID.String(),
                Predicate: rel.Label,
                Weight:    rel.Weight,
            }); err != nil {
                // Non-fatal: log and continue (target might not exist yet)
                _ = err
            }
        }

        // Step 5: Embed only IndexFields → Qdrant
        if err := uc.embedIndexFields(ctx, *dp); err != nil {
            // Non-fatal: embedding failure doesn't break graph
            _ = err
        }

        // Step 6: Persist metadata in Postgres
        uc.dataPointRepo.Upsert(ctx, *dp)
        result.Upserted++
    }

    // Emit NATS event
    uc.eventPub.Publish(ctx, "cognee.ingestion.datapoints.added", map[string]any{
        "dataset_id":    req.DatasetID.String(),
        "tenant_id":     req.TenantID,
        "datapoint_ids": extractIDs(req.DataPoints),
        "count":         len(req.DataPoints),
    })

    return result, nil
}

// buildGraphNode maps DataPoint → GraphNode for Neo4j
// Labels: [DataPoint, DataPoint.Type, ...NodeSets]
func buildGraphNode(dp domain.DataPoint) domain.GraphNode {
    labels := []string{"DataPoint", dp.Type}
    labels = append(labels, dp.NodeSets...)

    props := make(map[string]any)
    props["id"]         = dp.ID.String()
    props["dataset_id"] = dp.DatasetID.String()
    props["tenant_id"]  = dp.TenantID
    props["version"]    = dp.Version
    props["type"]       = dp.Type
    for k, v := range dp.Fields {
        props[k] = v
    }

    return domain.GraphNode{
        ID:         dp.ID.String(),
        Labels:     labels,
        Properties: props,
    }
}

// embedIndexFields embeds only the specified index_fields
func (uc *AddDataPointsUseCase) embedIndexFields(ctx context.Context, dp domain.DataPoint) error {
    if len(dp.IndexFields) == 0 { return nil }

    // Build text from index_fields only
    var parts []string
    for _, field := range dp.IndexFields {
        if val, ok := dp.Fields[field]; ok {
            parts = append(parts, fmt.Sprint(val))
        }
    }
    if len(parts) == 0 { return nil }

    text := strings.Join(parts, " ")
    vec, err := uc.embedder.Embed(ctx, text)
    if err != nil { return err }

    collectionName := fmt.Sprintf("cognee_%s", dp.TenantID)
    return uc.vectorRepo.UpsertPoint(ctx, collectionName, dp.ID.String(), vec, map[string]any{
        "dataset_id":   dp.DatasetID.String(),
        "datapoint_id": dp.ID.String(),
        "type":         dp.Type,
        "node_sets":    dp.NodeSets,
        "index_fields": dp.IndexFields,
    })
}

func validateDataPoint(dp domain.DataPoint) error {
    if dp.Type == "" { return fmt.Errorf("type is required") }
    if len(dp.Fields) == 0 { return fmt.Errorf("fields cannot be empty") }
    // index_fields must be subset of fields
    for _, field := range dp.IndexFields {
        if _, ok := dp.Fields[field]; !ok {
            return fmt.Errorf("index_field %q not found in fields", field)
        }
    }
    return nil
}
```

### 2.3. [NEW] Repository Port — `internal/usecase/port/output.go`

```go
// Thêm vào port/output.go

type DataPointRepository interface {
    GetByID(ctx context.Context, id uuid.UUID) (*domain.DataPoint, error)
    Upsert(ctx context.Context, dp domain.DataPoint) error
    ListByDataset(ctx context.Context, datasetID uuid.UUID, tenantID string, limit, offset int) ([]domain.DataPoint, error)
    Delete(ctx context.Context, id uuid.UUID) error
}
```

### 2.4. [MODIFY] gRPC Handler

```go
// services/cognee-ingestion/internal/adapter/grpc/handler.go

func (h *IngestionHandler) AddDataPoints(ctx context.Context, req *ingestionpb.AddDataPointsRequest) (*ingestionpb.AddDataPointsResponse, error) {
    datasetID, err := uuid.Parse(req.DatasetId)
    if err != nil { return nil, status.Errorf(codes.InvalidArgument, "invalid dataset_id") }

    dps := make([]domain.DataPoint, 0, len(req.DataPoints))
    for _, pbDp := range req.DataPoints {
        dpID, _ := uuid.Parse(pbDp.Id)
        fields := pbDp.Fields.AsMap()
        relations := mapProtoRelations(pbDp.Relations)

        dps = append(dps, domain.DataPoint{
            ID:          dpID,
            Type:        pbDp.Type,
            Fields:      fields,
            IndexFields: pbDp.IndexFields,
            Relations:   relations,
        })
    }

    result, err := h.addDataPointsUC.Execute(ctx, usecase.AddDataPointsRequest{
        DatasetID:  datasetID,
        TenantID:   req.TenantId,
        DataPoints: dps,
        NodeSets:   req.NodeSets,
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "add datapoints: %v", err) }

    return &ingestionpb.AddDataPointsResponse{
        Upserted: int32(result.Upserted),
        Created:  int32(result.Created),
        Updated:  int32(result.Updated),
    }, nil
}
```

### 2.5. [MODIFY] Proto — `api/proto/cognee/ingestion/v1/ingestion.proto`

```protobuf
syntax = "proto3";
import "google/protobuf/struct.proto";

service IngestionService {
  rpc AddData(AddDataRequest) returns (AddDataResponse);
  rpc AddDataPoints(AddDataPointsRequest) returns (AddDataPointsResponse);  // [NEW]
  rpc ListDatasets(ListDatasetsRequest) returns (ListDatasetsResponse);
  rpc DeleteDataset(DeleteDatasetRequest) returns (DeleteDatasetResponse);
}

message AddDataPointsRequest {
  string              dataset_id  = 1;
  string              tenant_id   = 2;
  repeated DataPoint  data_points = 3;
  repeated string     node_sets   = 4;
}

message DataPoint {
  string                   id           = 1;
  string                   type         = 2;
  google.protobuf.Struct   fields       = 3;
  repeated string          index_fields = 4;
  repeated Relation        relations    = 5;
}

message Relation {
  string target_id = 1;
  string label     = 2;
  double weight    = 3;
}

message AddDataPointsResponse {
  int32 upserted = 1;
  int32 created  = 2;
  int32 updated  = 3;
}
```

### 2.6. [NEW] PostgreSQL Schema

```sql
-- Migration: 0021_cognee_datapoints.up.sql

CREATE TABLE cognee_datapoints (
    id          UUID PRIMARY KEY,
    version     INT NOT NULL DEFAULT 1,
    dataset_id  UUID NOT NULL,
    tenant_id   TEXT NOT NULL,
    type        TEXT NOT NULL,
    fields      JSONB NOT NULL DEFAULT '{}',
    index_fields TEXT[] DEFAULT '{}',
    node_sets   TEXT[] DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cognee_datapoints_dataset ON cognee_datapoints(dataset_id, tenant_id);
CREATE INDEX idx_cognee_datapoints_type ON cognee_datapoints(tenant_id, type);
```

### 2.7. [MODIFY] Gateway Routes

```go
// gateway/internal/adapter/handler/router.go

r.Post("/api/v1/cognee/datasets/{id}/datapoints",
    h.ForwardTo("cognee-ingestion", "IngestionService/AddDataPoints"))
```

### 2.8. [MODIFY] `cognee-cognify` — subscribe `cognee.ingestion.datapoints.added`

DataPoints không cần full cognify pipeline. Cognify chỉ chạy optional **community detection** nếu cấu hình:

```go
// services/cognee-cognify/internal/adapter/event/subscriber.go

func (s *Subscriber) handleDataPointsAdded(msg *nats.Msg) {
    var evt struct {
        DatasetID string `json:"dataset_id"`
        TenantID  string `json:"tenant_id"`
        Count     int    `json:"count"`
    }
    json.Unmarshal(msg.Data, &evt)

    // Only run community detection if configured (heavy operation)
    if s.config.RunCommunityDetectOnDataPoints {
        s.cognifyJobQueue.Enqueue(CognifyJob{
            DatasetID: evt.DatasetID,
            TenantID:  evt.TenantID,
            Steps:     []string{"DETECT_COMMUNITY"},  // partial pipeline (CR-006)
        })
    }
}
```

---

## 3. Files

### [NEW]

| File | Mô tả |
|------|-------|
| `services/cognee-ingestion/internal/domain/datapoint.go` | DataPoint, DataPointRelation domain types |
| `services/cognee-ingestion/internal/usecase/add_data_points.go` | AddDataPointsUseCase |
| `services/cognee-ingestion/internal/adapter/repository/postgres/datapoint_repo.go` | DataPoint CRUD |
| `db/migrations/0021_cognee_datapoints.up.sql` | cognee_datapoints table |

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `api/proto/cognee/ingestion/v1/ingestion.proto` | + AddDataPoints RPC |
| `services/cognee-ingestion/internal/adapter/grpc/handler.go` | + AddDataPoints() |
| `services/cognee-ingestion/internal/usecase/port/output.go` | + DataPointRepository interface |
| `services/cognee-ingestion/internal/adapter/repository/neo4j/graph_repo.go` | + UpsertNode(labels[]) |
| `services/cognee-cognify/internal/adapter/event/subscriber.go` | + handleDataPointsAdded() |
| `gateway/internal/adapter/handler/router.go` | + POST /datasets/{id}/datapoints route |
| `apps/memory/internal/bootstrap/cognee.go` | + addDataPointsUC init |

---

## 4. Acceptance Criteria Mapping

| AC từ CR-COGNEE-003 | Covered by |
|--------------------|-----------|
| POST `/datasets/{id}/datapoints` → 201 Created | gRPC handler + gateway route |
| DataPoint = Neo4j Node với label = Type, properties = Fields | buildGraphNode() |
| Relations = Neo4j Edges với label = Relation.Label | UpsertEdge() |
| Chỉ index_fields được embed vào Qdrant | embedIndexFields() |
| Search SIMILARITY trên DataPoint fields hoạt động | Qdrant point upserted |
| Cùng UUID submit lại → upsert, version tăng | GetByID() version check |
| Không có LLM call trong flow AddDataPoints | Zero llmClient call in usecase |
