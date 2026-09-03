# Solution: SOL-001 — Episode Ingestion Pipeline (9-Step Orchestrator)

**CR ID:** CR-GR-001  
**Solution ID:** SOL-001  
**Priority:** Critical (Wave 2)  
**Architecture:** REBUILD `services/graphiti-ingestion/` — hiện tại chỉ có skeleton, cần thêm 9-step pipeline đầy đủ

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md §2.2`:
- `graphiti-ingestion` đã tồn tại trong monolith (service #4 trong Graphiti group).
- Giao tiếp qua `InProcessRegistry` bufconn, không có network hop.
- **Neo4j** là primary graph backend (đã configured).
- **NATS JetStream** embedded — stream `graphiti.*` đã có trong kiến trúc.
- `graphiti-knowledge` (port 9003) và `graphiti-store` (port 9004) là dependencies của service này.

**Vấn đề cốt lõi:** Service hiện tại thiếu toàn bộ orchestration logic — chỉ basic skeleton, không có 9-step pipeline, không có sequential group worker, không có chunking, không có saga.

---

## 2. Shared Package — `pkg/graph/`

Trước tiên cần tạo shared types dùng cho tất cả graphiti services:

```go
// pkg/graph/node.go

package graph

import "time"

type EpisodeType string
const (
    EpisodeTypeMessage    EpisodeType = "message"
    EpisodeTypeText       EpisodeType = "text"
    EpisodeTypeJSON       EpisodeType = "json"
    EpisodeTypeFactTriple EpisodeType = "fact_triple"
)

