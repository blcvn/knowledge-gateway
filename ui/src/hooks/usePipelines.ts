import { useQuery } from '@tanstack/react-query';
import { pipelineService } from '../services/pipeline.service';
import { pipelineMock } from '../mock/pipeline.mock';
import { API_CONFIG } from '../config/api.config';

const useMock = API_CONFIG.useMockData;

export function usePipelineJobs(engine: string) {
  return useQuery({
    queryKey: ['pipelines', 'jobs', engine],
    queryFn: useMock
      ? () => Promise.resolve(pipelineMock.jobs)
      : () => pipelineService.getJobs(engine),
  });
}

export function useQueueMetrics() {
  return useQuery({
    queryKey: ['pipelines', 'queueMetrics'],
    queryFn: useMock
      ? () => Promise.resolve(pipelineMock.metrics)
      : () => pipelineService.getQueueMetrics(),
  });
}
