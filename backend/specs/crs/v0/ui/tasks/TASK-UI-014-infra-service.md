# TASK-UI-014 — Tạo `infrastructure.service.ts` + Refactor `useInfrastructure.ts`

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-014 |
| **Layer** | Frontend — TypeScript |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-007 §10](../solutions/SOL-007-Gap-Fixes.md) |
| **Priority** | 🟡 P2 |
| **Depends On** | TASK-UI-001 |
| **Estimated** | 1h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `ui/src/services/infrastructure.service.ts` |
| MODIFY | `ui/src/hooks/useInfrastructure.ts` |

---

## Implementation

### File: `ui/src/services/infrastructure.service.ts`

```typescript
import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';

const BASE = API_CONFIG.infra;

export interface ServiceInfo {
  name: string;
  version: string;
  status: 'Healthy' | 'Warning' | 'Critical';
  uptime: number;  // seconds
}

export interface DatabaseHealth {
  name: string;
  type: 'PostgreSQL' | 'Neo4j' | 'Redis' | 'NATS' | 'MinIO' | 'Qdrant';
  status: 'Healthy' | 'Warning' | 'Critical';
  latency_ms: number;
}

export interface ResourceMetrics {
  service: string;
  cpu_usage_pct: number;
  memory_usage_mb: number;
  disk_usage_pct: number;
}

export interface InfraTopology {
  mode: 'monolith' | 'gateway';
  node_count: number;
  services: ServiceInfo[];
}

export interface DeploymentInfo {
  service: string;
  version: string;
  git_commit: string;
  started_at: string;
}

export const infrastructureService = {
  getTopology: () =>
    apiClient.get<InfraTopology>(`${BASE}/topology`),

  getServiceHealth: () =>
    apiClient.get<ServiceInfo[]>(`${BASE}/services`),

  getServiceDetail: (name: string) =>
    apiClient.get<ServiceInfo>(`${BASE}/services/${name}`),

  getDatabaseHealth: () =>
    apiClient.get<DatabaseHealth[]>(`${BASE}/databases`),

  getResourceMetrics: () =>
    apiClient.get<ResourceMetrics[]>(`${BASE}/resources`),

  getDeployments: () =>
    apiClient.get<DeploymentInfo[]>(`${BASE}/deployments`),
};
```

### File: `ui/src/hooks/useInfrastructure.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import { infrastructureService } from '../services/infrastructure.service';

export function useTopology() {
  return useQuery({
    queryKey: ['infra', 'topology'],
    queryFn: () => infrastructureService.getTopology(),
    staleTime: 5 * 60_000,  // Topology ít thay đổi
  });
}

export function useServiceHealth() {
  return useQuery({
    queryKey: ['infra', 'services'],
    queryFn: () => infrastructureService.getServiceHealth(),
    refetchInterval: 30_000,
  });
}

export function useDatabaseHealth() {
  return useQuery({
    queryKey: ['infra', 'databases'],
    queryFn: () => infrastructureService.getDatabaseHealth(),
    refetchInterval: 30_000,
  });
}

export function useResourceMetrics() {
  return useQuery({
    queryKey: ['infra', 'resources'],
    queryFn: () => infrastructureService.getResourceMetrics(),
    refetchInterval: 30_000,
  });
}

export function useDeployments() {
  return useQuery({
    queryKey: ['infra', 'deployments'],
    queryFn: () => infrastructureService.getDeployments(),
    staleTime: 5 * 60_000,
  });
}
```

---

## Verification

```bash
cd ui
npx tsc --noEmit
grep -r "infrastructureMock\|infrastructure\.mock" ui/src/hooks/ # phải trống
```
