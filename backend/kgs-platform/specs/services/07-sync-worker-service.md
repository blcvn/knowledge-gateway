# sync-worker-service — Sync Worker Service

> **Role:** Background worker xử lý bất đồng bộ: đồng bộ dữ liệu từ PostgreSQL (source-of-truth) sang Neo4j và Qdrant qua Outbox pattern, batch processing, và reconciliation định kỳ.

---

## 1. Trách Nhiệm (Single Responsibility)

`sync-worker-service` là **background worker thuần túy** — không có gRPC server, không expose API. Chịu trách nhiệm:
- **Outbox Worker**: Poll `kg_sync_outbox` table và sync records sang Neo4j và Qdrant
- **Batch Processor**: Xử lý bulk import/export theo yêu cầu
- **Reconcile Job**: Kiểm tra và sửa inconsistency giữa PostgreSQL ↔ Neo4j ↔ Qdrant
- **Distributed Lock**: Đảm bảo chỉ một instance xử lý một record tại một thời điểm

---

## 2. Kiến Trúc Nội Tại

```
┌─────────────────────────────────────────────────────────────────────────┐
│                       sync-worker-service                                │
│                     (No gRPC/HTTP server — background only)             │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     OutboxWorker                                  │   │
│  │                                                                  │   │
│  │  Poll interval: 500ms                                            │   │
│  │  Batch size: 100 records per poll                                │   │
│  │                                                                  │   │
│  │  kg_sync_outbox (PostgreSQL)                                     │   │
│  │       │                                                          │   │
│  │       ├── UPSERT_ENTITY ──→ neo4jSyncer.UpsertEntity()          │   │
│  │       │                        ↓ Neo4j                           │   │
│  │       │                     qdrantSyncer.UpsertVector()          │   │
│  │       │                        ↓ Qdrant                          │   │
│  │       │                                                          │   │
│  │       ├── UPSERT_EDGE   ──→ neo4jSyncer.UpsertEdge()            │   │
│  │       │                        ↓ Neo4j                           │   │
│  │       │                                                          │   │
│  │       ├── DELETE_ENTITY ──→ neo4jSyncer.SoftDeleteEntity()      │   │
│  │       │                     qdrantSyncer.DeleteVector()          │   │
│  │       │                                                          │   │
│  │       └── DELETE_EDGE   ──→ neo4jSyncer.DeleteEdge()            │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     ReconcileJob                                  │   │
│  │                                                                  │   │
│  │  Schedule: Daily at 2AM (cron: "0 2 * * *")                     │   │
│  │                                                                  │   │
│  │  1. Count PostgreSQL entities vs Neo4j nodes per app_id         │   │
│  │  2. Find missing records → re-queue in outbox                   │   │
│  │  3. Find orphan Neo4j nodes (PG deleted, Neo4j not) → delete    │   │
│  │  4. Report inconsistency metrics                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     BatchProcessor                               │   │
│  │                                                                  │   │
│  │  Triggered by NATS: sync.batch.requested                        │   │
│  │                                                                  │   │
│  │  1. Read batch job spec from PostgreSQL (batch_jobs table)      │   │
│  │  2. Process in chunks (chunk_size = 500)                        │   │
│  │  3. Dedup: skip if entity_id already exists in Neo4j            │   │
│  │  4. Bulk write Neo4j (UNWIND ... CREATE/MERGE)                  │   │
│  │  5. Bulk index Qdrant (batch upsert vectors)                    │   │
│  │  6. Update batch_job status                                     │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                  DistributedLock (Redis)                          │   │
│  │                                                                  │   │
│  │  Lock key: "sync:outbox:{record_id}"                            │   │
│  │  TTL: 30 seconds (auto-release if worker crashes)               │   │
│  │  Purpose: prevent duplicate processing in multi-instance deploy  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Outbox Pattern Detail

### 3.1 Outbox Record Lifecycle

```
[graph-service writes]
      │
      ▼
kg_sync_outbox: status=PENDING, attempts=0
      │
      ▼ (poll by OutboxWorker, every 500ms)
status=PROCESSING (set với distributed lock)
      │
      ├─ Success ──→ status=DONE, processed_at=now()
      │
      └─ Failure ──→ status=PENDING, attempts++
                     If attempts >= max_attempts (10) → status=FAILED
                     Alert: sync.dead_letter metric
```

### 3.2 UPSERT_ENTITY to Neo4j

```cypher
// Cypher được execute bởi OutboxWorker:
MERGE (n:`{neo4j_label}` {id: $id})
ON CREATE SET
    n += $properties,
    n.created_at = $created_at,
    n.kgs_namespace = $namespace
ON MATCH SET
    n += $properties,
    n.updated_at = $updated_at

