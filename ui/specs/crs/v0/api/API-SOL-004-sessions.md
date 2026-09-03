# API-SOL-004 — Sessions API Client + Hooks

| Field | Value |
|---|---|
| **Solution ID** | API-SOL-004 |
| **Status** | ✅ IMPLEMENTED — 2026-06-17 |
| **CR** | [CR-003 — Sessions](../../../../specs/crs/v0/ui/CR-003-SESSIONS.md) |
| **Kiến trúc ref** | `frontend_architecture.md §3.2` TanStack Query `keepPreviousData` cho pagination |
| **Target files** | `ui/src/api/clients/sessions.client.ts`, `ui/src/api/hooks/useSessions.ts` |
| **Implemented files** | `ui/src/services/session.service.ts` · `ui/src/hooks/useSessions.ts` |

---

## API Endpoints

| Method | Endpoint | Mô tả |
|---|---|---|
| `GET` | `/v1/console/sessions` | Danh sách có pagination + filter |
| `GET` | `/v1/console/sessions/live` | Active sessions (poll 10s) |
| `GET` | `/v1/console/sessions/{id}` | Detail + messages |
| `GET` | `/v1/console/sessions/{id}/timeline` | Memory operation events |
| `GET` | `/v1/console/sessions/{id}/diff` | Memory diff sau session |
| `GET` | `/v1/console/sessions/{id}/working-memory` | Zep working memory state |
| `GET` | `/v1/console/sessions/{id}/user-summary` | Memobase context summary |

---

## Types

### `ui/src/types/session.ts`

```typescript
export interface Session {
  id:            string;
  user_id:       string;
  title:         string;
  agent_id?:     string;
  status:        'active' | 'completed' | 'failed';
  message_count: number;
  created_at:    string;
  updated_at:    string;
}

export interface Message {
  id:             string;
  role:           'user' | 'assistant' | 'system' | 'tool';
  content:        string;
  timestamp:      string;
  memory_sources?: string[]; // e.g. ["graphiti:ep_abc", "memobase:prof_xyz"]
}

export interface Conversation {
  session_id: string;
  messages:   Message[];
}

export interface PaginatedResponse<T> {
  data:      T[];
  total:     number;
  page:      number;
  page_size: number;
  has_more:  boolean;
}

export interface SessionTimeline {
  event_type: string;
  engine:     string;
  memory_id:  string;
  timestamp:  string;
  latency_ms: number;
  details:    Record<string, unknown>;
}

export interface SessionDiff {
  session_id: string;
  added:   Array<{ engine: string; memory_id: string; content: string }>;
  updated: Array<{ engine: string; memory_id: string; field: string; before: unknown; after: unknown }>;
  deleted: Array<{ engine: string; memory_id: string }>;
}

export interface WorkingMemory {
  session_id: string;
  summary:    string;
  entities:   string[];
}

export interface UserSummary {
  user_id:        string;
  context_string: string;
  token_count:    number;
}

export interface SessionFilters {
  status?:    'active' | 'completed' | 'failed';
  user_id?:   string;
  agent_id?:  string;
  search?:    string;
  sort?:      string;
  page?:      number;
  page_size?: number;
}
```

---

## Implementation

### `ui/src/api/clients/sessions.client.ts`

```typescript
import { httpClient } from './http.client';
import type { Session, Conversation, PaginatedResponse, SessionTimeline, SessionDiff, WorkingMemory, UserSummary, SessionFilters } from '../../types/session';

const BASE = '/v1/console/sessions';

export const sessionsClient = {
  getSessions: async (filters: SessionFilters = {}): Promise<PaginatedResponse<Session>> => {
    const { data } = await httpClient.get<PaginatedResponse<Session>>(BASE, {
      params: {
        ...filters,
        // Rename pageSize → page_size để khớp API
        page_size: filters.page_size,
      },
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

### `ui/src/api/hooks/useSessions.ts`

```typescript
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { sessionsClient } from '../clients/sessions.client';
import type { SessionFilters } from '../../types/session';

export const sessionKeys = {
  all:          () => ['sessions'] as const,
  list:         (f: SessionFilters) => [...sessionKeys.all(), 'list', f] as const,
  live:         () => [...sessionKeys.all(), 'live'] as const,
  detail:       (id: string) => [...sessionKeys.all(), id, 'detail'] as const,
  timeline:     (id: string) => [...sessionKeys.all(), id, 'timeline'] as const,
  diff:         (id: string) => [...sessionKeys.all(), id, 'diff'] as const,
  workingMem:   (id: string) => [...sessionKeys.all(), id, 'working-memory'] as const,
  userSummary:  (id: string) => [...sessionKeys.all(), id, 'user-summary'] as const,
};

/**
 * Danh sách sessions có pagination và filter.
 * Dùng keepPreviousData để tránh flash khi đổi trang.
 */
export function useSessionList(filters: SessionFilters = {}) {
  return useQuery({
    queryKey:         sessionKeys.list(filters),
    queryFn:          () => sessionsClient.getSessions(filters),
    placeholderData:  keepPreviousData,
    staleTime:        30_000,
  });
}

/**
 * Active sessions — poll mỗi 10s (realtime).
 * Tắt background poll để tiết kiệm request khi user rời khỏi tab.
 */
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

/**
 * Working memory của session — poll 5s nếu session đang active.
 * @param isActive - khi true sẽ poll liên tục
 */
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

---

## Verification

```bash
cd ui && npx tsc --noEmit
grep -r "sessionMock" ui/src/   # phải trống
```
