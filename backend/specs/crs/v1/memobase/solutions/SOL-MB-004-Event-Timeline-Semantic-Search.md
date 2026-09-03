# Solution: SOL-MB-004 — Event Timeline & Semantic Search Service

**CR:** [CR-MB-004](../CR-MB-004-Event-Timeline-Semantic-Search.md)  
**Wave:** 3 (Read Path)  
**Priority:** High  
**Status:** Draft  
**Date:** 2026-06-17

---

## 1. Tổng quan Giải pháp

Xây dựng `services/memobase-event` — event timeline service với **vector similarity search** qua pgvector HNSW index. Service này cung cấp cả event-level search và gist-level fine-grained search.

### Chiến lược chính

| Vấn đề | Giải pháp |
|---|---|
| Không có semantic search | pgvector cosine similarity trên `user_events.embedding` |
| Không có event gist | Separate `user_event_gists` table với per-gist embeddings |
| Không có tag filtering | JSONB `@>` containment queries trên `event_data.event_tags` |
| Không có HNSW index | Migration tạo HNSW index `m=16, ef_construction=200` |
| Không có embedding dim validation | Startup check: SELECT vector_dims() vs config |
| Không có time range filter | `created_at > NOW() - ($days * interval '1 day')` |

---

## 2. Thiết kế Service

### 2.1 Mối quan hệ với memobase-engine

`memobase-event` **không tự tạo events**. Events được tạo bởi `memobase-engine` sau mỗi LLM pipeline flush:
1. Engine tính toán `event_tip`, `event_tags`, `event_gists`
2. Engine tính embedding (openai/jina/ollama)
3. Engine gọi `event.AppendEvent()` qua gRPC
4. memobase-event lưu vào PostgreSQL

### 2.2 Luồng dữ liệu

```
memobase-engine (sau flush):
    → gRPC: event.AppendEvent({
        user_id, project_id,
        event_data: {event_tip, event_tags[], profile_delta},
        embedding: [1536 floats],  ← pre-computed bởi engine
        gist_embeddings: [[floats], ...]  ← per-gist embeddings
      })
    
memobase-event:
    → INSERT INTO user_events (embedding = $vector)
    → INSERT INTO user_event_gists (embedding = $gist_vector) × N
    → NATS ACK (implicit via AppendEvent response)

memobase-context (sau đó):
    → gRPC: event.SearchEventGists({user_id, query_embedding, topk, threshold})
    → pgvector: SELECT ... ORDER BY embedding <=> $vec LIMIT topk
    → return []EventGist
```

---

## 3. Database Schema & Indexes

**File:** `services/memobase-event/internal/infra/migrations/001_init.up.sql`

```sql
-- Bảng user_events (cùng PG instance với engine service)
-- Schema đã được tạo bởi memobase-engine migration
-- memobase-event chỉ cần READ + index management

-- HNSW Vector Index (tạo sau khi bảng có dữ liệu)
-- Migration 002: thêm sau initial data load
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_events_embedding_hnsw
    ON user_events
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 200);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_event_gists_embedding_hnsw
    ON user_event_gists
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 200);

-- JSONB index for tag filtering
CREATE INDEX IF NOT EXISTS idx_user_events_tags
    ON user_events USING gin(event_data);
```

**Lý do dùng `CONCURRENTLY`:** Tránh lock toàn bảng khi tạo HNSW index trên dữ liệu lớn.

---

## 4. Startup Validation

```go
// infra/startup/validator.go

func ValidateEmbeddingDimension(db *sql.DB, configDim int) error {
    var dbDim int
    err := db.QueryRow(`
        SELECT vector_dims(embedding)
        FROM user_events
        WHERE embedding IS NOT NULL
        LIMIT 1
    `).Scan(&dbDim)

    if err == sql.ErrNoRows {
        // Bảng trống → OK, chưa có data
        return nil
    }
    if err != nil {
        return fmt.Errorf("startup: cannot validate embedding dimension: %w", err)
    }

    if dbDim != configDim {
        return fmt.Errorf(
            "Embedding dimension mismatch: config=%d, DB=%d. "+
                "Run migration to recreate embeddings or update embedding_dim in config.",
            configDim, dbDim,
        )
    }
    return nil
}
```

---

## 5. Semantic Search Implementation

### 5.1 Event-Level Search

