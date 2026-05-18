import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';

const BASE = API_CONFIG.engines.cognee.baseUrl;

export const cogneeService = {
  add: (data: any) => apiClient.post<any>(`${BASE}/add`, data),
  getDatasets: () => apiClient.get<any[]>(`${BASE}/datasets`),
  cognify: (data: any) => apiClient.post<any>(`${BASE}/cognify`, data),
  getCognifyStatus: (id: string) => apiClient.get<any>(`${BASE}/cognify/${id}/status`),
  search: (query: any) => apiClient.post<any>(`${BASE}/search`, query),
  explore: (params: any) => {
    const qs = new URLSearchParams(params).toString();
    return apiClient.get<any>(`${BASE}/search/explore?${qs}`);
  },
  ragSearch: (query: any) => apiClient.post<any>(`${BASE}/search/rag`, query),
};
