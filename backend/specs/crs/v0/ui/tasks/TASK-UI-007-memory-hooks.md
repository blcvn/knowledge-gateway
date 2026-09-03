# TASK-UI-007 — Refactor `hooks/useMemory.ts` + tạo `memory.service.ts` đầy đủ

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-007 |
| **Layer** | Frontend — TypeScript |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-005 §3](../solutions/SOL-005-Memory-Solution.md) |
| **Priority** | 🔴 P0 — Critical |
| **Depends On** | TASK-UI-001 |
| **Estimated** | 1.5h |

---

## Context

`hooks/useMemory.ts` dùng `memoryMock`. Service đã có nhưng thiếu `getNeighbors`, `getVersions` và `encodeURIComponent` cho ID có dạng `engine:local_id`.

---

## Goal

- Xóa mock imports
- Thêm `useMemoryNeighbors`, `useMemoryVersions`
- Đảm bảo `encodeURIComponent(id)` khi build URL
- `useMemorySearch` chỉ query khi có `query.query` text

---

## Target Files

| Action | File Path |
|---|---|
| MODIFY | `ui/src/hooks/useMemory.ts` |
| MODIFY | `ui/src/services/memory.service.ts` |

---

## Implementation

### File: `ui/src/services/memory.service.ts`

```typescript
import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type { MemorySearchQuery, MemorySearchResult, MemoryItem, MemoryVersion } from '../types/memory';

const BASE = API_CONFIG.memory;

export const memoryService = {
  search: (query: MemorySearchQuery) =>
    apiClient.post<MemorySearchResult>(`${BASE}/search`, query),

  getById: (id: string) =>
    // ID format: "graphiti:ep_abc123" — must be URL-encoded
    apiClient.get<MemoryItem>(`${BASE}/${encodeURIComponent(id)}`),

  getNeighbors: (id: string, strategy = 'semantic', limit = 10) =>
    apiClient.get<MemorySearchResult>(
      `${BASE}/${encodeURIComponent(id)}/neighbors?strategy=${strategy}&limit=${limit}`
    ),

  getVersions: (id: string) =>
    apiClient.get<MemoryVersion[]>(`${BASE}/${encodeURIComponent(id)}/versions`),
};
```

### File: `ui/src/hooks/useMemory.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import { memoryService } from '../services/memory.service';
import type { MemorySearchQuery } from '../types/memory';

const DEFAULT_ENGINES = ['graphiti', 'cognee', 'memobase', 'zep', 'supermemory', 'openviking'];

export function useMemorySearch(query: Partial<MemorySearchQuery>) {
  const fullQuery: MemorySearchQuery = {
    query: '',
    mode: 'hybrid',
    engines: DEFAULT_ENGINES,
    filters: {},
    limit: 20,
    offset: 0,
    reranking: 'cross_encoder',
    ...query,
  };

  return useQuery({
    queryKey: ['memories', 'search', fullQuery],
    queryFn: () => memoryService.search(fullQuery),
    enabled: !!fullQuery.query && fullQuery.query.trim().length > 0,
    staleTime: 2 * 60_000,
  });
}

export function useMemoryDetail(id: string) {
  return useQuery({
    queryKey: ['memories', 'detail', id],
    queryFn: () => memoryService.getById(id),
    enabled: !!id,
    staleTime: 5 * 60_000,
  });
}

export function useMemoryNeighbors(id: string, strategy = 'semantic') {
  return useQuery({
    queryKey: ['memories', 'neighbors', id, strategy],
    queryFn: () => memoryService.getNeighbors(id, strategy),
    enabled: !!id,
  });
}

export function useMemoryVersions(id: string) {
  const isSupermemory = id.startsWith('sm:') || id.startsWith('supermemory:');
  return useQuery({
    queryKey: ['memories', 'versions', id],
    queryFn: () => memoryService.getVersions(id),
    enabled: !!id && isSupermemory,  // Chỉ Supermemory có versioning
  });
}
```

---

## Verification

```bash
cd ui
npx tsc --noEmit
grep -r "memoryMock" ui/src/hooks/ ui/src/components/ # phải trống
```
