export interface MetricPoint {
  timestamp: string;
  value: number;
}

export interface MetricsSummary {
  avgLatency: number;
  requestRate: number;
  errorRate: number;
  uptime: string;
  p50: number;
  p95: number;
  p99: number;
}

export interface TraceSpan {
  id?: string;
  trace_id: string;
  span_id: string;
  name?: string;
  operation?: string;
  service: string;
  duration_ms?: number;
  duration?: number;
  status?: string;
  timestamp?: string;
}

export interface ErrorEntry {
  id: string;
  message: string;
  service: string;
  timestamp?: string;
  count?: number;
  lastOccurrence?: string;
  stack?: string;
}
