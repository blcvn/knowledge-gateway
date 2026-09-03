# TASK-API-007 — Adaptive Memory API Client + Hooks

**Task ID:** TASK-API-007
**Status:** ✅ COMPLETED — 2026-06-17
**Sprint:** 3 — P1 Modules
**Solution:** [API-SOL-006](../API-SOL-006-adaptive.md)
**Depends on:** TASK-API-001, TASK-API-002
**Ước tính:** 1.5h
**Priority:** P1

---

## Mục tiêu

Implement Adaptive Memory (Supermemory) Data Access Layer:
1. `adaptive.client.ts` — memories, connectors, analytics, forget-rules
2. `useAdaptive.ts` — hooks với mutations cho connector sync

---

## Công việc cụ thể

### 1. Tạo `ui/src/api/clients/adaptive.client.ts`

```typescript
import { httpClient } from './http.client';
import type {
  AdaptiveMemory, ExternalConnector, AdaptiveAnalytics, ForgetRules
} from '../../types/adaptive';
import type { MemoryVersion } from '../../types/memory';

const BASE = '/v1/console/adaptive';

export const adaptiveClient = {
  getMemories: async (): Promise<AdaptiveMemory[]> => {
    const { data } = await httpClient.get<AdaptiveMemory[]>(`${BASE}/memories`);
    return data;
  },

  getMemoryVersions: async (id: string): Promise<MemoryVersion[]> => {
    const { data } = await httpClient.get<MemoryVersion[]>(
      `${BASE}/memories/${encodeURIComponent(id)}/versions`,
    );
    return data;
  },

  getConnectors: async (): Promise<ExternalConnector[]> => {
    const { data } = await httpClient.get<ExternalConnector[]>(`${BASE}/connectors`);
    return data;
  },

  createConnector: async (config: Partial<ExternalConnector>): Promise<ExternalConnector> => {
    const { data } = await httpClient.post<ExternalConnector>(`${BASE}/connectors`, config);
    return data;
  },

  syncConnector: async (id: string): Promise<{ job_id: string }> => {
    const { data } = await httpClient.post<{ job_id: string }>(
      `${BASE}/connectors/${id}/sync`, {},
    );
    return data;
  },

  getAnalytics: async (): Promise<AdaptiveAnalytics> => {
    const { data } = await httpClient.get<AdaptiveAnalytics>(`${BASE}/analytics`);
    return data;
  },

  getForgetRules: async (): Promise<ForgetRules> => {
    const { data } = await httpClient.get<ForgetRules>(`${BASE}/forget-rules`);
    return data;
  },

  updateForgetRules: async (rules: Partial<ForgetRules>): Promise<ForgetRules> => {
    const { data } = await httpClient.put<ForgetRules>(`${BASE}/forget-rules`, rules);
    return data;
  },
};
```

### 2. Tạo `ui/src/api/hooks/useAdaptive.ts`

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { adaptiveClient } from '../clients/adaptive.client';
import type { ExternalConnector, ForgetRules } from '../../types/adaptive';

const keys = {
  memories:    () => ['adaptive', 'memories'] as const,
  versions:    (id: string) => ['adaptive', 'memories', id, 'versions'] as const,
  connectors:  () => ['adaptive', 'connectors'] as const,
  analytics:   () => ['adaptive', 'analytics'] as const,
  forgetRules: () => ['adaptive', 'forget-rules'] as const,
};

export const useAdaptiveMemories = () => useQuery({
  queryKey:  keys.memories(),
  queryFn:   () => adaptiveClient.getMemories(),
  staleTime: 60_000,
});

export const useAdaptiveVersions = (id: string) => useQuery({
  queryKey: keys.versions(id),
  queryFn:  () => adaptiveClient.getMemoryVersions(id),
  enabled:  !!id,
});

export const useConnectors = () => useQuery({
  queryKey: keys.connectors(),
  queryFn:  () => adaptiveClient.getConnectors(),
});

/** Poll 60s — analytics ít thay đổi */
export const useAdaptiveAnalytics = () => useQuery({
  queryKey:        keys.analytics(),
  queryFn:         () => adaptiveClient.getAnalytics(),
  refetchInterval: 60_000,
});

export const useForgetRules = () => useQuery({
  queryKey: keys.forgetRules(),
  queryFn:  () => adaptiveClient.getForgetRules(),
});

export function useCreateConnector() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (config: Partial<ExternalConnector>) => adaptiveClient.createConnector(config),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.connectors() }),
  });
}

export function useSyncConnector() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => adaptiveClient.syncConnector(id),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.connectors() }),
  });
}

export function useUpdateForgetRules() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (rules: Partial<ForgetRules>) => adaptiveClient.updateForgetRules(rules),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.forgetRules() }),
  });
}
```

---

## Files tạo ra

```
ui/src/api/
├── clients/adaptive.client.ts  ← NEW
└── hooks/useAdaptive.ts        ← NEW
```

---

## Acceptance Criteria

- [x] `GET /v1/console/adaptive/memories` → `AdaptiveMemory[]`
- [x] `GET /v1/console/adaptive/memories/{id}/versions` → `MemoryVersion[]`
- [x] `GET /v1/console/adaptive/connectors` → `ExternalConnector[]`
- [x] `POST /v1/console/adaptive/connectors/{id}/sync` → triggers job, invalidates connectors cache
- [x] `GET /v1/console/adaptive/analytics` → `AdaptiveAnalytics` (5 fields)
- [x] `PUT /v1/console/adaptive/forget-rules` → cập nhật rules, invalidates cache
- [x] `npx tsc --noEmit` không lỗi
