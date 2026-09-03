# Change Request: CR-GR-001 — Episode Ingestion Pipeline (9-Step)

**CR ID:** CR-GR-001  
**Component:** `services/graphiti-ingestion` [NEW SERVICE]  
**Priority:** Critical  
**Status:** In Progress
**Reference:** graphiti PRD §5.1, SRS §3.1, specs/services/02-ingestion-service.md  
**Maps to Python:** `graphiti_core/graphiti.py` — `add_episode()`, `add_episode_bulk()`, `add_triplet()`

---

## 1. Mô tả

Xây dựng **graphiti-ingestion** service — pipeline orchestrator cho toàn bộ quá trình ingest episodic data vào knowledge graph. Service này điều phối 9-step pipeline: chunk → context → extract entities → resolve entities → extract edges ↕ extract attributes → resolve edges → persist → update community.

**4 loại episode source:** `text`, `json`, `message`, `fact_triple`

---

## 2. Vấn đề hiện tại

`services/graph-service` hiện tại trong VNP Memory (cognee-based) hỗ trợ basic graph ingestion nhưng:
- ❌ Không có **9-step orchestration pipeline** đầy đủ.
- ❌ Không có **per-group-id sequential processing** (race condition khi nhiều agents cùng ingest).
- ❌ Không có **content chunking** (density-based cho large documents).
- ❌ Không có **Saga management** (nhóm episodes liên quan theo sequence).
- ❌ Không có **direct triplet insertion** (`add_triplet` API).
- ❌ Không có **bulk episode processing** với in-memory dedup.
- ❌ Không có streaming gRPC cho bulk import.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/graphiti-ingestion/`

**Port:** `9001` (gRPC internal)  
**Binary:** `cmd/server/main.go`

**Cấu trúc (Clean Architecture):**
```
services/graphiti-ingestion/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── episode.go          # RawEpisode, EpisodeMetadata, EpisodeType
│   │   ├── saga.go             # Saga, SagaSummary
│   │   ├── pipeline.go         # PipelineState, PipelineStep enum, PipelineStats
│   │   ├── chunk.go            # ContentChunk, ChunkConfig
│   │   └── errors.go           # IngestionError types
│   ├── usecase/
│   │   ├── ingest_episode.go   # 9-step pipeline orchestrator
│   │   ├── ingest_episode_bulk.go
│   │   ├── add_triplet.go      # Direct (S,P,O) triple insertion
│   │   ├── remove_episode.go   # Episode cascade delete
│   │   ├── list_episodes.go
│   │   ├── manage_saga.go      # Saga CRUD + summarization trigger
│   │   ├── chunk_content.go    # Density-based chunking
│   │   └── port/
│   │       ├── input.go        # IngestionUseCase interfaces
│   │       └── output.go       # KnowledgePort, StorePort, EventPort
│   ├── adapter/
│   │   ├── grpc/handler.go     # gRPC service implementation
│   │   ├── client/
│   │   │   ├── knowledge_client.go  # Knowledge gRPC client
│   │   │   └── store_client.go      # Store gRPC client
│   │   ├── queue/
│   │   │   ├── worker.go       # Per-group-id async worker (sequential)
│   │   │   └── queue.go        # In-memory queue with backpressure
│   │   └── event/publisher.go  # NATS publisher
│   └── infra/
│       ├── config/config.go
│       └── wire/
├── api/proto/ingestion/v1/ingestion.proto
└── Makefile
```

### 3.2. Domain Models

```go
// internal/domain/episode.go

type EpisodeType string
const (
    EpisodeTypeMessage   EpisodeType = "message"
    EpisodeTypeText      EpisodeType = "text"
    EpisodeTypeJSON      EpisodeType = "json"
    EpisodeTypeFactTriple EpisodeType = "fact_triple"
)

type RawEpisode struct {
    Name             string
    Body             string
    Source           EpisodeType
    SourceDescription string
    ReferenceTime    time.Time
    GroupID          string
    SagaID           string     // optional
    PrevEpisodeUUID  string     // optional, for saga linking
    EntityTypes      map[string]EntityTypeSchema  // custom ontology
    EdgeTypes        map[string]EdgeTypeSchema
    Options          IngestionOptions
}

type IngestionOptions struct {
    StoreRawContent    bool
    ChunkTokenSize     int  // default: 3000
    ChunkOverlapTokens int  // default: 200
    ContextWindowSize  int  // default: 10
}

type PipelineStats struct {
    EntitiesExtracted  int
    EntitiesResolved   int
    EntitiesNew        int
    EdgesExtracted     int
    EdgesResolved      int
    EdgesNew           int
    CommunitiesUpdated int
    ProcessingTimeMs   int64
    TokenUsage         TokenUsage
}
```

### 3.3. 9-Step Ingestion Pipeline

```go
// internal/usecase/ingest_episode.go

