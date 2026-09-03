---
id: MERGE-P2-T1
title: "kg-service: Tạo mới — Merge tất cả graphiti-* services (5 services)"
phase: P2
service: kg-service (NEW)
priority: P1
status: Done
estimated: 12h
created: 2026-06-11
linked_sol: SOL-003
depends_on: [MERGE-P1-T1]
---

## Mục Tiêu

Tạo `kg-service` (Knowledge Graph Service) mới bằng cách merge toàn bộ 5 Graphiti services. Graphiti là Go-native knowledge graph engine — có thể implement đầy đủ trong 1 binary.

## Services Bị Absorb

| Service | Lines | Chức Năng |
|---------|-------|-----------|
| `graphiti-ingestion` | 1,785 | Episode ingest + enrichment |
| `graphiti-knowledge` | 2,296 | Knowledge extraction + ontology |
| `graphiti-pipeline` | 1,706 | Batch processing pipeline |
| `graphiti-search` | 3,005 | Graph search queries |
| `graphiti-store` | 3,934 | Node/Edge CRUD storage |

**Tổng: 12,726 lines** → 1 service (giai đoạn này)

## Cấu Trúc Service Mới

```
services/kg-service/
├── Dockerfile
├── go.mod
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   └── graphiti/
│   │       ├── entity.go        # Episode, Node, Edge, Ontology, Fact
│   │       ├── errors.go        # ErrNodeNotFound, ErrInvalidEpisode
│   │       └── search.go        # SearchQuery, SearchResult, SearchConfig
│   ├── usecase/
│   │   └── graphiti/
│   │       ├── ingest.go        # IngestEpisode usecase
│   │       ├── store.go         # GetNode, GetEdge, CreateNode, CreateEdge
│   │       ├── search.go        # Search usecase (BFS + semantic)
│   │       ├── knowledge.go     # ExtractKnowledge, UpdateOntology
│   │       └── port/
│   │           └── interfaces.go # EpisodeRepository, GraphRepository, SearchIndex
│   ├── adapter/
│   │   ├── grpc/
│   │   │   └── router.go        # ForwardService route registration
│   │   └── handler/
│   │       ├── episode.go       # IngestEpisode handler
│   │       ├── store.go         # GetNode, GetEdge handlers
│   │       ├── search.go        # Search handler
│   │       └── knowledge.go     # Ontology handlers
│   └── infra/
│       ├── neo4j/               # Neo4j graph database client
│       ├── pgvector/            # Vector embeddings for semantic search
│       ├── nats/                # Async pipeline events
│       └── config/              # Config loader
└── migrations/
    └── 001_kg_init.cypher       # Neo4j schema (constraints + indexes)
```

## Domain Entities

```go
// domain/graphiti/entity.go

type Episode struct {
    UUID      string
    Name      string
    Content   string      // Raw text content
    Source    string      // "message" | "document" | "json"
    SourceID  string      // Optional external ID
    TenantID  string
    CreatedAt time.Time
    Embedding []float32   // Vector embedding
}

type Node struct {
    UUID       string
    Name       string
    Type       string          // Entity type (Person, Place, Concept, etc.)
    Summary    string
    Attributes map[string]any
    TenantID   string
    Episodes   []string        // Related episode UUIDs
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type Edge struct {
    UUID       string
    SourceUUID string
    TargetUUID string
    Relation   string          // Relation label
    Weight     float64
    Facts      []Fact
    TenantID   string
    CreatedAt  time.Time
}

type Fact struct {
    Content  string
    Valid     bool
    ValidAt  *time.Time
    InvalidAt *time.Time
}

type Ontology struct {
    TenantID    string
    EntityTypes []string
    Relations   []RelationType
    UpdatedAt   time.Time
}

type SearchQuery struct {
    Query     string
    TenantID  string
    Mode      string    // "semantic" | "graph" | "hybrid"
    Limit     int
    Filter    map[string]any
}

type SearchResult struct {
    Episodes  []*Episode
    Nodes     []*Node
    Edges     []*Edge
    Score     float64
}
```

## Usecase Implementation

### `usecase/graphiti/ingest.go`

