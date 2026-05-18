import { HealthStatus } from './api';

export interface ServiceInfo {
  name: string;
  version: string;
  status: HealthStatus;
  uptime: number;
}

export interface DatabaseHealth {
  name: string;
  type: 'PostgreSQL' | 'Redis' | 'Neo4j' | 'Qdrant' | 'NATS';
  status: HealthStatus;
  latency_ms: number;
}

export interface ResourceMetrics {
  service: string;
  cpu_usage_pct: number;
  memory_usage_mb: number;
  disk_usage_pct: number;
}
