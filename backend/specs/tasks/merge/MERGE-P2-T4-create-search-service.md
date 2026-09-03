---
id: MERGE-P2-T4
title: "search-service: Tạo mới — Expand vnp-search-hub + absorb ov-search + sm-search + sm-connector + sm-mcp"
phase: P2
service: search-service (NEW — expanded from vnp-search-hub)
priority: P1
status: Done
estimated: 10h
created: 2026-06-11
linked_sol: SOL-003
depends_on: [MERGE-P2-T1, MERGE-P2-T3]
---

## Mục Tiêu

Mở rộng `vnp-search-hub` thành `search-service` — single service xử lý tất cả cross-engine search, RAG, connector management, và MCP tool integrations.

## Services Bị Absorb

| Service | Lines | Chức Năng |
|---------|-------|-----------|
| `vnp-search-hub` | 1,120 | Multi-engine orchestration (RRF/MMR) — **partially implemented** |
| `ov-search` | 1,259 | OpenViking semantic file search |
| `sm-search` | 127 | Supermemory search |
| `sm-connector` | 196 | External data connectors (partially real) |
| `sm-mcp` | 142 | MCP tool integrations |

**Tổng: 2,844 lines** → 1 service

## Architecture

```
services/search-service/
├── Dockerfile
├── go.mod
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── search/
│   │   │   ├── entity.go       # SearchQuery, SearchResult, RankStrategy, HybridResult
│   │   │   └── errors.go
│   │   ├── connector/
│   │   │   ├── entity.go       # Connector, SyncJob, ConnectorType
│   │   │   └── errors.go
│   │   └── mcp/
│   │       ├── entity.go       # MCPTool, ToolCall, ToolResponse
│   │       └── errors.go
│   ├── usecase/
│   │   ├── orchestrator/
│   │   │   ├── search.go       # Multi-engine search + reranking (RRF, MMR)
│   │   │   └── rag.go          # RAG pipeline
│   │   ├── connector/
│   │   │   └── service.go      # CreateConnector, SyncConnector, ListConnectors
│   │   └── mcp/
│   │       └── service.go      # ToolRegistry, ExecuteTool
│   ├── adapter/
│   │   ├── grpc/
│   │   │   └── router.go       # ForwardService routes
│   │   ├── kg/
│   │   │   └── client.go       # HTTP client → kg-service
│   │   ├── memory/
│   │   │   └── client.go       # HTTP client → memory-service (or gRPC)
│   │   └── storage/
│   │       └── client.go       # HTTP client → storage-service
│   └── infra/
│       ├── pgvector/           # Direct vector search (fallback)
│       ├── nats/
│       └── config/
└── migrations/
    └── 001_search_init.sql
```

## Domain Entities

```go
// domain/search/entity.go

type SearchQuery struct {
    Query       string
    TenantID    string
    Engines     []string         // ["graphiti", "cognee", "memobase", "storage", "sm"]
    Mode        string           // "semantic" | "keyword" | "hybrid"
    RankStrategy string          // "rrf" | "mmr" | "simple"
    Limit       int
    Offset      int
    Filter      map[string]any
}

type SearchResult struct {
    Items     []*SearchItem
    Total     int
    Took      time.Duration
    Engines   []string          // Which engines were queried
}

type SearchItem struct {
    ID        string
    Content   string
    Score     float64
    Source    string            // "graphiti" | "cognee" | "memobase" | "storage" | "sm"
    Metadata  map[string]any
    Highlight string
}

// Reranking strategies
type RRFConfig struct {
    K int // RRF constant (default: 60)
}

type MMRConfig struct {
    Lambda    float64 // diversity vs relevance tradeoff [0,1]
    Embedding []float32
}
```

```go
// domain/connector/entity.go

type Connector struct {
    ID           string
    TenantID     string
    Name         string
    Type         ConnectorType    // "github" | "notion" | "gdrive" | "slack" | "web"
    Config       map[string]any
    SyncFrequency string          // "hourly" | "daily" | "weekly" | "manual"
    LastSyncAt   *time.Time
    Status       string          // "active" | "paused" | "error"
    CreatedAt    time.Time
}

type SyncJob struct {
    ID          string
    ConnectorID string
    Status      string    // "pending" | "running" | "completed" | "failed"
    ItemsSynced int
    StartedAt   time.Time
    CompletedAt *time.Time
    Error       string
}

type ConnectorType string
const (
    ConnectorGitHub ConnectorType = "github"
    ConnectorNotion ConnectorType = "notion"
    ConnectorGDrive ConnectorType = "gdrive"
    ConnectorSlack  ConnectorType = "slack"
    ConnectorWeb    ConnectorType = "web"
)
```

