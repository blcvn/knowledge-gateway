/**
 * Adaptive Service — calls real /v1/console/adaptive/* endpoints
 * TASK-API-007: typed ForgetRules (single object, not array), syncConnector returns job_id
 */

import { apiClient }  from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type {
  AdaptiveMemory, MemoryVersion, ForgetRule,
  ExternalConnector, AdaptiveAnalytics,
} from '../types/adaptive';

const BASE = API_CONFIG.console.adaptive;

export const adaptiveService = {
  /** GET /v1/console/adaptive/memories */
  getMemories: (): Promise<AdaptiveMemory[]> =>
    apiClient.get<AdaptiveMemory[]>(`${BASE}/memories`),

  /** GET /v1/console/adaptive/memories/{id}/versions */
  getMemoryVersions: (id: string): Promise<MemoryVersion[]> =>
    apiClient.get<MemoryVersion[]>(`${BASE}/memories/${encodeURIComponent(id)}/versions`),

  /** GET /v1/console/adaptive/connectors */
  getConnectors: (): Promise<ExternalConnector[]> =>
    apiClient.get<ExternalConnector[]>(`${BASE}/connectors`),

  /** POST /v1/console/adaptive/connectors */
  createConnector: (data: Partial<ExternalConnector>): Promise<ExternalConnector> =>
    apiClient.post<ExternalConnector>(`${BASE}/connectors`, data),

  /** POST /v1/console/adaptive/connectors/{id}/sync — returns job_id */
  syncConnector: (id: string): Promise<{ job_id: string }> =>
    apiClient.post<{ job_id: string }>(`${BASE}/connectors/${id}/sync`, {}),

  /** GET /v1/console/adaptive/analytics */
  getAnalytics: (): Promise<AdaptiveAnalytics> =>
    apiClient.get<AdaptiveAnalytics>(`${BASE}/analytics`),

  /** GET /v1/console/adaptive/forget-rules */
  getForgetRules: (): Promise<ForgetRule[]> =>
    apiClient.get<ForgetRule[]>(`${BASE}/forget-rules`),

  /** PUT /v1/console/adaptive/forget-rules */
  updateForgetRules: (rules: ForgetRule[]): Promise<ForgetRule[]> =>
    apiClient.put<ForgetRule[]>(`${BASE}/forget-rules`, rules),
};
