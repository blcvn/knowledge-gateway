/**
 * Infrastructure Hooks — real API, no mock
 * TASK-API-012: services/databases poll 30s, topology staleTime 5min
 */

import { useQuery } from '@tanstack/react-query';
import { infrastructureService } from '../services/infrastructure.service';

// ─── Query Key Factory ────────────────────────────────────────────────────────

const keys = {
  topology:    () => ['infra', 'topology'] as const,
  services:    () => ['infra', 'services'] as const,
  service:     (name: string) => ['infra', 'services', name] as const,
  databases:   () => ['infra', 'databases'] as const,
  resources:   () => ['infra', 'resources'] as const,
  deployments: () => ['infra', 'deployments'] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

/**
 * GET /v1/console/infra/topology
 * staleTime 5min — monolith topology rarely changes
 */
export function useTopology() {
  return useQuery({
    queryKey:  keys.topology(),
    queryFn:   () => infrastructureService.getTopology(),
    staleTime: 5 * 60_000,
  });
}

/**
 * GET /v1/console/infra/services — poll 30s
 * 35-service status changes with restarts
 */
export function useServiceHealth() {
  return useQuery({
    queryKey:        keys.services(),
    queryFn:         () => infrastructureService.getServiceHealth(),
    refetchInterval: 30_000,
  });
}

/** GET /v1/console/infra/services/{name} — single service detail */
export function useServiceDetail(name: string) {
  return useQuery({
    queryKey: keys.service(name),
    queryFn:  () => infrastructureService.getServiceDetail(name),
    enabled:  !!name,
  });
}

/**
 * GET /v1/console/infra/databases — poll 30s
 * PostgreSQL/Redis/Neo4j/NATS health
 */
export function useDatabaseHealth() {
  return useQuery({
    queryKey:        keys.databases(),
    queryFn:         () => infrastructureService.getDatabaseHealth(),
    refetchInterval: 30_000,
  });
}

/** GET /v1/console/infra/resources — CPU/mem/disk — poll 30s */
export function useResourceMetrics() {
  return useQuery({
    queryKey:        keys.resources(),
    queryFn:         () => infrastructureService.getResourceMetrics(),
    refetchInterval: 30_000,
  });
}

/** GET /v1/console/infra/deployments — staleTime 5min */
export function useDeployments() {
  return useQuery({
    queryKey:  keys.deployments(),
    queryFn:   () => infrastructureService.getDeployments(),
    staleTime: 5 * 60_000,
  });
}