```go
type IngestUseCase struct {
    episodes  port.EpisodeRepository
    nodes     port.GraphRepository
    embedder  port.EmbeddingService
    publisher port.EventPublisher
}

func (uc *IngestUseCase) IngestEpisode(ctx context.Context, req IngestRequest) (*Episode, error) {
    // 1. Create Episode entity
    episode := &Episode{
        UUID:     uuid.New().String(),
        Name:     req.Name,
        Content:  req.Content,
        Source:   req.Source,
        TenantID: req.TenantID,
    }

    // 2. Generate embedding
    embedding, err := uc.embedder.Embed(ctx, req.Content)
    if err != nil { return nil, err }
    episode.Embedding = embedding

    // 3. Extract entities + relations (knowledge extraction)
    entities, edges := extractKnowledge(req.Content)

    // 4. Persist episode
    if err := uc.episodes.Create(ctx, episode); err != nil { return nil, err }

    // 5. Upsert nodes + edges
    for _, node := range entities {
        uc.nodes.UpsertNode(ctx, node)
    }
    for _, edge := range edges {
        uc.nodes.UpsertEdge(ctx, edge)
    }

    // 6. Publish event
    uc.publisher.Publish(ctx, "kg.episode.ingested", episode)

    return episode, nil
}
```

### `usecase/graphiti/search.go`

```go
type SearchUseCase struct {
    episodes  port.EpisodeRepository
    nodes     port.GraphRepository
    searcher  port.SearchIndex
}

func (uc *SearchUseCase) Search(ctx context.Context, query SearchQuery) (*SearchResult, error) {
    switch query.Mode {
    case "semantic":
        // Vector similarity search on episode embeddings
        return uc.searcher.SemanticSearch(ctx, query)
    case "graph":
        // BFS/DFS traversal from relevant nodes
        return uc.searcher.GraphSearch(ctx, query)
    default:
        // Hybrid: semantic + graph fusion
        semantic, _ := uc.searcher.SemanticSearch(ctx, query)
        graph, _ := uc.searcher.GraphSearch(ctx, query)
        return mergeResults(semantic, graph), nil
    }
}
```

## ForwardService Routes

```go
// adapter/grpc/router.go
func RegisterRoutes(router *forward.Router, ingest IngestHandler, store StoreHandler, search SearchHandler, knowledge KnowledgeHandler) {
    // Episode ingestion
    router.Handle("POST", "/v1/graphiti/episodes",          ingest.IngestEpisode)

    // Graph store CRUD
    router.Handle("GET",  "/v1/graphiti/nodes/*",           store.GetNode)
    router.Handle("GET",  "/v1/graphiti/edges/*",           store.GetEdge)

    // Search
    router.Handle("POST", "/v1/graphiti/search",            search.Search)

    // Knowledge/Ontology (console routes)
    router.Handle("POST", "/v1/console/graph/subgraph",     knowledge.Subgraph)
    router.Handle("GET",  "/v1/console/graph/entity/*",     knowledge.GetEntity)
    router.Handle("POST", "/v1/console/graph/timeline",     knowledge.Timeline)
    router.Handle("GET",  "/v1/console/graph/ontology",     knowledge.GetOntology)
    router.Handle("PUT",  "/v1/console/graph/ontology",     knowledge.UpdateOntology)
    router.Handle("POST", "/v1/console/graph/query",        knowledge.Query)

    // Memory routes (adaptive connectors referencing KG data)
    router.Handle("GET",  "/v1/console/adaptive/memories",      knowledge.ListMemories)
    router.Handle("GET",  "/v1/console/adaptive/memories/*/versions", knowledge.GetVersions)
}
```

## Infrastructure — Neo4j

```go
// infra/neo4j/graph_repo.go
type Neo4jGraphRepo struct {
    driver neo4j.DriverWithContext
}

func (r *Neo4jGraphRepo) UpsertNode(ctx context.Context, node *Node) error {
    session := r.driver.NewSession(ctx, neo4j.SessionConfig{})
    _, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
        _, err := tx.Run(ctx, `
            MERGE (n:Entity {uuid: $uuid, tenant_id: $tenantID})
            ON CREATE SET n.name = $name, n.type = $type, n.summary = $summary, n.created_at = datetime()
            ON MATCH SET n.summary = $summary, n.updated_at = datetime()
            RETURN n
        `, map[string]any{"uuid": node.UUID, "tenantID": node.TenantID, "name": node.Name, "type": node.Type, "summary": node.Summary})
        return nil, err
    })
    return err
}
```

