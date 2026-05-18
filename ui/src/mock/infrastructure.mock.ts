import type { ServiceInfo, DatabaseHealth, ResourceMetrics } from '../types/infrastructure';

export const infrastructureMock = {
  services: [{ name: 'vnp-gateway', version: '1.0.0', status: 'Healthy', uptime: 36000 }] as ServiceInfo[],
  databases: [{ name: 'Postgres', type: 'PostgreSQL', status: 'Healthy', latency_ms: 5 }] as DatabaseHealth[],
  metrics: [{ service: 'vnp-gateway', cpu_usage_pct: 15, memory_usage_mb: 256, disk_usage_pct: 45 }] as ResourceMetrics[],
};
