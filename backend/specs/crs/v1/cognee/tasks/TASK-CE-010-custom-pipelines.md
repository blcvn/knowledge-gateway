# TASK-CE-010 — Custom Pipelines Orchestration (Chain-of-Responsibility)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-CE-010 |
| **Wave** | 4 |
| **Component** | `services/cognee-cognify/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-006 §2.1 → §2.6 |
| **Priority** | Low |
| **Depends On** | TASK-CE-006 |
| **Estimated** | 6h |

---

## Context

Refactor `start_cognify.go` từ 7-bước hardcoded → **Chain-of-Responsibility** pattern. Mỗi step là `StepHandler` interface. `PipelineConfig` quyết định bước nào chạy.

**Key requirements:**
- Backward compatible: không truyền `template/steps` → chạy đủ 7 bước như trước
- Predefined templates: `STANDARD`, `EMBED_ONLY`, `FAST_INDEX`, `TEMPORAL`, `GRAPH_ONLY`
- Custom steps: user truyền explicit `steps` list
- `GetPipelineTemplates` RPC để list templates

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/cognee-cognify/internal/domain/pipeline.go` |
| CREATE | `services/cognee-cognify/internal/usecase/step_handler.go` |
| CREATE | `services/cognee-cognify/internal/usecase/steps/classify_step.go` |
| CREATE | `services/cognee-cognify/internal/usecase/steps/chunk_step.go` |
| CREATE | `services/cognee-cognify/internal/usecase/steps/extract_graph_step.go` |
| CREATE | `services/cognee-cognify/internal/usecase/steps/build_ontology_step.go` |
| CREATE | `services/cognee-cognify/internal/usecase/steps/add_datapoints_step.go` |
| CREATE | `services/cognee-cognify/internal/usecase/steps/detect_community_step.go` |
| CREATE | `services/cognee-cognify/internal/usecase/steps/summarize_community_step.go` |
| CREATE | `services/cognee-cognify/internal/usecase/steps/extract_temporal_step.go` |
| REFACTOR | `services/cognee-cognify/internal/usecase/start_cognify.go` |
| MODIFY | `services/cognee-cognify/internal/adapter/grpc/handler.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |
| MODIFY | `apps/memory/internal/bootstrap/cognee.go` |

---

## Implementation

### File 1: `internal/domain/pipeline.go`

```go
package domain

// PipelineStep — identifies each pipeline step
type PipelineStep string

const (
    StepClassify            PipelineStep = "CLASSIFY"
    StepChunk               PipelineStep = "CHUNK"
    StepExtractGraph        PipelineStep = "EXTRACT_GRAPH"
    StepBuildOntology       PipelineStep = "BUILD_ONTOLOGY"
    StepAddDatapoints       PipelineStep = "ADD_DATAPOINTS"
    StepDetectCommunity     PipelineStep = "DETECT_COMMUNITY"
    StepSummarizeCommunity  PipelineStep = "SUMMARIZE_COMMUNITY"
    StepExtractTemporalGraph PipelineStep = "EXTRACT_TEMPORAL_GRAPH"
)

// Pipeline template names — named presets
type PipelineTemplateName string
const (
    TemplateStandard  PipelineTemplateName = "STANDARD"    // all 7 steps (default)
    TemplateEmbedOnly PipelineTemplateName = "EMBED_ONLY"  // CHUNK + ADD_DATAPOINTS (fastest)
    TemplateFastIndex PipelineTemplateName = "FAST_INDEX"  // CLASSIFY + CHUNK + ADD_DATAPOINTS
    TemplateTemporal  PipelineTemplateName = "TEMPORAL"    // uses EXTRACT_TEMPORAL_GRAPH
    TemplateGraphOnly PipelineTemplateName = "GRAPH_ONLY"  // no embedding
)

var templateSteps = map[PipelineTemplateName][]PipelineStep{
    TemplateStandard: {
        StepClassify, StepChunk, StepExtractGraph,
        StepBuildOntology, StepAddDatapoints,
        StepDetectCommunity, StepSummarizeCommunity,
    },
    TemplateEmbedOnly: {StepChunk, StepAddDatapoints},
    TemplateFastIndex: {StepClassify, StepChunk, StepAddDatapoints},
    TemplateTemporal:  {StepClassify, StepChunk, StepExtractTemporalGraph, StepAddDatapoints},
    TemplateGraphOnly: {StepClassify, StepChunk, StepExtractGraph, StepBuildOntology},
}

