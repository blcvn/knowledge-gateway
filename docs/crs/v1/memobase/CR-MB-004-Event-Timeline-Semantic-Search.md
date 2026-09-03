# Change Request: CR-MB-004 — Event Timeline & Semantic Search Service

**CR ID:** CR-MB-004  
**Component:** `services/memobase-event` [NEW SERVICE]  
**Priority:** High  
**Status:** In Progress
**Reference:** memobase PRD §5.4 (F-4), SRS §3.7, specs/services/05-memobase-event.md  
**Maps to Python:** `controllers/event.py`, `controllers/event_gist.py`

---

## 1. Mô tả

Xây dựng **memobase-event** service — event timeline management với **vector similarity search**:
1. **Event CRUD** — get, update, delete timeline events.
2. **Semantic Search on Events** — cosine similarity via pgvector (event-level).
3. **Event Gist Search** — fine-grained semantic search trên event descriptions.
4. **Tag-Based Filtering** — JSONB query với custom tags (emotion, goal, etc.).
5. **Time-Range Filtering** — filter events theo `created_at` window.
6. **Embedding Indexing** — HNSW vector index cho efficient ANN search.

---

## 2. Vấn đề hiện tại

VNP Memory hiện tại:
- ✅ Có basic event storage.
- ❌ Không có **semantic event search** (cosine similarity với pgvector HNSW).
- ❌ Không có **event gist** concept (fine-grained descriptions tách từ event_tip).
- ❌ Không có **event gist search** (separate pgvector search on gists table).
- ❌ Không có **event tag filtering** (JSONB `@>` containment queries).
- ❌ Không có **time_range_in_days** filter.
- ❌ Không có **similarity threshold** filtering (`> 0.2` default).
- ❌ Không có **HNSW index** setup (`m=16, ef_construction=200`).
- ❌ Không có **embedding dimension validation** at startup.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/memobase-event/`

**Port:** `9044` (gRPC internal), **Health:** `9094`

### 3.2. Domain Models

```go
// internal/domain/entity.go

