# CR-002 — Dashboard: Mock KPIs/Health/Throughput → Real API

| Field | Value |
|---|---|
| **CR ID** | CR-002 |
| **Title** | Dashboard: Kết nối KPIs, Engine Health, Throughput với backend API |
| **Type** | Feature Implementation |
| **Priority** | P0 — Critical |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Module** | Dashboard |
| **Files thay đổi** | `ui/src/mock/dashboard.mock.ts`, `ui/src/hooks/useDashboard.ts`, `ui/src/services/dashboard.service.ts` |

---

## 1. Hiện trạng

### Mock data hiện tại ([`dashboard.mock.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/mock/dashboard.mock.ts))

```typescript
export const dashboardMock = {
  kpis: {
    activeAgents: 42,
    recallLatencyP50Ms: 45,
    recallLatencyP95Ms: 120,
    contextSavingsPct: 35.5,
    graphNodesTotal: 154200,
    graphEdgesTotal: 489000,
    graphGrowth24h: 1200,
    errorRatePct: 0.5,
    activeSessions: 128,
    activeProfiles: 1500,
    memoryVersions: 45000,
  },

  engineHealth: [
    { name: 'memobase', role: 'Profile Engine', status: 'Healthy', latencyP50Ms: 45, ... },
    { name: 'openviking', role: 'Procedural Memory', status: 'Warning', latencyP50Ms: 150, ... },
    // 7 engines hardcoded
  ],

  throughput: {
    window: '1h',
    engines: {
      memobase: { ingestPerSec: 10, recallPerSec: 50, ... },
      // 7 engines hardcoded
    }
  }
};
```

### Hooks hiện tại ([`useDashboard.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/hooks/useDashboard.ts))

```typescript
const useMock = API_CONFIG.useMockData;

export function useMetrics() {
  return useQuery({
    queryKey: ['metrics'],
    queryFn: useMock
      ? () => Promise.resolve(dashboardMock.kpis)   // ← fake
      : () => dashboardService.getMetrics(),
    staleTime: 5 * 60 * 1000,
  });
}
```

### Service đã định nghĩa ([`dashboard.service.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/services/dashboard.service.ts))

```typescript
// Service đã có, backend chưa implement đầy đủ
export const dashboardService = {
  getHealth:     () => apiClient.get<EngineHealth[]>(`${BASE}/health`),
  getMetrics:    () => apiClient.get<KPIData>(`${BASE}/metrics`),
  getThroughput: () => apiClient.get<ThroughputData>(`${BASE}/throughput`),
  getHeatmap:    () => apiClient.get<Record<string, unknown>>(`${BASE}/heatmap`),
};
```

---

## 2. Backend API cần implement

Base path: `/v1/console/dashboard`

### 2.1 GET /v1/console/dashboard/metrics

Trả về KPI tổng hợp từ database.

**Response schema** (khớp với [`KPIData`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/types/dashboard.ts)):
```json
{
  "activeAgents": 42,
  "recallLatencyP50Ms": 45,
  "recallLatencyP95Ms": 120,
  "contextSavingsPct": 35.5,
  "graphNodesTotal": 154200,
  "graphEdgesTotal": 489000,
  "graphGrowth24h": 1200,
  "errorRatePct": 0.5,
  "activeSessions": 128,
  "activeProfiles": 1500,
  "memoryVersions": 45000
}
```

**Nguồn dữ liệu**:
| Field | Nguồn |
|---|---|
| `activeAgents` | PostgreSQL — count active API keys đang có request trong 1h gần nhất |
| `recallLatencyP50Ms` / `P95Ms` | Prometheus metrics — `vnp_recall_latency_ms` histogram |
| `contextSavingsPct` | Calculated: `(1 - avg_context_tokens / naive_tokens) * 100` |
| `graphNodesTotal` / `graphEdgesTotal` | Neo4j: `MATCH (n) RETURN count(n)`, `MATCH ()-[r]->() RETURN count(r)` |
| `graphGrowth24h` | Neo4j: nodes created in last 24h |
| `errorRatePct` | Prometheus: `rate(vnp_errors_total[1h]) / rate(vnp_requests_total[1h]) * 100` |
| `activeSessions` | PostgreSQL: count sessions với `status='active'` |
| `activeProfiles` | PostgreSQL: count distinct user_ids với profile entries |
| `memoryVersions` | PostgreSQL/Redis: total memory version count (Supermemory) |

### 2.2 GET /v1/console/dashboard/health

Trả về health status của từng engine.

**Response schema** (array of [`EngineHealth`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/types/dashboard.ts)):
```json
[
  {
    "name": "memobase",
    "role": "Profile Engine",
    "status": "Healthy",
    "latencyP50Ms": 45,
    "latencyP95Ms": 110,
    "queueDepth": 0,
    "uptimeSeconds": 36000,
    "lastCheck": "2026-06-16T13:00:00Z"
  },
  {
    "name": "openviking",
    "role": "Procedural Memory",
    "status": "Warning",
    "latencyP50Ms": 150,
    "latencyP95Ms": 400,
    "queueDepth": 45,
    "uptimeSeconds": 36000,
    "lastCheck": "2026-06-16T13:00:00Z"
  }
  // ... tất cả 7 engines
]
```

