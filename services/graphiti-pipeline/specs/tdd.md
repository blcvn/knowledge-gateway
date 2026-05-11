---
id: TDD-graphiti-pipeline
title: Technical Design Document — graphiti-pipeline
service: graphiti-pipeline
version: 2.0.0
status: Ready
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
group: Graphiti
---

# graphiti-pipeline — Technical Design Document

> **Group**: Graphiti | **gRPC Port**: 9021 | **Health Port**: 9094
> **Origin**: Consolidated from graphiti-ingestion + graphiti-knowledge

## 1. Service Overview

Unified episodic knowledge ingestion and LLM-powered extraction service. Consolidates the ingestion saga orchestrator and knowledge processing engine into a single binary, converting 5 cross-service gRPC hops into local function calls.

**Key Characteristics:**
- 6-step saga pipeline: Extract Entities → Resolve Entities → Extract Edges → Resolve Edges → Generate Embeddings → Save + Update Community
- Per-group serialized ingestion ensuring consistency within tenant partitions
- Bi-temporal data model (valid_at, invalid_at, expired_at, created_at)
- Bifrost LLM gateway for provider-agnostic AI processing
- Compensating actions with saga state persistence

## 2. Clean Architecture Layers

### 2.1 Domain Layer (Layer 1) — ZERO external imports

```
internal/domain/
├── ingestion/
│   ├── entity.go          # Episode, EpisodeType, Saga, SagaState, PipelineStep
│   ├── value_object.go    # GroupID, EpisodeID, ContentHash
│   ├── event.go           # EpisodeIngested, EpisodeFailed, SagaStepCompleted
│   └── errors.go          # ErrDuplicateEpisode, ErrPipelineFailed, ErrGroupLocked
└── knowledge/
    ├── entity.go          # ExtractedEntity, ExtractedEdge, Resolution, DuplicateDecision
    ├── value_object.go    # PromptTemplate, TokenUsage, ModelConfig, EmbeddingDimension
    ├── embedding.go       # EmbeddingVector, EmbeddingRequest, EmbeddingResult
    ├── community.go       # CommunityNode, CommunityEdge, CommunityLevel
    └── errors.go          # ErrLLMTimeout, ErrPromptTooLong, ErrProviderUnavailable
```

### 2.2 Usecase Layer (Layer 2) — imports domain only

```
internal/usecase/
├── ingest/
│   ├── ingest_episode.go       # Single episode saga orchestration
│   ├── bulk_ingest.go          # Batch ingestion with cross-episode dedup
│   ├── get_status.go           # Episode status query
│   ├── list_episodes.go        # Episode listing with pagination
│   ├── remove_episode.go       # Episode removal with cascade
│   └── saga_orchestrator.go    # Generic saga state machine + compensations
├── knowledge/
│   ├── extract_entities.go     # LLM entity extraction (prompt → parse → validate)
│   ├── resolve_entities.go     # Search existing → LLM compare → merge/create
│   ├── extract_edges.go        # LLM fact triple extraction with temporal metadata
│   ├── resolve_edges.go        # Contradiction detection + invalidation
│   ├── generate_embedding.go   # Vector embedding via Bifrost embedder
│   ├── update_community.go     # Label propagation + LLM summarization
│   └── rerank.go               # Cross-encoder neural reranking
├── port/
│   ├── ingest_input.go         # IngestUseCase, BulkIngestUseCase, StatusUseCase
│   ├── ingest_output.go        # SagaStateRepo, EpisodeRepo, EventPublisher
│   ├── knowledge_input.go      # ExtractUseCase, ResolveUseCase, EmbedUseCase
│   └── knowledge_output.go     # LLMClient, EmbedderClient, StoreClient, GraphReader
└── dto/
    ├── ingest_dto.go
    └── knowledge_dto.go
```

**Key Port Interfaces:**

```go
// ingest_output.go
type StoreClient interface {
    SaveBulk(ctx context.Context, req SaveBulkRequest) error
    RollbackBulk(ctx context.Context, episodeID string) error
}

// knowledge_output.go
type LLMClient interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

type EmbedderClient interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
}

type GraphReader interface {
    FindSimilarEntities(ctx context.Context, groupID string, embedding []float32, limit int) ([]EntityNode, error)
    FindSimilarEdges(ctx context.Context, groupID string, embedding []float32, limit int) ([]EntityEdge, error)
    GetEntityByName(ctx context.Context, groupID string, name string) (*EntityNode, error)
}
```

### 2.3 Adapter Layer (Layer 3) — implements ports

```
internal/adapter/
├── grpc/
│   ├── ingestion_handler.go     # GraphitiIngestionService proto server
│   ├── knowledge_handler.go     # GraphitiKnowledgeService proto server
│   └── mapper.go                # Proto ↔ Domain bidirectional mapping
├── client/
│   └── store_client.go          # gRPC → graphiti-store:9024
├── repository/
│   ├── postgres/
│   │   ├── episode_repo.go      # Episode CRUD + dedup
│   │   ├── saga_repo.go         # Saga state persistence
│   │   └── migrations/          # SQL migration files
│   └── neo4j/
│       ├── entity_reader.go     # Entity reads for resolution
│       └── community_reader.go  # Community detection queries
├── llm/
│   ├── bifrost_client.go        # HTTP → Bifrost gateway
│   ├── prompt_registry.go       # Template management
│   └── response_parser.go      # JSON extraction from LLM responses
├── embedder/
│   └── bifrost_embedder.go      # Embedding via Bifrost
└── event/
    └── nats_publisher.go        # NATS JetStream publisher
```

