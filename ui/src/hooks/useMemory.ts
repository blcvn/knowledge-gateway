/**
 * Memory Hooks — real API, no mock
 * TASK-API-006: query key factory, URL filter state, MemoryVersion support
 */

import { useQuery }      from '@tanstack/react-query';
import { memoryService } from '../services/memory.service';
import { ALL_ENGINES }   from '../types/api';
import type { MemorySearchQuery } from '../types/memory';

// ─── Query Key Factory ────────────────────────────────────────────────────────

export const memoryKeys = {
  all:       () => ['memories'] as const,
  search:    (q: Partial<MemorySearchQuery>) => [...memoryKeys.all(), 'search', q] as const,
  detail:    (id: string) => [...memoryKeys.all(), 'detail', id] as const,
  neighbors: (id: string, s: string) => [...memoryKeys.all(), 'neighbors', id, s] as const,
  versions:  (id: string) => [...memoryKeys.all(), 'versions', id] as const,
};

// ─── Default Query ────────────────────────────────────────────────────────────

const DEFAULT_QUERY: MemorySearchQuery = {
  query:     '',
  mode:      'hybrid',
  engines:   [...ALL_ENGINES],
  filters:   {},
  limit:     20,
  offset:    0,
  reranking: 'cross_encoder',
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

/**
 * Cross-engine memory search
 * API: POST /v1/console/memory/search
 * Only fires when query text is non-empty
 */
export function useMemorySearch(partial: Partial<MemorySearchQuery>) {
  const fullQuery: MemorySearchQuery = { ...DEFAULT_QUERY, ...partial };

  return useQuery({
    queryKey: memoryKeys.search(partial),
    queryFn:  () => memoryService.search(fullQuery),
    enabled:  !!partial.query && partial.query.trim().length > 0,
    staleTime: 2 * 60_000,
  });
}

/**
 * Memory item detail — cache 5 min (items rarely change)
 * API: GET /v1/console/memory/{id}  (URL-encoded)
 */
export function useMemoryDetail(id: string) {
  return useQuery({
    queryKey:  memoryKeys.detail(id),
    queryFn:   () => memoryService.getById(id),
    enabled:   !!id,
    staleTime: 5 * 60_000,
  });
}

/**
 * Neighbor memories by strategy
 * API: GET /v1/console/memory/{id}/neighbors?strategy=<s>
 */
export function useMemoryNeighbors(
  id: string,
  strategy: 'semantic' | 'graph' | 'temporal' = 'semantic',
) {
  return useQuery({
    queryKey: memoryKeys.neighbors(id, strategy),
    queryFn:  () => memoryService.getNeighbors(id, strategy),
    enabled:  !!id,
  });
}

/**
 * Memory version chain — only for Supermemory items (sm: prefix)
 * API: GET /v1/console/memory/{id}/versions
 */
export function useMemoryVersions(id: string) {
  const isSupermemory = id.startsWith('sm:') || id.startsWith('supermemory:');
  return useQuery({
    queryKey: memoryKeys.versions(id),
    queryFn:  () => memoryService.getVersions(id),
    enabled:  !!id && isSupermemory,
  });
}
