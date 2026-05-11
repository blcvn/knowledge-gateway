---
id: DOC-S04
service: graphiti-search
version: 2.0.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# graphiti-search — Data Model

> **Group**: Graphiti | **Cache**: Redis | **Upstream**: graphiti-store (Neo4j)

## Domain Types

graphiti-search is a stateless query service — it does not own any persistent data. It reads from graphiti-store and caches results in Redis.

### Search Query

```go
type SearchQuery struct {
    Query          string         // Natural language query
    GroupID        string         // Tenant partition
    Methods        []SearchMethod // cosine_similarity, bm25, breadth_first_search
    Rerankers      []RerankerType // rrf, mmr, cross_encoder, node_distance, episode_mentions
    Limit          int            // Max results to return
    TemporalFilter *TemporalWindow // Optional time range filter
    EntityLabels   []string       // Optional label filter (e.g. "Person", "Organization")
}
```

### Search Result

```go
type SearchResult struct {
    EntityUUID   string         // Node/Edge UUID from graphiti-store
    EntityType   string         // "entity", "edge", "community"
    Name         string         // Entity name
    Summary      string         // Entity summary
    Fact         string         // Edge fact (if edge result)
    Score        float64        // Relevance score
    Method       SearchMethod   // Which method produced this result
    ValidAt      *time.Time     // Temporal validity (if applicable)
    InvalidAt    *time.Time
}

type RankedResult struct {
    SearchResult
    FinalScore    float64       // After reranking
    RankPosition  int           // Final rank (1-based)
    RerankerScores map[RerankerType]float64 // Score breakdown per reranker
}
```

### Redis Cache Schema

| Key Pattern | Value | TTL |
|------------|-------|-----|
| `search:{group_id}:{query_hash}` | Protobuf-encoded `[]RankedResult` | 5min (configurable) |

- `query_hash` = SHA-256 of `query + methods + rerankers + limit + temporal_filter`
- Cache invalidated on `graphiti.episode.ingested` NATS event (per group_id)

## No Persistent Storage

graphiti-search does not own any PostgreSQL tables or Neo4j labels. All graph data is accessed via graphiti-store gRPC client.
