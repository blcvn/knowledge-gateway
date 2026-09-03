# TASK-UI-012 — Tạo `observability.service.ts` + Refactor `useObservability.ts`

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-012 |
| **Layer** | Frontend — TypeScript |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-007 §8](../solutions/SOL-007-Gap-Fixes.md) |
| **Priority** | 🟠 P1 |
| **Depends On** | TASK-UI-001 |
| **Estimated** | 1.5h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `ui/src/services/observability.service.ts` |
| MODIFY | `ui/src/hooks/useObservability.ts` |

---

## Implementation

### File: `ui/src/services/observability.service.ts`

```typescript
import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';

const BASE = API_CONFIG.observability;

export interface MetricPoint {
  timestamp: string;
  value: number;
  label: string; // "p50" | "p95" | "error_rate" | "throughput"
}

export interface MetricsResponse {
  latency: MetricPoint[];
  error_rate: MetricPoint[];
  throughput: MetricPoint[];
}

export interface TraceSpan {
  id: string;
  trace_id: string;
  span_id: string;
  operation: string;
  service: string;
  duration: number;    // ms
  status: 'ok' | 'slow' | 'error';
  timestamp: string;
}

export interface ErrorEntry {
  id: string;
  message: string;
  service: string;
  count: number;
  lastOccurrence: string;
  stack?: string;
}

export interface CostEntry {
  model: string;
  engine: string;
  tokens_input: number;
  tokens_output: number;
  cost_usd: number;
  date: string;
}

export const observabilityService = {
  getMetrics: () =>
    apiClient.get<MetricsResponse>(`${BASE}/metrics`),

  getTraces: (filters: Record<string, string> = {}) => {
    const qs = new URLSearchParams(filters).toString();
    return apiClient.get<TraceSpan[]>(`${BASE}/traces?${qs}`);
  },

  getTraceDetail: (id: string) =>
    apiClient.get<TraceSpan>(`${BASE}/traces/${id}`),

  getErrors: (filters: Record<string, string> = {}) => {
    const qs = new URLSearchParams(filters).toString();
    return apiClient.get<ErrorEntry[]>(`${BASE}/errors?${qs}`);
  },

  getCosts: () =>
    apiClient.get<CostEntry[]>(`${BASE}/costs`),
};
```

### File: `ui/src/hooks/useObservability.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import { observabilityService } from '../services/observability.service';

export function useObsMetrics() {
  return useQuery({
    queryKey: ['observability', 'metrics'],
    queryFn: () => observabilityService.getMetrics(),
    refetchInterval: 60_000,
  });
}

export function useTraces(filters: Record<string, string> = {}) {
  return useQuery({
    queryKey: ['observability', 'traces', filters],
    queryFn: () => observabilityService.getTraces(filters),
  });
}

export function useTraceDetail(id: string) {
  return useQuery({
    queryKey: ['observability', 'traces', id],
    queryFn: () => observabilityService.getTraceDetail(id),
    enabled: !!id,
  });
}

export function useErrors(filters: Record<string, string> = {}) {
  return useQuery({
    queryKey: ['observability', 'errors', filters],
    queryFn: () => observabilityService.getErrors(filters),
  });
}

export function useCosts() {
  return useQuery({
    queryKey: ['observability', 'costs'],
    queryFn: () => observabilityService.getCosts(),
  });
}
```

---

## Verification

```bash
cd ui
npx tsc --noEmit
grep -r "observabilityMock" ui/src/hooks/ # phải trống
```
