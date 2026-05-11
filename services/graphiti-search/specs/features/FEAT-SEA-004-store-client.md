---
id: FEAT-SEA-004
title: graphiti-store Client Adapter
service: graphiti-search
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement gRPC client adapter kết nối graphiti-search → graphiti-store:9024 cho search primitives delegation (cosine, fulltext, BFS).

## Scope

- `internal/adapter/client/store_client.go` — implements StoreSearchClient port
- Circuit breaker cho store gRPC calls
- Deadline propagation + OTel trace injection

## Acceptance Criteria

- [ ] AC-1: CosineSimilaritySearch delegates to graphiti-store and returns results
- [ ] AC-2: FulltextSearch delegates with proper BM25 parameters
- [ ] AC-3: BFSSearch delegates with depth + start node
- [ ] AC-4: Circuit breaker opens after 5 consecutive failures
- [ ] AC-5: OTel traces span outgoing calls

## Test Requirements
- **Unit tests**: Mock gRPC server, circuit breaker transitions
- **Minimum coverage**: 80%