### 2.4 Infrastructure Layer (Layer 4)

```
internal/infra/
├── config/config.go             # Viper loader + env validation
├── server/grpc.go               # gRPC server + interceptors (OTel, recovery, logging)
├── telemetry/
│   ├── tracer.go               # OTel tracer provider
│   └── metrics.go              # Prometheus counters/histograms
└── wire/wire.go                 # Google Wire DI + injector generation
```

## 3. gRPC API

```protobuf
// GraphitiIngestionService
service GraphitiIngestionService {
  rpc IngestEpisode(IngestEpisodeRequest) returns (IngestEpisodeResponse);
  rpc BulkIngest(stream BulkIngestRequest) returns (BulkIngestResponse);
  rpc GetEpisodeStatus(GetStatusRequest) returns (EpisodeStatus);
  rpc ListEpisodes(ListEpisodesRequest) returns (ListEpisodesResponse);
  rpc RemoveEpisode(RemoveEpisodeRequest) returns (RemoveEpisodeResponse);
}

// GraphitiKnowledgeService
service GraphitiKnowledgeService {
  rpc ExtractEntities(ExtractEntitiesRequest) returns (ExtractEntitiesResponse);
  rpc ResolveEntities(ResolveEntitiesRequest) returns (ResolveEntitiesResponse);
  rpc ExtractEdges(ExtractEdgesRequest) returns (ExtractEdgesResponse);
  rpc ResolveEdges(ResolveEdgesRequest) returns (ResolveEdgesResponse);
  rpc GenerateEmbedding(GenerateEmbeddingRequest) returns (GenerateEmbeddingResponse);
  rpc Rerank(RerankRequest) returns (RerankResponse);
  rpc UpdateCommunity(UpdateCommunityRequest) returns (UpdateCommunityResponse);
}
```

## 4. NATS Events

| Subject | Direction | Payload | Trigger |
|---------|-----------|---------|---------|
| `graphiti.episode.ingested` | Published | episode_id, group_id, nodes_count, edges_count | Saga completed |
| `graphiti.entity.resolved` | Published | entity_id, group_id, merged_from[] | Entity dedup merge |
| `graphiti.community.rebuilt` | Published | community_id, group_id, member_count | Community update |

## 5. Cross-Service Dependencies

| Service | Direction | Protocol | Operations |
|---------|-----------|----------|-----------|
| graphiti-store | Outbound | gRPC :9024 | SaveBulk, RollbackBulk |
| Bifrost | Outbound | HTTP | LLM completion, embedding generation |
| PostgreSQL | Outbound | SQL | Saga state, episode metadata, dedup |
| Neo4j | Outbound (read) | Bolt | Entity/edge similarity for resolution |
| NATS | Outbound | JetStream | Event publishing |

## 6. Observability

- **Metrics**: `graphiti_pipeline_saga_duration_seconds`, `graphiti_pipeline_saga_step_duration_seconds{step}`, `graphiti_pipeline_episodes_total{status}`, `graphiti_pipeline_llm_tokens_total{model}`, `graphiti_pipeline_circuit_breaker_state{target}`
- **Traces**: OTel spans for every usecase method + downstream calls, saga span grouping
- **Logs**: Structured JSON via slog: request_id, tenant_id, episode_id, saga_step, duration_ms
- **Health**: gRPC health + HTTP /healthz, /readyz, /livez on :9094

## 7. Multi-Tenancy

- `x-tenant-id` gRPC metadata → domain `GroupID`
- All queries scoped by `group_id`
- Per-group serialization mutex prevents cross-tenant interference
- Downstream calls propagate `group_id`

## Architecture Specs Registry

| ID | Title | Status |
|----|-------|--------|
| SOL-001 | Implement graphiti-pipeline | Approved |

## Feature Specs Registry

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| FEAT-PIP-001 | Domain layer | ⏳ Draft | P0 |
| FEAT-PIP-002 | Usecase layer | ⏳ Draft | P0 |
| FEAT-PIP-003 | gRPC handlers | ⏳ Draft | P0 |
| FEAT-PIP-004 | LLM adapter | ⏳ Draft | P0 |
| FEAT-PIP-005 | Repository adapters | ⏳ Draft | P0 |
| FEAT-PIP-006 | NATS publisher | ⏳ Draft | P1 |
| FEAT-PIP-007 | Store client | ⏳ Draft | P0 |
| FEAT-PIP-008 | Infrastructure | ⏳ Draft | P0 |

---

> **Next Steps**: Implement FEAT specs from SOL-001 in dependency order.
