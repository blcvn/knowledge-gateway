# Solution: SOL-002 — NodeSets Memory Scoping

**CR ID:** CR-COGNEE-002  
**Solution ID:** SOL-002  
**Priority:** High (Wave 1)  
**Architecture:** EXTEND `services/cognee-ingestion` + `services/cognee-search` + `services/cognee-cognify`

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md`:
- **Neo4j** — graph store: nodes có labels (`:Concept`, `:Entity`, v.v.) và properties.
- **Qdrant** — vector store: collections per tenant, payload field filtering đã supported.
- `cognee-ingestion` gRPC port 9011, `cognee-search` port 9013.
- Monolith: `InProcessRegistry` bufconn giao tiếp nội bộ.
- NATS event `cognee.ingestion.data.ingested` truyền từ ingestion → cognify.

**Chiến lược NodeSets:** NodeSets là labels trên Neo4j nodes + payload field trên Qdrant points. Không cần thêm database table. Chỉ cần propagate `node_sets[]` qua data pipeline.

---

## 2. Giải pháp chi tiết

### 2.1. [MODIFY] `services/cognee-ingestion/internal/domain/entity.go`

```go
// services/cognee-ingestion/internal/domain/entity.go

type DataEntry struct {
    ID        uuid.UUID
    DatasetID uuid.UUID
    TenantID  string
    Content   string
    Type      ContentType   // TEXT | PDF | PDF_LAYOUT | HTML | URL | DOCX | CSV | TABULAR_FK
    URL       string
    Metadata  map[string]any
    NodeSets  []string    // [NEW] e.g. ["customer_123", "preferences", "contracts"]
    CreatedAt time.Time
}
```

### 2.2. [MODIFY] `services/cognee-ingestion/internal/usecase/add_data.go`

```go
// AddDataRequest — propagate node_sets
type AddDataRequest struct {
    DatasetID uuid.UUID
    TenantID  string
    Items     []DataItem
    NodeSets  []string    // [NEW]
}

// In AddDataUseCase.Execute():
func (uc *AddDataUseCase) Execute(ctx context.Context, req AddDataRequest) (*AddDataResult, error) {
    var entries []DataEntry
    for _, item := range req.Items {
        entry := DataEntry{
            ID:        uuid.New(),
            DatasetID: req.DatasetID,
            TenantID:  req.TenantID,
            Content:   item.Content,
            Type:      item.Type,
            URL:       item.URL,
            Metadata:  item.Metadata,
            NodeSets:  req.NodeSets,  // [ADDED] attach node_sets to every entry
        }
        entries = append(entries, entry)
    }

    // Persist entries (include node_sets in DB column)
    if err := uc.repo.SaveEntries(ctx, entries); err != nil { return nil, err }

    // Emit NATS event with node_sets
    uc.publisher.Publish(ctx, "cognee.ingestion.data.ingested", DataIngestedEvent{
        DatasetID: req.DatasetID.String(),
        EntryIDs:  extractIDs(entries),
        TenantID:  req.TenantID,
        NodeSets:  req.NodeSets,    // [ADDED] propagate to cognify
    })

    return &AddDataResult{Count: len(entries), DatasetID: req.DatasetID}, nil
}
```

### 2.3. [MODIFY] Proto — `api/proto/cognee/ingestion/v1/ingestion.proto`

```protobuf
message AddDataRequest {
  string            dataset_id = 1;
  string            tenant_id  = 2;
  repeated DataItem items      = 3;
  repeated string   node_sets  = 4;  // [NEW]
}
```

### 2.4. [MODIFY] `services/cognee-cognify` — Neo4j label attachment

Khi cognify nhận event `cognee.ingestion.data.ingested` chứa `node_sets`:

```go
// services/cognee-cognify/internal/adapter/event/subscriber.go

type DataIngestedEvent struct {
    DatasetID string   `json:"dataset_id"`
    EntryIDs  []string `json:"entry_ids"`
    TenantID  string   `json:"tenant_id"`
    NodeSets  []string `json:"node_sets"`  // [NEW]
}

