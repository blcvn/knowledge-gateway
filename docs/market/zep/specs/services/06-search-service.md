# 06 — Search Service (zep-search)

> **gRPC**: 9045 | **Health**: 9145  
> **Origin**: L4 — Graph Intelligence Layer (Search + Reranking)

---

## 1. Purpose

Semantic search across Zep's Temporal Knowledge Graph. Cung cấp:
- **Graph Search**: Multi-scope search (edges/nodes/episodes) with 5 reranking strategies
- **Session Search**: Search across sessions via graph facts
- **Context Retrieval**: Get relevant facts for memory assembly (called by zep-memory)
- **Fact Filtering**: Label, type, rating, and temporal filters
- **Cache Layer**: Redis-backed search result caching

---

## 2. Clean Architecture Layout

```
services/zep-search/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── search_query.go        # GraphSearchQuery, SessionSearchQuery
│   │   ├── search_result.go       # SearchResult, ranked items
│   │   ├── reranker.go            # RerankerType enum + strategy interface
│   │   ├── search_scope.go        # SearchScope (edges|nodes|episodes)
│   │   ├── search_filter.go       # NodeLabels, EdgeTypes, MinFactRating, MmrLambda
│   │   ├── event.go               # SearchPerformed (telemetry)
│   │   └── errors.go
│   │
│   ├── usecase/
│   │   ├── graph_search.go        # Multi-scope graph search with reranking
│   │   ├── session_search.go      # Search across sessions
│   │   ├── get_relevant_facts.go  # Get facts for context assembly (used by zep-memory)
│   │   ├── port/
│   │   │   ├── input.go           # SearchService interface
│   │   │   └── output.go          # GraphitiSearchClient, CacheStore
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go
│   │
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── client/
│   │   │   └── graphiti_client.go  # HTTP client → Graphiti search endpoint
│   │   ├── repository/
│   │   │   └── redis/
│   │   │       └── cache.go       # Search result cache
│   │   └── event/
│   │       └── subscriber.go     # Listen for graph.extraction.completed
│   │
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

---

## 3. Domain Layer

### 3.1 Search Query

```go
package domain

type GraphSearchQuery struct {
    Query          string           // semantic search text
    UserID         *string          // scope to user's graph
    GroupIDs       []string         // scope to specific groups
    Scope          SearchScope      // edges | nodes | episodes
    Reranker       RerankerType     // rrf | mmr | cross_encoder | node_distance | episode_mentions
    NodeLabels     []string         // filter by node labels
    EdgeTypes      []string         // filter by edge types
    Limit          int              // max results
    MinFactRating  *float64         // minimum fact quality rating
    MmrLambda      *float64         // diversity vs relevance tradeoff (for MMR)
    CenterNodeUUID *string          // center node for node_distance reranker
}

type SessionSearchQuery struct {
    Query      string
    UserID     *string
    SessionIDs []string
    Limit      int
    MaxFacts   int
}
```

### 3.2 Search Scope

```go
type SearchScope string

const (
    SearchScopeEdges    SearchScope = "edges"    // search facts (temporal edges)
    SearchScopeNodes    SearchScope = "nodes"    // search entities
    SearchScopeEpisodes SearchScope = "episodes" // search temporal events
    SearchScopeAll      SearchScope = "all"      // search everything (expensive)
)
```

### 3.3 Reranker Strategies

```go
type RerankerType string

const (
    RerankerRRF             RerankerType = "rrf"              // Reciprocal Rank Fusion — balanced multi-signal
    RerankerMMR             RerankerType = "mmr"              // Maximal Marginal Relevance — diversity-focused
    RerankerCrossEncoder    RerankerType = "cross_encoder"    // Neural cross-encoder — best accuracy, slower
    RerankerNodeDistance    RerankerType = "node_distance"    // Graph proximity — relationship-aware
    RerankerEpisodeMentions RerankerType = "episode_mentions" // Episode frequency — recency-aware
)
```

### 3.4 Search Result

```go
type SearchResult struct {
    Items     []SearchItem
    Total     int
    Query     string
    Scope     SearchScope
    Reranker  RerankerType
    LatencyMs int64
}

