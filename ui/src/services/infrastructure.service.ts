/**
 * Infrastructure Service — calls real /v1/console/infra/* endpoints
 * TASK-API-012: typed topology, deployments
 */

import { apiClient }  from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type {
  ServiceInfo, DatabaseHealth, ResourceMetrics,
  InfraTopology, DeploymentInfo,
} from '../types/infrastructure';

const BASE = API_CONFIG.console.infra;

export const infrastructureService = {
  /** GET /v1/console/infra/topology */
  getTopology: (): Promise<InfraTopology> =>
    apiClient.get<InfraTopology>(`${BASE}/topology`),

  /** GET /v1/console/infra/services */
  getServiceHealth: (): Promise<ServiceInfo[]> =>
    apiClient.get<ServiceInfo[]>(`${BASE}/services`),

  /** GET /v1/console/infra/services/{name} */
  getServiceDetail: (name: string): Promise<ServiceInfo> =>
    apiClient.get<ServiceInfo>(`${BASE}/services/${encodeURIComponent(name)}`),

  /** GET /v1/console/infra/databases */
  getDatabaseHealth: (): Promise<DatabaseHealth[]> =>
    apiClient.get<DatabaseHealth[]>(`${BASE}/databases`),

  /** GET /v1/console/infra/resources */
  getResourceMetrics: (): Promise<ResourceMetrics[]> =>
    apiClient.get<ResourceMetrics[]>(`${BASE}/resources`),

  /** GET /v1/console/infra/deployments */
  getDeployments: (): Promise<DeploymentInfo[]> =>
    apiClient.get<DeploymentInfo[]>(`${BASE}/deployments`),
};
