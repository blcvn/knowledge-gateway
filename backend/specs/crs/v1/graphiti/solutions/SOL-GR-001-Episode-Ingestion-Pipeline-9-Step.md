# SOL-GR-001 — Solution: Episode Ingestion Pipeline (9-Step)

| Field | Value |
|---|---|
| **Solution ID** | SOL-GR-001 |
| **CR** | CR-GR-001 |
| **TDD ref** | [03-graphiti-services.md](../../../tdd/architecture/03-graphiti-services.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |
| **Component** | `services/graphiti-ingestion` |

---

## 1. Phân tích

Graphiti episode ingestion pipeline: Extract entities → deduplicate → embed → store in Neo4j + pgvector.

### Key implementation: `services/graphiti-ingestion/internal/usecase/ingest.go` [MODIFY]

```go
// 9-Step pipeline:
// 1. Validate episode content
// 2. Entity extraction (LLM → structured entities)
// 3. Entity deduplication (fuzzy match existing nodes)
// 4. Relationship extraction (LLM → typed edges)
// 5. Temporal edge creation (with timestamps)
// 6. Node embedding (entity text → vector)
// 7. Store nodes/edges in Neo4j
// 8. Store embeddings in pgvector
// 9. Publish NATS event: graphiti.episode.ingested

func (u *IngestUseCase) IngestEpisode(ctx context.Context, req *EpisodeRequest) (*Episode, error) {
    entities, _ := u.llm.ExtractEntities(ctx, req.Content)
    deduped, _ := u.graphRepo.DeduplicateEntities(ctx, req.TenantID, entities)
    edges, _ := u.llm.ExtractRelationships(ctx, req.Content, deduped)
    temporalEdges := addTemporalEdges(edges, req.Timestamp)
    embeddings, _ := u.embedder.EmbedEntities(ctx, deduped)
    u.graphRepo.UpsertNodes(ctx, req.TenantID, deduped)
    u.graphRepo.UpsertEdges(ctx, req.TenantID, temporalEdges)
    u.vectorRepo.UpsertEntityEmbeddings(ctx, req.TenantID, embeddings)
    u.pub.Publish(ctx, "graphiti.episode.ingested", ...)
    return &Episode{...}, nil
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/graphiti-ingestion/internal/usecase/ingest.go` | MODIFY — 9-step pipeline |
| `services/graphiti-ingestion/internal/port/graph.go` | MODIFY — DeduplicateEntities |
| `deployment/dev/migrations/0XX_graphiti_episodes.sql` | NEW |

---

## 3. Acceptance Criteria

- [ ] Entity extraction with 85%+ precision
- [ ] Deduplication reduces graph size by 30%+
- [ ] Temporal edges store created_at correctly
- [ ] Pipeline completes < 5s per episode
