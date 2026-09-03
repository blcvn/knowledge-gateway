# TASK-MB-010 — `services/memobase-event` Event Timeline & pgvector Semantic Search

**Wave:** 3 (Read Path — song song với TASK-MB-009)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-MB-005 (pkg/adapters/embedder), TASK-MB-007 (engine tạo DB schema + events)  
**Ước tính:** 5 giờ  
**Solution tham chiếu:** [SOL-MB-004](../solutions/SOL-MB-004-Event-Timeline-Semantic-Search.md)  
**Port gRPC:** 9044

**Trạng thái:** ✅ Implemented  
**Ghi chú:** memobase-event: 4 .go - event domain + service  
---

## Mục tiêu

Tạo `services/memobase-event/` — event timeline + vector similarity search service. Event được tạo bởi `memobase-engine`; service này xử lý: `AppendEvent` (lưu event + gists từ engine), `SearchEventGists` (pgvector cosine similarity), `FilterEventsByTags` (JSONB query), `GetRecentEventGists` (fallback cho context service).

---

## Cấu trúc thư mục

```
services/memobase-event/
├── cmd/server/main.go
├── api/proto/memobase/event/v1/event.proto
├── internal/
│   ├── domain/
│   │   ├── event.go         # UserEvent, UserEventGist, EventTag, GistData
│   │   └── errors.go        # ErrEmbeddingDisabled, ErrEventNotFound
│   ├── usecase/
│   │   ├── append_event.go      # Called by engine (save event + gists)
│   │   ├── get_events.go        # List events (timeline)
│   │   ├── search_events.go     # Semantic search on events
│   │   ├── search_event_gists.go # Fine-grained gist search
│   │   ├── filter_by_tags.go    # JSONB tag filtering
│   │   ├── get_recent_gists.go  # Recent gists for context service
│   │   ├── delete_event.go
│   │   ├── delete_user_events.go # Triggered by NATS user.deleted
│   │   └── port/
│   │       ├── input.go
│   │       └── output.go        # EventRepository, GistRepository
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── repository/postgres/
│   │   │   ├── event_repo.go
│   │   │   └── gist_repo.go
│   │   └── event/
│   │       └── subscriber.go    # NATS: user.deleted → cleanup
│   └── infra/
│       ├── migrations/
│       │   └── 004_event_hnsw.up.sql   # HNSW indexes
│       └── startup/
│           └── validator.go            # Embedding dimension check
```

---

## 1. Database Migration (HNSW Indexes)

**File: `internal/infra/migrations/004_event_hnsw.up.sql`**

```sql
-- HNSW vector indexes for fast cosine similarity search
-- Chạy CONCURRENTLY để không lock table (safe for existing data)
-- Phụ thuộc: bảng user_events và user_event_gists đã tồn tại (tạo bởi engine migration 003)

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_events_embedding_hnsw
    ON user_events
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 200);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_event_gists_embedding_hnsw
    ON user_event_gists
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 200);

-- JSONB GIN index for tag filtering
CREATE INDEX IF NOT EXISTS idx_user_events_tags
    ON user_events USING gin(event_data jsonb_path_ops);

-- Time range query optimization
CREATE INDEX IF NOT EXISTS idx_user_events_user_time
    ON user_events(user_id, project_id, created_at DESC);
```

---

## 2. Domain Models

**File: `internal/domain/event.go`**

