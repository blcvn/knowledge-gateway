import { useQuery } from '@tanstack/react-query';
import { API_CONFIG } from '../config/api.config';
import { apiClient } from '../lib/api-client';

const useMock = API_CONFIG.useMockData;
const BASE = API_CONFIG.engines.gateway.baseUrl;

/* ─── Mock Data ─── */
const mockApiKeys = [
  { id: 'key_1', name: 'Production Agent', key: 'vnp_prod_sk_3f9a2b8c...', scopes: ['memory:read', 'memory:write', 'graph:read'], createdAt: '2026-01-15', lastUsed: '2026-05-12', status: 'active' },
  { id: 'key_2', name: 'Staging Environment', key: 'vnp_stg_sk_7d1e4f2a...', scopes: ['memory:read'], createdAt: '2026-02-20', lastUsed: '2026-04-30', status: 'active' },
  { id: 'key_3', name: 'Dev CLI Tool', key: 'vnp_dev_sk_2c8b5e9f...', scopes: ['memory:read', 'debug:context'], createdAt: '2026-03-10', lastUsed: '2026-05-01', status: 'inactive' },
];

const mockRateLimits = [
  { scope: 'Global (Default)', rps: 1000, rpm: 60000, burst: 2000, tier: 'enterprise' },
  { scope: 'memory:write', rps: 100, rpm: 6000, burst: 200, tier: 'standard' },
  { scope: 'graph:query', rps: 50, rpm: 3000, burst: 100, tier: 'standard' },
  { scope: 'debug:context', rps: 10, rpm: 600, burst: 20, tier: 'restricted' },
];

const mockWebhooks = [
  { id: 'wh_1', url: 'https://api.example.com/webhooks/vnp', events: ['memory.created', 'memory.deleted', 'profile.updated'], status: 'active', successRate: 99.2 },
  { id: 'wh_2', url: 'https://monitoring.example.com/vnp-hook', events: ['engine.health_degraded', 'pipeline.failed'], status: 'active', successRate: 100 },
];

/* ─── Hooks ─── */
export function useApiKeys() {
  return useQuery({
    queryKey: ['api-sdk', 'keys'],
    queryFn: useMock
      ? () => Promise.resolve(mockApiKeys)
      : () => apiClient.get<typeof mockApiKeys>(`${BASE}/v1/api-keys`),
  });
}

export function useRateLimits() {
  return useQuery({
    queryKey: ['api-sdk', 'rate-limits'],
    queryFn: useMock
      ? () => Promise.resolve(mockRateLimits)
      : () => apiClient.get<typeof mockRateLimits>(`${BASE}/v1/rate-limits`),
  });
}

export function useWebhooks() {
  return useQuery({
    queryKey: ['api-sdk', 'webhooks'],
    queryFn: useMock
      ? () => Promise.resolve(mockWebhooks)
      : () => apiClient.get<typeof mockWebhooks>(`${BASE}/v1/webhooks`),
  });
}
