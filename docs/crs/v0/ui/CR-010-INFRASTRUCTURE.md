# CR-010 — Infrastructure Health: Mock → Real API

| Field | Value |
|---|---|
| **CR ID** | CR-010 |
| **Title** | Infrastructure Health: Quản lý topology, resources và database status |
| **Type** | Feature Implementation |
| **Priority** | P2 — Medium |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Module** | Infrastructure |
| **Files thay đổi** | `ui/src/mock/infrastructure.mock.ts`, `ui/src/hooks/useInfrastructure.ts`, `ui/src/services/infrastructure.service.ts` |

---

## 1. Hiện trạng

Mock data ([`infrastructure.mock.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/mock/infrastructure.mock.ts)):
Hardcode status của gateway, postgres database và system resource.

---

## 2. Backend API cần implement

Base path: `/v1/console/infra`
Data source: Service Mesh (nếu distributed) hoặc InProcessRegistry (nếu monolith), Postgres/Neo4j/Redis native pings.

### 2.1 Service Topology & Health

- `GET /v1/console/infra/topology`
- `GET /v1/console/infra/services`
- `GET /v1/console/infra/services/{name}`

**Response schema** (`ServiceInfo`):
```json
[
  {
    "name": "vnp-gateway",
    "version": "1.0.0",
    "status": "Healthy",
    "uptime": 86400
  }
]
```

### 2.2 Database Health

- `GET /v1/console/infra/databases`

Ping trực tiếp DB, lấy connection pool metrics, latency.

**Response schema** (`DatabaseHealth`):
```json
[
  {
    "name": "Postgres-Primary",
    "type": "PostgreSQL",
    "status": "Healthy",
    "latency_ms": 2
  },
  {
    "name": "Neo4j-Graph",
    "type": "Neo4j",
    "status": "Healthy",
    "latency_ms": 15
  }
]
```

### 2.3 Resource Metrics

- `GET /v1/console/infra/resources`

OS level metrics (Prometheus node_exporter data).

**Response schema** (`ResourceMetrics`):
```json
[
  {
    "service": "vnp-gateway",
    "cpu_usage_pct": 25.5,
    "memory_usage_mb": 512,
    "disk_usage_pct": 42
  }
]
```

---

## 3. Frontend thay đổi

### 3.1 Xóa mock dependency trong `useInfrastructure.ts`

```typescript
// SAU
import { useQuery } from '@tanstack/react-query';
import { infrastructureService } from '../services/infrastructure.service';

export function useServiceHealth() {
  return useQuery({
    queryKey: ['infra', 'services'],
    queryFn: () => infrastructureService.getServiceHealth(),
    refetchInterval: 30 * 1000,
  });
}

export function useDatabaseHealth() {
  return useQuery({
    queryKey: ['infra', 'databases'],
    queryFn: () => infrastructureService.getDatabaseHealth(),
    refetchInterval: 30 * 1000,
  });
}

export function useResourceMetrics() {
  return useQuery({
    queryKey: ['infra', 'resources'],
    queryFn: () => infrastructureService.getResourceMetrics(),
    refetchInterval: 30 * 1000,
  });
}
```

---

## 4. Điều kiện hoàn thành

- [ ] Infra view render được danh sách databases thực (PostgreSQL, Neo4j, Redis, NATS).
- [ ] Dữ liệu ping latency nhảy đúng.
- [ ] Không còn dùng mock data.
