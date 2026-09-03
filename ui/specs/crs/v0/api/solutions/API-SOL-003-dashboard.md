# API-SOL-003 — Dashboard API Client + Hooks

| Field | Value |
|---|---|
| **Solution ID** | API-SOL-003 |
| **Status** | ✅ IMPLEMENTED — 2026-06-17 |
| **CR** | [CR-002 — Dashboard](../../../../specs/crs/v0/ui/CR-002-DASHBOARD.md) |
| **Kiến trúc ref** | `frontend_architecture.md §3.3, §3.2` TanStack Query auto-refetch |
| **Target files** | `ui/src/api/clients/dashboard.client.ts`, `ui/src/api/hooks/useDashboard.ts` |
| **Implemented files** | `ui/src/services/dashboard.service.ts` · `ui/src/hooks/useDashboard.ts` |

---

## API Endpoints

| Method | Endpoint | Cache/Poll |
|---|---|---|
| `GET` | `/v1/console/dashboard/metrics` | staleTime 30s, poll 60s |
| `GET` | `/v1/console/dashboard/health` | staleTime 15s, poll 30s |
| `GET` | `/v1/console/dashboard/throughput?window={w}` | staleTime 15s, poll 30s |
| `GET` | `/v1/console/dashboard/heatmap` | staleTime 5m |

---

## Types

### `ui/src/types/dashboard.ts`

```typescript
export interface KPIData {
  activeAgents:        number;
  recallLatencyP50Ms:  number;
  recallLatencyP95Ms:  number;
  contextSavingsPct:   number;
  graphNodesTotal:     number;
  graphEdgesTotal:     number;
  graphGrowth24h:      number;
  errorRatePct:        number;
  activeSessions:      number;
  activeProfiles:      number;
  memoryVersions:      number;
}

export interface EngineHealth {
  name:          string;
  role:          string;
  status:        'Healthy' | 'Warning' | 'Critical';
  latencyP50Ms:  number;
  latencyP95Ms:  number;
  queueDepth:    number;
  uptimeSeconds: number;
  lastCheck:     string; // ISO 8601
}

export type ThroughputWindow = '5m' | '15m' | '1h' | '6h' | '24h';

export interface EngineMetrics {
  ingestPerSec:             number;
  recallPerSec:             number;
  embedPerSec:              number;
  profileExtractionsPerSec?: number;
}

export interface ThroughputData {
  window:  ThroughputWindow;
  engines: Record<string, EngineMetrics>;
}

export interface HeatmapPoint {
  x:       number; // 0-23 (hour)
  y:       number; // 0-6  (day of week)
  density: number;
}

export interface HeatmapData {
  points:     HeatmapPoint[];
  xLabel:     string;
  yLabel:     string;
  maxDensity: number;
}
```

---

## Implementation

### `ui/src/api/clients/dashboard.client.ts`

```typescript
import { httpClient } from './http.client';
import type { KPIData, EngineHealth, ThroughputData, ThroughputWindow, HeatmapData } from '../../types/dashboard';

const BASE = '/v1/console/dashboard';

export const dashboardClient = {
  getMetrics: async (): Promise<KPIData> => {
    const { data } = await httpClient.get<KPIData>(`${BASE}/metrics`);
    return data;
  },

  getHealth: async (): Promise<EngineHealth[]> => {
    const { data } = await httpClient.get<EngineHealth[]>(`${BASE}/health`);
    return data;
  },

  getThroughput: async (window: ThroughputWindow = '1h'): Promise<ThroughputData> => {
    const { data } = await httpClient.get<ThroughputData>(`${BASE}/throughput`, {
      params: { window },
    });
    return data;
  },

  getHeatmap: async (): Promise<HeatmapData> => {
    const { data } = await httpClient.get<HeatmapData>(`${BASE}/heatmap`);
    return data;
  },
};
```

### `ui/src/api/hooks/useDashboard.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import { dashboardClient } from '../clients/dashboard.client';
import type { ThroughputWindow } from '../../types/dashboard';

export const dashboardKeys = {
  all:        () => ['dashboard'] as const,
  metrics:    () => [...dashboardKeys.all(), 'metrics'] as const,
  health:     () => [...dashboardKeys.all(), 'health'] as const,
  throughput: (w: ThroughputWindow) => [...dashboardKeys.all(), 'throughput', w] as const,
  heatmap:    () => [...dashboardKeys.all(), 'heatmap'] as const,
};

/**
 * Dashboard KPI metrics — tổng hợp từ PostgreSQL + Prometheus + Neo4j
 * Poll: mỗi 60s
 */
export function useMetrics() {
  return useQuery({
    queryKey:                dashboardKeys.metrics(),
    queryFn:                 () => dashboardClient.getMetrics(),
    staleTime:               30_000,
    refetchInterval:         60_000,
    refetchIntervalInBackground: false,
  });
}

/**
 * Engine health status — gọi health check đến 7 engines
 * Poll: mỗi 30s (critical data)
 */
export function useEngineHealth() {
  return useQuery({
    queryKey:                dashboardKeys.health(),
    queryFn:                 () => dashboardClient.getHealth(),
    staleTime:               15_000,
    refetchInterval:         30_000,
    refetchIntervalInBackground: false,
  });
}

/**
 * Throughput metrics theo time window
 * @param window - '5m' | '15m' | '1h' | '6h' | '24h'
 */
export function useThroughput(window: ThroughputWindow = '1h') {
  return useQuery({
    queryKey:        dashboardKeys.throughput(window),
    queryFn:         () => dashboardClient.getThroughput(window),
    staleTime:       15_000,
    refetchInterval: 30_000,
  });
}

/**
 * Activity heatmap — 24h × 7 days
 * Ít thay đổi → staleTime 5 phút
 */
export function useDashboardHeatmap() {
  return useQuery({
    queryKey:  dashboardKeys.heatmap(),
    queryFn:   () => dashboardClient.getHeatmap(),
    staleTime: 5 * 60_000,
  });
}
```

---

## Realtime Alternative (SSE)

Theo kiến trúc §5.2, dashboard có thể dùng SSE thay vì polling về sau:

```typescript
// ui/src/api/clients/dashboard-sse.client.ts (future)
export function subscribeToHealthUpdates(
  onUpdate: (health: EngineHealth[]) => void,
  onError: (err: Error) => void,
): () => void {
  const token    = localStorage.getItem('access_token');
  const tenantId = localStorage.getItem('tenant_id');
  const url      = `/v1/console/ws?token=${token}&tenant=${tenantId}`;

  const es = new EventSource(url);
  es.addEventListener('engine.health', (e) => {
    onUpdate(JSON.parse(e.data));
    // Also update TanStack Query cache:
    queryClient.setQueryData(dashboardKeys.health(), JSON.parse(e.data));
  });
  es.onerror = () => onError(new Error('SSE connection lost'));
  return () => es.close();
}
```

---

## Verification

```bash
cd ui && npx tsc --noEmit

# Kiểm tra types:
grep -r "dashboardMock" ui/src/  # phải trống
```
