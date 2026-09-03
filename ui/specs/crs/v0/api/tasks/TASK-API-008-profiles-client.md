# TASK-API-008 — Profiles API Client + Hooks

**Task ID:** TASK-API-008
**Status:** ✅ COMPLETED — 2026-06-17
**Sprint:** 2 — P0 Modules
**Solution:** [API-SOL-007](../API-SOL-007-profiles.md)
**Depends on:** TASK-API-001, TASK-API-002
**Ước tính:** 1.5h
**Priority:** P0 — Critical

---

## Mục tiêu

Implement Memobase User Profiles Data Access Layer:
1. `profiles.client.ts` — 7 endpoints: list, detail, buffers, context, events, config
2. `useProfiles.ts` — hooks với polling cho buffer status

---

## Công việc cụ thể

### 1. Tạo `ui/src/api/clients/profiles.client.ts`

```typescript
import { httpClient } from './http.client';
import type {
  UserProfile, BufferZone, ContextAssembly, UserEvent, ProfileConfig
} from '../../types/profile';

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

### 2. Tạo `ui/src/api/hooks/useProfiles.ts`

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

/** Poll 30s — buffer fill status thay đổi theo ingestion */
export const useBufferStatus = (userId: string) => useQuery({
  queryKey:        keys.buffers(userId),
  queryFn:         () => profilesClient.getBuffers(userId),
  enabled:         !!userId,
  refetchInterval: 30_000,
});

/** Context assembly — latency_ms phản ánh thời gian thực */
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

---

## Files tạo ra

```
ui/src/api/
├── clients/profiles.client.ts  ← NEW
└── hooks/useProfiles.ts        ← NEW
```

---

## Acceptance Criteria

- [x] `GET /v1/console/profiles` → `UserProfile[]`
- [x] `GET /v1/console/profiles/{user_id}` → single profile với `profiles[]` array
- [x] `GET /v1/console/profiles/{user_id}/buffers` → `BufferZone` với `flush_pct`
- [x] `GET /v1/console/profiles/{user_id}/context` → `ContextAssembly` với `latency_ms`
- [x] `GET /v1/console/profiles/{user_id}/events` → `UserEvent[]`
- [x] `GET /v1/console/profiles/config` → `ProfileConfig`
- [x] `PUT /v1/console/profiles/config` → cập nhật và invalidate config cache
- [x] `useBufferStatus` poll 30s
- [x] `npx tsc --noEmit` không lỗi
