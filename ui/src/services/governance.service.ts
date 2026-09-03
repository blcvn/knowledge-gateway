/**
 * Governance Service — calls real /v1/console/governance/* endpoints
 * TASK-API-009: typed GDPR preview/forget responses, AuditFilters
 */

import { apiClient }  from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type {
  Tenant, Policy, AuditLogEntry, AuditFilters,
  GDPRPreviewResponse, GDPRForgetResponse,
} from '../types/governance';

const BASE = API_CONFIG.console.governance;

function buildQuery(filters: Record<string, string | undefined>): string {
  const params = new URLSearchParams();
  Object.entries(filters).forEach(([k, v]) => { if (v) params.set(k, v); });
  const qs = params.toString();
  return qs ? `?${qs}` : '';
}

export const governanceService = {
  // ── Tenants ──────────────────────────────────────────────────────────────

  /** GET /v1/console/governance/tenants */
  getTenants: (): Promise<Tenant[]> =>
    apiClient.get<Tenant[]>(`${BASE}/tenants`),

  /** POST /v1/console/governance/tenants */
  createTenant: (data: Partial<Tenant>): Promise<Tenant> =>
    apiClient.post<Tenant>(`${BASE}/tenants`, data),

  /** PUT /v1/console/governance/tenants/{id} */
  updateTenant: (id: string, data: Partial<Tenant>): Promise<Tenant> =>
    apiClient.put<Tenant>(`${BASE}/tenants/${id}`, data),

  // ── Policies ─────────────────────────────────────────────────────────────

  /** GET /v1/console/governance/policies */
  getPolicies: (): Promise<Policy[]> =>
    apiClient.get<Policy[]>(`${BASE}/policies`),

  /** POST /v1/console/governance/policies */
  createPolicy: (policy: Partial<Policy>): Promise<Policy> =>
    apiClient.post<Policy>(`${BASE}/policies`, policy),

  /** PUT /v1/console/governance/policies/{id} */
  updatePolicy: (id: string, policy: Partial<Policy>): Promise<Policy> =>
    apiClient.put<Policy>(`${BASE}/policies/${id}`, policy),

  // ── Audit Logs ────────────────────────────────────────────────────────────

  /** GET /v1/console/governance/audit[?...filters] */
  getAuditLogs: (filters: AuditFilters = {}): Promise<AuditLogEntry[]> =>
    apiClient.get<AuditLogEntry[]>(`${BASE}/audit${buildQuery(filters as Record<string, string>)}`),

  // ── GDPR ──────────────────────────────────────────────────────────────────

  /**
   * POST /v1/console/governance/gdpr/forget/preview
   * Step 1: preview what would be deleted — UI must show this before executing
   */
  gdprForgetPreview: (userId: string): Promise<GDPRPreviewResponse> =>
    apiClient.post<GDPRPreviewResponse>(`${BASE}/gdpr/forget/preview`, { user_id: userId }),

  /**
   * POST /v1/console/governance/gdpr/forget
   * Step 2: execute deletion (requires user confirmation in UI)
   */
  gdprForget: (userId: string): Promise<GDPRForgetResponse> =>
    apiClient.post<GDPRForgetResponse>(`${BASE}/gdpr/forget`, { user_id: userId }),
};
