# API-SOL-006 — Adaptive Memory API Client + Hooks

| Field | Value |
|---|---|
| **Solution ID** | API-SOL-006 |
| **Status** | ✅ IMPLEMENTED — 2026-06-17 |
| **CR** | [CR-005 — Adaptive Memory](../../../../specs/crs/v0/ui/CR-005-ADAPTIVE.md) |
| **Target files** | `ui/src/api/clients/adaptive.client.ts`, `ui/src/api/hooks/useAdaptive.ts` |
| **Implemented files** | `ui/src/services/adaptive.service.ts` · `ui/src/hooks/useAdaptiveMemory.ts` |

---

## Types

### `ui/src/types/adaptive.ts`

```typescript
export interface AdaptiveMemory {
  id:          string;
  content:     string;
  source:      string;
  is_latest:   boolean;
  version:     number;
  created_at:  string;
  updated_at:  string;
}

export interface MemoryVersion {
  id:             string;
  memory_id:      string;
  content:        string;
  version_number: number;
  is_latest:      boolean;
  diff:           string;
  created_at:     string;
}

export interface ExternalConnector {
  id:             string;
  type:           'google_drive' | 'gmail' | 'notion' | 'onedrive' | 'github';
  status:         'Connected' | 'Disconnected' | 'Error';
  document_count: number;
  last_sync?:     string;
}

export interface AdaptiveAnalytics {
  creation_rate:        number;  // memories/hour
  deletion_rate:        number;
  contradiction_count:  number;
  connector_sync_count: number;  // last 24h
  storage_usage_bytes:  number;
}

export interface ForgetRules {
  ttl_days:          number;
  inactivity_days:   number;
  low_score_threshold: number;
  auto_prune:        boolean;
}
```

---

## Implementation

### `ui/src/api/clients/adaptive.client.ts`

```typescript
import { httpClient } from './http.client';
import type { AdaptiveMemory, MemoryVersion, ExternalConnector, AdaptiveAnalytics, ForgetRules } from '../../types/adaptive';

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
    const { data } = await httpClient.post<{ job_id: string }>(`${BASE}/connectors/${id}/sync`, {});
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

### `ui/src/api/hooks/useAdaptive.ts`

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

export const useMemoryVersions = (id: string) => useQuery({
  queryKey: keys.versions(id),
  queryFn:  () => adaptiveClient.getMemoryVersions(id),
  enabled:  !!id,
});

export const useConnectors = () => useQuery({
  queryKey: keys.connectors(),
  queryFn:  () => adaptiveClient.getConnectors(),
});

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
    mutationFn: adaptiveClient.createConnector,
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
