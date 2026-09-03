# TASK-API-005 — Sessions API Client + Hooks

**Task ID:** TASK-API-005
**Status:** ✅ COMPLETED — 2026-06-17
**Sprint:** 2 — P0 Modules
**Solution:** [API-SOL-004](../API-SOL-004-sessions.md)
**Depends on:** TASK-API-001, TASK-API-002
**Ước tính:** 2h
**Priority:** P0 — Critical

---

## Mục tiêu

Thay thế `sessionMock` bằng API calls thực:
1. `sessions.client.ts` — 7 endpoints
2. `useSessions.ts` — hooks với đúng polling và `keepPreviousData` cho pagination

---

## Công việc cụ thể

### 1. Tạo `ui/src/api/clients/sessions.client.ts`

```typescript
import { httpClient } from './http.client';
import type {
  Session, Conversation, PaginatedResponse, SessionTimeline,
  SessionDiff, WorkingMemory, UserSummary, SessionFilters
} from '../../types/session';

const BASE = '/v1/console/sessions';

export const sessionsClient = {
  getSessions: async (filters: SessionFilters = {}): Promise<PaginatedResponse<Session>> => {
    const { data } = await httpClient.get<PaginatedResponse<Session>>(BASE, {
      params: filters,
    });
    return data;
  },

  getLiveSessions: async (): Promise<Session[]> => {
    const { data } = await httpClient.get<Session[]>(`${BASE}/live`);
    return data;
  },

  getDetail: async (id: string): Promise<Conversation> => {
    const { data } = await httpClient.get<Conversation>(`${BASE}/${id}`);
    return data;
  },

  getTimeline: async (id: string): Promise<SessionTimeline[]> => {
    const { data } = await httpClient.get<SessionTimeline[]>(`${BASE}/${id}/timeline`);
    return data;
  },

  getDiff: async (id: string): Promise<SessionDiff> => {
    const { data } = await httpClient.get<SessionDiff>(`${BASE}/${id}/diff`);
    return data;
  },

  getWorkingMemory: async (id: string): Promise<WorkingMemory> => {
    const { data } = await httpClient.get<WorkingMemory>(`${BASE}/${id}/working-memory`);
    return data;
  },

  getUserSummary: async (id: string): Promise<UserSummary> => {
    const { data } = await httpClient.get<UserSummary>(`${BASE}/${id}/user-summary`);
    return data;
  },
};
```

### 2. Tạo `ui/src/api/hooks/useSessions.ts`

```typescript
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { sessionsClient } from '../clients/sessions.client';
import type { SessionFilters } from '../../types/session';

export const sessionKeys = {
  all:         () => ['sessions'] as const,
  list:        (f: SessionFilters) => [...sessionKeys.all(), 'list', f] as const,
  live:        () => [...sessionKeys.all(), 'live'] as const,
  detail:      (id: string) => [...sessionKeys.all(), id, 'detail'] as const,
  timeline:    (id: string) => [...sessionKeys.all(), id, 'timeline'] as const,
  diff:        (id: string) => [...sessionKeys.all(), id, 'diff'] as const,
  workingMem:  (id: string) => [...sessionKeys.all(), id, 'working-memory'] as const,
  userSummary: (id: string) => [...sessionKeys.all(), id, 'user-summary'] as const,
};

/** keepPreviousData = tránh flash khi đổi trang */
export function useSessionList(filters: SessionFilters = {}) {
  return useQuery({
    queryKey:        sessionKeys.list(filters),
    queryFn:         () => sessionsClient.getSessions(filters),
    placeholderData: keepPreviousData,
    staleTime:       30_000,
  });
}

/** Poll 10s — live active sessions */
export function useLiveSessions() {
  return useQuery({
    queryKey:                sessionKeys.live(),
    queryFn:                 () => sessionsClient.getLiveSessions(),
    refetchInterval:         10_000,
    refetchIntervalInBackground: false,
  });
}

export function useSessionDetail(id: string) {
  return useQuery({
    queryKey:  sessionKeys.detail(id),
    queryFn:   () => sessionsClient.getDetail(id),
    enabled:   !!id,
    staleTime: 60_000,
  });
}

export function useSessionTimeline(id: string) {
  return useQuery({
    queryKey: sessionKeys.timeline(id),
    queryFn:  () => sessionsClient.getTimeline(id),
    enabled:  !!id,
  });
}

export function useSessionDiff(id: string) {
  return useQuery({
    queryKey: sessionKeys.diff(id),
    queryFn:  () => sessionsClient.getDiff(id),
    enabled:  !!id,
  });
}

/** isActive=true → poll 5s (session đang chạy) */
export function useWorkingMemory(id: string, isActive = false) {
  return useQuery({
    queryKey:        sessionKeys.workingMem(id),
    queryFn:         () => sessionsClient.getWorkingMemory(id),
    enabled:         !!id,
    refetchInterval: isActive ? 5_000 : false,
  });
}

export function useUserSummary(id: string) {
  return useQuery({
    queryKey: sessionKeys.userSummary(id),
    queryFn:  () => sessionsClient.getUserSummary(id),
    enabled:  !!id,
  });
}
```

### 3. Tìm và cập nhật imports cũ

```bash
# Tìm tất cả files dùng sessionMock hoặc session.service.ts cũ
grep -r "sessionMock\|session\.service\|useMock" ui/src --include="*.ts" --include="*.tsx" -l

# Cập nhật import paths:
# from '../hooks/useSessions' → from '../api/hooks/useSessions'
# from '../services/session.service' → (xóa, dùng hook)
```

---

## Files tạo ra / chỉnh sửa

```
ui/src/
├── api/
│   ├── clients/
│   │   └── sessions.client.ts  ← NEW
│   └── hooks/
│       └── useSessions.ts      ← NEW
└── hooks/
    └── useSessions.ts          ← MODIFY (xóa mock, re-export hoặc xóa)
```

---

## Acceptance Criteria

- [x] `GET /v1/console/sessions?page=1&page_size=20` → `PaginatedResponse<Session>`
- [x] `GET /v1/console/sessions?status=active` → filter đúng
- [x] `GET /v1/console/sessions/live` → chỉ active sessions
- [x] `GET /v1/console/sessions/{id}` → `Conversation` với messages array
- [x] `GET /v1/console/sessions/{id}/working-memory` → `WorkingMemory`
- [x] `GET /v1/console/sessions/{id}/timeline` → `SessionTimeline[]`
- [x] `useSessionList()` dùng `keepPreviousData` (không flash khi đổi trang)
- [x] `useWorkingMemory(id, true)` poll 5s khi session active
- [x] Không còn import `sessionMock` trong `ui/src/`
- [x] Zep 503 được handle gracefully (hiển thị "Working memory unavailable")

```bash
grep -r "sessionMock" ui/src/  # → 0 results
```
