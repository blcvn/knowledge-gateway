# TASK-CE-003 — NodeSets: Ingestion Service

| Field | Value |
|-------|-------|
| **Task ID** | TASK-CE-003 |
| **Wave** | 2 |
| **Component** | `services/cognee-ingestion/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-002 §2.1, §2.2, §2.3 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-CE-001 |
| **Estimated** | 4h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** cognee-ingestion: 49 .go - NodeSets ingestion pipeline  
---

## Context

NodeSets là cơ chế memory scoping — gán "label tags" (`customer_123`, `project_alpha`) vào dữ liệu khi ingest, để sau này filter search results theo scope. 

**Chiến lược:** NodeSets là Neo4j multi-labels + Qdrant payload field. Không cần thêm DB table — chỉ propagate `node_sets[]` field qua pipeline.

---

## Goal

- Thêm `NodeSets []string` vào `DataEntry` domain type
- `AddDataUseCase.Execute()` nhận + attach `node_sets` vào mỗi entry
- NATS event `cognee.ingestion.data.ingested` bao gồm `node_sets`
- gRPC handler đọc `node_sets` từ proto request

---

## Target Files

| Action | File Path |
|--------|-----------|
| MODIFY | `services/cognee-ingestion/internal/domain/entity.go` |
| MODIFY | `services/cognee-ingestion/internal/usecase/add_data.go` |
| MODIFY | `services/cognee-ingestion/internal/adapter/event/publisher.go` |
| MODIFY | `services/cognee-ingestion/internal/adapter/grpc/handler.go` |

---

## Implementation

### MODIFY `domain/entity.go` — Add NodeSets field

```go
// services/cognee-ingestion/internal/domain/entity.go

type DataEntry struct {
    ID        uuid.UUID
    DatasetID uuid.UUID
    TenantID  string
    Content   string
    Type      ContentType    // TEXT | PDF | PDF_LAYOUT | HTML | URL | DOCX | CSV | TABULAR_FK
    URL       string
    Metadata  map[string]any
    NodeSets  []string       // [NEW] CR-002 — e.g. ["customer_123", "preferences", "contracts"]
    CreatedAt time.Time
}
```

### MODIFY `usecase/add_data.go` — Propagate NodeSets

```go
// services/cognee-ingestion/internal/usecase/add_data.go

// AddDataRequest — add NodeSets field
type AddDataRequest struct {
    DatasetID   uuid.UUID
    DatasetName string
    TenantID    string
    Items       []DataItem
    NodeSets    []string  // [NEW]
}

// AddDataItem
type DataItem struct {
    Content     string
    URL         string
    ContentType ContentType
    Metadata    map[string]any
    Config      *DataItemConfig
}

type DataItemConfig struct {
    PDFMode string  // "LAYOUT_AWARE" | "PLAIN_TEXT"
}

// Execute — attach NodeSets to each DataEntry
func (uc *AddDataUseCase) Execute(ctx context.Context, req AddDataRequest) (*AddDataResult, error) {
    dataset, err := uc.datasetRepo.GetOrCreate(ctx, req.DatasetID, req.DatasetName, req.TenantID)
    if err != nil { return nil, fmt.Errorf("get or create dataset: %w", err) }

    var entries []DataEntry
    for _, item := range req.Items {
        chunks, err := uc.extractChunks(ctx, item)
        if err != nil { return nil, fmt.Errorf("extract %s: %w", item.ContentType, err) }

        for _, chunk := range chunks {
            entry := DataEntry{
                ID:        uuid.New(),
                DatasetID: dataset.ID,
                TenantID:  req.TenantID,
                Content:   chunk.Content,
                Type:      item.ContentType,
                URL:       item.URL,
                Metadata:  mergeMetadata(item.Metadata, chunk.Metadata),
                NodeSets:  req.NodeSets,  // [ADDED] attach to every entry
                CreatedAt: time.Now(),
            }
            entries = append(entries, entry)
        }
    }

    // Persist entries (NodeSets stored as JSON column)
    if err := uc.dataEntryRepo.SaveBulk(ctx, entries); err != nil {
        return nil, fmt.Errorf("save entries: %w", err)
    }

    // Emit NATS event — include node_sets for downstream cognify
    uc.publisher.PublishDataIngested(ctx, DataIngestedEvent{
        DatasetID: dataset.ID.String(),
        TenantID:  req.TenantID,
        EntryIDs:  extractEntryIDs(entries),
        NodeSets:  req.NodeSets,  // [ADDED] propagate to cognify
    })

    return &AddDataResult{
        DatasetID: dataset.ID.String(),
        EntryIDs:  extractEntryIDs(entries),
        Count:     len(entries),
    }, nil
}
```

### MODIFY `adapter/event/publisher.go` — NodeSets in NATS event

```go
// services/cognee-ingestion/internal/adapter/event/publisher.go

