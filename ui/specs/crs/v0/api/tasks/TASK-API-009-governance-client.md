# TASK-API-009 — Governance API Client + Hooks

**Task ID:** TASK-API-009
**Status:** ✅ COMPLETED — 2026-06-17
**Sprint:** 3 — P1 Modules
**Solution:** [API-SOL-008](../API-SOL-008-governance.md)
**Depends on:** TASK-API-001, TASK-API-002
**Ước tính:** 2h
**Priority:** P1

---

## Mục tiêu

Implement Governance Data Access Layer (tenants, policies, audit, GDPR):
1. `governance.client.ts` — 9 endpoints
2. `useGovernance.ts` — hooks + GDPR preview/forget mutations

---

## Công việc cụ thể

### 1. Tạo `ui/src/api/clients/governance.client.ts`

```typescript
import { httpClient } from './http.client';
import type {
  Tenant, Policy, AuditLogEntry, AuditFilters, GDPRPreviewResponse
} from '../../types/governance';

const BASE = '/v1/console/governance';

export const governanceClient = {
  // Tenants
  getTenants:    async (): Promise<Tenant[]> => {
    const { data } = await httpClient.get<Tenant[]>(`${BASE}/tenants`);
    return data;
  },
  createTenant:  async (payload: Partial<Tenant>): Promise<Tenant> => {
    const { data } = await httpClient.post<Tenant>(`${BASE}/tenants`, payload);
    return data;
  },
  updateTenant:  async (id: string, payload: Partial<Tenant>): Promise<Tenant> => {
    const { data } = await httpClient.put<Tenant>(`${BASE}/tenants/${id}`, payload);
    return data;
  },

  // Policies
  getPolicies:   async (): Promise<Policy[]> => {
    const { data } = await httpClient.get<Policy[]>(`${BASE}/policies`);
    return data;
  },
  createPolicy:  async (payload: Partial<Policy>): Promise<Policy> => {
    const { data } = await httpClient.post<Policy>(`${BASE}/policies`, payload);
    return data;
  },
  updatePolicy:  async (id: string, payload: Partial<Policy>): Promise<Policy> => {
    const { data } = await httpClient.put<Policy>(`${BASE}/policies/${id}`, payload);
    return data;
  },

  // Audit Logs
  getAuditLogs:  async (filters: AuditFilters = {}): Promise<AuditLogEntry[]> => {
    const { data } = await httpClient.get<AuditLogEntry[]>(`${BASE}/audit`, {
      params: filters,
    });
    return data;
  },

  // GDPR
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

### 2. Tạo `ui/src/api/hooks/useGovernance.ts`

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { governanceClient } from '../clients/governance.client';
import type { Tenant, Policy, AuditFilters } from '../../types/governance';

const keys = {
  tenants:  () => ['governance', 'tenants'] as const,
  policies: () => ['governance', 'policies'] as const,
  audit:    (f: AuditFilters) => ['governance', 'audit', f] as const,
};

export const useTenants   = () => useQuery({ queryKey: keys.tenants(),   queryFn: governanceClient.getTenants });
export const usePolicies  = () => useQuery({ queryKey: keys.policies(),  queryFn: governanceClient.getPolicies });
export const useAuditLogs = (f: AuditFilters = {}) => useQuery({
  queryKey: keys.audit(f), queryFn: () => governanceClient.getAuditLogs(f),
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

/** Step 1: preview số items sẽ bị xóa */
export function useGDPRPreview() {
  return useMutation({
    mutationFn: (userId: string) => governanceClient.previewForget(userId),
  });
}

/** Step 2: thực thi xóa — cần xác nhận của user trước */
export function useGDPRForget() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => governanceClient.executeForget(userId),
    onSuccess:  () => qc.invalidateQueries({ queryKey: ['governance'] }),
  });
}
```

---

## GDPR UX Flow

UI phải implement 2-step confirm trước khi gọi `useGDPRForget`:

```typescript
// Step 1: Preview
const preview = useGDPRPreview();
preview.mutate('user_123', {
  onSuccess: (data) => {
    setShowConfirm(true);  // Hiển thị: "Sẽ xóa 450 items từ 5 engines"
  }
});

// Step 2: User confirm → Execute
const forget = useGDPRForget();
forget.mutate('user_123', {
  onSuccess: () => toast.success('User data deleted successfully'),
  onError:   () => toast.error('Deletion failed'),
});
```

---

## Files tạo ra

```
ui/src/api/
├── clients/governance.client.ts  ← NEW
└── hooks/useGovernance.ts        ← NEW
```

---

## Acceptance Criteria

- [x] `GET /v1/console/governance/tenants` → `Tenant[]`
- [x] `POST /v1/console/governance/tenants` → tạo tenant, invalidate cache
- [x] `GET /v1/console/governance/policies` → `Policy[]` với `rego_code`
- [x] `GET /v1/console/governance/audit?action=LOGIN` → filtered `AuditLogEntry[]`
- [x] `POST /v1/console/governance/gdpr/forget/preview` → `GDPRPreviewResponse` với breakdown
- [x] `POST /v1/console/governance/gdpr/forget` → xóa thực, `deleted_count > 0`
- [x] GDPR UI có 2-step confirm (Preview → Confirm → Execute)
- [x] `npx tsc --noEmit` không lỗi
