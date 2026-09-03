/**
 * Org & SDK Service — calls real /v1/console/org/* and /v1/console/sdk/* endpoints
 * TASK-API-013
 */

import { apiClient }  from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type {
  OrgSettings, OrgMember, OrgRole,
  APIKey, CreateKeyResponse, CreateKeyPayload,
  RateLimitConfig, Webhook, CreateWebhookPayload,
} from '../types/org';

// ─── Base URLs ────────────────────────────────────────────────────────────────

const ORG_BASE = '/v1/console/org';
const SDK_BASE = '/v1/console/sdk';

// ─── Service ──────────────────────────────────────────────────────────────────

export const orgService = {
  // ── Org Settings ─────────────────────────────────────────────────────────

  /** GET /v1/console/org/settings */
  getSettings: (): Promise<OrgSettings> =>
    apiClient.get<OrgSettings>(`${ORG_BASE}/settings`),

  /** PUT /v1/console/org/settings */
  updateSettings: (payload: Partial<OrgSettings>): Promise<OrgSettings> =>
    apiClient.put<OrgSettings>(`${ORG_BASE}/settings`, payload),

  /** GET /v1/console/org/members */
  getMembers: (): Promise<OrgMember[]> =>
    apiClient.get<OrgMember[]>(`${ORG_BASE}/members`),

  /** GET /v1/console/org/roles */
  getRoles: (): Promise<OrgRole[]> =>
    apiClient.get<OrgRole[]>(`${ORG_BASE}/roles`),

  // ── API Keys ──────────────────────────────────────────────────────────────

  /**
   * GET /v1/console/sdk/keys
   * NOTE: Response does NOT include raw_key — only masked prefix
   */
  getKeys: (): Promise<APIKey[]> =>
    apiClient.get<APIKey[]>(`${SDK_BASE}/keys`),

  /**
   * POST /v1/console/sdk/keys
   * IMPORTANT: response.raw_key is returned only ONCE — UI must show immediately
   */
  createKey: (payload: CreateKeyPayload): Promise<CreateKeyResponse> =>
    apiClient.post<CreateKeyResponse>(`${SDK_BASE}/keys`, payload),

  /** DELETE /v1/console/sdk/keys/{id} */
  revokeKey: (id: string): Promise<void> =>
    apiClient.delete(`${SDK_BASE}/keys/${id}`),

  // ── Rate Limits ───────────────────────────────────────────────────────────

  /** GET /v1/console/sdk/rate-limits */
  getRateLimits: (): Promise<RateLimitConfig[]> =>
    apiClient.get<RateLimitConfig[]>(`${SDK_BASE}/rate-limits`),

  // ── Webhooks ──────────────────────────────────────────────────────────────

  /** GET /v1/console/sdk/webhooks */
  getWebhooks: (): Promise<Webhook[]> =>
    apiClient.get<Webhook[]>(`${SDK_BASE}/webhooks`),

  /** POST /v1/console/sdk/webhooks */
  createWebhook: (payload: CreateWebhookPayload): Promise<Webhook> =>
    apiClient.post<Webhook>(`${SDK_BASE}/webhooks`, payload),

  /** DELETE /v1/console/sdk/webhooks/{id} */
  deleteWebhook: (id: string): Promise<void> =>
    apiClient.delete(`${SDK_BASE}/webhooks/${id}`),
};
