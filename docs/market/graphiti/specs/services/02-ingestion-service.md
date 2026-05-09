# graphiti-ingestion — Ingestion Service

**Version:** 2.0 | **Date:** 2026-05-09  
**Origin:** Python L6 (Graphiti.add_episode/add_episode_bulk/add_triplet) + L5 partial (pipeline orchestration)  
**Architecture:** Clean Architecture | **Protocol:** gRPC

---

## 1. Service Overview

Ingestion Service là **pipeline orchestrator** cho toàn bộ quá trình ingestion. Nó quản lý episode lifecycle, điều phối extraction/resolution qua Knowledge Service, và persist kết quả qua Store Service.

### Responsibilities

| Concern | Description |
|---------|-------------|
| **Episode Ingestion** | Single + bulk episode processing pipeline |
| **Pipeline Orchestration** | Coordinate 9-step ingestion flow across services |
| **Serialization Control** | Ensure sequential processing per group_id |
| **Triplet Insertion** | Direct (S,P,O) triple insertion with resolution |
| **Saga Management** | Saga lifecycle, episode linking, incremental summarization |
| **Content Chunking** | Density-based chunking for large inputs |
| **Episode Lifecycle** | Create, retrieve, remove episodes with cascade |

---

## 2. Clean Architecture Layout

```
services/graphiti-ingestion/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── domain/                         # Layer 1: Entities
│   │   ├── episode.go                  #   RawEpisode, EpisodeMetadata
│   │   ├── saga.go                     #   Saga domain model
│   │   ├── pipeline.go                 #   PipelineState, PipelineStep enum
│   │   ├── chunk.go                    #   ContentChunk, ChunkConfig
│   │   └── errors.go                   #   IngestionError types
│   ├── usecase/                        # Layer 2: Use Cases
│   │   ├── ingest_episode.go           #   Single episode ingestion (9-step pipeline)
│   │   ├── ingest_episode_bulk.go      #   Bulk episode ingestion
│   │   ├── add_triplet.go             #   Direct triple insertion
│   │   ├── remove_episode.go           #   Episode removal with cascade
│   │   ├── list_episodes.go            #   Episode listing/retrieval
│   │   ├── manage_saga.go             #   Saga CRUD + summarization trigger
│   │   ├── chunk_content.go           #   Content chunking logic
│   │   ├── port/
│   │   │   ├── input.go               #   IngestionUseCase interfaces
│   │   │   └── output.go             #   KnowledgePort, StorePort, EventPort
│   │   └── dto/
│   │       ├── request.go             #   IngestEpisodeRequest, etc.
│   │       └── response.go            #   IngestResult, PipelineStats
│   ├── adapter/                        # Layer 3: Interface Adapters
│   │   ├── grpc/
│   │   │   ├── handler.go             #   gRPC service implementation
│   │   │   └── mapper.go             #   Proto ↔ Domain mapping
│   │   ├── client/
│   │   │   ├── knowledge_client.go    #   Knowledge Service gRPC client
│   │   │   └── store_client.go        #   Store Service gRPC client
│   │   ├── queue/
│   │   │   ├── worker.go             #   Per-group-id async worker
│   │   │   └── queue.go              #   In-memory queue with backpressure
│   │   └── event/
│   │       └── publisher.go          #   NATS publisher
│   └── infra/
│       ├── config/
│       │   └── config.go
│       ├── server/
│       │   └── grpc.go
│       ├── telemetry/
│       ├── middleware/
│       └── wire/
│           ├── wire.go
│           └── wire_gen.go
├── api/
│   └── proto/
│       └── ingestion/v1/
│           └── ingestion.proto
├── Dockerfile
└── Makefile
```

---

## 3. gRPC API (Protobuf)

