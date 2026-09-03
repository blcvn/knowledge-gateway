# 05 — Memobase Event Service

> **gRPC**: 9044 | **Health**: 9094

---

## 1. Purpose

Quản lý event timeline và semantic search: CRUD events, vector similarity search trên events/gists, tag-based filtering. Tách biệt workload vector search khỏi profile operations.

---

## 2. Clean Architecture

```
services/memobase-event/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # Event, EventGist, EventTag
│   │   ├── value_object.go     # EmbeddingVector, SimilarityScore, TimeRange
│   │   └── errors.go           # ErrEventNotFound, ErrEmbeddingDisabled
│   ├── usecase/
│   │   ├── get_events.go       # List events by user (paginated)
│   │   ├── update_event.go
│   │   ├── delete_event.go
│   │   ├── search_events.go    # Semantic vector search on events
│   │   ├── search_gists.go     # Semantic vector search on event gists
│   │   ├── filter_by_tags.go   # JSONB tag filtering
│   │   ├── append_event.go     # Store event + generate embeddings + gists
│   │   ├── port/
│   │   │   ├── input.go        # SearchEventsUseCase, FilterByTagsUseCase
│   │   │   └── output.go       # EventRepository, GistRepository, EmbedderClient
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go      # memobase.event.v1.EventService impl
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── event_repo.go     # user_events + pgvector search
│   │   │       └── gist_repo.go      # user_event_gists + pgvector search
│   │   ├── client/
│   │   │   └── embedder_client.go    # Embedding provider
│   │   └── event/
│   │       └── subscriber.go   # NATS: memobase.event.created
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

---

## 3. Domain Entities

```go
type Event struct {
    ID         uuid.UUID
    UserID     uuid.UUID
    ProjectID  string
    EventData  EventData
    Embedding  []float32        // pgvector
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type EventData struct {
    EventTip     string      `json:"event_tip"`
    EventTags    []EventTag  `json:"event_tags"`
    ProfileDelta []any       `json:"profile_delta"`
}

type EventGist struct {
    ID        uuid.UUID
    EventID   uuid.UUID
    UserID    uuid.UUID
    ProjectID string
    GistData  GistData
    Embedding []float32
    CreatedAt time.Time
}

type GistData struct {
    GistContent string `json:"gist_content"`
}

type SearchResult struct {
    Event      Event    // or EventGist
    Similarity float64  // cosine similarity score
}
```

---

## 4. Use Case Flow: SearchEvents

```
Client → Gateway → gRPC SearchEvents(user_id, query, topk, threshold)
                        │
                        ▼
        ┌──── SearchEventsUseCase ────────────────────┐
        │ 1. Embed query text                          │
        │    embedder.Embed(ctx, []string{query})      │
        │                                              │
        │ 2. pgvector cosine similarity search         │
        │    SELECT *, 1 - (embedding <=> $query)      │
        │      AS similarity                           │
        │    FROM user_events                          │
        │    WHERE user_id = $1                        │
        │      AND project_id = $2                     │
        │      AND similarity > $threshold (0.2)       │
        │      AND created_at > now() - interval $days │
        │    ORDER BY similarity DESC                  │
        │    LIMIT $topk                               │
        │                                              │
        │ 3. Return []SearchResult                     │
        └──────────────────────────────────────────────┘
```

## 5. Use Case Flow: FilterByTags

```go
// SQL: JSONB containment query
// WHERE event_data @> '{"event_tags": [{"tag": "emotion"}]}'
// AND   event_data @> '{"event_tags": [{"tag": "goal", "value": "fitness"}]}'

func (r *EventRepo) FilterByTags(ctx context.Context,
    userID, projectID string,
    hasTags []string,           // has_event_tag: check tag exists
    equalTags map[string]string, // event_tag_equal: check tag=value
) ([]Event, error) {
    query := `SELECT * FROM user_events WHERE user_id = $1 AND project_id = $2`
    for _, tag := range hasTags {
        query += fmt.Sprintf(` AND event_data->'event_tags' @> '[{"tag":"%s"}]'`, tag)
    }
    for tag, value := range equalTags {
        query += fmt.Sprintf(` AND event_data->'event_tags' @> '[{"tag":"%s","value":"%s"}]'`, tag, value)
    }
    return r.db.QueryContext(ctx, query, userID, projectID)
}
```

---

## 6. Embedding Port Interface

```go
type EmbedderClient interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedQuery(ctx context.Context, query string) ([]float32, error)
    Dimension() int
    IsEnabled() bool
}
```

**Providers**: OpenAI, Jina, Ollama, LMStudio (from `pkg/adapters/embedder/`)

---

## 7. Vector Index Management

```sql
-- pgvector index (HNSW for better recall)
CREATE INDEX idx_user_events_embedding
    ON user_events
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 200);

CREATE INDEX idx_user_event_gists_embedding
    ON user_event_gists
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 200);

-- Dimension validation at startup
SELECT vector_dims(embedding)
FROM user_events LIMIT 1;
-- Must match config.embedding.dimension
```

---

## 8. Configuration

```yaml
event:
  grpc:
    port: 9044
  health:
    port: 9094
  embedding:
    provider: "openai"
    model: "text-embedding-3-small"
    dimension: 1536
    enabled: true
  search:
    default_topk: 10
    default_threshold: 0.2
    default_time_range_days: 21
  database:
    url: "${DATABASE_URL}"
    pool_size: 25
```
