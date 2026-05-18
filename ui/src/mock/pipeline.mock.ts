import type { PipelineJob, QueueMetrics } from '../types/pipeline';

export const pipelineMock = {
  jobs: [{ id: 'job_1', engine: 'cognee', status: 'Running', progress: 50, created_at: new Date().toISOString(), updated_at: new Date().toISOString() }] as PipelineJob[],
  metrics: { depth: 10, throughput: 5, retry_count: 0 } as QueueMetrics,
};
