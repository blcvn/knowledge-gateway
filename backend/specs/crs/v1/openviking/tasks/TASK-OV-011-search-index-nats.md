# TASK-OV-011 — `services/openviking-search` Index Sync, Hotness & gRPC Server

**Wave:** 4 (Search)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-OV-010  
**Ước tính:** 3 giờ  
**Solution tham chiếu:** [SOL-OV-003 §6, §7, §8](../solutions/SOL-OV-003-Search-Service.md)

**Trạng thái:** ✅ Implemented  
**Ghi chú:** ov-search NATS index subscription  
---

## Mục tiêu

Hoàn thiện `services/openviking-search/` với: NATS event-driven index sync, hotness update, Redis search cache, gRPC handler, debug replay, config, và main.go.

---

## Các file cần tạo

### 1. `internal/usecase/index_content.go` — Embed + Upsert

```go
type IndexContentUseCase struct {
    vectorStore vectordb.VectorStore
    embedder    embedder.EmbedderClient
    fsClient    port.FSClient
}

func (uc *IndexContentUseCase) Execute(ctx context.Context, req IndexRequest) error {
    // 1. Read content from FS (Level 0,1,2 depending on what was written)
    content, err := uc.fsClient.Read(ctx, req.URI, req.Level)
    if err != nil { return err }

    // 2. Embed content (isQuery=false for document embedding)
    embedResult, err := uc.embedder.Embed(ctx, string(content), false)
    if err != nil { return err }

    // 3. Extract abstract
    abstract := extractAbstract(content, req.Level)
    // Level 0: content IS the abstract
    // Level 1,2: first 200 chars

    // 4. Upsert to VectorDB
    parentURI := path.Dir(req.URI) + "/"
    return uc.vectorStore.UpsertContext(ctx, vectordb.ContextVector{
        URI:            req.URI,
        ParentURI:      parentURI,
        ContextType:    parseContextType(req.ContextType),
        Level:          req.Level,
        OwnerAccountID: req.AccountID,
        Abstract:       abstract,
        DenseVector:    embedResult.DenseVector,
        SparseVector:   embedResult.SparseVector,
        CreatedAt:      time.Now(),
        UpdatedAt:      time.Now(),
    })
}
```

### 2. `internal/usecase/remove_content.go`

```go
func (uc *RemoveContentUseCase) Execute(ctx context.Context, uri string) error {
    return uc.vectorStore.DeleteContext(ctx, uri)
}
```

### 3. `internal/usecase/update_hotness.go`

```go
func (uc *UpdateHotnessUseCase) Execute(ctx context.Context, uris []string) error {
    g, gCtx := errgroup.WithContext(ctx)
    for _, uri := range uris {
        uri := uri
        g.Go(func() error {
            return uc.vectorStore.UpdateActiveCount(gCtx, uri, +1)
        })
    }
    return g.Wait()
}
```

### 4. `internal/adapter/event/subscriber.go` — NATS Event Sync

```go
type Subscriber struct {
    indexUC   *usecase.IndexContentUseCase
    removeUC  *usecase.RemoveContentUseCase
    hotnessUC *usecase.UpdateHotnessUseCase
    cache     SearchCache
}

func (s *Subscriber) Start(ctx context.Context) error {
    natspkg.Subscribe(s.js, natspkg.SubjectContentWritten, "openviking-search-index", s.HandleContentWritten)
    natspkg.Subscribe(s.js, natspkg.SubjectContentDeleted, "openviking-search-delete", s.HandleContentDeleted)
    natspkg.Subscribe(s.js, natspkg.SubjectSessionCommitted, "openviking-search-hotness", s.HandleSessionCommitted)
    natspkg.Subscribe(s.js, natspkg.SubjectResourceIngested, "openviking-search-resource", s.HandleResourceIngested)
}

func (s *Subscriber) HandleContentWritten(msg *nats.Msg) {
    var payload natspkg.ContentWrittenPayload
    json.Unmarshal(msg.Data, &payload)
    
    err := s.indexUC.Execute(context.Background(), usecase.IndexRequest{
        URI:         payload.URI,
        AccountID:   payload.AccountID,
        ContextType: payload.ContextType,
        Level:       payload.Level,
    })
    if err != nil {
        slog.Warn("index content failed", "uri", payload.URI, "error", err)
        msg.Nak()
        return
    }
    
    // Invalidate cache for this account
    s.cache.InvalidateAccount(context.Background(), payload.AccountID)
    msg.Ack()
}

func (s *Subscriber) HandleContentDeleted(msg *nats.Msg) {
    var payload natspkg.ContentDeletedPayload
    json.Unmarshal(msg.Data, &payload)
    s.removeUC.Execute(context.Background(), payload.URI)
    msg.Ack()
}

func (s *Subscriber) HandleSessionCommitted(msg *nats.Msg) {
    var payload natspkg.SessionCommittedPayload
    json.Unmarshal(msg.Data, &payload)
    s.hotnessUC.Execute(context.Background(), payload.UsedURIs)
    msg.Ack()
}

func (s *Subscriber) HandleResourceIngested(msg *nats.Msg) {
    // Resource ingestion: FS service already emits content.written per file
    // This handler can trigger directory-level abstract indexing
    msg.Ack()
}
```

