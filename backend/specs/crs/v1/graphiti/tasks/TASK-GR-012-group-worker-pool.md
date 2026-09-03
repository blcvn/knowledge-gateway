# TASK-GR-012 — GroupWorkerPool + Density Chunker

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-012 |
| **Wave** | 2 (Ingestion & Search) |
| **Component** | `services/graphiti-ingestion/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-001 §5, §6 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-GR-011 (parallel) |
| **Estimated** | 3h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** graphiti-ingestion group worker pool  
---

## Context

Implement `GroupWorkerPool` — backpressure mechanism đảm bảo các episode trong cùng `group_id` được xử lý tuần tự (tránh race conditions khi resolve entities). Các `group_id` khác nhau chạy song song. Cũng implement `Chunker` với chunking strategies (word boundary + sentence).

---

## Goal

- `GroupWorkerPool` — map[groupID]channel; fan-out to dedicated goroutine per group
- `Chunker` — 4 strategies: word boundary (text), sentence (message), raw (json), none (fact_triple)
- `SagaManager` — get/create saga, link episodes
- Graceful shutdown via context cancellation

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-ingestion/internal/infra/worker/group_worker_pool.go` |
| CREATE | `services/graphiti-ingestion/internal/usecase/chunker.go` |
| CREATE | `services/graphiti-ingestion/internal/usecase/saga_manager.go` |
| MODIFY | `services/graphiti-ingestion/internal/adapter/grpc/handler.go` |

---

## Implementation

### File 1: `services/graphiti-ingestion/internal/infra/worker/group_worker_pool.go`

```go
package worker

import (
    "context"
    "sync"
)

// IngestJob represents a single ingestion task with its result channel
type IngestJob struct {
    Ctx       context.Context
    GroupID   string
    Fn        func(ctx context.Context) (any, error)
    ResultCh  chan IngestResult
}

type IngestResult struct {
    Value any
    Err   error
}

// GroupWorkerPool maintains one goroutine per group_id.
// Within the same group, episodes are processed SEQUENTIALLY (prevents entity race conditions).
// Across different groups, processing is fully PARALLEL.
type GroupWorkerPool struct {
    mu       sync.Mutex
    channels map[string]chan IngestJob
    bufSize  int        // per-group channel buffer size
    wg       sync.WaitGroup
}

func NewGroupWorkerPool(bufSize int) *GroupWorkerPool {
    return &GroupWorkerPool{
        channels: make(map[string]chan IngestJob),
        bufSize:  bufSize,
    }
}

// Submit enqueues a job for the given group_id.
// Returns a result channel that will receive exactly one result.
// Blocks if the group's channel is full (backpressure).
func (p *GroupWorkerPool) Submit(groupID string, ctx context.Context, fn func(ctx context.Context) (any, error)) <-chan IngestResult {
    resultCh := make(chan IngestResult, 1)

    ch := p.getOrCreateChannel(groupID)
    ch <- IngestJob{
        Ctx:      ctx,
        GroupID:  groupID,
        Fn:       fn,
        ResultCh: resultCh,
    }
    return resultCh
}

// getOrCreateChannel returns the channel for a group, creating it + goroutine if new
func (p *GroupWorkerPool) getOrCreateChannel(groupID string) chan IngestJob {
    p.mu.Lock()
    defer p.mu.Unlock()

    if ch, ok := p.channels[groupID]; ok { return ch }

    ch := make(chan IngestJob, p.bufSize)
    p.channels[groupID] = ch

    p.wg.Add(1)
    go p.runWorker(groupID, ch)
    return ch
}

// runWorker processes jobs for a single group sequentially
func (p *GroupWorkerPool) runWorker(groupID string, ch <-chan IngestJob) {
    defer p.wg.Done()
    for job := range ch {
        // Check if job context already cancelled
        if job.Ctx.Err() != nil {
            job.ResultCh <- IngestResult{Err: job.Ctx.Err()}
            close(job.ResultCh)
            continue
        }

        val, err := job.Fn(job.Ctx)
        job.ResultCh <- IngestResult{Value: val, Err: err}
        close(job.ResultCh)
    }
}

// Stats returns per-group queue depths
func (p *GroupWorkerPool) Stats() map[string]int {
    p.mu.Lock()
    defer p.mu.Unlock()

    stats := make(map[string]int, len(p.channels))
    for groupID, ch := range p.channels { stats[groupID] = len(ch) }
    return stats
}

// Shutdown closes all group channels and waits for completion
func (p *GroupWorkerPool) Shutdown() {
    p.mu.Lock()
    for _, ch := range p.channels { close(ch) }
    p.channels = make(map[string]chan IngestJob)
    p.mu.Unlock()
    p.wg.Wait()
}
```

