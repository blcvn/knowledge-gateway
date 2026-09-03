/**
 * Memory Service — calls real /v1/console/memory/* endpoints
 * TASK-API-006: encodeURIComponent for compound IDs, typed MemoryVersion
 */

import { apiClient }  from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type { MemorySearchQuery, MemorySearchResult, MemoryItem, MemoryVersion } from '../types/memory';

const BASE = API_CONFIG.console.memory;

export const memoryService = {
  /** POST /v1/console/memory/search */
  search: (query: MemorySearchQuery): Promise<MemorySearchResult> =>
    apiClient.post<MemorySearchResult>(`${BASE}/search`, query),

  /**
   * GET /v1/console/memory/{id}
   * IMPORTANT: IDs có format "engine:local_id" → phải encodeURIComponent
   * Ví dụ: "graphiti:ep_abc" → "graphiti%3Aep_abc"
   */
  getById: (id: string): Promise<MemoryItem> =>
    apiClient.get<MemoryItem>(`${BASE}/${encodeURIComponent(id)}`),

  /**
   * GET /v1/console/memory/{id}/neighbors?strategy=<s>&limit=<n>
   */
  getNeighbors: (
    id: string,
    strategy: 'semantic' | 'graph' | 'temporal' = 'semantic',
    limit = 10,
  ): Promise<MemorySearchResult> =>
    apiClient.get<MemorySearchResult>(
      `${BASE}/${encodeURIComponent(id)}/neighbors?strategy=${strategy}&limit=${limit}`,
    ),

  /**
   * GET /v1/console/memory/{id}/versions
   * Only available for Supermemory items (id starts with "sm:")
   */
  getVersions: (id: string): Promise<MemoryVersion[]> =>
    apiClient.get<MemoryVersion[]>(`${BASE}/${encodeURIComponent(id)}/versions`),
};
