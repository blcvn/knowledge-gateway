# SOL-GR-003 — Solution: Knowledge Processing Service

| Field | Value |
|---|---|
| **Solution ID** | SOL-GR-003 |
| **CR** | CR-GR-003 |
| **TDD ref** | [03-graphiti-services.md](../../../tdd/architecture/03-graphiti-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |
| **Component** | `services/graphiti-knowledge` |

---

## 1. Phân tích

LLM-powered entity/edge extraction as a dedicated microservice.

### Key: `services/graphiti-knowledge/internal/usecase/extract.go` [NEW]

```go
func (u *ExtractionUseCase) ExtractKnowledge(ctx context.Context, content string) (*KnowledgeGraph, error) {
    prompt := buildExtractionPrompt(content, u.ontology)
    result, _ := u.llm.ExtractStructured(ctx, prompt, KnowledgeSchema)
    return parseKnowledgeGraph(result), nil
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/graphiti-knowledge/internal/usecase/extract.go` | NEW |
| `services/graphiti-knowledge/internal/port/llm.go` | NEW — extraction interface |

---

## 3. Acceptance Criteria

- [ ] Extraction returns typed entities + edges
- [ ] Handles ambiguous references via coreference resolution
- [ ] < 2s per 1000 tokens
