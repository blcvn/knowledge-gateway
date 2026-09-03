---
id: FEAT-KNW-007
title: Prompt Registry — 7 Extraction/Resolution Templates
service: graphiti-knowledge
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement prompt template registry cho 7 LLM extraction/resolution tasks. Bao gồm template loading, variable interpolation, và model selection per template.

## Scope

- `internal/adapter/llm/prompt_registry.go` — Template management
- 7 prompt templates with Go text/template syntax

### Templates

| Template ID | Purpose | Model | Vars |
|------------|---------|-------|------|
| `extract_entities` | Extract named entities from content | gpt-4o | `{{.Content}}`, `{{.PreviousEpisodes}}`, `{{.EntityTypes}}` |
| `resolve_entities` | Compare extracted vs existing for dedup | gpt-4o-mini | `{{.Extracted}}`, `{{.Existing}}` |
| `extract_edges` | Extract fact triples with temporal info | gpt-4o | `{{.Content}}`, `{{.Entities}}`, `{{.PreviousEpisodes}}` |
| `resolve_edges` | Detect contradictions between edges | gpt-4o-mini | `{{.NewEdge}}`, `{{.ExistingEdges}}` |
| `summarize_community` | Generate community summary | gpt-4o-mini | `{{.Members}}`, `{{.Edges}}` |
| `classify_entity` | Classify entity type from context | gpt-4o-mini | `{{.Entity}}`, `{{.Context}}` |
| `expand_summary` | Expand entity summary with new info | gpt-4o-mini | `{{.Entity}}`, `{{.NewContext}}` |

### Prompt Registry Interface

```go
type PromptRegistry interface {
    Render(templateID string, vars map[string]interface{}) (string, error)
    GetModel(templateID string) string
    List() []PromptTemplate
}
```

## Acceptance Criteria

- [ ] AC-1: All 7 templates render with correct variable interpolation
- [ ] AC-2: Templates return structured JSON instructions to LLM
- [ ] AC-3: Each template has associated model (gpt-4o or gpt-4o-mini)
- [ ] AC-4: Missing template returns ErrPromptNotFound
- [ ] AC-5: Templates are testable without LLM (render-only tests)

## Test Requirements
- **Unit tests**: Render all templates with sample data, verify output structure
- **Minimum coverage**: 90%
