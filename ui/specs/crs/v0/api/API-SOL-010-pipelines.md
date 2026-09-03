# API-SOL-010 — Pipelines API Client + Hooks

| Field | Value |
|---|---|
| **Solution ID** | API-SOL-010 |
| **Status** | ✅ IMPLEMENTED — 2026-06-17 |
| **CR** | [CR-009 — Pipelines](../../../../specs/crs/v0/ui/CR-009-PIPELINES.md) |
| **Target files** | `ui/src/api/clients/pipelines.client.ts`, `ui/src/api/hooks/usePipelines.ts` |
| **Implemented files** | `ui/src/services/pipeline.service.ts` · `ui/src/hooks/usePipelines.ts` |

---

## Types

### `ui/src/types/pipeline.ts`

```typescript
export interface QueueMetrics {
  depth:       number;
  throughput:  number;
  retry_count: number;
}

export interface PipelineJob {
  id:          string;
  engine:      string;
  type:        'ingest' | 'index' | 'sync' | 'cognify';
  status:      'Running' | 'Idle' | 'Failed' | 'Completed';
  progress:    number;       // 0-100 = items_done / items_total * 100
  items_total: number;
  items_done:  number;
  created_at:  string;
  updated_at:  string;
}

export interface PipelineWorker {
  id:     string;
  engine: string;
  status: 'idle' | 'busy' | 'offline';
}

export interface PipelineTemplate {
  id:     string;
  name:   string;
  engine: string;
  config: Record<string, unknown>;
}

export interface PipelineStatus {
  engine:    string;
  status:    'idle' | 'running' | 'paused' | 'error';
  job_count: number;
}
```

---

## Implementation

### `ui/src/api/clients/pipelines.client.ts`

```typescript
import { httpClient } from './http.client';
import type { QueueMetrics, PipelineJob, PipelineWorker, PipelineTemplate, PipelineStatus } from '../../types/pipeline';

const BASE = '/v1/console/pipelines';

export const pipelinesClient = {
  getQueues: async (): Promise<QueueMetrics> => {
    const { data } = await httpClient.get<QueueMetrics>(`${BASE}/queues`);
    return data;
  },

  getStatus: async (): Promise<PipelineStatus[]> => {
    const { data } = await httpClient.get<PipelineStatus[]>(`${BASE}/status`);
    return data;
  },

  getWorkers: async (): Promise<PipelineWorker[]> => {
    const { data } = await httpClient.get<PipelineWorker[]>(`${BASE}/workers`);
    return data;
  },

  getTemplates: async (): Promise<PipelineTemplate[]> => {
    const { data } = await httpClient.get<PipelineTemplate[]>(`${BASE}/templates`);
    return data;
  },

  getJobs: async (engine: string): Promise<PipelineJob[]> => {
    const { data } = await httpClient.get<PipelineJob[]>(`${BASE}/${engine}/jobs`);
    return data;
  },

  getJobDetail: async (engine: string, jobId: string): Promise<PipelineJob> => {
    const { data } = await httpClient.get<PipelineJob>(`${BASE}/${engine}/jobs/${jobId}`);
    return data;
  },
};
```

### `ui/src/api/hooks/usePipelines.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import { pipelinesClient } from '../clients/pipelines.client';

const keys = {
  queues:    () => ['pipelines', 'queues'] as const,
  status:    () => ['pipelines', 'status'] as const,
  workers:   () => ['pipelines', 'workers'] as const,
  templates: () => ['pipelines', 'templates'] as const,
  jobs:      (e: string) => ['pipelines', e, 'jobs'] as const,
  job:       (e: string, id: string) => ['pipelines', e, 'jobs', id] as const,
};

/** NATS queue depth — poll 10s */
export const useQueueMetrics = () => useQuery({
  queryKey:                keys.queues(),
  queryFn:                 () => pipelinesClient.getQueues(),
  refetchInterval:         10_000,
  refetchIntervalInBackground: false,
});

export const usePipelineStatus = () => useQuery({
  queryKey:        keys.status(),
  queryFn:         () => pipelinesClient.getStatus(),
  refetchInterval: 10_000,
});

export const useWorkers = () => useQuery({
  queryKey:        keys.workers(),
  queryFn:         () => pipelinesClient.getWorkers(),
  refetchInterval: 15_000,
});

export const useTemplates = () => useQuery({
  queryKey: keys.templates(),
  queryFn:  () => pipelinesClient.getTemplates(),
});

export const useEngineJobs = (engine: string) => useQuery({
  queryKey:        keys.jobs(engine),
  queryFn:         () => pipelinesClient.getJobs(engine),
  enabled:         !!engine,
  refetchInterval: 10_000,
});

/** Job detail — poll 5s khi đang xem job đang chạy */
export const useJobDetail = (engine: string, jobId: string) => useQuery({
  queryKey:        keys.job(engine, jobId),
  queryFn:         () => pipelinesClient.getJobDetail(engine, jobId),
  enabled:         !!engine && !!jobId,
  refetchInterval: 5_000,
});
```
