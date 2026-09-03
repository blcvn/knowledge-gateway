# SOL-SM-003 — Solution: Hybrid Search Engine (RAG + Memory)

| Field | Value |
|---|---|
| **Solution ID** | SOL-SM-003 |
| **CR** | CR-SM-003 |
| **TDD ref** | [07-supermemory-services.md](../../../tdd/architecture/07-supermemory-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |
| **Component** | `services/sm-search` |

---

## 1. Giải pháp

RAG pipeline: semantic search + knowledge graph traversal → LLM-augmented answer.

```go
func (u *SearchUseCase) RAGSearch(ctx context.Context, req *RAGRequest) (*RAGResult, error) {
    // 1. Vector search
    embedding, _ := u.embedder.Embed(ctx, req.Query)
    chunks, _ := u.vectorRepo.Search(ctx, req.TenantID, embedding, 10)
    
    // 2. Graph context
    entities, _ := u.graphRepo.GetRelated(ctx, req.TenantID, req.Query)
    
    // 3. LLM augment
    answer, _ := u.llm.GenerateAnswer(ctx, req.Query, chunks, entities)
    return &RAGResult{Answer: answer, Sources: chunks}, nil
}
```

## 2. Acceptance Criteria

- [ ] RAG returns answer with source citations
- [ ] Graph context enriches LLM prompt
- [ ] p95 < 2s for RAG query

