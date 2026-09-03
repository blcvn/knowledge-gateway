# API-SOL-005 — Memory API Client + Hooks

| Field | Value |
|---|---|
| **Solution ID** | API-SOL-005 |
| **Status** | ✅ IMPLEMENTED — 2026-06-17 |
| **CR** | [CR-004 — Memory Explorer](../../../../specs/crs/v0/ui/CR-004-MEMORY.md) |
| **Kiến trúc ref** | `frontend_architecture.md §6.1` Unified Memory Explorer — `useUnifiedSearch(query, filters)` |
| **Target files** | `ui/src/api/clients/memory.client.ts`, `ui/src/api/hooks/useMemory.ts` |
| **Implemented files** | `ui/src/services/memory.service.ts` · `ui/src/hooks/useMemory.ts` |

---

## API Endpoints

| Method | Endpoint | Mô tả |
|---|---|---|
| `POST` | `/v1/console/memory/search` | Cross-engine search (fan-out) |
| `GET` | `/v1/console/memory/{id}` | Chi tiết theo `engine:local_id` |
| `GET` | `/v1/console/memory/{id}/neighbors` | Semantic/graph neighbors |
| `GET` | `/v1/console/memory/{id}/versions` | Version chain (Supermemory) |

---

## Types

### `ui/src/types/memory.ts`

```typescript
export type SearchMode = 'semantic' | 'bm25' | 'hybrid' | 'graph';
export type RerankingStrategy = 'cross_encoder' | 'rrf' | 'none';

export interface MemorySearchQuery {
  query:     string;
  mode:      SearchMode;
  engines:   string[];
  filters:   MemoryFilters;
  limit:     number;
  offset:    number;
  reranking: RerankingStrategy;
}

export interface MemoryFilters {
  memory_type?: string;
  date_from?:   string;
  date_to?:     string;
  policy_tags?: string[];
}

export interface MemoryItem {
  id:               string;         // format: "engine:local_id"
  engine:           string;
  memoryType:       string;
  title:            string;
  summary:          string;
  content:          string;
  score:            number;
  entities:         string[];
  sourceSessions:   string[];
  temporalValidity: { from: string | null; to: string | null };
  policyTags:       string[];
  versionChain:     string | null;
  metadata:         Record<string, unknown>;
}

export interface MemorySearchResult {
  results:   MemoryItem[];
  total:     number;
  facets:    MemoryFacets;
  latencyMs: number;
}

export interface MemoryFacets {
  byEngine: Record<string, number>;
  byType:   Record<string, number>;
}

export interface MemoryVersion {
  id:             string;
  memory_id:      string;
  content:        string;
  version_number: number;
  is_latest:      boolean;
  diff:           string;
  created_at:     string;
}

// Default search engines (all 6 engines)
export const ALL_ENGINES = [
  'graphiti',
  'cognee',
  'memobase',
  'zep',
  'supermemory',
  'openviking',
] as const;

export type EngineId = typeof ALL_ENGINES[number];
```

---

## Implementation

### `ui/src/api/clients/memory.client.ts`

```typescript
import { httpClient } from './http.client';
import type { MemorySearchQuery, MemorySearchResult, MemoryItem, MemoryVersion, ALL_ENGINES } from '../../types/memory';

const BASE = '/v1/console/memory';

export const memoryClient = {
  /**
   * Cross-engine unified search.
   * ID trong kết quả có format "engine:local_id"
   */
  search: async (query: MemorySearchQuery): Promise<MemorySearchResult> => {
    const { data } = await httpClient.post<MemorySearchResult>(`${BASE}/search`, query);
    return data;
  },

  /**
   * Chi tiết một memory item.
   * @param id - Format: "graphiti:ep_abc123" hoặc "sm:mem_xyz"
   * QUAN TRỌNG: encodeURIComponent vì ID chứa ký tự ":"
   */
  getById: async (id: string): Promise<MemoryItem> => {
    const { data } = await httpClient.get<MemoryItem>(
      `${BASE}/${encodeURIComponent(id)}`,
    );
    return data;
  },

  /**
   * Related memories (semantic/graph/temporal neighbors)
   */
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

  /**
   * Version chain — chỉ hỗ trợ Supermemory (id bắt đầu bằng "sm:")
   */
  getVersions: async (id: string): Promise<MemoryVersion[]> => {
    const { data } = await httpClient.get<MemoryVersion[]>(
      `${BASE}/${encodeURIComponent(id)}/versions`,
    );
    return data;
  },
};
```

