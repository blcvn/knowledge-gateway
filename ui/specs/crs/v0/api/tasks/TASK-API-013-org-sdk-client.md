# TASK-API-013 — Org & SDK API Client + Hooks

**Task ID:** TASK-API-013
**Status:** ✅ COMPLETED — 2026-06-17
**Sprint:** 4 — P2 Modules
**Solution:** [API-SOL-012](../API-SOL-012-org-sdk.md)
**Depends on:** TASK-API-001, TASK-API-002
**Ước tính:** 2h
**Priority:** P2

---

## Mục tiêu

Implement Org Settings và SDK Management Data Access Layer:
1. `org.client.ts` — org settings, members, API keys, webhooks
2. `useOrg.ts` — hooks với special UX cho API key creation (raw_key shown once)

---

## Công việc cụ thể

### 1. Tạo `ui/src/api/clients/org.client.ts`

```typescript
import { httpClient } from './http.client';
import type {
  OrgSettings, OrgMember, OrgRole,
  APIKey, CreateKeyResponse, RateLimitConfig, Webhook
} from '../../types/org';

const ORG_BASE = '/v1/console/org';
const SDK_BASE = '/v1/console/sdk';

export const orgClient = {
  // Org Settings
  getSettings:    async (): Promise<OrgSettings> => {
    const { data } = await httpClient.get<OrgSettings>(`${ORG_BASE}/settings`);
    return data;
  },
  updateSettings: async (payload: Partial<OrgSettings>): Promise<OrgSettings> => {
    const { data } = await httpClient.put<OrgSettings>(`${ORG_BASE}/settings`, payload);
    return data;
  },
  getMembers:     async (): Promise<OrgMember[]> => {
    const { data } = await httpClient.get<OrgMember[]>(`${ORG_BASE}/members`);
    return data;
  },
  getRoles:       async (): Promise<OrgRole[]> => {
    const { data } = await httpClient.get<OrgRole[]>(`${ORG_BASE}/roles`);
    return data;
  },

  // API Keys
  getKeys: async (): Promise<APIKey[]> => {
    const { data } = await httpClient.get<APIKey[]>(`${SDK_BASE}/keys`);
    return data;
  },
  createKey: async (payload: {
    name: string;
    permissions: string[];
    expires_in_days?: number;
  }): Promise<CreateKeyResponse> => {
    const { data } = await httpClient.post<CreateKeyResponse>(`${SDK_BASE}/keys`, payload);
    return data;
    // IMPORTANT: data.raw_key chỉ trả về 1 lần — UI phải hiển thị ngay
  },
  revokeKey: async (id: string): Promise<void> => {
    await httpClient.delete(`${SDK_BASE}/keys/${id}`);
  },

  // Rate Limits
  getRateLimits: async (): Promise<RateLimitConfig[]> => {
    const { data } = await httpClient.get<RateLimitConfig[]>(`${SDK_BASE}/rate-limits`);
    return data;
  },

  // Webhooks
  getWebhooks: async (): Promise<Webhook[]> => {
    const { data } = await httpClient.get<Webhook[]>(`${SDK_BASE}/webhooks`);
    return data;
  },
  createWebhook: async (payload: {
    url: string;
    events: string[];
    secret?: string;
  }): Promise<Webhook> => {
    const { data } = await httpClient.post<Webhook>(`${SDK_BASE}/webhooks`, payload);
    return data;
  },
  deleteWebhook: async (id: string): Promise<void> => {
    await httpClient.delete(`${SDK_BASE}/webhooks/${id}`);
  },
};
```

### 2. Tạo `ui/src/api/hooks/useOrg.ts`

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { orgClient } from '../clients/org.client';
import type { OrgSettings } from '../../types/org';

const keys = {
  settings:   () => ['org', 'settings'] as const,
  members:    () => ['org', 'members'] as const,
  roles:      () => ['org', 'roles'] as const,
  keys:       () => ['sdk', 'keys'] as const,
  rateLimits: () => ['sdk', 'rate-limits'] as const,
  webhooks:   () => ['sdk', 'webhooks'] as const,
};

