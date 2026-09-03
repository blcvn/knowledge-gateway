# cognee-cognify — Data Models

> **Service**: `services/cognee-cognify`
> **Role**: Cognee knowledge graph construction engine — pipeline of chunking, entity extraction, relationship mapping, embedding, and community detection.

---

## GraphNode & GraphEdge

```go
type GraphNode struct {
    ID         string
    Name       string
    Type       string
    Labels     []string       // includes node type + NodeSet tags
    Properties map[string]any
    Derived    bool
    VectorID   string
}

type GraphEdge struct {
    ID         string
    SourceID   string
    TargetID   string
    Label      string
    Weight     float64
    Properties map[string]any
    // Memify aliases
    Subject    string
    Predicate  string
    Object     string
    Derived    bool
}
```

---

## Entity & Relationship

```go
type Entity struct {
    ID          string
    Name        string
    Type        string
    Description string
    SourceChunk uuid.UUID   // chunk UUID this entity was extracted from
    Properties  map[string]any
}

type Relationship struct {
    ID         string
    SourceID   string
    TargetID   string
    Label      string
    Weight     float64
    Properties map[string]any
}

type Community struct {
    ID      string
    Summary string
    Members []string // entity IDs
}
```

---

## Chunk

```go
type Chunk struct {
    ID        uuid.UUID
    Index     int
    Content   string
    CharCount int
    Source    string
    DatasetID string
    TenantID  string
    Metadata  map[string]any
}
```

---

## CognifyJob

```go
type CognifyJob struct {
    ID         uuid.UUID
    TenantID   string
    DatasetID  uuid.UUID
    Status     JobStatus       // pending | running | completed | failed
    Stage      StageType
    Progress   float64
    EntryIDs   []string
    NodeSets   []string
    Config     CognifyConfig
    Metrics    PipelineMetrics
    Error      string
    CreatedAt  time.Time
    UpdatedAt  time.Time
    StartedAt  *time.Time
    FinishedAt *time.Time
}

type CognifyConfig struct {
    Template      string
    Steps         []string
    ChunkSize     int
    ChunkOverlap  int
    CustomPrompt  string
    SkipDedup     bool
    SkipSummarize bool
    OntologyID    string
}
```

---

## PipelineMetrics

```go
type PipelineMetrics struct {
    TotalDurationMs        int64 `json:"total_duration_ms"`
    ChunksCreated          int   `json:"chunks_created"`
    EntitiesFound          int   `json:"entities_found"`
    RelationsFound         int   `json:"relations_found"`
    EmbeddingsCreated      int   `json:"embeddings_created"`
    LLMCallsTotal          int   `json:"llm_calls_total"`
    EmbeddingsGenerated    int   `json:"embeddings_generated"`
    EntitiesDeduplicated   int   `json:"entities_deduplicated"`
    EntitiesExtracted      int   `json:"entities_extracted"`
    RelationshipsExtracted int   `json:"relationships_extracted"`
    CommunitiesFound       int   `json:"communities_found"`
}
```

---

## GraphDiff (Memify)

```go
// Non-destructive graph enrichment — additions only, no deletes.
type GraphDiff struct {
    Nodes []GraphNode
    Edges []GraphEdge
}

type GraphFact struct {
    Subject   string
    Predicate string
    Object    string
}

type PipelineRun struct {
    ID        string
    DatasetID string
    TenantID  string
    Type      string     // "cognify" | "memify"
    Status    string     // QUEUED | RUNNING | COMPLETED | FAILED
    NewNodes  int
    NewEdges  int
    Error     string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

---

## Ontology

```go
type Ontology struct {
    ID         uuid.UUID          `json:"id"`
    TenantID   string             `json:"tenant_id"`
    Name       string             `json:"name"`
    Categories []OntologyCategory `json:"categories"`
}

type OntologyCategory struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Parents     []string `json:"parents"`
}
```

---

## PipelineConfig (Pipeline Templates)

```go
type PipelineConfig struct {
    Template PipelineTemplateName
    Steps    []PipelineStep
    Options  PipelineOptions
}

type PipelineOptions struct {
    ChunkSize    int
    CustomPrompt string
    TemporalMode bool
    SkipCache    bool
}

// Templates: STANDARD | EMBED_ONLY | FAST_INDEX | TEMPORAL | GRAPH_ONLY
```

---

## Sources
- [`services/cognee-cognify/internal/domain/entity.go`](../../services/cognee-cognify/internal/domain/entity.go)
- [`services/cognee-cognify/internal/domain/chunk.go`](../../services/cognee-cognify/internal/domain/chunk.go)
- [`services/cognee-cognify/internal/domain/job.go`](../../services/cognee-cognify/internal/domain/job.go)
- [`services/cognee-cognify/internal/domain/graph_diff.go`](../../services/cognee-cognify/internal/domain/graph_diff.go)
- [`services/cognee-cognify/internal/domain/ontology.go`](../../services/cognee-cognify/internal/domain/ontology.go)
