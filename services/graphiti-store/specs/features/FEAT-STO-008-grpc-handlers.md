---
id: FEAT-STO-008
title: gRPC Handler Adapters
service: graphiti-store
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement gRPC handler adapters cho GraphitiStoreService: 15 RPCs covering node CRUD, edge CRUD (bi-temporal), bulk operations, search primitives, và index management.

## Scope

- `internal/adapter/grpc/handler.go` — GraphitiStoreService implementation
- `internal/adapter/grpc/mapper.go` — Proto ↔ Domain bidirectional mapping
- Tenant extraction from `x-tenant-id` gRPC metadata
- OTel span per RPC

### RPCs

| Category | RPCs |
|---------|------|
| Node | SaveNode, GetNode, DeleteNode |
| Edge | SaveEdge, GetEdge, DeleteEdge, InvalidateEdge |
| Bulk | SaveBulk, RollbackBulk, DeleteByGroupID |
| Search | CosineSimilaritySearch, FulltextSearch, BFSSearch |
| Index | BuildIndices, DropIndices |

## Acceptance Criteria

- [ ] AC-1: All 15 RPCs correctly delegate to usecase layer
- [ ] AC-2: Proto ↔ Domain mapper handles all entity types losslessly
- [ ] AC-3: `x-tenant-id` metadata extracted and propagated as GroupID
- [ ] AC-4: gRPC status codes: NOT_FOUND for missing entities, INVALID_ARGUMENT for bad input
- [ ] AC-5: OTel span created per RPC

## Test Requirements
- **Unit tests**: Handlers with mock usecases, mapper round-trip
- **Minimum coverage**: 80%