```go
type EventTag struct {
    Tag   string `json:"tag"`
    Value string `json:"value,omitempty"`
}

type EventData struct {
    EventTip     string     `json:"event_tip"`
    EventTags    []EventTag `json:"event_tags"`
    ProfileDelta []any      `json:"profile_delta"`
}

type UserEvent struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    ProjectID string
    EventData EventData
    Embedding []float32   // nil if embedding disabled
    CreatedAt time.Time
    UpdatedAt time.Time
}

type GistData struct {
    GistContent string `json:"gist_content"`
}

type UserEventGist struct {
    ID          uuid.UUID
    EventID     uuid.UUID
    UserID      uuid.UUID
    ProjectID   string
    GistData    GistData
    Embedding   []float32
    Similarity  float64     // populated during search
    CreatedAt   time.Time
}

// parseGistsFromEventTip: split "- line1\n- line2" → ["line1", "line2"]
func ParseGistsFromEventTip(tip string) []string {
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

## 3. Use Cases

### `internal/usecase/append_event.go`

```go
type AppendEventRequest struct {
    UserID         uuid.UUID
    ProjectID      string
    EventData      domain.EventData
    Embedding      []float32    // Pre-computed by engine
    GistEmbeddings [][]float32  // Per-gist embeddings (optional)
}

func (uc *AppendEventUseCase) Execute(ctx context.Context, req AppendEventRequest) (*AppendEventResult, error) {
    // 1. Save event (with pre-computed embedding from engine)
    event := &domain.UserEvent{
        ID:        uuid.New(),
        UserID:    req.UserID,
        ProjectID: req.ProjectID,
        EventData: req.EventData,
        Embedding: req.Embedding,
        CreatedAt: time.Now(),
    }
    saved, err := uc.eventRepo.Save(ctx, event)
    if err != nil { return nil, fmt.Errorf("save event: %w", err) }

    // 2. Parse and save gists (bulk insert)
    gistContents := domain.ParseGistsFromEventTip(req.EventData.EventTip)
    gists := make([]*domain.UserEventGist, len(gistContents))
    for i, content := range gistContents {
        var emb []float32
        if i < len(req.GistEmbeddings) { emb = req.GistEmbeddings[i] }
        gists[i] = &domain.UserEventGist{
            ID:        uuid.New(),
            EventID:   saved.ID,
            UserID:    req.UserID,
            ProjectID: req.ProjectID,
            GistData:  domain.GistData{GistContent: content},
            Embedding: emb,
            CreatedAt: time.Now(),
        }
    }
    if len(gists) > 0 {
        if err := uc.gistRepo.SaveBatch(ctx, gists); err != nil {
            slog.Warn("append_event: failed to save gists", "error", err, "event_id", saved.ID)
            // Non-fatal: event saved successfully
        }
    }

    return &AppendEventResult{EventID: saved.ID, GistCount: len(gists)}, nil
}
```

### `internal/usecase/search_event_gists.go`

```go
type SearchEventGistsRequest struct {
    UserID              uuid.UUID
    ProjectID           string
    Embedding           []float32  // Pre-computed (Option B: sent by context service)
    Query               string     // Text query (Option A: embed here)
    TopK                int        // default: 10
    SimilarityThreshold float64    // default: 0.2
    TimeRangeDays       int        // default: 21
}

func (uc *SearchEventGistsUseCase) Execute(ctx context.Context, req SearchEventGistsRequest) ([]*domain.UserEventGist, error) {
    var queryVec []float32

    if len(req.Embedding) > 0 {
        // Option B: use pre-computed embedding (faster path)
        queryVec = req.Embedding
    } else if req.Query != "" {
        // Option A: embed the query text
        if !uc.embedder.IsEnabled() {
            return nil, domain.ErrEmbeddingDisabled
        }
        var err error
        queryVec, err = uc.embedder.EmbedQuery(ctx, req.Query)
        if err != nil { return nil, fmt.Errorf("embed query: %w", err) }
    } else {
        return nil, fmt.Errorf("search_event_gists: must provide either embedding or query")
    }

    topK := req.TopK
    if topK <= 0 { topK = 10 }
    threshold := req.SimilarityThreshold
    if threshold <= 0 { threshold = 0.2 }
    timeRange := req.TimeRangeDays
    if timeRange <= 0 { timeRange = 21 }

    return uc.gistRepo.SearchByEmbedding(ctx, port.SearchByEmbeddingQuery{
        UserID:    req.UserID,
        ProjectID: req.ProjectID,
        Vector:    queryVec,
        Threshold: threshold,
        TimeRange: timeRange,
        Limit:     topK,
    })
}
```

### `internal/usecase/filter_by_tags.go`

```go
type FilterByTagsRequest struct {
    UserID        uuid.UUID
    ProjectID     string
    HasEventTag   []string          // tag key exists
    EventTagEqual map[string]string // tag=value pair
    TimeRangeDays int               // default: 21
}

