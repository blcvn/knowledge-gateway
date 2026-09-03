# TASK-GR-011 — 9-Step Ingestion Pipeline

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-011 |
| **Wave** | 2 (Ingestion & Search) |
| **Component** | `services/graphiti-ingestion/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-001 §4 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-GR-009, TASK-GR-010 |
| **Estimated** | 6h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** graphiti-ingestion: 24 .go - full ingestion pipeline  
---

## Context

Implement `IngestEpisodeUseCase` — orchestrator 9-bước chính của toàn bộ graphiti pipeline. Phối hợp với `graphiti-knowledge` (LLM calls) và `graphiti-store` (Neo4j I/O) qua InProcess gRPC.

---

## Goal

- 9-step orchestrator: chunk → context → extract entities → resolve entities → extract edges → resolve edges → summarize nodes → persist → publish
- `AddTripletUseCase` — ingest raw fact_triple (no extraction needed)
- `ListEpisodesUseCase` — retrieve recent episodes
- `RemoveEpisodeUseCase` — delete episode + cascade MENTIONS edges
- gRPC handler cho `IngestionService`

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-ingestion/internal/usecase/ingest_episode.go` |
| CREATE | `services/graphiti-ingestion/internal/usecase/add_triplet.go` |
| CREATE | `services/graphiti-ingestion/internal/usecase/list_episodes.go` |
| CREATE | `services/graphiti-ingestion/internal/usecase/remove_episode.go` |
| CREATE | `services/graphiti-ingestion/internal/usecase/port/input.go` |
| CREATE | `services/graphiti-ingestion/internal/usecase/port/output.go` |
| CREATE | `services/graphiti-ingestion/internal/adapter/grpc/handler.go` |

---

## Implementation

### File 1: `services/graphiti-ingestion/internal/usecase/port/input.go`

```go
package port

import "github.com/vnp-memory/pkg/graph"

type IngestEpisodeReq struct {
    Name              string
    Body              string
    Source            graph.EpisodeType
    SourceDescription string
    ReferenceTime     *string  // ISO8601 or nil (defaults to now)
    GroupID           string
    SagaID            string   // optional saga association
    PrevEpisodeUUID   string   // optional for NEXT_EPISODE linking
}

type AddTripletReq struct {
    SourceEntity string
    Relation     string
    TargetEntity string
    Fact         string
    ValidAt      *string
    GroupID      string
}

type RemoveEpisodeReq struct {
    EpisodeUUID string
    GroupID     string
}

type ListEpisodesReq struct {
    GroupID string
    LastN   int
    Source  *graph.EpisodeType
    SagaID  string
}

type IngestStats struct {
    EntitiesExtracted int32
    EntitiesNew       int32
    EdgesExtracted    int32
    EdgesNew          int32
    ProcessingTimeMs  int64
}
```

### File 2: `services/graphiti-ingestion/internal/usecase/port/output.go`

```go
package port

import (
    "context"

    "github.com/vnp-memory/pkg/graph"
    storepb "github.com/vnp-memory/api/proto/graphiti/store/v1"
    knowledgepb "github.com/vnp-memory/api/proto/graphiti/knowledge/v1"
)

// StorePort — graphiti-store operations needed by ingestion service
type StorePort interface {
    // Episode/entity persistence
    SaveBulk(ctx context.Context, req storepb.SaveBulkRequest) (*storepb.SaveBulkResponse, error)
    // Episode retrieval for context
    RetrieveEpisodes(ctx context.Context, groupID string, lastN int) ([]*graph.EpisodicNode, error)
    // Entity candidates for resolution
    NodeSimilaritySearch(ctx context.Context, vector []float32, groupID string, limit int, minScore float64) ([]*graph.EntityNode, error)
    NodeFulltextSearch(ctx context.Context, query, groupID string, limit int) ([]*graph.EntityNode, error)
    // Edge candidates for resolution
    EdgeSimilaritySearch(ctx context.Context, vector []float32, srcUUID, tgtUUID, groupID string, limit int, minScore float64) ([]*graph.EntityEdge, error)
    // Delete episode
    DeleteEpisode(ctx context.Context, episodeUUID string) error
    // Saga
    GetOrCreateSaga(ctx context.Context, sagaID, groupID string) (*graph.SagaNode, error)
    GetLastEpisodeInSaga(ctx context.Context, sagaID, groupID string) (*graph.EpisodicNode, error)
}

// KnowledgePort — graphiti-knowledge operations needed by ingestion
type KnowledgePort interface {
    ExtractEntities(ctx context.Context, req knowledgepb.ExtractEntitiesRequest) (*knowledgepb.ExtractEntitiesResponse, error)
    ResolveEntity(ctx context.Context, req knowledgepb.ResolveEntityRequest) (*knowledgepb.ResolveEntityResponse, error)
    ExtractEdges(ctx context.Context, req knowledgepb.ExtractEdgesRequest) (*knowledgepb.ExtractEdgesResponse, error)
    ResolveEdge(ctx context.Context, req knowledgepb.ResolveEdgeRequest) (*knowledgepb.ResolveEdgeResponse, error)
    ExtractAttributes(ctx context.Context, req knowledgepb.ExtractAttributesRequest) (*knowledgepb.ExtractAttributesResponse, error)
    GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
    GetOntology(ctx context.Context, groupID string) (*knowledgepb.GetOntologyResponse, error)
}

// EventPublisher — NATS event publisher
type EventPublisher interface {
    Publish(ctx context.Context, subject string, data interface{}) error
}
```

