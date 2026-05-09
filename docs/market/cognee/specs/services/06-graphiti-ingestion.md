# 06 — Graphiti Ingestion Service

> **gRPC**: 9021 | **Health**: 9095

---

## 1. Purpose

Temporal episodic KG ingestion: nhận episodes (text/JSON/message), orchestrate qua saga pattern, gọi Knowledge Service để extract, và lưu qua Store Service.

---

## 2. Clean Architecture

```
services/graphiti-ingestion/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # Episode, Saga, StepResult
│   │   ├── value_object.go     # EpisodeType(TEXT,JSON,MSG), SagaState
│   │   ├── event.go            # EpisodeIngestedEvent
│   │   └── errors.go
│   ├── usecase/
│   │   ├── ingest_episode.go   # Orchestrate saga: extract → resolve → persist
│   │   ├── list_episodes.go
│   │   ├── bulk_ingest.go      # Batch with sequential group_id ordering
│   │   ├── port/
│   │   │   └── output.go       # SagaRepository, KnowledgeClient, StoreClient, EventPublisher
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/handler.go     # graphiti.ingestion.v1.IngestionService impl
│   │   ├── repository/
│   │   │   └── postgres/       # Saga state, episode metadata
│   │   ├── client/
│   │   │   ├── knowledge_client.go  # gRPC → graphiti-knowledge
│   │   │   └── store_client.go      # gRPC → graphiti-store
│   │   └── event/publisher.go       # NATS: graphiti.episode.ingested
│   └── infra/
│       ├── config/config.go
│       └── wire/wire.go
```

---

## 3. Saga Orchestration

```
IngestEpisode(episode)
     │
     ▼
┌─── Saga Steps ────────────────────────────────────┐
│ 1. ExtractEntities (→ knowledge-svc)              │
│    - LLM extracts entities from episode text       │
│    - Returns candidate entities + edges            │
│                                                    │
│ 2. ResolveEntities (→ knowledge-svc)              │
│    - Deduplicate with existing graph entities      │
│    - Merge/update entity properties                │
│                                                    │
│ 3. PersistToGraph (→ store-svc)                   │
│    - Write nodes, edges, episode to Neo4j          │
│    - Apply bi-temporal timestamps                  │
│    - Set valid_from/valid_to on edges              │
│                                                    │
│ 4. UpdateEmbeddings (→ knowledge-svc)             │
│    - Generate embeddings for new entities          │
│    - Index embeddings for search                   │
│                                                    │
│ 5. DetectCommunities (→ knowledge-svc, optional)  │
│    - Incremental community rebuild                 │
│    - LLM summary for affected communities         │
│                                                    │
│ 6. Emit EpisodeIngested                           │
└───────────────────────────────────────────────────┘
```

Each saga step has compensation logic for rollback on failure.

---

## 4. Domain: Saga State Machine

```go
type SagaState string
const (
    SagaPending   SagaState = "PENDING"
    SagaRunning   SagaState = "RUNNING"
    SagaCompleted SagaState = "COMPLETED"
    SagaFailed    SagaState = "FAILED"
    SagaRolledBack SagaState = "ROLLED_BACK"
)

type Saga struct {
    ID          uuid.UUID
    EpisodeID   uuid.UUID
    GroupID     string          // Graphiti's tenant isolation primitive
    State       SagaState
    CurrentStep int
    Steps       []StepResult
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

---

## 5. Group-ID Sequential Processing

Episodes with the same `group_id` are processed **sequentially** (ordered by `created_at`). Different `group_id` values are processed **concurrently**.

```go
// NATS consumer with key-based partitioning on group_id
// Ensures causal ordering within a group
```
