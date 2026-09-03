# ov-search — Data Models

> **Service**: `services/ov-search`
> **Role**: OpenViking search — hybrid semantic + hotness-weighted retrieval.

---

## EmbeddingVector

```go
type EmbeddingVector struct {
    ID           string    `json:"id"`
    Vector       []float32 `json:"vector"`         // 1536-dim dense
    SparseVector []float32 `json:"sparse_vector"`  // BM25 sparse
}
```

---

## UpsertPayload

```go
type UpsertPayload struct {
    Path         string    `json:"path"`
    AccountID    string    `json:"account_id"`
    UserID       string    `json:"user_id"`
    ContentHash  string    `json:"content_hash"`
    ContextLevel string    `json:"context_level"`
    ChunkIndex   int       `json:"chunk_index"`
    ParentDir    string    `json:"parent_dir"`
    MimeType     string    `json:"mime_type"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

---

## SearchResult

```go
type SearchResult struct {
    ID             string         `json:"id"`
    Path           string         `json:"path"`
    SemanticScore  Score          `json:"semantic_score"`
    HotnessScore   Score          `json:"hotness_score"`
    FinalScore     Score          `json:"final_score"`
    MatchedContext MatchedContext `json:"matched_context"`
}

type MatchedContext struct {
    Content     string `json:"content"`
    ContextType string `json:"context_type"`
    DepthLevel  string `json:"depth_level"`
}
```

---

## HotnessScore

```go
type HotnessScore struct {
    Path            string    `json:"path"`
    AccountID       string    `json:"account_id"`
    BaseScore       float64   `json:"base_score"`
    AccessCount     int       `json:"access_count"`
    SessionRefCount int       `json:"session_ref_count"`
    ComputedHotness float64   `json:"computed_hotness"`
    LastAccessedAt  time.Time `json:"last_accessed_at"`
    LastModifiedAt  time.Time `json:"last_modified_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}

type DecayConfig struct {
    HalfLifeHours float64
    SessionBoost  float64
}
```

---

## Sources
- [`services/ov-search/internal/domain/model/embedding.go`](../../services/ov-search/internal/domain/model/embedding.go)
- [`services/ov-search/internal/domain/model/search_result.go`](../../services/ov-search/internal/domain/model/search_result.go)
- [`services/ov-search/internal/domain/model/hotness.go`](../../services/ov-search/internal/domain/model/hotness.go)
- [`services/ov-search/internal/domain/model/context_type.go`](../../services/ov-search/internal/domain/model/context_type.go)
