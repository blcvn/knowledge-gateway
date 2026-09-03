/**
 * Governance Hooks — real API, no mock
 * TASK-API-009: mutations for CRUD + GDPR 2-step flow, query key factory
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { governanceService }  from '../services/governance.service';
import type { Tenant, Policy, AuditFilters } from '../types/governance';

// ─── Query Key Factory ────────────────────────────────────────────────────────

const keys = {
  tenants:  () => ['governance', 'tenants'] as const,
  policies: () => ['governance', 'policies'] as const,
  audit:    (f: AuditFilters) => ['governance', 'audit', f] as const,
};

// ─── Query Hooks ──────────────────────────────────────────────────────────────

/** GET /v1/console/governance/tenants */
export function useTenants() {
  return useQuery({
    queryKey: keys.tenants(),
    queryFn:  () => governanceService.getTenants(),
  });
}

/** GET /v1/console/governance/policies */
export function usePolicies() {
  return useQuery({
    queryKey: keys.policies(),
    queryFn:  () => governanceService.getPolicies(),
  });
}

/** GET /v1/console/governance/audit[?...filters] */
export function useAuditLogs(filters: AuditFilters = {}) {
  return useQuery({
    queryKey: keys.audit(filters),
    queryFn:  () => governanceService.getAuditLogs(filters),
  });
}

// ─── Tenant Mutations ─────────────────────────────────────────────────────────

export function useCreateTenant() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Tenant>) => governanceService.createTenant(data),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.tenants() }),
  });
}

export function useUpdateTenant() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Tenant> }) =>
      governanceService.updateTenant(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.tenants() }),
  });
}

// ─── Policy Mutations ─────────────────────────────────────────────────────────

export function useCreatePolicy() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Policy>) => governanceService.createPolicy(data),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.policies() }),
  });
}

export function useUpdatePolicy() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Policy> }) =>
      governanceService.updatePolicy(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.policies() }),
  });
}

// ─── GDPR 2-Step Flow ─────────────────────────────────────────────────────────

/**
 * Step 1: Preview GDPR deletion
 * UI must show breakdown_by_engine and warnings before calling useGDPRForget
 *
 * Usage:
 *   const preview = useGDPRPreview();
 *   preview.mutate('user_123', {
 *     onSuccess: (data) => setShowConfirm(true) // show: "will delete 450 items"
 *   });
 */
export function useGDPRPreview() {
  return useMutation({
    mutationFn: (userId: string) => governanceService.gdprForgetPreview(userId),
  });
}

/**
 * Step 2: Execute GDPR forget (irreversible)
 * Only call after user explicitly confirms the preview
 */
export function useGDPRForget() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => governanceService.gdprForget(userId),
    onSuccess:  () => qc.invalidateQueries({ queryKey: ['governance'] }),
  });
}
