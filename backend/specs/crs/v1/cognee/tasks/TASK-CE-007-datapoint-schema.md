# TASK-CE-007 — DataPoint Schema (Structured Ingestion, Zero LLM)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-CE-007 |
| **Wave** | 2 |
| **Component** | `services/cognee-ingestion/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-003 §2.1 → §2.8 |
| **Priority** | High |
| **Depends On** | TASK-CE-001, TASK-CE-011 |
| **Estimated** | 5h |

---

## Context

DataPoints là cơ chế ingestion **structured data không qua LLM**. Người dùng truyền schema rõ ràng (type, fields, relations) → system tự mapping thành Neo4j nodes + Qdrant embeddings mà không cần entity extraction.

**Key constraint:** Zero LLM call trong `AddDataPoints` flow.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/cognee-ingestion/internal/domain/datapoint.go` |
| CREATE | `services/cognee-ingestion/internal/usecase/add_data_points.go` |
| CREATE | `services/cognee-ingestion/internal/adapter/repository/postgres/datapoint_repo.go` |
| MODIFY | `services/cognee-ingestion/internal/usecase/port/output.go` |
| MODIFY | `services/cognee-ingestion/internal/adapter/grpc/handler.go` |
| MODIFY | `services/cognee-cognify/internal/adapter/event/subscriber.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |
| MODIFY | `apps/memory/internal/bootstrap/cognee.go` |

---

## Implementation

### File 1: `domain/datapoint.go`

```go
package domain

import (
    "crypto/sha1"
    "fmt"
    "time"

    "github.com/google/uuid"
)

// DataPoint — atomic knowledge unit with stable UUID identity
type DataPoint struct {
    ID          uuid.UUID              // Stable: deterministic UUID from content hash
    Version     int                    // Increment on update, start at 1
    DatasetID   uuid.UUID
    TenantID    string
    Type        string                 // Schema type: "Paper", "User", "Product", "Employee"
    Fields      map[string]any         // All field values (dynamic schema)
    IndexFields []string               // Only these fields embedded into Qdrant
    Relations   []DataPointRelation    // Explicit edges to other DataPoints
    NodeSets    []string               // NodeSet tags (CR-002 integration)
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// DataPointRelation — explicit FK edge to another DataPoint
type DataPointRelation struct {
    TargetID uuid.UUID
    Label    string    // Edge label: "authored_by", "belongs_to", "cites", "works_in"
    Weight   float64   // Default 1.0
}

// DeterministicUUID generates stable UUID from namespace + key
// Same content → same UUID → idempotent ingestion
func DeterministicUUID(namespace, key string) uuid.UUID {
    return uuid.NewSHA1(uuid.NameSpaceURL, []byte(namespace+":"+key))
}

// ContentHash generates SHA1 hash of fields for deterministic ID
func ContentHash(typeName string, fields map[string]any) string {
    h := sha1.New()
    fmt.Fprintf(h, "%s:%v", typeName, fields)
    return fmt.Sprintf("%x", h.Sum(nil))
}
```

### File 2: `usecase/add_data_points.go`

```go
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
    dataPointRepo port.DataPointRepository
    graphRepo     port.GraphRepository
    vectorRepo    port.VectorRepository
    embedder      port.EmbedderClient  // Bifrost (embedding only, NO LLM chat)
    eventPub      port.EventPublisher
}

func NewAddDataPointsUseCase(
    dataPointRepo port.DataPointRepository,
    graphRepo     port.GraphRepository,
    vectorRepo    port.VectorRepository,
    embedder      port.EmbedderClient,
    eventPub      port.EventPublisher,
) *AddDataPointsUseCase {
    return &AddDataPointsUseCase{dataPointRepo, graphRepo, vectorRepo, embedder, eventPub}
}

