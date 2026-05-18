import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type { ServiceInfo, DatabaseHealth, ResourceMetrics } from '../types/infrastructure';

const BASE = API_CONFIG.console.infra;

export const infrastructureService = {
  /** GET /v1/console/infra/topology */
  getTopology: () =>
    apiClient.get<unknown>(`${BASE}/topology`),

  /** GET /v1/console/infra/services */
  getServiceHealth: () =>
    apiClient.get<ServiceInfo[]>(`${BASE}/services`),

  /** GET /v1/console/infra/services/{name} */
  getServiceDetail: (name: string) =>
    apiClient.get<ServiceInfo>(`${BASE}/services/${name}`),

  /** GET /v1/console/infra/databases */
  getDatabaseHealth: () =>
    apiClient.get<DatabaseHealth[]>(`${BASE}/databases`),

  /** GET /v1/console/infra/resources */
  getResourceMetrics: () =>
    apiClient.get<ResourceMetrics[]>(`${BASE}/resources`),

  /** GET /v1/console/infra/deployments */
  getDeployments: () =>
    apiClient.get<unknown[]>(`${BASE}/deployments`),
};
