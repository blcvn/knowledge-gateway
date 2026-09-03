# API-SOL-007 — Profiles API Client + Hooks

| Field | Value |
|---|---|
| **Solution ID** | API-SOL-007 |
| **Status** | ✅ IMPLEMENTED — 2026-06-17 |
| **CR** | [CR-006 — User Profiles](../../../../specs/crs/v0/ui/CR-006-PROFILES.md) |
| **Target files** | `ui/src/api/clients/profiles.client.ts`, `ui/src/api/hooks/useProfiles.ts` |
| **Implemented files** | `ui/src/services/profile.service.ts` · `ui/src/hooks/useProfiles.ts` |

---

## Types

### `ui/src/types/profile.ts`

```typescript
export interface ProfileEntry {
  topic:      string;
  sub_topic:  string;
  content:    string;
  updated_at: string;
}

export interface UserProfile {
  user_id:    string;
  profiles:   ProfileEntry[];
  created_at: string;
  updated_at: string;
}

export interface BufferZone {
  user_id:     string;
  pending:     string[];           // raw strings pending extraction
  token_count: number;
  threshold:   number;
  flush_pct:   number;             // 0-100
}

export interface ContextAssembly {
  user_id:                string;
  context_string:         string;
  token_count:            number;
  profile_section_tokens: number;
  event_section_tokens:   number;
  latency_ms:             number;
}

export interface UserEvent {
  id:         string;
  type:       string;
  content:    string;
  timestamp:  string;
  session_id?: string;
}

export interface ProfileConfig {
  flush_threshold:   number;
  buffer_token_limit: number;
  ttl_days:          number;
}
```

---

## Implementation

### `ui/src/api/clients/profiles.client.ts`

```typescript
import { httpClient } from './http.client';
import type { UserProfile, BufferZone, ContextAssembly, UserEvent, ProfileConfig } from '../../types/profile';

const BASE = '/v1/console/profiles';

export const profilesClient = {
  listProfiles: async (): Promise<UserProfile[]> => {
    const { data } = await httpClient.get<UserProfile[]>(BASE);
    return data;
  },

  getProfile: async (userId: string): Promise<UserProfile> => {
    const { data } = await httpClient.get<UserProfile>(`${BASE}/${userId}`);
    return data;
  },

  getBuffers: async (userId: string): Promise<BufferZone> => {
    const { data } = await httpClient.get<BufferZone>(`${BASE}/${userId}/buffers`);
    return data;
  },

  getContext: async (userId: string): Promise<ContextAssembly> => {
    const { data } = await httpClient.get<ContextAssembly>(`${BASE}/${userId}/context`);
    return data;
  },

  getEvents: async (userId: string): Promise<UserEvent[]> => {
    const { data } = await httpClient.get<UserEvent[]>(`${BASE}/${userId}/events`);
    return data;
  },

  getConfig: async (): Promise<ProfileConfig> => {
    const { data } = await httpClient.get<ProfileConfig>(`${BASE}/config`);
    return data;
  },

  updateConfig: async (config: Partial<ProfileConfig>): Promise<ProfileConfig> => {
    const { data } = await httpClient.put<ProfileConfig>(`${BASE}/config`, config);
    return data;
  },
};
```

### `ui/src/api/hooks/useProfiles.ts`

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { profilesClient } from '../clients/profiles.client';
import type { ProfileConfig } from '../../types/profile';

const keys = {
  all:     () => ['profiles'] as const,
  list:    () => [...keys.all(), 'list'] as const,
  detail:  (id: string) => [...keys.all(), id] as const,
  buffers: (id: string) => [...keys.all(), id, 'buffers'] as const,
  context: (id: string) => [...keys.all(), id, 'context'] as const,
  events:  (id: string) => [...keys.all(), id, 'events'] as const,
  config:  () => [...keys.all(), 'config'] as const,
};

export const useProfileList = () => useQuery({
  queryKey: keys.list(),
  queryFn:  () => profilesClient.listProfiles(),
});

export const useProfileDetail = (userId: string) => useQuery({
  queryKey: keys.detail(userId),
  queryFn:  () => profilesClient.getProfile(userId),
  enabled:  !!userId,
});

/** Buffer status — poll 30s khi đang xem */
export const useBufferStatus = (userId: string) => useQuery({
  queryKey:        keys.buffers(userId),
  queryFn:         () => profilesClient.getBuffers(userId),
  enabled:         !!userId,
  refetchInterval: 30_000,
});

/** Context assembly — hiển thị assembled prompt context */
export const useContextAssembly = (userId: string) => useQuery({
  queryKey: keys.context(userId),
  queryFn:  () => profilesClient.getContext(userId),
  enabled:  !!userId,
});

export const useUserEvents = (userId: string) => useQuery({
  queryKey: keys.events(userId),
  queryFn:  () => profilesClient.getEvents(userId),
  enabled:  !!userId,
});

export const useProfileConfig = () => useQuery({
  queryKey: keys.config(),
  queryFn:  () => profilesClient.getConfig(),
});

export function useUpdateProfileConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (config: Partial<ProfileConfig>) => profilesClient.updateConfig(config),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.config() }),
  });
}
```