### File 3: `services/graphiti-ingestion/internal/usecase/ingest_episode.go`

```go
package usecase

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-ingestion/internal/usecase/port"
)

// IngestEpisodeUseCase orchestrates the 9-step graphiti ingestion pipeline.
// All external I/O is done via port interfaces (store + knowledge gRPC).
type IngestEpisodeUseCase struct {
    store     port.StorePort
    knowledge port.KnowledgePort
    publisher port.EventPublisher
    chunker   *Chunker
}

func NewIngestEpisodeUseCase(
    store port.StorePort,
    knowledge port.KnowledgePort,
    publisher port.EventPublisher,
) *IngestEpisodeUseCase {
    return &IngestEpisodeUseCase{
        store:     store,
        knowledge: knowledge,
        publisher: publisher,
        chunker:   NewChunker(DefaultChunkerConfig),
    }
}

type IngestResult struct {
    EpisodeUUID string
    Stats       port.IngestStats
}

// Execute runs all 9 ingestion pipeline steps.
func (uc *IngestEpisodeUseCase) Execute(ctx context.Context, req port.IngestEpisodeReq) (*IngestResult, error) {
    start := time.Now()
    stats := port.IngestStats{}

    // Determine reference time
    referenceTime := time.Now()
    if req.ReferenceTime != nil {
        if t, err := time.Parse(time.RFC3339, *req.ReferenceTime); err == nil { referenceTime = t }
    }

    // ─── Step 1: Chunk content ────────────────────────────────────────────
    chunks := uc.chunker.Chunk(req.Body, req.Source)
    if len(chunks) == 0 { chunks = []string{req.Body} }

    // ─── Step 2: Retrieve previous episodes (context window) ─────────────
    prevEpisodes, _ := uc.store.RetrieveEpisodes(ctx, req.GroupID, 3)
    prevContents := make([]string, 0, len(prevEpisodes))
    for _, ep := range prevEpisodes { prevContents = append(prevContents, ep.Content) }

    // ─── Step 3: Get ontology for this group ─────────────────────────────
    var entityTypes, edgeTypes []byte  // serialized ontology types
    if ontologyResp, err := uc.knowledge.GetOntology(ctx, req.GroupID); err == nil && ontologyResp != nil {
        // Pass serialized ontology proto to knowledge service
        entityTypes = ontologyResp.EntityTypesJson
        edgeTypes   = ontologyResp.EdgeTypesJson
    }

    // ─── Step 4: Extract entities ─────────────────────────────────────────
    extractEntitiesResp, err := uc.knowledge.ExtractEntities(ctx, knowledgepb.ExtractEntitiesRequest{
        Chunks:         chunks,
        PrevEpisodes:   prevContents,
        Source:         string(req.Source),
        GroupId:        req.GroupID,
        EntityTypesJson: entityTypes,
    })
    if err != nil { return nil, fmt.Errorf("step4 extract entities: %w", err) }

    stats.EntitiesExtracted = int32(len(extractEntitiesResp.Entities))

    // ─── Step 5: Resolve entities (get/create EntityNode UUIDs) ──────────
    resolvedNodes := make(map[string]string)  // normalizedName → UUID
    var savedEntityNodes []graph.EntityNode
    var savedEntityNodeUUIDs []string

    for _, entity := range extractEntitiesResp.Entities {
        // Search for similar existing entities
        var emb []float32
        if len(entity.NameEmbedding) > 0 { emb = entity.NameEmbedding }

        candidates, _ := uc.store.NodeSimilaritySearch(ctx, emb, req.GroupID, 5, 0.8)
        resolveResp, err := uc.knowledge.ResolveEntity(ctx, knowledgepb.ResolveEntityRequest{
            EntityName:    entity.Name,
            EntityLabel:   entity.Label,
            EntitySummary: entity.Summary,
            NameEmbedding: emb,
            Candidates:    candidatesToProto(candidates),
        })

        entityUUID := ""
        if err == nil && resolveResp.Decision == "merge" {
            entityUUID = resolveResp.ExistingUuid
        } else {
            // New entity
            entityUUID = uuid.New().String()
            stats.EntitiesNew++
            node := graph.EntityNode{
                UUID:          entityUUID,
                Name:          entity.Name,
                Labels:        []string{entity.Label},
                Summary:       entity.Summary,
                NameEmbedding: emb,
                GroupID:       req.GroupID,
                CreatedAt:     time.Now(),
            }
            savedEntityNodes = append(savedEntityNodes, node)
        }
        resolvedNodes[normalizeEntityName(entity.Name)] = entityUUID
        savedEntityNodeUUIDs = append(savedEntityNodeUUIDs, entityUUID)
    }

    // ─── Step 6: Extract edges ────────────────────────────────────────────
    entityNames := make([]string, 0, len(resolvedNodes))
    for name := range resolvedNodes { entityNames = append(entityNames, name) }

    extractEdgesResp, err := uc.knowledge.ExtractEdges(ctx, knowledgepb.ExtractEdgesRequest{
        Chunks:        chunks,
        EntityNames:   entityNames,
        GroupId:       req.GroupID,
        EdgeTypesJson: edgeTypes,
        ReferenceTime: referenceTime.Format(time.RFC3339),
    })
    if err != nil { return nil, fmt.Errorf("step6 extract edges: %w", err) }

    stats.EdgesExtracted = int32(len(extractEdgesResp.Edges))

    // ─── Step 7: Resolve edges (detect duplicates/contradictions) ─────────
    var newEntityEdges []graph.EntityEdge
    var invalidatedEdgeIDs []string

    for _, rawEdge := range extractEdgesResp.Edges {
        srcUUID := resolvedNodes[normalizeEntityName(rawEdge.SourceEntityName)]
        tgtUUID := resolvedNodes[normalizeEntityName(rawEdge.TargetEntityName)]
        if srcUUID == "" || tgtUUID == "" { continue }

        newEdge := protoToEntityEdge(rawEdge, srcUUID, tgtUUID, req.GroupID, referenceTime)

        resolveResp, err := uc.knowledge.ResolveEdge(ctx, knowledgepb.ResolveEdgeRequest{
            NewFact:       newEdge.Fact,
            NewFactEmb:    newEdge.FactEmbedding,
            SrcNodeUuid:   srcUUID,
            TgtNodeUuid:   tgtUUID,
            GroupId:       req.GroupID,
            ReferenceTime: referenceTime.Format(time.RFC3339),
        })
        if err != nil { continue }

        switch resolveResp.Resolution {
        case "DUPLICATE":
            continue  // skip duplicate
        case "NEW", "CONTRADICTION", "UPDATE":
            newEntityEdges = append(newEntityEdges, newEdge)
            stats.EdgesNew++
            if resolveResp.Resolution == "CONTRADICTION" || resolveResp.Resolution == "UPDATE" {
                invalidatedEdgeIDs = append(invalidatedEdgeIDs, resolveResp.InvalidatedEdgeUuids...)
            }
        }
    }

    // ─── Step 8: Update entity summaries (attributes) ────────────────────
    // Build per-entity fact lists
    entityFacts := make(map[string][]string)
    for _, edge := range newEntityEdges {
        entityFacts[edge.SourceNodeUUID] = append(entityFacts[edge.SourceNodeUUID], edge.Fact)
        entityFacts[edge.TargetNodeUUID] = append(entityFacts[edge.TargetNodeUUID], edge.Fact)
    }
    // Update summaries via knowledge service (fire-and-forget for now)

    // ─── Step 9: Build and persist all objects atomically ────────────────
    episodeUUID := uuid.New().String()
    episode := graph.EpisodicNode{
        UUID:              episodeUUID,
        Name:              req.Name,
        Content:           req.Body,
        Source:            req.Source,
        SourceDescription: req.SourceDescription,
        ValidAt:           referenceTime,
        GroupID:           req.GroupID,
        CreatedAt:         time.Now(),
    }

    // Build MENTIONS edges
    episodicEdges := make([]graph.EpisodicEdge, 0, len(savedEntityNodeUUIDs))
    for _, entityUUID := range savedEntityNodeUUIDs {
        episodicEdges = append(episodicEdges, graph.EpisodicEdge{
            UUID:       uuid.New().String(),
            SourceUUID: episodeUUID,
            TargetUUID: entityUUID,
            GroupID:    req.GroupID,
            CreatedAt:  time.Now(),
        })
    }

    // Build saga associations if requested
    var sagaNode *graph.SagaNode
    var hasEpisodeEdges []graph.HasEpisodeEdge
    var nextEpisodeEdges []graph.NextEpisodeEdge
    if req.SagaID != "" {
        saga, err := uc.store.GetOrCreateSaga(ctx, req.SagaID, req.GroupID)
        if err == nil && saga != nil {
            sagaNode = saga
            hasEpisodeEdges = append(hasEpisodeEdges, graph.HasEpisodeEdge{
                UUID:       uuid.New().String(),
                SourceUUID: saga.UUID,
                TargetUUID: episodeUUID,
                GroupID:    req.GroupID,
                CreatedAt:  time.Now(),
            })
            // Link to previous episode if exists
            prevEp, _ := uc.store.GetLastEpisodeInSaga(ctx, req.SagaID, req.GroupID)
            if prevEp != nil {
                nextEpisodeEdges = append(nextEpisodeEdges, graph.NextEpisodeEdge{
                    UUID:       uuid.New().String(),
                    SourceUUID: prevEp.UUID,
                    TargetUUID: episodeUUID,
                    GroupID:    req.GroupID,
                    CreatedAt:  time.Now(),
                })
            }
        }
    }

    // Persist atomically
    if err := uc.store.SaveBulk(ctx, buildSaveBulkRequest(
        episode, savedEntityNodes, newEntityEdges,
        episodicEdges, sagaNode, hasEpisodeEdges, nextEpisodeEdges,
        invalidatedEdgeIDs, req.GroupID,
    )); err != nil {
        return nil, fmt.Errorf("step9 save bulk: %w", err)
    }

    // Publish event for downstream services (search cache invalidation, etc.)
    uc.publisher.Publish(ctx, "graphiti.episode.ingested", map[string]interface{}{
        "episode_uuid": episodeUUID,
        "group_id":     req.GroupID,
        "entity_count": len(savedEntityNodes),
        "edge_count":   len(newEntityEdges),
    })

    stats.ProcessingTimeMs = time.Since(start).Milliseconds()
    return &IngestResult{EpisodeUUID: episodeUUID, Stats: stats}, nil
}

func normalizeEntityName(name string) string {
    return strings.ToLower(strings.TrimSpace(name))
}
```

