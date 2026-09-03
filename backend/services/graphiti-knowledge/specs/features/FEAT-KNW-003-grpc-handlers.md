---
id: FEAT-KNW-003
title: gRPC Handlers — Knowledge Service
service: graphiti-knowledge
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement gRPC handlers cho GraphitiKnowledgeService: 9 RPCs.

## Scope

- `internal/adapter/grpc/handler.go` — ExtractEntities, ResolveEntities, ExtractEdges, ResolveEdges, GenerateEmbedding, GenerateEmbeddingBulk, Rerank, UpdateCommunity, GetTokenUsage
- `internal/adapter/grpc/mapper.go` — Proto ↔ Domain mapping
- Tenant extraction, OTel spans

## Acceptance Criteria

- [ ] AC-1: All 9 RPCs correctly delegate to usecase layer
- [ ] AC-2: Same proto as graphiti-pipeline/knowledge handler (swappable)
- [ ] AC-3: `x-tenant-id` → group_id propagation
- [ ] AC-4: TokenUsage tracked per RPC and returned in response
- [ ] AC-5: OTel span per RPC with model + template attributes

## Test Requirements
- **Unit tests**: Handlers with mock usecases
- **Minimum coverage**: 80%