```go
// domain/mcp/entity.go

type MCPTool struct {
    Name        string
    Description string
    InputSchema map[string]any    // JSON Schema
    Endpoint    string            // Internal service endpoint
}

type ToolCall struct {
    ToolName  string
    Input     map[string]any
    SessionID string
}

type ToolResponse struct {
    ToolName string
    Output   any
    Error    string
    Duration time.Duration
}
```

## Usecase — Search Orchestrator

```go
// usecase/orchestrator/search.go
type SearchOrchestrator struct {
    kgClient      port.KGClient      // → kg-service
    memoryClient  port.MemoryClient  // → memory-service
    storageClient port.StorageClient // → storage-service
    vectorSearch  port.VectorSearch  // Direct pgvector
    reranker      port.Reranker
}

func (o *SearchOrchestrator) Search(ctx context.Context, query *SearchQuery) (*SearchResult, error) {
    // 1. Fan-out to requested engines concurrently
    var wg sync.WaitGroup
    results := make(chan *EngineResult, len(query.Engines))

    for _, engine := range query.Engines {
        wg.Add(1)
        go func(eng string) {
            defer wg.Done()
            switch eng {
            case "graphiti":
                r, _ := o.kgClient.GraphitiSearch(ctx, query)
                results <- &EngineResult{Engine: eng, Items: r}
            case "cognee":
                r, _ := o.kgClient.CogneeSearch(ctx, query)
                results <- &EngineResult{Engine: eng, Items: r}
            case "memobase":
                r, _ := o.memoryClient.MemobaseSearch(ctx, query)
                results <- &EngineResult{Engine: eng, Items: r}
            case "storage":
                r, _ := o.storageClient.FileSearch(ctx, query)
                results <- &EngineResult{Engine: eng, Items: r}
            }
        }(engine)
    }

    wg.Wait()
    close(results)

    // 2. Collect all results
    engineResults := collectResults(results)

    // 3. Rerank using configured strategy
    switch query.RankStrategy {
    case "rrf":
        return o.reranker.RRF(engineResults, 60), nil
    case "mmr":
        return o.reranker.MMR(engineResults, 0.7), nil
    default:
        return mergeByScore(engineResults), nil
    }
}

// usecase/orchestrator/rag.go
func (o *SearchOrchestrator) RAG(ctx context.Context, query string, tenantID string) (*RAGResponse, error) {
    // 1. Search across all engines
    results, _ := o.Search(ctx, &SearchQuery{
        Query: query, TenantID: tenantID,
        Engines: []string{"graphiti", "memobase", "sm"},
        Limit: 10, RankStrategy: "rrf",
    })

    // 2. Build context from top results
    context := buildContext(results.Items[:min(5, len(results.Items))])

    // 3. Return context + source references (LLM call is upstream)
    return &RAGResponse{Context: context, Sources: results.Items}, nil
}
```

## Usecase — Connector

```go
// usecase/connector/service.go
type ConnectorService struct {
    repo      port.ConnectorRepository
    publisher port.EventPublisher
}

func (s *ConnectorService) CreateConnector(ctx context.Context, req CreateConnectorRequest) (*Connector, error) {
    conn := &Connector{
        ID:           uuid.New().String(),
        TenantID:     req.TenantID,
        Type:         ConnectorType(req.Type),
        Name:         req.Name,
        Config:       req.Config,
        SyncFrequency: req.SyncFrequency,
        Status:       "active",
    }
    if err := s.repo.Create(ctx, conn); err != nil { return nil, err }
    s.publisher.Publish(ctx, "search.connector.created", conn)
    return conn, nil
}

func (s *ConnectorService) SyncConnector(ctx context.Context, connectorID string) (*SyncJob, error) {
    conn, err := s.repo.GetByID(ctx, connectorID)
    if err != nil { return nil, err }

    job := &SyncJob{ID: uuid.New().String(), ConnectorID: connectorID, Status: "pending"}
    if err := s.repo.CreateJob(ctx, job); err != nil { return nil, err }

    // Async: trigger sync via NATS
    s.publisher.Publish(ctx, "search.connector.sync.requested", job)
    return job, nil
}
```

