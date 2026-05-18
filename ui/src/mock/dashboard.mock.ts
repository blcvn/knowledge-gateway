import type { KPIData, EngineHealth, ThroughputData } from '../types/dashboard';

export const dashboardMock = {
  kpis: {
    activeAgents: 42,
    recallLatencyP50Ms: 45,
    recallLatencyP95Ms: 120,
    contextSavingsPct: 35.5,
    graphNodesTotal: 154200,
    graphEdgesTotal: 489000,
    graphGrowth24h: 1200,
    errorRatePct: 0.5,
    activeSessions: 128,
    activeProfiles: 1500,
    memoryVersions: 45000,
  } as KPIData,
  
  engineHealth: [
    { name: 'memobase', role: 'Profile Engine', status: 'Healthy', latencyP50Ms: 45, latencyP95Ms: 110, queueDepth: 0, uptimeSeconds: 36000, lastCheck: new Date().toISOString() },
    { name: 'supermemory', role: 'Adaptive Engine', status: 'Healthy', latencyP50Ms: 50, latencyP95Ms: 120, queueDepth: 5, uptimeSeconds: 36000, lastCheck: new Date().toISOString() },
    { name: 'graphiti', role: 'Knowledge Graph', status: 'Healthy', latencyP50Ms: 30, latencyP95Ms: 80, queueDepth: 2, uptimeSeconds: 36000, lastCheck: new Date().toISOString() },
    { name: 'cognee', role: 'Semantic Search', status: 'Healthy', latencyP50Ms: 25, latencyP95Ms: 70, queueDepth: 0, uptimeSeconds: 36000, lastCheck: new Date().toISOString() },
    { name: 'zep', role: 'Conversational Memory', status: 'Healthy', latencyP50Ms: 20, latencyP95Ms: 60, queueDepth: 0, uptimeSeconds: 36000, lastCheck: new Date().toISOString() },
    { name: 'openviking', role: 'Procedural Memory', status: 'Warning', latencyP50Ms: 150, latencyP95Ms: 400, queueDepth: 45, uptimeSeconds: 36000, lastCheck: new Date().toISOString() },
    { name: 'kgs', role: 'Vector DB', status: 'Healthy', latencyP50Ms: 10, latencyP95Ms: 30, queueDepth: 0, uptimeSeconds: 36000, lastCheck: new Date().toISOString() },
  ] as EngineHealth[],

  throughput: {
    window: '1h',
    engines: {
      memobase: { ingestPerSec: 10, recallPerSec: 50, embedPerSec: 10, profileExtractionsPerSec: 5 },
      supermemory: { ingestPerSec: 5, recallPerSec: 20, embedPerSec: 5, queueBacklog: 0 },
      graphiti: { ingestPerSec: 15, recallPerSec: 80, embedPerSec: 15 },
      cognee: { ingestPerSec: 20, recallPerSec: 100, embedPerSec: 20 },
      zep: { ingestPerSec: 50, recallPerSec: 200, embedPerSec: 50 },
      openviking: { ingestPerSec: 2, recallPerSec: 10, embedPerSec: 2 },
      kgs: { ingestPerSec: 100, recallPerSec: 500, embedPerSec: 100 },
    }
  } as ThroughputData,
};
