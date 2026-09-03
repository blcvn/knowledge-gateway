# TASK-GR-009 — Edge Extraction & Temporal Resolution

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-009 |
| **Wave** | 2 (Ingestion & Search) |
| **Component** | `services/graphiti-knowledge/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-003 §6 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-GR-008 |
| **Estimated** | 4h |

---

## Context

Implement edge extraction và resolution use cases. `ResolveEdgeUseCase` quyết định xử lý fact mới như thế nào: DUPLICATE (bỏ qua), NEW (thêm), CONTRADICTION (invalidate cũ + thêm mới), UPDATE (invalidate cũ + thêm mới).

---

## Goal

- `ExtractEdgesUseCase` — parse LLM output thành EntityEdge list với fact embeddings
- `ResolveEdgeUseCase` — DUPLICATE/NEW/CONTRADICTION/UPDATE decision via LLM `dedupe_edges`
- Validate edges against edge type ontology nếu prescribed
- `SummarizeSagaUseCase` — incremental saga summarization

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-knowledge/internal/usecase/extract_edges.go` |
| CREATE | `services/graphiti-knowledge/internal/usecase/resolve_edges.go` |
| CREATE | `services/graphiti-knowledge/internal/usecase/summarize_saga.go` |

---

## Implementation

### File 1: `services/graphiti-knowledge/internal/usecase/extract_edges.go`

```go
package usecase

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/client/llm"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/prompt"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/usecase/port"
)

type ExtractEdgesUseCase struct {
    llm     port.LLMPort
    prompts *prompt.PromptRegistry
}

func NewExtractEdgesUseCase(llm port.LLMPort, prompts *prompt.PromptRegistry) *ExtractEdgesUseCase {
    return &ExtractEdgesUseCase{llm: llm, prompts: prompts}
}

type ExtractEdgesResult struct {
    Edges      []port.ExtractedEdge
    TokenUsage llm.TokenUsage
}

// Execute extracts relationships from content chunks.
// resolvedNodes map[name]uuid is used to match entity names to graph UUIDs.
func (uc *ExtractEdgesUseCase) Execute(ctx context.Context, req port.ExtractEdgesReq, resolvedNodes map[string]string) (*ExtractEdgesResult, error) {
    tmpl := uc.prompts.MustGet("extract_edges")

    userMsg := tmpl.BuildUser(prompt.PromptContext{
        Chunks:        req.Chunks,
        ExistingNodes: req.EntityNames,
        EdgeTypes:     req.EdgeTypes,
        ReferenceTime: req.ReferenceTime,
        Language:      req.Language,
    })

    resp, err := uc.llm.Generate(ctx, "extract_edges", []llm.Message{
        {Role: "system", Content: tmpl.SystemPrompt},
        {Role: "user",   Content: userMsg},
    }, llm.GenerateOpts{
        ResponseSchema: tmpl.Schema,
        PromptName:     "extract_edges",
        ModelSize:      llm.ModelSizeMedium,
        Temperature:    0.0,
    })
    if err != nil { return nil, fmt.Errorf("extract edges LLM: %w", err) }

    var raw struct {
        Edges []struct {
            SourceEntity string  `json:"source_entity"`
            TargetEntity string  `json:"target_entity"`
            RelationType string  `json:"relation_type"`
            Fact         string  `json:"fact"`
            ValidAt      *string `json:"valid_at"`
            InvalidAt    *string `json:"invalid_at"`
        } `json:"edges"`
    }
    if err := json.Unmarshal(resp.Content, &raw); err != nil {
        return &ExtractEdgesResult{TokenUsage: resp.TokenUsage}, nil
    }

    var edges []port.ExtractedEdge
    for _, e := range raw.Edges {
        // Both entities must exist in resolved nodes map
        srcUUID, srcOK := resolvedNodes[normalizeEntityName(e.SourceEntity)]
        tgtUUID, tgtOK := resolvedNodes[normalizeEntityName(e.TargetEntity)]
        if !srcOK || !tgtOK { continue }
        if e.Fact == "" { continue }

        // Validate against edge ontology
        if len(req.EdgeTypes) > 0 {
            if _, ok := req.EdgeTypes[e.RelationType]; !ok { continue }
        }

        // Generate fact embedding
        factEmb, _ := uc.llm.Embed(ctx, e.Fact)

        edges = append(edges, port.ExtractedEdge{
            SourceEntityName: srcUUID,  // now UUID, not name
            TargetEntityName: tgtUUID,
            RelationType:     strings.ToUpper(e.RelationType),
            Fact:             e.Fact,
            FactEmbedding:    factEmb,
            ValidAt:          e.ValidAt,
            InvalidAt:        e.InvalidAt,
        })
    }

    return &ExtractEdgesResult{Edges: edges, TokenUsage: resp.TokenUsage}, nil
}

// ToEntityEdge converts an ExtractedEdge to a graph.EntityEdge with UUID + timestamps
func ToEntityEdge(e port.ExtractedEdge, groupID string, referenceTime time.Time) graph.EntityEdge {
    edge := graph.EntityEdge{
        UUID:           uuid.New().String(),
        SourceNodeUUID: e.SourceEntityName,  // already UUID from extract
        TargetNodeUUID: e.TargetEntityName,
        Name:           e.RelationType,
        Fact:           e.Fact,
        FactEmbedding:  e.FactEmbedding,
        GroupID:        groupID,
        CreatedAt:      time.Now(),
    }

    validAt := referenceTime
    edge.ValidAt = &validAt

    if e.ValidAt != nil {
        if t, err := time.Parse(time.RFC3339, *e.ValidAt); err == nil { edge.ValidAt = &t }
    }
    if e.InvalidAt != nil {
        if t, err := time.Parse(time.RFC3339, *e.InvalidAt); err == nil { edge.InvalidAt = &t }
    }

    return edge
}

func normalizeEntityName(name string) string {
    return strings.ToLower(strings.TrimSpace(name))
}
```

