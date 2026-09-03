# SOL-SM-001 — Solution: Document Ingestion Pipeline

| Field | Value |
|---|---|
| **Solution ID** | SOL-SM-001 |
| **CR** | CR-SM-001 |
| **TDD ref** | [07-supermemory-services.md](../../../tdd/architecture/07-supermemory-services.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |
| **Component** | `services/sm-memory` |

---

## 1. Giải pháp

Ingest documents → chunk → embed → store in pgvector + knowledge graph.

```go
func (u *IngestUseCase) IngestDocument(ctx context.Context, req *DocumentRequest) (*Document, error) {
    chunks, _ := u.chunker.Chunk(req.Content, 512)
    for _, chunk := range chunks {
        embedding, _ := u.embedder.Embed(ctx, chunk.Text)
        u.vectorRepo.Store(ctx, chunk.ID, req.TenantID, embedding)
    }
    // Knowledge graph: extract entities
    entities, _ := u.llm.ExtractEntities(ctx, req.Content)
    u.graphRepo.StoreEntities(ctx, req.TenantID, entities)
    return &Document{ID: uuid.NewString(), ChunkCount: len(chunks)}, nil
}
```

## 2. Acceptance Criteria

- [ ] Document chunked with overlap
- [ ] Chunks embedded and stored in pgvector
- [ ] Entities extracted and stored in knowledge graph