// PipelineConfig — runtime configuration for pipeline execution
type PipelineConfig struct {
    Template PipelineTemplateName  // named template
    Steps    []PipelineStep        // custom step list (overrides Template)
    Options  PipelineOptions
}

type PipelineOptions struct {
    ChunkSize    int     // override chunk size (default: 512)
    CustomPrompt string  // override LLM extraction prompt
    TemporalMode bool
    SkipCache    bool
}

// Resolve returns the ordered steps based on config
// Empty config → STANDARD template (backward compatible)
func (c PipelineConfig) Resolve() []PipelineStep {
    if c.Template != "" {
        if steps, ok := templateSteps[c.Template]; ok { return steps }
    }
    if len(c.Steps) > 0 { return c.Steps }
    return templateSteps[TemplateStandard]  // default: all 7 steps
}

type PipelineTemplateInfo struct {
    Name  string
    Steps []string
}

func ListTemplates() []PipelineTemplateInfo {
    infos := make([]PipelineTemplateInfo, 0, len(templateSteps))
    for name, steps := range templateSteps {
        strSteps := make([]string, len(steps))
        for i, s := range steps { strSteps[i] = string(s) }
        infos = append(infos, PipelineTemplateInfo{Name: string(name), Steps: strSteps})
    }
    return infos
}
```

### File 2: `internal/usecase/step_handler.go`

```go
package usecase

import "context"

// PipelineState carries data between pipeline steps
type PipelineState struct {
    DatasetID   string
    TenantID    string
    EntryIDs    []string
    NodeSets    []string
    RawContent  []string
    ContentType string
    Chunks      []Chunk
    Nodes       []domain.GraphNode
    Edges       []domain.GraphEdge
    Embeddings  map[string][]float32
    Options     domain.PipelineOptions
}

// StepHandler — interface for each pipeline step
type StepHandler interface {
    Name() domain.PipelineStep
    Execute(ctx context.Context, state *PipelineState) (*PipelineState, error)
}
```

### File 3: `steps/chunk_step.go` (representative step)

```go
package steps

import (
    "context"
    "strings"

    "github.com/vnp-memory/services/cognee-cognify/internal/domain"
    "github.com/vnp-memory/services/cognee-cognify/internal/usecase"
)

type ChunkStep struct {
    defaultSize int  // 512 default
}

func NewChunkStep(defaultSize int) *ChunkStep {
    if defaultSize <= 0 { defaultSize = 512 }
    return &ChunkStep{defaultSize: defaultSize}
}

func (s *ChunkStep) Name() domain.PipelineStep { return domain.StepChunk }

func (s *ChunkStep) Execute(ctx context.Context, state *usecase.PipelineState) (*usecase.PipelineState, error) {
    size := s.defaultSize
    if state.Options.ChunkSize > 0 { size = state.Options.ChunkSize }

    overlap := size / 5  // 20% overlap

    var allChunks []usecase.Chunk
    for _, content := range state.RawContent {
        allChunks = append(allChunks, slidingWindowChunk(content, size, overlap)...)
    }
    state.Chunks = allChunks
    return state, nil
}

func slidingWindowChunk(text string, size, overlap int) []usecase.Chunk {
    words := strings.Fields(text)
    var chunks []usecase.Chunk
    step := size - overlap
    if step <= 0 { step = size }

    for i := 0; i < len(words); i += step {
        end := i + size
        if end > len(words) { end = len(words) }
        chunk := strings.Join(words[i:end], " ")
        chunks = append(chunks, usecase.Chunk{Content: chunk})
        if end == len(words) { break }
    }
    return chunks
}
```

### REFACTOR `start_cognify.go` — Chain-of-Responsibility

```go
package usecase

import (
    "context"
    "fmt"

    "github.com/vnp-memory/services/cognee-cognify/internal/domain"
    "github.com/vnp-memory/services/cognee-cognify/internal/usecase/port"
)

type StartCognifyUseCase struct {
    stepHandlers map[domain.PipelineStep]StepHandler
    runRepo      port.PipelineRunRepository
    publisher    port.EventPublisher
}