type EntityNode struct {
    UUID          string
    Name          string
    Labels        []string
    Summary       string
    Attributes    map[string]any
    NameEmbedding []float32
    GroupID       string
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type EpisodicNode struct {
    UUID              string
    Name              string
    Content           string
    Source            EpisodeType
    SourceDescription string
    ValidAt           time.Time
    EntityEdges       []string
    EpisodeMetadata   map[string]any
    GroupID           string
    CreatedAt         time.Time
}

type CommunityNode struct {
    UUID          string
    Name          string
    Summary       string
    NameEmbedding []float32
    GroupID       string
    CreatedAt     time.Time
}

type SagaNode struct {
    UUID             string
    Name             string
    GroupID          string
    Summary          string
    FirstEpisodeUUID string
    LastEpisodeUUID  string
    LastSummarizedAt *time.Time
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

```go
// pkg/graph/edge.go

type EntityEdge struct {
    UUID           string
    SourceNodeUUID string
    TargetNodeUUID string
    Name           string
    Fact           string
    FactEmbedding  []float32
    Episodes       []string
    ValidAt        *time.Time
    InvalidAt      *time.Time
    ExpiredAt      *time.Time
    GroupID        string
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type EpisodicEdge struct {
    UUID       string
    SourceUUID string
    TargetUUID string
    GroupID    string
    CreatedAt  time.Time
}

type CommunityEdge struct {
    UUID       string
    SourceUUID string
    TargetUUID string
    GroupID    string
    CreatedAt  time.Time
}

type HasEpisodeEdge struct {
    UUID       string
    SourceUUID string
    TargetUUID string
    GroupID    string
    CreatedAt  time.Time
}

type NextEpisodeEdge struct {
    UUID       string
    SourceUUID string
    TargetUUID string
    GroupID    string
    CreatedAt  time.Time
}
```

---

## 3. Domain Models — `services/graphiti-ingestion/internal/domain/`

### 3.1. Episode Domain

```go
// internal/domain/episode.go

type RawEpisode struct {
    Name              string
    Body              string
    Source            graph.EpisodeType
    SourceDescription string
    ReferenceTime     time.Time
    GroupID           string
    SagaID            string     // optional saga linking
    PrevEpisodeUUID   string     // optional, for NEXT_EPISODE edge
    EntityTypes       map[string]graph.EntityTypeSchema
    EdgeTypes         map[string]graph.EdgeTypeSchema
    Options           IngestionOptions
}

type IngestionOptions struct {
    StoreRawContent    bool
    ChunkTokenSize     int    // default: 3000
    ChunkOverlapTokens int    // default: 200
    ContextWindowSize  int    // default: 10
}

type IngestResult struct {
    EpisodeUUID string
    Stats       PipelineStats
}

type PipelineStats struct {
    EntitiesExtracted  int
    EntitiesResolved   int
    EntitiesNew        int
    EdgesExtracted     int
    EdgesResolved      int
    EdgesNew           int
    CommunitiesUpdated int
    ProcessingTimeMs   int64
    TokenUsage         TokenUsage
}

type TokenUsage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}
```

### 3.2. Chunk Domain

```go
// internal/domain/chunk.go

type ContentChunk struct {
    Text   string
    Index  int    // chunk position in original content
    Tokens int    // estimated token count
}

type ChunkConfig struct {
    TokenSize        int     // default: 3000
    OverlapTokens    int     // default: 200
    MinTokens        int     // default: 1000 (below = single chunk)
    DensityThreshold float64 // default: 0.15
}
```

---

## 4. 9-Step Pipeline — `internal/usecase/ingest_episode.go`

```go
// services/graphiti-ingestion/internal/usecase/ingest_episode.go

package usecase

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-ingestion/internal/domain"
    "github.com/vnp-memory/services/graphiti-ingestion/internal/usecase/port"
)

type IngestEpisodeUseCase struct {
    chunker       *ChunkContentUseCase
    storePort     port.StorePort      // → graphiti-store bufconn
    knowledgePort port.KnowledgePort  // → graphiti-knowledge bufconn
    sagaManager   *SagaManager
    publisher     port.EventPublisher
    config        domain.PipelineConfig
}

func (uc *IngestEpisodeUseCase) Execute(ctx context.Context, req domain.RawEpisode) (*domain.IngestResult, error) {
    start := time.Now()
    stats := &domain.PipelineStats{}
    episodeUUID := uuid.New().String()

    // ─── Step 1: Content Chunking (local, no I/O) ───────────────────────
    chunks, err := uc.chunker.Chunk(req.Body, domain.ChunkConfig{
        TokenSize:     req.Options.ChunkTokenSize,
        OverlapTokens: req.Options.ChunkOverlapTokens,
        MinTokens:     1000,
    })
    if err != nil { return nil, fmt.Errorf("step 1 chunk: %w", err) }

    // ─── Step 2: Retrieve Context (→ Store) ─────────────────────────────
    prevEpisodes, err := uc.storePort.RetrieveEpisodes(ctx, port.RetrieveEpisodesReq{
        GroupID:  req.GroupID,
        LastN:    req.Options.ContextWindowSize,
    })
    if err != nil { return nil, fmt.Errorf("step 2 context: %w", err) }

    // ─── Step 3: Extract Entities (→ Knowledge) ─────────────────────────
    extractedEntities, tokenUsage, err := uc.knowledgePort.ExtractEntities(ctx, port.ExtractEntitiesReq{
        Chunks:       chunks,
        PrevEpisodes: prevEpisodes,
        EntityTypes:  req.EntityTypes,
        Source:       req.Source,
    })
    if err != nil { return nil, fmt.Errorf("step 3 extract entities: %w", err) }
    stats.EntitiesExtracted = len(extractedEntities)
    stats.TokenUsage.Add(tokenUsage)

    // ─── Step 4: Resolve Entities (→ Knowledge + Store) ─────────────────
    resolvedNodes, newEntities, resolveTokens, err := uc.resolveEntities(ctx, extractedEntities, req.GroupID)
    if err != nil { return nil, fmt.Errorf("step 4 resolve entities: %w", err) }
    stats.EntitiesResolved = len(resolvedNodes)
    stats.EntitiesNew = len(newEntities)
    stats.TokenUsage.Add(resolveTokens)

    // ─── Steps 5 & 6: PARALLEL — Extract Edges + Extract Attributes ─────
    var (
        extractedEdges  []graph.EntityEdge
        updatedSummaries map[string]string
        edgeErr, attrErr error
        wg sync.WaitGroup
    )
    wg.Add(2)

    go func() {
        defer wg.Done()
        extractedEdges, _, edgeErr = uc.knowledgePort.ExtractEdges(ctx, port.ExtractEdgesReq{
            Chunks:        chunks,
            ResolvedNodes: resolvedNodes,
            EdgeTypes:     req.EdgeTypes,
            ReferenceTime: req.ReferenceTime,
        })
    }()

    go func() {
        defer wg.Done()
        updatedSummaries, attrErr = uc.knowledgePort.ExtractAttributes(ctx, port.ExtractAttributesReq{
            ResolvedNodes: resolvedNodes,
        })
    }()

    wg.Wait()
    if edgeErr != nil { return nil, fmt.Errorf("step 5 extract edges: %w", edgeErr) }
    if attrErr != nil { return nil, fmt.Errorf("step 6 extract attributes: %w", attrErr) }
    stats.EdgesExtracted = len(extractedEdges)

    // ─── Step 7: Resolve Edges (→ Knowledge + Store) ─────────────────────
    finalEdges, invalidated, resolveEdgeTokens, err := uc.resolveEdges(ctx, extractedEdges, resolvedNodes, req.GroupID, req.ReferenceTime)
    if err != nil { return nil, fmt.Errorf("step 7 resolve edges: %w", err) }
    stats.EdgesResolved = len(finalEdges)
    stats.EdgesNew = len(finalEdges) - len(invalidated)
    stats.TokenUsage.Add(resolveEdgeTokens)

    // Apply updated summaries to entity nodes
    for i := range resolvedNodes {
        if sum, ok := updatedSummaries[resolvedNodes[i].UUID]; ok {
            resolvedNodes[i].Summary = sum
        }
    }

    // ─── Step 8: Persist (→ Store, atomic SaveBulk) ─────────────────────
    episode := graph.EpisodicNode{
        UUID:              episodeUUID,
        Name:              req.Name,
        Content:           req.Body,
        Source:            req.Source,
        SourceDescription: req.SourceDescription,
        ValidAt:           req.ReferenceTime,
        GroupID:           req.GroupID,
        CreatedAt:         time.Now(),
    }

    // Build episodic edges (MENTIONS: episode → entity)
    episodicEdges := buildEpisodicEdges(episode.UUID, resolvedNodes, req.GroupID)

    // Handle saga linking
    var sagaNode *graph.SagaNode
    var hasEpEdges []graph.HasEpisodeEdge
    var nextEpEdges []graph.NextEpisodeEdge
    if req.SagaID != "" {
        sagaNode, hasEpEdges, nextEpEdges, err = uc.sagaManager.PrepareLinks(ctx, req.SagaID, episode.UUID, req.PrevEpisodeUUID, req.GroupID)
        if err != nil {
            // Non-fatal: saga linking failure doesn't abort ingestion
            sagaNode = nil
        }
    }

    err = uc.storePort.SaveBulk(ctx, port.SaveBulkReq{
        Episode:           episode,
        EntityNodes:       resolvedNodes,
        EntityEdges:       finalEdges,
        EpisodicEdges:     episodicEdges,
        SagaNode:          sagaNode,
        HasEpisodeEdges:   hasEpEdges,
        NextEpisodeEdges:  nextEpEdges,
        InvalidatedEdgeIDs: invalidated,
        GroupID:           req.GroupID,
    })
    if err != nil { return nil, fmt.Errorf("step 8 persist: %w", err) }

    // ─── Step 9: Update Community (→ Knowledge + Store) ──────────────────
    affectedUUIDs := extractNodeUUIDs(resolvedNodes)
    go uc.updateCommunity(context.Background(), affectedUUIDs, req.GroupID)

    // ─── Publish NATS event ───────────────────────────────────────────────
    stats.ProcessingTimeMs = time.Since(start).Milliseconds()
    uc.publisher.Publish(ctx, "graphiti.episode.ingested", map[string]any{
        "episode_uuid": episodeUUID,
        "group_id":     req.GroupID,
        "stats":        stats,
    })

    return &domain.IngestResult{EpisodeUUID: episodeUUID, Stats: *stats}, nil
}

// resolveEntities — Phase 1 (deterministic) + Phase 2 (LLM)
func (uc *IngestEpisodeUseCase) resolveEntities(ctx context.Context,
    extracted []port.ExtractedEntity, groupID string) (
    resolved []graph.EntityNode, newEntities []graph.EntityNode, tokenUsage domain.TokenUsage, err error) {

    for _, entity := range extracted {
        // Phase 1: Embed entity name + search similar nodes in Store
        candidates, _ := uc.storePort.NodeSimilaritySearch(ctx, port.NodeSimilaritySearchReq{
            Vector:  entity.NameEmbedding,
            GroupID: groupID,
            Limit:   15,
            MinScore: 0.6,
        })
        candidates2, _ := uc.storePort.NodeFulltextSearch(ctx, entity.Name, groupID, 5)
        candidates = mergeUnique(candidates, candidates2)

        // Deterministic fast path: exact match or high cosine (>0.95)
        if existing := findExactMatch(entity, candidates); existing != nil {
            resolved = append(resolved, *existing)
            continue
        }

        // Phase 2: LLM disambiguation
        resolution, tokens, err2 := uc.knowledgePort.ResolveEntity(ctx, port.ResolveEntityReq{
            Entity:     entity,
            Candidates: candidates,
        })
        tokenUsage.Add(tokens)
        if err2 != nil { continue }

        if resolution.ExistingUUID != "" {
            // Merge with existing node
            node, _ := uc.storePort.GetEntityNode(ctx, resolution.ExistingUUID)
            if node != nil { resolved = append(resolved, *node); continue }
        }
        // New entity: create EntityNode with UUID + name embedding
        newNode := graph.EntityNode{
            UUID:          uuid.New().String(),
            Name:          entity.Name,
            Labels:        []string{entity.Label},
            Summary:       entity.Summary,
            NameEmbedding: entity.NameEmbedding,
            GroupID:       groupID,
            CreatedAt:     time.Now(),
        }
        resolved = append(resolved, newNode)
        newEntities = append(newEntities, newNode)
    }
    return
}

// resolveEdges — handles contradiction/duplicate/new/update decisions
func (uc *IngestEpisodeUseCase) resolveEdges(ctx context.Context,
    extracted []graph.EntityEdge, nodes []graph.EntityNode,
    groupID string, referenceTime time.Time) (
    finalEdges []graph.EntityEdge, invalidatedUUIDs []string, tokenUsage domain.TokenUsage, err error) {

    for _, edge := range extracted {
        // Get existing edges between same nodes
        existing, _ := uc.storePort.GetEdgesBetweenNodes(ctx, edge.SourceNodeUUID, edge.TargetNodeUUID)

        // Fast path: exact fact text match = DUPLICATE
        if findExactEdge(edge.Fact, existing) != nil {
            continue
        }

        // Search similar existing edges (cosine similarity)
        similar, _ := uc.storePort.EdgeSimilaritySearch(ctx, port.EdgeSimilaritySearchReq{
            Vector:       edge.FactEmbedding,
            SourceUUID:   edge.SourceNodeUUID,
            TargetUUID:   edge.TargetNodeUUID,
            GroupID:      groupID,
            MinScore:     0.5,
        })

        // LLM resolution decision
        decision, tokens, err2 := uc.knowledgePort.ResolveEdge(ctx, port.ResolveEdgeReq{
            NewEdge:       edge,
            ExistingEdges: similar,
            ReferenceTime: referenceTime,
        })
        tokenUsage.Add(tokens)
        if err2 != nil { finalEdges = append(finalEdges, edge); continue }

        switch decision.Resolution {
        case "DUPLICATE":
            continue  // skip, no action
        case "NEW":
            edge.UUID = uuid.New().String()
            edge.ValidAt = &referenceTime
            finalEdges = append(finalEdges, edge)
        case "CONTRADICTION", "UPDATE":
            // Invalidate old edge(s), create new
            for _, invalidUUID := range decision.InvalidatedEdgeUUIDs {
                invalidatedUUIDs = append(invalidatedUUIDs, invalidUUID)
            }
            edge.UUID = uuid.New().String()
            edge.ValidAt = &referenceTime
            finalEdges = append(finalEdges, edge)
        }
    }
    return
}

// updateCommunity — async post-step, non-blocking
func (uc *IngestEpisodeUseCase) updateCommunity(ctx context.Context, nodeUUIDs []string, groupID string) {
    uc.knowledgePort.UpdateCommunity(ctx, port.UpdateCommunityReq{
        EntityUUIDs: nodeUUIDs,
        GroupID:     groupID,
    })
}
```

---

## 5. Per-Group Sequential Worker — `internal/adapter/queue/`

```go
// services/graphiti-ingestion/internal/adapter/queue/worker.go

package queue

import (
    "context"
    "sync"
    "time"
    "fmt"
)

const (
    DefaultMaxGroups    = 100
    DefaultQueuePerGroup = 50
    WorkerIdleTimeout   = 5 * time.Minute
)

type Job struct {
    Req        any  // domain.RawEpisode or domain.BulkEpisode
    ResultChan chan JobResult
    Ctx        context.Context
}

type JobResult struct {
    Result any
    Err    error
}

type GroupWorkerPool struct {
    mu        sync.RWMutex
    workers   map[string]*groupWorker   // key: group_id
    maxGroups int
    maxQueue  int
    processor JobProcessor
}

type groupWorker struct {
    queue  chan Job
    cancel context.CancelFunc
    lastJobAt time.Time
}

type JobProcessor interface {
    Process(ctx context.Context, job Job) JobResult
}

func NewGroupWorkerPool(processor JobProcessor, maxGroups, maxQueue int) *GroupWorkerPool {
    p := &GroupWorkerPool{
        workers:   make(map[string]*groupWorker),
        maxGroups: maxGroups,
        maxQueue:  maxQueue,
        processor: processor,
    }
    go p.cleanupIdleWorkers()
    return p
}

// Enqueue adds a job to the group's sequential queue.
// Returns 429-style error if queue is full.
func (p *GroupWorkerPool) Enqueue(groupID string, job Job) error {
    p.mu.Lock()
    w, ok := p.workers[groupID]
    if !ok {
        if len(p.workers) >= p.maxGroups {
            p.mu.Unlock()
            return fmt.Errorf("max groups reached (%d), try again later", p.maxGroups)
        }
        w = p.startWorker(groupID)
    }
    p.mu.Unlock()

    select {
    case w.queue <- job:
        return nil
    default:
        return fmt.Errorf("queue full for group %s (max %d), backpressure: 429", groupID, p.maxQueue)
    }
}

func (p *GroupWorkerPool) startWorker(groupID string) *groupWorker {
    ctx, cancel := context.WithCancel(context.Background())
    w := &groupWorker{
        queue:  make(chan Job, p.maxQueue),
        cancel: cancel,
    }
    p.workers[groupID] = w

    go func() {
        defer func() {
            p.mu.Lock()
            delete(p.workers, groupID)
            p.mu.Unlock()
        }()
        for {
            select {
            case job, ok := <-w.queue:
                if !ok { return }
                result := p.processor.Process(job.Ctx, job)
                job.ResultChan <- result
                w.lastJobAt = time.Now()
            case <-ctx.Done():
                return
            case <-time.After(WorkerIdleTimeout):
                return  // idle cleanup
            }
        }
    }()
    return w
}

// cleanupIdleWorkers reclaims goroutines for groups that haven't had jobs
func (p *GroupWorkerPool) cleanupIdleWorkers() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        p.mu.Lock()
        for groupID, w := range p.workers {
            if time.Since(w.lastJobAt) > WorkerIdleTimeout {
                w.cancel()
                delete(p.workers, groupID)
            }
        }
        p.mu.Unlock()
    }
}
```

---

## 6. Content Chunking — `internal/usecase/chunk_content.go`

```go
// services/graphiti-ingestion/internal/usecase/chunk_content.go

package usecase

import (
    "strings"
    "github.com/vnp-memory/services/graphiti-ingestion/internal/domain"
)

type ChunkContentUseCase struct{}

// Chunk splits content into chunks using density-based algorithm
func (uc *ChunkContentUseCase) Chunk(content string, cfg domain.ChunkConfig) ([]domain.ContentChunk, error) {
    if cfg.TokenSize == 0    { cfg.TokenSize = 3000 }
    if cfg.OverlapTokens == 0 { cfg.OverlapTokens = 200 }
    if cfg.MinTokens == 0    { cfg.MinTokens = 1000 }

    estimated := estimateTokens(content)
    if estimated < cfg.MinTokens {
        return []domain.ContentChunk{{Text: content, Index: 0, Tokens: estimated}}, nil
    }

    // Density estimation: quick entity scan
    density := estimateDensity(content)
    chunkSize := cfg.TokenSize
    if density > 0.15 {
        // High-density content: smaller chunks for better entity isolation
        chunkSize = cfg.TokenSize * 2 / 3
    }

    return slidingWindowChunk(content, chunkSize, cfg.OverlapTokens), nil
}

// estimateTokens approximates token count (chars/4 heuristic)
func estimateTokens(text string) int { return len(text) / 4 }

// estimateDensity estimates entity/token ratio using simple NER heuristics
func estimateDensity(text string) float64 {
    // Quick heuristic: count capitalized words (proxy for named entities)
    words := strings.Fields(text)
    if len(words) == 0 { return 0 }
    capitalCount := 0
    for _, w := range words {
        if len(w) > 1 && w[0] >= 'A' && w[0] <= 'Z' { capitalCount++ }
    }
    return float64(capitalCount) / float64(len(words))
}

// slidingWindowChunk splits by token estimate with overlap
func slidingWindowChunk(text string, chunkTokens, overlapTokens int) []domain.ContentChunk {
    chunkChars   := chunkTokens * 4
    overlapChars := overlapTokens * 4
    var chunks []domain.ContentChunk
    start := 0
    idx := 0
    for start < len(text) {
        end := start + chunkChars
        if end > len(text) { end = len(text) }
        chunks = append(chunks, domain.ContentChunk{
            Text:   text[start:end],
            Index:  idx,
            Tokens: (end - start) / 4,
        })
        idx++
        next := end - overlapChars
        if next <= start { next = start + 1 }  // prevent infinite loop
        start = next
    }
    return chunks
}
```

---

## 7. Saga Manager — `internal/usecase/manage_saga.go`

```go
// services/graphiti-ingestion/internal/usecase/manage_saga.go

package usecase

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-ingestion/internal/usecase/port"
)

type SagaManager struct {
    storePort     port.StorePort
    knowledgePort port.KnowledgePort
}

// PrepareLinks returns saga node + edges to create when persisting episode
func (m *SagaManager) PrepareLinks(ctx context.Context, sagaID, episodeUUID, prevEpisodeUUID, groupID string) (
    *graph.SagaNode, []graph.HasEpisodeEdge, []graph.NextEpisodeEdge, error) {

    // Ensure saga exists
    saga, err := m.storePort.GetSagaNode(ctx, sagaID, groupID)
    if err != nil || saga == nil {
        // Create new saga
        saga = &graph.SagaNode{
            UUID:             sagaID,
            Name:             fmt.Sprintf("Saga %s", sagaID),
            GroupID:          groupID,
            FirstEpisodeUUID: episodeUUID,
            LastEpisodeUUID:  episodeUUID,
            CreatedAt:        time.Now(),
        }
    } else {
        saga.LastEpisodeUUID = episodeUUID
        saga.UpdatedAt = time.Now()
    }

    hasEpEdge := graph.HasEpisodeEdge{
        UUID:       uuid.New().String(),
        SourceUUID: saga.UUID,
        TargetUUID: episodeUUID,
        GroupID:    groupID,
        CreatedAt:  time.Now(),
    }

    var nextEpEdges []graph.NextEpisodeEdge
    if prevEpisodeUUID != "" {
        nextEpEdges = append(nextEpEdges, graph.NextEpisodeEdge{
            UUID:       uuid.New().String(),
            SourceUUID: prevEpisodeUUID,
            TargetUUID: episodeUUID,
            GroupID:    groupID,
            CreatedAt:  time.Now(),
        })
    }

    return saga, []graph.HasEpisodeEdge{hasEpEdge}, nextEpEdges, nil
}

// Summarize triggers incremental LLM summary of saga (only new episodes since last_summarized_at)
func (m *SagaManager) Summarize(ctx context.Context, sagaID, groupID string) error {
    saga, err := m.storePort.GetSagaNode(ctx, sagaID, groupID)
    if err != nil || saga == nil { return fmt.Errorf("saga %s not found", sagaID) }

    summary, err := m.knowledgePort.SummarizeSaga(ctx, port.SummarizeSagaReq{
        SagaID:          sagaID,
        GroupID:         groupID,
        LastSummarizedAt: saga.LastSummarizedAt,
    })
    if err != nil { return err }

    now := time.Now()
    saga.Summary = summary.Text
    saga.LastSummarizedAt = &now
    return m.storePort.SaveSagaNode(ctx, *saga)
}
```

---

## 8. gRPC Handler — `internal/adapter/grpc/handler.go`

```go
// services/graphiti-ingestion/internal/adapter/grpc/handler.go

func (h *IngestionHandler) IngestEpisode(ctx context.Context, req *pb.IngestEpisodeRequest) (*pb.IngestEpisodeResponse, error) {
    groupID := extractGroupID(ctx)  // from gRPC metadata "x-group-id"

    job := queue.Job{
        Req: domain.RawEpisode{
            Name:              req.Name,
            Body:              req.Body,
            Source:            graph.EpisodeType(req.Source),
            SourceDescription: req.SourceDescription,
            ReferenceTime:     req.ReferenceTime.AsTime(),
            GroupID:           groupID,
            SagaID:            req.SagaId,
            PrevEpisodeUUID:   req.PrevEpisodeUuid,
            Options: domain.IngestionOptions{
                ChunkTokenSize:     3000,
                ChunkOverlapTokens: 200,
                ContextWindowSize:  10,
            },
        },
        ResultChan: make(chan queue.JobResult, 1),
        Ctx:        ctx,
    }

    // Enqueue with backpressure
    if err := h.workerPool.Enqueue(groupID, job); err != nil {
        return nil, status.Errorf(codes.ResourceExhausted, "backpressure: %v", err)
    }

    // Synchronous wait (client awaits completion)
    select {
    case result := <-job.ResultChan:
        if result.Err != nil { return nil, status.Errorf(codes.Internal, "%v", result.Err) }
        res := result.Result.(*domain.IngestResult)
        return &pb.IngestEpisodeResponse{
            EpisodeUuid: res.EpisodeUUID,
            Stats:       domainStatsToPB(res.Stats),
        }, nil
    case <-ctx.Done():
        return nil, status.Errorf(codes.Canceled, "context canceled")
    }
}

func (h *IngestionHandler) AddTriplet(ctx context.Context, req *pb.AddTripletRequest) (*pb.AddTripletResponse, error) {
    result, err := h.addTripletUC.Execute(ctx, domain.TripletRequest{
        Source: req.Source,
        Edge:   req.Edge,
        Target: req.Target,
        GroupID: extractGroupID(ctx),
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "%v", err) }
    return &pb.AddTripletResponse{EpisodeUuid: result.EpisodeUUID}, nil
}
```

---

## 9. Protobuf — `api/proto/graphiti/ingestion/v1/ingestion.proto`

```protobuf
syntax = "proto3";
package graphiti.ingestion.v1;
import "google/protobuf/timestamp.proto";

service IngestionService {
    rpc IngestEpisode(IngestEpisodeRequest) returns (IngestEpisodeResponse);
    rpc IngestEpisodeBulk(IngestEpisodeBulkRequest) returns (IngestEpisodeBulkResponse);
    rpc IngestEpisodeStream(stream IngestEpisodeRequest) returns (stream IngestEpisodeResponse);
    rpc RemoveEpisode(RemoveEpisodeRequest) returns (RemoveEpisodeResponse);
    rpc ListEpisodes(ListEpisodesRequest) returns (ListEpisodesResponse);
    rpc GetEpisode(GetEpisodeRequest) returns (GetEpisodeResponse);
    rpc AddTriplet(AddTripletRequest) returns (AddTripletResponse);
    rpc CreateSaga(CreateSagaRequest) returns (CreateSagaResponse);
    rpc SummarizeSaga(SummarizeSagaRequest) returns (SummarizeSagaResponse);
    rpc GetSaga(GetSagaRequest) returns (GetSagaResponse);
    rpc GetPipelineStatus(GetPipelineStatusRequest) returns (GetPipelineStatusResponse);
}

message IngestEpisodeRequest {
    string name               = 1;
    string body               = 2;
    string source             = 3;  // message|text|json|fact_triple
    string source_description = 4;
    google.protobuf.Timestamp reference_time = 5;
    string saga_id            = 6;  // optional
    string prev_episode_uuid  = 7;  // optional
}

message IngestEpisodeResponse {
    string episode_uuid = 1;
    PipelineStats stats = 2;
}

message PipelineStats {
    int32 entities_extracted  = 1;
    int32 entities_resolved   = 2;
    int32 entities_new        = 3;
    int32 edges_extracted     = 4;
    int32 edges_resolved      = 5;
    int32 edges_new           = 6;
    int32 communities_updated = 7;
    int64 processing_time_ms  = 8;
    TokenUsage token_usage    = 9;
}

message TokenUsage {
    int64 prompt_tokens     = 1;
    int64 completion_tokens = 2;
    int64 total_tokens      = 3;
}

message AddTripletRequest {
    string source   = 1;  // entity name
    string edge     = 2;  // relation type
    string target   = 3;  // entity name
}

message AddTripletResponse {
    string episode_uuid = 1;
}
```

---

## 10. Bootstrap — `apps/memory/internal/bootstrap/graphiti.go`

```go
// Thêm IngestEpisodeUseCase và GroupWorkerPool vào graphiti-ingestion init

func InitGraphitiIngestion(reg *bus.InProcessRegistry, knowledgeConn, storeConn *bufconn.Listener,
    nats *nats.Conn) {

    knowledgeClient := port.NewKnowledgeGRPCClient(knowledgeConn)
    storeClient     := port.NewStoreGRPCClient(storeConn)
    publisher       := natevent.NewPublisher(nats, "graphiti")

    chunker    := &usecase.ChunkContentUseCase{}
    sagaMgr    := usecase.NewSagaManager(storeClient, knowledgeClient)
    ingestUC   := usecase.NewIngestEpisodeUseCase(chunker, storeClient, knowledgeClient, sagaMgr, publisher, cfg)
    addTriplet := usecase.NewAddTripletUseCase(storeClient, knowledgeClient)
    bulkUC     := usecase.NewIngestEpisodeBulkUseCase(knowledgeClient, storeClient, publisher)

    processor  := queue.NewIngestionProcessor(ingestUC, bulkUC)
    workerPool := queue.NewGroupWorkerPool(processor, cfg.MaxConcurrentGroups, cfg.QueueSizePerGroup)

    handler := grpchandler.NewIngestionHandler(workerPool, addTriplet, sagaMgr, ingestUC)
    pb.RegisterIngestionServiceServer(grpcServer, handler)
}
```

---

## 11. Files

### [NEW]

| File | Mô tả |
|------|-------|
| `pkg/graph/node.go` | EntityNode, EpisodicNode, CommunityNode, SagaNode |
| `pkg/graph/edge.go` | EntityEdge, EpisodicEdge, CommunityEdge, HasEpisodeEdge, NextEpisodeEdge |
| `services/graphiti-ingestion/internal/domain/episode.go` | RawEpisode, PipelineStats |
| `services/graphiti-ingestion/internal/domain/chunk.go` | ContentChunk, ChunkConfig |
| `services/graphiti-ingestion/internal/usecase/ingest_episode.go` | 9-step pipeline |
| `services/graphiti-ingestion/internal/usecase/chunk_content.go` | density-based chunking |
| `services/graphiti-ingestion/internal/usecase/manage_saga.go` | Saga CRUD + summarize |
| `services/graphiti-ingestion/internal/usecase/ingest_episode_bulk.go` | Parallel bulk |
| `services/graphiti-ingestion/internal/usecase/add_triplet.go` | Direct triplet insertion |
| `services/graphiti-ingestion/internal/adapter/queue/worker.go` | GroupWorkerPool |
| `services/graphiti-ingestion/internal/adapter/queue/queue.go` | Per-group channel |
| `api/proto/graphiti/ingestion/v1/ingestion.proto` | Full gRPC contract |

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `services/graphiti-ingestion/internal/adapter/grpc/handler.go` | Full handler implementation |
| `apps/memory/internal/bootstrap/graphiti.go` | Init IngestEpisodeUseCase + GroupWorkerPool |
| `gateway/internal/adapter/handler/router.go` | + graphiti episode/triplet/saga routes |

---

## 12. Acceptance Criteria Mapping

| AC từ CR-GR-001 | Covered by |
|----------------|-----------|
| POST /v1/graphiti/episodes → episode_uuid + stats.entities_new ≥ 1 | IngestEpisodeUseCase step 3 |
| "Alice joined engineering" → EntityNode "Alice" tồn tại | resolveEntities() + SaveBulk() |
| Contradicting episode → old edge invalid_at set, new edge created | resolveEdges() CONTRADICTION case |
| 2 episodes cùng group_id xử lý tuần tự | GroupWorkerPool per-group channel |
| Content > 1000 tokens → auto chunk with overlap | ChunkContentUseCase |
| Bulk 100 episodes → entities deduplicated | IngestEpisodeBulkUseCase |
| POST /v1/triplets → EntityEdge created | AddTripletUseCase |
| saga_id → SagaNode + HAS_EPISODE edge | SagaManager.PrepareLinks() |
| Saga summary after 3 episodes | SagaManager.Summarize() → knowledge.SummarizeSaga |
| Queue full (>50) → 429 | GroupWorkerPool.Enqueue() backpressure |
| DELETE /v1/graphiti/episodes/{uuid} → cascade delete | RemoveEpisodeUseCase |
