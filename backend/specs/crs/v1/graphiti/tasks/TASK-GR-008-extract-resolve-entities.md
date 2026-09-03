# TASK-GR-008 — Entity Extraction & Resolution Use Cases

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-008 |
| **Wave** | 2 (Ingestion & Search) |
| **Component** | `services/graphiti-knowledge/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-003 §4, §5 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-GR-007 |
| **Estimated** | 5h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** graphiti-pipeline: 39 .go - entity extraction + resolve  
---

## Context

Implement hai use cases cốt lõi của `graphiti-knowledge`:
1. **ExtractEntitiesUseCase** — LLM-based entity extraction từ content chunks (với ontology filtering)
2. **ResolveEntityUseCase** — 2-phase resolution: cosine fast path → LLM disambiguation

---

## Goal

- `ExtractEntitiesUseCase` — parse LLM output, validate vs ontology, embed entity names
- `ResolveEntityUseCase` — Phase 1: exact/cosine fast path; Phase 2: LLM dedupe_nodes
- `ExtractAttributesUseCase` — update entity summaries from new facts
- Port interfaces: `StorePort` (for candidate search), `KnowledgePort`

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-knowledge/internal/usecase/extract_entities.go` |
| CREATE | `services/graphiti-knowledge/internal/usecase/resolve_entities.go` |
| CREATE | `services/graphiti-knowledge/internal/usecase/extract_attributes.go` |
| CREATE | `services/graphiti-knowledge/internal/usecase/port/input.go` |
| CREATE | `services/graphiti-knowledge/internal/usecase/port/output.go` |

---

## Implementation

### File 1: `services/graphiti-knowledge/internal/usecase/port/input.go`

```go
package port

import "github.com/vnp-memory/pkg/graph"

type ExtractEntitiesReq struct {
    Chunks       []string
    PrevEpisodes []string
    EntityTypes  map[string]graph.EntityTypeSchema
    EdgeTypes    map[string]graph.EdgeTypeSchema
    Source       graph.EpisodeType
    Language     string
    GroupID      string
}

type ResolveEntityReq struct {
    EntityName    string
    EntityLabel   string
    EntitySummary string
    NameEmbedding []float32
    Candidates    []*graph.EntityNode  // from store NodeSimilaritySearch
}

type ExtractAttributesReq struct {
    EntityName    string
    ExistingSummary string
    NewFacts      []string
}

type ExtractEdgesReq struct {
    Chunks        []string
    EntityNames   []string           // resolved entity names
    EdgeTypes     map[string]graph.EdgeTypeSchema
    ReferenceTime string
    Language      string
}

type ResolveEdgeReq struct {
    NewFact        string
    NewFactEmb     []float32
    ExistingEdges  []*graph.EntityEdge  // from store EdgeSimilaritySearch
    ReferenceTime  string
}
```

### File 2: `services/graphiti-knowledge/internal/usecase/port/output.go`