func (uc *FilterByTagsUseCase) Execute(ctx context.Context, req FilterByTagsRequest) ([]*domain.UserEvent, error) {
    return uc.eventRepo.FilterByTags(ctx, req)
}
```

### `internal/usecase/get_recent_gists.go`

```go
// Fallback for context service when no embedding available
func (uc *GetRecentEventGistsUseCase) Execute(ctx context.Context, userID uuid.UUID, projectID string, limit int) ([]*domain.UserEventGist, error) {
    if limit <= 0 { limit = 5 }
    // SELECT g.* FROM user_event_gists g
    // JOIN user_events e ON g.event_id=e.id AND g.project_id=e.project_id
    // WHERE g.user_id=$1 AND g.project_id=$2
    // ORDER BY e.created_at DESC, g.created_at DESC
    // LIMIT $3
    return uc.gistRepo.GetRecent(ctx, userID, projectID, limit)
}
```

---

## 4. Repository Implementations

**File: `internal/adapter/repository/postgres/gist_repo.go`**

```go
func (r *GistRepository) SearchByEmbedding(ctx context.Context, q port.SearchByEmbeddingQuery) ([]*domain.UserEventGist, error) {
    // pgvector cosine similarity: 1 - (embedding <=> $vec)
    const query = `
        SELECT
            g.id, g.event_id, g.user_id, g.project_id, g.gist_data, g.created_at,
            1 - (g.embedding <=> $3) AS similarity
        FROM user_event_gists g
        JOIN user_events e ON g.event_id = e.id AND g.project_id = e.project_id
        WHERE
            g.user_id = $1
            AND g.project_id = $2
            AND 1 - (g.embedding <=> $3) > $4
            AND e.created_at > NOW() - ($5 * INTERVAL '1 day')
        ORDER BY similarity DESC
        LIMIT $6`

    vec := pgvector.NewVector(q.Vector)
    rows, err := r.db.Query(ctx, query, q.UserID, q.ProjectID, vec, q.Threshold, q.TimeRange, q.Limit)
    // Scan results including similarity score
}

