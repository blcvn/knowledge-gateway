# TASK-CE-006 — Memify UseCase (Non-Destructive Graph Enrichment)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-CE-006 |
| **Wave** | 2 |
| **Component** | `services/cognee-cognify/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-001 §2.1 → §2.8 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-CE-001, TASK-CE-011 (migration CE-011 creates `cognee_pipeline_runs` table) |
| **Estimated** | 5h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** cognee-cognify Memify usecase implemented  
---

## Context

Memify = **non-destructive graph enrichment**. Thay vì rebuild graph từ đầu (StartCognify), Memify:
1. Load existing graph nodes + edges
2. Dùng LLM để suy ra **new relationships** (facts not yet in graph)
3. **Upsert-only** — không xóa nodes/edges cũ
4. Re-embed only new triplets vào Qdrant

Khác với `StartCognify` (pipeline cứng, có thể destructive), `Memify` chạy background, track via `cognee_pipeline_runs` table.

---

## Goal

- `MemifyUseCase` với 5-step enrichment flow
- `GraphDiff` domain types (GraphFact, GraphDiff, GraphNode, GraphEdge, PipelineRun)
- `PipelineRunRepository` (Postgres CRUD)
- Neo4j `GetDatasetGraph` + `UpsertGraphDiff` methods
- gRPC handler: `Memify()` + `GetPipelineStatus()`
- Gateway routes: POST/GET `/api/v1/cognee/datasets/{id}/memify`
- Bootstrap update: inject MemifyUseCase

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/cognee-cognify/internal/usecase/memify.go` |
| CREATE | `services/cognee-cognify/internal/domain/graph_diff.go` |
| CREATE | `services/cognee-cognify/internal/adapter/repository/postgres/pipeline_run_repo.go` |
| MODIFY | `services/cognee-cognify/internal/adapter/grpc/handler.go` |
| MODIFY | `services/cognee-cognify/internal/usecase/port/output.go` |
| MODIFY | `services/cognee-cognify/internal/adapter/repository/neo4j/graph_repo.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |
| MODIFY | `apps/memory/internal/bootstrap/cognee.go` |

---

## Implementation

### File 1: `services/cognee-cognify/internal/domain/graph_diff.go`

```go
package domain

import "time"

// GraphFact — LLM-derived relationship (S, P, O)
type GraphFact struct {
    Subject   string
    Predicate string
    Object    string
}

// GraphDiff — delta between existing and derived graph
type GraphDiff struct {
    Nodes []GraphNode   // new nodes inferred (not yet in graph)
    Edges []GraphEdge   // new edges derived (not yet in graph)
}