```go
package port

import (
    "context"

    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/client/llm"
)

// ExtractedEntity — raw LLM extraction result
type ExtractedEntity struct {
    Name          string
    Label         string
    Summary       string
    NameEmbedding []float32  // populated by ExtractEntitiesUseCase after embedding
}

// ExtractedEdge — raw LLM extraction result for a relationship
type ExtractedEdge struct {
    SourceEntityName string
    TargetEntityName string
    RelationType     string
    Fact             string
    FactEmbedding    []float32
    ValidAt          *string   // ISO8601 or nil
    InvalidAt        *string   // ISO8601 or nil
}

// EntityResolution — outcome of two-phase entity resolution
type EntityResolution struct {
    Decision     string  // "merge" | "new"
    ExistingUUID string  // populated when Decision=="merge"
}

// EdgeResolution — outcome of edge resolution
type EdgeResolution struct {
    Resolution           string    // DUPLICATE | NEW | CONTRADICTION | UPDATE
    InvalidatedEdgeUUIDs []string
}

// StorePort — operations that knowledge service needs from graphiti-store
type StorePort interface {
    NodeSimilaritySearch(ctx context.Context, vector []float32, groupID string, limit int, minScore float64) ([]*graph.EntityNode, error)
    NodeFulltextSearch(ctx context.Context, query, groupID string, limit int) ([]*graph.EntityNode, error)
    EdgeSimilaritySearch(ctx context.Context, vector []float32, srcUUID, tgtUUID, groupID string, limit int, minScore float64) ([]*graph.EntityEdge, error)
    GetEntityNode(ctx context.Context, uuid string) (*graph.EntityNode, error)
    GetCommunityClusters(ctx context.Context, groupID string) ([][]string, error)
    GetEntityNodes(ctx context.Context, uuids []string) ([]*graph.EntityNode, error)
    SaveCommunityNode(ctx context.Context, node graph.CommunityNode) error
    SaveCommunityEdge(ctx context.Context, edge graph.CommunityEdge) error
    RemoveCommunities(ctx context.Context, groupID string) error
}

// LLMPort — LLM operations used within knowledge service
type LLMPort interface {
    Generate(ctx context.Context, promptName string, msgs []llm.Message, opts llm.GenerateOpts) (*llm.LLMResponse, error)
    Embed(ctx context.Context, text string) ([]float32, error)
    Rerank(ctx context.Context, query string, passages []string) ([]float64, error)
}
```

### File 3: `services/graphiti-knowledge/internal/usecase/extract_entities.go`

```go
package usecase

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"

    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/client/llm"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/prompt"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/usecase/port"
)

type ExtractEntitiesUseCase struct {
    llm     port.LLMPort
    prompts *prompt.PromptRegistry
}

func NewExtractEntitiesUseCase(llm port.LLMPort, prompts *prompt.PromptRegistry) *ExtractEntitiesUseCase {
    return &ExtractEntitiesUseCase{llm: llm, prompts: prompts}
}

type ExtractedEntitiesResult struct {
    Entities   []port.ExtractedEntity
    TokenUsage llm.TokenUsage
}

func (uc *ExtractEntitiesUseCase) Execute(ctx context.Context, req port.ExtractEntitiesReq) (*ExtractedEntitiesResult, error) {
    tmpl := uc.prompts.MustGet("extract_nodes")

    userMsg := tmpl.BuildUser(prompt.PromptContext{
        Chunks:       req.Chunks,
        PrevEpisodes: req.PrevEpisodes,
        EntityTypes:  req.EntityTypes,
        Source:       string(req.Source),
        Language:     req.Language,
    })

    resp, err := uc.llm.Generate(ctx, "extract_nodes", []llm.Message{
        {Role: "system", Content: tmpl.SystemPrompt},
        {Role: "user",   Content: userMsg},
    }, llm.GenerateOpts{
        ResponseSchema: tmpl.Schema,
        PromptName:     "extract_nodes",
        ModelSize:      llm.ModelSizeMedium,
        Temperature:    0.0,
    })
    if err != nil { return nil, fmt.Errorf("extract entities LLM: %w", err) }

    // Parse LLM response
    var raw struct {
        Entities []struct {
            Name    string `json:"name"`
            Label   string `json:"label"`
            Summary string `json:"summary"`
        } `json:"entities"`
    }
    if err := json.Unmarshal(resp.Content, &raw); err != nil {
        return nil, fmt.Errorf("parse extract_nodes response: %w", err)
    }

    // Deduplicate by normalized name + validate against ontology
    seen := make(map[string]bool)
    var entities []port.ExtractedEntity

    for _, e := range raw.Entities {
        e.Name = strings.TrimSpace(e.Name)
        if e.Name == "" { continue }

        key := strings.ToLower(e.Name)
        if seen[key] { continue }
        seen[key] = true

        // Validate against prescribed ontology
        if len(req.EntityTypes) > 0 {
            if _, ok := req.EntityTypes[e.Label]; !ok { continue }
        }

        // Generate name embedding
        emb, err := uc.llm.Embed(ctx, e.Name)
        if err != nil { emb = nil }  // non-fatal: continue without embedding

        entities = append(entities, port.ExtractedEntity{
            Name:          e.Name,
            Label:         e.Label,
            Summary:       e.Summary,
            NameEmbedding: emb,
        })
    }

    return &ExtractedEntitiesResult{
        Entities:   entities,
        TokenUsage: resp.TokenUsage,
    }, nil
}
```

