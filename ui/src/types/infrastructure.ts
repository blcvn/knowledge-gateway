/**
 * Infrastructure Types — extended with InfraTopology and DeploymentInfo
 * TASK-API-012
 */

import { HealthStatus } from './api';

export interface ServiceInfo {
  name:      string;
  version:   string;
  status:    HealthStatus;
  uptime:    number;
  port?:     number;
  address?:  string;
}

export interface DatabaseHealth {
  name:       string;
  type:       'PostgreSQL' | 'Redis' | 'Neo4j' | 'Qdrant' | 'NATS';
  status:     HealthStatus;
  latency_ms: number;
  host?:      string;
  version?:   string;
}

export interface ResourceMetrics {
  service:          string;
  cpu_usage_pct:    number;
  memory_usage_mb:  number;
  disk_usage_pct:   number;
  pod?:             string;
}

export interface InfraTopology {
  mode:        'monolith' | 'microservices';
  node_count:  number;
  services:    string[];
  deployed_at: string;
}

export interface DeploymentInfo {
  service:     string;
  version:     string;
  deployed_at: string;
  status:      'running' | 'stopped' | 'error';
  replicas:    number;
}
