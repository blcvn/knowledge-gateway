# Solution: SOL-006 — Custom Pipelines Orchestration

**CR ID:** CR-COGNEE-006  
**Solution ID:** SOL-006  
**Priority:** Low (Wave 4)  
**Architecture:** REFACTOR `services/cognee-cognify/internal/usecase/start_cognify.go` — Chain-of-Responsibility

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md §5.1`:
- `pipeline-service/internal/domain/pipeline/` đã có `PipelineTemplate` entity — có thể tái dùng concept.
- `cognee-cognify/internal/usecase/start_cognify.go` — 7 bước hardcoded theo thứ tự cứng.
- `FEAT-015` console routes đã có: `/v1/console/pipelines/` — 7 routes (status, queues, workers, templates, jobs).
- Bifrost LLM chỉ được gọi ở bước 3 (ExtractGraph) và 4 (BuildOntology).

**Chiến lược:** Refactor `StartCognifyUseCase` theo **Chain-of-Responsibility** pattern. Mỗi step là `StepHandler` interface. `PipelineConfig` quyết định step nào chạy. **Backward compatible:** không truyền `template/steps` → chạy đủ 7 bước như trước.

---

## 2. Giải pháp chi tiết

### 2.1. [NEW] Domain — `internal/domain/pipeline.go`

```go
// services/cognee-cognify/internal/domain/pipeline.go

package domain

// PipelineStep — định danh từng step
type PipelineStep string

const (
    StepClassify            PipelineStep = "CLASSIFY"
    StepChunk               PipelineStep = "CHUNK"
    StepExtractGraph        PipelineStep = "EXTRACT_GRAPH"
    StepBuildOntology       PipelineStep = "BUILD_ONTOLOGY"
    StepAddDatapoints       PipelineStep = "ADD_DATAPOINTS"
    StepDetectCommunity     PipelineStep = "DETECT_COMMUNITY"
    StepSummarizeCommunity  PipelineStep = "SUMMARIZE_COMMUNITY"
    StepExtractTemporalGraph PipelineStep = "EXTRACT_TEMPORAL_GRAPH"  // temporal variant
)

// Pipeline templates — named presets
type PipelineTemplateName string
const (
    TemplateStandard   PipelineTemplateName = "STANDARD"    // All 7 steps (default)
    TemplateEmbedOnly  PipelineTemplateName = "EMBED_ONLY"  // CHUNK + ADD_DATAPOINTS
    TemplateFastIndex  PipelineTemplateName = "FAST_INDEX"  // CLASSIFY + CHUNK + ADD_DATAPOINTS
    TemplateTemporal   PipelineTemplateName = "TEMPORAL"    // CLASSIFY + CHUNK + EXTRACT_TEMPORAL_GRAPH + ADD_DATAPOINTS
    TemplateGraphOnly  PipelineTemplateName = "GRAPH_ONLY"  // CLASSIFY + CHUNK + EXTRACT_GRAPH + BUILD_ONTOLOGY
)

// templateSteps maps template names to their ordered steps
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

func GetTemplateSteps(name PipelineTemplateName) ([]PipelineStep, bool) {
    steps, ok := templateSteps[name]
    return steps, ok
}

func ListTemplates() []PipelineTemplateInfo {
    infos := make([]PipelineTemplateInfo, 0, len(templateSteps))
    for name, steps := range templateSteps {
        infos = append(infos, PipelineTemplateInfo{Name: string(name), Steps: stepsToStrings(steps)})
    }
    return infos
}

type PipelineTemplateInfo struct {
    Name  string
    Steps []string
}

// PipelineConfig — runtime configuration for a pipeline execution
type PipelineConfig struct {
    Template     PipelineTemplateName   // Use named template (overrides Steps if set)
    Steps        []PipelineStep         // Custom step list (used when Template is empty)
    Options      PipelineOptions
}