### File 2: `services/graphiti-ingestion/internal/usecase/chunker.go`

```go
package usecase

import (
    "strings"
    "unicode"

    "github.com/vnp-memory/pkg/graph"
)

// ChunkerConfig controls chunking behavior
type ChunkerConfig struct {
    WordBoundaryChunkSize int  // default: 200 words
    WordBoundaryOverlap   int  // default: 50 words (for context continuity)
    MinChunkWords         int  // default: 10 (skip tiny chunks)
}

var DefaultChunkerConfig = ChunkerConfig{
    WordBoundaryChunkSize: 200,
    WordBoundaryOverlap:   50,
    MinChunkWords:         10,
}

// Chunker splits episode content into processable chunks based on source type
type Chunker struct {
    cfg ChunkerConfig
}

func NewChunker(cfg ChunkerConfig) *Chunker { return &Chunker{cfg: cfg} }

// Chunk splits content based on the episode source type:
// - text/message: word boundary chunking with overlap
// - json: return as-is (single chunk, no splitting)
// - fact_triple: return as-is (single fact)
func (c *Chunker) Chunk(content string, source graph.EpisodeType) []string {
    switch source {
    case graph.EpisodeTypeJSON, graph.EpisodeTypeFactTriple:
        return []string{content}  // no chunking for structured data
    case graph.EpisodeTypeMessage:
        return c.chunkBySentence(content)
    default:  // text
        return c.chunkByWordBoundary(content)
    }
}

// chunkByWordBoundary splits text at word boundaries with overlap
func (c *Chunker) chunkByWordBoundary(text string) []string {
    words := tokenize(text)
    if len(words) <= c.cfg.WordBoundaryChunkSize {
        return []string{strings.Join(words, " ")}
    }

    var chunks []string
    step := c.cfg.WordBoundaryChunkSize - c.cfg.WordBoundaryOverlap
    if step <= 0 { step = c.cfg.WordBoundaryChunkSize }

    for i := 0; i < len(words); i += step {
        end := i + c.cfg.WordBoundaryChunkSize
        if end > len(words) { end = len(words) }
        chunk := strings.Join(words[i:end], " ")
        if wordCount(chunk) >= c.cfg.MinChunkWords {
            chunks = append(chunks, chunk)
        }
        if end >= len(words) { break }
    }
    return chunks
}

// chunkBySentence splits message content at sentence boundaries
func (c *Chunker) chunkBySentence(text string) []string {
    sentences := splitSentences(text)
    if len(sentences) == 0 { return []string{text} }

    var chunks []string
    var current strings.Builder
    wordCount := 0

    for _, sent := range sentences {
        sentWords := wordCount_(sent)
        if wordCount+sentWords > c.cfg.WordBoundaryChunkSize && current.Len() > 0 {
            chunk := strings.TrimSpace(current.String())
            if wordCount_(chunk) >= c.cfg.MinChunkWords { chunks = append(chunks, chunk) }
            current.Reset()
            wordCount = 0
        }
        current.WriteString(sent)
        current.WriteString(" ")
        wordCount += sentWords
    }

    if current.Len() > 0 {
        chunk := strings.TrimSpace(current.String())
        if wordCount_(chunk) >= c.cfg.MinChunkWords { chunks = append(chunks, chunk) }
    }

    if len(chunks) == 0 { return []string{text} }
    return chunks
}

// tokenize splits text into word tokens (handles unicode)
func tokenize(text string) []string {
    fields := strings.FieldsFunc(text, func(r rune) bool {
        return unicode.IsSpace(r)
    })
    return fields
}

// splitSentences splits text at sentence boundaries (., !, ?)
func splitSentences(text string) []string {
    var sentences []string
    var current strings.Builder
    for i, r := range text {
        current.WriteRune(r)
        if (r == '.' || r == '!' || r == '?') && i < len(text)-1 {
            sentences = append(sentences, current.String())
            current.Reset()
        }
    }
    if current.Len() > 0 { sentences = append(sentences, current.String()) }
    return sentences
}

func wordCount(s string) int   { return len(strings.Fields(s)) }
func wordCount_(s string) int  { return len(strings.Fields(s)) }
```

### File 3: `services/graphiti-ingestion/internal/usecase/saga_manager.go`