## Neo4j Schema Init

```cypher
// migrations/001_kg_init.cypher

// Constraints
CREATE CONSTRAINT entity_uuid IF NOT EXISTS FOR (n:Entity) REQUIRE n.uuid IS UNIQUE;
CREATE CONSTRAINT episode_uuid IF NOT EXISTS FOR (e:Episode) REQUIRE e.uuid IS UNIQUE;

// Indexes
CREATE INDEX entity_tenant IF NOT EXISTS FOR (n:Entity) ON (n.tenant_id);
CREATE INDEX entity_name IF NOT EXISTS FOR (n:Entity) ON (n.name);
CREATE INDEX episode_tenant IF NOT EXISTS FOR (e:Episode) ON (e.tenant_id);
CREATE INDEX episode_source IF NOT EXISTS FOR (e:Episode) ON (e.source_id);

// Vector index (Neo4j 5.x+)
CREATE VECTOR INDEX episode_embedding IF NOT EXISTS
FOR (e:Episode) ON (e.embedding)
OPTIONS {indexConfig: {`vector.dimensions`: 1536, `vector.similarity_function`: 'cosine'}};
```

## PostgreSQL — Episode Embedding Backup

```sql
-- migrations/001_kg_episodes.sql (pgvector fallback)
CREATE TABLE IF NOT EXISTS kg_episodes (
    uuid       UUID PRIMARY KEY,
    tenant_id  TEXT NOT NULL,
    name       TEXT NOT NULL,
    content    TEXT NOT NULL,
    source     TEXT NOT NULL,
    source_id  TEXT,
    embedding  vector(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_kg_episodes_tenant ON kg_episodes(tenant_id);
CREATE INDEX idx_kg_episodes_embedding ON kg_episodes USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
```

## Config Environment Variables

```bash
GRPC_PORT=9090
HEALTH_PORT=9120
DATABASE_URL=postgres://...           # pgvector fallback
NEO4J_URL=bolt://neo4j:7687          # Required for graph
NEO4J_USER=neo4j
NEO4J_PASSWORD=...
NATS_URL=nats://nats:4222
EMBEDDING_URL=http://llm-proxy:8080   # Embedding service endpoint
EMBEDDING_MODEL=text-embedding-3-small
KG_TENANT_ISOLATION=true             # Enable tenant namespace in Neo4j
```

## go.mod

```
module vnp-memory/services/kg-service

go 1.25.0

require (
    vnp-memory/pkg/forward      v0.0.0
    vnp-memory/pkg/telemetry    v0.0.0
    vnp-memory/pkg/tenant       v0.0.0
    vnp-memory/pkg/vectorstore  v0.0.0
    google.golang.org/grpc      v1.72.1
    github.com/jackc/pgx/v5     v5.7.0
    github.com/pgvector/pgvector-go v0.2.0
    github.com/neo4j/neo4j-go-driver/v5 v5.x.x
)
```

## Acceptance Criteria

- [ ] `POST /v1/graphiti/episodes` với JSON body → ingest episode, persist vào Neo4j + pgvector
- [ ] `GET /v1/graphiti/nodes/{uuid}` returns node JSON
- [ ] `GET /v1/graphiti/edges/{uuid}` returns edge JSON
- [ ] `POST /v1/graphiti/search` với query → returns ranked results
- [ ] `GET /v1/console/graph/ontology` returns entity types + relations
- [ ] Episode embedding stored trong pgvector (semantic search hoạt động)
- [ ] Graph traversal qua Neo4j hoạt động (graph search)
- [ ] Tenant isolation: tenant A không thể query data của tenant B
- [ ] `/healthz` returns 200 OK
- [ ] `go build ./services/kg-service/...` passes
- [ ] `go test ./services/kg-service/...` passes

## Ghi Chú

- **Cognee sẽ được thêm ở MERGE-P2-T2** — task này chỉ implement Graphiti
- **Neo4j** là infra dependency mới cần thêm vào docker-compose
- Nếu NEO4J_URL không set → fallback sang pure pgvector (graph features disabled)
- Knowledge extraction (entity + relation detection) dùng LLM call — cần EMBEDDING_URL
- Tất cả 5 graphiti services gốc giữ nguyên cho đến P4 cleanup