### File 2: `services/graphiti-knowledge/internal/usecase/resolve_edges.go`

```go
package usecase

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/client/llm"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/prompt"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/usecase/port"
)

type ResolveEdgeUseCase struct {
    store   port.StorePort
    llm     port.LLMPort
    prompts *prompt.PromptRegistry
}

func NewResolveEdgeUseCase(store port.StorePort, llm port.LLMPort, prompts *prompt.PromptRegistry) *ResolveEdgeUseCase {
    return &ResolveEdgeUseCase{store: store, llm: llm, prompts: prompts}
}

type ResolveEdgeResult struct {
    Resolution           string   // DUPLICATE | NEW | CONTRADICTION | UPDATE
    InvalidatedEdgeUUIDs []string
    TokenUsage           llm.TokenUsage
}

// Execute resolves a single new edge fact against existing facts.
// Fast paths: exact text match = DUPLICATE; no similar candidates = NEW.
// LLM path: when similar candidates exist (cosine > 0.5).
func (uc *ResolveEdgeUseCase) Execute(ctx context.Context, newEdge graph.EntityEdge, groupID string) (*ResolveEdgeResult, error) {
    var tokenUsage llm.TokenUsage

    // ─── Fast path: no embedding = NEW ────────────────────────────────────
    if len(newEdge.FactEmbedding) == 0 {
        return &ResolveEdgeResult{Resolution: "NEW"}, nil
    }

    // ─── Get similar existing edges from store ────────────────────────────
    similar, err := uc.store.EdgeSimilaritySearch(ctx,
        newEdge.FactEmbedding,
        newEdge.SourceNodeUUID, newEdge.TargetNodeUUID,
        groupID, 10, 0.5,
    )
    if err != nil || len(similar) == 0 {
        return &ResolveEdgeResult{Resolution: "NEW"}, nil
    }

    // ─── Exact text match = DUPLICATE ─────────────────────────────────────
    for _, e := range similar {
        if e.Fact == newEdge.Fact {
            return &ResolveEdgeResult{Resolution: "DUPLICATE"}, nil
        }
    }

    // ─── LLM decision for ambiguous cases ─────────────────────────────────
    tmpl := uc.prompts.MustGet("dedupe_edges")

    existingLines := make([]string, 0, len(similar))
    for _, e := range similar {
        line := fmt.Sprintf("UUID: %s | Fact: %s", e.UUID, e.Fact)
        if e.ValidAt != nil { line += fmt.Sprintf(" | valid_at: %s", e.ValidAt.Format("2006-01-02")) }
        if e.InvalidAt != nil { line += " | [INVALIDATED]" }
        existingLines = append(existingLines, line)
    }

    refTime := ""
    if newEdge.ValidAt != nil { refTime = newEdge.ValidAt.Format("2006-01-02") }

    userMsg := tmpl.BuildUser(prompt.PromptContext{
        Chunks:        []string{newEdge.Fact},
        ExistingNodes: existingLines,
        ReferenceTime: refTime,
    })

    resp, err := uc.llm.Generate(ctx, "dedupe_edges", []llm.Message{
        {Role: "system", Content: tmpl.SystemPrompt},
        {Role: "user",   Content: userMsg},
    }, llm.GenerateOpts{
        ResponseSchema: tmpl.Schema,
        PromptName:     "dedupe_edges",
        ModelSize:      llm.ModelSizeSmall,
        Temperature:    0.0,
    })
    if err != nil {
        // Non-fatal: fallback to NEW on LLM error
        return &ResolveEdgeResult{Resolution: "NEW"}, nil
    }
    tokenUsage = resp.TokenUsage

    var decision struct {
        Resolution           string   `json:"resolution"`
        InvalidatedEdgeUUIDs []string `json:"invalidated_edge_uuids"`
    }
    if err := json.Unmarshal(resp.Content, &decision); err != nil {
        return &ResolveEdgeResult{Resolution: "NEW", TokenUsage: tokenUsage}, nil
    }
    if decision.Resolution == "" { decision.Resolution = "NEW" }

    return &ResolveEdgeResult{
        Resolution:           decision.Resolution,
        InvalidatedEdgeUUIDs: decision.InvalidatedEdgeUUIDs,
        TokenUsage:           tokenUsage,
    }, nil
}
```