func (uc *AddDataPointsUseCase) Execute(ctx context.Context, req AddDataPointsRequest) (*AddDataPointsResult, error) {
    result := &AddDataPointsResult{}

    for i := range req.DataPoints {
        dp := &req.DataPoints[i]

        // Attach dataset context + NodeSets
        dp.DatasetID = req.DatasetID
        dp.TenantID  = req.TenantID
        if len(req.NodeSets) > 0 && len(dp.NodeSets) == 0 {
            dp.NodeSets = req.NodeSets
        }

        // Step 1: Validate
        if err := validateDataPoint(*dp); err != nil {
            return nil, fmt.Errorf("datapoint %s: %w", dp.ID, err)
        }

        // Step 2: Version check (upsert semantics)
        existing, _ := uc.dataPointRepo.GetByID(ctx, dp.ID)
        if existing != nil {
            dp.Version = existing.Version + 1
            result.Updated++
        } else {
            dp.Version = 1
            result.Created++
        }

        // Step 3: Upsert Neo4j node with type label + NodeSet labels
        node := buildGraphNode(*dp)
        if err := uc.graphRepo.UpsertNode(ctx, node); err != nil {
            return nil, fmt.Errorf("upsert neo4j node %s: %w", dp.ID, err)
        }

        // Step 4: Create Neo4j edges from Relations (non-fatal if target not yet ingested)
        for _, rel := range dp.Relations {
            edge := domain.GraphEdge{
                ID:        buildEdgeID(dp.ID, rel.TargetID, rel.Label),
                Subject:   dp.ID.String(),
                Object:    rel.TargetID.String(),
                Predicate: rel.Label,
                Weight:    rel.Weight,
            }
            _ = uc.graphRepo.UpsertEdge(ctx, edge)  // non-fatal
        }

        // Step 5: Embed only IndexFields → Qdrant (non-fatal)
        _ = uc.embedIndexFields(ctx, *dp)

        // Step 6: Persist metadata in Postgres
        uc.dataPointRepo.Upsert(ctx, *dp)
        result.Upserted++
    }

    // Emit NATS event
    uc.eventPub.Publish(ctx, "cognee.ingestion.datapoints.added", map[string]any{
        "dataset_id": req.DatasetID.String(),
        "tenant_id":  req.TenantID,
        "count":      len(req.DataPoints),
    })

    return result, nil
}

// buildGraphNode maps DataPoint → GraphNode for Neo4j
// Labels: [DataPoint, DataPoint.Type, ...NodeSets]
func buildGraphNode(dp domain.DataPoint) domain.GraphNode {
    labels := []string{"DataPoint", dp.Type}
    labels = append(labels, dp.NodeSets...)

    props := map[string]any{
        "id":         dp.ID.String(),
        "dataset_id": dp.DatasetID.String(),
        "tenant_id":  dp.TenantID,
        "version":    dp.Version,
        "type":       dp.Type,
    }
    for k, v := range dp.Fields { props[k] = v }

    return domain.GraphNode{
        ID:         dp.ID.String(),
        Name:       extractName(dp.Fields),
        Labels:     labels,
        Properties: props,
    }
}

// embedIndexFields embeds only the specified index_fields into Qdrant
func (uc *AddDataPointsUseCase) embedIndexFields(ctx context.Context, dp domain.DataPoint) error {
    if len(dp.IndexFields) == 0 { return nil }

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
    })
}

func validateDataPoint(dp domain.DataPoint) error {
    if dp.Type == "" { return fmt.Errorf("type is required") }
    if len(dp.Fields) == 0 { return fmt.Errorf("fields cannot be empty") }
    for _, field := range dp.IndexFields {
        if _, ok := dp.Fields[field]; !ok {
            return fmt.Errorf("index_field %q not found in fields", field)
        }
    }
    return nil
}

func buildEdgeID(src, dst domain.UUID, label string) string {
    return domain.DeterministicUUID(fmt.Sprintf("%s_%s", label, dst.String()), src.String()).String()
}

func extractName(fields map[string]any) string {
    for _, key := range []string{"name", "title", "label", "id"} {
        if v, ok := fields[key]; ok { return fmt.Sprint(v) }
    }
    return "unnamed"
}
```

### File 3: `adapter/repository/postgres/datapoint_repo.go`

```go
package postgres

import (
    "context"
    "encoding/json"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
    "github.com/vnp-memory/services/cognee-ingestion/internal/domain"
)

type DataPointRepo struct {
    db *pgxpool.Pool
}

func NewDataPointRepo(db *pgxpool.Pool) *DataPointRepo {
    return &DataPointRepo{db: db}
}

func (r *DataPointRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.DataPoint, error) {
    row := r.db.QueryRow(ctx, `
        SELECT id, version, dataset_id, tenant_id, type, fields, index_fields, node_sets
        FROM cognee_datapoints WHERE id = $1
    `, id)

    var dp domain.DataPoint
    var fieldsJSON []byte
    err := row.Scan(&dp.ID, &dp.Version, &dp.DatasetID, &dp.TenantID, &dp.Type,
        &fieldsJSON, &dp.IndexFields, &dp.NodeSets)
    if err != nil { return nil, err }
    json.Unmarshal(fieldsJSON, &dp.Fields)
    return &dp, nil
}

