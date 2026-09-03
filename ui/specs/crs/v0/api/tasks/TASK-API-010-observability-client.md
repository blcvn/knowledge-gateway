# TASK-API-010 — Observability + Pipelines API Client + Hooks

**Task ID:** TASK-API-010
**Status:** ✅ COMPLETED — 2026-06-17
**Sprint:** 3 — P1 Modules
**Solution:** [API-SOL-009](../API-SOL-009-observability.md)
**Depends on:** TASK-API-001, TASK-API-002
**Ước tính:** 1h
**Priority:** P1

---

## Công việc cụ thể

### 1. Tạo `ui/src/api/clients/observability.client.ts`

```typescript
import { httpClient } from './http.client';
import type {
  MetricsResponse, TraceSpan, ErrorEntry, CostEntry, TraceFilters
} from '../../types/observability';

const BASE = '/v1/console/observability';

export const observabilityClient = {
  getMetrics: async (): Promise<MetricsResponse> => {
    const { data } = await httpClient.get<MetricsResponse>(`${BASE}/metrics`);
    return data;
  },

  getTraces: async (filters: TraceFilters = {}): Promise<TraceSpan[]> => {
    const { data } = await httpClient.get<TraceSpan[]>(`${BASE}/traces`, {
      params: filters,
    });
    return data;
  },

  getTraceDetail: async (id: string): Promise<TraceSpan> => {
    const { data } = await httpClient.get<TraceSpan>(`${BASE}/traces/${id}`);
    return data;
  },

  getErrors: async (filters: { service?: string } = {}): Promise<ErrorEntry[]> => {
    const { data } = await httpClient.get<ErrorEntry[]>(`${BASE}/errors`, {
      params: filters,
    });
    return data;
  },

  getCosts: async (): Promise<CostEntry[]> => {
    const { data } = await httpClient.get<CostEntry[]>(`${BASE}/costs`);
    return data;
  },
};
```

### 2. Tạo `ui/src/api/hooks/useObservability.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import { observabilityClient } from '../clients/observability.client';
import type { TraceFilters } from '../../types/observability';

const keys = {
  metrics:     () => ['observability', 'metrics'] as const,
  traces:      (f: TraceFilters) => ['observability', 'traces', f] as const,
  traceDetail: (id: string) => ['observability', 'traces', id] as const,
  errors:      (f: object) => ['observability', 'errors', f] as const,
  costs:       () => ['observability', 'costs'] as const,
};

/** Poll 60s — Prometheus time-series */
export const useObsMetrics = () => useQuery({
  queryKey:        keys.metrics(),
  queryFn:         () => observabilityClient.getMetrics(),
  refetchInterval: 60_000,
});

export const useTraces = (filters: TraceFilters = {}) => useQuery({
  queryKey: keys.traces(filters),
  queryFn:  () => observabilityClient.getTraces(filters),
});

export const useTraceDetail = (id: string) => useQuery({
  queryKey: keys.traceDetail(id),
  queryFn:  () => observabilityClient.getTraceDetail(id),
  enabled:  !!id,
});

export const useErrors = (filters: { service?: string } = {}) => useQuery({
  queryKey: keys.errors(filters),
  queryFn:  () => observabilityClient.getErrors(filters),
});

export const useCosts = () => useQuery({
  queryKey: keys.costs(),
  queryFn:  () => observabilityClient.getCosts(),
});
```

---

## Files tạo ra

```
ui/src/api/
├── clients/observability.client.ts  ← NEW
└── hooks/useObservability.ts        ← NEW
```

---

## Acceptance Criteria

- [x] `GET /v1/console/observability/metrics` → `{ latency, error_rate, throughput }` time-series arrays
- [x] `GET /v1/console/observability/traces?service=gateway` → filtered `TraceSpan[]`
- [x] `GET /v1/console/observability/traces/{id}` → single trace detail
- [x] `GET /v1/console/observability/errors?service=cognee` → `ErrorEntry[]` với count
- [x] `GET /v1/console/observability/costs` → `CostEntry[]` per model/engine
- [x] `useObsMetrics()` poll 60s
- [x] `npx tsc --noEmit` không lỗi

# TASK-API-011 — Pipelines API Client + Hooks

**Task ID:** TASK-API-011
**Sprint:** 3 — P1 Modules
**Solution:** [API-SOL-010](../API-SOL-010-pipelines.md)
**Depends on:** TASK-API-001, TASK-API-002
**Ước tính:** 1h
**Priority:** P1

---

## Công việc cụ thể

### 1. Tạo `ui/src/api/clients/pipelines.client.ts`

```typescript
import { httpClient } from './http.client';
import type {
  QueueMetrics, PipelineJob, PipelineWorker, PipelineTemplate, PipelineStatus
} from '../../types/pipeline';

const BASE = '/v1/console/pipelines';

export const pipelinesClient = {
  getQueues:   async (): Promise<QueueMetrics> => {
    const { data } = await httpClient.get<QueueMetrics>(`${BASE}/queues`);
    return data;
  },
  getStatus:   async (): Promise<PipelineStatus[]> => {
    const { data } = await httpClient.get<PipelineStatus[]>(`${BASE}/status`);
    return data;
  },
  getWorkers:  async (): Promise<PipelineWorker[]> => {
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

### 2. Tạo `ui/src/api/hooks/usePipelines.ts`

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

/** Poll 10s — NATS queue depth thay đổi nhanh */
export const useQueueMetrics  = () => useQuery({
  queryKey: keys.queues(), queryFn: pipelinesClient.getQueues,
  refetchInterval: 10_000, refetchIntervalInBackground: false,
});
export const usePipelineStatus = () => useQuery({
  queryKey: keys.status(), queryFn: pipelinesClient.getStatus,
  refetchInterval: 10_000,
});
export const useWorkers       = () => useQuery({
  queryKey: keys.workers(), queryFn: pipelinesClient.getWorkers,
  refetchInterval: 15_000,
});
export const useTemplates     = () => useQuery({
  queryKey: keys.templates(), queryFn: pipelinesClient.getTemplates,
});

export const useEngineJobs = (engine: string) => useQuery({
  queryKey:        keys.jobs(engine),
  queryFn:         () => pipelinesClient.getJobs(engine),
  enabled:         !!engine,
  refetchInterval: 10_000,
});

/** Poll 5s khi đang xem running job */
export const useJobDetail = (engine: string, jobId: string) => useQuery({
  queryKey:        keys.job(engine, jobId),
  queryFn:         () => pipelinesClient.getJobDetail(engine, jobId),
  enabled:         !!engine && !!jobId,
  refetchInterval: 5_000,
});
```

---

## Files tạo ra

```
ui/src/api/
├── clients/pipelines.client.ts  ← NEW
└── hooks/usePipelines.ts        ← NEW
```

---

## Acceptance Criteria

- [x] `GET /v1/console/pipelines/queues` → `{ depth, throughput, retry_count }`
- [x] `GET /v1/console/pipelines/status` → `PipelineStatus[]` cho tất cả engines
- [x] `GET /v1/console/pipelines/cognee/jobs` → `PipelineJob[]` với progress 0-100
- [x] `useQueueMetrics()` poll 10s
- [x] `useJobDetail()` poll 5s
- [x] `npx tsc --noEmit` không lỗi
