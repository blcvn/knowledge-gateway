---
id: FEAT-ING-003
title: gRPC Handlers + Knowledge/Store Clients
service: graphiti-ingestion
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement gRPC handlers (inbound) cho GraphitiIngestionService và gRPC clients (outbound) cho graphiti-knowledge + graphiti-store.

## Scope

### In Scope
- `internal/adapter/grpc/handler.go` — IngestEpisode, BulkIngest, GetEpisodeStatus, ListEpisodes, RemoveEpisode
- `internal/adapter/grpc/mapper.go` — Proto ↔ Domain mapping
- `internal/adapter/client/knowledge_client.go` — gRPC → graphiti-knowledge:9023
  - ExtractEntities, ResolveEntities, ExtractEdges, ResolveEdges, UpdateCommunity
- `internal/adapter/client/store_client.go` — gRPC → graphiti-store:9024
  - SaveBulk, RollbackBulk
- Circuit breaker on both outbound clients

## Acceptance Criteria

- [ ] AC-1: IngestEpisode RPC accepts request and triggers saga pipeline
- [ ] AC-2: Knowledge client delegates all 5 extraction/resolution RPCs
- [ ] AC-3: Store client handles SaveBulk + RollbackBulk with circuit breaker
- [ ] AC-4: Proto ↔ Domain mapping covers all types losslessly
- [ ] AC-5: `x-tenant-id` extracted from metadata

## Test Requirements
- **Unit tests**: Handlers with mock usecases, clients with mock gRPC servers
- **Minimum coverage**: 80%
