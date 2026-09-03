# SOL-COGNEE-001 — Solution: Memify (Graph Enrichment)

| Field | Value |
|---|---|
| **Solution ID** | SOL-COGNEE-001 |
| **CR** | [CR-COGNEE-001](../../../../docs/crs/v1/cognee/CR-COGNEE-001*.md) |
| **TDD ref** | [02-cognee-services.md](../../../tdd/architecture/02-cognee-services.md) §cognify |
| **Status** | Open |
| **Priority** | 🟡 High |

---
## 1. Giải pháp

`cognee-cognify` đã có `StartCognify`. Thêm `Memify` (non-destructive enrichment).

### 1.1 `services/cognee-cognify/internal/usecase/memify.go` [NEW]

```go
type MemifyUseCase struct {
    graphRepo  port.GraphRepository  // Neo4j
    vectorRepo port.VectorRepository // pgvector
    llm        port.LLMClient
    embedder   port.EmbedderClient
    pub        port.EventPublisher
}

// Steps: LoadGraph → DeriveFacts(LLM) → BuildDiff → UpsertGraph → EmbedTriplets → Publish
func (u *MemifyUseCase) Memify(ctx context.Context, req *MemifyRequest) (*MemifyResult, error) {
    existing, _ := u.graphRepo.GetDatasetGraph(ctx, req.DatasetID, req.TenantID)
    derived, _ := u.llm.DeriveFacts(ctx, existing)
    diff := buildDiff(existing, derived)  // only new facts
    u.graphRepo.UpsertNodes(ctx, diff.Nodes)
    u.graphRepo.UpsertEdges(ctx, diff.Edges)
    embeddings, _ := u.embedder.EmbedTriplets(ctx, diff.Triplets)
    u.vectorRepo.UpsertTriplets(ctx, embeddings)
    u.pub.Publish(ctx, "cognee.cognify.memify.completed", map[string]any{
        "dataset_id": req.DatasetID, "new_nodes": len(diff.Nodes), "new_edges": len(diff.Edges),
    })
    return &MemifyResult{NewNodes: len(diff.Nodes), NewEdges: len(diff.Edges)}, nil
}
```

### 1.2 gRPC method

```protobuf
// api/proto/cognee/cognify/v1/cognify.proto
rpc Memify(MemifyRequest) returns (MemifyResponse);
message MemifyRequest { string dataset_id = 1; string tenant_id = 2; }
message MemifyResponse { int32 new_nodes = 1; int32 new_edges = 2; }
```

## 2. File Changes

| File | Action |
|---|---|
| `services/cognee-cognify/internal/usecase/memify.go` | NEW |
| `api/proto/cognee/cognify/v1/cognify.proto` | MODIFY — add Memify rpc |
| `services/cognee-cognify/internal/adapter/grpc/handler.go` | MODIFY |

## 3. Acceptance Criteria

- [ ] Memify không xóa existing graph nodes/edges
- [ ] Chỉ thêm derived facts chưa tồn tại
- [ ] Triplet embeddings cập nhật trong pgvector
- [ ] Event published sau khi hoàn thành
