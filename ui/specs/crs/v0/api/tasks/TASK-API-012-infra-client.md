# TASK-API-012 — Infrastructure API Client + Hooks

**Task ID:** TASK-API-012
**Status:** ✅ COMPLETED — 2026-06-17
**Sprint:** 4 — P2 Modules
**Solution:** [API-SOL-011](../API-SOL-011-infra.md)
**Depends on:** TASK-API-001, TASK-API-002
**Ước tính:** 1h
**Priority:** P2

---

## Công việc cụ thể

### 1. Tạo `ui/src/api/clients/infra.client.ts`

```typescript
import { httpClient } from './http.client';
import type {
  ServiceInfo, DatabaseHealth, ResourceMetrics, InfraTopology, DeploymentInfo
} from '../../types/infrastructure';

const BASE = '/v1/console/infra';

export const infraClient = {
  getTopology:   async (): Promise<InfraTopology> => {
    const { data } = await httpClient.get<InfraTopology>(`${BASE}/topology`);
    return data;
  },
  getServices:   async (): Promise<ServiceInfo[]> => {
    const { data } = await httpClient.get<ServiceInfo[]>(`${BASE}/services`);
    return data;
  },
  getDatabases:  async (): Promise<DatabaseHealth[]> => {
    const { data } = await httpClient.get<DatabaseHealth[]>(`${BASE}/databases`);
    return data;
  },
  getResources:  async (): Promise<ResourceMetrics[]> => {
    const { data } = await httpClient.get<ResourceMetrics[]>(`${BASE}/resources`);
    return data;
  },
  getDeployments: async (): Promise<DeploymentInfo[]> => {
    const { data } = await httpClient.get<DeploymentInfo[]>(`${BASE}/deployments`);
    return data;
  },
};
```

### 2. Tạo `ui/src/api/hooks/useInfrastructure.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import { infraClient } from '../clients/infra.client';

const keys = {
  topology:    () => ['infra', 'topology'] as const,
  services:    () => ['infra', 'services'] as const,
  databases:   () => ['infra', 'databases'] as const,
  resources:   () => ['infra', 'resources'] as const,
  deployments: () => ['infra', 'deployments'] as const,
};

/** staleTime 5m — topology không thay đổi thường xuyên */
export const useTopology    = () => useQuery({ queryKey: keys.topology(),    queryFn: infraClient.getTopology,    staleTime: 5 * 60_000 });
export const useServiceHealth = () => useQuery({ queryKey: keys.services(), queryFn: infraClient.getServices,   refetchInterval: 30_000 });
export const useDatabaseHealth = () => useQuery({ queryKey: keys.databases(), queryFn: infraClient.getDatabases, refetchInterval: 30_000 });
export const useResourceMetrics = () => useQuery({ queryKey: keys.resources(), queryFn: infraClient.getResources, refetchInterval: 30_000 });
export const useDeployments  = () => useQuery({ queryKey: keys.deployments(), queryFn: infraClient.getDeployments, staleTime: 5 * 60_000 });
```

---

## Acceptance Criteria

- [x] `GET /v1/console/infra/topology` → `{ mode: 'monolith', node_count: 35 }`
- [x] `GET /v1/console/infra/services` → 35 services với status
- [x] `GET /v1/console/infra/databases` → PostgreSQL, Neo4j, Redis, NATS health
- [x] `useServiceHealth()` poll 30s
- [x] `npx tsc --noEmit` không lỗi
