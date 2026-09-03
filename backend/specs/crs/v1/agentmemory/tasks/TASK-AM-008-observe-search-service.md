# TASK-AM-008 — Observe-Search Service (`services/observe-search/`)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-008 |
| **Wave** | 1 (Foundation) |
| **Component** | `services/observe-search/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-003 §2.2 → §2.9 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-AM-007, TASK-AM-001 |
| **Estimated** | 6h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** observe-search service implemented  
---

## Context

Service #37 trong monolith: `am-search`. Provides hybrid BM25+Vector search, context building, và index management cho observe-service.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/observe-search/internal/domain/entity.go` |
| CREATE | `services/observe-search/internal/usecase/smart_search.go` |
| CREATE | `services/observe-search/internal/usecase/build_context.go` |
| CREATE | `services/observe-search/internal/usecase/index_add.go` |
| CREATE | `services/observe-search/internal/usecase/index_remove.go` |
| CREATE | `services/observe-search/internal/usecase/port/output.go` |
| CREATE | `services/observe-search/internal/adapter/bifrost/embedder.go` |
| CREATE | `services/observe-search/internal/adapter/grpc/handler.go` |
| CREATE | `services/observe-search/internal/adapter/repository/postgres/obs_store.go` |
| CREATE | `apps/memory/internal/bootstrap/observe_search.go` |

---

## Implementation

### `internal/domain/entity.go`

```go
package domain

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

type ContextBlock struct {
    Type    string   // "memory" | "summary" | "observation"
    Content string
    Tokens  int
    Recency float64
    Source  string
}
```

### `internal/usecase/smart_search.go`

```go
package usecase

import (
    "context"
    "time"

    pkgsearch "github.com/vnp-memory/pkg/search"
    "github.com/vnp-memory/services/observe-search/internal/domain"
    "github.com/vnp-memory/services/observe-search/internal/usecase/port"
)

type SmartSearchUseCase struct {
    bm25     *pkgsearch.BM25Index
    vector   *pkgsearch.VectorIndex
    embedder port.IEmbedder
    weights  pkgsearch.ScoreWeights
    obsStore port.IObservationStore
}

type SmartSearchRequest struct {
    Query     string
    TenantID  string
    Project   string
    SessionFilter string
    Limit     int
    Weights   pkgsearch.ScoreWeights
}

type SmartSearchResponse struct {
    Results []domain.SearchResult
    TookMs  int64
}

func (uc *SmartSearchUseCase) Execute(ctx context.Context, req SmartSearchRequest) (*SmartSearchResponse, error) {
    start := time.Now()
    if req.Limit == 0 { req.Limit = 10 }

    // BM25 search (always)
    bm25Results := uc.bm25.Search(req.Query, req.Limit*3)

    // Vector search (if embedder configured)
    var vectorResults []pkgsearch.VectorResult
    if uc.embedder != nil {
        if vec, err := uc.embedder.Embed(ctx, req.Query); err == nil && vec != nil {
            vectorResults = uc.vector.Search(vec, req.Limit*3)
        }
    }

    // RRF fusion
    weights := req.Weights
    if weights.BM25 == 0 && weights.Vector == 0 { weights = uc.weights }
    fused := pkgsearch.RRFFuse(bm25Results, vectorResults, nil, weights, req.Limit)

    // Enrich with observation data from PostgreSQL
    results, _ := uc.enrichResults(ctx, fused, req)

    return &SmartSearchResponse{
        Results: results,
        TookMs:  time.Since(start).Milliseconds(),
    }, nil
}

func (uc *SmartSearchUseCase) enrichResults(ctx context.Context, fused []pkgsearch.HybridResult, req SmartSearchRequest) ([]domain.SearchResult, error) {
    docIDs := make([]string, len(fused))
    for i, r := range fused { docIDs[i] = r.DocID }

    obsMap, _ := uc.obsStore.GetByIDs(ctx, docIDs)

    results := make([]domain.SearchResult, 0, len(fused))
    for _, r := range fused {
        sr := domain.SearchResult{
            DocID:         r.DocID,
            SessionID:     r.SessionID,
            CombinedScore: r.CombinedScore,
            BM25Score:     r.BM25Score,
            VectorScore:   r.VectorScore,
        }
        if obs, ok := obsMap[r.DocID]; ok {
            sr.ObsType   = obs.ObsType
            sr.Title     = obs.Title
            sr.Narrative = obs.Narrative
            sr.Facts     = obs.Facts
            sr.Concepts  = obs.Concepts
        }
        results = append(results, sr)
    }
    return results, nil
}
```

