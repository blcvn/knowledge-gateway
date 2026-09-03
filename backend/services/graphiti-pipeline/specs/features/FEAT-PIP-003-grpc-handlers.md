---
id: FEAT-PIP-003
title: gRPC Handler Adapters — Ingestion + Knowledge Services
service: graphiti-pipeline
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement gRPC handler adapters cho cả 2 proto services (GraphitiIngestionService + GraphitiKnowledgeService) registered trên cùng port :9021. Bao gồm proto ↔ domain mapper.

## Scope

### In Scope
- `internal/adapter/grpc/ingestion_handler.go` — IngestEpisode, BulkIngest, GetEpisodeStatus, ListEpisodes, RemoveEpisode
- `internal/adapter/grpc/knowledge_handler.go` — ExtractEntities, ResolveEntities, ExtractEdges, ResolveEdges, GenerateEmbedding, Rerank, UpdateCommunity
- `internal/adapter/grpc/mapper.go` — bidirectional Proto ↔ Domain mapping
- gRPC interceptors: OTel tracing, panic recovery, logging, tenant extraction

### Out of Scope
- Proto file generation (shared pkg/)
- LLM adapter (FEAT-PIP-004)

## Acceptance Criteria

- [ ] AC-1: Both gRPC services registered on single port :9021
- [ ] AC-2: `x-tenant-id` metadata extracted and propagated as GroupID to usecases
- [ ] AC-3: All RPCs return proper gRPC status codes (see api.md)
- [ ] AC-4: OTel span created per RPC with service.method attribute
- [ ] AC-5: Proto ↔ Domain mapping is lossless (round-trip test)

## Test Requirements
- **Unit tests**: Handler methods with mocked usecases, mapper round-trip tests
- **Minimum coverage**: 80%
