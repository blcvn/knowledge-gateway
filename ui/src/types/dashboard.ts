import { HealthStatus, EngineType } from './api';

export interface KPIData {
  activeAgents: number;
  recallLatencyP50Ms: number;
  recallLatencyP95Ms: number;
  contextSavingsPct: number;
  graphNodesTotal: number;
  graphEdgesTotal: number;
  graphGrowth24h: number;
  errorRatePct: number;
  activeSessions: number;
  activeProfiles: number;
  memoryVersions: number;
}

export interface EngineHealth {
  name: EngineType;
  role: string;
  status: HealthStatus;
  latencyP50Ms: number;
  latencyP95Ms: number;
  queueDepth: number;
  uptimeSeconds: number;
  lastCheck: string;
}

export interface MemoryFlowMetrics {
  ingestPerSec: number;
  recallPerSec: number;
  embedPerSec: number;
  profileExtractionsPerSec?: number;
  queueBacklog?: number;
}

export interface ThroughputData {
  window: string;
  engines: Record<EngineType, MemoryFlowMetrics>;
}

export interface HeatmapData {
  points: { x: number; y: number; density: number }[];
}
