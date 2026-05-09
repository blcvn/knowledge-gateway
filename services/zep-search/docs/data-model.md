---
id: DOC-S04
service: zep-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-search — Data Model

> **Storage**: Redis (cache) + Graphiti (search engine)

## Cache Structure (Redis)

| Key Pattern | Value | TTL |
|------------|-------|-----|
| `search:{hash(query+scope+reranker+filters)}` | JSON SearchResult | 30s |
| `group:{group_id}:*` | — | Invalidated on extraction.completed |

## Search Result Model

```go
type SearchResult struct {
    Items     []SearchItem
    Total     int
    Query     string
    Scope     SearchScope    // edges | nodes | episodes | all
    Reranker  RerankerType   // rrf | mmr | cross_encoder | node_distance | episode_mentions
    LatencyMs int64
}

type SearchItem struct {
    UUID    string
    Score   float64
    Fact    *FactResult    // for edges
    Node    *NodeResult    // for nodes
    Episode *EpisodeResult // for episodes
}
```

## Reranker Configuration

```go
type RerankerConfig struct {
    RRF: { K: 60 }
    MMR: { Lambda: 0.5 }
    CrossEncoder: { Model: "ms-marco-MiniLM-L-12-v2", BatchSize: 32 }
    NodeDistance: { MaxDepth: 3 }
    EpisodeMentions: { TimeDecay: 0.95 }
}
```
