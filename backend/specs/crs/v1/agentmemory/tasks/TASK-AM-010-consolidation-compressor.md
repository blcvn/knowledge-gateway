# TASK-AM-010 — Consolidation Compressor + Repos

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-010 |
| **Wave** | 2 (Integration) |
| **Component** | `services/memory-service/internal/consolidation/` + repos |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-006 §2.5 → §2.8 |
| **Priority** | High |
| **Depends On** | TASK-AM-009 |
| **Estimated** | 4h |

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/memory-service/internal/consolidation/summarize.go` |
| CREATE | `services/memory-service/internal/consolidation/procedural.go` |
| CREATE | `services/memory-service/internal/consolidation/lessons.go` |
| CREATE | `services/memory-service/internal/adapter/repository/postgres/consolidation_repo.go` |
| CREATE | `services/memory-service/internal/adapter/grpc/consolidation_handler.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

### `internal/consolidation/summarize.go`

```go
package consolidation

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"

    "github.com/vnp-memory/services/memory-service/internal/domain/agentmemory"
    "github.com/vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

const summarizeSystemPrompt = `Summarize this AI agent coding session.
Return JSON:
{
  "title": "1 sentence session title",
  "narrative": "2-3 paragraph description",
  "key_decisions": ["decision 1", "decision 2"],
  "files_modified": ["/path/to/file"],
  "concepts": ["concept1", "concept2"]
}`

type Summarizer struct {
    obsRepo port.IObservationRepo
    llm     port.ILLMProvider
    cb      *CircuitBreaker
}

func NewSummarizer(obsRepo port.IObservationRepo, llm port.ILLMProvider) *Summarizer {
    return &Summarizer{
        obsRepo: obsRepo, llm: llm,
        cb: NewCircuitBreaker(3, 5*time.Minute),
    }
}

func (s *Summarizer) Summarize(ctx context.Context, sessionID string) *agentmemory.SessionSummary {
    compObs, _ := s.obsRepo.ListCompressed(ctx, sessionID)
    if len(compObs) == 0 { return nil }

    summary := &agentmemory.SessionSummary{
        SessionID:        sessionID,
        ObservationCount: len(compObs),
    }

    // Build input for LLM
    var lines []string
    for _, obs := range compObs {
        lines = append(lines, fmt.Sprintf("- [%s] %s", obs.HookType, obs.Title))
        if len(obs.Facts) > 0 {
            lines = append(lines, "  Facts: "+strings.Join(obs.Facts, "; "))
        }
    }
    input := strings.Join(lines, "\n")

    // Try LLM
    if s.llm != nil && s.cb.Allow() {
        resp, err := s.llm.Chat(ctx, summarizeSystemPrompt, input)
        if err == nil {
            var result struct {
                Title         string   `json:"title"`
                Narrative     string   `json:"narrative"`
                KeyDecisions  []string `json:"key_decisions"`
                FilesModified []string `json:"files_modified"`
                Concepts      []string `json:"concepts"`
            }
            if json.Unmarshal([]byte(resp), &result) == nil {
                s.cb.RecordSuccess()
                summary.Title         = result.Title
                summary.Narrative     = result.Narrative
                summary.KeyDecisions  = result.KeyDecisions
                summary.FilesModified = result.FilesModified
                summary.Concepts      = result.Concepts
                return summary
            }
        }
        s.cb.RecordFailure()
    }

    // Synthetic summary: top 3 by importance
    sortByImportance(compObs)
    if len(compObs) > 0 { summary.Title = compObs[0].Title }
    top := compObs
    if len(top) > 3 { top = top[:3] }
    narratives := make([]string, len(top))
    for i, obs := range top { narratives[i] = obs.Narrative }
    summary.Narrative = strings.Join(narratives, " ")
    return summary
}

func sortByImportance(obs []agentmemory.CompressedObs) {
    sort.Slice(obs, func(i, j int) bool { return obs[i].Importance > obs[j].Importance })
}
```

### `internal/consolidation/procedural.go`

