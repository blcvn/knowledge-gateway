# TASK-UI-010 — Tạo `profile.service.ts` + Refactor `useProfiles.ts`

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-010 |
| **Layer** | Frontend — TypeScript |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-007 §6](../solutions/SOL-007-Gap-Fixes.md) |
| **Priority** | 🔴 P0 |
| **Depends On** | TASK-UI-001 |
| **Estimated** | 1.5h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `ui/src/services/profile.service.ts` |
| MODIFY | `ui/src/hooks/useProfiles.ts` |

---

## Implementation

### File: `ui/src/services/profile.service.ts`

```typescript
import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type { UserProfile, BufferZone, UserEvent } from '../types/profile';

const BASE = API_CONFIG.profiles;

export interface ContextAssembly {
  user_id: string;
  context_string: string;
  token_count: number;
  profile_section_tokens: number;
  event_section_tokens: number;
  latency_ms: number;
}

export interface ProfileConfig {
  flush_threshold: number;
  buffer_token_limit: number;
  ttl_days: number;
}

export const profileService = {
  listProfiles: () =>
    apiClient.get<UserProfile[]>(BASE),

  getProfile: (userId: string) =>
    apiClient.get<UserProfile>(`${BASE}/${userId}`),

  getBuffers: (userId: string) =>
    apiClient.get<BufferZone>(`${BASE}/${userId}/buffers`),

  getContext: (userId: string) =>
    apiClient.get<ContextAssembly>(`${BASE}/${userId}/context`),

  getEvents: (userId: string) =>
    apiClient.get<UserEvent[]>(`${BASE}/${userId}/events`),

  getConfig: () =>
    apiClient.get<ProfileConfig>(`${BASE}/config`),

  updateConfig: (config: Partial<ProfileConfig>) =>
    apiClient.put<ProfileConfig>(`${BASE}/config`, config),
};
```

### File: `ui/src/hooks/useProfiles.ts`

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { profileService } from '../services/profile.service';

export function useProfileList() {
  return useQuery({
    queryKey: ['profiles'],
    queryFn: () => profileService.listProfiles(),
  });
}

export function useProfileDetail(userId: string) {
  return useQuery({
    queryKey: ['profiles', userId],
    queryFn: () => profileService.getProfile(userId),
    enabled: !!userId,
  });
}

export function useBufferStatus(userId: string) {
  return useQuery({
    queryKey: ['profiles', userId, 'buffers'],
    queryFn: () => profileService.getBuffers(userId),
    enabled: !!userId,
    refetchInterval: 30_000,
  });
}

export function useContextAssembly(userId: string) {
  return useQuery({
    queryKey: ['profiles', userId, 'context'],
    queryFn: () => profileService.getContext(userId),
    enabled: !!userId,
  });
}

export function useUserEvents(userId: string) {
  return useQuery({
    queryKey: ['profiles', userId, 'events'],
    queryFn: () => profileService.getEvents(userId),
    enabled: !!userId,
  });
}

export function useProfileConfig() {
  return useQuery({
    queryKey: ['profiles', 'config'],
    queryFn: () => profileService.getConfig(),
  });
}

export function useUpdateProfileConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: profileService.updateConfig,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['profiles', 'config'] }),
  });
}
```

---

## Verification

```bash
cd ui
npx tsc --noEmit
grep -r "profileMock" ui/src/hooks/ ui/src/components/ # phải trống
```
