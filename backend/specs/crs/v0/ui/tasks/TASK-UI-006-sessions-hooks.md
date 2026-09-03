# TASK-UI-006 — Refactor `hooks/useSessions.ts`: Xóa mock, thêm pagination

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-006 |
| **Layer** | Frontend — TypeScript |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-004 §4](../solutions/SOL-004-Sessions-Solution.md) |
| **Priority** | 🔴 P0 — Critical |
| **Depends On** | TASK-UI-001 |
| **Estimated** | 1h |

---

## Context

`hooks/useSessions.ts` dùng `sessionMock` hardcode. Cần thêm pagination params, filter theo status, và thêm các hooks còn thiếu (`useSessionTimeline`, `useSessionDiff`, `useWorkingMemory`).

---

## Goal

- Xóa mock imports và `useMock` ternary
- `useSessionList` nhận filter params (status, page, pageSize, search)
- `keepPreviousData` khi đổi page
- Thêm `useLiveSessions` (poll 10s)
- Thêm `useSessionTimeline`, `useSessionDiff`, `useWorkingMemory`, `useUserSummary`

---

## Target Files

| Action | File Path |
|---|---|
| MODIFY | `ui/src/hooks/useSessions.ts` |
| MODIFY | `ui/src/services/session.service.ts` |

---

## Implementation

### File: `ui/src/hooks/useSessions.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import { keepPreviousData } from '@tanstack/react-query';
import { sessionService } from '../services/session.service';

interface SessionFilters {
  status?: 'active' | 'completed' | 'failed';
  page?: number;
  pageSize?: number;
  search?: string;
}

export function useSessionList(filters?: SessionFilters) {
  return useQuery({
    queryKey: ['sessions', 'list', filters],
    queryFn: () => sessionService.getSessions(filters),
    placeholderData: keepPreviousData,  // Giữ data cũ khi đổi page
    staleTime: 30_000,
  });
}

export function useLiveSessions() {
  return useQuery({
    queryKey: ['sessions', 'live'],
    queryFn: () => sessionService.getLiveSessions(),
    refetchInterval: 10_000,
    refetchIntervalInBackground: false,
  });
}

export function useSessionDetail(id: string) {
  return useQuery({
    queryKey: ['sessions', id, 'detail'],
    queryFn: () => sessionService.getSessionDetail(id),
    enabled: !!id,
  });
}

export function useSessionTimeline(id: string) {
  return useQuery({
    queryKey: ['sessions', id, 'timeline'],
    queryFn: () => sessionService.getTimeline(id),
    enabled: !!id,
  });
}

export function useSessionDiff(id: string) {
  return useQuery({
    queryKey: ['sessions', id, 'diff'],
    queryFn: () => sessionService.getDiff(id),
    enabled: !!id,
  });
}

export function useWorkingMemory(id: string, isActive = false) {
  return useQuery({
    queryKey: ['sessions', id, 'working-memory'],
    queryFn: () => sessionService.getWorkingMemory(id),
    enabled: !!id,
    refetchInterval: isActive ? 5_000 : false,  // Poll chỉ khi session đang active
  });
}

export function useUserSummary(id: string) {
  return useQuery({
    queryKey: ['sessions', id, 'user-summary'],
    queryFn: () => sessionService.getUserSummary(id),
    enabled: !!id,
  });
}
```

### File: `ui/src/services/session.service.ts`

```typescript
import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type { Session, Conversation, WorkingMemory } from '../types/session';

const BASE = API_CONFIG.sessions;

interface SessionFilters {
  status?: string;
  page?: number;
  pageSize?: number;
  search?: string;
}

export const sessionService = {
  getSessions: (filters?: SessionFilters) => {
    const qs = new URLSearchParams();
    if (filters?.status)   qs.set('status', filters.status);
    if (filters?.page)     qs.set('page', String(filters.page));
    if (filters?.pageSize) qs.set('page_size', String(filters.pageSize));
    if (filters?.search)   qs.set('search', filters.search);
    return apiClient.get<{ data: Session[]; total: number; page: number; has_more: boolean }>(
      `${BASE}?${qs.toString()}`
    );
  },

  getLiveSessions: () =>
    apiClient.get<Session[]>(`${BASE}/live`),

  getSessionDetail: (id: string) =>
    apiClient.get<Conversation>(`${BASE}/${id}`),

  getTimeline: (id: string) =>
    apiClient.get<unknown[]>(`${BASE}/${id}/timeline`),

  getDiff: (id: string) =>
    apiClient.get<unknown>(`${BASE}/${id}/diff`),

  getWorkingMemory: (id: string) =>
    apiClient.get<WorkingMemory>(`${BASE}/${id}/working-memory`),

  getUserSummary: (id: string) =>
    apiClient.get<{ user_id: string; context_string: string; token_count: number }>(
      `${BASE}/${id}/user-summary`
    ),
};
```

---

## Verification

```bash
cd ui
npx tsc --noEmit
grep -r "sessionMock" ui/src/hooks/ ui/src/components/ # phải trống
```
