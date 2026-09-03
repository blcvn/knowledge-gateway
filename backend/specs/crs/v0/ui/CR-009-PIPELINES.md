# CR-009 — Pipelines Monitor: Mock → Real API

| Field | Value |
|---|---|
| **CR ID** | CR-009 |
| **Title** | Pipelines Monitor: Theo dõi queues và processing jobs từ backend NATS JetStream |
| **Type** | Feature Implementation |
| **Priority** | P1 — High |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Module** | Pipelines |
| **Files thay đổi** | `ui/src/mock/pipeline.mock.ts`, `ui/src/hooks/usePipelines.ts`, `ui/src/services/pipeline.service.ts` |

---

## 1. Hiện trạng

Mock data ([`pipeline.mock.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/mock/pipeline.mock.ts)):
Hardcode dữ liệu của queue (depth: 10, throughput: 5) và một job fake đang "Running" của engine cognee.

---

## 2. Backend API cần implement

Base path: `/v1/console/pipelines`
Data source: NATS JetStream management APIs & Internal workers status.

### 2.1 Queue Metrics

- `GET /v1/console/pipelines/queues`

Lấy trạng thái các NATS streams/consumers.

**Response schema** (`QueueMetrics`):
```json
{
  "depth": 145,
  "throughput": 12.5,
  "retry_count": 3
}
```

### 2.2 Job Status & Workers

- `GET /v1/console/pipelines/status`
- `GET /v1/console/pipelines/workers`
- `GET /v1/console/pipelines/templates`

**Response schema** (`PipelineJob`):
```json
[
  {
    "id": "job_123",
    "engine": "cognee",
    "status": "Running",
    "progress": 55,
    "created_at": "2026-06-16T12:00:00Z",
    "updated_at": "2026-06-16T12:05:00Z"
  }
]
```

### 2.3 Engine-specific Pipelines

- `GET /v1/console/pipelines/{engine}/jobs`
- `GET /v1/console/pipelines/{engine}/jobs/{id}`

Ví dụ lấy các jobs đang xử lý của `cognee` (Cognify pipeline) hoặc `memobase` (Buffer flush YOLO pipeline).

---

## 3. Frontend thay đổi

### 3.1 Xóa mock dependency trong `usePipelines.ts`

```typescript
// SAU
import { useQuery } from '@tanstack/react-query';
import { pipelineService } from '../services/pipeline.service';

export function useQueueMetrics() {
  return useQuery({
    queryKey: ['pipelines', 'queues'],
    queryFn: () => pipelineService.getQueues(),
    refetchInterval: 10 * 1000, // Poll 10s
  });
}

export function useEngineJobs(engine: string) {
  return useQuery({
    queryKey: ['pipelines', engine, 'jobs'],
    queryFn: () => pipelineService.getJobs(engine),
    enabled: !!engine,
    refetchInterval: 10 * 1000,
  });
}

// ... UseWorkers, UseStatus
```

---

## 4. Điều kiện hoàn thành

- [ ] Queue metrics hiển thị số liệu thật từ NATS (pending messages = depth).
- [ ] Danh sách active jobs của các engine (như Cognee Cognify) render chuẩn.
- [ ] Bỏ toàn bộ mock pipeline.
