# TASK-GR-007 — Prompt Registry (6 Templates)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-007 |
| **Wave** | 2 (Ingestion & Search) |
| **Component** | `services/graphiti-knowledge/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-003 §3 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-GR-006 |
| **Estimated** | 3h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** graphiti-knowledge prompt registry  
---

## Context

Xây dựng `PromptRegistry` với 6 prompt templates dùng cho tất cả LLM calls trong graphiti pipeline. Mỗi template có system prompt, user prompt generator (theo context), và JSON response schema.

---

## Goal

- `PromptRegistry` — map tên prompt → template
- 6 templates: `extract_nodes`, `extract_edges`, `dedupe_nodes`, `dedupe_edges`, `summarize_nodes`, `summarize_sagas`
- JSON Schema definitions cho structured LLM output
- Multilingual support (language hint trong user prompt)
- Ontology-aware variants (prescribed entity/edge types)

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-knowledge/internal/adapter/prompt/registry.go` |
| CREATE | `services/graphiti-knowledge/internal/adapter/prompt/schemas.go` |
| CREATE | `services/graphiti-knowledge/internal/adapter/prompt/extract_nodes.go` |
| CREATE | `services/graphiti-knowledge/internal/adapter/prompt/extract_edges.go` |
| CREATE | `services/graphiti-knowledge/internal/adapter/prompt/dedupe.go` |
| CREATE | `services/graphiti-knowledge/internal/adapter/prompt/summarize.go` |

---

## Implementation

### File 1: `services/graphiti-knowledge/internal/adapter/prompt/registry.go`

```go
package prompt

import (
    "fmt"

    "github.com/vnp-memory/pkg/graph"
)

// PromptContext contains the inputs needed to build a user prompt message
type PromptContext struct {
    Chunks        []string
    PrevEpisodes  []string  // recent episode content for context window
    ExistingNodes []string  // for dedupe: candidate entity names
    EntityTypes   map[string]graph.EntityTypeSchema
    EdgeTypes     map[string]graph.EdgeTypeSchema
    ReferenceTime string    // ISO8601 for temporal context
    Language      string    // e.g. "vi", "en" (empty = auto-detect)
    Source        string    // episode source type
}

// PromptTemplate defines a complete LLM prompt configuration
type PromptTemplate struct {
    Name         string
    SystemPrompt string
    BuildUser    func(ctx PromptContext) string
    Schema       interface{}  // JSON schema for structured output
}

// PromptRegistry stores all available prompt templates
type PromptRegistry struct {
    templates map[string]PromptTemplate
}

// NewPromptRegistry creates a registry pre-loaded with all 6 graphiti templates
func NewPromptRegistry() *PromptRegistry {
    reg := &PromptRegistry{templates: make(map[string]PromptTemplate)}
    reg.Register(extractNodesPrompt())
    reg.Register(extractEdgesPrompt())
    reg.Register(dedupeNodesPrompt())
    reg.Register(dedupeEdgesPrompt())
    reg.Register(summarizeNodesPrompt())
    reg.Register(summarizeSagasPrompt())
    return reg
}

func (r *PromptRegistry) Register(t PromptTemplate) {
    r.templates[t.Name] = t
}

func (r *PromptRegistry) Get(name string) (PromptTemplate, error) {
    t, ok := r.templates[name]
    if !ok { return PromptTemplate{}, fmt.Errorf("prompt template %q not found", name) }
    return t, nil
}

func (r *PromptRegistry) MustGet(name string) PromptTemplate {
    t, err := r.Get(name)
    if err != nil { panic(err) }
    return t
}

// Sanitize removes control characters that can cause LLM API errors
func Sanitize(text string) string {
    result := make([]rune, 0, len(text))
    for _, r := range text {
        if r < 32 && r != '\n' && r != '\t' && r != '\r' { continue }
        result = append(result, r)
    }
    return string(result)
}
```

### File 2: `services/graphiti-knowledge/internal/adapter/prompt/schemas.go`