type SearchItem struct {
    // Common
    UUID      string
    Score     float64
    
    // For edges (facts)
    Fact      *FactResult
    
    // For nodes (entities)
    Node      *NodeResult
    
    // For episodes
    Episode   *EpisodeResult
}

type FactResult struct {
    Name       string
    Fact       string
    ValidAt    *time.Time
    InvalidAt  *time.Time
}

type SessionSearchResult struct {
    SessionID string
    Facts     []FactResult
    Score     float64
}
```

---

## 4. Use Case Layer

### 4.1 Port Interfaces

```go
package port

type SearchService interface {
    GraphSearch(ctx context.Context, req dto.GraphSearchRequest) (*dto.SearchResponse, error)
    SearchSessions(ctx context.Context, req dto.SessionSearchRequest) (*dto.SessionSearchResponse, error)
    GetRelevantFacts(ctx context.Context, groupID string, queryMessages []string, maxFacts int) ([]dto.FactResponse, error)
}

type GraphitiSearchClient interface {
    Search(ctx context.Context, req dto.GraphitiSearchRequest) ([]domain.SearchItem, error)
    GetMemory(ctx context.Context, groupID string, maxFacts int, queryMessages []string) ([]domain.FactResult, error)
}

type SearchCacheStore interface {
    Get(ctx context.Context, key string) (*domain.SearchResult, error)
    Set(ctx context.Context, key string, result *domain.SearchResult, ttl time.Duration) error
    Invalidate(ctx context.Context, groupID string) error
}
```

### 4.2 GraphSearch Use Case

```go
func (uc *GraphSearchUseCase) Execute(ctx context.Context, req dto.GraphSearchRequest) (*dto.SearchResponse, error) {
    start := time.Now()
    
    // 1. Check cache
    cacheKey := buildCacheKey(req)
    if cached, err := uc.cache.Get(ctx, cacheKey); err == nil && cached != nil {
        return dto.FromSearchResult(cached), nil
    }
    
    // 2. Build Graphiti search request
    searchReq := dto.GraphitiSearchRequest{
        Query:          req.Query,
        GroupIDs:       buildGroupIDs(req.UserID, req.GroupIDs),
        Scope:          req.Scope,
        Reranker:       req.Reranker,
        NodeLabels:     req.NodeLabels,
        EdgeTypes:      req.EdgeTypes,
        Limit:          req.Limit,
        MinFactRating:  req.MinFactRating,
        MmrLambda:      req.MmrLambda,
        CenterNodeUUID: req.CenterNodeUUID,
    }
    
    // 3. Execute search via Graphiti
    items, err := uc.graphitiClient.Search(ctx, searchReq)
    if err != nil {
        return nil, err
    }
    
    // 4. Build result
    result := &domain.SearchResult{
        Items:     items,
        Total:     len(items),
        Query:     req.Query,
        Scope:     domain.SearchScope(req.Scope),
        Reranker:  domain.RerankerType(req.Reranker),
        LatencyMs: time.Since(start).Milliseconds(),
    }
    
    // 5. Cache result (short TTL for temporal data freshness)
    uc.cache.Set(ctx, cacheKey, result, 30*time.Second)
    
    return dto.FromSearchResult(result), nil
}
```

### 4.3 GetRelevantFacts Use Case (for Memory Assembly)

```go
func (uc *GetRelevantFactsUseCase) Execute(ctx context.Context, groupID string, queryMessages []string, maxFacts int) ([]dto.FactResponse, error) {
    // Called by zep-memory during GetMemory
    // Uses last 4 messages as search context
    facts, err := uc.graphitiClient.GetMemory(ctx, groupID, maxFacts, queryMessages)
    if err != nil {
        return nil, err
    }
    return dto.FromFacts(facts), nil
}
```

---

## 5. gRPC Service Definition

```protobuf
syntax = "proto3";
package zep.search.v1;

service SearchService {
  rpc GraphSearch(GraphSearchRequest) returns (SearchResponse);
  rpc SearchSessions(SessionSearchRequest) returns (SessionSearchResponse);
  rpc GetRelevantFacts(GetRelevantFactsRequest) returns (FactListResponse);
}