### 5. `internal/adapter/cache/redis/search_cache.go`

```go
type SearchCache struct {
    redis  *redis.Client
    ttl    time.Duration  // default: 120s
}

// Cache key: sha256(query|account_id|context_type|sorted_dirs)[:32]
func (c *SearchCache) Get(ctx context.Context, query *domain.TypedQuery) (*domain.QueryResult, bool)
func (c *SearchCache) Set(ctx context.Context, query *domain.TypedQuery, result *domain.QueryResult)

// Invalidate all cache entries for an account (called after content.written)
// Key pattern: "ov_search:{account_id}:*" — use Redis SCAN + DEL
func (c *SearchCache) InvalidateAccount(ctx context.Context, accountID string)
```

### 6. `internal/adapter/grpc/handler.go` — gRPC Handler

```go
type Handler struct {
    searchv1.UnimplementedSearchServiceServer
    findUC      *usecase.FindUseCase
    searchUC    *usecase.SearchUseCase
    indexUC     *usecase.IndexContentUseCase
    removeUC    *usecase.RemoveContentUseCase
    hotnessUC   *usecase.UpdateHotnessUseCase
}

func (h *Handler) Find(ctx context.Context, req *searchv1.FindRequest) (*searchv1.FindResponse, error) {
    // Map req → domain.TypedQuery
    // Call findUC.Execute
    // Map result → proto response
}

func (h *Handler) Search(ctx context.Context, req *searchv1.SearchRequest) (*searchv1.SearchResponse, error)
func (h *Handler) IndexContent(ctx context.Context, req *searchv1.IndexRequest) (*searchv1.IndexResponse, error)
func (h *Handler) RemoveContent(ctx context.Context, req *searchv1.RemoveRequest) (*searchv1.RemoveResponse, error)
func (h *Handler) UpdateHotness(ctx context.Context, req *searchv1.UpdateHotnessRequest) (*searchv1.UpdateHotnessResponse, error)
```

### 7. Config

```yaml
search:
  grpc:
    port: 9012
  health:
    port: 9092
  vectordb:
    provider: "qdrant"        # qdrant | memory (for tests)
    url: "http://qdrant:6333"
    collection_prefix: "openviking_"
  retrieval:
    global_search_topk: 10
    max_convergence_rounds: 3
    score_propagation_alpha: 0.7
    hotness_alpha: 0.1
    threshold: 0.0
  reranker:
    provider: "disabled"      # disabled | jina | cohere
    enabled: false
    threshold: 0.35
  embedding:
    provider: "bifrost"
    model: "text-embedding-3-small"
    dense_dim: 1536
    sparse_enabled: false
  cache:
    redis_url: "redis://redis:6379/2"
    ttl: 120s
    enabled: true
  nats:
    url: "nats://nats:4222"
    stream: "openviking"
    consumer_group: "openviking-search"
  clients:
    fs: "openviking-fs:9011"
    session: "openviking-session:9013"
```

---

## Unit Tests

```
TestIndexContent_EmbedAndUpsert        → content written → embed called → vectorStore.Upsert called
TestIndexContent_ExtractsAbstract_L0   → level=0 → abstract = full content
TestIndexContent_ExtractsAbstract_L2   → level=2 → abstract = first 200 chars
TestRemoveContent_DeletesFromStore     → RemoveContent → vectorStore.Delete called
TestUpdateHotness_IncrementsAll        → 5 URIs → UpdateActiveCount called 5 times
TestNATSHandleContentWritten_Indexes   → NATS event → index called
TestNATSHandleContentDeleted_Removes   → NATS event → remove called
TestNATSHandleSessionCommitted_Updates → NATS event → hotness updated for used URIs
TestSearchCache_HitReturnsResult       → cached → retriever not called
TestSearchCache_MissCallsRetriever     → not cached → retriever called → cached
TestSearchCache_InvalidateByAccount    → invalidate → next call misses cache
TestGRPCHandler_Find_MapsCorrectly    → proto request → domain query → proto response
TestGRPCHandler_Find_Empty            → no matches → empty response (not error)
```

### Integration Test

```
TestQdrantSearchE2E                   → upsert 10 vectors → search → correct topK
TestNATSIndexSyncE2E                  → write to FS → NATS event → search finds content
TestHotnessUpdateE2E                  → session committed → hotness → higher in search
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
buf generate services/openviking-search/
go build ./services/openviking-search/...
go test ./services/openviking-search/... -v -count=1
```

---

## Ghi chú triển khai

- Qdrant collection được tạo tự động khi service start (`CreateCollection` idempotent)
- Cache invalidation: dùng `SCAN` + `DEL` pattern `ov_search:{accountID}:*` (Redis 6+)
- Embedding batch không implement trong phase 1 (index mỗi file riêng)
- Debug replay feature (SearchIO): implement nếu còn thời gian, không block launch