func (r *DataPointRepo) Upsert(ctx context.Context, dp domain.DataPoint) error {
    fieldsJSON, _ := json.Marshal(dp.Fields)
    _, err := r.db.Exec(ctx, `
        INSERT INTO cognee_datapoints (id, version, dataset_id, tenant_id, type, fields, index_fields, node_sets, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
        ON CONFLICT (id) DO UPDATE
        SET version = EXCLUDED.version, fields = EXCLUDED.fields,
            index_fields = EXCLUDED.index_fields, node_sets = EXCLUDED.node_sets,
            updated_at = NOW()
    `, dp.ID, dp.Version, dp.DatasetID, dp.TenantID, dp.Type,
        fieldsJSON, dp.IndexFields, dp.NodeSets)
    return err
}

func (r *DataPointRepo) ListByDataset(ctx context.Context, datasetID uuid.UUID, tenantID string, limit, offset int) ([]domain.DataPoint, error) {
    rows, err := r.db.Query(ctx, `
        SELECT id, version, type, fields FROM cognee_datapoints
        WHERE dataset_id = $1 AND tenant_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4
    `, datasetID, tenantID, limit, offset)
    if err != nil { return nil, err }
    defer rows.Close()

    var result []domain.DataPoint
    for rows.Next() {
        var dp domain.DataPoint
        var fieldsJSON []byte
        rows.Scan(&dp.ID, &dp.Version, &dp.Type, &fieldsJSON)
        json.Unmarshal(fieldsJSON, &dp.Fields)
        result = append(result, dp)
    }
    return result, nil
}
```

### MODIFY `grpc/handler.go` — AddDataPoints handler

```go
func (h *IngestionHandler) AddDataPoints(ctx context.Context, req *ingestionpb.AddDataPointsRequest) (*ingestionpb.AddDataPointsResponse, error) {
    datasetID, err := uuid.Parse(req.DatasetId)
    if err != nil { return nil, status.Errorf(codes.InvalidArgument, "invalid dataset_id") }

    dps := make([]domain.DataPoint, 0, len(req.DataPoints))
    for _, pbDp := range req.DataPoints {
        dpID, _ := uuid.Parse(pbDp.Id)
        fields := pbDp.Fields.AsMap()

        var relations []domain.DataPointRelation
        for _, rel := range pbDp.Relations {
            targetID, _ := uuid.Parse(rel.TargetId)
            relations = append(relations, domain.DataPointRelation{
                TargetID: targetID,
                Label:    rel.Label,
                Weight:   rel.Weight,
            })
        }

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

### MODIFY `gateway/router.go` — Add datapoints route

```go
r.Post("/api/v1/cognee/datasets/{id}/datapoints",
    h.ForwardTo("cognee-ingestion", "IngestionService/AddDataPoints"))
```

### MODIFY `cognee-cognify/subscriber.go` — Handle datapoints.added event

```go
func (s *Subscriber) handleDataPointsAdded(msg *nats.Msg) {
    var evt struct {
        DatasetID string `json:"dataset_id"`
        TenantID  string `json:"tenant_id"`
        Count     int    `json:"count"`
    }
    json.Unmarshal(msg.Data, &evt)

    // Optional: run community detection only (not full pipeline)
    if s.config.RunCommunityDetectOnDataPoints {
        s.cognifyJobQueue.Enqueue(CognifyJob{
            DatasetID: evt.DatasetID,
            TenantID:  evt.TenantID,
            Steps:     []string{"DETECT_COMMUNITY"},
        })
    }
}

// Register the subscriber in Init():
s.natsConn.Subscribe("cognee.ingestion.datapoints.added", s.handleDataPointsAdded)
```

---

## Verification

```bash
cd services/cognee-ingestion
go build ./...
go test ./internal/usecase/... -run TestAddDataPoints -v
```

**Test Zero LLM:**
```go
func TestAddDataPoints_ZeroLLMCalls(t *testing.T) {
    mockLLM := &MockLLMClient{}
    uc := NewAddDataPointsUseCase(mockRepo, mockGraph, mockVector, mockEmbedder, mockEvent)
    
    _, err := uc.Execute(ctx, AddDataPointsRequest{
        TenantID: "t1",
        DataPoints: []domain.DataPoint{{
            ID:    domain.DeterministicUUID("Employee", "e1"),
            Type:  "Employee",
            Fields: map[string]any{"name": "Alice", "dept_id": "d1"},
            IndexFields: []string{"name"},
        }},
    })
    
    require.NoError(t, err)
    assert.Equal(t, 0, mockLLM.ChatCallCount(), "LLM must NOT be called")
    assert.Equal(t, 1, mockVector.UpsertCallCount(), "Embedding called once")
}
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| `POST /datasets/{id}/datapoints` → 201 Created | ✅ |
| DataPoint = Neo4j Node với label = Type | ✅ |
| Relations = Neo4j Edges | ✅ |
| Chỉ `index_fields` được embed vào Qdrant | ✅ |
| Same UUID submit lại → version tăng (upsert) | ✅ |
| Zero LLM call trong flow | ✅ |
