---
id: FEAT-SEA-003
title: gRPC Handlers — Search Service
service: graphiti-search
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement gRPC handlers cho GraphitiSearchService: HybridSearch, NodeSearch, EdgeSearch, CommunitySearch.

## Scope

- `internal/adapter/grpc/handler.go` — 4 RPC handlers
- `internal/adapter/grpc/mapper.go` — Proto ↔ Domain mapping
- Tenant extraction, OTel spans, error code mapping

## Acceptance Criteria

- [ ] AC-1: HybridSearch accepts query + methods + rerankers → returns ranked results
- [ ] AC-2: NodeSearch filters by entity labels + temporal window
- [ ] AC-3: EdgeSearch filters by edge type + temporal window
- [ ] AC-4: CommunitySearch returns communities matching query
- [ ] AC-5: `x-tenant-id` → group_id propagation

## Test Requirements
- **Unit tests**: Handlers with mock usecases
- **Minimum coverage**: 80%