message GraphSearchRequest {
  string query = 1;
  optional string user_id = 2;
  repeated string group_ids = 3;
  string scope = 4;              // "edges" | "nodes" | "episodes" | "all"
  string reranker = 5;           // "rrf" | "mmr" | "cross_encoder" | "node_distance" | "episode_mentions"
  repeated string node_labels = 6;
  repeated string edge_types = 7;
  int32 limit = 8;
  optional double min_fact_rating = 9;
  optional double mmr_lambda = 10;
  optional string center_node_uuid = 11;
}

message SearchResponse {
  repeated SearchItem items = 1;
  int32 total = 2;
  string query = 3;
  string scope = 4;
  string reranker = 5;
  int64 latency_ms = 6;
}

message SearchItem {
  string uuid = 1;
  double score = 2;
  oneof result {
    FactResult fact = 3;
    NodeResult node = 4;
    EpisodeResult episode = 5;
  }
}

message FactResult {
  string name = 1;
  string fact = 2;
  optional google.protobuf.Timestamp valid_at = 3;
  optional google.protobuf.Timestamp invalid_at = 4;
}

message NodeResult {
  string name = 1;
  string node_type = 2;
  string summary = 3;
  repeated string labels = 4;
}

message EpisodeResult {
  string name = 1;
  string content = 2;
  string group_id = 3;
  google.protobuf.Timestamp created_at = 4;
}

message SessionSearchRequest {
  string query = 1;
  optional string user_id = 2;
  repeated string session_ids = 3;
  int32 limit = 4;
  int32 max_facts = 5;
}

message SessionSearchResponse {
  repeated SessionSearchItem results = 1;
}

message SessionSearchItem {
  string session_id = 1;
  repeated FactResult facts = 2;
  double score = 3;
}

message GetRelevantFactsRequest {
  string group_id = 1;
  repeated string query_messages = 2;   // last 4 messages as context
  int32 max_facts = 3;                  // default: 5
}
```

---

## 6. Reranking Strategy Details

| Reranker | Algorithm | Latency | Best For |
|----------|-----------|---------|----------|
| `rrf` | Reciprocal Rank Fusion | Low | General-purpose, balanced multi-signal |
| `mmr` | Maximal Marginal Relevance | Low | Diverse results, avoid redundancy |
| `cross_encoder` | Neural cross-encoder | High | Best accuracy, important queries |
| `node_distance` | Graph shortest path | Medium | Relationship-aware, context-sensitive |
| `episode_mentions` | Episode co-occurrence frequency | Low | Recency-aware, trending topics |

### Configuration per Reranker

```go
type RerankerConfig struct {
    RRF struct {
        K int  // fusion constant, default 60
    }
    MMR struct {
        Lambda float64  // 0.0=diversity, 1.0=relevance, default 0.5
    }
    CrossEncoder struct {
        Model string  // model name
        BatchSize int // inference batch size
    }
    NodeDistance struct {
        MaxDepth int  // graph traversal depth, default 3
    }
    EpisodeMentions struct {
        TimeDecay float64  // exponential decay factor
    }
}
```

---

## 7. NATS Events

### Consumed

| Subject | Source | Action |
|---------|--------|--------|
| `zep.graph.extraction.completed` | zep-graph | Invalidate search cache for group |
| `zep.graph.fact.created` | zep-graph | Update cache with new fact |
| `zep.graph.fact.invalidated` | zep-graph | Remove invalidated fact from cache |

---

## 8. Configuration

```yaml
search:
  grpc:
    port: 9045
  health:
    port: 9145
  graphiti:
    service_url: "http://graphiti:8003"
    timeout: 30s
  redis:
    url: "redis://redis:6379/1"
    cache_ttl: 30s                  # short TTL for temporal data freshness
  reranker:
    default: "rrf"
    rrf:
      k: 60
    mmr:
      default_lambda: 0.5
    cross_encoder:
      model: "cross-encoder/ms-marco-MiniLM-L-12-v2"
      batch_size: 32
    node_distance:
      max_depth: 3
    episode_mentions:
      time_decay: 0.95
  nats:
    url: "nats://nats:4222"
    stream: "zep"
    consumer_group: "zep-search"
  telemetry:
    service_name: "zep-search"
    otel_endpoint: "otel-collector:4317"
```