### File 4: `services/graphiti-knowledge/internal/usecase/resolve_entities.go`

```go
package usecase

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"

    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/client/llm"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/prompt"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/usecase/port"
)

const (
    exactMatchThreshold   = 0.98  // cosine similarity threshold for fast-path
    noLLMThreshold        = 0.95  // if no candidate is this similar, skip LLM
)

type ResolveEntityUseCase struct {
    store   port.StorePort
    llm     port.LLMPort
    prompts *prompt.PromptRegistry
}

func NewResolveEntityUseCase(store port.StorePort, llm port.LLMPort, prompts *prompt.PromptRegistry) *ResolveEntityUseCase {
    return &ResolveEntityUseCase{store: store, llm: llm, prompts: prompts}
}

// Execute resolves a single extracted entity against existing graph nodes.
// Returns (decision, tokenUsage, error).
// Decision.ExistingUUID is empty if decision is "new".
func (uc *ResolveEntityUseCase) Execute(ctx context.Context, req port.ResolveEntityReq) (*port.EntityResolution, llm.TokenUsage, error) {
    var tokenUsage llm.TokenUsage

    // ─── Phase 1: Deterministic fast path ─────────────────────────────────
    if len(req.Candidates) == 0 {
        return &port.EntityResolution{Decision: "new"}, tokenUsage, nil
    }

    // Exact name match (case-insensitive)
    normalizedNew := strings.ToLower(strings.TrimSpace(req.EntityName))
    for _, candidate := range req.Candidates {
        if strings.ToLower(candidate.Name) == normalizedNew {
            return &port.EntityResolution{Decision: "merge", ExistingUUID: candidate.UUID}, tokenUsage, nil
        }
    }

    // High cosine similarity fast path (≥0.98 = very likely same entity)
    if len(req.NameEmbedding) > 0 {
        for _, candidate := range req.Candidates {
            if len(candidate.NameEmbedding) == 0 { continue }
            sim := cosineSimilarity(req.NameEmbedding, candidate.NameEmbedding)
            if sim >= exactMatchThreshold {
                return &port.EntityResolution{Decision: "merge", ExistingUUID: candidate.UUID}, tokenUsage, nil
            }
        }

        // If best candidate similarity < noLLMThreshold, skip LLM entirely
        bestSim := 0.0
        for _, c := range req.Candidates {
            if len(c.NameEmbedding) == 0 { continue }
            sim := cosineSimilarity(req.NameEmbedding, c.NameEmbedding)
            if sim > bestSim { bestSim = sim }
        }
        if bestSim < 0.6 {
            return &port.EntityResolution{Decision: "new"}, tokenUsage, nil
        }
    }

    // ─── Phase 2: LLM disambiguation ──────────────────────────────────────
    tmpl := uc.prompts.MustGet("dedupe_nodes")

    // Format candidates for LLM
    candidateLines := make([]string, 0, len(req.Candidates))
    for _, c := range req.Candidates {
        line := fmt.Sprintf("UUID: %s | Name: %s", c.UUID, c.Name)
        if c.Summary != "" { line += fmt.Sprintf(" | Summary: %s", c.Summary) }
        candidateLines = append(candidateLines, line)
    }

    userMsg := tmpl.BuildUser(prompt.PromptContext{
        Chunks:        []string{req.EntityName, req.EntitySummary},
        ExistingNodes: candidateLines,
        Source:        req.EntityLabel,
    })

    resp, err := uc.llm.Generate(ctx, "dedupe_nodes", []llm.Message{
        {Role: "system", Content: tmpl.SystemPrompt},
        {Role: "user",   Content: userMsg},
    }, llm.GenerateOpts{
        ResponseSchema: tmpl.Schema,
        PromptName:     "dedupe_nodes",
        ModelSize:      llm.ModelSizeSmall,  // cheap model for resolution
        Temperature:    0.0,
    })
    if err != nil {
        // Non-fatal: fallback to "new" on LLM error
        return &port.EntityResolution{Decision: "new"}, tokenUsage, nil
    }
    tokenUsage = resp.TokenUsage

    var decision struct {
        Decision     string `json:"decision"`
        ExistingUUID string `json:"existing_uuid"`
    }
    if err := json.Unmarshal(resp.Content, &decision); err != nil {
        return &port.EntityResolution{Decision: "new"}, tokenUsage, nil
    }
    if decision.Decision == "" { decision.Decision = "new" }

    return &port.EntityResolution{
        Decision:     decision.Decision,
        ExistingUUID: decision.ExistingUUID,
    }, tokenUsage, nil
}

// cosineSimilarity computes cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
    if len(a) != len(b) || len(a) == 0 { return 0 }
    var dot, normA, normB float64
    for i := range a {
        dot  += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }
    if normA == 0 || normB == 0 { return 0 }
    return dot / (sqrtF(normA) * sqrtF(normB))
}

func sqrtF(x float64) float64 {
    // Newton-Raphson sqrt
    if x <= 0 { return 0 }
    z := x
    for i := 0; i < 10; i++ { z -= (z*z - x) / (2 * z) }
    return z
}
```

