# Change Request: CR-COGNEE-006 — Custom Pipelines Orchestration

**CR ID:** CR-COGNEE-006  
**Component:** `services/kg-service` | `gateway`  
**Priority:** Low  
**Status:** Implemented  
**Reference:** Cognee PRD §4.4, SRS FR-PIPE-01/FR-PIPE-02, URD UR-DX-04  
**Spec:** `references/cognee/specs/services/03-cognee-cognify.md`

---

## 1. Mô tả

Cho phép người dùng cấu hình và thực thi **Custom Pipeline** bằng cách:
1. Chọn tập hợp con các pipeline steps (skip một số bước).
2. Thay thế một bước bằng custom implementation.
3. Chạy pipeline theo **named template** được pre-defined.

---

## 2. Vấn đề hiện tại

`StartCognify` usecase trong `services/cognee-cognify/internal/usecase/start_cognify.go` thực thi **tất cả 7 pipeline steps** theo thứ tự cứng nhắc:

```
Step 1: ClassifyContent
Step 2: ChunkContent
Step 3: ExtractGraph (LLM)
Step 4: BuildOntology
Step 5: AddDatapoints (Embed)
Step 6: DetectCommunity (optional)
Step 7: Emit PipelineCompleted
```

Không có cơ chế để:
- Skip `ClassifyContent` khi content type đã biết.
- Skip `DetectCommunity` cho datasets nhỏ.
- Chạy chỉ `ChunkContent + AddDatapoints` (embed-only, no graph).
- Inject custom logic giữa các bước.

---

## 3. Thay đổi đề xuất

### 3.1. Service: `services/cognee-cognify`

**[NEW]** `internal/domain/pipeline.go`

```go
// PipelineStep — định danh từng step
type PipelineStep string

const (
    StepClassify         PipelineStep = "CLASSIFY"
    StepChunk            PipelineStep = "CHUNK"
    StepExtractGraph     PipelineStep = "EXTRACT_GRAPH"
    StepBuildOntology    PipelineStep = "BUILD_ONTOLOGY"
    StepAddDatapoints    PipelineStep = "ADD_DATAPOINTS"
    StepDetectCommunity  PipelineStep = "DETECT_COMMUNITY"
    StepSummarizeCommunity PipelineStep = "SUMMARIZE_COMMUNITY"
)

// PipelineConfig — cấu hình pipeline execution
type PipelineConfig struct {
    Steps         []PipelineStep  // Ordered list of steps to execute
    ChunkSize     int             // Override default chunk_size
    CustomPrompt  string          // Override LLM extraction prompt
    TemporalMode  bool            // Use temporal extraction variant
    SkipCache     bool            // Force re-run even if cached
}

// PipelineTemplate — named preset
type PipelineTemplate string

const (
    TemplateStandard     PipelineTemplate = "STANDARD"     // All 7 steps (default)
    TemplateEmbedOnly    PipelineTemplate = "EMBED_ONLY"   // Chunk + AddDatapoints (no LLM)
    TemplateFastIndex    PipelineTemplate = "FAST_INDEX"   // Classify + Chunk + Embed (no graph)
    TemplateTemporal     PipelineTemplate = "TEMPORAL"     // Temporal extraction variant
    TemplateGraphOnly    PipelineTemplate = "GRAPH_ONLY"   // Classify + Chunk + ExtractGraph + Ontology
)
```

**[MODIFY]** `internal/usecase/start_cognify.go`

Refactor từ hardcoded flow sang **chain-of-responsibility**:

```go
type StartCognifyUseCase struct {
    // same deps...
    stepHandlers map[PipelineStep]StepHandler  // Registry of step implementations
}

type StepHandler interface {
    Execute(ctx context.Context, state *PipelineState) (*PipelineState, error)
    Name() PipelineStep
}

// Resolve pipeline steps from PipelineConfig:
func (uc *StartCognifyUseCase) resolveSteps(config PipelineConfig) []StepHandler {
    if config.Template != "" {
        return getTemplateSteps(config.Template)
    }
    // Return only requested steps in order
    steps := []StepHandler{}
    for _, s := range config.Steps {
        if h, ok := uc.stepHandlers[s]; ok {
            steps = append(steps, h)
        }
    }
    return steps
}
```

**[MODIFY]** `internal/adapter/grpc/handler.go`

