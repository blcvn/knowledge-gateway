# kg-service — Data Models

> **Service**: `services/kg-service`
> **Role**: Knowledge Graph facade — unified HTTP adapter for Cognee (Python) and Graphiti engines.
> Absorbed from: `cognee-ingestion`, `cognee-cognify`, `cognee-search`, `cognee-pipeline`, `graphiti-ingestion`, `graphiti-knowledge`, `graphiti-store`, `graphiti-search`, `graphiti-pipeline`.

---

## cognee — Cognee Engine Domain

### Dataset

```go
type Dataset struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    TenantID  string    `json:"tenant_id"`
    Status    string    `json:"status"` // "empty" | "uploading" | "ready" | "cognifying" | "indexed"
    DataCount int       `json:"data_count"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### DataItem (Ingestion Input)

```go
type DataItem struct {
    DatasetID   string         `json:"dataset_id"`
    ContentType ContentType    `json:"content_type"`
    Content     []byte         `json:"content,omitempty"`
    URI         string         `json:"uri,omitempty"`
    Metadata    map[string]any `json:"metadata,omitempty"`
    NodeSets    []string       `json:"node_sets,omitempty"` // tag-based memory partitioning
    Config      map[string]any `json:"config,omitempty"`    // e.g. {"pdf_mode": "LAYOUT_AWARE"}
}

type ContentType string
// text | pdf | url | json | PDF_LAYOUT | TABULAR_FK
```

### CognifyJob

```go
type CognifyJob struct {
    JobID     string     `json:"job_id"`
    DatasetID string     `json:"dataset_id"`
    Status    string     `json:"status"` // "pending" | "running" | "completed" | "failed"
    Progress  float64    `json:"progress"`
    StartedAt time.Time  `json:"started_at"`
    DoneAt    *time.Time `json:"done_at,omitempty"`
}
```

### MemifyJob (Non-Destructive Graph Enrichment)

```go
type MemifyJob struct {
    PipelineRunID string     `json:"pipeline_run_id"`
    DatasetID     string     `json:"dataset_id"`
    TenantID      string     `json:"tenant_id"`
    Status        string     `json:"status"` // "QUEUED" | "RUNNING" | "COMPLETED" | "FAILED"
    DeriveFacts   bool       `json:"derive_facts"`
    EmbedTriplets bool       `json:"embed_triplets"`
    BatchSize     int        `json:"batch_size"`
    NewNodes      int        `json:"new_nodes,omitempty"`
    NewEdges      int        `json:"new_edges,omitempty"`
    StartedAt     time.Time  `json:"started_at"`
    DoneAt        *time.Time `json:"done_at,omitempty"`
}

type MemifyConfig struct {
    DeriveFacts   bool `json:"derive_facts"`    // default true
    EmbedTriplets bool `json:"embed_triplets"`  // default true
    BatchSize     int  `json:"batch_size"`      // default 50
}
```

### PipelineConfig

```go
type PipelineConfig struct {
    Template     PipelineTemplate `json:"template,omitempty"`
    Steps        []string         `json:"steps,omitempty"`
    ChunkSize    int              `json:"chunk_size,omitempty"`
    CustomPrompt string           `json:"custom_prompt,omitempty"`
    TemporalMode bool             `json:"temporal_mode,omitempty"`
}

type PipelineTemplate string
// STANDARD | EMBED_ONLY | FAST_INDEX | TEMPORAL | GRAPH_ONLY
```

### SearchRequest & Result

```go
type SearchRequest struct {
    Query           string   `json:"query"`
    SearchType      string   `json:"search_type"` // "semantic" | "graph" | "hybrid" | "GRAPH_COMPLETION" | "FEEDBACK"
    DatasetID       string   `json:"dataset_id,omitempty"`
    NodeSets        []string `json:"node_sets,omitempty"`
    TopK            int      `json:"top_k,omitempty"`
    SaveInteraction bool     `json:"save_interaction,omitempty"`
    FeedbackFor     *string  `json:"feedback_for,omitempty"`
    FeedbackScore   *float64 `json:"feedback_score,omitempty"` // -1.0 to 1.0
}

type SearchResult struct {
    Content  string         `json:"content"`
    Score    float64        `json:"score"`
    Source   string         `json:"source"`
    Metadata map[string]any `json:"metadata,omitempty"`
}
```

