# TASK-API-004 — Dashboard API Client + Hooks

**Task ID:** TASK-API-004
**Status:** ✅ COMPLETED — 2026-06-17
**Sprint:** 2 — P0 Modules
**Solution:** [API-SOL-003](../API-SOL-003-dashboard.md)
**Depends on:** TASK-API-001, TASK-API-002
**Ước tính:** 1.5h
**Priority:** P0 — Critical

---

## Mục tiêu

Thay thế `dashboardMock` bằng API calls thực:
1. `dashboard.client.ts` — 4 endpoints: metrics, health, throughput, heatmap
2. `useDashboard.ts` — hooks với polling intervals đúng theo priority

---

## Công việc cụ thể

### 1. Tạo `ui/src/api/clients/dashboard.client.ts`

```typescript
import { httpClient } from './http.client';
import type {
  KPIData, EngineHealth, ThroughputData, ThroughputWindow, HeatmapData
} from '../../types/dashboard';

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

### 2. Tạo `ui/src/api/hooks/useDashboard.ts`

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

/** Poll 60s — KPI metrics tổng hợp */
export function useMetrics() {
  return useQuery({
    queryKey:                dashboardKeys.metrics(),
    queryFn:                 () => dashboardClient.getMetrics(),
    staleTime:               30_000,
    refetchInterval:         60_000,
    refetchIntervalInBackground: false,
  });
}

/** Poll 30s — Health check 7 engines */
export function useEngineHealth() {
  return useQuery({
    queryKey:                dashboardKeys.health(),
    queryFn:                 () => dashboardClient.getHealth(),
    staleTime:               15_000,
    refetchInterval:         30_000,
    refetchIntervalInBackground: false,
  });
}

/** Poll 30s — Prometheus throughput metrics */
export function useThroughput(window: ThroughputWindow = '1h') {
  return useQuery({
    queryKey:        dashboardKeys.throughput(window),
    queryFn:         () => dashboardClient.getThroughput(window),
    staleTime:       15_000,
    refetchInterval: 30_000,
  });
}

/** staleTime 5m — Heatmap ít thay đổi */
export function useDashboardHeatmap() {
  return useQuery({
    queryKey:  dashboardKeys.heatmap(),
    queryFn:   () => dashboardClient.getHeatmap(),
    staleTime: 5 * 60_000,
  });
}
```

### 3. Cập nhật `ui/src/hooks/useDashboard.ts` (file cũ)

Nếu file cũ tồn tại, thay thế nội dung bằng re-export từ hooks mới:

```typescript
// ui/src/hooks/useDashboard.ts — compatibility re-export
export { useMetrics, useEngineHealth, useThroughput, useDashboardHeatmap } from '../api/hooks/useDashboard';
```

Hoặc tìm tất cả imports và cập nhật đường dẫn:

```bash
grep -r "from.*useDashboard\|from.*dashboard.service\|dashboardMock" ui/src --include="*.ts" --include="*.tsx"
```

### 4. Xóa mock dependency

Tìm và xóa hoặc vô hiệu hóa:
- `ui/src/mock/dashboard.mock.ts`
- Tất cả `const useMock = API_CONFIG.useMockData` trong dashboard hooks

---

## Files tạo ra / chỉnh sửa

```
ui/src/
├── api/
│   ├── clients/
│   │   └── dashboard.client.ts  ← NEW
│   └── hooks/
│       └── useDashboard.ts      ← NEW
└── hooks/
    └── useDashboard.ts          ← MODIFY (re-export hoặc xóa mock)
```

---

## Acceptance Criteria

- [x] `GET /v1/console/dashboard/metrics` → `KPIData` object với 11 fields
- [x] `GET /v1/console/dashboard/health` → array `EngineHealth[]` với 7 engines
- [x] `GET /v1/console/dashboard/throughput?window=1h` → `ThroughputData`
- [x] `GET /v1/console/dashboard/heatmap` → `HeatmapData` với points array
- [x] `useMetrics()` auto-refetch mỗi 60s
- [x] `useEngineHealth()` auto-refetch mỗi 30s
- [x] Không còn import `dashboardMock` ở bất kỳ đâu trong `ui/src/`
- [x] Dashboard hiển thị loading skeleton khi đang fetch
- [x] Dashboard hiển thị error state khi API fail

```bash
grep -r "dashboardMock" ui/src/  # phải trả về 0 kết quả
```
