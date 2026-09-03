/**
 * Adaptive Memory Hooks — real API, no mock
 * TASK-API-007: mutations for create/sync connectors, update forget-rules, poll analytics 60s
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { adaptiveService }                        from '../services/adaptive.service';
import type { ExternalConnector, ForgetRule }     from '../types/adaptive';

// ─── Query Key Factory ────────────────────────────────────────────────────────

const keys = {
  memories:    () => ['adaptive', 'memories'] as const,
  versions:    (id: string) => ['adaptive', 'memories', id, 'versions'] as const,
  connectors:  () => ['adaptive', 'connectors'] as const,
  analytics:   () => ['adaptive', 'analytics'] as const,
  forgetRules: () => ['adaptive', 'forget-rules'] as const,
};

// ─── Query Hooks ──────────────────────────────────────────────────────────────

/** GET /v1/console/adaptive/memories */
export function useAdaptiveMemories() {
  return useQuery({
    queryKey:  keys.memories(),
    queryFn:   () => adaptiveService.getMemories(),
    staleTime: 60_000,
  });
}

/** GET /v1/console/adaptive/memories/{id}/versions */
export function useMemoryVersions(id: string) {
  return useQuery({
    queryKey: keys.versions(id),
    queryFn:  () => adaptiveService.getMemoryVersions(id),
    enabled:  !!id,
  });
}

/** GET /v1/console/adaptive/connectors */
export function useConnectors() {
  return useQuery({
    queryKey: keys.connectors(),
    queryFn:  () => adaptiveService.getConnectors(),
  });
}

/** GET /v1/console/adaptive/analytics — poll 60s */
export function useAdaptiveAnalytics() {
  return useQuery({
    queryKey:        keys.analytics(),
    queryFn:         () => adaptiveService.getAnalytics(),
    refetchInterval: 60_000,
  });
}

/** GET /v1/console/adaptive/forget-rules */
export function useForgetRules() {
  return useQuery({
    queryKey: keys.forgetRules(),
    queryFn:  () => adaptiveService.getForgetRules(),
  });
}

// ─── Mutation Hooks ───────────────────────────────────────────────────────────

/** POST /v1/console/adaptive/connectors — create new external connector */
export function useCreateConnector() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<ExternalConnector>) => adaptiveService.createConnector(data),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.connectors() }),
  });
}

/** POST /v1/console/adaptive/connectors/{id}/sync — trigger sync job */
export function useSyncConnector() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => adaptiveService.syncConnector(id),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.connectors() }),
  });
}

/** PUT /v1/console/adaptive/forget-rules — update forget policy */
export function useUpdateForgetRules() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (rules: ForgetRule[]) => adaptiveService.updateForgetRules(rules),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.forgetRules() }),
  });
}
