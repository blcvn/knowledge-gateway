# Feature 23 — Pipeline Monitor

> **Loại:** Operations | **Priority:** Medium | **Status:** Implemented (UI)

## Mô tả

Pipeline Monitor theo dõi trạng thái background processing jobs trên tất cả engines. AI Engineer có thể xem jobs đang chạy, queue depth, worker status, và diagnose stuck/failed jobs.

---

## Business Logic

### Per-Engine Pipeline Status

Mỗi memory engine có background pipeline:
- **Cognee**: Cognify jobs (chunking → embedding → KG construction)
- **Graphiti**: Episode ingestion jobs
- **Memobase**: Flush jobs (YOLO engine)
- **Supermemory**: Document ingestion, connector sync
- **Consolidation**: Memory compression, summarization
- **AgentMemory**: Observation processing

### Job Management

Mỗi job có:
- Status: `pending` | `running` | `completed` | `failed`
- Engine: job thuộc engine nào
- Created/started/completed timestamps
- Error message (nếu failed)
- Retry count

### Queue Monitoring

- **Queue depth**: Bao nhiêu jobs đang pending
- **Throughput**: Jobs/minute
- **Backpressure alert**: Queue depth quá cao → cần thêm workers

### Worker Status

- List workers đang active
- Current job per worker
- Worker uptime, processed count, error rate

### Templates

Reusable job templates:
- "Full re-index dataset" (Cognee)
- "Batch episode ingestion" (Graphiti)
- "Force profile rebuild" (Memobase)

---

## Dataflow

```
Console UI (Pipeline Monitor)
        │
        ├── GET /v1/console/pipelines/status
        │         └── Overview: all engines, job counts, queue depths
        │
        ├── GET /v1/console/pipelines/{engine}
        │         └── Per-engine detail: running/failed/completed counts
        │
        ├── GET /v1/console/pipelines/{engine}/jobs
        │         └── Job list (filterable by status)
        │
        ├── GET /v1/console/pipelines/{engine}/jobs/{id}
        │         └── Job detail + logs
        │
        ├── GET /v1/console/pipelines/queues
        │         └── Queue depth per engine
        │
        ├── GET /v1/console/pipelines/workers
        │         └── Active workers status
        │
        └── GET /v1/console/pipelines/templates
                  └── Reusable job templates
```

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/v1/console/pipelines/status` | Overall pipeline status |
| `GET` | `/v1/console/pipelines/queues` | Queue depths |
| `GET` | `/v1/console/pipelines/workers` | Worker status |
| `GET` | `/v1/console/pipelines/templates` | Job templates |
| `GET` | `/v1/console/pipelines/{engine}` | Per-engine status |
| `GET` | `/v1/console/pipelines/{engine}/jobs` | Job list |
| `GET` | `/v1/console/pipelines/{engine}/jobs/{id}` | Job detail |

---

## Business Value

### Pain Points được giải quyết

- **PP-P2-05 (Pipeline failures im lặng)**

### Actors hưởng lợi

P2 Platform Engineer, P3 ML/AI Engineer

### Giải pháp tham chiếu

- [S10 — Zero-config Infrastructure](../../bussiness/solutions/S10-infrastructure-simplicity.md)

### ROI / Kết quả đo được

> Job status real-time | Failed job alerts | Worker throughput metrics

---

*Xem thêm: [Pain Points](../../bussiness/painpoints/README.md) | [Solutions](../../bussiness/solutions/README.md)*