```go
// usecase/search_events.go

func (uc *SearchEventsUseCase) Execute(ctx context.Context, req SearchEventsRequest) ([]SearchResult, error) {
    if !uc.embedder.IsEnabled() {
        return nil, domain.ErrEmbeddingDisabled  // → HTTP 501
    }

    // Embed query
    queryVec, err := uc.embedder.EmbedQuery(ctx, req.Query)
    if err != nil {
        return nil, fmt.Errorf("embed query: %w", err)
    }

    // pgvector cosine similarity search
    return uc.eventRepo.SearchByEmbedding(ctx, SearchByEmbeddingQuery{
        UserID:     req.UserID,
        ProjectID:  req.ProjectID,
        Vector:     queryVec,
        Threshold:  req.SimilarityThreshold,  // default: 0.2
        TimeRange:  req.TimeRangeDays,          // default: 21
        Limit:      req.TopK,                   // default: 10
    })
}
```

```sql
-- adapter/repository/postgres/event_repo.go

SELECT 
    e.id, e.user_id, e.project_id, e.event_data, e.created_at,
    1 - (e.embedding <=> $4) AS similarity
FROM user_events e
WHERE 
    e.user_id = $1 
    AND e.project_id = $2
    AND 1 - (e.embedding <=> $4) > $5  -- threshold
    AND e.created_at > NOW() - ($3 * INTERVAL '1 day')  -- time range
ORDER BY similarity DESC
LIMIT $6;
```

### 5.2 Event Gist Search (fine-grained)

```sql
-- adapter/repository/postgres/gist_repo.go

SELECT 
    g.id, g.event_id, g.user_id, g.project_id, g.gist_data, g.created_at,
    1 - (g.embedding <=> $3) AS similarity
FROM user_event_gists g
WHERE 
    g.user_id = $1 
    AND g.project_id = $2
    AND 1 - (g.embedding <=> $3) > $4  -- threshold
ORDER BY similarity DESC
LIMIT $5;
```

**Ưu điểm của gist search:** Event_tip thường là 50-200 tokens, embedding đại diện cho toàn bộ event. Gist search tìm kiếm trên từng "- line" riêng biệt → precision cao hơn khi query ngắn.

---

## 6. Tag-Based Filtering

### 6.1 JSONB Structure trong user_events.event_data

```json
{
  "event_tip": "- Discussed Q3 deadline\n- Mentioned feeling stressed",
  "event_tags": [
    {"tag": "emotion", "value": "stressed"},
    {"tag": "goal", "value": "meet_deadline"}
  ],
  "profile_delta": [...]
}
```

### 6.2 Tag Query SQL

```go
// usecase/filter_by_tags.go

func (r *EventRepo) FilterByTags(ctx context.Context, req FilterByTagsRequest) ([]Event, error) {
    query := `
        SELECT id, user_id, project_id, event_data, created_at
        FROM user_events
        WHERE user_id = $1 AND project_id = $2
        AND created_at > NOW() - ($3 * INTERVAL '1 day')
    `
    args := []any{req.UserID, req.ProjectID, req.TimeRangeDays}

    // has_event_tag: check tag key exists
    for _, tag := range req.HasEventTag {
        n := len(args) + 1
        query += fmt.Sprintf(` AND event_data->'event_tags' @> $%d::jsonb`, n)
        args = append(args, fmt.Sprintf(`[{"tag":"%s"}]`, tag))
    }

    // event_tag_equal: check tag=value pair
    for tag, value := range req.EventTagEqual {
        n := len(args) + 1
        query += fmt.Sprintf(` AND event_data->'event_tags' @> $%d::jsonb`, n)
        args = append(args, fmt.Sprintf(`[{"tag":"%s","value":"%s"}]`, tag, value))
    }

    query += ` ORDER BY created_at DESC LIMIT 100`
    return r.db.QueryContext(ctx, query, args...)
}
```

---

## 7. AppendEvent (Internal API)

```go
// usecase/append_event.go
// Called by memobase-engine after successful LLM flush

func (uc *AppendEventUseCase) Execute(ctx context.Context, req AppendEventRequest) (*AppendEventResult, error) {
    // 1. Store event với pre-computed embedding từ engine
    event, err := uc.eventRepo.Save(ctx, Event{
        UserID:    req.UserID,
        ProjectID: req.ProjectID,
        EventData: req.EventData,
        Embedding: req.Embedding,  // đã được engine tính
    })
    if err != nil {
        return nil, err
    }

    // 2. Store event gists (bulk insert)
    gists := parseGistsFromEventTip(req.EventData.EventTip)
    for i, gistContent := range gists {
        var gistEmbedding []float32
        if i < len(req.GistEmbeddings) {
            gistEmbedding = req.GistEmbeddings[i]
        }
        uc.gistRepo.Save(ctx, EventGist{
            EventID:   event.ID,
            UserID:    req.UserID,
            ProjectID: req.ProjectID,
            GistData:  GistData{GistContent: gistContent},
            Embedding: gistEmbedding,
        })
    }

    return &AppendEventResult{
        EventID:   event.ID,
        GistCount: len(gists),
    }, nil
}

// parseGistsFromEventTip: split "- line1\n- line2" → ["line1", "line2"]
func parseGistsFromEventTip(tip string) []string {
    var gists []string
    for _, line := range strings.Split(tip, "\n") {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "- ") {
            gists = append(gists, strings.TrimPrefix(line, "- "))
        }
    }
    return gists
}
```

