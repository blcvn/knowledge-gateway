# API-SOL-011 — Infrastructure API Client + Hooks

| Field | Value |
|---|---|
| **Solution ID** | API-SOL-011 |
| **Status** | ✅ IMPLEMENTED — 2026-06-17 |
| **CR** | [CR-010 — Infrastructure](../../../../specs/crs/v0/ui/CR-010-INFRASTRUCTURE.md) |
| **Target files** | `ui/src/api/clients/infra.client.ts`, `ui/src/api/hooks/useInfrastructure.ts` |
| **Implemented files** | `ui/src/services/infrastructure.service.ts` · `ui/src/hooks/useInfrastructure.ts` · `ui/src/types/infrastructure.ts` |

---

## Types

### `ui/src/types/infrastructure.ts`

```typescript
export interface ServiceInfo {
  name:    string;
  version: string;
  status:  'Healthy' | 'Warning' | 'Critical';
  uptime:  number;    // seconds
}

export interface DatabaseHealth {
  name:       string;
  type:       'PostgreSQL' | 'Neo4j' | 'Redis' | 'NATS' | 'MinIO' | 'Qdrant';
  status:     'Healthy' | 'Warning' | 'Critical';
  latency_ms: number;
}

export interface ResourceMetrics {
  service:         string;
  cpu_usage_pct:   number;
  memory_usage_mb: number;
  disk_usage_pct:  number;
}

export interface InfraTopology {
  mode:        'monolith' | 'gateway';
  node_count:  number;
  services:    ServiceInfo[];
}

export interface DeploymentInfo {
  service:    string;
  version:    string;
  git_commit: string;
  started_at: string;
}
```

---

## Implementation

### `ui/src/api/clients/infra.client.ts`

```typescript
import { httpClient } from './http.client';
import type { ServiceInfo, DatabaseHealth, ResourceMetrics, InfraTopology, DeploymentInfo } from '../../types/infrastructure';

const BASE = '/v1/console/infra';

export const infraClient = {
  getTopology: async (): Promise<InfraTopology> => {
    const { data } = await httpClient.get<InfraTopology>(`${BASE}/topology`);
    return data;
  },

  getServices: async (): Promise<ServiceInfo[]> => {
    const { data } = await httpClient.get<ServiceInfo[]>(`${BASE}/services`);
    return data;
  },

  getDatabases: async (): Promise<DatabaseHealth[]> => {
    const { data } = await httpClient.get<DatabaseHealth[]>(`${BASE}/databases`);
    return data;
  },

  getResources: async (): Promise<ResourceMetrics[]> => {
    const { data } = await httpClient.get<ResourceMetrics[]>(`${BASE}/resources`);
    return data;
  },

  getDeployments: async (): Promise<DeploymentInfo[]> => {
    const { data } = await httpClient.get<DeploymentInfo[]>(`${BASE}/deployments`);
    return data;
  },
};
```

### `ui/src/api/hooks/useInfrastructure.ts`

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

/** Topology ít thay đổi → staleTime 5 phút */
export const useTopology = () => useQuery({
  queryKey:  keys.topology(),
  queryFn:   () => infraClient.getTopology(),
  staleTime: 5 * 60_000,
});

export const useServiceHealth = () => useQuery({
  queryKey:        keys.services(),
  queryFn:         () => infraClient.getServices(),
  refetchInterval: 30_000,
});

export const useDatabaseHealth = () => useQuery({
  queryKey:        keys.databases(),
  queryFn:         () => infraClient.getDatabases(),
  refetchInterval: 30_000,
});

export const useResourceMetrics = () => useQuery({
  queryKey:        keys.resources(),
  queryFn:         () => infraClient.getResources(),
  refetchInterval: 30_000,
});

export const useDeployments = () => useQuery({
  queryKey:  keys.deployments(),
  queryFn:   () => infraClient.getDeployments(),
  staleTime: 5 * 60_000,
});
```
