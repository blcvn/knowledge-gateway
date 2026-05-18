import { useQuery } from '@tanstack/react-query';
import { API_CONFIG } from '../config/api.config';
import { apiClient } from '../lib/api-client';

const useMock = API_CONFIG.useMockData;
const BASE = API_CONFIG.engines.gateway.baseUrl;

const mockSettings = {
  name: 'VNP Platform',
  slug: 'vnp-platform',
  domain: 'vnp-memory.io',
  timezone: 'Asia/Ho_Chi_Minh',
  maxAgents: 100,
  maxMemoriesPerUser: 10000,
};

const mockMembers = [
  { id: 'm1', name: 'Nguyen Binh', email: 'binh@vnp.io', role: 'owner', status: 'active', joinedAt: '2025-01-01' },
  { id: 'm2', name: 'Alice Chen', email: 'alice@vnp.io', role: 'admin', status: 'active', joinedAt: '2025-02-15' },
  { id: 'm3', name: 'Bob Kim', email: 'bob@vnp.io', role: 'developer', status: 'active', joinedAt: '2025-03-20' },
  { id: 'm4', name: 'Carol Liu', email: 'carol@vnp.io', role: 'developer', status: 'active', joinedAt: '2025-04-10' },
  { id: 'm5', name: 'Dave Park', email: 'dave@vnp.io', role: 'viewer', status: 'inactive', joinedAt: '2025-05-05' },
];

const mockRoles = [
  { id: 'r1', name: 'owner', permissions: ['*'] },
  { id: 'r2', name: 'admin', permissions: ['memory:*', 'graph:*', 'governance:read'] },
  { id: 'r3', name: 'developer', permissions: ['memory:read', 'memory:write', 'graph:read', 'debug:*'] },
  { id: 'r4', name: 'viewer', permissions: ['memory:read', 'graph:read'] },
];

export function useOrgSettings() {
  return useQuery({
    queryKey: ['org', 'settings'],
    queryFn: useMock
      ? () => Promise.resolve(mockSettings)
      : () => apiClient.get<typeof mockSettings>(`${BASE}/v1/org/settings`),
  });
}

export function useMembers() {
  return useQuery({
    queryKey: ['org', 'members'],
    queryFn: useMock
      ? () => Promise.resolve(mockMembers)
      : () => apiClient.get<typeof mockMembers>(`${BASE}/v1/org/members`),
  });
}

export function useRoles() {
  return useQuery({
    queryKey: ['org', 'roles'],
    queryFn: useMock
      ? () => Promise.resolve(mockRoles)
      : () => apiClient.get<typeof mockRoles>(`${BASE}/v1/org/roles`),
  });
}