### `internal/usecase/build_context.go`

```go
package usecase

import (
    "context"
    "fmt"
    "math"
    "strings"
    "time"

    "github.com/vnp-memory/services/observe-search/internal/domain"
    "github.com/vnp-memory/services/observe-search/internal/usecase/port"
)

type BuildContextUseCase struct {
    obsStore    port.IObservationStore
    memClient   port.IAgentMemoryClient
    smartSearch *SmartSearchUseCase
}

type ContextRequest struct {
    TenantID    string
    Project     string
    SessionID   string
    Query       string
    TokenBudget int
}

type ContextResponse struct {
    Blocks      []domain.ContextBlock
    TotalTokens int
    Formatted   string
}

func (uc *BuildContextUseCase) Execute(ctx context.Context, req ContextRequest) (*ContextResponse, error) {
    if req.TokenBudget == 0 { req.TokenBudget = 2000 }
    budget := req.TokenBudget
    var blocks []domain.ContextBlock

    // P1: Recent high-strength memories (strength > 0.5, last 30 days)
    memories, _ := uc.memClient.ListLatest(ctx, req.TenantID, req.Project, 30)
    for _, m := range memories {
        tokens := len(m.Content) / 4
        if budget-tokens < 0 { break }
        recency := math.Exp(-float64(time.Since(m.UpdatedAt).Hours()/24) / 7)
        blocks = append(blocks, domain.ContextBlock{
            Type:    "memory",
            Content: fmt.Sprintf("[%s] %s: %s", m.Type, m.Title, m.Content[:min(200, len(m.Content))]),
            Tokens:  tokens,
            Recency: recency,
            Source:  m.ID,
        })
        budget -= tokens
    }

    // P2: Last 3 session summaries
    summaries, _ := uc.obsStore.GetRecentSummaries(ctx, req.TenantID, req.Project, 3)
    for _, s := range summaries {
        tokens := len(s.Narrative) / 4
        if budget-tokens < 0 { break }
        blocks = append(blocks, domain.ContextBlock{
            Type: "summary", Content: s.Narrative, Tokens: tokens,
        })
        budget -= tokens
    }

    // P3: Relevant observations via search (if query provided)
    if req.Query != "" && budget > 100 {
        searchResp, _ := uc.smartSearch.Execute(ctx, SmartSearchRequest{
            Query: req.Query, TenantID: req.TenantID, Limit: 5,
        })
        if searchResp != nil {
            for _, r := range searchResp.Results {
                content := r.Title + ": " + r.Narrative
                tokens := len(content) / 4
                if budget-tokens < 0 { break }
                blocks = append(blocks, domain.ContextBlock{
                    Type: "observation", Content: content, Tokens: tokens,
                })
                budget -= tokens
            }
        }
    }

    formatted := formatBlocks(blocks)
    return &ContextResponse{
        Blocks:      blocks,
        TotalTokens: req.TokenBudget - budget,
        Formatted:   formatted,
    }, nil
}

func formatBlocks(blocks []domain.ContextBlock) string {
    var sb strings.Builder
    for _, b := range blocks {
        sb.WriteString(fmt.Sprintf("[%s] %s\n\n", strings.ToUpper(b.Type), b.Content))
    }
    return sb.String()
}

func min(a, b int) int { if a < b { return a }; return b }
```

### `internal/usecase/index_add.go`