**Nguồn dữ liệu**:
- Gọi gRPC health check đến từng engine service (tích hợp với `InProcessRegistry`)
- Lấy latency từ Prometheus metrics per service
- `queueDepth`: từ NATS JetStream consumer pending messages
- `uptimeSeconds`: từ process start time

**Status mapping**:
```
latencyP95Ms < 200ms  → "Healthy"
200ms ≤ P95 < 500ms   → "Warning"
P95 ≥ 500ms hoặc unreachable → "Critical"
```

### 2.3 GET /v1/console/dashboard/throughput

Trả về throughput metrics theo thời gian.

**Query params**: `?window=1h` (default `1h`, options: `5m`, `15m`, `1h`, `6h`, `24h`)

**Response schema** ([`ThroughputData`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/types/dashboard.ts)):
```json
{
  "window": "1h",
  "engines": {
    "memobase": {
      "ingestPerSec": 10.5,
      "recallPerSec": 52.3,
      "embedPerSec": 10.5,
      "profileExtractionsPerSec": 5.2
    },
    "graphiti": {
      "ingestPerSec": 15.1,
      "recallPerSec": 83.7,
      "embedPerSec": 15.1
    }
    // ... tất cả engines
  }
}
```

**Nguồn dữ liệu**: Prometheus `rate()` queries per engine per operation type.

### 2.4 GET /v1/console/dashboard/heatmap

Trả về memory activity heatmap data (24h × 7 days grid).

**Response schema**:
```json
{
  "points": [
    { "x": 0, "y": 0, "density": 45 },
    { "x": 1, "y": 0, "density": 12 }
  ],
  "xLabel": "Hour of day",
  "yLabel": "Day of week",
  "maxDensity": 500
}
```

**Nguồn dữ liệu**: PostgreSQL — count requests grouped by hour + day_of_week, aggregated từ audit log hoặc NATS event log.

---

## 3. Frontend thay đổi

### 3.1 Xóa mock dependency trong `useDashboard.ts`

```typescript
// TRƯỚC
import { dashboardMock } from '../mock/dashboard.mock';
const useMock = API_CONFIG.useMockData;

export function useMetrics() {
  return useQuery({
    queryKey: ['metrics'],
    queryFn: useMock
      ? () => Promise.resolve(dashboardMock.kpis)
      : () => dashboardService.getMetrics(),
  });
}
```

```typescript
// SAU — chỉ dùng service thực
import { dashboardService } from '../services/dashboard.service';

export function useMetrics() {
  return useQuery({
    queryKey: ['metrics'],
    queryFn: () => dashboardService.getMetrics(),
    staleTime: 5 * 60 * 1000,
    refetchInterval: 60 * 1000,  // Auto-refresh mỗi phút
  });
}

export function useEngineHealth() {
  return useQuery({
    queryKey: ['engineHealth'],
    queryFn: () => dashboardService.getHealth(),
    staleTime: 30 * 1000,
    refetchInterval: 30 * 1000,  // Refresh mỗi 30s
  });
}

export function useThroughput() {
  return useQuery({
    queryKey: ['throughput'],
    queryFn: () => dashboardService.getThroughput(),
    staleTime: 30 * 1000,
    refetchInterval: 30 * 1000,
  });
}
```

### 3.2 Thêm heatmap hook

```typescript
export function useDashboardHeatmap() {
  return useQuery({
    queryKey: ['dashboard', 'heatmap'],
    queryFn: () => dashboardService.getHeatmap(),
    staleTime: 5 * 60 * 1000,
  });
}
```

### 3.3 Cập nhật `dashboard.service.ts`

Thêm `window` param vào `getThroughput`:

```typescript
getThroughput: (window = '1h') =>
  apiClient.get<ThroughputData>(`${BASE}/throughput?window=${window}`),
```

---

## 4. Điều kiện hoàn thành

- [ ] `GET /v1/console/dashboard/metrics` trả về dữ liệu từ PostgreSQL + Prometheus
- [ ] `GET /v1/console/dashboard/health` gọi health check đến 7 engines và tổng hợp kết quả
- [ ] `GET /v1/console/dashboard/throughput` trả về Prometheus rate metrics
- [ ] `GET /v1/console/dashboard/heatmap` trả về activity heatmap từ database
- [ ] Dashboard UI không còn import từ `dashboard.mock.ts`
- [ ] Dashboard tự động refresh mỗi 30-60 giây (polling)
- [ ] Loading skeleton và error state hiển thị đúng khi API slow/fail
- [ ] Dashboard hiển thị `status: "Warning"` cho engine openviking nếu thực sự đang quá tải

---

## 5. Notes

> **Caching**: Các metrics dashboard có thể cache ở Redis với TTL 15-30s để tránh overload Prometheus.

> **Real-time**: WebSocket `/v1/console/ws` (đã có trong PRD) có thể được dùng về sau để push health updates thay vì polling.