```go
package prompt

// JSON schemas for structured LLM output — used with response_format: json_schema

var ExtractedNodeListSchema = map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "entities": map[string]interface{}{
            "type": "array",
            "items": map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "name":    map[string]interface{}{"type": "string"},
                    "label":   map[string]interface{}{"type": "string"},
                    "summary": map[string]interface{}{"type": "string"},
                },
                "required": []string{"name", "label"},
            },
        },
    },
    "required": []string{"entities"},
}

var ExtractedEdgeListSchema = map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "edges": map[string]interface{}{
            "type": "array",
            "items": map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "source_entity": map[string]interface{}{"type": "string"},
                    "target_entity": map[string]interface{}{"type": "string"},
                    "relation_type": map[string]interface{}{"type": "string"},
                    "fact":          map[string]interface{}{"type": "string"},
                    "valid_at":      map[string]interface{}{"type": ["string", "null"]},
                    "invalid_at":    map[string]interface{}{"type": ["string", "null"]},
                },
                "required": []string{"source_entity", "target_entity", "relation_type", "fact"},
            },
        },
    },
    "required": []string{"edges"},
}

var EntityResolutionSchema = map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "decision":      map[string]interface{}{"type": "string", "enum": []string{"merge", "new"}},
        "existing_uuid": map[string]interface{}{"type": "string"},
        "reasoning":     map[string]interface{}{"type": "string"},
    },
    "required": []string{"decision"},
}

var EdgeResolutionSchema = map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "resolution": map[string]interface{}{
            "type": "string",
            "enum": []string{"DUPLICATE", "NEW", "CONTRADICTION", "UPDATE"},
        },
        "invalidated_edge_uuids": map[string]interface{}{
            "type":  "array",
            "items": map[string]interface{}{"type": "string"},
        },
        "reasoning": map[string]interface{}{"type": "string"},
    },
    "required": []string{"resolution"},
}

var NodeSummarySchema = map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "summary": map[string]interface{}{"type": "string"},
    },
    "required": []string{"summary"},
}

var SagaSummarySchema = map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "summary": map[string]interface{}{"type": "string"},
        "title":   map[string]interface{}{"type": "string"},
    },
    "required": []string{"summary"},
}
```

### File 3: `services/graphiti-knowledge/internal/adapter/prompt/extract_nodes.go`

```go
package prompt

import (
    "fmt"
    "strings"
)

func extractNodesPrompt() PromptTemplate {
    return PromptTemplate{
        Name: "extract_nodes",
        SystemPrompt: `You are an expert knowledge graph builder. Extract named entities from the provided text.

Extract entities that are clearly identifiable and semantically meaningful. For each entity provide:
- name: the entity's canonical name (normalized, not a pronoun)
- label: entity type classification (e.g. Person, Organization, Location, Concept, Event, Product)
- summary: brief 1-2 sentence description based only on what's stated in the context

Rules:
- Do NOT extract generic nouns (e.g. "meeting", "issue") unless they are specific named things
- Do NOT extract pronouns
- Normalize entity names (e.g. "John" and "John Smith" in same text → use "John Smith")
- Return entities as a JSON object with "entities" array`,

        BuildUser: func(ctx PromptContext) string {
            var sb strings.Builder

            sb.WriteString("Text to analyze:\n```\n")
            for _, chunk := range ctx.Chunks {
                sb.WriteString(Sanitize(chunk))
                sb.WriteString("\n\n")
            }
            sb.WriteString("```\n")

            if len(ctx.PrevEpisodes) > 0 {
                sb.WriteString("\nPrevious context (for disambiguation):\n")
                for _, ep := range ctx.PrevEpisodes {
                    sb.WriteString("- ")
                    sb.WriteString(Sanitize(ep))
                    sb.WriteString("\n")
                }
            }

            if len(ctx.EntityTypes) > 0 {
                sb.WriteString("\nIMPORTANT: Extract ONLY entities matching these prescribed types:\n")
                for name, schema := range ctx.EntityTypes {
                    sb.WriteString(fmt.Sprintf("- **%s**: %s\n", name, schema.Description))
                    if len(schema.Examples) > 0 {
                        sb.WriteString(fmt.Sprintf("  Examples: %s\n", strings.Join(schema.Examples, ", ")))
                    }
                }
                sb.WriteString("Do NOT extract entities that don't match these types.\n")
            }

            if ctx.Language != "" && ctx.Language != "en" {
                sb.WriteString(fmt.Sprintf("\nNote: Text is in %s language. Keep entity names in their original language.\n", ctx.Language))
            }

            if ctx.ReferenceTime != "" {
                sb.WriteString(fmt.Sprintf("\nReference time: %s\n", ctx.ReferenceTime))
            }

            return sb.String()
        },

        Schema: ExtractedNodeListSchema,
    }
}
```

### File 4: `services/graphiti-knowledge/internal/adapter/prompt/extract_edges.go`

```go
package prompt

