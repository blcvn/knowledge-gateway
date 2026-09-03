---
id: FEAT-KNW-006
title: graphiti-store Client — Read-Only
service: graphiti-knowledge
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement gRPC client cho graphiti-store (read-only) implementing GraphReader port. Dùng cho entity resolution queries (FindSimilarEntities, GetEntityByName).

## Scope

- `internal/adapter/client/store_client.go` — GraphReader port implementation
- FindSimilarEntities: cosine search on entity name_embedding
- FindSimilarEdges: cosine search on edge fact_embedding
- GetEntityByName: exact name lookup within group_id
- Circuit breaker + deadline propagation

## Acceptance Criteria

- [ ] AC-1: FindSimilarEntities delegates to graphiti-store CosineSimilaritySearch
- [ ] AC-2: GetEntityByName returns exact match within group_id scope
- [ ] AC-3: Circuit breaker opens after 5 failures
- [ ] AC-4: OTel traces span outgoing calls

## Test Requirements
- **Unit tests**: Mock gRPC server
- **Minimum coverage**: 80%