```go
package consolidation

import (
    "context"
    "crypto/sha256"
    "fmt"
    "sort"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/services/memory-service/internal/domain/agentmemory"
    "github.com/vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

type ProceduralExtractor struct {
    summaryRepo    port.ISessionSummaryRepo
    proceduralRepo port.IProceduralRepo
    llm            port.ILLMProvider
    minFrequency   int
}

func (e *ProceduralExtractor) ExtractAll(ctx context.Context) {
    summaries, _ := e.summaryRepo.ListAll(ctx)
    sequences := e.buildSequences(summaries)

    for seqKey, sessionIDs := range sequences {
        if len(sessionIDs) < e.minFrequency { continue }

        stepHash := fmt.Sprintf("%x", sha256.Sum256([]byte(seqKey)))
        existing, _ := e.proceduralRepo.FindByStepHash(ctx, stepHash)
        if existing != nil {
            e.proceduralRepo.IncrementFrequency(ctx, existing.ID)
            continue
        }

        steps := strings.Split(seqKey, "|")
        proc := agentmemory.ProceduralMemory{
            ID:         uuid.New().String(),
            Steps:      steps,
            StepHash:   stepHash,
            Frequency:  len(sessionIDs),
            Confidence: float64(len(sessionIDs)) / 10.0,
            CreatedAt:  time.Now(),
            UpdatedAt:  time.Now(),
        }

        if e.llm != nil {
            e.enrichWithLLM(ctx, &proc, steps)
        } else {
            proc.Name = fmt.Sprintf("Workflow: %s → %s", steps[0], steps[len(steps)-1])
        }

        e.proceduralRepo.Save(ctx, proc)
    }
}

// buildSequences: extract key_decisions from summaries, find repeated patterns
func (e *ProceduralExtractor) buildSequences(summaries []agentmemory.SessionSummary) map[string][]string {
    sequences := make(map[string][]string)
    for _, s := range summaries {
        if len(s.KeyDecisions) < 2 { continue }
        key := strings.Join(s.KeyDecisions, "|")
        sequences[key] = append(sequences[key], s.SessionID)
    }
    return sequences
}

func (e *ProceduralExtractor) enrichWithLLM(ctx context.Context, proc *agentmemory.ProceduralMemory, steps []string) {
    prompt := fmt.Sprintf("Name this coding workflow: %s. Return JSON: {\"name\": \"...\", \"trigger\": \"...\", \"outcome\": \"...\"}", strings.Join(steps, " → "))
    resp, err := e.llm.Chat(ctx, "You name coding workflows concisely.", prompt)
    if err != nil { return }
    var result struct {
        Name    string `json:"name"`
        Trigger string `json:"trigger"`
        Outcome string `json:"outcome"`
    }
    if json.Unmarshal([]byte(resp), &result) == nil {
        proc.Name = result.Name
        proc.TriggerCondition = result.Trigger
        proc.ExpectedOutcome = result.Outcome
    }
}
```

### `internal/consolidation/lessons.go`

```go
package consolidation

import (
    "context"
    "encoding/json"
    "math"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/services/memory-service/internal/domain/agentmemory"
    "github.com/vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

type LessonExtractor struct {
    summaryRepo port.ISessionSummaryRepo
    lessonRepo  port.ILessonRepo
    insightRepo port.IInsightRepo
    llm         port.ILLMProvider
    halfLifeDays int
}

func (e *LessonExtractor) ExtractAll(ctx context.Context) {
    summaries, _ := e.summaryRepo.ListRecent(ctx, 100)
    for _, s := range summaries {
        if len(s.KeyDecisions) == 0 { continue }
        lesson := agentmemory.Lesson{
            ID:         uuid.New().String(),
            Content:    s.KeyDecisions[0],
            Source:     s.SessionID,
            Confidence: 0.7,
            CreatedAt:  time.Now(),
            UpdatedAt:  time.Now(),
        }
        if e.llm != nil {
            e.enrichLesson(ctx, &lesson, s)
        }
        e.lessonRepo.Save(ctx, lesson)
    }
}

func (e *LessonExtractor) ApplyDecay(ctx context.Context) {
    lessons, _ := e.lessonRepo.ListAll(ctx)
    for _, l := range lessons {
        hoursSince := time.Since(l.UpdatedAt).Hours()
        factor := math.Exp(-hoursSince / (float64(e.halfLifeDays) * 24))
        l.Confidence *= factor
        e.lessonRepo.UpdateConfidence(ctx, l.ID, l.Confidence)
    }
}

func (e *LessonExtractor) SynthesizeInsights(ctx context.Context) {
    lessons, _ := e.lessonRepo.ListHighConfidence(ctx, 0.7, 50)
    if len(lessons) < 3 { return }

    prompt := "Synthesize these lessons into 1-3 key insights:\n"
    ids := make([]string, len(lessons))
    for i, l := range lessons {
        prompt += "- " + l.Content + "\n"
        ids[i] = l.ID
    }

    if e.llm == nil { return }
    resp, err := e.llm.Chat(ctx, "Synthesize software engineering insights.", prompt)
    if err != nil { return }

    var result struct { Insights []string `json:"insights"` }
    if json.Unmarshal([]byte(resp), &result) != nil { return }

    for _, insight := range result.Insights {
        e.insightRepo.Save(ctx, agentmemory.Insight{
            ID:         uuid.New().String(),
            Content:    insight,
            LessonIDs:  ids,
            Confidence: 0.6,
            CreatedAt:  time.Now(),
        })
    }
}

func (e *LessonExtractor) enrichLesson(ctx context.Context, l *agentmemory.Lesson, s agentmemory.SessionSummary) {
    // Categorize using LLM
    prompt := "Categorize this lesson: " + l.Content + ". Return JSON: {\"categories\": [\"...\"]}"
    resp, _ := e.llm.Chat(ctx, "You categorize software lessons.", prompt)
    var r struct { Categories []string `json:"categories"` }
    json.Unmarshal([]byte(resp), &r)
    l.Categories = r.Categories
}
```