type PipelineOptions struct {
    ChunkSize    int     // Override chunk size (default: 512)
    CustomPrompt string  // Override LLM extraction prompt
    TemporalMode bool    // Use temporal extraction variant
    SkipCache    bool    // Force re-run even if cached
}

// Resolve returns the ordered steps to execute based on config
func (c PipelineConfig) Resolve() []PipelineStep {
    if c.Template != "" {
        if steps, ok := templateSteps[c.Template]; ok { return steps }
    }
    if len(c.Steps) > 0 { return c.Steps }
    return templateSteps[TemplateStandard]  // default: all 7 steps
}
```

### 2.2. [NEW] StepHandler Interface

```go
// services/cognee-cognify/internal/usecase/step_handler.go

package usecase

import "context"

// PipelineState carries data between pipeline steps
type PipelineState struct {
    DatasetID   string
    TenantID    string
    EntryIDs    []string
    NodeSets    []string           // from CR-002
    RawContent  []string           // raw text from extractor
    ContentType string             // classified content type
    Chunks      []Chunk            // after ChunkStep
    Nodes       []domain.GraphNode // after ExtractGraphStep
    Edges       []domain.GraphEdge // after ExtractGraphStep
    Embeddings  map[string][]float32 // docID → vector
    Options     domain.PipelineOptions
}

// StepHandler — interface for each pipeline step
type StepHandler interface {
    Execute(ctx context.Context, state *PipelineState) (*PipelineState, error)
    Name() domain.PipelineStep
}

// Step implementations (one file per step):
// - ClassifyStep: uses content type detection
// - ChunkStep: splits content into chunks with sliding window
// - ExtractGraphStep: calls Bifrost LLM for entity extraction
// - BuildOntologyStep: calls Bifrost for ontology inference
// - AddDatapointsStep: upserts Neo4j nodes + Qdrant vectors
// - DetectCommunityStep: Louvain algorithm via Neo4j GDS
// - SummarizeCommunityStep: LLM summarization of communities
// - ExtractTemporalGraphStep: temporal variant of ExtractGraph
```

### 2.3. [REFACTOR] `start_cognify.go` — Chain-of-Responsibility

```go
// services/cognee-cognify/internal/usecase/start_cognify.go

package usecase

import (
    "context"
    "fmt"
    "github.com/vnp-memory/services/cognee-cognify/internal/domain"
)

type StartCognifyUseCase struct {
    stepHandlers map[domain.PipelineStep]StepHandler
    runRepo      port.PipelineRunRepository
    publisher    port.EventPublisher
}

func NewStartCognifyUseCase(
    classifyStep *ClassifyStep,
    chunkStep *ChunkStep,
    extractGraphStep *ExtractGraphStep,
    buildOntologyStep *BuildOntologyStep,
    addDatapointsStep *AddDatapointsStep,
    detectCommunityStep *DetectCommunityStep,
    summarizeCommunityStep *SummarizeCommunityStep,
    extractTemporalStep *ExtractTemporalGraphStep,
    runRepo port.PipelineRunRepository,
    publisher port.EventPublisher,
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
    DatasetID  string
    TenantID   string
    EntryIDs   []string
    NodeSets   []string
    Config     domain.PipelineConfig   // [NEW]
}

