# TASK-UI-015 — Tạo `org.service.ts` + `sdk.service.ts` + Refactor hooks

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-015 |
| **Layer** | Frontend — TypeScript |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-007 §11](../solutions/SOL-007-Gap-Fixes.md) |
| **Priority** | 🟡 P2 |
| **Depends On** | TASK-UI-001 |
| **Estimated** | 2h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `ui/src/services/org.service.ts` |
| CREATE | `ui/src/services/sdk.service.ts` |
| MODIFY | `ui/src/hooks/useOrganizationSettings.ts` |
| MODIFY | `ui/src/hooks/useApiSdk.ts` |

---

## Implementation

### File: `ui/src/services/org.service.ts`

```typescript
import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';

const BASE = API_CONFIG.org;

export interface OrgSettings {
  name: string;
  slug: string;
  timezone: string;
  max_agents: number;
  max_memories_per_user: number;
}

export interface OrgMember {
  id: string;
  name: string;
  email: string;
  role: 'owner' | 'admin' | 'editor' | 'viewer';
  avatar_url?: string;
  joined_at: string;
}

export interface OrgRole {
  id: string;
  name: string;
  permissions: string[];
}

export const orgService = {
  getSettings: () =>
    apiClient.get<OrgSettings>(`${BASE}/settings`),

  updateSettings: (data: Partial<OrgSettings>) =>
    apiClient.put<OrgSettings>(`${BASE}/settings`, data),

  getMembers: () =>
    apiClient.get<OrgMember[]>(`${BASE}/members`),

  getRoles: () =>
    apiClient.get<OrgRole[]>(`${BASE}/roles`),
};
```

### File: `ui/src/services/sdk.service.ts`

```typescript
import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';

const BASE = API_CONFIG.sdk;

export interface APIKey {
  id: string;
  name: string;
  prefix: string;         // First 8 chars, e.g. "vnp_prod"
  masked_key: string;     // e.g. "vnp_prod...xxxx"
  permissions: string[];
  expires_at?: string;
  created_at: string;
  last_used_at?: string;
}

export interface RateLimitConfig {
  scope: 'global' | 'per_key' | 'per_endpoint';
  rps: number;
  rpm: number;
  burst: number;
  tier_name: 'free' | 'pro' | 'enterprise';
}

export interface Webhook {
  id: string;
  url: string;
  events: string[];
  status: 'active' | 'disabled';
  success_rate: number;
  created_at: string;
}

export const sdkService = {
  getKeys: () =>
    apiClient.get<APIKey[]>(`${BASE}/keys`),

  createKey: (data: { name: string; permissions: string[]; expires_in_days?: number }) =>
    apiClient.post<{ key: APIKey; raw_key: string }>(`${BASE}/keys`, data),
    // raw_key được trả về 1 lần duy nhất — hiển thị popup cho user copy

  revokeKey: (id: string) =>
    apiClient.delete<void>(`${BASE}/keys/${id}`),

  getRateLimits: () =>
    apiClient.get<RateLimitConfig[]>(`${BASE}/rate-limits`),

  getWebhooks: () =>
    apiClient.get<Webhook[]>(`${BASE}/webhooks`),

  createWebhook: (data: { url: string; events: string[]; secret?: string }) =>
    apiClient.post<Webhook>(`${BASE}/webhooks`, data),

  deleteWebhook: (id: string) =>
    apiClient.delete<void>(`${BASE}/webhooks/${id}`),
};
```

### File: `ui/src/hooks/useOrganizationSettings.ts`

**Xóa tất cả `const mockSettings`, `const mockMembers`, `const mockRoles` và `useMock` ternary:**

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { orgService } from '../services/org.service';

export function useOrgSettings() {
  return useQuery({
    queryKey: ['org', 'settings'],
    queryFn: () => orgService.getSettings(),
  });
}

export function useOrgMembers() {
  return useQuery({
    queryKey: ['org', 'members'],
    queryFn: () => orgService.getMembers(),
  });
}

export function useOrgRoles() {
  return useQuery({
    queryKey: ['org', 'roles'],
    queryFn: () => orgService.getRoles(),
  });
}

export function useUpdateOrgSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: orgService.updateSettings,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['org', 'settings'] }),
  });
}
```

### File: `ui/src/hooks/useApiSdk.ts`

**Xóa tất cả `const mockApiKeys`, `const mockRateLimits`, `const mockWebhooks`:**

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { sdkService } from '../services/sdk.service';

export function useApiKeys() {
  return useQuery({
    queryKey: ['sdk', 'keys'],
    queryFn: () => sdkService.getKeys(),
  });
}

export function useRateLimits() {
  return useQuery({
    queryKey: ['sdk', 'rate-limits'],
    queryFn: () => sdkService.getRateLimits(),
  });
}

export function useWebhooks() {
  return useQuery({
    queryKey: ['sdk', 'webhooks'],
    queryFn: () => sdkService.getWebhooks(),
  });
}

export function useCreateApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: sdkService.createKey,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['sdk', 'keys'] }),
  });
}

export function useRevokeApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: sdkService.revokeKey,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['sdk', 'keys'] }),
  });
}

export function useCreateWebhook() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: sdkService.createWebhook,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['sdk', 'webhooks'] }),
  });
}
```

---

## Verification

```bash
cd ui
npx tsc --noEmit
grep -r "mockApiKeys\|mockRateLimits\|mockWebhooks\|mockSettings\|mockMembers" ui/src/hooks/ # phải trống
```