type Event struct {
    ID         uuid.UUID
    UserID     uuid.UUID
    ProjectID  string
    EventData  EventData
    Embedding  []float32   // pgvector (1536 dim)
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type EventData struct {
    EventTip     string       `json:"event_tip"`       // Full summary
    EventTags    []EventTag   `json:"event_tags"`
    ProfileDelta []any        `json:"profile_delta"`
}

type EventTag struct {
    Tag   string `json:"tag"`
    Value string `json:"value"`
}

type EventGist struct {
    ID        uuid.UUID
    EventID   uuid.UUID
    UserID    uuid.UUID
    ProjectID string
    GistData  GistData
    Embedding []float32   // pgvector (1536 dim)
    CreatedAt time.Time
}

type GistData struct {
    GistContent string `json:"gist_content"`  // Single line from event_tip
}

type SearchResult struct {
    EventID    uuid.UUID
    Event      Event   // or EventGist
    Similarity float64 // cosine similarity score (0.0 - 1.0)
}
```

### 3.3. Semantic Event Search (pgvector)

```go
// internal/usecase/search_events.go

func (uc *SearchEventsUseCase) Execute(ctx context.Context, req SearchEventsRequest) ([]SearchResult, error) {
    // 1. Embed query text
    if !uc.embedder.IsEnabled() {
        return nil, domain.ErrEmbeddingDisabled
    }
    queryEmbedding, err := uc.embedder.EmbedQuery(ctx, req.Query)

    // 2. pgvector cosine similarity search
    // SELECT e.*, 1 - (embedding <=> $queryVec) AS similarity
    // FROM user_events e
    // WHERE e.user_id = $1 AND e.project_id = $2
    //   AND 1 - (embedding <=> $queryVec) > $threshold  -- default 0.2
    //   AND e.created_at > now() - ($timeRangeDays * interval '1 day')
    // ORDER BY similarity DESC
    // LIMIT $topk
    results, err := uc.eventRepo.SearchByEmbedding(ctx, SearchByEmbeddingQuery{
        UserID:     req.UserID,
        ProjectID:  req.ProjectID,
        Vector:     queryEmbedding,
        Threshold:  req.SimilarityThreshold,  // default: 0.2
        TimeRange:  req.TimeRangeDays,         // default: 21
        Limit:      req.TopK,                  // default: 10
    })

    return results, err
}
```

### 3.4. Event Gist Search

```go
// internal/usecase/search_gists.go
// Similar to search_events but on user_event_gists table
// Fine-grained results — each gist is a single line from event_tip

// pgvector query:
// SELECT g.*, 1 - (embedding <=> $queryVec) AS similarity
// FROM user_event_gists g
// WHERE g.user_id = $1 AND g.project_id = $2
//   AND 1 - (embedding <=> $queryVec) > $threshold
// ORDER BY similarity DESC LIMIT $topk
```

### 3.5. Tag-Based Filtering

```go
// internal/usecase/filter_by_tags.go

func (r *EventRepo) FilterByTags(ctx context.Context,
    userID, projectID string,
    hasTags []string,             // has_event_tag: check tag exists
    equalTags map[string]string,  // event_tag_equal: check tag=value
    timeRangeDays int,
) ([]Event, error) {
    query := `SELECT * FROM user_events
              WHERE user_id = $1 AND project_id = $2
              AND created_at > now() - ($3 * interval '1 day')`

    // JSONB containment: check tag exists
    for _, tag := range hasTags {
        query += fmt.Sprintf(` AND event_data->'event_tags' @> '[{"tag":"%s"}]'`, tag)
    }

    // JSONB containment: check tag=value pair
    for tag, value := range equalTags {
        query += fmt.Sprintf(` AND event_data->'event_tags' @> '[{"tag":"%s","value":"%s"}]'`, tag, value)
    }

    return r.db.QueryContext(ctx, query, userID, projectID, timeRangeDays)
}

// API:
// GET /api/v1/users/event_tags/search/{user_id}
//   ?has_event_tag=emotion&has_event_tag=goal
//   &event_tag_equal[goal]=fitness&time_range_in_days=21
```

### 3.6. Append Event (from Engine)

```go
// internal/usecase/append_event.go
// Called by Engine service via NATS or gRPC to store new event

func (uc *AppendEventUseCase) Execute(ctx context.Context, req AppendEventRequest) (*AppendEventResult, error) {
    // 1. Store event in user_events
    event, err := uc.eventRepo.Save(ctx, Event{
        UserID:    req.UserID,
        ProjectID: req.ProjectID,
        EventData: req.EventData,
        Embedding: req.Embedding,  // pre-computed by engine
    })

    // 2. Create EventGists from event_tip lines
    gists := parseGistsFromEventTip(req.EventData.EventTip)
    for i, gistContent := range gists {
        gistEmbedding := req.GistEmbeddings[i]  // pre-computed by engine
        uc.gistRepo.Save(ctx, EventGist{
            EventID:   event.ID,
            UserID:    req.UserID,
            ProjectID: req.ProjectID,
            GistData:  GistData{GistContent: gistContent},
            Embedding: gistEmbedding,
        })
    }

    return &AppendEventResult{EventID: event.ID, GistCount: len(gists)}, nil
}
```

### 3.7. Vector Index Management

```sql
-- HNSW index for better recall (PostgreSQL pgvector)
CREATE INDEX idx_user_events_embedding
    ON user_events
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 200);

CREATE INDEX idx_user_event_gists_embedding
    ON user_event_gists
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 200);

-- Startup dimension validation:
-- SELECT vector_dims(embedding) FROM user_events LIMIT 1
-- → must match config.embedding.dimension (1536)
-- → if mismatch: fail fast with clear error message
```

### 3.8. Database Schema

```sql
CREATE TABLE user_events (
    id          UUID NOT NULL,
    project_id  VARCHAR NOT NULL,
    user_id     UUID NOT NULL,
    event_data  JSONB NOT NULL,    -- {event_tip, event_tags[], profile_delta}
    embedding   vector(1536),      -- pgvector
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE
);
CREATE INDEX idx_user_events_user_id_project_id ON user_events(user_id, project_id);
CREATE INDEX idx_user_events_created_at ON user_events(user_id, project_id, created_at DESC);

