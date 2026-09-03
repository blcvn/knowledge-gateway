# TASK-UI-009 — Tạo `adaptive.service.ts` + Refactor `useAdaptiveMemory.ts`

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-009 |
| **Layer** | Frontend — TypeScript |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-007 §5](../solutions/SOL-007-Gap-Fixes.md) |
| **Priority** | 🟠 P1 |
| **Depends On** | TASK-UI-001 |
| **Estimated** | 1.5h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `ui/src/services/adaptive.service.ts` |
| MODIFY | `ui/src/hooks/useAdaptiveMemory.ts` |

---

## Implementation

### File: `ui/src/services/adaptive.service.ts`

```typescript
import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type { AdaptiveMemory, MemoryVersion, AdaptiveAnalytics, ForgetRules } from '../types/adaptive';

const BASE = API_CONFIG.adaptive;

export interface ExternalConnector {
  id: string;
  type: 'google_drive' | 'gmail' | 'notion' | 'onedrive' | 'github';
  status: 'Connected' | 'Disconnected' | 'Error';
  document_count: number;
  last_sync?: string;
}

export const adaptiveService = {
  getMemories: () =>
    apiClient.get<AdaptiveMemory[]>(`${BASE}/memories`),

  getMemoryVersions: (id: string) =>
    apiClient.get<MemoryVersion[]>(`${BASE}/memories/${encodeURIComponent(id)}/versions`),

  getConnectors: () =>
    apiClient.get<ExternalConnector[]>(`${BASE}/connectors`),

  createConnector: (config: Partial<ExternalConnector>) =>
    apiClient.post<ExternalConnector>(`${BASE}/connectors`, config),

  syncConnector: (id: string) =>
    apiClient.post<{ job_id: string }>(`${BASE}/connectors/${id}/sync`, {}),

  getAnalytics: () =>
    apiClient.get<AdaptiveAnalytics>(`${BASE}/analytics`),

  getForgetRules: () =>
    apiClient.get<ForgetRules>(`${BASE}/forget-rules`),

  updateForgetRules: (rules: ForgetRules) =>
    apiClient.put<ForgetRules>(`${BASE}/forget-rules`, rules),
};
```

### File: `ui/src/hooks/useAdaptiveMemory.ts`

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { adaptiveService } from '../services/adaptive.service';

export function useAdaptiveMemories() {
  return useQuery({
    queryKey: ['adaptive', 'memories'],
    queryFn: () => adaptiveService.getMemories(),
    staleTime: 60_000,
  });
}

export function useAdaptiveMemoryVersions(id: string) {
  return useQuery({
    queryKey: ['adaptive', 'memories', id, 'versions'],
    queryFn: () => adaptiveService.getMemoryVersions(id),
    enabled: !!id,
  });
}

export function useConnectors() {
  return useQuery({
    queryKey: ['adaptive', 'connectors'],
    queryFn: () => adaptiveService.getConnectors(),
  });
}

export function useAdaptiveAnalytics() {
  return useQuery({
    queryKey: ['adaptive', 'analytics'],
    queryFn: () => adaptiveService.getAnalytics(),
    refetchInterval: 60_000,
  });
}

export function useForgetRules() {
  return useQuery({
    queryKey: ['adaptive', 'forget-rules'],
    queryFn: () => adaptiveService.getForgetRules(),
  });
}

export function useSyncConnector() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => adaptiveService.syncConnector(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['adaptive', 'connectors'] }),
  });
}

export function useCreateConnector() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: adaptiveService.createConnector,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['adaptive', 'connectors'] }),
  });
}

export function useUpdateForgetRules() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: adaptiveService.updateForgetRules,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['adaptive', 'forget-rules'] }),
  });
}
```

---

## Verification

```bash
cd ui
npx tsc --noEmit
grep -r "adaptiveMock\|defaultAnalyticsMock" ui/src/hooks/ # phải trống
```