func (r *GistRepository) SaveBatch(ctx context.Context, gists []*domain.UserEventGist) error {
    // pgx CopyFrom for efficient bulk insert
    _, err := r.db.CopyFrom(ctx,
        pgx.Identifier{"user_event_gists"},
        []string{"id", "event_id", "user_id", "project_id", "gist_data", "embedding", "created_at"},
        pgx.CopyFromSlice(len(gists), func(i int) ([]any, error) {
            g := gists[i]
            var emb *pgvector.Vector
            if g.Embedding != nil { v := pgvector.NewVector(g.Embedding); emb = &v }
            gistJSON, _ := json.Marshal(g.GistData)
            return []any{g.ID, g.EventID, g.UserID, g.ProjectID, gistJSON, emb, g.CreatedAt}, nil
        }),
    )
    return err
}
```

**File: `internal/adapter/repository/postgres/event_repo.go`**

```go
func (r *EventRepository) FilterByTags(ctx context.Context, req usecase.FilterByTagsRequest) ([]*domain.UserEvent, error) {
    // Dynamic query building with JSONB @> operator
    query := `
        SELECT id, user_id, project_id, event_data, created_at
        FROM user_events
        WHERE user_id = $1 AND project_id = $2
        AND created_at > NOW() - ($3 * INTERVAL '1 day')`
    args := []any{req.UserID, req.ProjectID, req.TimeRangeDays}

    for _, tag := range req.HasEventTag {
        n := len(args) + 1
        query += fmt.Sprintf(` AND event_data->'event_tags' @> $%d::jsonb`, n)
        args = append(args, fmt.Sprintf(`[{"tag":"%s"}]`, tag))
    }
    for tag, value := range req.EventTagEqual {
        n := len(args) + 1
        query += fmt.Sprintf(` AND event_data->'event_tags' @> $%d::jsonb`, n)
        args = append(args, fmt.Sprintf(`[{"tag":"%s","value":"%s"}]`, tag, value))
    }
    query += ` ORDER BY created_at DESC LIMIT 100`

    rows, err := r.db.Query(ctx, query, args...)
    // scan...
}
```

---

## 5. Startup Validation

**File: `internal/infra/startup/validator.go`**

```go
func ValidateEmbeddingDimension(db *pgxpool.Pool, configDim int) error {
    var dbDim int
    err := db.QueryRow(context.Background(), `
        SELECT vector_dims(embedding) FROM user_events
        WHERE embedding IS NOT NULL LIMIT 1
    `).Scan(&dbDim)

    if err == pgx.ErrNoRows { return nil }  // No data yet → OK
    if err != nil { return fmt.Errorf("startup: validate embedding dim: %w", err) }

    if dbDim != configDim {
        return fmt.Errorf("embedding dimension mismatch: config=%d, DB=%d. "+
            "Recreate embeddings or update config", configDim, dbDim)
    }
    return nil
}
```

---

## 6. gRPC Proto

**File: `api/proto/memobase/event/v1/event.proto`**

```protobuf
syntax = "proto3";
package memobase.event.v1;
option go_package = "vnp-memory/services/memobase-event/api/gen/event/v1;eventv1";

service EventService {
  // CRUD
  rpc GetEvents(GetEventsRequest) returns (GetEventsResponse);
  rpc DeleteEvent(DeleteEventRequest) returns (DeleteEventResponse);

  // Search
  rpc SearchEvents(SearchEventsRequest) returns (SearchEventsResponse);
  rpc SearchEventGists(SearchEventGistsRequest) returns (SearchEventGistsResponse);

  // Tag filtering
  rpc FilterEventsByTags(FilterByTagsRequest) returns (FilterByTagsResponse);

  // Internal (called by engine after LLM flush)
  rpc AppendEvent(AppendEventRequest) returns (AppendEventResponse);

  // For context service (no embedding required)
  rpc GetRecentEventGists(GetRecentEventGistsRequest) returns (GetRecentEventGistsResponse);
}

message SearchEventGistsRequest {
  string user_id    = 1;
  string project_id = 2;
  string query      = 3;             // Option A: text (service embeds)
  repeated float embedding = 4;      // Option B: pre-computed (packed=true)
  int32  topk                = 5;    // default: 10
  double similarity_threshold = 6;   // default: 0.2
  int32  time_range_in_days  = 7;   // default: 21
}

message AppendEventRequest {
  string user_id    = 1;
  string project_id = 2;
  string event_data = 3;            // JSON-encoded EventData
  repeated float event_embedding = 4;  // pre-computed (packed=true)
  repeated GistEmbedding gist_embeddings = 5;
}

message GistEmbedding {
  repeated float embedding = 1;  // packed=true
}

message AppendEventResponse {
  string event_id  = 1;
  int32  gist_count = 2;
}