---

## 8. gRPC API

```protobuf
syntax = "proto3";
package memobase.event.v1;

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

    // Internal (called by engine)
    rpc AppendEvent(AppendEventRequest) returns (AppendEventResponse);

    // Recent events for context service (no embedding)
    rpc GetRecentEventGists(GetRecentEventGistsRequest) returns (GetRecentEventGistsResponse);
}

message SearchEventGistsRequest {
    string user_id = 1;
    string project_id = 2;
    // Option A: text query (event service embeds)
    string query = 3;
    // Option B: pre-computed embedding (from context service)
    repeated float embedding = 4;  // [packed=true]
    int32 topk = 5;
    double similarity_threshold = 6;
    int32 time_range_in_days = 7;
}
```

**Lưu ý về SearchEventGists:** Context service có thể gửi pre-computed embedding (option B) để tránh embedding API call thêm lần nữa. Event service kiểm tra nếu `embedding` field populated thì dùng trực tiếp.

---

## 9. NATS Events Consumed

```go
// adapter/event/subscriber.go

// Subscribe user deletion → cascade cleanup
s.js.Subscribe("memobase.admin.user.deleted", func(msg *nats.Msg) {
    var p struct{ UserID, ProjectID string }
    json.Unmarshal(msg.Data, &p)
    // DELETE FROM user_events WHERE user_id=$1 AND project_id=$2 (cascade xóa gists)
    uc.deleteUserEvents.Execute(ctx, p.UserID, p.ProjectID)
    msg.Ack()
}, nats.Durable("memobase-event-user-deleted"))
```

---

## 10. Configuration

```yaml
event:
  server:
    grpc_port: 9044
    health_port: 9094

  embedding:
    provider: "openai"                   # MEMOBASE_EMBEDDING_PROVIDER
    model: "text-embedding-3-small"
    dimension: 1536                       # MEMOBASE_EMBEDDING_DIM
    enabled: true                         # MEMOBASE_ENABLE_EVENT_EMBEDDING

  search:
    default_topk: 10
    default_threshold: 0.2
    default_time_range_days: 21

  database:
    url: "${DATABASE_URL}"
    pool_size: 25
    max_overflow: 10

  nats:
    url: "${NATS_URL}"
    stream: "memobase"
```

---

## 11. Testing Strategy

### Unit Tests
- `TestSearchEventsUseCase_EmbeddingDisabled` → `ErrEmbeddingDisabled` (HTTP 501)
- `TestFilterByTags_HasEventTag` — JSONB query formed correctly
- `TestFilterByTags_EventTagEqual` — tag=value pair query
- `TestAppendEvent_GistParsing` — "- line1\n- line2" → 2 gists saved
- `TestStartupValidation_DimensionMismatch` → fail fast with error message

### Integration Tests
- `TestSearchE2E` — append event với embedding → search → similarity > threshold
- `TestGistSearchE2E` — gist search returns finer results than event search
- `TestCascadeDelete` — delete user → events + gists gone

### Performance Tests
- `BenchmarkHNSWSearch_1kEvents` — HNSW vs Sequential scan comparison
- `BenchmarkTagFilter_JSONBIndex` — tag query với GIN index

---

## 12. Rủi ro & Biện pháp

| Rủi ro | Mức độ | Biện pháp |
|---|---|---|
| HNSW index build time trên large dataset | Trung bình | Dùng `CREATE INDEX CONCURRENTLY` trong migration |
| pgvector không available | Thấp | Kiểm tra extension trong startup validation |
| Embedding dimension thay đổi sau khi có data | Cao | Fail-fast ở startup, yêu cầu explicit migration |
| Gist count tăng nhanh (nhiều gists/event) | Thấp | Max ~10 gists/event (event_tip thường 3-10 lines) |
| SearchEventGists gọi embedder 2 lần (context + event) | Trung bình | Context service gửi pre-computed embedding (option B) |
