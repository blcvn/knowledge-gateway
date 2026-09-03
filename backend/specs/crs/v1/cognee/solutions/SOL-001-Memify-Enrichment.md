# Solution: SOL-001 — Memify (Graph Enrichment)

**CR ID:** CR-COGNEE-001  
**Solution ID:** SOL-001  
**Priority:** High (Wave 2)  
**Architecture:** EXTEND `services/cognee-cognify/` + Gateway routes

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md §5.1`:
- `services/cognee-cognify/internal/usecase/start_cognify.go` — chạy 7 pipeline steps theo thứ tự cứng.
- **Step 3 (ExtractGraph)** dùng LLM → destructive: rebuild graph từ đầu mỗi lần cognify.
- **Step 5 (AddDatapoints)** upsert Neo4j nodes/edges + embed vào Qdrant.
- `cognee-cognify` expose gRPC port 9012, trong monolith qua bufconn.
- NATS stream `cognee.*` đã tồn tại (events: `cognee.pipeline.completed`).
- `pipeline-service` domain có `PipelineRun` entity — tái dùng để track memify job.

**Vấn đề cốt lõi:** Pipeline hiện tại delete + rebuild (bước `ExtractGraph` dùng LLM làm mất graph context cũ). Memify cần **upsert-only** — không xóa nodes/edges cũ.

---

## 2. Giải pháp chi tiết

### 2.1. [NEW] `internal/usecase/memify.go` trong `services/cognee-cognify`

```go
// services/cognee-cognify/internal/usecase/memify.go

package usecase

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/services/cognee-cognify/internal/domain"
    "github.com/vnp-memory/services/cognee-cognify/internal/usecase/port"
)

const (
    defaultMemifyBatchSize  = 50
    memifyDefaultDeriveFacts = true
    memifyDefaultEmbed       = true
)

type MemifyRequest struct {
    DatasetID   uuid.UUID
    TenantID    string
    Config      MemifyConfig
}

type MemifyConfig struct {
    DeriveFacts  bool   // default: true
    EmbedTriplets bool  // default: true
    BatchSize    int    // default: 50
}

type MemifyResult struct {
    PipelineRunID string
    Status        string // QUEUED | RUNNING | COMPLETED | FAILED
    NewNodes      int
    NewEdges      int
}

type MemifyUseCase struct {
    graphRepo   port.GraphRepository      // Neo4j: read existing, upsert new
    vectorRepo  port.VectorRepository     // Qdrant: update triplet embeddings
    llmClient   port.LLMClient            // Bifrost: structured fact derivation
    embedder    port.EmbedderClient       // Bifrost: re-embed triplets
    runRepo     port.PipelineRunRepository // Postgres: track job status
    eventPub    port.EventPublisher        // NATS
}

func NewMemifyUseCase(
    graphRepo port.GraphRepository,
    vectorRepo port.VectorRepository,
    llmClient port.LLMClient,
    embedder port.EmbedderClient,
    runRepo port.PipelineRunRepository,
    eventPub port.EventPublisher,
) *MemifyUseCase {
    return &MemifyUseCase{graphRepo, vectorRepo, llmClient, embedder, runRepo, eventPub}
}

// Execute creates a PipelineRun and launches memify in background goroutine
func (uc *MemifyUseCase) Execute(ctx context.Context, req MemifyRequest) (*MemifyResult, error) {
    // Apply defaults
    if req.Config.BatchSize == 0 { req.Config.BatchSize = defaultMemifyBatchSize }

    // Create PipelineRun record (QUEUED)
    runID := uuid.New().String()
    run := domain.PipelineRun{
        ID:        runID,
        DatasetID: req.DatasetID.String(),
        TenantID:  req.TenantID,
        Type:      "memify",
        Status: ✅ Implemented   "QUEUED",
        CreatedAt: time.Now(),
    }
    if err := uc.runRepo.Save(ctx, run); err != nil {
        return nil, fmt.Errorf("save pipeline run: %w", err)
    }

    // Launch background
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
    if err != nil {
        return nil, fmt.Errorf("load graph: %w", err)
    }

    // Step 2: Derive facts via LLM (batch)
    var derivedFacts []domain.GraphFact
    if req.Config.DeriveFacts {
        derivedFacts, err = uc.deriveFacts(ctx, existingNodes, existingEdges, req.Config.BatchSize)
        if err != nil {
            return nil, fmt.Errorf("derive facts: %w", err)
        }
    }

    // Step 3: Build enrichment diff (only NEW facts not already in graph)
    diff := buildEnrichmentDiff(existingEdges, derivedFacts)

    // Step 4: Upsert nodes + edges (non-destructive)
    if len(diff.Nodes) > 0 || len(diff.Edges) > 0 {
        if err := uc.graphRepo.UpsertGraphDiff(ctx, req.DatasetID, req.TenantID, diff); err != nil {
            return nil, fmt.Errorf("upsert graph diff: %w", err)
        }
        result.NewNodes = len(diff.Nodes)
        result.NewEdges = len(diff.Edges)
    }

    // Step 5: Re-embed triplets (only new ones)
    if req.Config.EmbedTriplets && len(diff.Edges) > 0 {
        if err := uc.embedAndIndexTriplets(ctx, req.DatasetID, req.TenantID, diff.Edges); err != nil {
            // Non-fatal: log warning, don't fail the job
            // Triplet re-embedding failure doesn't corrupt graph
            _ = err
        }
    }

    return result, nil
}