### File 3: `services/graphiti-knowledge/internal/usecase/summarize_saga.go`

```go
package usecase

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/client/llm"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/prompt"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/usecase/port"
)

type SummarizeSagaUseCase struct {
    store   port.StorePort
    llm     port.LLMPort
    prompts *prompt.PromptRegistry
}

func NewSummarizeSagaUseCase(store port.StorePort, llm port.LLMPort, prompts *prompt.PromptRegistry) *SummarizeSagaUseCase {
    return &SummarizeSagaUseCase{store: store, llm: llm, prompts: prompts}
}

type SummarizeSagaReq struct {
    SagaID           string
    GroupID          string
    LastSummarizedAt *time.Time  // episodes after this time are "new"
    Episodes         []graph.EpisodicNode
}

type SummarizeSagaResult struct {
    Summary    string
    Title      string
    TokenUsage llm.TokenUsage
}

// Execute generates an incremental LLM summary for a saga.
// Only processes new episodes (created after LastSummarizedAt).
func (uc *SummarizeSagaUseCase) Execute(ctx context.Context, req SummarizeSagaReq) (*SummarizeSagaResult, error) {
    if len(req.Episodes) == 0 {
        return &SummarizeSagaResult{Summary: ""}, nil
    }

    // Filter to only new episodes if LastSummarizedAt is set
    episodes := req.Episodes
    if req.LastSummarizedAt != nil {
        var newEps []graph.EpisodicNode
        for _, ep := range req.Episodes {
            if ep.CreatedAt.After(*req.LastSummarizedAt) {
                newEps = append(newEps, ep)
            }
        }
        if len(newEps) == 0 { return &SummarizeSagaResult{Summary: ""}, nil }
        episodes = newEps
    }

    tmpl := uc.prompts.MustGet("summarize_sagas")

    // Build episode summaries (content truncated to 500 chars each)
    chunks := make([]string, 0, len(episodes))
    for _, ep := range episodes {
        content := ep.Content
        if len(content) > 500 { content = content[:500] + "..." }
        chunks = append(chunks, fmt.Sprintf("[%s] %s", ep.ValidAt.Format("2006-01-02"), content))
    }

    userMsg := tmpl.BuildUser(prompt.PromptContext{
        Chunks:        chunks,
        ReferenceTime: time.Now().Format(time.RFC3339),
    })

    resp, err := uc.llm.Generate(ctx, "summarize_sagas", []llm.Message{
        {Role: "system", Content: tmpl.SystemPrompt},
        {Role: "user",   Content: userMsg},
    }, llm.GenerateOpts{
        ResponseSchema: tmpl.Schema,
        PromptName:     "summarize_sagas",
        ModelSize:      llm.ModelSizeMedium,
        Temperature:    0.1,
    })
    if err != nil { return nil, fmt.Errorf("summarize saga LLM: %w", err) }

    var out struct {
        Summary string `json:"summary"`
        Title   string `json:"title"`
    }
    if err := json.Unmarshal(resp.Content, &out); err != nil {
        return nil, fmt.Errorf("parse saga summary: %w", err)
    }

    return &SummarizeSagaResult{
        Summary:    out.Summary,
        Title:      out.Title,
        TokenUsage: resp.TokenUsage,
    }, nil
}
```

---

## Verification

```bash
cd services/graphiti-knowledge
go build ./internal/usecase/...
```

**Key behaviors:**
1. `ExtractEdgesUseCase`: entity name not in `resolvedNodes` → edge skipped
2. `ResolveEdgeUseCase`: exact text match → `DUPLICATE` (no LLM)
3. `ResolveEdgeUseCase`: no similar candidates → `NEW` (no LLM)
4. `ResolveEdgeUseCase`: CONTRADICTION → returns `InvalidatedEdgeUUIDs`
5. `SummarizeSagaUseCase`: `LastSummarizedAt` set → only processes new episodes