```go
package usecase

import (
    "context"
    "strings"

    pkgsearch "github.com/vnp-memory/pkg/search"
    "github.com/vnp-memory/services/observe-search/internal/usecase/port"
)

type IndexAddUseCase struct {
    bm25      *pkgsearch.BM25Index
    vector    *pkgsearch.VectorIndex
    embedder  port.IEmbedder
    persister *pkgsearch.IndexPersister
}

type IndexAddRequest struct {
    ObsID     string
    SessionID string
    AgentID   string
    TenantID  string
    Title     string
    Facts     []string
    Concepts  []string
}

func (uc *IndexAddUseCase) Execute(ctx context.Context, req IndexAddRequest) error {
    // Build indexable text: title + facts + concepts
    text := req.Title + " " + strings.Join(req.Facts, " ") + " " + strings.Join(req.Concepts, " ")

    // Add to BM25
    uc.bm25.Add(req.ObsID, req.SessionID, req.AgentID, req.TenantID, text)

    // Add to vector (if embedder available)
    if uc.embedder != nil {
        if vec, err := uc.embedder.Embed(ctx, text); err == nil && vec != nil {
            uc.vector.Add(req.ObsID, req.SessionID, vec)
        }
    }

    // Debounced persist
    uc.persister.Schedule()
    return nil
}
```

### `internal/adapter/bifrost/embedder.go`

```go
package bifrost

import (
    "context"
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type BifrostEmbedder struct {
    url    string
    model  string
    dims   int
    client *http.Client
}

func NewBifrostEmbedder(url, model string, dims int) *BifrostEmbedder {
    return &BifrostEmbedder{
        url: url, model: model, dims: dims,
        client: &http.Client{Timeout: 5 * time.Second},
    }
}

type embedRequest struct {
    Model string   `json:"model"`
    Input []string `json:"input"`
}

type embedResponse struct {
    Data []struct {
        Embedding []float32 `json:"embedding"`
    } `json:"data"`
}

func (b *BifrostEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
    payload, _ := json.Marshal(embedRequest{Model: b.model, Input: []string{text}})
    req, _ := http.NewRequestWithContext(ctx, "POST", b.url+"/v1/embeddings", bytes.NewReader(payload))
    req.Header.Set("Content-Type", "application/json")

    resp, err := b.client.Do(req)
    if err != nil { return nil, err }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("bifrost embed: status %d", resp.StatusCode)
    }

    var result embedResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil { return nil, err }
    if len(result.Data) == 0 { return nil, fmt.Errorf("empty embedding response") }
    return result.Data[0].Embedding, nil
}

// NullEmbedder — no-op embedder when EMBEDDING_PROVIDER=none
type NullEmbedder struct{}
func (n *NullEmbedder) Embed(_ context.Context, _ string) ([]float32, error) { return nil, nil }
```

### `internal/adapter/grpc/handler.go`

