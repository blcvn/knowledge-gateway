import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type { PipelineJob, QueueMetrics } from '../types/pipeline';

const BASE = API_CONFIG.console.pipelines;

export const pipelineService = {
  /** GET /v1/console/pipelines/status */
  getStatus: () =>
    apiClient.get<unknown>(`${BASE}/status`),

  /** GET /v1/console/pipelines/queues */
  getQueues: () =>
    apiClient.get<QueueMetrics>(`${BASE}/queues`),

  /** GET /v1/console/pipelines/workers */
  getWorkers: () =>
    apiClient.get<unknown[]>(`${BASE}/workers`),

  /** GET /v1/console/pipelines/templates */
  getTemplates: () =>
    apiClient.get<unknown[]>(`${BASE}/templates`),

  /** GET /v1/console/pipelines/{engine} */
  getEngine: (engine: string) =>
    apiClient.get<unknown>(`${BASE}/${engine}`),

  /** GET /v1/console/pipelines/{engine}/jobs */
  getJobs: (engine: string) =>
    apiClient.get<PipelineJob[]>(`${BASE}/${engine}/jobs`),

  /** GET /v1/console/pipelines/{engine}/jobs/{id} */
  getJob: (engine: string, jobId: string) =>
    apiClient.get<PipelineJob>(`${BASE}/${engine}/jobs/${jobId}`),
};