import (
    "fmt"
    "strings"
)

func extractEdgesPrompt() PromptTemplate {
    return PromptTemplate{
        Name: "extract_edges",
        SystemPrompt: `You are an expert knowledge graph builder. Extract factual relationships between named entities.

For each relationship extract:
- source_entity: name of the source entity (must be an extracted entity)
- target_entity: name of the target entity (must be an extracted entity)  
- relation_type: UPPERCASE relationship type (e.g. WORKS_AT, REPORTS_TO, FOUNDED, ACQUIRED)
- fact: natural language statement of the specific fact
- valid_at: ISO8601 when fact became true (null if unknown)
- invalid_at: ISO8601 when fact ceased to be true (null if still valid)

Rules:
- Only extract relationships between entities present in the provided entity list
- Be specific: "Alice joined Engineering team at Acme in March 2024" not "Alice works somewhere"
- For temporal facts: extract valid_at/invalid_at when explicitly mentioned
- Return as JSON object with "edges" array`,

        BuildUser: func(ctx PromptContext) string {
            var sb strings.Builder

            sb.WriteString("Text:\n```\n")
            for _, chunk := range ctx.Chunks { sb.WriteString(Sanitize(chunk) + "\n\n") }
            sb.WriteString("```\n")

            if len(ctx.ExistingNodes) > 0 {
                sb.WriteString("\nEntities to use (source/target must be from this list):\n")
                for _, n := range ctx.ExistingNodes {
                    sb.WriteString("- ")
                    sb.WriteString(n)
                    sb.WriteString("\n")
                }
            }

            if len(ctx.EdgeTypes) > 0 {
                sb.WriteString("\nIMPORTANT: Extract ONLY relationships of these prescribed types:\n")
                for name, schema := range ctx.EdgeTypes {
                    sb.WriteString(fmt.Sprintf("- **%s**: %s", name, schema.Description))
                    if len(schema.SourceTypes) > 0 {
                        sb.WriteString(fmt.Sprintf(" (from %s to %s)",
                            strings.Join(schema.SourceTypes, "/"),
                            strings.Join(schema.TargetTypes, "/")))
                    }
                    sb.WriteString("\n")
                }
            }

            if ctx.ReferenceTime != "" {
                sb.WriteString(fmt.Sprintf("\nReference time for temporal context: %s\n", ctx.ReferenceTime))
            }

            return sb.String()
        },

        Schema: ExtractedEdgeListSchema,
    }
}
```

### File 5: `services/graphiti-knowledge/internal/adapter/prompt/dedupe.go`

```go
package prompt

import (
    "fmt"
    "strings"
)

func dedupeNodesPrompt() PromptTemplate {
    return PromptTemplate{
        Name: "dedupe_nodes",
        SystemPrompt: `You are resolving whether a new entity mention refers to an existing entity in the knowledge graph.

Given:
1. A new entity to add
2. A list of candidate existing entities that might be the same

Decide:
- "merge" + existing_uuid: if the new entity IS the same real-world entity as an existing one
- "new": if the new entity is genuinely different from all candidates

Consider: same person/org/thing with different name spelling, abbreviation, nickname, or alias → MERGE.
Different entities that happen to share a similar name → NEW.`,

        BuildUser: func(ctx PromptContext) string {
            var sb strings.Builder
            if len(ctx.Chunks) > 0 {
                sb.WriteString(fmt.Sprintf("New entity: **%s** (label: %s)\n", ctx.Chunks[0], ctx.Source))
                if len(ctx.Chunks) > 1 { sb.WriteString(fmt.Sprintf("Summary: %s\n\n", ctx.Chunks[1])) }
            }
            if len(ctx.ExistingNodes) > 0 {
                sb.WriteString("Candidate existing entities:\n")
                for _, n := range ctx.ExistingNodes { sb.WriteString("- " + n + "\n") }
            }
            return sb.String()
        },

        Schema: EntityResolutionSchema,
    }
}