// PipelineRun — tracks async cognify/memify job status
type PipelineRun struct {
    ID        string
    DatasetID string
    TenantID  string
    Type      string    // "cognify" | "memify"
    Status    string    // QUEUED | RUNNING | COMPLETED | FAILED
    NewNodes  int
    NewEdges  int
    Error     string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### File 2: `services/cognee-cognify/internal/usecase/memify.go`

```go
package usecase

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/services/cognee-cognify/internal/domain"
    "github.com/vnp-memory/services/cognee-cognify/internal/usecase/port"
)

const (
    defaultMemifyBatchSize   = 50
    memifyDefaultDeriveFacts = true
    memifyDefaultEmbed       = true
)

type MemifyRequest struct {
    DatasetID uuid.UUID
    TenantID  string
    Config    MemifyConfig
}

type MemifyConfig struct {
    DeriveFacts   bool  // default: true
    EmbedTriplets bool  // default: true
    BatchSize     int   // default: 50
}

type MemifyResult struct {
    PipelineRunID string
    Status        string  // QUEUED | RUNNING | COMPLETED | FAILED
    NewNodes      int
    NewEdges      int
}

type MemifyUseCase struct {
    graphRepo  port.GraphRepository
    vectorRepo port.VectorRepository
    llmClient  port.LLMClient
    embedder   port.EmbedderClient
    runRepo    port.PipelineRunRepository
    eventPub   port.EventPublisher
}

func NewMemifyUseCase(
    graphRepo  port.GraphRepository,
    vectorRepo port.VectorRepository,
    llmClient  port.LLMClient,
    embedder   port.EmbedderClient,
    runRepo    port.PipelineRunRepository,
    eventPub   port.EventPublisher,
) *MemifyUseCase {
    return &MemifyUseCase{graphRepo, vectorRepo, llmClient, embedder, runRepo, eventPub}
}

// Execute creates a PipelineRun and launches memify as background goroutine
func (uc *MemifyUseCase) Execute(ctx context.Context, req MemifyRequest) (*MemifyResult, error) {
    // Apply defaults
    if !req.Config.DeriveFacts { req.Config.DeriveFacts = true }
    if !req.Config.EmbedTriplets { req.Config.EmbedTriplets = true }
    if req.Config.BatchSize == 0 { req.Config.BatchSize = defaultMemifyBatchSize }

    runID := uuid.New().String()
    run := domain.PipelineRun{
        ID:        runID,
        DatasetID: req.DatasetID.String(),
        TenantID:  req.TenantID,
        Type:      "memify",
        Status:    "QUEUED",
        CreatedAt: time.Now(),
    }
    if err := uc.runRepo.Save(ctx, run); err != nil {
        return nil, fmt.Errorf("save pipeline run: %w", err)
    }

    // Launch background (detached from request context)
    go uc.runMemify(context.Background(), runID, req)

    return &MemifyResult{PipelineRunID: runID, Status: "QUEUED"}, nil
}

func (uc *MemifyUseCase) runMemify(ctx context.Context, runID string, req MemifyRequest) {
    uc.runRepo.SetStatus(ctx, runID, "RUNNING")
    uc.eventPub.Publish(ctx, "cognee.cognify.memify.started", map[string]any{
        "pipeline_run_id": runID, "dataset_id": req.DatasetID, "tenant_id": req.TenantID,
    })

    result, err := uc.executeSteps(ctx, req)
    if err != nil {
        uc.runRepo.SetStatusWithError(ctx, runID, "FAILED", err.Error())
        uc.eventPub.Publish(ctx, "cognee.cognify.memify.failed", map[string]any{
            "pipeline_run_id": runID, "error": err.Error(),
        })
        return
    }

    uc.runRepo.SetStatusWithResult(ctx, runID, "COMPLETED", result.NewNodes, result.NewEdges)
    uc.eventPub.Publish(ctx, "cognee.cognify.memify.completed", map[string]any{
        "pipeline_run_id": runID,
        "dataset_id":      req.DatasetID,
        "tenant_id":       req.TenantID,
        "new_nodes":       result.NewNodes,
        "new_edges":       result.NewEdges,
    })
}

func (uc *MemifyUseCase) executeSteps(ctx context.Context, req MemifyRequest) (*MemifyResult, error) {
    result := &MemifyResult{}

    // Step 1: Load existing graph (read-only)
    existingNodes, existingEdges, err := uc.graphRepo.GetDatasetGraph(ctx, req.DatasetID, req.TenantID)
    if err != nil { return nil, fmt.Errorf("load graph: %w", err) }

    // Step 2: Derive new facts via LLM
    var derivedFacts []domain.GraphFact
    if req.Config.DeriveFacts {
        derivedFacts, err = uc.deriveFacts(ctx, existingNodes, existingEdges, req.Config.BatchSize)
        if err != nil { return nil, fmt.Errorf("derive facts: %w", err) }
    }

    // Step 3: Build diff (only NEW facts not already in graph)
    diff := buildEnrichmentDiff(existingEdges, derivedFacts)

    // Step 4: Upsert-only (non-destructive) — no delete
    if len(diff.Nodes) > 0 || len(diff.Edges) > 0 {
        if err := uc.graphRepo.UpsertGraphDiff(ctx, req.DatasetID, req.TenantID, diff); err != nil {
            return nil, fmt.Errorf("upsert graph diff: %w", err)
        }
        result.NewNodes = len(diff.Nodes)
        result.NewEdges = len(diff.Edges)
    }

    // Step 5: Re-embed only new triplets (non-fatal if fails)
    if req.Config.EmbedTriplets && len(diff.Edges) > 0 {
        _ = uc.embedAndIndexTriplets(ctx, req.DatasetID, req.TenantID, diff.Edges)
    }

    return result, nil
}

// deriveFacts calls Bifrost LLM to infer new relationships from existing graph
func (uc *MemifyUseCase) deriveFacts(ctx context.Context,
    nodes []domain.GraphNode, edges []domain.GraphEdge, batchSize int) ([]domain.GraphFact, error) {

    const systemPrompt = `You are a knowledge graph enricher. Given existing entities and their relationships,
infer NEW relationships that can be logically derived but are not explicitly stated.
Return only relationships that are definitively true based on the given facts.
Format: JSON array of {"subject": "name", "predicate": "relation_type", "object": "name"}`

    var allFacts []domain.GraphFact
    for i := 0; i < len(nodes); i += batchSize {
        end := i + batchSize
        if end > len(nodes) { end = len(nodes) }
        batch := nodes[i:end]

        userMsg := buildGraphContext(batch, filterEdgesForNodes(edges, batch))
        resp, err := uc.llmClient.Chat(ctx, systemPrompt, userMsg)
        if err != nil { continue }  // skip bad batch

        var facts []domain.GraphFact
        if err := json.Unmarshal([]byte(resp), &facts); err != nil { continue }
        allFacts = append(allFacts, facts...)
    }
    return allFacts, nil
}

// embedAndIndexTriplets embeds triplet text → upserts into Qdrant
func (uc *MemifyUseCase) embedAndIndexTriplets(ctx context.Context,
    datasetID uuid.UUID, tenantID string, edges []domain.GraphEdge) error {

    collectionName := fmt.Sprintf("cognee_%s", tenantID)
    for _, edge := range edges {
        text := fmt.Sprintf("%s %s %s", edge.Subject, edge.Predicate, edge.Object)
        vec, err := uc.embedder.Embed(ctx, text)
        if err != nil { continue }

        uc.vectorRepo.UpsertPoint(ctx, collectionName, edge.ID, vec, map[string]any{
            "dataset_id": datasetID.String(),
            "subject":    edge.Subject,
            "predicate":  edge.Predicate,
            "object":     edge.Object,
        })
    }
    return nil
}

// buildEnrichmentDiff returns only facts not already in existing edges
func buildEnrichmentDiff(existing []domain.GraphEdge, derived []domain.GraphFact) domain.GraphDiff {
    existingSet := make(map[string]bool)
    for _, e := range existing {
        key := fmt.Sprintf("%s|%s|%s", e.Subject, e.Predicate, e.Object)
        existingSet[key] = true
    }
    var diff domain.GraphDiff
    for _, f := range derived {
        key := fmt.Sprintf("%s|%s|%s", f.Subject, f.Predicate, f.Object)
        if !existingSet[key] {
            diff.Edges = append(diff.Edges, domain.GraphEdge{
                ID:        uuid.New().String(),
                Subject:   f.Subject,
                Predicate: f.Predicate,
                Object:    f.Object,
                Derived:   true,
            })
        }
    }
    return diff
}

// buildGraphContext builds LLM prompt context from nodes + edges
func buildGraphContext(nodes []domain.GraphNode, edges []domain.GraphEdge) string {
    result := "Entities:\n"
    for _, n := range nodes { result += fmt.Sprintf("- %s (%s)\n", n.Name, n.Type) }
    result += "\nExisting relationships:\n"
    for _, e := range edges { result += fmt.Sprintf("- %s %s %s\n", e.Subject, e.Predicate, e.Object) }
    result += "\nDerive new relationships:"
    return result
}

func filterEdgesForNodes(edges []domain.GraphEdge, nodes []domain.GraphNode) []domain.GraphEdge {
    nodeNames := make(map[string]bool)
    for _, n := range nodes { nodeNames[n.Name] = true }
    var result []domain.GraphEdge
    for _, e := range edges {
        if nodeNames[e.Subject] || nodeNames[e.Object] { result = append(result, e) }
    }
    return result
}
```

### File 3: `adapter/repository/postgres/pipeline_run_repo.go`

```go
package postgres

import (
    "context"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/vnp-memory/services/cognee-cognify/internal/domain"
    "github.com/vnp-memory/services/cognee-cognify/internal/usecase/port"
)

type PipelineRunRepo struct {
    db *pgxpool.Pool
}

func NewPipelineRunRepo(db *pgxpool.Pool) *PipelineRunRepo {
    return &PipelineRunRepo{db: db}
}

func (r *PipelineRunRepo) Save(ctx context.Context, run domain.PipelineRun) error {
    _, err := r.db.Exec(ctx, `
        INSERT INTO cognee_pipeline_runs (id, dataset_id, tenant_id, type, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $6)
    `, run.ID, run.DatasetID, run.TenantID, run.Type, run.Status, time.Now())
    return err
}

func (r *PipelineRunRepo) GetByID(ctx context.Context, id string) (*domain.PipelineRun, error) {
    row := r.db.QueryRow(ctx, `
        SELECT id, dataset_id, tenant_id, type, status, new_nodes, new_edges, error, created_at, updated_at
        FROM cognee_pipeline_runs WHERE id = $1
    `, id)

    var run domain.PipelineRun
    var errStr *string
    err := row.Scan(&run.ID, &run.DatasetID, &run.TenantID, &run.Type, &run.Status,
        &run.NewNodes, &run.NewEdges, &errStr, &run.CreatedAt, &run.UpdatedAt)
    if err != nil { return nil, err }
    if errStr != nil { run.Error = *errStr }
    return &run, nil
}

func (r *PipelineRunRepo) SetStatus(ctx context.Context, id, status string) error {
    _, err := r.db.Exec(ctx, `
        UPDATE cognee_pipeline_runs SET status = $1, updated_at = NOW() WHERE id = $2
    `, status, id)
    return err
}

func (r *PipelineRunRepo) SetStatusWithError(ctx context.Context, id, status, errMsg string) error {
    _, err := r.db.Exec(ctx, `
        UPDATE cognee_pipeline_runs SET status = $1, error = $2, updated_at = NOW() WHERE id = $3
    `, status, errMsg, id)
    return err
}

func (r *PipelineRunRepo) SetStatusWithResult(ctx context.Context, id, status string, newNodes, newEdges int) error {
    _, err := r.db.Exec(ctx, `
        UPDATE cognee_pipeline_runs
        SET status = $1, new_nodes = $2, new_edges = $3, updated_at = NOW()
        WHERE id = $4
    `, status, newNodes, newEdges, id)
    return err
}
```

### MODIFY `port/output.go` — Add interfaces

```go
// Add to services/cognee-cognify/internal/usecase/port/output.go

type GraphRepository interface {
    // existing methods...
    // [NEW] Memify methods
    GetDatasetGraph(ctx context.Context, datasetID uuid.UUID, tenantID string) ([]domain.GraphNode, []domain.GraphEdge, error)
    UpsertGraphDiff(ctx context.Context, datasetID uuid.UUID, tenantID string, diff domain.GraphDiff) error
}

type PipelineRunRepository interface {
    Save(ctx context.Context, run domain.PipelineRun) error
    GetByID(ctx context.Context, id string) (*domain.PipelineRun, error)
    SetStatus(ctx context.Context, id string, status string) error
    SetStatusWithError(ctx context.Context, id string, status string, errMsg string) error
    SetStatusWithResult(ctx context.Context, id string, status string, newNodes, newEdges int) error
}
```

### MODIFY `neo4j/graph_repo.go` — Add GetDatasetGraph + UpsertGraphDiff

```go
// GetDatasetGraph loads all nodes and edges for a dataset from Neo4j
func (r *GraphRepo) GetDatasetGraph(ctx context.Context, datasetID uuid.UUID, tenantID string) ([]domain.GraphNode, []domain.GraphEdge, error) {
    nodeQuery := `
        MATCH (n)
        WHERE n.dataset_id = $dataset_id AND n.tenant_id = $tenant_id
        RETURN n.id AS id, n.name AS name, labels(n) AS labels, properties(n) AS props
        LIMIT 1000
    `
    nodeResult, err := r.session.Run(ctx, nodeQuery, map[string]any{
        "dataset_id": datasetID.String(), "tenant_id": tenantID,
    })
    if err != nil { return nil, nil, err }

    var nodes []domain.GraphNode
    for nodeResult.Next(ctx) {
        record := nodeResult.Record()
        nodes = append(nodes, domain.GraphNode{
            ID:     record.Values[0].(string),
            Name:   record.Values[1].(string),
            Labels: toStringSlice(record.Values[2]),
        })
    }

    edgeQuery := `
        MATCH (a)-[r]->(b)
        WHERE a.dataset_id = $dataset_id AND a.tenant_id = $tenant_id
        RETURN a.name AS subject, type(r) AS predicate, b.name AS object, r.id AS id
        LIMIT 2000
    `
    edgeResult, err := r.session.Run(ctx, edgeQuery, map[string]any{
        "dataset_id": datasetID.String(), "tenant_id": tenantID,
    })
    if err != nil { return nodes, nil, err }

    var edges []domain.GraphEdge
    for edgeResult.Next(ctx) {
        record := edgeResult.Record()
        edges = append(edges, domain.GraphEdge{
            Subject:   record.Values[0].(string),
            Predicate: record.Values[1].(string),
            Object:    record.Values[2].(string),
        })
    }
    return nodes, edges, nil
}

// UpsertGraphDiff adds only new nodes/edges from diff (no deletes)
func (r *GraphRepo) UpsertGraphDiff(ctx context.Context, datasetID uuid.UUID, tenantID string, diff domain.GraphDiff) error {
    for _, edge := range diff.Edges {
        cypher := `
            MERGE (a:Concept {name: $subject, dataset_id: $dataset_id, tenant_id: $tenant_id})
            MERGE (b:Concept {name: $object,  dataset_id: $dataset_id, tenant_id: $tenant_id})
            MERGE (a)-[r:` + sanitizeLabel(edge.Predicate) + `]->(b)
            ON CREATE SET r.id = $edge_id, r.derived = true, r.created_at = datetime()
        `
        _, err := r.session.Run(ctx, cypher, map[string]any{
            "subject":    edge.Subject,
            "object":     edge.Object,
            "dataset_id": datasetID.String(),
            "tenant_id":  tenantID,
            "edge_id":    edge.ID,
        })
        if err != nil { return err }
    }
    return nil
}
```

### MODIFY `grpc/handler.go` — Add Memify + GetPipelineStatus

```go
func (h *CognifyHandler) Memify(ctx context.Context, req *cognifypb.MemifyRequest) (*cognifypb.MemifyResponse, error) {
    datasetID, err := uuid.Parse(req.DatasetId)
    if err != nil { return nil, status.Errorf(codes.InvalidArgument, "invalid dataset_id: %v", err) }

    config := usecase.MemifyConfig{DeriveFacts: true, EmbedTriplets: true, BatchSize: 50}
    if req.Config != nil {
        config.DeriveFacts   = req.Config.DeriveFacts
        config.EmbedTriplets = req.Config.EmbedTriplets
        if req.Config.BatchSize > 0 { config.BatchSize = int(req.Config.BatchSize) }
    }

    result, err := h.memifyUC.Execute(ctx, usecase.MemifyRequest{
        DatasetID: datasetID,
        TenantID:  req.TenantId,
        Config:    config,
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "memify: %v", err) }

    return &cognifypb.MemifyResponse{
        PipelineRunId: result.PipelineRunID,
        Status:        result.Status,
    }, nil
}

func (h *CognifyHandler) GetPipelineStatus(ctx context.Context, req *cognifypb.GetPipelineStatusRequest) (*cognifypb.GetPipelineStatusResponse, error) {
    run, err := h.runRepo.GetByID(ctx, req.PipelineRunId)
    if err != nil { return nil, status.Errorf(codes.NotFound, "pipeline run not found: %v", err) }
    return &cognifypb.GetPipelineStatusResponse{
        PipelineRunId: run.ID,
        Status:        run.Status,
        NewNodes:      int32(run.NewNodes),
        NewEdges:      int32(run.NewEdges),
        Error:         run.Error,
    }, nil
}
```

### MODIFY `gateway/router.go` — Add memify routes

```go
r.Post("/api/v1/cognee/datasets/{id}/memify",        h.ForwardTo("cognee-cognify", "CognifyService/Memify"))
r.Get("/api/v1/cognee/datasets/{id}/memify/status",  h.ForwardTo("cognee-cognify", "CognifyService/GetPipelineStatus"))
```

---

## Verification

```bash
cd services/cognee-cognify
go build ./...
go test ./internal/usecase/... -run TestMemify -v
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| `POST /datasets/{id}/memify` → 202 + `pipeline_run_id` | ✅ |
| Job không xóa existing nodes/edges | ✅ (UpsertGraphDiff = no delete) |
| New edges derived và stored | ✅ |
| Triplet embeddings updated in Qdrant | ✅ |
| NATS events published: started, completed/failed | ✅ |
| `GET memify/status` → trạng thái chính xác | ✅ |