```protobuf
syntax = "proto3";
package graphiti.ingestion.v1;

import "google/protobuf/timestamp.proto";
import "common/pagination.proto";
import "common/temporal.proto";

service IngestionService {
  // Episode lifecycle
  rpc IngestEpisode(IngestEpisodeRequest) returns (IngestEpisodeResponse);
  rpc IngestEpisodeBulk(IngestEpisodeBulkRequest) returns (IngestEpisodeBulkResponse);
  rpc IngestEpisodeStream(stream IngestEpisodeRequest) returns (stream IngestEpisodeResponse);
  rpc RemoveEpisode(RemoveEpisodeRequest) returns (RemoveEpisodeResponse);
  rpc ListEpisodes(ListEpisodesRequest) returns (ListEpisodesResponse);
  rpc GetEpisode(GetEpisodeRequest) returns (GetEpisodeResponse);
  
  // Triplet management
  rpc AddTriplet(AddTripletRequest) returns (AddTripletResponse);
  
  // Saga management
  rpc CreateSaga(CreateSagaRequest) returns (CreateSagaResponse);
  rpc SummarizeSaga(SummarizeSagaRequest) returns (SummarizeSagaResponse);
  rpc GetSaga(GetSagaRequest) returns (GetSagaResponse);
  
  // Pipeline status
  rpc GetPipelineStatus(GetPipelineStatusRequest) returns (GetPipelineStatusResponse);
}

message IngestEpisodeRequest {
  string name = 1;
  string body = 2;
  EpisodeSource source = 3;
  google.protobuf.Timestamp reference_time = 4;
  string group_id = 5;
  
  // Optional ontology constraints
  map<string, EntityTypeSchema> entity_types = 6;
  map<string, EdgeTypeSchema> edge_types = 7;
  
  // Saga binding
  optional string saga_id = 8;
  optional string saga_previous_episode_uuid = 9;
  
  // Processing options
  IngestionOptions options = 10;
}

message IngestionOptions {
  bool store_raw_content = 1;      // Persist raw episode text
  int32 chunk_token_size = 2;       // Override chunk size (default: 3000)
  int32 chunk_overlap_tokens = 3;   // Override overlap (default: 200)
  int32 context_window_size = 4;    // Previous episodes for context (default: 10)
}

message IngestEpisodeResponse {
  string episode_uuid = 1;
  PipelineStats stats = 2;
}

message PipelineStats {
  int32 entities_extracted = 1;
  int32 entities_resolved = 2;
  int32 entities_new = 3;
  int32 edges_extracted = 4;
  int32 edges_resolved = 5;
  int32 edges_new = 6;
  int32 communities_updated = 7;
  int64 processing_time_ms = 8;
  TokenUsage token_usage = 9;
}

enum EpisodeSource {
  EPISODE_SOURCE_UNSPECIFIED = 0;
  EPISODE_SOURCE_MESSAGE = 1;
  EPISODE_SOURCE_TEXT = 2;
  EPISODE_SOURCE_JSON = 3;
}
```

---

## 4. Episode Ingestion Pipeline (9 Steps)

### 4.1 Single Episode (`IngestEpisode`)

```
IngestEpisodeRequest
  │
  ├─1─► Content Chunking (local)
  │     IF content > CHUNK_MIN_TOKENS:
  │       split_by_density(content, chunk_size, overlap)
  │     ELSE:
  │       single chunk
  │
  ├─2─► Retrieve Context (→ Store)
  │     store.GetRecentEpisodes(group_id, last_n=context_window)
  │
  ├─3─► Extract Entities (→ Knowledge)
  │     knowledge.ExtractEntities(chunks, prev_episodes, entity_types)
  │     Returns: []ExtractedEntity{name, label, summary}
  │
  ├─4─► Resolve Entities (→ Knowledge → Store)
  │     FOR each extracted entity:
  │       candidates = store.SearchNodes(semantic + BM25)
  │       resolved = knowledge.ResolveEntity(entity, candidates)
  │     Returns: []ResolvedEntity + UUID mapping
  │
  ├─5─► Extract Edges (→ Knowledge)  ─── parallel with step 6
  │     knowledge.ExtractEdges(chunks, resolved_nodes, edge_types)
  │     Returns: []ExtractedEdge{source, target, fact, valid_at}
  │
  ├─6─► Extract Attributes (→ Knowledge)  ─── parallel with step 5
  │     knowledge.ExtractAttributes(resolved_nodes, new_edges)
  │     Returns: updated node summaries
  │
  ├─7─► Resolve Edges (→ Knowledge → Store)
  │     FOR each extracted edge:
  │       existing = store.GetEdgesBetweenNodes(src, tgt)
  │       resolved = knowledge.ResolveEdge(edge, existing)
  │       IF contradiction: store.InvalidateEdge(old_edge, reference_time)
  │
  ├─8─► Persist (→ Store)
  │     store.SaveBulk(episode, nodes, entity_edges, episodic_edges)
  │     Embedding generation happens in Knowledge before persist
  │
  └─9─► Update Community (→ Knowledge → Store)
        knowledge.UpdateCommunity(affected_entity_uuids, group_id)
```

### 4.2 Bulk Episode (`IngestEpisodeBulk`)

```
IngestEpisodeBulkRequest ([]RawEpisode)
  │
  ├─1─► Parallel Extraction (→ Knowledge)
  │     knowledge.ExtractEntitiesAndEdgesBulk(all_episodes)
  │
  ├─2─► In-Memory Dedup (→ Knowledge)
  │     knowledge.DedupeEntitiesBulk(all_extracted)
  │     → Cross-episode duplicate resolution before DB
  │
  ├─3─► Graph Resolution per episode (→ Knowledge → Store)
  │     FOR each episode: resolve_nodes + resolve_edges
  │
  ├─4─► Bulk Persist (→ Store)
  │     store.SaveBulkBatch(all_episodes, all_nodes, all_edges)
  │
  └─5─► (Community updates SKIPPED — run admin.RebuildCommunities separately)
```

### 4.3 Sequential Processing Guarantee

