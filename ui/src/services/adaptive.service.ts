import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type {
  AdaptiveMemory, MemoryVersion, ForgetRule,
  ExternalConnector, AdaptiveAnalytics
} from '../types/adaptive';

const BASE = API_CONFIG.console.adaptive;

export const adaptiveService = {
  /** GET /v1/console/adaptive/memories */
  getMemories: () =>
    apiClient.get<AdaptiveMemory[]>(`${BASE}/memories`),

  /** GET /v1/console/adaptive/memories/{id}/versions */
  getMemoryVersions: (id: string) =>
    apiClient.get<MemoryVersion[]>(`${BASE}/memories/${id}/versions`),

  /** GET /v1/console/adaptive/connectors */
  getConnectors: () =>
    apiClient.get<ExternalConnector[]>(`${BASE}/connectors`),

  /** POST /v1/console/adaptive/connectors */
  createConnector: (data: Partial<ExternalConnector>) =>
    apiClient.post<ExternalConnector>(`${BASE}/connectors`, data),

  /** POST /v1/console/adaptive/connectors/{id}/sync */
  syncConnector: (id: string) =>
    apiClient.post<void>(`${BASE}/connectors/${id}/sync`, {}),

  /** GET /v1/console/adaptive/analytics */
  getAnalytics: () =>
    apiClient.get<AdaptiveAnalytics>(`${BASE}/analytics`),

  /** GET /v1/console/adaptive/forget-rules */
  getForgetRules: () =>
    apiClient.get<ForgetRule[]>(`${BASE}/forget-rules`),

  /** PUT /v1/console/adaptive/forget-rules */
  updateForgetRules: (rules: ForgetRule[]) =>
    apiClient.put<ForgetRule[]>(`${BASE}/forget-rules`, rules),
};
