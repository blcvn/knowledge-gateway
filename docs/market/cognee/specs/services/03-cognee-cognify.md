# 03 — Cognee Cognify Service

> **gRPC**: 9012 | **Health**: 9092

---

## 1. Purpose

KG construction pipeline: nhận data từ Ingestion, classify, chunk, extract entities/edges via LLM, build knowledge graph trên Neo4j, và index embeddings trên Qdrant.

---

## 2. Clean Architecture

```
services/cognee-cognify/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # Chunk, GraphNode, GraphEdge, Ontology, Community
│   │   ├── value_object.go     # ChunkStrategy, OntologyType, EdgeWeight
│   │   ├── event.go            # PipelineCompletedEvent, GraphBuiltEvent
│   │   └── errors.go           # ErrLLMTimeout, ErrOntologyInvalid
│   ├── usecase/
│   │   ├── start_cognify.go    # Orchestrate full pipeline
│   │   ├── classify_content.go # LLM-based content classification
│   │   ├── chunk_content.go    # Smart chunking (recursive, semantic, paragraph)
│   │   ├── extract_graph.go    # LLM entity/edge extraction
│   │   ├── build_ontology.go   # Ontology construction/merge
│   │   ├── add_datapoints.go   # Embed + index to VectorDB
│   │   ├── detect_community.go # Graph community detection
│   │   ├── summarize_community.go
│   │   ├── port/
│   │   │   ├── input.go        # StartCognifyUseCase interface
│   │   │   └── output.go       # GraphRepository, VectorRepository, LLMClient, EmbedderClient
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go      # cognee.cognify.v1.CognifyService impl
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   ├── neo4j/          # KG node/edge CRUD (using pkg/adapters/graphdb)
│   │   │   └── qdrant/         # Vector indexing (using pkg/adapters/vectordb)
│   │   ├── client/
│   │   │   ├── llm_client.go   # Uses pkg/adapters/llm — structured extraction
│   │   │   └── embedder.go     # Uses pkg/adapters/embedder
│   │   └── event/
│   │       ├── subscriber.go   # Subscribe cognee.ingestion.data.ingested
│   │       └── publisher.go    # Publish cognee.cognify.pipeline.completed
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

---

## 3. Pipeline Flow

```
cognee.ingestion.data.ingested
         │
         ▼
┌─── StartCognify Pipeline ────────────────────────┐
│                                                   │
│  Step 1: ClassifyContent (LLM)                   │
│     → Determine doc type, language, topics         │
│                                                   │
│  Step 2: ChunkContent                             │
│     → RecursiveChunker / SemanticChunker          │
│     → Chunks with overlap + metadata              │
│                                                   │
│  Step 3: ExtractGraph (LLM — structured output)  │
│     → For each chunk:                              │
│       - Extract entities (name, type, description) │
│       - Extract edges (source, target, relation)  │
│     → Deduplicate + merge with existing graph     │
│                                                   │
│  Step 4: BuildOntology                            │
│     → Generate/update ontology from extracted      │
│       entity types and relation types              │
│                                                   │
│  Step 5: AddDatapoints                            │
│     → Embed chunks + entities → Qdrant            │
│     → Index with tenant_id + dataset_id           │
│                                                   │
│  Step 6: DetectCommunity (optional)               │
│     → Louvain/Leiden community detection          │
│     → SummarizeCommunity via LLM                  │
│                                                   │
│  Step 7: Emit PipelineCompleted                   │
└───────────────────────────────────────────────────┘
         │
         ▼
cognee.cognify.pipeline.completed
         │
         ▼
cognee-search (reindex subscriber)
```

---

## 4. Domain Entities

```go
type Chunk struct {
    ID          uuid.UUID
    DataEntryID uuid.UUID
    Content     string
    Index       int
    TokenCount  int
    Embedding   []float32
    Metadata    map[string]string
}

type GraphNode struct {
    ID          uuid.UUID
    TenantID    string
    Name        string
    Type        string      // Person, Organization, Concept, Event
    Description string
    Properties  map[string]any
    Embedding   []float32
}

type GraphEdge struct {
    ID         uuid.UUID
    SourceID   uuid.UUID
    TargetID   uuid.UUID
    Relation   string      // works_at, is_part_of, related_to
    Weight     float64
    Properties map[string]any
}

type Community struct {
    ID       uuid.UUID
    NodeIDs  []uuid.UUID
    Summary  string
    Level    int
}
```

---

## 5. LLM Structured Output

```go
// ExtractGraph prompt produces structured JSON
type ExtractionResult struct {
    Entities []ExtractedEntity `json:"entities"`
    Edges    []ExtractedEdge   `json:"edges"`
}

type ExtractedEntity struct {
    Name        string `json:"name"`
    Type        string `json:"type"`
    Description string `json:"description"`
}

type ExtractedEdge struct {
    Source   string `json:"source"`
    Target  string `json:"target"`
    Relation string `json:"relation"`
    Weight   float64 `json:"weight"`
}
```

---

## 6. NATS Events

| Subject | Direction | Payload |
|---------|-----------|---------|
| `cognee.ingestion.data.ingested` | Subscribe | `{dataset_id, entry_ids[], tenant_id}` |
| `cognee.cognify.pipeline.completed` | Publish | `{dataset_id, node_count, edge_count, tenant_id}` |
| `cognee.memory.session.persisted` | Subscribe | `{session_id, messages[], tenant_id}` |
