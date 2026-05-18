import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';

const BASE = API_CONFIG.engines.zep.baseUrl;

export const zepService = {
  createUser: (data: any) => apiClient.post<any>(`${BASE}/users`, data),
  getThreads: () => apiClient.get<any[]>(`${BASE}/threads`),
  addMemory: (data: any) => apiClient.post<any>(`${BASE}/memories`, data),
  getGraph: () => apiClient.get<any>(`${BASE}/graph`),
  search: (query: string) => apiClient.get<any>(`${BASE}/search?q=${query}`),
};