func dedupeEdgesPrompt() PromptTemplate {
    return PromptTemplate{
        Name: "dedupe_edges",
        SystemPrompt: `You are resolving how a new fact relates to existing facts in the knowledge graph.

Given a new fact and similar existing facts, categorize the relationship:
- DUPLICATE: the same fact already exists (same meaning, same entities)
- NEW: a genuinely independent fact that doesn't conflict
- CONTRADICTION: the new fact directly contradicts an existing fact (e.g. "Alice works at X" vs "Alice works at Y" for the same time period)
- UPDATE: the new fact is a temporal update to an existing fact (supersedes it)

For CONTRADICTION and UPDATE: provide the UUIDs of edges to invalidate in invalidated_edge_uuids.`,

        BuildUser: func(ctx PromptContext) string {
            var sb strings.Builder
            if len(ctx.Chunks) > 0 {
                sb.WriteString(fmt.Sprintf("New fact: \"%s\"\n", ctx.Chunks[0]))
                if ctx.ReferenceTime != "" { sb.WriteString(fmt.Sprintf("Valid at: %s\n\n", ctx.ReferenceTime)) }
            }
            if len(ctx.ExistingNodes) > 0 {
                sb.WriteString("Similar existing facts in the graph:\n")
                for i, n := range ctx.ExistingNodes {
                    sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, n))
                }
            }
            return sb.String()
        },

        Schema: EdgeResolutionSchema,
    }
}
```

### File 6: `services/graphiti-knowledge/internal/adapter/prompt/summarize.go`

```go
package prompt

import "strings"

func summarizeNodesPrompt() PromptTemplate {
    return PromptTemplate{
        Name: "summarize_nodes",
        SystemPrompt: `You are updating an entity summary in a knowledge graph.

Given an entity's existing summary (may be empty) and new facts about it, produce an updated summary.
The summary should be concise (1-3 sentences), factual, and incorporate the new information.
Do not include speculation. Use only information explicitly stated.`,

        BuildUser: func(ctx PromptContext) string {
            var sb strings.Builder
            if len(ctx.Chunks) > 0 {
                sb.WriteString("Entity name: " + ctx.Chunks[0] + "\n")
            }
            if len(ctx.Chunks) > 1 && ctx.Chunks[1] != "" {
                sb.WriteString("Existing summary: " + ctx.Chunks[1] + "\n\n")
            }
            if len(ctx.ExistingNodes) > 0 {
                sb.WriteString("New facts about this entity:\n")
                for _, f := range ctx.ExistingNodes { sb.WriteString("- " + f + "\n") }
            }
            return sb.String()
        },

        Schema: NodeSummarySchema,
    }
}

func summarizeSagasPrompt() PromptTemplate {
    return PromptTemplate{
        Name: "summarize_sagas",
        SystemPrompt: `You are summarizing a sequence of related events (a saga/narrative) in a knowledge graph.

Given episode summaries in chronological order, create:
- summary: A coherent narrative summary (2-5 sentences) of the overall saga
- title: A brief descriptive title for this saga (5-10 words)

Focus on the arc: what happened, who was involved, and the outcome.`,

        BuildUser: func(ctx PromptContext) string {
            var sb strings.Builder
            sb.WriteString("Episodes in chronological order:\n\n")
            for i, ep := range ctx.Chunks {
                sb.WriteString(fmt.Sprintf("%d. %s\n\n", i+1, ep))
            }
            if ctx.ReferenceTime != "" {
                sb.WriteString("Last updated: " + ctx.ReferenceTime + "\n")
            }
            return sb.String()
        },

        Schema: SagaSummarySchema,
    }
}
```

---

## Verification

```bash
cd services/graphiti-knowledge
go build ./internal/adapter/prompt/...
go test ./internal/adapter/prompt/... -v
```

**Expected:** All prompts build without error. Schema JSON marshals correctly.