```go
// internal/adapter/queue/worker.go

// GroupWorker ensures sequential episode processing per group_id
// This is CRITICAL for graph consistency (dedup races)
type GroupWorker struct {
    mu       sync.RWMutex
    workers  map[string]*asyncWorker  // key: group_id
    maxQueue int                       // backpressure limit per group
}

type asyncWorker struct {
    queue    chan *IngestionJob
    done     chan struct{}
    groupID  string
}

// Enqueue returns error if queue is full (backpressure)
func (gw *GroupWorker) Enqueue(ctx context.Context, job *IngestionJob) error {
    worker := gw.getOrCreateWorker(job.GroupID)
    select {
    case worker.queue <- job:
        return nil
    default:
        return ErrQueueFull  // 429 to client
    }
}

// Each worker processes jobs sequentially for its group_id
func (w *asyncWorker) run(ctx context.Context) {
    for {
        select {
        case job := <-w.queue:
            w.processJob(ctx, job)  // blocking, one at a time
        case <-ctx.Done():
            return
        }
    }
}
```

---

## 5. Content Chunking

```go
// internal/usecase/chunk_content.go

type ChunkConfig struct {
    TokenSize          int     // default: 3000
    OverlapTokens      int     // default: 200
    MinTokens          int     // default: 1000
    DensityThreshold   float64 // default: 0.15
}

type ContentChunker interface {
    Chunk(content string, config ChunkConfig) []ContentChunk
}

// Density-based chunking:
// 1. Count tokens in content
// 2. If below MinTokens → return single chunk
// 3. Estimate entity density = extracted_entities / total_tokens
// 4. If density > DensityThreshold → use smaller chunks
// 5. Split with overlap for context continuity
```

---

## 6. Saga Management

```go
// internal/usecase/manage_saga.go

type SagaManager interface {
    // Auto-create saga on first episode with saga_id
    EnsureSaga(ctx context.Context, sagaID string, groupID string) (*Saga, error)
    
    // Link episode to saga (HAS_EPISODE + NEXT_EPISODE edges)
    LinkEpisode(ctx context.Context, sagaID string, episode *Episode, prevUUID string) error
    
    // Incremental LLM summary (only new episodes since last_summarized_at)
    Summarize(ctx context.Context, sagaID string) (*SagaSummary, error)
}

// Episode linking creates:
// SagaNode ──HAS_EPISODE──► EpisodicNode₁ ──NEXT_EPISODE──► EpisodicNode₂
```

---

## 7. Domain Events Published

| Event | Payload | Published After |
|-------|---------|-----------------|
| `episode.ingested` | `{episode_uuid, group_id, stats}` | Successful single episode ingestion |
| `episode.bulk_ingested` | `{episode_uuids, group_id, count}` | Successful bulk ingestion |
| `episode.removed` | `{episode_uuid, group_id, cascade_count}` | Episode removal |
| `saga.updated` | `{saga_id, group_id, episode_count}` | Episode added to saga |

---

## 8. Configuration

```yaml
# config/ingestion.yaml
server:
  grpc_port: 9001
  max_concurrent_groups: 100       # max group workers
  queue_size_per_group: 50         # max queue depth per group_id
  shutdown_timeout: 30s

chunking:
  token_size: 3000
  overlap_tokens: 200
  min_tokens: 1000
  density_threshold: 0.15

pipeline:
  context_window_size: 10           # previous episodes for context
  max_concurrent_steps: 20          # semaphore for parallel steps
  store_raw_content: true
  timeout: 300s                     # max pipeline execution time

services:
  knowledge:
    address: "graphiti-knowledge:9003"
    timeout: 120s
  store:
    address: "graphiti-store:9004"
    timeout: 30s

events:
  nats_url: "nats://nats:4222"
  stream: "graphiti-ingestion"

telemetry:
  otel_endpoint: "otel-collector:4317"
  service_name: "graphiti-ingestion"
```

---

## 9. Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ingestion_episodes_total` | Counter | group_id, source, status | Total episodes processed |
| `ingestion_pipeline_duration_seconds` | Histogram | step | Per-step latency |
| `ingestion_pipeline_total_duration_seconds` | Histogram | source | End-to-end latency |
| `ingestion_entities_extracted_total` | Counter | group_id | Entities extracted |
| `ingestion_edges_extracted_total` | Counter | group_id | Edges extracted |
| `ingestion_queue_depth` | Gauge | group_id | Current queue depth |
| `ingestion_active_workers` | Gauge | — | Active group workers |
| `ingestion_tokens_used_total` | Counter | prompt_type | LLM tokens consumed |

---

## 10. Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Per-group-id sequential processing** | Prevents dedup races — same as Python AsyncWorker pattern |
| **Pipeline as orchestrator, not processor** | Ingestion coordinates; Knowledge does the LLM work; Store does persistence |
| **Content chunking in Ingestion, not Knowledge** | Chunking is a pipeline concern; Knowledge works on individual chunks |
| **NATS for events, not for pipeline steps** | Pipeline steps need sync response; events are fire-and-forget notifications |
| **Streaming gRPC for bulk** | Allows backpressure, progress reporting, partial failure handling |
| **Saga state in Store** | Saga is a graph structure; Ingestion orchestrates but doesn't own data |