func (uc *StartCognifyUseCase) Execute(ctx context.Context, req CognifyRequest) (*CognifyResult, error) {
    // Resolve steps from config (backward compatible: empty config → STANDARD template)
    steps := req.Config.Resolve()

    // Validate: ensure all requested steps have handlers
    for _, step := range steps {
        if _, ok := uc.stepHandlers[step]; !ok {
            return nil, fmt.Errorf("unknown pipeline step: %s", step)
        }
    }

    // Create pipeline run record
    runID := newID()
    uc.runRepo.Save(ctx, domain.PipelineRun{
        ID: runID, DatasetID: req.DatasetID, TenantID: req.TenantID,
        Type: "cognify", Status: ✅ Implemented
    })

    // Initialize state
    state := &PipelineState{
        DatasetID: req.DatasetID,
        TenantID:  req.TenantID,
        EntryIDs:  req.EntryIDs,
        NodeSets:  req.NodeSets,
        Options:   req.Config.Options,
    }

    // Execute steps in order (chain-of-responsibility)
    for _, stepName := range steps {
        handler := uc.stepHandlers[stepName]
        var err error
        state, err = handler.Execute(ctx, state)
        if err != nil {
            uc.runRepo.SetStatusWithError(ctx, runID, "FAILED", err.Error())
            return nil, fmt.Errorf("step %s failed: %w", stepName, err)
        }
    }

    uc.runRepo.SetStatus(ctx, runID, "COMPLETED")
    uc.publisher.Publish(ctx, "cognee.pipeline.completed", map[string]any{
        "pipeline_run_id": runID,
        "dataset_id":      req.DatasetID,
        "tenant_id":       req.TenantID,
        "steps_executed":  stepsToStrings(steps),
    })

    return &CognifyResult{PipelineRunID: runID, Status: "COMPLETED", StepsExecuted: steps}, nil
}

// GetTemplates returns available pipeline templates (for REST endpoint)
func (uc *StartCognifyUseCase) GetTemplates() []domain.PipelineTemplateInfo {
    return domain.ListTemplates()
}
```

### 2.4. [MODIFY] Proto — `api/proto/cognee/cognify/v1/cognify.proto`

```protobuf
message StartCognifyRequest {
  string           dataset_id   = 1;
  string           tenant_id    = 2;
  repeated string  entry_ids    = 3;
  repeated string  node_sets    = 4;   // CR-002
  // [NEW] Pipeline config
  string           template     = 5;   // "STANDARD" | "EMBED_ONLY" | "FAST_INDEX" | "TEMPORAL" | "GRAPH_ONLY"
  repeated string  steps        = 6;   // or custom step list
  PipelineOptions  options      = 7;
}

message PipelineOptions {
  int32  chunk_size     = 1;
  string custom_prompt  = 2;
  bool   temporal_mode  = 3;
  bool   skip_cache     = 4;
}

message StartCognifyResponse {
  string         pipeline_run_id = 1;
  string         status          = 2;
  repeated string steps_executed = 3;  // [NEW]
}

// [NEW] List available templates
message GetPipelineTemplatesRequest {}
message GetPipelineTemplatesResponse {
  repeated PipelineTemplateInfo templates = 1;
}
message PipelineTemplateInfo {
  string          name  = 1;
  repeated string steps = 2;
}

service CognifyService {
  rpc StartCognify(StartCognifyRequest) returns (StartCognifyResponse);
  rpc Memify(MemifyRequest) returns (MemifyResponse);                   // CR-001
  rpc GetPipelineStatus(GetPipelineStatusRequest) returns (GetPipelineStatusResponse);  // CR-001
  rpc GetPipelineTemplates(GetPipelineTemplatesRequest) returns (GetPipelineTemplatesResponse);  // [NEW]
}
```

### 2.5. [NEW] Step Files Structure

```
services/cognee-cognify/internal/usecase/steps/
├── classify_step.go         # ContentType detection (no LLM)
├── chunk_step.go            # Sliding window chunker, configurable ChunkSize
├── extract_graph_step.go    # Bifrost LLM entity/relation extraction
├── build_ontology_step.go   # Bifrost LLM ontology inference
├── add_datapoints_step.go   # Neo4j upsert + Qdrant embed
├── detect_community_step.go # Neo4j GDS Louvain algorithm
├── summarize_community_step.go  # Bifrost LLM community summarization
└── extract_temporal_step.go # Temporal variant of extract_graph
```

Each step is injectable (constructor params = deps). Example:

```go
// services/cognee-cognify/internal/usecase/steps/chunk_step.go

