# CR-008 — Observability: Mock → Real API (OpenTelemetry)

| Field | Value |
|---|---|
| **CR ID** | CR-008 |
| **Title** | Observability: Lấy Metrics, Traces, Errors từ backend infra |
| **Type** | Feature Implementation |
| **Priority** | P1 — High |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Module** | Observability |
| **Files thay đổi** | `ui/src/mock/observability.mock.ts`, `ui/src/hooks/useObservability.ts`, `ui/src/services/observability.service.ts` |

---

## 1. Hiện trạng

Mock data ([`observability.mock.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/mock/observability.mock.ts)):
Hardcode 7 traces, 3 errors (ví dụ: Neo4j conn refused, embedding rate limit) và metrics tổng hợp.

---

## 2. Backend API cần implement

Base path: `/v1/console/observability`
Data source: Prometheus queries, OpenTelemetry trace store.

### 2.1 Metrics

- `GET /v1/console/observability/metrics`

Trả về time-series data để vẽ chart (latency, request rate, error rate). Khớp `MetricPoint[]`.

### 2.2 Traces

- `GET /v1/console/observability/traces`
- `GET /v1/console/observability/traces/{id}`

Tìm kiếm distributed tracing spans của hệ thống (ví dụ trace qua vnp-gateway -> vnp-search-hub -> graphiti).

**Response schema** (`TraceSpan`):
```json
{
  "id": "tr_1",
  "trace_id": "abc123xyz",
  "span_id": "sp_001",
  "operation": "POST /v1/memory/search",
  "service": "vnp-gateway",
  "duration": 310,
  "status": "slow",
  "timestamp": "2026-06-16T12:00:00Z"
}
```

### 2.3 Errors

- `GET /v1/console/observability/errors`

Aggregated error logs từ Elasticsearch/slog.

**Response schema** (`ErrorEntry`):
```json
{
  "id": "err_1",
  "message": "API rate limit exceeded",
  "service": "supermemory",
  "count": 7,
  "lastOccurrence": "2026-06-16T12:00:00Z",
  "stack": "..."
}
```

### 2.4 Costs

- `GET /v1/console/observability/costs`

Tổng hợp chi phí LLM token (qua Bifrost gateway).

---

## 3. Frontend thay đổi

### 3.1 Xóa mock dependency trong `useObservability.ts`

```typescript
// SAU
import { useQuery } from '@tanstack/react-query';
import { observabilityService } from '../services/observability.service';

export function useMetrics() {
  return useQuery({
    queryKey: ['observability', 'metrics'],
    queryFn: () => observabilityService.getMetrics(),
    refetchInterval: 60 * 1000,
  });
}

export function useTraces(filters: Record<string, string>) {
  return useQuery({
    queryKey: ['observability', 'traces', filters],
    queryFn: () => observabilityService.getTraces(filters),
  });
}

// ... UseErrors, UseTraceDetail
```

---

## 4. Điều kiện hoàn thành

- [ ] Backend API query được metrics thực từ Prometheus.
- [ ] UI load traces list và error lists thành công.
- [ ] Không còn import data từ file mock.
