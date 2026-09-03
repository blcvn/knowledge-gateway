# TASK-UI-011 — Tạo `governance.service.ts` + Refactor `useGovernance.ts`

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-011 |
| **Layer** | Frontend — TypeScript |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-007 §7](../solutions/SOL-007-Gap-Fixes.md) |
| **Priority** | 🟠 P1 |
| **Depends On** | TASK-UI-001 |
| **Estimated** | 1.5h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `ui/src/services/governance.service.ts` |
| MODIFY | `ui/src/hooks/useGovernance.ts` |

---

## Implementation

### File: `ui/src/services/governance.service.ts`

```typescript
import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type { Tenant, Policy, AuditLogEntry } from '../types/governance';

const BASE = API_CONFIG.governance;

export interface GDPRPreviewResponse {
  user_id: string;
  estimated_items: number;
  breakdown_by_engine: Record<string, number>;
  warnings: string[];
}

export const governanceService = {
  // Tenants
  getTenants: () =>
    apiClient.get<Tenant[]>(`${BASE}/tenants`),
  createTenant: (data: Partial<Tenant>) =>
    apiClient.post<Tenant>(`${BASE}/tenants`, data),
  updateTenant: (id: string, data: Partial<Tenant>) =>
    apiClient.put<Tenant>(`${BASE}/tenants/${id}`, data),

  // Policies
  getPolicies: () =>
    apiClient.get<Policy[]>(`${BASE}/policies`),
  createPolicy: (data: Partial<Policy>) =>
    apiClient.post<Policy>(`${BASE}/policies`, data),
  updatePolicy: (id: string, data: Partial<Policy>) =>
    apiClient.put<Policy>(`${BASE}/policies/${id}`, data),

  // Audit
  getAuditLogs: (filters: Record<string, string> = {}) => {
    const qs = new URLSearchParams(filters).toString();
    return apiClient.get<AuditLogEntry[]>(`${BASE}/audit?${qs}`);
  },

  // GDPR
  previewForget: (userId: string) =>
    apiClient.post<GDPRPreviewResponse>(`${BASE}/gdpr/forget/preview`, { user_id: userId }),
  executeForget: (userId: string) =>
    apiClient.post<{ success: boolean; deleted_count: number }>(
      `${BASE}/gdpr/forget`, { user_id: userId }
    ),
};
```

### File: `ui/src/hooks/useGovernance.ts`

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { governanceService } from '../services/governance.service';

export function useTenants() {
  return useQuery({
    queryKey: ['governance', 'tenants'],
    queryFn: () => governanceService.getTenants(),
  });
}

export function usePolicies() {
  return useQuery({
    queryKey: ['governance', 'policies'],
    queryFn: () => governanceService.getPolicies(),
  });
}

export function useAuditLogs(filters: Record<string, string> = {}) {
  return useQuery({
    queryKey: ['governance', 'auditLogs', filters],
    queryFn: () => governanceService.getAuditLogs(filters),
  });
}

export function useCreateTenant() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: governanceService.createTenant,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['governance', 'tenants'] }),
  });
}

export function useUpdateTenant() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Record<string, unknown> }) =>
      governanceService.updateTenant(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['governance', 'tenants'] }),
  });
}

export function useCreatePolicy() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: governanceService.createPolicy,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['governance', 'policies'] }),
  });
}

export function useGDPRPreview() {
  return useMutation({
    mutationFn: governanceService.previewForget,
  });
}

export function useGDPRForget() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: governanceService.executeForget,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['governance', 'auditLogs'] }),
  });
}
```

---

## Verification

```bash
cd ui
npx tsc --noEmit
grep -r "governanceMock" ui/src/hooks/ # phải trống
```