func NewStartCognifyUseCase(
    classifyStep           StepHandler,
    chunkStep              StepHandler,
    extractGraphStep       StepHandler,
    buildOntologyStep      StepHandler,
    addDatapointsStep      StepHandler,
    detectCommunityStep    StepHandler,
    summarizeCommunityStep StepHandler,
    extractTemporalStep    StepHandler,
    runRepo                port.PipelineRunRepository,
    publisher              port.EventPublisher,
) *StartCognifyUseCase {
    handlers := map[domain.PipelineStep]StepHandler{
        domain.StepClassify:            classifyStep,
        domain.StepChunk:               chunkStep,
        domain.StepExtractGraph:        extractGraphStep,
        domain.StepBuildOntology:       buildOntologyStep,
        domain.StepAddDatapoints:       addDatapointsStep,
        domain.StepDetectCommunity:     detectCommunityStep,
        domain.StepSummarizeCommunity:  summarizeCommunityStep,
        domain.StepExtractTemporalGraph: extractTemporalStep,
    }
    return &StartCognifyUseCase{stepHandlers: handlers, runRepo: runRepo, publisher: publisher}
}

type CognifyRequest struct {
    DatasetID string
    TenantID  string
    EntryIDs  []string
    NodeSets  []string
    Config    domain.PipelineConfig  // [NEW] pipeline config
}

type CognifyResult struct {
    PipelineRunID string
    Status        string
    StepsExecuted []string
}

func (uc *StartCognifyUseCase) Execute(ctx context.Context, req CognifyRequest) (*CognifyResult, error) {
    // Resolve steps (backward compatible: empty config → STANDARD)
    steps := req.Config.Resolve()

    // Validate step handlers exist
    for _, step := range steps {
        if _, ok := uc.stepHandlers[step]; !ok {
            return nil, fmt.Errorf("unknown pipeline step: %s", step)
        }
    }

    // Create pipeline run record
    runID := uuid.New().String()
    uc.runRepo.Save(ctx, domain.PipelineRun{
        ID: runID, DatasetID: req.DatasetID, TenantID: req.TenantID,
        Type: "cognify", Status: "RUNNING",
    })

    // Initialize state
    state := &PipelineState{
        DatasetID: req.DatasetID,
        TenantID:  req.TenantID,
        EntryIDs:  req.EntryIDs,
        NodeSets:  req.NodeSets,
        Options:   req.Config.Options,
    }

    // Execute chain-of-responsibility
    var executedSteps []string
    for _, stepName := range steps {
        handler := uc.stepHandlers[stepName]
        var err error
        state, err = handler.Execute(ctx, state)
        if err != nil {
            uc.runRepo.SetStatusWithError(ctx, runID, "FAILED", err.Error())
            return nil, fmt.Errorf("step %s failed: %w", stepName, err)
        }
        executedSteps = append(executedSteps, string(stepName))
    }

    uc.runRepo.SetStatus(ctx, runID, "COMPLETED")
    uc.publisher.Publish(ctx, "cognee.pipeline.completed", map[string]any{
        "pipeline_run_id": runID,
        "dataset_id":      req.DatasetID,
        "tenant_id":       req.TenantID,
        "steps_executed":  executedSteps,
    })

    return &CognifyResult{PipelineRunID: runID, Status: "COMPLETED", StepsExecuted: executedSteps}, nil
}

// GetTemplates returns available templates (for REST endpoint)
func (uc *StartCognifyUseCase) GetTemplates() []domain.PipelineTemplateInfo {
    return domain.ListTemplates()
}
```

### MODIFY `grpc/handler.go` — Parse template + steps from proto; GetPipelineTemplates

```go
func (h *CognifyHandler) StartCognify(ctx context.Context, req *cognifypb.StartCognifyRequest) (*cognifypb.StartCognifyResponse, error) {
    // Parse pipeline config from proto
    config := domain.PipelineConfig{}
    if req.Template != "" {
        config.Template = domain.PipelineTemplateName(req.Template)
    }
    for _, s := range req.Steps {
        config.Steps = append(config.Steps, domain.PipelineStep(s))
    }
    if req.Options != nil {
        config.Options = domain.PipelineOptions{
            ChunkSize:    int(req.Options.ChunkSize),
            CustomPrompt: req.Options.CustomPrompt,
            TemporalMode: req.Options.TemporalMode,
            SkipCache:    req.Options.SkipCache,
        }
    }

    result, err := h.startCognifyUC.Execute(ctx, usecase.CognifyRequest{
        DatasetID: req.DatasetId,
        TenantID:  req.TenantId,
        EntryIDs:  req.EntryIds,
        NodeSets:  req.NodeSets,
        Config:    config,
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "cognify: %v", err) }

    return &cognifypb.StartCognifyResponse{
        PipelineRunId: result.PipelineRunID,
        Status:        result.Status,
        StepsExecuted: result.StepsExecuted,
    }, nil
}