### `ui/src/api/hooks/useMemory.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import { memoryClient } from '../clients/memory.client';
import { ALL_ENGINES, type MemorySearchQuery, type MemoryFilters } from '../../types/memory';

export const memoryKeys = {
  all:       () => ['memories'] as const,
  search:    (q: Partial<MemorySearchQuery>) => [...memoryKeys.all(), 'search', q] as const,
  detail:    (id: string) => [...memoryKeys.all(), 'detail', id] as const,
  neighbors: (id: string, s: string) => [...memoryKeys.all(), 'neighbors', id, s] as const,
  versions:  (id: string) => [...memoryKeys.all(), 'versions', id] as const,
};

// Default query shape
const DEFAULT_QUERY: MemorySearchQuery = {
  query:     '',
  mode:      'hybrid',
  engines:   [...ALL_ENGINES],
  filters:   {},
  limit:     20,
  offset:    0,
  reranking: 'cross_encoder',
};

/**
 * Unified Memory Search.
 * - Chỉ kích hoạt khi có query text (enabled: !!query.query)
 * - Filter URL search state theo kiến trúc §6.1
 */
export function useMemorySearch(partial: Partial<MemorySearchQuery>) {
  const fullQuery: MemorySearchQuery = { ...DEFAULT_QUERY, ...partial };

  return useQuery({
    queryKey:  memoryKeys.search(partial),
    queryFn:   () => memoryClient.search(fullQuery),
    enabled:   !!partial.query && partial.query.trim().length > 0,
    staleTime: 2 * 60_000,
  });
}

/**
 * Memory detail — cache 5 phút (ít thay đổi)
 */
export function useMemoryDetail(id: string) {
  return useQuery({
    queryKey:  memoryKeys.detail(id),
    queryFn:   () => memoryClient.getById(id),
    enabled:   !!id,
    staleTime: 5 * 60_000,
  });
}

/**
 * Semantic/graph/temporal neighbors
 */
export function useMemoryNeighbors(id: string, strategy: 'semantic' | 'graph' | 'temporal' = 'semantic') {
  return useQuery({
    queryKey: memoryKeys.neighbors(id, strategy),
    queryFn:  () => memoryClient.getNeighbors(id, strategy),
    enabled:  !!id,
  });
}

/**
 * Version history — chỉ enable cho Supermemory items (id.startsWith('sm:'))
 */
export function useMemoryVersions(id: string) {
  const isSupermemory = id.startsWith('sm:') || id.startsWith('supermemory:');
  return useQuery({
    queryKey: memoryKeys.versions(id),
    queryFn:  () => memoryClient.getVersions(id),
    enabled:  !!id && isSupermemory,
  });
}
```

---

## Filter State trong URL (theo kiến trúc §6.1)

```typescript
// ui/src/pages/memory-explorer/MemoryExplorer.tsx
import { useSearchParams } from 'react-router-dom';
import { useMemorySearch } from '../../api/hooks/useMemory';

function MemoryExplorer() {
  const [searchParams, setSearchParams] = useSearchParams();

  const query = searchParams.get('q') ?? '';
  const mode  = (searchParams.get('mode') ?? 'hybrid') as SearchMode;

  const { data, isLoading, isError } = useMemorySearch({
    query,
    mode,
    engines: searchParams.getAll('engine').length > 0
      ? searchParams.getAll('engine')
      : [...ALL_ENGINES],
  });

  // URL sync giúp share link và back/forward hoạt động đúng
  const updateFilter = (key: string, value: string) => {
    setSearchParams(prev => { prev.set(key, value); return prev; });
  };

  // ...
}
```

---

## Verification

```bash
cd ui && npx tsc --noEmit
grep -r "memoryMock" ui/src/  # phải trống

# Test URL search params:
# /memory-explorer?q=graphiti+temporal&mode=hybrid&engine=graphiti&engine=memobase
```