type DataIngestedEvent struct {
    DatasetID string   `json:"dataset_id"`
    TenantID  string   `json:"tenant_id"`
    EntryIDs  []string `json:"entry_ids"`
    NodeSets  []string `json:"node_sets"`  // [NEW]
}

func (p *Publisher) PublishDataIngested(ctx context.Context, evt DataIngestedEvent) {
    data, err := json.Marshal(evt)
    if err != nil { return }
    p.natsConn.Publish("cognee.ingestion.data.ingested", data)
}
```

### MODIFY `adapter/grpc/handler.go` — Read node_sets from proto

```go
// services/cognee-ingestion/internal/adapter/grpc/handler.go

func (h *IngestionHandler) AddData(ctx context.Context, req *ingestionpb.AddDataRequest) (*ingestionpb.AddDataResponse, error) {
    datasetID := uuid.UUID{}
    if req.DatasetId != "" {
        var err error
        datasetID, err = uuid.Parse(req.DatasetId)
        if err != nil { return nil, status.Errorf(codes.InvalidArgument, "invalid dataset_id") }
    }

    items := make([]usecase.DataItem, 0, len(req.Items))
    for _, pbItem := range req.Items {
        item := usecase.DataItem{
            Content:     pbItem.Content,
            URL:         pbItem.Url,
            ContentType: domain.ContentType(pbItem.ContentType),
            Metadata:    pbItem.Metadata,
        }
        if pbItem.Config != nil {
            item.Config = &usecase.DataItemConfig{PDFMode: pbItem.Config.PdfMode}
        }
        items = append(items, item)
    }

    result, err := h.addDataUC.Execute(ctx, usecase.AddDataRequest{
        DatasetID:   datasetID,
        DatasetName: req.DatasetName,
        TenantID:    req.TenantId,
        Items:       items,
        NodeSets:    req.NodeSets,  // [NEW] propagate node_sets
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "add data: %v", err) }

    return &ingestionpb.AddDataResponse{
        DatasetId: result.DatasetID,
        EntryIds:  result.EntryIDs,
        Count:     int32(result.Count),
    }, nil
}
```

---

## Verification

```bash
cd services/cognee-ingestion
go build ./...
go test ./internal/usecase/... -run TestAddData_NodeSets -v
```

**Unit test:**
```go
func TestAddData_NodeSets_PropagatedToEntries(t *testing.T) {
    uc := NewAddDataUseCase(mockDatasetRepo, mockDataEntryRepo, mockExtractor, mockPublisher)
    
    result, err := uc.Execute(ctx, AddDataRequest{
        TenantID: "tenant-1",
        Items: []DataItem{{Content: "Alice works at TechCorp", ContentType: ContentTypeText}},
        NodeSets: []string{"customer_alice", "project_gamma"},
    })
    
    require.NoError(t, err)
    
    // Check saved entries have node_sets
    savedEntries := mockDataEntryRepo.LastSaved()
    for _, entry := range savedEntries {
        assert.Equal(t, []string{"customer_alice", "project_gamma"}, entry.NodeSets)
    }
    
    // Check NATS event has node_sets
    publishedEvent := mockPublisher.LastEvent()
    assert.Equal(t, []string{"customer_alice", "project_gamma"}, publishedEvent.NodeSets)
}
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| `POST /api/v1/cognee/add` với `node_sets` → DataEntry có NodeSets | ✅ |
| NATS event `cognee.ingestion.data.ingested` có `node_sets` field | ✅ |
| Không truyền `node_sets` → `[]` (empty, không lỗi) | ✅ |
| Multiple items → tất cả đều có cùng `node_sets` | ✅ |
