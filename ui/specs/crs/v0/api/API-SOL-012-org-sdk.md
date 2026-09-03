# API-SOL-012 — Org & SDK API Client + Hooks

| Field | Value |
|---|---|
| **Solution ID** | API-SOL-012 |
| **Status** | ✅ IMPLEMENTED — 2026-06-17 |
| **CR** | [CR-011 — Org Settings & API SDK](../../../../specs/crs/v0/ui/CR-011-ORG-SDK.md) |
| **Target files** | `ui/src/api/clients/org.client.ts`, `ui/src/api/hooks/useOrg.ts` |
| **Implemented files** | `ui/src/services/org.service.ts` · `ui/src/hooks/useOrganizationSettings.ts` · `ui/src/hooks/useApiSdk.ts` (re-export) · `ui/src/types/org.ts` |

---

## Types

### `ui/src/types/org.ts`

```typescript
export interface OrgSettings {
  name:                  string;
  slug:                  string;
  timezone:              string;
  max_agents:            number;
  max_memories_per_user: number;
}

export interface OrgMember {
  id:         string;
  name:       string;
  email:      string;
  role:       'owner' | 'admin' | 'editor' | 'viewer';
  avatar_url?: string;
  joined_at:  string;
}

export interface OrgRole {
  id:          string;
  name:        string;
  permissions: string[];
}

export interface APIKey {
  id:           string;
  name:         string;
  prefix:       string;           // "vnp_prod"
  masked_key:   string;           // "vnp_prod...xxxx"
  permissions:  string[];
  expires_at?:  string;
  created_at:   string;
  last_used_at?: string;
}

export interface CreateKeyResponse {
  key:     APIKey;
  raw_key: string;   // Shown ONCE — user must copy
}

export interface RateLimitConfig {
  scope:     'global' | 'per_key' | 'per_endpoint';
  rps:       number;
  rpm:       number;
  burst:     number;
  tier_name: 'free' | 'pro' | 'enterprise';
}

export interface Webhook {
  id:           string;
  url:          string;
  events:       string[];
  status:       'active' | 'disabled';
  success_rate: number;
  created_at:   string;
}
```

---

## Implementation

### `ui/src/api/clients/org.client.ts`

```typescript
import { httpClient } from './http.client';
import type {
  OrgSettings, OrgMember, OrgRole,
  APIKey, CreateKeyResponse, RateLimitConfig,
  Webhook,
} from '../../types/org';

const ORG_BASE = '/v1/console/org';
const SDK_BASE = '/v1/console/sdk';

export const orgClient = {
  // ── Org Settings ──────────────────────────────────────────────────────────
  getSettings: async (): Promise<OrgSettings> => {
    const { data } = await httpClient.get<OrgSettings>(`${ORG_BASE}/settings`);
    return data;
  },
  updateSettings: async (payload: Partial<OrgSettings>): Promise<OrgSettings> => {
    const { data } = await httpClient.put<OrgSettings>(`${ORG_BASE}/settings`, payload);
    return data;
  },

  // ── Members ───────────────────────────────────────────────────────────────
  getMembers: async (): Promise<OrgMember[]> => {
    const { data } = await httpClient.get<OrgMember[]>(`${ORG_BASE}/members`);
    return data;
  },
  getRoles: async (): Promise<OrgRole[]> => {
    const { data } = await httpClient.get<OrgRole[]>(`${ORG_BASE}/roles`);
    return data;
  },

  // ── API Keys ──────────────────────────────────────────────────────────────
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
    // IMPORTANT: raw_key chỉ trả về 1 lần — UI cần show modal "Copy now"
  },
  revokeKey: async (id: string): Promise<void> => {
    await httpClient.delete(`${SDK_BASE}/keys/${id}`);
  },

  // ── Rate Limits ───────────────────────────────────────────────────────────
  getRateLimits: async (): Promise<RateLimitConfig[]> => {
    const { data } = await httpClient.get<RateLimitConfig[]>(`${SDK_BASE}/rate-limits`);
    return data;
  },

  // ── Webhooks ──────────────────────────────────────────────────────────────
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

### `ui/src/api/hooks/useOrg.ts`

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

export const useOrgSettings = () => useQuery({
  queryKey: keys.settings(),
  queryFn:  () => orgClient.getSettings(),
});

export const useOrgMembers = () => useQuery({
  queryKey: keys.members(),
  queryFn:  () => orgClient.getMembers(),
});

export const useOrgRoles = () => useQuery({
  queryKey: keys.roles(),
  queryFn:  () => orgClient.getRoles(),
});

export function useUpdateOrgSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: Partial<OrgSettings>) => orgClient.updateSettings(payload),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.settings() }),
  });
}

export const useApiKeys = () => useQuery({
  queryKey: keys.keys(),
  queryFn:  () => orgClient.getKeys(),
});

export function useCreateApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: orgClient.createKey,
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.keys() }),
    // raw_key ở trong mutate result — component cần dùng onSuccess callback để show
  });
}

export function useRevokeApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => orgClient.revokeKey(id),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.keys() }),
  });
}

export const useRateLimits = () => useQuery({
  queryKey: keys.rateLimits(),
  queryFn:  () => orgClient.getRateLimits(),
});

export const useWebhooks = () => useQuery({
  queryKey: keys.webhooks(),
  queryFn:  () => orgClient.getWebhooks(),
});

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

---

## UX Pattern cho API Key Creation

Sau khi `useCreateApiKey` thành công, `raw_key` chỉ có 1 lần:

```tsx
function CreateKeyModal({ onClose }: { onClose: () => void }) {
  const [rawKey, setRawKey] = useState<string | null>(null);
  const { mutate, isPending } = useCreateApiKey();

  const handleCreate = (form: { name: string; permissions: string[] }) => {
    mutate(form, {
      onSuccess: ({ raw_key }) => {
        setRawKey(raw_key);  // Lưu để hiển thị một lần
      },
    });
  };

  if (rawKey) {
    return (
      <Dialog>
        <h3>⚠️ Copy your API key now — it will not be shown again</h3>
        <code>{rawKey}</code>
        <Button onClick={() => { navigator.clipboard.writeText(rawKey); onClose(); }}>
          Copy & Close
        </Button>
      </Dialog>
    );
  }

  return <CreateKeyForm onSubmit={handleCreate} isPending={isPending} />;
}
```
