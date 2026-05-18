import { PipelineStatus, EngineType } from './api';

export interface PipelineJob {
  id: string;
  engine: EngineType;
  status: PipelineStatus;
  progress: number;
  created_at: string;
  updated_at: string;
}

export interface PipelineStage {
  name: string;
  status: PipelineStatus;
  duration_ms: number;
}

export interface QueueMetrics {
  depth: number;
  throughput: number;
  retry_count: number;
}

export interface WorkerStatus {
  engine: EngineType;
  running: number;
  idle: number;
}