CREATE TABLE user_event_gists (
    id          UUID NOT NULL,
    project_id  VARCHAR NOT NULL,
    event_id    UUID NOT NULL,
    user_id     UUID NOT NULL,
    gist_data   JSONB NOT NULL,    -- {gist_content}
    embedding   vector(1536),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (event_id, project_id) REFERENCES user_events(id, project_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE
);
CREATE INDEX idx_user_event_gists_user_id_project_id ON user_event_gists(user_id, project_id);
```

### 3.9. gRPC API

```protobuf
service EventService {
    // CRUD
    rpc GetEvents(GetEventsRequest) returns (GetEventsResponse);
    rpc UpdateEvent(UpdateEventRequest) returns (UpdateEventResponse);
    rpc DeleteEvent(DeleteEventRequest) returns (DeleteEventResponse);

    // Semantic Search
    rpc SearchEvents(SearchEventsRequest) returns (SearchEventsResponse);
    rpc SearchEventGists(SearchEventGistsRequest) returns (SearchEventGistsResponse);

    // Tag Filtering
    rpc FilterEventsByTags(FilterByTagsRequest) returns (FilterByTagsResponse);

    // Internal (from engine)
    rpc AppendEvent(AppendEventRequest) returns (AppendEventResponse);

    // Recent events for context
    rpc GetRecentEventGists(GetRecentEventGistsRequest) returns (GetRecentEventGistsResponse);
}

message SearchEventsRequest {
    string user_id = 1;
    string project_id = 2;
    string query = 3;
    int32 topk = 4;               // default: 10
    double similarity_threshold = 5; // default: 0.2
    int32 time_range_in_days = 6; // default: 21
}

message FilterByTagsRequest {
    string user_id = 1;
    string project_id = 2;
    repeated string has_event_tag = 3;          // tag exists check
    map<string, string> event_tag_equal = 4;    // tag=value check
    int32 time_range_in_days = 5;
}
```

### 3.10. REST Endpoints

```
GET    /api/v1/users/event/{user_id}?topk=10&time_range_in_days=21
PUT    /api/v1/users/event/{user_id}/{event_id}
DELETE /api/v1/users/event/{user_id}/{event_id}
GET    /api/v1/users/event/search/{user_id}?query=...&topk=10&threshold=0.2
GET    /api/v1/users/event_gist/search/{user_id}?query=...&topk=10
GET    /api/v1/users/event_tags/search/{user_id}?has_event_tag=emotion&event_tag_equal[goal]=fitness
```

---

## 4. Configuration

```yaml
event:
  grpc:
    port: 9044
  health:
    port: 9094
  embedding:
    provider: "openai"                # openai | jina | ollama
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
    max_overflow: 10
```

---

## 5. Acceptance Criteria

- [ ] Sau engine flush với "discussed project deadline and stress" → `GET /api/v1/users/event/{user_id}` trả về event với `event_data.event_tip` chứa summary.
- [ ] `GET /api/v1/users/event/search/{user_id}?query=project+deadline` → trả về event liên quan với `similarity > 0.2`.
- [ ] Gist search: `GET /api/v1/users/event_gist/search/{user_id}?query=stress` → trả về gist "Mentioned feeling stressed" (fine-grained match).
- [ ] Tag filter: event có `event_tags=[{tag: "emotion", value: "stressed"}]` → `?has_event_tag=emotion` → event appears trong kết quả.
- [ ] `?event_tag_equal[goal]=fitness` → chỉ events có `tag: goal, value: fitness` được trả về.
- [ ] `time_range_in_days=7` → chỉ events trong 7 ngày gần nhất.
- [ ] Embedding dimension mismatch (config=1536, DB=768) → startup error message rõ ràng.
- [ ] `enable_event_embedding=false` → `GET /api/v1/users/event/search/{user_id}` → HTTP 501 Not Implemented.
- [ ] HNSW index created: `\d user_events` trong Neo4j → index exists (check migration).
- [ ] `DELETE /api/v1/users/event/{user_id}/{event_id}` → event + all gists bị xóa (cascade).