func (s *Subscriber) handleDataIngested(msg *nats.Msg) {
    var evt DataIngestedEvent
    json.Unmarshal(msg.Data, &evt)

    // Queue cognify job, passing NodeSets
    s.cognifyJobQueue.Enqueue(CognifyJob{
        DatasetID: evt.DatasetID,
        TenantID:  evt.TenantID,
        EntryIDs:  evt.EntryIDs,
        NodeSets:  evt.NodeSets,  // [NEW]
    })
}
```

Trong `start_cognify.go` — Step 3 (ExtractGraph) và Step 5 (AddDatapoints):

```go
// Step 3: ExtractGraph → when building GraphNodes, attach NodeSet labels
func (h *ExtractGraphStep) Execute(ctx context.Context, state *PipelineState) (*PipelineState, error) {
    nodes, edges := h.llmClient.ExtractGraph(ctx, state.Chunks)

    // [ADDED] Attach NodeSet labels to all extracted nodes
    for i := range nodes {
        nodes[i].Labels = append(nodes[i].Labels, state.NodeSets...)
    }

    state.Nodes = nodes
    state.Edges = edges
    return state, nil
}

// Step 5: AddDatapoints → persist with Neo4j multi-labels + Qdrant payload
func (h *AddDatapointsStep) Execute(ctx context.Context, state *PipelineState) (*PipelineState, error) {
    for _, node := range state.Nodes {
        // Neo4j: set additional labels for each NodeSet tag
        // Result: (:Concept:customer_123:preferences) — multi-label node
        cypher := `MERGE (n {id: $id}) SET n += $props`
        for _, tag := range node.Labels {
            cypher += fmt.Sprintf(" SET n:%s", sanitizeLabel(tag))
        }
        h.graphRepo.RunCypher(ctx, cypher, node.Properties)

        // Qdrant: attach node_sets to point payload for filtering
        h.vectorRepo.UpsertPointPayload(ctx, node.VectorID, map[string]any{
            "node_sets": node.Labels,  // [ADDED]
        })
    }
    return state, nil
}

// sanitizeLabel ensures label is valid Cypher identifier
func sanitizeLabel(tag string) string {
    // Replace non-alphanumeric chars with underscore
    return regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(tag, "_")
}
```

### 2.5. [MODIFY] `services/cognee-search` — NodeSet Filtering

#### Search Use Case

```go
// services/cognee-search/internal/usecase/search.go

type SearchRequest struct {
    Query           string
    Strategies      []SearchStrategy
    DatasetID       *uuid.UUID
    TenantID        string
    NodeSets        []string   // [NEW]
    TopK            int
    SaveInteraction bool
}
```

#### Qdrant Retriever (SIMILARITY strategy)

```go
// services/cognee-search/internal/adapter/retriever/vector.go

func (r *VectorRetriever) Retrieve(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
    vec, _ := r.embedder.Embed(ctx, req.Query)

    filter := &qdrant.Filter{}
    // Existing: filter by dataset
    if req.DatasetID != nil {
        filter.Must = append(filter.Must, qdrant.NewMatch("dataset_id", req.DatasetID.String()))
    }
    // [NEW] NodeSet filter
    if len(req.NodeSets) > 0 {
        filter.Must = append(filter.Must, qdrant.NewMatchAny("node_sets", req.NodeSets))
    }

    points, _ := r.qdrantClient.Search(ctx, qdrant.SearchRequest{
        CollectionName: fmt.Sprintf("cognee_%s", req.TenantID),
        Vector:         vec,
        Filter:         filter,
        Limit:          uint64(req.TopK),
    })
    return mapPoints(points), nil
}
```

#### Neo4j Graph Retriever (GRAPH_COMPLETION strategy)

```go
// services/cognee-search/internal/adapter/retriever/graph.go

func (r *GraphRetriever) Retrieve(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
    var cypher string
    params := map[string]any{
        "query":      req.Query,
        "dataset_id": req.DatasetID.String(),
        "top_k":      req.TopK,
    }

    if len(req.NodeSets) > 0 {
        // [MODIFIED] Filter by ALL NodeSet labels using Cypher label predicates
        params["node_sets"] = req.NodeSets
        cypher = `
            MATCH (n)-[r]->(m)
            WHERE n.dataset_id = $dataset_id
              AND all(tag IN $node_sets WHERE tag IN labels(n))
            WITH n, r, m
            CALL db.index.fulltext.queryNodes('nodeTextIndex', $query) YIELD node, score
            WHERE node.id = n.id
            RETURN n, r, m, score
            ORDER BY score DESC
            LIMIT $top_k
        `
    } else {
        cypher = `
            MATCH (n)-[r]->(m)
            WHERE n.dataset_id = $dataset_id
            CALL db.index.fulltext.queryNodes('nodeTextIndex', $query) YIELD node, score
            WHERE node.id = n.id
            RETURN n, r, m, score
            ORDER BY score DESC
            LIMIT $top_k
        `
    }
    return r.neo4jClient.QueryNodes(ctx, cypher, params)
}
```

#### Keyword Retriever (KEYWORD strategy)

```go
// services/cognee-search/internal/adapter/retriever/keyword.go