## ForwardService Routes

```go
// adapter/grpc/router.go
func RegisterRoutes(router *forward.Router, searchH SearchHandler, connH ConnectorHandler, mcpH MCPHandler) {
    // OV Search
    router.Handle("POST", "/v1/ov/search",                  searchH.OVSearch)

    // SM Search + RAG
    router.Handle("POST", "/v1/sm/search",                  searchH.SMSearch)
    router.Handle("POST", "/v1/sm/rag",                     searchH.RAG)

    // SM Connectors
    router.Handle("GET",  "/v1/console/adaptive/connectors",        connH.ListConnectors)
    router.Handle("POST", "/v1/console/adaptive/connectors",        connH.CreateConnector)
    router.Handle("POST", "/v1/console/adaptive/connectors/*/sync", connH.SyncConnector)
    router.Handle("POST", "/v1/sm/connections",                     connH.CreateConnection)
    router.Handle("POST", "/v1/sm/connections/*/sync",              connH.SyncConnection)

    // Console Memory Search (primary entry point for explorer)
    router.Handle("POST", "/v1/console/memory/search",              searchH.ConsoleSearch)
    router.Handle("GET",  "/v1/console/memory/*",                   searchH.GetMemory)
    router.Handle("GET",  "/v1/console/memory/*/neighbors",         searchH.GetNeighbors)
    router.Handle("GET",  "/v1/console/memory/*/versions",          searchH.GetVersions)
}
```

## Database Migration

```sql
-- migrations/001_search_init.sql

CREATE TABLE IF NOT EXISTS search_connectors (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      TEXT NOT NULL,
    name           TEXT NOT NULL,
    type           TEXT NOT NULL,
    config         JSONB NOT NULL DEFAULT '{}',
    sync_frequency TEXT NOT NULL DEFAULT 'manual',
    status         TEXT NOT NULL DEFAULT 'active',
    last_sync_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_connectors_tenant ON search_connectors(tenant_id);

CREATE TABLE IF NOT EXISTS search_sync_jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connector_id  UUID NOT NULL REFERENCES search_connectors(id),
    status        TEXT NOT NULL DEFAULT 'pending',
    items_synced  INT NOT NULL DEFAULT 0,
    started_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ,
    error         TEXT
);
```

## Config Environment Variables

```bash
GRPC_PORT=9090
HEALTH_PORT=9150
DATABASE_URL=postgres://...
NATS_URL=nats://nats:4222
# Backend service addresses (internal gRPC)
KG_SERVICE_ADDR=kg-service:9090
MEMORY_SERVICE_ADDR=memory-service:9090
STORAGE_SERVICE_ADDR=storage-service:9090
# Search config
SEARCH_DEFAULT_LIMIT=10
SEARCH_MAX_LIMIT=100
RRF_K=60
MMR_LAMBDA=0.7
```

## go.mod

```
module vnp-memory/services/search-service

go 1.25.0

require (
    vnp-memory/pkg/forward     v0.0.0
    vnp-memory/pkg/telemetry   v0.0.0
    vnp-memory/pkg/tenant      v0.0.0
    google.golang.org/grpc     v1.72.1
    github.com/jackc/pgx/v5    v5.7.0
)
```

## Acceptance Criteria

- [ ] `POST /v1/sm/search` với query → returns search results from all enabled engines
- [ ] `POST /v1/sm/rag` → returns RAG context with source references
- [ ] `POST /v1/ov/search` → delegates to kg-service + memory-service
- [ ] `POST /v1/console/memory/search` → cross-engine search với RRF reranking
- [ ] `POST /v1/sm/connections` → creates connector record
- [ ] `POST /v1/sm/connections/{id}/sync` → triggers async sync job via NATS
- [ ] RRF reranking combines results from 2+ engines correctly
- [ ] Fan-out concurrent search to multiple engines với timeout
- [ ] When engine unavailable → partial results returned (not full failure)
- [ ] `go build ./services/search-service/...` passes
- [ ] Unit tests pass (mock backend clients)

## Ghi Chú

- **Backend clients** (kg-service, memory-service) giao tiếp qua gRPC ForwardService protocol
- **MCP tools** (`sm-mcp`) → implement MCPTool registry trong service, expose qua gateway MCP server
- **vnp-search-hub** code tái sử dụng được: `SearchOrchestrator`, RRF/MMR reranking logic
- Tất cả 5 services gốc giữ nguyên cho đến P4 cleanup
