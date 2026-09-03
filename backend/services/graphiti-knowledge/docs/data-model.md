---
id: DOC-S04
service: graphiti-knowledge
version: 2.0.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# graphiti-knowledge — Data Model

> **Group**: Graphiti | **Storage**: Stateless (no own DB) | **Upstream**: graphiti-store (read), Bifrost (LLM)

## Domain Types

graphiti-knowledge is a **stateless** LLM processing engine. It does not persist data — reads from graphiti-store for entity resolution, and all processed output is returned to the caller (graphiti-pipeline or graphiti-ingestion).

### Extracted Entity

```go
type ExtractedEntity struct {
    Name    string `json:"name"`    // Entity name extracted from content
    Label   string `json:"label"`   // Entity type: Person, Organization, Concept, etc.
    Summary string `json:"summary"` // Brief description from extraction context
}
```

### Extracted Edge (Fact Triple)

```go
type ExtractedEdge struct {
    SourceEntity string     `json:"source_entity"` // Subject entity name
    TargetEntity string     `json:"target_entity"` // Object entity name
    Relationship string     `json:"relationship"`  // Predicate/edge name
    Fact         string     `json:"fact"`           // Full factual statement
    ValidAt      *time.Time `json:"valid_at"`       // When fact became true (from context)
    InvalidAt    *time.Time `json:"invalid_at"`     // When fact stopped being true (if known)
}
```

### Resolution Decision

```go
type Resolution struct {
    ExistingEntityID string            `json:"existing_entity_id"`
    ExtractedEntity  ExtractedEntity   `json:"extracted_entity"`
    Decision         DuplicateDecision `json:"decision"`     // merge | create | skip
    Confidence       float64           `json:"confidence"`   // 0.0 - 1.0
    MergedSummary    string            `json:"merged_summary"` // Updated summary after merge
}
```

### Token Usage Tracking

```go
type TokenUsage struct {
    Model            string `json:"model"`
    PromptTokens     int    `json:"prompt_tokens"`
    CompletionTokens int    `json:"completion_tokens"`
    TotalTokens      int    `json:"total_tokens"`
    CostUSD          float64 `json:"cost_usd"`
}
```

### Prompt Templates

| Template ID | Input Variables | Output Schema | Model |
|------------|----------------|---------------|-------|
| `extract_entities` | Content, PreviousEpisodes, EntityTypes | `[{name, label, summary}]` | gpt-4o |
| `resolve_entities` | Extracted, Existing | `{decision, confidence, merged_summary}` | gpt-4o-mini |
| `extract_edges` | Content, Entities, PreviousEpisodes | `[{source, target, relationship, fact, valid_at}]` | gpt-4o |
| `resolve_edges` | NewEdge, ExistingEdges | `{action, invalidate_ids[], reason}` | gpt-4o-mini |
| `summarize_community` | Members[], Edges[] | `{summary}` | gpt-4o-mini |
| `classify_entity` | Entity, Context | `{label, confidence}` | gpt-4o-mini |
| `expand_summary` | Entity, NewContext | `{updated_summary}` | gpt-4o-mini |

## No Persistent Storage

graphiti-knowledge does not own any PostgreSQL tables or Neo4j labels. All graph data is read from graphiti-store via gRPC.
