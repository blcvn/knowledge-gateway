import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type { MemorySearchResult, MemoryItem } from '../types/memory';

const BASE = API_CONFIG.console.memory;

export const memoryService = {
  /** POST /v1/console/memory/search */
  search: (query: Record<string, unknown>) =>
    apiClient.post<MemorySearchResult>(`${BASE}/search`, query),

  /** GET /v1/console/memory/{id} */
  getById: (id: string) =>
    apiClient.get<MemoryItem>(`${BASE}/${id}`),

  /** GET /v1/console/memory/{id}/neighbors */
  getNeighbors: (id: string) =>
    apiClient.get<MemorySearchResult>(`${BASE}/${id}/neighbors`),

  /** GET /v1/console/memory/{id}/versions */
  getVersions: (id: string) =>
    apiClient.get<unknown[]>(`${BASE}/${id}/versions`),
};