export const useOrgSettings  = () => useQuery({ queryKey: keys.settings(), queryFn: orgClient.getSettings });
export const useOrgMembers   = () => useQuery({ queryKey: keys.members(),  queryFn: orgClient.getMembers });
export const useOrgRoles     = () => useQuery({ queryKey: keys.roles(),    queryFn: orgClient.getRoles });
export const useApiKeys      = () => useQuery({ queryKey: keys.keys(),     queryFn: orgClient.getKeys });
export const useRateLimits   = () => useQuery({ queryKey: keys.rateLimits(), queryFn: orgClient.getRateLimits });
export const useWebhooks     = () => useQuery({ queryKey: keys.webhooks(), queryFn: orgClient.getWebhooks });

export function useUpdateOrgSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: Partial<OrgSettings>) => orgClient.updateSettings(payload),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.settings() }),
  });
}

/**
 * IMPORTANT: raw_key chỉ có trong mutate result — lưu trong local state ngay lập tức.
 * Sau khi component unmount, raw_key không thể recover.
 */
export function useCreateApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: orgClient.createKey,
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.keys() }),
  });
}

export function useRevokeApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => orgClient.revokeKey(id),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.keys() }),
  });
}

export function useCreateWebhook() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: orgClient.createWebhook,
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.webhooks() }),
  });
}

export function useDeleteWebhook() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => orgClient.deleteWebhook(id),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.webhooks() }),
  });
}
```

### 3. Implement API Key "Show Once" Modal

```typescript
// ui/src/components/sdk/CreateKeyModal.tsx
import { useState } from 'react';
import { useCreateApiKey } from '../../api/hooks/useOrg';

export function CreateKeyModal({ onClose }: { onClose: () => void }) {
  const [rawKey, setRawKey] = useState<string | null>(null);
  const { mutate, isPending } = useCreateApiKey();

  const handleCreate = (form: { name: string; permissions: string[] }) => {
    mutate(form, {
      onSuccess: ({ raw_key }) => {
        setRawKey(raw_key);
      },
    });
  };

  if (rawKey) {
    return (
      <div className="p-4 border border-yellow-400 rounded bg-yellow-50">
        <h3 className="font-bold text-yellow-800">⚠️ Copy your API key now</h3>
        <p className="text-sm text-yellow-700 mb-2">This key will not be shown again.</p>
        <code className="block p-2 bg-gray-100 rounded text-sm font-mono break-all">
          {rawKey}
        </code>
        <button
          className="mt-3 px-4 py-2 bg-blue-600 text-white rounded"
          onClick={() => {
            navigator.clipboard.writeText(rawKey);
            onClose();
          }}
        >
          Copy & Close
        </button>
      </div>
    );
  }

  return (
    <CreateKeyForm onSubmit={handleCreate} isPending={isPending} />
  );
}
```

---

## Files tạo ra

```
ui/src/
├── api/
│   ├── clients/org.client.ts       ← NEW
│   └── hooks/useOrg.ts             ← NEW
└── components/sdk/
    └── CreateKeyModal.tsx          ← NEW
```

---

## Acceptance Criteria

- [x] `GET /v1/console/org/settings` → `OrgSettings` với 5 fields
- [x] `PUT /v1/console/org/settings` → cập nhật settings, invalidate cache
- [x] `GET /v1/console/sdk/keys` → `APIKey[]` (không bao giờ có `raw_key`)
- [x] `POST /v1/console/sdk/keys` → response có `raw_key` (chỉ lần đầu)
- [x] Modal hiển thị `raw_key` với nút "Copy & Close"
- [x] `DELETE /v1/console/sdk/keys/{id}` → revoke, invalidate cache
- [x] `GET /v1/console/sdk/webhooks` → `Webhook[]` với `success_rate`
- [x] `POST /v1/console/sdk/webhooks` → tạo webhook mới
- [x] `npx tsc --noEmit` không lỗi
