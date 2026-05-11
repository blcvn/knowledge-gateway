---
id: FEAT-STO-002
title: Usecase Layer + Port Interfaces
service: graphiti-store
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement usecase layer orchestration và port interfaces cho graphiti-store. Mỗi operation group (node, edge, bulk, search, index) có usecase riêng. Single output port = GraphDriver interface.

## Scope

### In Scope
- `internal/usecase/node_ops.go` — SaveNode, GetNode, DeleteNode, ListNodes
- `internal/usecase/edge_ops.go` — SaveEdge, GetEdge, DeleteEdge, InvalidateEdge, GetEdgesInTimeRange
- `internal/usecase/community_ops.go` — SaveCommunity, GetCommunity, DeleteCommunity
- `internal/usecase/bulk_ops.go` — SaveBulk (atomic), RollbackBulk, DeleteByGroupID
- `internal/usecase/search_ops.go` — CosineSimilarity, Fulltext, BFS (delegation to driver)
- `internal/usecase/index_ops.go` — BuildIndices, DropIndices, ListIndices
- Port interfaces (input + output)
- DTOs for request/response mapping

### Out of Scope
- Driver implementations (FEAT-STO-003..007)

## Thiết Kế Kỹ Thuật

### Usecase Pattern

```go
type NodeOpsUseCase struct {
    driver domain.GraphDriver
}

func (uc *NodeOpsUseCase) SaveNode(ctx context.Context, req dto.SaveNodeRequest) error {
    node := mapToEntity(req)
    if err := node.Validate(); err != nil {
        return err
    }
    return uc.driver.SaveNode(ctx, node)
}
```

### Bulk Operations — Transaction Guarantee

```go
func (uc *BulkOpsUseCase) SaveBulk(ctx context.Context, req dto.SaveBulkRequest) error {
    return uc.driver.WithTransaction(ctx, func(tx domain.Transaction) error {
        for _, node := range req.Nodes { /* save each node */ }
        for _, edge := range req.Edges { /* save each edge */ }
        /* save episode node */
        return nil
    })
}
```

## Acceptance Criteria

- [ ] AC-1: All usecases depend only on GraphDriver interface (single output port)
- [ ] AC-2: SaveBulk wraps all operations in a single transaction
- [ ] AC-3: RollbackBulk deletes all nodes/edges created by a specific episode
- [ ] AC-4: DeleteByGroupID removes all tenant data (purge operation)
- [ ] AC-5: All usecases validate input before delegating to driver
- [ ] AC-6: Search usecases pass-through to driver (no additional logic)

## Test Requirements
- **Unit tests**: Usecases with mock GraphDriver
- **Minimum coverage**: 80%
