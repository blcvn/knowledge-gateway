# SOL-OV-005 — Solution: Resource Service (Ingestion Pipeline)

| Field | Value |
|---|---|
| **Solution ID** | SOL-OV-005 |
| **CR** | CR-OV-005 |
| **TDD ref** | [05-openviking-services.md](../../../tdd/architecture/05-openviking-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |
| **Component** | `services/ov-resource` |

---

## 1. Giải pháp

Ingest resources (code files, docs) → extract text → embed → index for search.

```go
func (u *ResourceUseCase) IngestResource(ctx context.Context, req *IngestRequest) error {
    chunks, _ := u.splitter.ChunkFile(req.Content, req.Language, chunkSize: 512)
    for _, chunk := range chunks {
        embedding, _ := u.embedder.Embed(ctx, chunk.Text)
        u.vectorRepo.Store(ctx, &ChunkVector{
            TenantID: req.TenantID, Path: req.Path,
            Chunk: chunk, Embedding: embedding,
        })
    }
    u.bm25.Index(ctx, req.TenantID, req.Path, chunks)
    return nil
}
```

## 2. Acceptance Criteria

- [ ] Code files chunked with language-aware splitter
- [ ] Chunks indexed in both BM25 and pgvector
- [ ] Re-index triggered on file update

