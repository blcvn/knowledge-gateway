# TASK-UI-005 — Refactor `hooks/useDashboard.ts`: Xóa mock, thêm refetchInterval

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-005 |
| **Layer** | Frontend — TypeScript |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-003 §3.1](../solutions/SOL-003-Dashboard-Solution.md) |
| **Priority** | 🔴 P0 — Critical |
| **Depends On** | TASK-UI-001 |
| **Estimated** | 1h |

---

## Context

`hooks/useDashboard.ts` hiện dùng `useMock` ternary để return `dashboardMock.*`. Cần xóa toàn bộ mock logic và chỉ gọi `dashboardService` với các polling intervals phù hợp.

---

## Goal

- Xóa `import { dashboardMock }` và `const useMock = ...`
- Thêm `refetchInterval` cho từng hook
- Thêm `useDashboardHeatmap` hook còn thiếu
- Cập nhật `getThroughput` để nhận `window` param

---

## Target Files

| Action | File Path |
|---|---|
| MODIFY | `ui/src/hooks/useDashboard.ts` |
| MODIFY | `ui/src/services/dashboard.service.ts` |

---

## Implementation

### File: `ui/src/hooks/useDashboard.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import { dashboardService } from '../services/dashboard.service';

export function useMetrics() {
  return useQuery({
    queryKey: ['dashboard', 'metrics'],
    queryFn: () => dashboardService.getMetrics(),
    staleTime: 30_000,
    refetchInterval: 60_000,      // Refresh mỗi phút
    refetchIntervalInBackground: false,
  });
}

export function useEngineHealth() {
  return useQuery({
    queryKey: ['dashboard', 'health'],
    queryFn: () => dashboardService.getHealth(),
    staleTime: 15_000,
    refetchInterval: 30_000,      // Refresh mỗi 30s
    refetchIntervalInBackground: false,
  });
}

export function useThroughput(window = '1h') {
  return useQuery({
    queryKey: ['dashboard', 'throughput', window],
    queryFn: () => dashboardService.getThroughput(window),
    staleTime: 15_000,
    refetchInterval: 30_000,
  });
}

export function useDashboardHeatmap() {
  return useQuery({
    queryKey: ['dashboard', 'heatmap'],
    queryFn: () => dashboardService.getHeatmap(),
    staleTime: 5 * 60_000,        // Heatmap ít thay đổi — 5 phút
  });
}
```

### File: `ui/src/services/dashboard.service.ts`

Cập nhật `getThroughput` để nhận `window` param:

```typescript
import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type { KPIData, EngineHealth, ThroughputData } from '../types/dashboard';

const BASE = API_CONFIG.dashboard;

export const dashboardService = {
  getMetrics: () =>
    apiClient.get<KPIData>(`${BASE}/metrics`),

  getHealth: () =>
    apiClient.get<EngineHealth[]>(`${BASE}/health`),

  getThroughput: (window = '1h') =>
    apiClient.get<ThroughputData>(`${BASE}/throughput?window=${window}`),

  getHeatmap: () =>
    apiClient.get<{ points: Array<{ x: number; y: number; density: number }>; xLabel: string; yLabel: string; maxDensity: number }>(`${BASE}/heatmap`),
};
```

---

## Verification

```bash
cd ui
npx tsc --noEmit
grep -r "dashboardMock" ui/src/hooks/ ui/src/components/ # phải trống
```
