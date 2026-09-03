# TASK-API-006 — Memory API Client + Hooks

**Task ID:** TASK-API-006
**Status:** ✅ COMPLETED — 2026-06-17
**Sprint:** 2 — P0 Modules
**Solution:** [API-SOL-005](../API-SOL-005-memory.md)
**Depends on:** TASK-API-001, TASK-API-002
**Ước tính:** 2h
**Priority:** P0 — Critical

---

## Mục tiêu

Thay thế `memoryMock` bằng cross-engine search API thực:
1. `memory.client.ts` — 4 endpoints, quan trọng: `encodeURIComponent(id)` cho ID dạng `engine:local_id`
2. `useMemory.ts` — hooks với URL filter state pattern

---

## Công việc cụ thể

### 1. Tạo `ui/src/api/clients/memory.client.ts`

```typescript
import { httpClient } from './http.client';
import type {
  MemorySearchQuery, MemorySearchResult, MemoryItem, MemoryVersion
} from '../../types/memory';

const BASE = '/v1/console/memory';

export const memoryClient = {
  search: async (query: MemorySearchQuery): Promise<MemorySearchResult> => {
    const { data } = await httpClient.post<MemorySearchResult>(`${BASE}/search`, query);
    return data;
  },

  /**
   * QUAN TRỌNG: ID có format "engine:local_id" → phải encodeURIComponent
   * Ví dụ: "graphiti:ep_abc" → "graphiti%3Aep_abc"
   */
  getById: async (id: string): Promise<MemoryItem> => {
    const { data } = await httpClient.get<MemoryItem>(
      `${BASE}/${encodeURIComponent(id)}`,
    );
    return data;
  },

  getNeighbors: async (
    id: string,
    strategy: 'semantic' | 'graph' | 'temporal' = 'semantic',
    limit = 10,
  ): Promise<MemorySearchResult> => {
    const { data } = await httpClient.get<MemorySearchResult>(
      `${BASE}/${encodeURIComponent(id)}/neighbors`,
      { params: { strategy, limit } },
    );
    return data;
  },

  /** Chỉ Supermemory items có version chain (id bắt đầu bằng "sm:") */
  getVersions: async (id: string): Promise<MemoryVersion[]> => {
    const { data } = await httpClient.get<MemoryVersion[]>(
      `${BASE}/${encodeURIComponent(id)}/versions`,
    );
    return data;
  },
};
```

### 2. Tạo `ui/src/api/hooks/useMemory.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import { memoryClient } from '../clients/memory.client';
import { ALL_ENGINES, type MemorySearchQuery } from '../../types/memory';

export const memoryKeys = {
  all:       () => ['memories'] as const,
  search:    (q: Partial<MemorySearchQuery>) => [...memoryKeys.all(), 'search', q] as const,
  detail:    (id: string) => [...memoryKeys.all(), 'detail', id] as const,
  neighbors: (id: string, s: string) => [...memoryKeys.all(), 'neighbors', id, s] as const,
  versions:  (id: string) => [...memoryKeys.all(), 'versions', id] as const,
};

const DEFAULT_QUERY: MemorySearchQuery = {
  query:     '',
  mode:      'hybrid',
  engines:   [...ALL_ENGINES],
  filters:   {},
  limit:     20,
  offset:    0,
  reranking: 'cross_encoder',
};

/** Chỉ search khi có query text — avoid empty request */
export function useMemorySearch(partial: Partial<MemorySearchQuery>) {
  const fullQuery: MemorySearchQuery = { ...DEFAULT_QUERY, ...partial };

  return useQuery({
    queryKey:  memoryKeys.search(partial),
    queryFn:   () => memoryClient.search(fullQuery),
    enabled:   !!partial.query && partial.query.trim().length > 0,
    staleTime: 2 * 60_000,
  });
}

/** Cache 5 phút — memory items ít thay đổi */
export function useMemoryDetail(id: string) {
  return useQuery({
    queryKey:  memoryKeys.detail(id),
    queryFn:   () => memoryClient.getById(id),
    enabled:   !!id,
    staleTime: 5 * 60_000,
  });
}

export function useMemoryNeighbors(
  id: string,
  strategy: 'semantic' | 'graph' | 'temporal' = 'semantic',
) {
  return useQuery({
    queryKey: memoryKeys.neighbors(id, strategy),
    queryFn:  () => memoryClient.getNeighbors(id, strategy),
    enabled:  !!id,
  });
}

/** Chỉ enable với Supermemory items */
export function useMemoryVersions(id: string) {
  const isSupermemory = id.startsWith('sm:') || id.startsWith('supermemory:');
  return useQuery({
    queryKey: memoryKeys.versions(id),
    queryFn:  () => memoryClient.getVersions(id),
    enabled:  !!id && isSupermemory,
  });
}
```

### 3. Implement URL filter state cho Memory Explorer

Trong `ui/src/pages/memory-explorer/MemoryExplorer.tsx` hoặc tương đương:

```typescript
import { useSearchParams } from 'react-router-dom';
import { useMemorySearch } from '../../api/hooks/useMemory';
import type { SearchMode } from '../../types/memory';

export function MemoryExplorer() {
  const [searchParams, setSearchParams] = useSearchParams();

  const query   = searchParams.get('q') ?? '';
  const mode    = (searchParams.get('mode') ?? 'hybrid') as SearchMode;
  const engines = searchParams.getAll('engine');

  const { data, isLoading, isError } = useMemorySearch({
    query,
    mode,
    engines: engines.length > 0 ? engines : [...ALL_ENGINES],
  });

  const handleSearch = (newQuery: string) => {
    setSearchParams(prev => { prev.set('q', newQuery); return prev; });
  };

  // ... render với URL state
}
```

---

## Files tạo ra / chỉnh sửa

```
ui/src/
├── api/
│   ├── clients/
│   │   └── memory.client.ts       ← NEW
│   └── hooks/
│       └── useMemory.ts           ← NEW
└── pages/
    └── memory-explorer/
        └── MemoryExplorer.tsx     ← MODIFY (add URL state)
```

---

## Acceptance Criteria

- [x] `POST /v1/console/memory/search` với body JSON → `MemorySearchResult`
- [x] `GET /v1/console/memory/graphiti%3Aep_abc123` → `MemoryItem` (URL encoded)
- [x] `GET /v1/console/memory/graphiti%3Aep_abc/neighbors?strategy=semantic` → kết quả liên quan
- [x] `GET /v1/console/memory/sm%3Amem_xyz/versions` → version chain
- [x] `useMemoryVersions('graphiti:ep_abc')` → disabled (không phải sm:)
- [x] `useMemorySearch({query: ''})` → disabled (không gọi API)
- [x] Search filter state được lưu trong URL query params
- [x] Empty state hiển thị khi search không có kết quả
- [x] Không còn import `memoryMock` trong `ui/src/`

```bash
grep -r "memoryMock" ui/src/  # → 0 results
```
