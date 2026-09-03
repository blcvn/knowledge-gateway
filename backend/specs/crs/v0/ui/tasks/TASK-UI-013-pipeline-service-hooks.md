# TASK-UI-013 — Tạo `pipeline.service.ts` + Refactor `usePipelines.ts`

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-013 |
| **Layer** | Frontend — TypeScript |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-007 §9](../solutions/SOL-007-Gap-Fixes.md) |
| **Priority** | 🟠 P1 |
| **Depends On** | TASK-UI-001 |
| **Estimated** | 1h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `ui/src/services/pipeline.service.ts` |
| MODIFY | `ui/src/hooks/usePipelines.ts` |

---

## Implementation

### File: `ui/src/services/pipeline.service.ts`

```typescript
import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';

const BASE = API_CONFIG.pipelines;

export interface QueueMetrics {
  depth: number;
  throughput: number;
  retry_count: number;
}

export interface PipelineJob {
  id: string;
  engine: string;
  type: 'ingest' | 'index' | 'sync' | 'cognify';
  status: 'Running' | 'Idle' | 'Failed' | 'Completed';
  progress: number;           // 0-100
  items_total: number;
  items_done: number;
  created_at: string;
  updated_at: string;
}

export interface PipelineWorker {
  id: string;
  engine: string;
  status: 'idle' | 'busy' | 'offline';
}

export interface PipelineTemplate {
  id: string;
  name: string;
  engine: string;
  config: Record<string, unknown>;
}

export interface PipelineStatus {
  engine: string;
  status: 'idle' | 'running' | 'paused' | 'error';
  job_count: number;
}

export const pipelineService = {
  getQueues: () =>
    apiClient.get<QueueMetrics>(`${BASE}/queues`),

  getStatus: () =>
    apiClient.get<PipelineStatus[]>(`${BASE}/status`),

  getWorkers: () =>
    apiClient.get<PipelineWorker[]>(`${BASE}/workers`),

  getTemplates: () =>
    apiClient.get<PipelineTemplate[]>(`${BASE}/templates`),

  getJobs: (engine: string) =>
    apiClient.get<PipelineJob[]>(`${BASE}/${engine}/jobs`),

  getJobDetail: (engine: string, jobId: string) =>
    apiClient.get<PipelineJob>(`${BASE}/${engine}/jobs/${jobId}`),
};
```

### File: `ui/src/hooks/usePipelines.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import { pipelineService } from '../services/pipeline.service';

export function useQueueMetrics() {
  return useQuery({
    queryKey: ['pipelines', 'queues'],
    queryFn: () => pipelineService.getQueues(),
    refetchInterval: 10_000,
    refetchIntervalInBackground: false,
  });
}

export function usePipelineStatus() {
  return useQuery({
    queryKey: ['pipelines', 'status'],
    queryFn: () => pipelineService.getStatus(),
    refetchInterval: 10_000,
  });
}

export function useWorkers() {
  return useQuery({
    queryKey: ['pipelines', 'workers'],
    queryFn: () => pipelineService.getWorkers(),
    refetchInterval: 15_000,
  });
}

export function useTemplates() {
  return useQuery({
    queryKey: ['pipelines', 'templates'],
    queryFn: () => pipelineService.getTemplates(),
  });
}

export function useEngineJobs(engine: string) {
  return useQuery({
    queryKey: ['pipelines', engine, 'jobs'],
    queryFn: () => pipelineService.getJobs(engine),
    enabled: !!engine,
    refetchInterval: 10_000,
  });
}

export function useJobDetail(engine: string, jobId: string) {
  return useQuery({
    queryKey: ['pipelines', engine, 'jobs', jobId],
    queryFn: () => pipelineService.getJobDetail(engine, jobId),
    enabled: !!engine && !!jobId,
    refetchInterval: 5_000,  // Refresh nhanh hơn khi đang xem job đang chạy
  });
}
```

---

## Verification

```bash
cd ui
npx tsc --noEmit
grep -r "pipelineMock\|pipeline\.mock" ui/src/hooks/ # phải trống
```
