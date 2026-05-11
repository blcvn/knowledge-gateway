---
id: TASK-PIP-002
title: "Implement Consolidated Adapters"
service: cognee-pipeline
status: Done
priority: P1
linked_feat: FEAT-PIP-001
---

## Objective
Consolidate the adapter layer to share resources (gRPC, DB pools, messaging) across both ingestion and cognify workflows within `cognee-pipeline`.

## Scope
1. **gRPC Handlers**:
   - Implement `internal/adapter/grpc/ingestion_handler.go` and `internal/adapter/grpc/cognify_handler.go`.
   - Map proto requests/responses to the consolidated domain layer.
2. **Repositories (Shared Pools)**:
   - Implement PostgreSQL repositories (`dataset_repo.go`, `job_repo.go`) sharing the same connection pool.
   - Implement Neo4j repository (`graph_repo.go`) for knowledge graph operations.
   - Implement pgvector repository (`vector_repo.go`) for embeddings (replacing standalone Qdrant).
3. **External Clients & Extractors**:
   - Implement MinIO storage adapter and format extractors (PDF, docx, etc.).
   - Implement LLM and Embedder clients (Bifrost).
4. **NATS Publisher**:
   - Implement NATS event publisher specifically for the `cognee.pipeline.completed` event to notify `cognee-search`.

## Acceptance Criteria
- [x] gRPC handlers correctly route to underlying combined usecases.
- [x] Shared PostgreSQL connection pool serves both dataset and job repositories.
- [x] NATS only publishes `cognee.pipeline.completed` (internal ingestion events removed).