func (h *CognifyHandler) GetPipelineTemplates(ctx context.Context, req *cognifypb.GetPipelineTemplatesRequest) (*cognifypb.GetPipelineTemplatesResponse, error) {
    templates := h.startCognifyUC.GetTemplates()
    pbTemplates := make([]*cognifypb.PipelineTemplateInfo, 0, len(templates))
    for _, t := range templates {
        pbTemplates = append(pbTemplates, &cognifypb.PipelineTemplateInfo{
            Name:  t.Name,
            Steps: t.Steps,
        })
    }
    return &cognifypb.GetPipelineTemplatesResponse{Templates: pbTemplates}, nil
}
```

### MODIFY `gateway/router.go` — Add templates route

```go
r.Get("/api/v1/cognee/pipeline/templates",
    h.ForwardTo("cognee-cognify", "CognifyService/GetPipelineTemplates"))
r.Get("/v1/console/pipelines/cognee",
    h.ForwardTo("cognee-cognify", "CognifyService/GetPipelineTemplates"))
```

### MODIFY `bootstrap/cognee.go` — Inject 8 step handlers

```go
// apps/memory/internal/bootstrap/cognee.go

func InitCogneeServices(reg *bus.InProcessRegistry, db *sql.DB, neo4j *neo4j.DriverWithContext,
    qdrant *qdrantclient.Client, bifrost *bifrost.Client, nats *nats.Conn) {

    // ... repos ...
    runRepo := postgresadapter.NewPipelineRunRepo(db)

    // Step handlers
    classifyStep           := steps.NewClassifyStep()
    chunkStep              := steps.NewChunkStep(512)
    extractGraphStep       := steps.NewExtractGraphStep(llmClient, graphRepo, embedder)
    buildOntologyStep      := steps.NewBuildOntologyStep(llmClient)
    addDatapointsStep      := steps.NewAddDatapointsStep(graphRepo, vectorRepo, embedder)
    detectCommunityStep    := steps.NewDetectCommunityStep(graphRepo)
    summarizeCommunityStep := steps.NewSummarizeCommunityStep(llmClient, graphRepo)
    extractTemporalStep    := steps.NewExtractTemporalGraphStep(llmClient, graphRepo)

    startCognifyUC := usecase.NewStartCognifyUseCase(
        classifyStep, chunkStep, extractGraphStep, buildOntologyStep,
        addDatapointsStep, detectCommunityStep, summarizeCommunityStep, extractTemporalStep,
        runRepo, publisher,
    )

    memifyUC := usecase.NewMemifyUseCase(graphRepo, vectorRepo, llmClient, embedder, runRepo, publisher)

    handler := grpchandler.NewCognifyHandler(startCognifyUC, memifyUC, runRepo)
    cognifypb.RegisterCognifyServiceServer(grpcServer, handler)
}
```

---

## Verification

```bash
cd services/cognee-cognify
go build ./...
go test ./internal/usecase/... -run TestPipeline -v
```

**Template tests:**
```go
func TestPipelineConfig_EmbedOnly(t *testing.T) {
    cfg := domain.PipelineConfig{Template: domain.TemplateEmbedOnly}
    steps := cfg.Resolve()
    assert.Equal(t, []domain.PipelineStep{domain.StepChunk, domain.StepAddDatapoints}, steps)
}

func TestPipelineConfig_BackwardCompatible(t *testing.T) {
    cfg := domain.PipelineConfig{}  // empty
    steps := cfg.Resolve()
    assert.Equal(t, 7, len(steps))  // STANDARD = 7 steps
}

func TestPipelineConfig_CustomSteps(t *testing.T) {
    cfg := domain.PipelineConfig{Steps: []domain.PipelineStep{
        domain.StepClassify, domain.StepChunk, domain.StepExtractGraph,
    }}
    steps := cfg.Resolve()
    assert.Equal(t, 3, len(steps))
}
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| `template: "EMBED_ONLY"` chỉ chạy CHUNK + ADD_DATAPOINTS | ✅ |
| `steps: ["CLASSIFY","CHUNK","EXTRACT_GRAPH"]` → 3 bước đúng thứ tự | ✅ |
| `GET /api/v1/cognee/pipeline/templates` → 5 templates | ✅ |
| Empty config → STANDARD (7 bước) — backward compatible | ✅ |
| Unknown step → error trả về | ✅ |