### File 4: `services/graphiti-ingestion/internal/usecase/add_triplet.go`

```go
package usecase

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-ingestion/internal/usecase/port"
)

// AddTripletUseCase ingests a structured (subject, predicate, object) triplet
// without requiring LLM extraction — the fact is already known.
type AddTripletUseCase struct {
    store     port.StorePort
    knowledge port.KnowledgePort
    publisher port.EventPublisher
}

func NewAddTripletUseCase(store port.StorePort, knowledge port.KnowledgePort, publisher port.EventPublisher) *AddTripletUseCase {
    return &AddTripletUseCase{store: store, knowledge: knowledge, publisher: publisher}
}

func (uc *AddTripletUseCase) Execute(ctx context.Context, req port.AddTripletReq) (*IngestResult, error) {
    // 1. Resolve/create source entity
    srcEmb, _ := uc.knowledge.GenerateEmbedding(ctx, req.SourceEntity)
    srcCandidates, _ := uc.store.NodeSimilaritySearch(ctx, srcEmb, req.GroupID, 5, 0.9)
    srcUUID := resolveOrCreate(req.SourceEntity, "Entity", srcEmb, srcCandidates, req.GroupID)

    // 2. Resolve/create target entity
    tgtEmb, _ := uc.knowledge.GenerateEmbedding(ctx, req.TargetEntity)
    tgtCandidates, _ := uc.store.NodeSimilaritySearch(ctx, tgtEmb, req.GroupID, 5, 0.9)
    tgtUUID := resolveOrCreate(req.TargetEntity, "Entity", tgtEmb, tgtCandidates, req.GroupID)

    // 3. Embed the fact
    factEmb, _ := uc.knowledge.GenerateEmbedding(ctx, req.Fact)

    // 4. Create entity edge
    validAt := time.Now()
    if req.ValidAt != nil {
        if t, err := time.Parse(time.RFC3339, *req.ValidAt); err == nil { validAt = t }
    }

    edge := graph.EntityEdge{
        UUID:           uuid.New().String(),
        SourceNodeUUID: srcUUID,
        TargetNodeUUID: tgtUUID,
        Name:           req.Relation,
        Fact:           req.Fact,
        FactEmbedding:  factEmb,
        ValidAt:        &validAt,
        GroupID:        req.GroupID,
        CreatedAt:      time.Now(),
    }

    // 5. Create episode record for the triplet
    episodeUUID := uuid.New().String()
    episode := graph.EpisodicNode{
        UUID:              episodeUUID,
        Name:              fmt.Sprintf("triplet_%s", uuid.New().String()[:8]),
        Content:           fmt.Sprintf("%s %s %s", req.SourceEntity, req.Relation, req.TargetEntity),
        Source:            graph.EpisodeTypeFactTriple,
        SourceDescription: "direct_triplet",
        ValidAt:           validAt,
        GroupID:           req.GroupID,
        CreatedAt:         time.Now(),
    }

    // 6. Persist
    srcNode := graph.EntityNode{UUID: srcUUID, Name: req.SourceEntity, Labels: []string{"Entity"}, GroupID: req.GroupID, NameEmbedding: srcEmb}
    tgtNode := graph.EntityNode{UUID: tgtUUID, Name: req.TargetEntity, Labels: []string{"Entity"}, GroupID: req.GroupID, NameEmbedding: tgtEmb}

    episodicEdges := []graph.EpisodicEdge{
        {UUID: uuid.New().String(), SourceUUID: episodeUUID, TargetUUID: srcUUID, GroupID: req.GroupID},
        {UUID: uuid.New().String(), SourceUUID: episodeUUID, TargetUUID: tgtUUID, GroupID: req.GroupID},
    }

    if err := uc.store.SaveBulk(ctx, buildSaveBulkRequest(
        episode, []graph.EntityNode{srcNode, tgtNode}, []graph.EntityEdge{edge},
        episodicEdges, nil, nil, nil, nil, req.GroupID,
    )); err != nil {
        return nil, fmt.Errorf("save triplet: %w", err)
    }

    uc.publisher.Publish(ctx, "graphiti.episode.ingested", map[string]interface{}{
        "episode_uuid": episodeUUID, "group_id": req.GroupID,
    })

    return &IngestResult{
        EpisodeUUID: episodeUUID,
        Stats:       port.IngestStats{EntitiesExtracted: 2, EntitiesNew: 2, EdgesExtracted: 1, EdgesNew: 1},
    }, nil
}
```