message FilterByTagsRequest {
  string user_id      = 1;
  string project_id   = 2;
  repeated string has_event_tag = 3;      // tag key exists
  map<string, string> event_tag_equal = 4; // tag=value pair
  int32 time_range_in_days = 5;
}
```

---

## 7. NATS Subscriber

**File: `internal/adapter/event/subscriber.go`**

```go
func (s *Subscriber) Start(ctx context.Context) error {
    // User deleted → clean up all events (PostgreSQL CASCADE handles gists)
    _, err := s.js.Subscribe("memobase.admin.user.deleted",
        func(msg *nats.Msg) {
            var p struct{ UserID, ProjectID string }
            json.Unmarshal(msg.Data, &p)
            userID, _ := uuid.Parse(p.UserID)
            if err := s.deleteUserEventsUC.Execute(ctx, userID, p.ProjectID); err != nil {
                slog.Warn("event: delete user events failed", "user_id", p.UserID, "error", err)
                msg.Nak()
                return
            }
            msg.Ack()
        },
        nats.Durable("memobase-event-user-deleted"),
        nats.AckExplicit(),
    )
    return err
}
```

---

## Unit Tests

```
TestParseGistsFromEventTip_Valid            → "- item1\n- item2" → ["item1","item2"]
TestParseGistsFromEventTip_NoDash           → "plain text" → []
TestParseGistsFromEventTip_Mixed            → mix dash + non-dash → only dash items
TestParseGistsFromEventTip_Empty            → "" → []
TestAppendEventUseCase_SavesEvent           → eventRepo.Save called with correct fields
TestAppendEventUseCase_SavesGists           → 2 gists in tip → gistRepo.SaveBatch(2 gists)
TestAppendEventUseCase_GistSaveFails        → gist save fails → event saved, no error returned
TestAppendEventUseCase_PrecomputedEmb       → embedding field set in saved event
TestSearchEventGistsUseCase_OptionB         → pre-computed embedding → embedder NOT called
TestSearchEventGistsUseCase_OptionA         → text query → embedder.EmbedQuery called
TestSearchEventGistsUseCase_EmbeddingDisabled_OptionA → ErrEmbeddingDisabled
TestSearchEventGistsUseCase_NoQueryOrEmb    → error returned
TestSearchEventGistsUseCase_DefaultsApplied → topK=0 → 10, threshold=0 → 0.2
TestFilterByTagsUseCase_HasTag              → has_event_tag → JSONB @> query built
TestFilterByTagsUseCase_TagEqual            → tag=value → @> with {tag,value} filter
TestFilterByTagsUseCase_Combined            → has_tag + tag_equal → both filters applied
TestGetRecentGists_DefaultLimit             → limit=0 → 5 results
TestGistRepo_SearchByEmbedding_SQL          → verify SQL uses cosine similarity
TestGistRepo_SaveBatch_BulkInsert           → CopyFrom called with all gists
TestEventRepo_FilterByTags_DynamicSQL       → 2 tags → 2 @> conditions
TestStartupValidator_DimensionMatch         → config=1536, db=1536 → nil
TestStartupValidator_DimensionMismatch      → config=1536, db=1024 → error
TestStartupValidator_EmptyTable             → ErrNoRows → nil (no data yet)
TestNATSSubscriber_UserDeleted_DeletesCalled → NATS event → deleteUserEventsUC called
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
go get github.com/pgvector/pgvector-go@latest

buf generate services/memobase-event/
go build ./services/memobase-event/...
go test ./services/memobase-event/... -v -count=1 -race

# Integration test (requires PostgreSQL + pgvector)
go test ./services/memobase-event/... -tags integration -v -count=1
```

---

## Ghi chú triển khai

- **pgvector-go**: `pgvector.NewVector([]float32{...})` → PostgreSQL `vector` type
- **HNSW vs IVFFlat**: HNSW `CREATE INDEX CONCURRENTLY` không lock bảng — an toàn cho production
- **`<=>` operator**: pgvector cosine distance; `1 - (a <=> b)` = cosine similarity
- **Option B priority**: context service luôn gửi pre-computed embedding để tránh double embedding cost
- **CopyFrom**: pgx batch insert efficient hơn multiple INSERTs cho gists (thường 3-10 gists/event)
- **Cascade**: xóa event → tự động xóa gists (FK ON DELETE CASCADE trong engine migration)