type ChunkStep struct {
    defaultSize int  // 512 default
}

func (s *ChunkStep) Name() domain.PipelineStep { return domain.StepChunk }

func (s *ChunkStep) Execute(ctx context.Context, state *PipelineState) (*PipelineState, error) {
    size := s.defaultSize
    if state.Options.ChunkSize > 0 { size = state.Options.ChunkSize }

    var allChunks []Chunk
    for _, content := range state.RawContent {
        allChunks = append(allChunks, slidingWindowChunk(content, size, size/5)...)
    }
    state.Chunks = allChunks
    return state, nil
}
```

### 2.6. [MODIFY] Gateway Routes

```go
// gateway/internal/adapter/handler/router.go

// Existing (updated to pass pipeline config)
r.Post("/api/v1/cognee/cognify", h.ForwardTo("cognee-cognify", "CognifyService/StartCognify"))

// [NEW] List available pipeline templates
r.Get("/api/v1/cognee/pipeline/templates", h.ForwardTo("cognee-cognify", "CognifyService/GetPipelineTemplates"))

// [MODIFY] Console integration (FEAT-015)
r.Get("/v1/console/pipelines/cognee", h.ForwardTo("cognee-cognify", "CognifyService/GetPipelineTemplates"))
```

---

## 3. Files

### [NEW]

| File | Mô tả |
|------|-------|
| `services/cognee-cognify/internal/domain/pipeline.go` | PipelineStep, PipelineConfig, templates |
| `services/cognee-cognify/internal/usecase/step_handler.go` | StepHandler interface + PipelineState |
| `services/cognee-cognify/internal/usecase/steps/classify_step.go` | ClassifyStep |
| `services/cognee-cognify/internal/usecase/steps/chunk_step.go` | ChunkStep |
| `services/cognee-cognify/internal/usecase/steps/extract_graph_step.go` | ExtractGraphStep |
| `services/cognee-cognify/internal/usecase/steps/build_ontology_step.go` | BuildOntologyStep |
| `services/cognee-cognify/internal/usecase/steps/add_datapoints_step.go` | AddDatapointsStep |
| `services/cognee-cognify/internal/usecase/steps/detect_community_step.go` | DetectCommunityStep |
| `services/cognee-cognify/internal/usecase/steps/summarize_community_step.go` | SummarizeCommunityStep |
| `services/cognee-cognify/internal/usecase/steps/extract_temporal_step.go` | ExtractTemporalGraphStep |

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `services/cognee-cognify/internal/usecase/start_cognify.go` | Refactor: chain-of-responsibility |
| `services/cognee-cognify/internal/adapter/grpc/handler.go` | + GetPipelineTemplates() |
| `api/proto/cognee/cognify/v1/cognify.proto` | + template, steps, options fields; + GetPipelineTemplates RPC |
| `gateway/internal/adapter/handler/router.go` | + GET /api/v1/cognee/pipeline/templates |
| `apps/memory/internal/bootstrap/cognee.go` | Inject 8 step handlers into StartCognifyUseCase |

---

## 4. Acceptance Criteria Mapping

| AC từ CR-COGNEE-006 | Covered by |
|--------------------|-----------|
| `template: "EMBED_ONLY"` chỉ chạy CHUNK + ADD_DATAPOINTS | PipelineConfig.Resolve() + templateSteps |
| `steps: ["CLASSIFY","CHUNK","EXTRACT_GRAPH"]` chạy 3 bước đúng thứ tự | Custom steps in Resolve() |
| GET /api/v1/cognee/pipeline/templates → danh sách templates | GetPipelineTemplates() |
| Không truyền template/steps → 7 bước như trước | Resolve() default: TemplateStandard |
| Console FEAT-015 hiển thị pipeline config | /v1/console/pipelines/cognee → GetPipelineTemplates |
| Backward compatible | Empty Config → STANDARD template |