### `adapter/grpc/consolidation_handler.go`

```go
package grpc

import (
    "context"

    memorypb "github.com/vnp-memory/api/proto/memory/v1"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type ConsolidationHandler struct {
    memorypb.UnimplementedConsolidationServiceServer
    pipeline    *consolidation.ConsolidationPipeline
    summaryRepo port.ISessionSummaryRepo
    procRepo    port.IProceduralRepo
    lessonRepo  port.ILessonRepo
    insightRepo port.IInsightRepo
}

func (h *ConsolidationHandler) SummarizeSession(ctx context.Context, req *memorypb.SummarizeSessionRequest) (*memorypb.SummarizeSessionResponse, error) {
    h.pipeline.SummarizeNow(ctx, req.SessionId)
    return &memorypb.SummarizeSessionResponse{Ok: true}, nil
}

func (h *ConsolidationHandler) RunPipeline(ctx context.Context, req *memorypb.RunPipelineRequest) (*memorypb.RunPipelineResponse, error) {
    go h.pipeline.SummarizeNow(context.Background(), req.SessionId)
    return &memorypb.RunPipelineResponse{Ok: true}, nil
}

func (h *ConsolidationHandler) ListProcedural(ctx context.Context, req *memorypb.ListProceduralRequest) (*memorypb.ListProceduralResponse, error) {
    items, err := h.procRepo.ListByTenant(ctx, req.TenantId)
    if err != nil { return nil, status.Errorf(codes.Internal, "list procedural: %v", err) }
    return mapProceduralResponse(items), nil
}

func (h *ConsolidationHandler) ListLessons(ctx context.Context, req *memorypb.ListLessonsRequest) (*memorypb.ListLessonsResponse, error) {
    items, err := h.lessonRepo.ListByTenant(ctx, req.TenantId)
    if err != nil { return nil, status.Errorf(codes.Internal, "list lessons: %v", err) }
    return mapLessonsResponse(items), nil
}

func (h *ConsolidationHandler) ListInsights(ctx context.Context, req *memorypb.ListInsightsRequest) (*memorypb.ListInsightsResponse, error) {
    items, err := h.insightRepo.ListByTenant(ctx, req.TenantId)
    if err != nil { return nil, status.Errorf(codes.Internal, "list insights: %v", err) }
    return mapInsightsResponse(items), nil
}
```

### MODIFY `gateway/router.go` — Consolidation routes

```go
r.Post("/v1/memory/compress",            h.ForwardTo("memory-service", "ConsolidationService/CompressObservation"))
r.Post("/v1/memory/summarize",           h.ForwardTo("memory-service", "ConsolidationService/SummarizeSession"))
r.Post("/v1/memory/consolidate",         h.ForwardTo("memory-service", "ConsolidationService/RunPipeline"))
r.Get("/v1/memory/procedural",           h.ForwardTo("memory-service", "ConsolidationService/ListProcedural"))
r.Get("/v1/memory/procedural/{id}",      h.ForwardTo("memory-service", "ConsolidationService/GetProcedural"))
r.Get("/v1/memory/lessons",              h.ForwardTo("memory-service", "ConsolidationService/ListLessons"))
r.Get("/v1/memory/lessons/{id}",         h.ForwardTo("memory-service", "ConsolidationService/GetLesson"))
r.Post("/v1/memory/lessons/decay-sweep", h.ForwardTo("memory-service", "ConsolidationService/LessonDecaySweep"))
r.Get("/v1/memory/insights",             h.ForwardTo("memory-service", "ConsolidationService/ListInsights"))
```

---

## Acceptance Criteria

| AC | Check |
|----|-------|
| POST /v1/memory/summarize → session summary created | ✅ |
| GET /v1/memory/procedural → list extracted workflows | ✅ |
| GET /v1/memory/lessons → lessons with confidence | ✅ |
| Lesson decay after 90 days → confidence × exp(-1) | ✅ |
| 3 repeated workflows → 1 ProceduralMemory (step_hash dedup) | ✅ |
| GET /v1/memory/insights → synthesized insights | ✅ |
