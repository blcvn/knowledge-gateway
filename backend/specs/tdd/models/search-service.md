# search-service — Data Models

> **Service**: `services/search-service`
> **Role**: Unified search facade — hybrid BM25 + vector retrieval over all memory engines.
> Orchestrates results from observation stores, memory stores, and agent context.

---

## SearchResult

```go
type SearchResult struct {
    DocID         string
    SessionID     string
    ObsType       string
    Title         string
    Narrative     string
    Facts         []string
    Concepts      []string
    CombinedScore float64
    BM25Score     float64
    VectorScore   float64
}
```

---

## ContextBlock

```go
type ContextBlock struct {
    Type    string  // "memory" | "summary" | "observation"
    Content string
    Tokens  int
    Recency float64
    Source  string
}
```

---

## Observation

```go
type Observation struct {
    ObsType   string
    Facts     []string
    Concepts  []string
    Title     string
    Narrative string
}
```

---

## Summary

```go
type Summary struct {
    Narrative string
}
```

---

## AgentMemory

```go
type AgentMemory struct {
    ID        string
    Type      string
    Title     string
    Content   string
    UpdatedAt time.Time
}
```

---

## connector — Data Source Connector

```go
// See services/search-service/internal/domain/connector/entity.go
```

---

## Sources
- [`services/search-service/internal/domain/entity.go`](../../services/search-service/internal/domain/entity.go)
- [`services/search-service/internal/domain/search/entity.go`](../../services/search-service/internal/domain/search/entity.go)
- [`services/search-service/internal/domain/connector/entity.go`](../../services/search-service/internal/domain/connector/entity.go)