### DataPoint (Schema-Defined Knowledge Unit)

```go
type DataPoint struct {
    ID          string              `json:"id"`
    Type        string              `json:"type"`         // "Paper" | "User" | "Product" | ...
    Fields      map[string]any      `json:"fields"`
    IndexFields []string            `json:"index_fields"`
    Relations   []DataPointRelation `json:"relations,omitempty"`
}

type DataPointRelation struct {
    TargetID string  `json:"target_id"`
    Label    string  `json:"label"`   // "authored_by" | "belongs_to" | ...
    Weight   float64 `json:"weight"`
}

type AddDataPointsRequest struct {
    DatasetID  string      `json:"dataset_id"`
    TenantID   string      `json:"tenant_id"`
    DataPoints []DataPoint `json:"data_points"`
    NodeSets   []string    `json:"node_sets,omitempty"`
}

type AddDataPointsResponse struct {
    Accepted int      `json:"accepted"`
    IDs      []string `json:"ids"`
}
```

---

## graphiti — Graphiti Knowledge Graph Domain

### Episode

```go
type Episode struct {
    UUID      string    `json:"uuid"`
    Name      string    `json:"name"`
    Content   string    `json:"content"`
    Source    string    `json:"source"`    // "message" | "document" | "json"
    SourceID  string    `json:"source_id"`
    TenantID  string    `json:"tenant_id"`
    Embedding []float32 `json:"embedding,omitempty"`
    CreatedAt time.Time `json:"created_at"`
}
```

### Node & Edge

```go
type Node struct {
    UUID       string         `json:"uuid"`
    Name       string         `json:"name"`
    Type       string         `json:"type"`
    Summary    string         `json:"summary"`
    Attributes map[string]any `json:"attributes,omitempty"`
    TenantID   string         `json:"tenant_id"`
    Episodes   []string       `json:"episodes,omitempty"`
    CreatedAt  time.Time      `json:"created_at"`
    UpdatedAt  time.Time      `json:"updated_at"`
}

type Edge struct {
    UUID       string    `json:"uuid"`
    SourceUUID string    `json:"source_uuid"`
    TargetUUID string    `json:"target_uuid"`
    Relation   string    `json:"relation"` // e.g. KNOWS, WORKS_AT
    Weight     float64   `json:"weight"`
    Facts      []Fact    `json:"facts,omitempty"`
    TenantID   string    `json:"tenant_id"`
    CreatedAt  time.Time `json:"created_at"`
}

type Fact struct {
    Content   string     `json:"content"`
    Valid     bool       `json:"valid"`
    ValidAt   *time.Time `json:"valid_at,omitempty"`
    InvalidAt *time.Time `json:"invalid_at,omitempty"`
}
```

### Ontology

```go
type Ontology struct {
    TenantID    string         `json:"tenant_id"`
    EntityTypes []string       `json:"entity_types"`
    Relations   []RelationType `json:"relations"`
    UpdatedAt   time.Time      `json:"updated_at"`
}

type RelationType struct {
    Name        string `json:"name"`
    SourceType  string `json:"source_type"`
    TargetType  string `json:"target_type"`
    Description string `json:"description,omitempty"`
}
```

### SearchQuery & Result

```go
type SearchQuery struct {
    Query    string         `json:"query"`
    TenantID string         `json:"tenant_id"`
    Mode     string         `json:"mode"`  // "semantic" | "graph" | "hybrid"
    Limit    int            `json:"limit"`
    Filter   map[string]any `json:"filter,omitempty"`
}

type SearchResult struct {
    Episodes []*Episode `json:"episodes,omitempty"`
    Nodes    []*Node    `json:"nodes,omitempty"`
    Edges    []*Edge    `json:"edges,omitempty"`
    Score    float64    `json:"score"`
}
```

---

## Sources
- [`services/kg-service/internal/domain/cognee/entity.go`](../../services/kg-service/internal/domain/cognee/entity.go)
- [`services/kg-service/internal/domain/graphiti/entity.go`](../../services/kg-service/internal/domain/graphiti/entity.go)
