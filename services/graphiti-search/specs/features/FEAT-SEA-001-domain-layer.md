---
id: FEAT-SEA-001
title: Domain Layer — Search Types + Reranker Interfaces
service: graphiti-search
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement domain layer cho graphiti-search: SearchQuery, SearchResult, SearchFilter, Reranker types, SearchMethod/RerankerType enums, và domain errors.

## Scope

- `internal/domain/entity.go` — SearchQuery, SearchResult, RankedResult
- `internal/domain/value_object.go` — SearchMethod, RerankerType, ScoreWeight, TemporalWindow
- `internal/domain/config.go` — SearchConfig, RerankerConfig, CacheConfig
- `internal/domain/errors.go` — ErrNoResults, ErrInvalidQuery, ErrCacheUnavailable

### Key Types

```go
type SearchMethod string
const (
    MethodCosine  SearchMethod = "cosine_similarity"
    MethodBM25    SearchMethod = "bm25"
    MethodBFS     SearchMethod = "breadth_first_search"
)

type RerankerType string
const (
    RerankerRRF              RerankerType = "rrf"
    RerankerMMR              RerankerType = "mmr"
    RerankerCrossEncoder     RerankerType = "cross_encoder"
    RerankerNodeDistance      RerankerType = "node_distance"
    RerankerEpisodeMentions  RerankerType = "episode_mentions"
)

type SearchQuery struct {
    Query          string
    GroupID        string
    Methods        []SearchMethod
    Rerankers      []RerankerType
    Limit          int
    TemporalFilter *TemporalWindow
    EntityLabels   []string
}
```

## Acceptance Criteria

- [ ] AC-1: Domain compiles with ZERO external imports
- [ ] AC-2: SearchMethod and RerankerType enums have validation
- [ ] AC-3: SearchQuery.Validate() enforces: non-empty query, at least 1 method, limit > 0
- [ ] AC-4: TemporalWindow.Validate() enforces: from < to

## Test Requirements
- **Unit tests**: Validation methods, value object constructors
- **Minimum coverage**: 90%