// neo4j_label example: "ba_agent__Requirement"
// Đã được graph-service inject trước khi write outbox
```

### 3.3 UPSERT_ENTITY to Qdrant (Vector Indexing)

```go
// qdrantSyncer.UpsertVector():
// 1. Extract searchable text từ properties
//    text = entity_type + " " + join(searchable_props values)
// 2. Generate embedding vector via embedding provider
//    vector = embeddingProvider.Embed(text) // OpenAI / AIProxy / Air-VNP
// 3. Upsert to Qdrant collection
//    collection = "kgs_{app_id}"
//    point = { id: entity_id, vector: vector, payload: {namespace, entity_type, ...} }
```

### 3.4 Retry Strategy

| Attempt | Delay before retry |
|---------|------------------|
| 1 | Immediate |
| 2 | 5 seconds |
| 3 | 30 seconds |
| 4-5 | 2 minutes |
| 6-10 | 10 minutes |
| > 10 | status=FAILED (Dead Letter) |

---

## 4. Batch Processor

### 4.1 Batch Job Model

```go
type BatchJob struct {
    ID          uint      `gorm:"primaryKey"`
    AppID       string    `gorm:"type:varchar(50);not null;index"`
    JobType     string    `gorm:"type:varchar(50);not null"`
    // BULK_IMPORT_NODES | BULK_IMPORT_EDGES | REINDEX_VECTORS
    Status      string    `gorm:"type:varchar(20);default:'PENDING'"`
    // PENDING | RUNNING | SUCCESS | FAILED | PARTIAL
    TotalCount  int
    ProcessedCount int
    FailedCount int
    Payload     datatypes.JSON `gorm:"type:jsonb"` // Job-specific params
    ErrorLog    string    `gorm:"type:text"`
    StartedAt   *time.Time
    CompletedAt *time.Time
    CreatedAt   time.Time
}
```

### 4.2 Bulk Import Flow

```
1. App tạo BatchJob via API (hoặc file upload)
2. BatchJob saved to PostgreSQL (status=PENDING)
3. NATS event: "sync.batch.requested" {job_id}
4. BatchProcessor picks up job:
   a. Mark status=RUNNING
   b. Load entities from job payload (JSON/CSV)
   c. Dedup check: filter out existing entity_ids
   d. Process in chunks of 500:
      - Write chunk to kg_entities (PostgreSQL)
      - Bulk UNWIND MERGE into Neo4j
      - Batch embed and upsert to Qdrant
   e. Update progress (processed_count) every chunk
5. Mark status=SUCCESS|FAILED|PARTIAL
6. Notify app via webhook (if configured)
```

---

## 5. Reconcile Job

### 5.1 Consistency Checks

```
Daily Reconcile Job (2 AM):

For each app_id:
  1. Count PG entities: SELECT COUNT(*) FROM kg_entities WHERE app_id=? AND deleted_at IS NULL
  2. Count Neo4j nodes: MATCH (n) WHERE any(l IN labels(n) WHERE l STARTS WITH '{app_id}__') RETURN count(n)
  3. Count Qdrant points: collection.count(filter={namespace: "graph/{app_id}/*"})

  If PG_count != Neo4j_count:
    → Find missing: SELECT id FROM kg_entities WHERE id NOT IN (Neo4j IDs)
    → Re-queue missing records in kg_sync_outbox

  If Neo4j_count > PG_count:
    → Find orphans: Neo4j nodes with no PG record (soft-deleted)
    → Mark orphan Neo4j nodes for deletion
```

### 5.2 Reconcile Report

```json
{
  "app_id": "ba_agent",
  "run_at": "2026-06-11T02:00:00Z",
  "duration_seconds": 45,
  "pg_entity_count": 1520,
  "neo4j_node_count": 1518,
  "qdrant_point_count": 1515,
  "discrepancy": {
    "missing_in_neo4j": 2,
    "orphan_in_neo4j": 0,
    "missing_in_qdrant": 5
  },
  "actions_taken": {
    "re_queued_for_sync": 2,
    "qdrant_reindex_queued": 5
  },
  "status": "REPAIRED"
}
```

---

## 6. Embedding Providers

`sync-worker-service` hỗ trợ multiple embedding providers qua factory pattern:

```go
type EmbeddingProvider interface {
    Embed(text string) ([]float32, error)
    Dimension() int
}

// Providers (từ code thực tế):
// - OpenAI (text-embedding-ada-002, dimension=1536)
// - AIProxy (OpenAI-compatible proxy)
// - Air-VNP (local VNP embedding model)

// Selection via config:
// embedding.provider: openai | aiproxy | air-vnp
```

---

## 7. Configuration

```yaml
# configs/sync-worker.yaml
sync_worker:
  # Outbox Worker
  outbox:
    poll_interval: 500ms
    batch_size: 100
    max_attempts: 10
    worker_concurrency: 10

  # Reconcile
  reconcile:
    cron: "0 2 * * *"
    chunk_size: 1000

  # Batch Processor
  batch:
    chunk_size: 500
    worker_pool_size: 5
    vector_batch_size: 100

  # Storage connections
  database:
    dsn: "postgres://kgs:password@postgres:5432/kgs_graph"

  neo4j:
    uri: bolt://neo4j:7687
    username: neo4j
    password: secret

  qdrant:
    addr: qdrant:6334
    api_key: ""

  redis:
    addr: redis:6379
    lock_ttl: 30s

  nats:
    addr: nats:4222
    subjects:
      subscribe:
        - sync.batch.requested

  embedding:
    provider: air-vnp
    endpoint: http://air-vnp:8080
    dimension: 1536

  observability:
    metrics_port: 9097
```

---

## 8. Observability

| Metric | Mô tả |
|--------|-------|
| `sync_outbox_pending_total{app_id}` | Số outbox records đang chờ |
| `sync_outbox_processed_total{app_id, operation, status}` | Số records đã xử lý |
| `sync_outbox_duration_seconds{operation}` | Thời gian xử lý mỗi record |
| `sync_outbox_dead_letter_total{app_id}` | Số records vào Dead Letter |
| `sync_neo4j_upserts_total{app_id}` | Số Neo4j upserts thành công |
| `sync_qdrant_upserts_total{app_id}` | Số Qdrant upserts thành công |
| `sync_reconcile_discrepancy_total{app_id, store}` | Số bất đồng bộ phát hiện |
| `sync_embedding_duration_seconds{provider}` | Latency tạo embedding |
| `sync_batch_progress{job_id}` | Tiến độ batch job |