func (uc *IngestEpisodeUseCase) Execute(ctx context.Context, req RawEpisode) (*IngestResult, error) {
    // [1] Content Chunking (local)
    //     IF tokens(content) > CHUNK_MIN_TOKENS (1000):
    //       chunks = chunk_by_density(content, chunk_size=3000, overlap=200)
    //     ELSE:
    //       chunks = [single_chunk]

    // [2] Retrieve Context (→ Store)
    //     prev_episodes = store.RetrieveEpisodes(group_id, last_n=context_window)

    // [3] Extract Entities (→ Knowledge)
    //     entities = knowledge.ExtractEntities(chunks, prev_episodes, entity_types)
    //     → []ExtractedEntity{name, label, summary}

    // [4] Resolve Entities (→ Knowledge + Store)
    //     FOR each entity:
    //       candidates = store.NodeSimilaritySearch(entity.name_embedding) + NodeFulltextSearch
    //       resolved = knowledge.ResolveEntity(entity, candidates)
    //     → UUID mapping (existing OR new)

    // [5] Extract Edges (→ Knowledge)  ─── PARALLEL with step 6
    //     edges = knowledge.ExtractEdges(chunks, resolved_nodes, edge_types)
    //     → []ExtractedEdge{source, target, relation, fact, valid_at}

    // [6] Extract Attributes (→ Knowledge)  ─── PARALLEL with step 5
    //     updated_summaries = knowledge.ExtractAttributes(resolved_nodes, new_edges)

    // [7] Resolve Edges (→ Knowledge + Store)
    //     FOR each edge:
    //       existing = store.GetEdgesBetweenNodes(src_uuid, tgt_uuid)
    //       resolution = knowledge.ResolveEdge(edge, existing, reference_time)
    //       IF contradiction: store.InvalidateEntityEdge(old_uuid, reference_time)

    // [8] Persist (→ Store)
    //     store.SaveBulk(episode, entity_nodes, entity_edges, episodic_edges, saga?)
    //     (embedding generation happens in Knowledge before step 8)

    // [9] Update Community (→ Knowledge + Store)
    //     knowledge.UpdateCommunity(affected_entity_uuids, group_id)
}
```

### 3.4. Per-Group-ID Sequential Worker

```go
// internal/adapter/queue/worker.go
// Critical: prevents dedup races between concurrent episodes

type GroupWorker struct {
    mu       sync.RWMutex
    workers  map[string]*asyncWorker  // key: group_id
    maxQueue int                       // backpressure limit per group
}

// Each group_id has its own channel — sequential processing
// If queue full: return 429 (backpressure)
// Max groups: 100 (configurable)
// Max queue per group: 50 (configurable)
```

### 3.5. Content Chunking (Density-Based)

```go
// internal/usecase/chunk_content.go

type ChunkConfig struct {
    TokenSize        int     // default: 3000 tokens
    OverlapTokens    int     // default: 200 tokens (context continuity)
    MinTokens        int     // default: 1000 (below = single chunk)
    DensityThreshold float64 // default: 0.15 (high-density = smaller chunks)
}

// Algorithm:
// 1. Count tokens (tiktoken-compatible, approximate by char/4)
// 2. If tokens < MinTokens → single chunk
// 3. Estimate entity density: extracted_entities / total_tokens
// 4. If density > DensityThreshold → use smaller chunk_size
// 5. Split with overlap for cross-chunk context continuity
```

### 3.6. Saga Management

```go
// internal/usecase/manage_saga.go
// Saga groups related episodes into ordered sequences

type SagaManager interface {
    EnsureSaga(ctx, sagaID, groupID) (*Saga, error)
    LinkEpisode(ctx, sagaID, episode, prevEpisodeUUID) error
    Summarize(ctx, sagaID) (*SagaSummary, error) // incremental LLM summary
}

// Episode linking creates graph edges:
// SagaNode ──HAS_EPISODE──► EpisodicNode₁ ──NEXT_EPISODE──► EpisodicNode₂

// Summarize: only processes new episodes since last_summarized_at
// → delegates to Knowledge.SummarizeSaga RPC
```

### 3.7. Bulk Episode Processing

```go
// internal/usecase/ingest_episode_bulk.go
// Optimized path for batch import (skips community detection)

// [1] Parallel extraction (→ Knowledge.ExtractEntitiesAndEdgesBulk)
// [2] In-memory dedup across all episodes (→ Knowledge.DedupeEntitiesBulk)
// [3] Graph resolution per episode
// [4] Bulk persist (→ Store.SaveBulk batch)
// [5] Community detection SKIPPED → run admin.RebuildCommunities separately