// deriveFacts calls Bifrost LLM to infer new relationships from existing nodes/edges
func (uc *MemifyUseCase) deriveFacts(ctx context.Context,
    nodes []domain.GraphNode, edges []domain.GraphEdge, batchSize int) ([]domain.GraphFact, error) {

    const systemPrompt = `You are a knowledge graph enricher. Given existing entities and their relationships,
infer NEW relationships that can be logically derived but are not explicitly stated.
Return only relationships that are definitively true based on the given facts.
Format: JSON array of {"subject": "name", "predicate": "relation_type", "object": "name"}`

    var allFacts []domain.GraphFact
    // Process in batches of batchSize nodes
    for i := 0; i < len(nodes); i += batchSize {
        end := i + batchSize
        if end > len(nodes) { end = len(nodes) }
        batch := nodes[i:end]

        userMsg := buildGraphContext(batch, filterEdgesForNodes(edges, batch))
        resp, err := uc.llmClient.Chat(ctx, systemPrompt, userMsg)
        if err != nil { return nil, err }

        facts, err := parseFactsFromLLMResponse(resp)
        if err != nil { continue } // skip bad batch
        allFacts = append(allFacts, facts...)
    }
    return allFacts, nil
}

// embedAndIndexTriplets embeds triplet text and upserts into Qdrant collection
func (uc *MemifyUseCase) embedAndIndexTriplets(ctx context.Context,
    datasetID uuid.UUID, tenantID string, edges []domain.GraphEdge) error {

    collectionName := fmt.Sprintf("cognee_triplets_%s", tenantID)
    for _, edge := range edges {
        tripletText := fmt.Sprintf("%s %s %s", edge.Subject, edge.Predicate, edge.Object)
        vec, err := uc.embedder.Embed(ctx, tripletText)
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

// buildEnrichmentDiff compares derivedFacts against existing edges
// Returns only truly new nodes/edges not already present in the graph
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
```

### 2.2. [MODIFY] gRPC Handler — `internal/adapter/grpc/handler.go`

```go
// Thêm phương thức Memify vào CognifyService gRPC handler

func (h *CognifyHandler) Memify(ctx context.Context, req *cognifypb.MemifyRequest) (*cognifypb.MemifyResponse, error) {
    datasetID, err := uuid.Parse(req.DatasetId)
    if err != nil { return nil, status.Errorf(codes.InvalidArgument, "invalid dataset_id: %v", err) }

    config := usecase.MemifyConfig{
        DeriveFacts:   true,
        EmbedTriplets: true,
        BatchSize:     50,
    }
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

### 2.3. [MODIFY] Proto — `api/proto/cognee/cognify/v1/cognify.proto`

```protobuf
syntax = "proto3";
package cognee.cognify.v1;
import "google/protobuf/timestamp.proto";

service CognifyService {
  // Existing
  rpc StartCognify(StartCognifyRequest) returns (StartCognifyResponse);
  rpc GetPipelineStatus(GetPipelineStatusRequest) returns (GetPipelineStatusResponse);
  // [NEW]
  rpc Memify(MemifyRequest) returns (MemifyResponse);
}

message MemifyRequest {
  string dataset_id = 1;
  string tenant_id  = 2;
  optional MemifyConfig config = 3;
}

message MemifyConfig {
  bool  derive_facts   = 1;   // default: true
  bool  embed_triplets = 2;   // default: true
  int32 batch_size     = 3;   // default: 50
}

message MemifyResponse {
  string pipeline_run_id = 1;
  string status          = 2;  // QUEUED | RUNNING | COMPLETED | FAILED
}

message GetPipelineStatusRequest {
  string pipeline_run_id = 1;
  string dataset_id      = 2;  // alternative to pipeline_run_id
}

message GetPipelineStatusResponse {
  string pipeline_run_id = 1;
  string status          = 2;
  int32  new_nodes       = 3;
  int32  new_edges       = 4;
  string error           = 5;
  google.protobuf.Timestamp completed_at = 6;
}
```

### 2.4. [NEW] Domain Types — `internal/domain/graph_diff.go`

```go
// services/cognee-cognify/internal/domain/graph_diff.go

type GraphFact struct {
    Subject   string
    Predicate string
    Object    string
}

type GraphDiff struct {
    Nodes []GraphNode   // new nodes inferred
    Edges []GraphEdge   // new edges derived
}

type GraphNode struct {
    ID         string
    Name       string
    Type       string
    Properties map[string]any
    Derived    bool
}

type GraphEdge struct {
    ID        string
    Subject   string
    Predicate string
    Object    string
    Weight    float64
    Derived   bool
}

type PipelineRun struct {
    ID        string
    DatasetID string
    TenantID  string
    Type      string   // "cognify" | "memify"
    Status    string   // QUEUED | RUNNING | COMPLETED | FAILED
    NewNodes  int
    NewEdges  int
    Error     string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 2.5. [NEW] Repository Port — `internal/usecase/port/output.go`

```go
// Thêm vào port/output.go

type GraphRepository interface {
    // Existing
    UpsertNodes(ctx, datasetID, tenantID, nodes) error
    UpsertEdges(ctx, datasetID, tenantID, edges) error
    // [NEW for Memify]
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

### 2.6. [NEW] PostgreSQL Schema

```sql
-- Migration: 0020_cognee_pipeline_runs.up.sql

CREATE TABLE cognee_pipeline_runs (
    id          TEXT PRIMARY KEY,
    dataset_id  TEXT NOT NULL,
    tenant_id   TEXT NOT NULL,
    type        TEXT NOT NULL,    -- 'cognify' | 'memify'
    status      TEXT NOT NULL DEFAULT 'QUEUED',
    new_nodes   INT DEFAULT 0,
    new_edges   INT DEFAULT 0,
    error       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cognee_pipeline_runs_dataset ON cognee_pipeline_runs(dataset_id, tenant_id, status);
```

### 2.7. [MODIFY] Gateway Routes — `gateway/internal/adapter/handler/router.go`

```go
// Thêm vào Cognee route group

// Memify — non-destructive graph enrichment
r.Post("/api/v1/cognee/datasets/{id}/memify",         h.ForwardTo("cognee-cognify", "CognifyService/Memify"))
r.Get("/api/v1/cognee/datasets/{id}/memify/status",   h.ForwardTo("cognee-cognify", "CognifyService/GetPipelineStatus"))
// Note: {id} = dataset_id, mapped thành MemifyRequest.DatasetId trong gRPC mapper
```

### 2.8. [MODIFY] Bootstrap — `apps/memory/internal/bootstrap/cognee.go`

```go
// Thêm MemifyUseCase vào CognifyService initialization

func InitCogneeServices(reg *bus.InProcessRegistry, db *sql.DB, neo4j *neo4j.Driver,
    qdrant *qdrantclient.Client, bifrost *bifrost.Client, nats *nats.Conn) {

    // Existing repos
    graphRepo   := neo4jadapter.NewGraphRepo(neo4j)
    vectorRepo  := qdrantadapter.NewVectorRepo(qdrant)
    llmClient   := bifrostadapter.NewLLMClient(bifrost)
    embedder    := bifrostadapter.NewEmbedder(bifrost)
    publisher   := natevent.NewPublisher(nats, "cognee")

    // [NEW] PipelineRun repo
    runRepo := postgresadapter.NewPipelineRunRepo(db)

    // [NEW] MemifyUseCase
    memifyUC := usecase.NewMemifyUseCase(graphRepo, vectorRepo, llmClient, embedder, runRepo, publisher)

    // Register updated gRPC handler
    handler := grpchandler.NewCognifyHandler(
        startCognifyUC, // existing
        memifyUC,       // [NEW]
        runRepo,        // [NEW]
    )
    cognifypb.RegisterCognifyServiceServer(grpcServer, handler)
}
```

---

## 3. Files

### [NEW]

| File | Mô tả |
|------|-------|
| `services/cognee-cognify/internal/usecase/memify.go` | MemifyUseCase: 5-step enrichment |
| `services/cognee-cognify/internal/domain/graph_diff.go` | GraphFact, GraphDiff, PipelineRun types |
| `services/cognee-cognify/internal/adapter/repository/postgres/pipeline_run_repo.go` | PipelineRun CRUD |
| `db/migrations/0020_cognee_pipeline_runs.up.sql` | cognee_pipeline_runs table |

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `api/proto/cognee/cognify/v1/cognify.proto` | + Memify RPC + GetPipelineStatus RPC |
| `services/cognee-cognify/internal/adapter/grpc/handler.go` | + Memify(), GetPipelineStatus() |
| `services/cognee-cognify/internal/usecase/port/output.go` | + GetDatasetGraph(), UpsertGraphDiff(), PipelineRunRepository |
| `services/cognee-cognify/internal/adapter/repository/neo4j/graph_repo.go` | + GetDatasetGraph(), UpsertGraphDiff() |
| `gateway/internal/adapter/handler/router.go` | + 2 memify routes |
| `apps/memory/internal/bootstrap/cognee.go` | + memifyUC init |

---

## 4. Acceptance Criteria Mapping

| AC từ CR-COGNEE-001 | Covered by |
|--------------------|-----------|
| `POST /api/v1/cognee/datasets/{id}/memify` → 202 + pipeline_run_id | gRPC handler Memify() |
| Job không xóa nodes/edges cũ | UpsertGraphDiff() — upsert-only, no delete |
| Số edges tăng sau memify | buildEnrichmentDiff() + result.NewEdges |
| Triplet embeddings updated trong Qdrant | embedAndIndexTriplets() |
| NATS event `cognee.cognify.memify.completed` | eventPub.Publish() in runMemify() |
| GET memify status → trạng thái chính xác | GetPipelineStatus() + runRepo |