func (r *KeywordRetriever) Retrieve(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
    // Keyword search via Neo4j full-text index with NodeSet filter
    params := map[string]any{"query": req.Query}
    cypher := `CALL db.index.fulltext.queryNodes('nodeTextIndex', $query) YIELD node, score`

    if len(req.NodeSets) > 0 {
        params["node_sets"] = req.NodeSets
        cypher += ` WHERE all(tag IN $node_sets WHERE tag IN labels(node))`
    }

    cypher += ` RETURN node, score ORDER BY score DESC LIMIT $top_k`
    params["top_k"] = req.TopK
    return r.neo4jClient.QueryNodes(ctx, cypher, params)
}
```

### 2.6. [MODIFY] Proto — `api/proto/cognee/search/v1/search.proto`

```protobuf
message SearchRequest {
  string            query            = 1;
  repeated Strategy strategies       = 2;
  string            dataset_id       = 3;
  string            tenant_id        = 4;
  repeated string   node_sets        = 5;   // [NEW]
  int32             top_k            = 6;
  bool              save_interaction = 7;
}
```

### 2.7. [MODIFY] Gateway Routes — Request Mapper

```go
// gateway/internal/adapter/handler/cognee_handler.go

// Map REST POST /api/v1/cognee/add → gRPC AddData
func (h *CogneeHandler) AddData(w http.ResponseWriter, r *http.Request) {
    var body struct {
        DatasetName string     `json:"dataset_name"`
        Items       []DataItem `json:"items"`
        NodeSets    []string   `json:"node_sets"` // [NEW]
    }
    json.NewDecoder(r.Body).Decode(&body)
    // ... forward to cognee-ingestion with node_sets
}

// Map REST POST /api/v1/cognee/search → gRPC Search
func (h *CogneeHandler) Search(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Query      string   `json:"query"`
        Strategies []string `json:"strategies"`
        NodeSets   []string `json:"node_sets"` // [NEW]
        TopK       int      `json:"top_k"`
    }
    json.NewDecoder(r.Body).Decode(&body)
    // ... forward to cognee-search with node_sets filter
}
```

### 2.8. [MODIFY] Neo4j Full-Text Index

NodeSets dùng Neo4j **label predicates** — không cần thay đổi index schema. Chỉ cần ensure labels được tạo khi upsert nodes.

```cypher
-- Verify label creation (dev/test)
MATCH (n)
WHERE "customer_123" IN labels(n)
RETURN count(n) as node_count
```

---

## 3. Files

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `services/cognee-ingestion/internal/domain/entity.go` | + `NodeSets []string` |
| `services/cognee-ingestion/internal/usecase/add_data.go` | + propagate NodeSets |
| `services/cognee-ingestion/internal/adapter/event/publisher.go` | + NodeSets in event |
| `services/cognee-cognify/internal/adapter/event/subscriber.go` | + read NodeSets from event |
| `services/cognee-cognify/internal/domain/entity.go` | + Labels on GraphNode |
| `services/cognee-cognify/internal/usecase/start_cognify.go` | + attach Labels to nodes (steps 3,5) |
| `services/cognee-cognify/internal/adapter/repository/neo4j/graph_repo.go` | + multi-label upsert |
| `services/cognee-search/internal/usecase/search.go` | + NodeSets in SearchRequest |
| `services/cognee-search/internal/adapter/retriever/vector.go` | + Qdrant payload filter |
| `services/cognee-search/internal/adapter/retriever/graph.go` | + Cypher label filter |
| `services/cognee-search/internal/adapter/retriever/keyword.go` | + Cypher label filter |
| `api/proto/cognee/ingestion/v1/ingestion.proto` | + `repeated string node_sets` |
| `api/proto/cognee/search/v1/search.proto` | + `repeated string node_sets` |
| `gateway/internal/adapter/handler/cognee_handler.go` | + parse node_sets from JSON body |

---

## 4. Acceptance Criteria Mapping

| AC từ CR-COGNEE-002 | Covered by |
|--------------------|-----------|
| `POST /api/v1/cognee/add` nhận node_sets và lưu vào DataEntry | add_data.go |
| Sau cognify, Neo4j node có thêm labels | ExtractGraphStep + AddDatapointsStep |
| Qdrant points có `payload.node_sets` | UpsertPointPayload() |
| Search với `node_sets: ["customer_123"]` → filtered results | VectorRetriever + GraphRetriever |
| Performance filter cao hơn full scan | Qdrant native payload filter (indexed) |
| NodeSet filter hoạt động với SIMILARITY, GRAPH_COMPLETION, KEYWORD, CHUNKS | 4 retriever files |
