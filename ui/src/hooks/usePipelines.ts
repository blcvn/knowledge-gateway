/**
 * Pipelines Hooks — real API, no mock
 * TASK-API-010: queue poll 10s, query key factory, retry job mutation
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { pipelineService } from '../services/pipeline.service';

// ─── Query Key Factory ────────────────────────────────────────────────────────

const keys = {
  status:  () => ['pipelines', 'status'] as const,
  queues:  () => ['pipelines', 'queues'] as const,
  workers: () => ['pipelines', 'workers'] as const,
  jobs:    (engine: string) => ['pipelines', 'jobs', engine] as const,
  job:     (engine: string, id: string) => ['pipelines', 'jobs', engine, id] as const,
};

// ─── Query Hooks ──────────────────────────────────────────────────────────────

/** GET /v1/console/pipelines/status */
export function usePipelineStatus() {
  return useQuery({
    queryKey:        keys.status(),
    queryFn:         () => pipelineService.getStatus(),
    refetchInterval: 30_000,
  });
}

/**
 * GET /v1/console/pipelines/queues — poll 10s
 * Queue depth changes rapidly during ingestion
 */
export function useQueueMetrics() {
  return useQuery({
    queryKey:        keys.queues(),
    queryFn:         () => pipelineService.getQueues(),
    refetchInterval: 10_000,
  });
}

/** GET /v1/console/pipelines/workers */
export function useWorkerStatus() {
  return useQuery({
    queryKey:        keys.workers(),
    queryFn:         () => pipelineService.getWorkers(),
    refetchInterval: 30_000,
  });
}

/** GET /v1/console/pipelines/{engine}/jobs */
export function usePipelineJobs(engine: string) {
  return useQuery({
    queryKey:        keys.jobs(engine),
    queryFn:         () => pipelineService.getJobs(engine),
    enabled:         !!engine,
    refetchInterval: 15_000,
  });
}

/** GET /v1/console/pipelines/{engine}/jobs/{id} */
export function usePipelineJob(engine: string, jobId: string) {
  return useQuery({
    queryKey: keys.job(engine, jobId),
    queryFn:  () => pipelineService.getJob(engine, jobId),
    enabled:  !!engine && !!jobId,
  });
}
