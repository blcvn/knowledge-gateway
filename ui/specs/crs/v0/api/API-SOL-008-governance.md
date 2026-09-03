# API-SOL-008 — Governance API Client + Hooks

| Field | Value |
|---|---|
| **Solution ID** | API-SOL-008 |
| **Status** | ✅ IMPLEMENTED — 2026-06-17 |
| **CR** | [CR-007 — Governance](../../../../specs/crs/v0/ui/CR-007-GOVERNANCE.md) |
| **Target files** | `ui/src/api/clients/governance.client.ts`, `ui/src/api/hooks/useGovernance.ts` |
| **Implemented files** | `ui/src/services/governance.service.ts` · `ui/src/hooks/useGovernance.ts` |

---

## Types

### `ui/src/types/governance.ts`

```typescript
export interface Tenant {
  id:        string;
  name:      string;
  slug:      string;
  plan:      'free' | 'pro' | 'enterprise';
  status:    'active' | 'suspended';
  created_at: string;
}

export interface Policy {
  id:         string;
  tenant_id:  string;
  name:       string;
  rego_code:  string;
  scope:      string;
  enabled:    boolean;
  created_at: string;
}

export interface AuditLogEntry {
  id:          string;
  actor_id:    string;
  action:      string;
  entity_type: string;
  entity_id:   string;
  result:      'success' | 'failure';
  created_at:  string;
}

export interface AuditFilters {
  action?:      string;
  actor_id?:    string;
  entity_type?: string;
  from?:        string;
  to?:          string;
}

export interface GDPRPreviewResponse {
  user_id:              string;
  estimated_items:      number;
  breakdown_by_engine:  Record<string, number>;
  warnings:             string[];
}
```

---

## Implementation

### `ui/src/api/clients/governance.client.ts`

```typescript
import { httpClient } from './http.client';
import type { Tenant, Policy, AuditLogEntry, AuditFilters, GDPRPreviewResponse } from '../../types/governance';

const BASE = '/v1/console/governance';

export const governanceClient = {
  // ── Tenants ──────────────────────────────────────────────────────────────
  getTenants: async (): Promise<Tenant[]> => {
    const { data } = await httpClient.get<Tenant[]>(`${BASE}/tenants`);
    return data;
  },
  createTenant: async (payload: Partial<Tenant>): Promise<Tenant> => {
    const { data } = await httpClient.post<Tenant>(`${BASE}/tenants`, payload);
    return data;
  },
  updateTenant: async (id: string, payload: Partial<Tenant>): Promise<Tenant> => {
    const { data } = await httpClient.put<Tenant>(`${BASE}/tenants/${id}`, payload);
    return data;
  },

  // ── Policies ─────────────────────────────────────────────────────────────
  getPolicies: async (): Promise<Policy[]> => {
    const { data } = await httpClient.get<Policy[]>(`${BASE}/policies`);
    return data;
  },
  createPolicy: async (payload: Partial<Policy>): Promise<Policy> => {
    const { data } = await httpClient.post<Policy>(`${BASE}/policies`, payload);
    return data;
  },
  updatePolicy: async (id: string, payload: Partial<Policy>): Promise<Policy> => {
    const { data } = await httpClient.put<Policy>(`${BASE}/policies/${id}`, payload);
    return data;
  },

  // ── Audit Logs ────────────────────────────────────────────────────────────
  getAuditLogs: async (filters: AuditFilters = {}): Promise<AuditLogEntry[]> => {
    const { data } = await httpClient.get<AuditLogEntry[]>(`${BASE}/audit`, {
      params: filters,
    });
    return data;
  },

  // ── GDPR ──────────────────────────────────────────────────────────────────
  previewForget: async (userId: string): Promise<GDPRPreviewResponse> => {
    const { data } = await httpClient.post<GDPRPreviewResponse>(
      `${BASE}/gdpr/forget/preview`,
      { user_id: userId },
    );
    return data;
  },
  executeForget: async (userId: string): Promise<{ success: boolean; deleted_count: number }> => {
    const { data } = await httpClient.post<{ success: boolean; deleted_count: number }>(
      `${BASE}/gdpr/forget`,
      { user_id: userId },
    );
    return data;
  },
};
```

### `ui/src/api/hooks/useGovernance.ts`

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { governanceClient } from '../clients/governance.client';
import type { Tenant, Policy, AuditFilters } from '../../types/governance';

const keys = {
  tenants:    () => ['governance', 'tenants'] as const,
  policies:   () => ['governance', 'policies'] as const,
  audit:      (f: AuditFilters) => ['governance', 'audit', f] as const,
};

export const useTenants = () => useQuery({
  queryKey: keys.tenants(),
  queryFn:  () => governanceClient.getTenants(),
});

export const usePolicies = () => useQuery({
  queryKey: keys.policies(),
  queryFn:  () => governanceClient.getPolicies(),
});

export const useAuditLogs = (filters: AuditFilters = {}) => useQuery({
  queryKey: keys.audit(filters),
  queryFn:  () => governanceClient.getAuditLogs(filters),
});

export function useCreateTenant() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: governanceClient.createTenant,
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.tenants() }),
  });
}

export function useUpdateTenant() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Tenant> }) =>
      governanceClient.updateTenant(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.tenants() }),
  });
}

export function useCreatePolicy() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: governanceClient.createPolicy,
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.policies() }),
  });
}

export function useUpdatePolicy() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Policy> }) =>
      governanceClient.updatePolicy(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.policies() }),
  });
}

/** Preview GDPR deletion trước khi thực thi — hiển thị số items sẽ bị xóa */
export function useGDPRPreview() {
  return useMutation({
    mutationFn: (userId: string) => governanceClient.previewForget(userId),
  });
}

/** Thực thi GDPR forget — xóa toàn bộ user data */
export function useGDPRForget() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => governanceClient.executeForget(userId),
    onSuccess:  () => qc.invalidateQueries({ queryKey: ['governance'] }),
  });
}
```