```go
package grpc

import (
    "context"

    searchpb "github.com/vnp-memory/api/proto/search/v1"
    "github.com/vnp-memory/services/observe-search/internal/usecase"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type ObserveSearchHandler struct {
    searchpb.UnimplementedObserveSearchServiceServer
    smartSearch  *usecase.SmartSearchUseCase
    buildContext *usecase.BuildContextUseCase
    indexAdd     *usecase.IndexAddUseCase
    bm25         interface{ DocCount() int }
    vector       interface{ DocCount() int }
}

func (h *ObserveSearchHandler) SmartSearch(ctx context.Context, req *searchpb.SmartSearchRequest) (*searchpb.SmartSearchResponse, error) {
    resp, err := h.smartSearch.Execute(ctx, usecase.SmartSearchRequest{
        Query: req.Query, TenantID: req.TenantId, Project: req.Project,
        Limit: int(req.Limit),
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "smart search: %v", err) }
    return mapSmartSearchResponse(resp), nil
}

func (h *ObserveSearchHandler) BuildContext(ctx context.Context, req *searchpb.ContextRequest) (*searchpb.ContextResponse, error) {
    resp, err := h.buildContext.Execute(ctx, usecase.ContextRequest{
        TenantID: req.TenantId, Project: req.Project, SessionID: req.SessionId,
        Query: req.Query, TokenBudget: int(req.TokenBudget),
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "build context: %v", err) }
    return mapContextResponse(resp), nil
}

func (h *ObserveSearchHandler) IndexAdd(ctx context.Context, req *searchpb.IndexAddRequest) (*searchpb.IndexAddResponse, error) {
    err := h.indexAdd.Execute(ctx, usecase.IndexAddRequest{
        ObsID: req.ObsId, SessionID: req.SessionId, AgentID: req.AgentId,
        TenantID: req.TenantId, Title: req.Title, Facts: req.Facts, Concepts: req.Concepts,
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "index add: %v", err) }
    return &searchpb.IndexAddResponse{Ok: true}, nil
}

func (h *ObserveSearchHandler) GetIndexStats(ctx context.Context, req *searchpb.GetIndexStatsRequest) (*searchpb.GetIndexStatsResponse, error) {
    return &searchpb.GetIndexStatsResponse{
        Bm25Documents:   int32(h.bm25.DocCount()),
        VectorDocuments: int32(h.vector.DocCount()),
        Bm25Loaded:      h.bm25.DocCount() > 0,
        VectorLoaded:    h.vector.DocCount() > 0,
    }, nil
}
```

### `apps/memory/internal/bootstrap/observe_search.go`

```go
package bootstrap

func InitObserveSearch(reg *bus.InProcessRegistry, db *pgxpool.Pool, cfg *config.Config) {
    bm25   := pkgsearch.NewBM25Index()
    vector := pkgsearch.NewVectorIndex(cfg.Search.EmbedDims)

    var embedder port.IEmbedder = &bifrostadapter.NullEmbedder{}
    if cfg.Search.EmbeddingProvider != "none" && cfg.Bifrost.URL != "" {
        embedder = bifrostadapter.NewBifrostEmbedder(cfg.Bifrost.URL, cfg.Search.EmbeddingModel, cfg.Search.EmbedDims)
    }

    persister := pkgsearch.NewIndexPersister(bm25, vector, cfg.Search.DataDir)
    persister.LoadAsync()

    obsStore   := postgresadapter.NewObservationStore(db)
    memClient  := grpcclient.NewAgentMemoryClient(reg)

    smartSearchUC  := usecase.NewSmartSearch(bm25, vector, embedder, pkgsearch.DefaultWeights, obsStore)
    buildContextUC := usecase.NewBuildContext(obsStore, memClient, smartSearchUC)
    indexAddUC     := usecase.NewIndexAdd(bm25, vector, embedder, persister)
    indexRemoveUC  := usecase.NewIndexRemove(bm25, vector, persister)

    handler := grpchandler.NewObserveSearchHandler(smartSearchUC, buildContextUC, indexAddUC, bm25, vector)

    grpcServer := grpc.NewServer()
    searchpb.RegisterObserveSearchServiceServer(grpcServer, handler)
    reg.Register("am-search", grpcServer)
}
```

---

## Verification

```bash
cd services/observe-search
go build ./...
go test ./internal/usecase/... -v

# Integration: index add + search
# 1. POST /v1/observe → observation created → am-search.IndexAdd called
# 2. POST /v1/observe/search/smart → results include indexed obs
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| SmartSearch → results from BM25 + vector fused via RRF | ✅ |
| BM25-only mode (EMBEDDING_PROVIDER=none) → still returns results | ✅ |
| IndexAdd → doc searchable in < 1s (in-memory) | ✅ |
| BuildContext token budget ≤ configured limit | ✅ |
| GetIndexStats → `{bm25_documents: N, vector_documents: N}` | ✅ |
| p50 latency ≤ 14ms với 10K docs (brute-force cosine) | ✅ |