// Streaming gRPC for progress reporting + partial failure handling
```

### 3.8. Direct Triplet Insertion

```go
// internal/usecase/add_triplet.go
// API: AddTriplet(source_node, edge, target_node, group_id)
// Fast path: no LLM extraction, directly resolves and persists
// Source/target may be new or existing nodes (by name + label matching)
```

### 3.9. gRPC API (Protobuf)

```protobuf
service IngestionService {
    rpc IngestEpisode(IngestEpisodeRequest) returns (IngestEpisodeResponse);
    rpc IngestEpisodeBulk(IngestEpisodeBulkRequest) returns (IngestEpisodeBulkResponse);
    rpc IngestEpisodeStream(stream IngestEpisodeRequest) returns (stream IngestEpisodeResponse);
    rpc RemoveEpisode(RemoveEpisodeRequest) returns (RemoveEpisodeResponse);
    rpc ListEpisodes(ListEpisodesRequest) returns (ListEpisodesResponse);
    rpc GetEpisode(GetEpisodeRequest) returns (GetEpisodeResponse);
    rpc AddTriplet(AddTripletRequest) returns (AddTripletResponse);
    rpc CreateSaga(CreateSagaRequest) returns (CreateSagaResponse);
    rpc SummarizeSaga(SummarizeSagaRequest) returns (SummarizeSagaResponse);
    rpc GetSaga(GetSagaRequest) returns (GetSagaResponse);
    rpc GetPipelineStatus(GetPipelineStatusRequest) returns (GetPipelineStatusResponse);
}
```

### 3.10. NATS Events Published

| Subject | Payload |
|---|---|
| `graphiti.episode.ingested` | `{episode_uuid, group_id, stats}` |
| `graphiti.episode.bulk_ingested` | `{episode_uuids[], group_id, count}` |
| `graphiti.episode.removed` | `{episode_uuid, group_id, cascade_count}` |
| `graphiti.saga.updated` | `{saga_id, group_id, episode_count}` |

### 3.11. Gateway REST Endpoints (via graphiti-gateway)

```
POST /v1/episodes
  Body: { name, body, source, source_description, reference_time, group_id, saga_id? }
  Response: { episode_uuid, stats: PipelineStats }

POST /v1/episodes/bulk
  Body: { episodes: [...], group_id }
  Response: [IngestResult]

POST /v1/triplets
  Body: { source, edge, target, group_id }
  Response: { episode_uuid }

DELETE /v1/episodes/{uuid}
  Response: 204 No Content

GET /v1/episodes?group_id=...&last_n=10
  Response: [EpisodeNode]

POST /v1/sagas
POST /v1/sagas/{id}/summarize
GET  /v1/sagas/{id}
```

---

## 4. Configuration

```yaml
server:
  grpc_port: 9001
  max_concurrent_groups: 100
  queue_size_per_group: 50
  shutdown_timeout: 30s

chunking:
  token_size: 3000
  overlap_tokens: 200
  min_tokens: 1000
  density_threshold: 0.15

pipeline:
  context_window_size: 10
  max_concurrent_steps: 20
  store_raw_content: true
  timeout: 300s

services:
  knowledge: { address: "graphiti-knowledge:9003", timeout: 120s }
  store:     { address: "graphiti-store:9004", timeout: 30s }

events:
  nats_url: "nats://nats:4222"
```

---

## 5. Metrics

| Metric | Type | Labels |
|---|---|---|
| `ingestion_episodes_total` | Counter | group_id, source, status |
| `ingestion_pipeline_duration_seconds` | Histogram | step |
| `ingestion_entities_extracted_total` | Counter | group_id |
| `ingestion_edges_extracted_total` | Counter | group_id |
| `ingestion_queue_depth` | Gauge | group_id |
| `ingestion_tokens_used_total` | Counter | prompt_type |

---

## 6. Acceptance Criteria

- [ ] `POST /v1/episodes` với body text → trả về `episode_uuid` + `stats.entities_new ≥ 1`.
- [ ] Ingest episode "Alice joined engineering in March" → `EntityNode` cho "Alice" tồn tại trong graph với `summary`.
- [ ] Ingest episode mâu thuẫn "Alice left engineering in June" → edge cũ có `invalid_at` set, edge mới tạo.
- [ ] 2 episodes với `group_id = "project-alpha"` được xử lý tuần tự (không race condition).
- [ ] Content > 1000 tokens → tự động chia chunk với overlap.
- [ ] `POST /v1/episodes/bulk` với 100 episodes → hoàn thành không lỗi, `entities` deduplicated across episodes.
- [ ] `POST /v1/triplets` với `(Alice, WORKS_AT, Acme Corp)` → EntityEdge được tạo trong graph.
- [ ] Ingest với `saga_id = "sprint-1"` → SagaNode được tạo, HAS_EPISODE edge liên kết episode.
- [ ] Saga summary sau 3 episodes → `summary` field populated trong SagaNode.
- [ ] Queue full (>50 per group): `POST /v1/episodes` → 429 Too Many Requests.
- [ ] `DELETE /v1/episodes/{uuid}` → episode + cascade (episodic edges) bị xóa.