### File 5: `services/graphiti-knowledge/internal/usecase/extract_attributes.go`

```go
package usecase

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/client/llm"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/prompt"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/usecase/port"
)

type ExtractAttributesUseCase struct {
    llm     port.LLMPort
    prompts *prompt.PromptRegistry
}

func NewExtractAttributesUseCase(llm port.LLMPort, prompts *prompt.PromptRegistry) *ExtractAttributesUseCase {
    return &ExtractAttributesUseCase{llm: llm, prompts: prompts}
}

// Execute updates entity summaries given new facts extracted from the episode.
// Returns a map[entityName]updatedSummary.
func (uc *ExtractAttributesUseCase) Execute(ctx context.Context, reqs []port.ExtractAttributesReq) (map[string]string, llm.TokenUsage, error) {
    var total llm.TokenUsage
    result := make(map[string]string, len(reqs))

    tmpl := uc.prompts.MustGet("summarize_nodes")

    for _, req := range reqs {
        if len(req.NewFacts) == 0 { continue }

        userMsg := tmpl.BuildUser(prompt.PromptContext{
            Chunks:        []string{req.EntityName, req.ExistingSummary},
            ExistingNodes: req.NewFacts,
        })

        resp, err := uc.llm.Generate(ctx, "summarize_nodes", []llm.Message{
            {Role: "system", Content: tmpl.SystemPrompt},
            {Role: "user",   Content: userMsg},
        }, llm.GenerateOpts{
            ResponseSchema: tmpl.Schema,
            PromptName:     "summarize_nodes",
            ModelSize:      llm.ModelSizeSmall,
            Temperature:    0.0,
        })
        if err != nil {
            // Non-fatal: skip summary update on error
            continue
        }
        total.Add(resp.TokenUsage)

        var out struct {
            Summary string `json:"summary"`
        }
        if err := json.Unmarshal(resp.Content, &out); err != nil { continue }
        if out.Summary != "" {
            result[req.EntityName] = out.Summary
        }
    }

    return result, total, nil
}
```

---

## Verification

```bash
cd services/graphiti-knowledge
go build ./internal/usecase/...
go vet ./internal/usecase/...
```

**Key behaviors to verify manually:**
1. Exact name match → `decision: merge` (no LLM call)
2. Cosine ≥ 0.98 → `decision: merge` (no LLM call)
3. Low similarity candidates → `decision: new` (no LLM call)
4. Ambiguous case → LLM call with `dedupe_nodes` prompt
5. LLM error → graceful fallback to `decision: new`
