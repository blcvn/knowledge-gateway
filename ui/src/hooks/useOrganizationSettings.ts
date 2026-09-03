/**
 * Org & SDK Hooks — real API, no mock
 * TASK-API-013: useCreateApiKey with raw_key show-once pattern, query key factory
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { orgService }          from '../services/org.service';
import type { OrgSettings, CreateKeyPayload, CreateWebhookPayload } from '../types/org';

// ─── Query Key Factory ────────────────────────────────────────────────────────

const keys = {
  settings:   () => ['org', 'settings'] as const,
  members:    () => ['org', 'members'] as const,
  roles:      () => ['org', 'roles'] as const,
  apiKeys:    () => ['sdk', 'keys'] as const,
  rateLimits: () => ['sdk', 'rate-limits'] as const,
  webhooks:   () => ['sdk', 'webhooks'] as const,
};

// ─── Org Query Hooks ──────────────────────────────────────────────────────────

/** GET /v1/console/org/settings */
export function useOrgSettings() {
  return useQuery({
    queryKey: keys.settings(),
    queryFn:  () => orgService.getSettings(),
  });
}

/** GET /v1/console/org/members */
export function useMembers() {
  return useQuery({
    queryKey: keys.members(),
    queryFn:  () => orgService.getMembers(),
  });
}

/** GET /v1/console/org/roles */
export function useRoles() {
  return useQuery({
    queryKey: keys.roles(),
    queryFn:  () => orgService.getRoles(),
  });
}

// ─── SDK Query Hooks ──────────────────────────────────────────────────────────

/** GET /v1/console/sdk/keys — list (no raw_key in response) */
export function useApiKeys() {
  return useQuery({
    queryKey: keys.apiKeys(),
    queryFn:  () => orgService.getKeys(),
  });
}

/** GET /v1/console/sdk/rate-limits */
export function useRateLimits() {
  return useQuery({
    queryKey: keys.rateLimits(),
    queryFn:  () => orgService.getRateLimits(),
  });
}

/** GET /v1/console/sdk/webhooks */
export function useWebhooks() {
  return useQuery({
    queryKey: keys.webhooks(),
    queryFn:  () => orgService.getWebhooks(),
  });
}

// ─── Mutation Hooks ───────────────────────────────────────────────────────────

/** PUT /v1/console/org/settings */
export function useUpdateOrgSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: Partial<OrgSettings>) => orgService.updateSettings(payload),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.settings() }),
  });
}

/**
 * POST /v1/console/sdk/keys
 *
 * ⚠️ raw_key pattern — IMPORTANT:
 * The `raw_key` in onSuccess data is only available ONCE.
 * Caller MUST save it to local component state immediately:
 *
 * const { mutate } = useCreateApiKey();
 * mutate(payload, {
 *   onSuccess: ({ raw_key }) => setNewKey(raw_key)  // show in modal
 * });
 */
export function useCreateApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateKeyPayload) => orgService.createKey(payload),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.apiKeys() }),
  });
}

/** DELETE /v1/console/sdk/keys/{id} */
export function useRevokeApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => orgService.revokeKey(id),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.apiKeys() }),
  });
}

/** POST /v1/console/sdk/webhooks */
export function useCreateWebhook() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateWebhookPayload) => orgService.createWebhook(payload),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.webhooks() }),
  });
}

/** DELETE /v1/console/sdk/webhooks/{id} */
export function useDeleteWebhook() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => orgService.deleteWebhook(id),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.webhooks() }),
  });
}