Cập nhật proto để nhận `PipelineConfig`:
```protobuf
// api/proto/cognee/cognify/v1/cognify.proto
rpc StartCognify(StartCognifyRequest) returns (StartCognifyResponse);

message StartCognifyRequest {
  string           dataset_id    = 1;
  string           tenant_id     = 2;
  PipelineTemplate template      = 3;  // [NEW] use named template
  repeated string  steps         = 4;  // [NEW] or custom steps
  PipelineOptions  options       = 5;  // [NEW]
}

message PipelineTemplate {
  // STANDARD | EMBED_ONLY | FAST_INDEX | TEMPORAL | GRAPH_ONLY
  string name = 1;
}

message PipelineOptions {
  int32  chunk_size     = 1;
  string custom_prompt  = 2;
  bool   temporal_mode  = 3;
}
```

### 3.2. Service: `services/vnp-gateway`

**[MODIFY]** `internal/adapter/http/cognee_routes.go`

Cập nhật `POST /api/v1/cognee/cognify`:
```json
{
  "dataset_id": "uuid",
  "template": "FAST_INDEX"
}
```

hoặc custom steps:
```json
{
  "dataset_id": "uuid",
  "steps": ["CLASSIFY", "CHUNK", "ADD_DATAPOINTS"],
  "options": {
    "chunk_size": 512
  }
}
```

**[NEW]** `GET /api/v1/cognee/pipeline/templates` — list available templates:
```json
{
  "templates": [
    {"name": "STANDARD",   "steps": ["CLASSIFY","CHUNK","EXTRACT_GRAPH","BUILD_ONTOLOGY","ADD_DATAPOINTS","DETECT_COMMUNITY","SUMMARIZE_COMMUNITY"]},
    {"name": "EMBED_ONLY", "steps": ["CHUNK","ADD_DATAPOINTS"]},
    {"name": "FAST_INDEX", "steps": ["CLASSIFY","CHUNK","ADD_DATAPOINTS"]},
    {"name": "TEMPORAL",   "steps": ["CLASSIFY","CHUNK","EXTRACT_TEMPORAL_GRAPH","ADD_DATAPOINTS"]},
    {"name": "GRAPH_ONLY", "steps": ["CLASSIFY","CHUNK","EXTRACT_GRAPH","BUILD_ONTOLOGY"]}
  ]
}
```

---

## 4. Traceability

| Item | Ref |
|---|---|
| Modified file | `services/cognee-cognify/internal/usecase/start_cognify.go` |
| New file | `services/cognee-cognify/internal/domain/pipeline.go` |
| gRPC port | `cognee-cognify:9012` |
| Proto change | `StartCognifyRequest` — thêm `template` + `steps` + `options` |
| New REST route | `GET /api/v1/cognee/pipeline/templates` |
| Console integration | `GET /v1/console/pipelines/cognee` (đã có trong architecture.md) |

---

## 5. Acceptance Criteria

- [x] `POST /api/v1/cognee/cognify` với `template: "EMBED_ONLY"` gửi cấu hình custom pipeline tới Python service.
- [x] `POST /api/v1/cognee/cognify` với `steps: ["CLASSIFY","CHUNK","EXTRACT_GRAPH"]` chạy đúng 3 bước theo thứ tự (do Python xử lý).
- [x] `PipelineConfig` bao gồm cả `ChunkSize`, `TemporalMode`, `CustomPrompt` được truyền thành công qua gRPC proxy.
- [x] Pipeline mặc định (không truyền `template` hay `steps`) vẫn chạy đủ 7 bước như trước.
- [x] Backward compatible: API clients không truyền `template/steps` sẽ có behavior không thay đổi.

---

## 6. Implementation Notes

**Implemented in:** `services/kg-service` + `gateway` (MERGE-P2-T2)
Implementation được thực hiện dưới dạng passthrough config cho Python service.

| File | Change |
|------|--------|
| `services/kg-service/internal/domain/cognee/entity.go` | `[NEW]` `PipelineTemplate`, `PipelineConfig` |
| `services/kg-service/internal/adapter/cognee/client.go` | `[MODIFY]` `Cognify()` truyền `PipelineConfig` trong payload |
| `services/kg-service/internal/usecase/cognee/service.go` | `[MODIFY]` `CognifyUseCase.Cognify()` |
| `services/kg-service/internal/adapter/grpc/router.go` | `[MODIFY]` `Cognify` handler parse `PipelineConfig` từ body |
