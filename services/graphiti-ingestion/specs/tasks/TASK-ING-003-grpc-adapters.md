---
id: TASK-ING-003
title: Implement gRPC Handlers + Knowledge/Store Clients
service: graphiti-ingestion
type: task
status: done
priority: P0
created: 2026-05-11
dependencies: [TASK-ING-002]
estimated_time: 8h
linked_feat: FEAT-ING-003
---

## Objective
Implement gRPC handlers (inbound) cho GraphitiIngestionService và gRPC clients (outbound) cho graphiti-knowledge + graphiti-store.

## Scope
- `internal/adapter/grpc/handler.go` — IngestEpisode, BulkIngest, GetEpisodeStatus, ListEpisodes, RemoveEpisode
- `internal/adapter/grpc/mapper.go` — Proto ↔ Domain mapping
- `internal/adapter/client/knowledge_client.go` — gRPC → graphiti-knowledge:9023 (ExtractEntities, ResolveEntities, ExtractEdges, ResolveEdges, UpdateCommunity)
- `internal/adapter/client/store_client.go` — gRPC → graphiti-store:9024 (SaveBulk, RollbackBulk)
- Circuit breaker on both outbound clients

## Acceptance Criteria
- [x] IngestEpisode RPC accepts request and triggers saga pipeline
- [x] Knowledge client delegates all 5 extraction/resolution RPCs
- [x] Store client handles SaveBulk + RollbackBulk with circuit breaker
- [x] Proto ↔ Domain mapping covers all types losslessly
- [x] `x-tenant-id` extracted from metadata

## Test Requirements
- Unit tests: Handlers with mock usecases, clients with mock gRPC servers
- Minimum coverage: 80%
