import type { MetricsSummary, TraceSpan, ErrorEntry } from '../types/observability';

const now = new Date();
const ts = (offsetMinutes: number) => new Date(now.getTime() - offsetMinutes * 60000).toISOString();

export const observabilityMock = {
  metrics: {
    avgLatency: 78,
    requestRate: 243,
    errorRate: 0.4,
    uptime: '99.98',
    p50: 45,
    p95: 180,
    p99: 420,
  } satisfies MetricsSummary,

  traces: [
    { id: 'tr_1', trace_id: 'trace_abc123', span_id: 'sp_001', operation: 'POST /v1/memory/add', service: 'vnp-gateway', duration: 89, status: 'ok', timestamp: ts(2) },
    { id: 'tr_2', trace_id: 'trace_def456', span_id: 'sp_002', operation: 'GET /v1/context/assemble', service: 'vnp-gateway', duration: 142, status: 'ok', timestamp: ts(5) },
    { id: 'tr_3', trace_id: 'trace_ghi789', span_id: 'sp_003', operation: 'POST /v1/graph/query', service: 'graphiti', duration: 56, status: 'ok', timestamp: ts(8) },
    { id: 'tr_4', trace_id: 'trace_jkl012', span_id: 'sp_004', operation: 'GET /v1/users/:id/profile', service: 'memobase', duration: 34, status: 'ok', timestamp: ts(12) },
    { id: 'tr_5', trace_id: 'trace_mno345', span_id: 'sp_005', operation: 'POST /v1/memory/search', service: 'supermemory', duration: 310, status: 'slow', timestamp: ts(15) },
    { id: 'tr_6', trace_id: 'trace_pqr678', span_id: 'sp_006', operation: 'GET /v1/sessions/:id', service: 'zep', duration: 22, status: 'ok', timestamp: ts(20) },
    { id: 'tr_7', trace_id: 'trace_stu901', span_id: 'sp_007', operation: 'POST /v1/procedures/execute', service: 'openviking', duration: 1240, status: 'error', timestamp: ts(25) },
  ] satisfies TraceSpan[],

  errors: [
    {
      id: 'err_1',
      message: 'Neo4j connection refused: connection pool exhausted',
      service: 'graphiti',
      count: 14,
      lastOccurrence: ts(5),
      stack: 'Error: connect ECONNREFUSED 127.0.0.1:7687\n  at createConnection (neo4j-driver/lib/connection.js:142)',
    },
    {
      id: 'err_2',
      message: 'OpenViking procedure timeout: exceeded 1200ms threshold',
      service: 'openviking',
      count: 3,
      lastOccurrence: ts(25),
      stack: 'TimeoutError: Procedure execution exceeded limit\n  at ProcedureRunner.execute (openviking/runner.go:234)',
    },
    {
      id: 'err_3',
      message: 'Supermemory embedding model rate limit: 429 Too Many Requests',
      service: 'supermemory',
      count: 7,
      lastOccurrence: ts(35),
      stack: 'APIError: 429 rate_limit_exceeded\n  at EmbeddingService.embed (supermemory/src/embedding.ts:55)',
    },
  ] satisfies ErrorEntry[],
};
