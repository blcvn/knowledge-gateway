/**
 * Observability Types — extended with MetricsResponse, CostEntry, TraceFilters
 * TASK-API-010
 */

export interface MetricPoint {
  timestamp: string;
  value:     number;
  label?:    string;  // "p95" | "error_rate" | "throughput"
}

export interface MetricsResponse {
  latency:    MetricPoint[];
  error_rate: MetricPoint[];
  throughput: MetricPoint[];
}

/** Legacy summary shape — kept for backward compat */
export interface MetricsSummary {
  avgLatency:  number;
  requestRate: number;
  errorRate:   number;
  uptime:      string;
  p50:         number;
  p95:         number;
  p99:         number;
}

export interface TraceSpan {
  id?:          string;
  trace_id:     string;
  span_id:      string;
  name?:        string;
  operation?:   string;
  service:      string;
  duration_ms?: number;
  duration?:    number;
  status?:      'ok' | 'slow' | 'error' | string;
  timestamp?:   string;
}

export interface ErrorEntry {
  id:              string;
  message:         string;
  service:         string;
  count?:          number;
  timestamp?:      string;
  lastOccurrence?: string;
  stack?:          string;
}

export interface CostEntry {
  model:         string;
  engine:        string;
  tokens_input:  number;
  tokens_output: number;
  cost_usd:      number;
  date:          string;
}

export interface TraceFilters {
  service?:   string;
  status?:    'ok' | 'slow' | 'error';
  operation?: string;
  from?:      string;
  to?:        string;
}