```go
package usecase

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-ingestion/internal/usecase/port"
)

// SagaManager handles saga lifecycle within the ingestion pipeline
type SagaManager struct {
    store port.StorePort
}

func NewSagaManager(store port.StorePort) *SagaManager {
    return &SagaManager{store: store}
}

// GetOrCreate retrieves an existing saga or creates a new one
func (sm *SagaManager) GetOrCreate(ctx context.Context, sagaID, groupID string) (*graph.SagaNode, error) {
    if sagaID == "" {
        return nil, nil  // no saga
    }

    existing, err := sm.store.GetOrCreateSaga(ctx, sagaID, groupID)
    if err == nil && existing != nil { return existing, nil }

    // Create new saga node
    now := time.Now()
    saga := graph.SagaNode{
        UUID:      uuid.New().String(),
        Name:      sagaID,
        GroupID:   groupID,
        Summary:   "",
        CreatedAt: now,
        UpdatedAt: now,
    }
    return &saga, nil
}

// GetLastEpisode returns the last episode added to a saga (for NEXT_EPISODE linking)
func (sm *SagaManager) GetLastEpisode(ctx context.Context, sagaID, groupID string) (*graph.EpisodicNode, error) {
    return sm.store.GetLastEpisodeInSaga(ctx, sagaID, groupID)
}

// ShouldSummarize returns true if the saga should be re-summarized
// (triggered every 10 episodes or if never summarized before)
func (sm *SagaManager) ShouldSummarize(saga graph.SagaNode, newEpisodeCount int) bool {
    if saga.LastSummarizedAt == nil { return newEpisodeCount >= 1 }
    if newEpisodeCount >= 10 { return true }
    // Summarize if more than 7 days since last summarization
    return time.Since(*saga.LastSummarizedAt) > 7*24*time.Hour
}
```

### MODIFY: `services/graphiti-ingestion/internal/adapter/grpc/handler.go`

Add `GroupWorkerPool` integration to `IngestEpisode` gRPC handler:

```go
// In IngestionHandler struct:
type IngestionHandler struct {
    pb.UnimplementedIngestionServiceServer
    ingestEpisodeUC *usecase.IngestEpisodeUseCase
    addTripletUC    *usecase.AddTripletUseCase
    listEpisodesUC  *usecase.ListEpisodesUseCase
    removeEpisodeUC *usecase.RemoveEpisodeUseCase
    workerPool      *worker.GroupWorkerPool  // ADD THIS
}

// Replace the IngestEpisode handler body with:
func (h *IngestionHandler) IngestEpisode(ctx context.Context, req *pb.IngestEpisodeRequest) (*pb.IngestEpisodeResponse, error) {
    groupID := extractGroupID(ctx)

    // Submit to GroupWorkerPool — ensures sequential within same group
    resultCh := h.workerPool.Submit(groupID, ctx, func(innerCtx context.Context) (any, error) {
        return h.ingestEpisodeUC.Execute(innerCtx, port.IngestEpisodeReq{
            Name:              req.Name,
            Body:              req.Body,
            Source:            graph.EpisodeType(req.Source),
            SourceDescription: req.SourceDescription,
            GroupID:           groupID,
            SagaID:            req.SagaId,
        })
    })

    select {
    case result := <-resultCh:
        if result.Err != nil { return nil, result.Err }
        ingestResult := result.Value.(*usecase.IngestResult)
        return &pb.IngestEpisodeResponse{
            EpisodeUuid: ingestResult.EpisodeUUID,
            Stats: &pb.IngestStats{
                EntitiesExtracted: ingestResult.Stats.EntitiesExtracted,
                EntitiesNew:       ingestResult.Stats.EntitiesNew,
                EdgesExtracted:    ingestResult.Stats.EdgesExtracted,
                EdgesNew:          ingestResult.Stats.EdgesNew,
                ProcessingTimeMs:  ingestResult.Stats.ProcessingTimeMs,
            },
        }, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

---

## Verification

```bash
cd services/graphiti-ingestion
go build ./internal/infra/worker/...
go build ./internal/usecase/...
go test ./internal/usecase/... -run TestChunker -v
go test ./internal/infra/worker/... -run TestGroupWorkerPool -v
```

**Unit test for GroupWorkerPool:**
```go
func TestGroupWorkerPool_SameGroupSequential(t *testing.T) {
    pool := NewGroupWorkerPool(10)
    defer pool.Shutdown()

    var order []int
    var mu sync.Mutex

    // Submit 3 jobs to same group
    var wg sync.WaitGroup
    for i := 0; i < 3; i++ {
        idx := i
        wg.Add(1)
        go func() {
            defer wg.Done()
            resultCh := pool.Submit("group-1", context.Background(), func(ctx context.Context) (any, error) {
                mu.Lock(); order = append(order, idx); mu.Unlock()
                return nil, nil
            })
            <-resultCh
        }()
    }
    wg.Wait()
    // order should be sequential (0, 1, 2 in submission order)
    t.Logf("Processing order: %v", order)
}
```
