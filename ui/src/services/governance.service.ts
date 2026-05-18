import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type { Tenant, Policy, AuditLogEntry } from '../types/governance';

const BASE = API_CONFIG.console.governance;

export const governanceService = {
  /** GET /v1/console/governance/tenants */
  getTenants: () =>
    apiClient.get<Tenant[]>(`${BASE}/tenants`),

  /** POST /v1/console/governance/tenants */
  createTenant: (data: Partial<Tenant>) =>
    apiClient.post<Tenant>(`${BASE}/tenants`, data),

  /** PUT /v1/console/governance/tenants/{id} */
  updateTenant: (id: string, data: Partial<Tenant>) =>
    apiClient.put<Tenant>(`${BASE}/tenants/${id}`, data),

  /** GET /v1/console/governance/policies */
  getPolicies: () =>
    apiClient.get<Policy[]>(`${BASE}/policies`),

  /** POST /v1/console/governance/policies */
  createPolicy: (policy: Partial<Policy>) =>
    apiClient.post<Policy>(`${BASE}/policies`, policy),

  /** PUT /v1/console/governance/policies/{id} */
  updatePolicy: (id: string, policy: Partial<Policy>) =>
    apiClient.put<Policy>(`${BASE}/policies/${id}`, policy),

  /** GET /v1/console/governance/audit */
  getAuditLogs: (filters: Record<string, string>) => {
    const qs = new URLSearchParams(filters).toString();
    return apiClient.get<AuditLogEntry[]>(`${BASE}/audit?${qs}`);
  },

  /** POST /v1/console/governance/gdpr/forget */
  gdprForget: (userId: string) =>
    apiClient.post<void>(`${BASE}/gdpr/forget`, { user_id: userId }),

  /** POST /v1/console/governance/gdpr/forget/preview */
  gdprForgetPreview: (userId: string) =>
    apiClient.post<unknown>(`${BASE}/gdpr/forget/preview`, { user_id: userId }),
};
