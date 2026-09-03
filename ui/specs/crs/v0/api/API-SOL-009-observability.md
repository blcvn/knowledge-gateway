# API-SOL-009 — Observability API Client + Hooks

| Field | Value |
|---|---|
| **Solution ID** | API-SOL-009 |
| **Status** | ✅ IMPLEMENTED — 2026-06-17 |
| **CR** | [CR-008 — Observability](../../../../specs/crs/v0/ui/CR-008-OBSERVABILITY.md) |
| **Target files** | `ui/src/api/clients/observability.client.ts`, `ui/src/api/hooks/useObservability.ts` |
| **Implemented files** | `ui/src/services/observability.service.ts` · `ui/src/hooks/useObservability.ts` |

---

## Types

### `ui/src/types/observability.ts`

```typescript
export interface MetricPoint {
  timestamp: string;    // ISO 8601
  value:     number;
  label:     string;    // "p95" | "error_rate" | "throughput"
}

export interface MetricsResponse {
  latency:     MetricPoint[];
  error_rate:  MetricPoint[];
  throughput:  MetricPoint[];
}

export interface TraceSpan {
  id:         string;
  trace_id:   string;
  span_id:    string;
  operation:  string;
  service:    string;
  duration:   number;   // ms
  status:     'ok' | 'slow' | 'error';
  timestamp:  string;
}

export interface ErrorEntry {
  id:              string;
  message:         string;
  service:         string;
  count:           number;
  lastOccurrence:  string;
  stack?:          string;
}

export interface CostEntry {
  model:         string;
  engine:        string;
  tokens_input:  number;
  tokens_output: number;
  cost_usd:      number;
  date:          string;
}

export interface TraceFilters {
  service?:   string;
  status?:    'ok' | 'slow' | 'error';
  operation?: string;
  from?:      string;
  to?:        string;
}
```

---

## Implementation

### `ui/src/api/clients/observability.client.ts`

```typescript
import { httpClient } from './http.client';
import type { MetricsResponse, TraceSpan, ErrorEntry, CostEntry, TraceFilters } from '../../types/observability';

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

### `ui/src/api/hooks/useObservability.ts`

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

/** Prometheus metrics — poll 60s */
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