### File 5: `services/graphiti-ingestion/internal/adapter/grpc/handler.go`

```go
package grpc

import (
    "context"

    pb "github.com/vnp-memory/api/proto/graphiti/ingestion/v1"
    "github.com/vnp-memory/services/graphiti-ingestion/internal/usecase"
    "github.com/vnp-memory/services/graphiti-ingestion/internal/usecase/port"
)

type IngestionHandler struct {
    pb.UnimplementedIngestionServiceServer
    ingestEpisodeUC *usecase.IngestEpisodeUseCase
    addTripletUC    *usecase.AddTripletUseCase
    listEpisodesUC  *usecase.ListEpisodesUseCase
    removeEpisodeUC *usecase.RemoveEpisodeUseCase
}

func NewIngestionHandler(
    ingest *usecase.IngestEpisodeUseCase,
    triplet *usecase.AddTripletUseCase,
    list *usecase.ListEpisodesUseCase,
    remove *usecase.RemoveEpisodeUseCase,
) *IngestionHandler {
    return &IngestionHandler{
        ingestEpisodeUC: ingest,
        addTripletUC:    triplet,
        listEpisodesUC:  list,
        removeEpisodeUC: remove,
    }
}

func (h *IngestionHandler) IngestEpisode(ctx context.Context, req *pb.IngestEpisodeRequest) (*pb.IngestEpisodeResponse, error) {
    result, err := h.ingestEpisodeUC.Execute(ctx, port.IngestEpisodeReq{
        Name:              req.Name,
        Body:              req.Body,
        Source:            graph.EpisodeType(req.Source),
        SourceDescription: req.SourceDescription,
        GroupID:           extractGroupID(ctx),
        SagaID:            req.SagaId,
        PrevEpisodeUUID:   req.PrevEpisodeUuid,
    })
    if err != nil { return nil, err }

    return &pb.IngestEpisodeResponse{
        EpisodeUuid: result.EpisodeUUID,
        Stats: &pb.IngestStats{
            EntitiesExtracted: result.Stats.EntitiesExtracted,
            EntitiesNew:       result.Stats.EntitiesNew,
            EdgesExtracted:    result.Stats.EdgesExtracted,
            EdgesNew:          result.Stats.EdgesNew,
            ProcessingTimeMs:  result.Stats.ProcessingTimeMs,
        },
    }, nil
}

func (h *IngestionHandler) RemoveEpisode(ctx context.Context, req *pb.RemoveEpisodeRequest) (*pb.RemoveEpisodeResponse, error) {
    err := h.removeEpisodeUC.Execute(ctx, port.RemoveEpisodeReq{
        EpisodeUUID: req.EpisodeUuid,
        GroupID:     extractGroupID(ctx),
    })
    return &pb.RemoveEpisodeResponse{}, err
}

func (h *IngestionHandler) ListEpisodes(ctx context.Context, req *pb.ListEpisodesRequest) (*pb.ListEpisodesResponse, error) {
    lastN := int(req.LastN)
    if lastN == 0 { lastN = 10 }
    episodes, err := h.listEpisodesUC.Execute(ctx, port.ListEpisodesReq{
        GroupID: extractGroupID(ctx),
        LastN:   lastN,
        SagaID:  req.SagaId,
    })
    if err != nil { return nil, err }
    // Convert to proto...
    return &pb.ListEpisodesResponse{Episodes: episodesToProto(episodes)}, nil
}

func (h *IngestionHandler) AddTriplet(ctx context.Context, req *pb.AddTripletRequest) (*pb.AddTripletResponse, error) {
    result, err := h.addTripletUC.Execute(ctx, port.AddTripletReq{
        SourceEntity: req.SourceEntity,
        Relation:     req.Relation,
        TargetEntity: req.TargetEntity,
        Fact:         req.Fact,
        ValidAt:      nullableString(req.ValidAt),
        GroupID:      extractGroupID(ctx),
    })
    if err != nil { return nil, err }
    return &pb.AddTripletResponse{EpisodeUuid: result.EpisodeUUID}, nil
}

func extractGroupID(ctx context.Context) string {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok { return "default" }
    vals := md.Get("x-group-id")
    if len(vals) > 0 { return vals[0] }
    return "default"
}
```

---

## Verification

```bash
cd services/graphiti-ingestion
go build ./...
go vet ./...
```

**Key integration points to verify:**
1. Pipeline completes without error for simple text episode
2. `SagaID` set → saga node created + HAS_EPISODE edge saved
3. CONTRADICTION detected → `invalidatedEdgeIDs` non-empty
4. `graphiti.episode.ingested` NATS event published after each ingest
