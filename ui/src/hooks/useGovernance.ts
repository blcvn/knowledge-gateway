import { useQuery } from '@tanstack/react-query';
import { governanceService } from '../services/governance.service';
import { governanceMock } from '../mock/governance.mock';
import { API_CONFIG } from '../config/api.config';

const useMock = API_CONFIG.useMockData;

export function useTenants() {
  return useQuery({
    queryKey: ['governance', 'tenants'],
    queryFn: useMock
      ? () => Promise.resolve(governanceMock.tenants)
      : () => governanceService.getTenants(),
  });
}

/** Fixed C2: now calls real governanceService.getPolicies() */
export function usePolicies() {
  return useQuery({
    queryKey: ['governance', 'policies'],
    queryFn: useMock
      ? () => Promise.resolve(governanceMock.policies)
      : () => governanceService.getPolicies(),
  });
}

export function useAuditLogs(filters: Record<string, string>) {
  return useQuery({
    queryKey: ['governance', 'auditLogs', filters],
    queryFn: useMock
      ? () => Promise.resolve(governanceMock.auditLogs)
      : () => governanceService.getAuditLogs(filters),
  });
}
