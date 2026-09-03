---
id: TDD-graphiti-ingestion
title: Technical Design — graphiti-ingestion
service: graphiti-ingestion
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-09
group: Graphiti
---

# Technical Design — graphiti-ingestion

> **Group**: Graphiti | **gRPC Port**: 9021 | **Health Port**: 9094 | **Origin**: Graphiti

## 1. Service Overview

Episode ingestion pipeline orchestrator using the Saga pattern. Entry point for all episodic knowledge into the Graphiti temporal knowledge graph. Coordinates entity extraction, resolution, edge extraction, edge resolution, bulk persistence, and community updates across `graphiti-knowledge` and `graphiti-store` services.

**Key Characteristics:**
- Per-group serialized ingestion ensuring consistency within tenant partitions
- 7-step saga pipeline with compensating actions for each step
- Support for message, JSON, text, and fact_triple episode types
- Saga-based episode grouping for conversation threading
- Idempotent processing with deduplication by (name, group_id, valid_at) hash

## 2. Clean Architecture Layers

### 2.1 Domain Layer (Layer 1)

```
internal/domain/
├── entity.go                 # Episode, EpisodeType, SagaState, PipelineStep
├── value_object.go           # GroupID, EpisodeID, ContentHash
├── event.go                  # EpisodeIngested, EpisodeFailed, SagaStepCompleted
├── errors.go                 # ErrDuplicateEpisode, ErrPipelineFailed, ErrGroupLocked
└── repository.go             # SagaStateRepository interface (port)
```

**Key Domain Types (from Graphiti reference):**
- `EpisodeType`: message | json | text | fact_triple
- `PipelineStep`: EXTRACT_ENTITIES | RESOLVE_ENTITIES | EXTRACT_EDGES | RESOLVE_EDGES | SAVE_BULK | UPDATE_COMMUNITY
- `SagaStatus`: QUEUED | PROCESSING | COMPLETED | FAILED | COMPENSATING

### 2.2 Usecase Layer (Layer 2)

```
internal/usecase/
├── ingest_episode.go          # Single episode saga orchestration
├── bulk_ingest.go             # Batch ingestion with cross-episode dedup
├── get_status.go              # Episode status query
├── saga_orchestrator.go       # Generic saga state machine
├── port/
│   ├── input.go              # IngestUseCase, BulkIngestUseCase, StatusUseCase
│   └── output.go             # KnowledgeClient, StoreClient, EventPublisher, EpisodeRepo
└── dto/
    ├── request.go            # IngestEpisodeRequest, BulkIngestRequest
    └── response.go           # IngestEpisodeResponse, EpisodeStatus
```

**Output Ports (interfaces defined here, implemented in adapter layer):**
```go
type KnowledgeClient interface {
    ExtractEntities(ctx context.Context, content string, groupID string) ([]EntityNode, error)
    ResolveEntities(ctx context.Context, extracted []EntityNode, groupID string) ([]EntityNode, map[string]string, error)
    ExtractEdges(ctx context.Context, episode EpisodicNode, resolvedNodes []EntityNode) ([]EntityEdge, error)
    ResolveEdges(ctx context.Context, extractedEdges []EntityEdge, groupID string) ([]EntityEdge, error)
    UpdateCommunity(ctx context.Context, affectedEntities []string) error
}

type StoreClient interface {
    SaveBulk(ctx context.Context, nodes []EntityNode, edges []EntityEdge, episode EpisodicNode) error
    RollbackBulk(ctx context.Context, episodeID string) error
}

type EventPublisher interface {
    PublishEpisodeIngested(ctx context.Context, event EpisodeIngestedEvent) error
}
```

### 2.3 Adapter Layer (Layer 3)

```
internal/adapter/
├── grpc/
│   ├── handler.go             # gRPC server implementation
│   └── mapper.go              # Proto ↔ Domain bidirectional mapping
├── client/
│   ├── knowledge_client.go    # gRPC client → graphiti-knowledge:9023
│   └── store_client.go        # gRPC client → graphiti-store:9024
├── repository/
│   └── postgres/
│       └── episode_repo.go    # SagaState persistence + episode dedup
└── event/
    └── nats_publisher.go      # NATS JetStream publisher
```

### 2.4 Infrastructure Layer (Layer 4)

```
internal/infra/
├── config/config.go            # Viper configuration loader + validation
├── server/grpc.go              # gRPC server setup with interceptors
├── telemetry/                  # OTel tracer + Prometheus registry
└── wire/wire.go                # Google Wire DI providers + injector
```

## 3. gRPC API

```protobuf
service GraphitiIngestionService {
  rpc IngestEpisode(IngestEpisodeRequest) returns (IngestEpisodeResponse);
  rpc BulkIngest(stream BulkIngestRequest) returns (BulkIngestResponse);
  rpc GetEpisodeStatus(GetStatusRequest) returns (EpisodeStatus);
}
```

## 4. NATS Events

### Published

| Subject | Payload | Trigger |
|---------|---------|---------|
| `graphiti.episode.ingested` | `{episode_id, group_id, nodes_count, edges_count, timestamp}` | Saga completed successfully |

### Subscribed

None — this service is the ingestion entry point.

## 5. Data Model

### PostgreSQL Tables

| Table | Purpose |
|-------|---------|
| `graphiti_saga_state` | Saga step tracking, retry counts, error logs |
| `graphiti_episode_dedup` | Content hash → episode_id deduplication index |

### Graph Labels (via graphiti-store)

| Label | Managed By |
|-------|-----------|
| `Episodic` | Created here, persisted via graphiti-store |
| `Saga` | Created here for episode grouping |

## 6. Cross-Service Dependencies

| Service | Direction | Protocol | Operations |
|---------|-----------|----------|-----------|
| `graphiti-knowledge` | Outbound gRPC | 9023 | ExtractEntities, ResolveEntities, ExtractEdges, ResolveEdges, UpdateCommunity |
| `graphiti-store` | Outbound gRPC | 9024 | SaveBulk, RollbackBulk |
| NATS JetStream | Outbound | `graphiti` stream | Publish `graphiti.episode.ingested` |
| PostgreSQL | Outbound | SQL | Saga state, deduplication |

## 7. Observability

- **Metrics**: Prometheus counters/histograms for all RPCs + saga steps
  - `graphiti_ingestion_episodes_total{status}` — episode counter by outcome
  - `graphiti_ingestion_saga_duration_seconds` — end-to-end pipeline latency
  - `graphiti_ingestion_saga_step_duration_seconds{step}` — per-step breakdown
  - `graphiti_ingestion_circuit_breaker_state{service}` — downstream CB state
- **Traces**: OTel spans for every usecase method + downstream gRPC calls
- **Logs**: Structured JSON via slog with `request_id`, `tenant_id`, `episode_id`, `saga_step`
- **Health**: gRPC health check + HTTP `/healthz`, `/readyz`, `/livez` on port 9094

## 8. Multi-Tenancy

Tenant isolation via gRPC metadata `x-tenant-id` → mapped to Graphiti `group_id`.
- All saga state queries scoped by `group_id`
- Per-group serialization mutex prevents cross-tenant interference
- Downstream calls propagate `group_id` to graphiti-knowledge and graphiti-store

---

> **Next Steps**: Decompose this TDD into individual FEAT/ARCH specs in `specs/features/` and `specs/architecture/` before implementation.
