# TASK-API-002 — TypeScript Types (All Domains)

**Task ID:** TASK-API-002
**Status:** ✅ COMPLETED — 2026-06-17
**Sprint:** 1 — Foundation
**Solution:** All API-SOL-001 → API-SOL-012
**Depends on:** —
**Ước tính:** 1.5h
**Priority:** P0 — Critical

---

## Mục tiêu

Tạo toàn bộ TypeScript type files trong `ui/src/types/` đồng bộ với response schema từ backend.
Đây là **Single Source of Truth** về contract giữa frontend và backend.

---

## Công việc cụ thể

### 1. `ui/src/types/auth.ts`

```typescript
export interface AuthUser {
  id:         string;
  name:       string;
  email:      string;
  role:       string;
  tenant_id:  string;
  avatar_url?: string;
}

export interface LoginRequest {
  email:    string;
  password: string;
}

export interface LoginResponse {
  access_token:  string;
  refresh_token: string;
  expires_in:    number;
  token_type:    'Bearer';
  user:          AuthUser;
}

export interface RefreshResponse {
  access_token: string;
  expires_in:   number;
}
```

### 2. `ui/src/types/dashboard.ts`

```typescript
export interface KPIData {
  activeAgents:        number;
  recallLatencyP50Ms:  number;
  recallLatencyP95Ms:  number;
  contextSavingsPct:   number;
  graphNodesTotal:     number;
  graphEdgesTotal:     number;
  graphGrowth24h:      number;
  errorRatePct:        number;
  activeSessions:      number;
  activeProfiles:      number;
  memoryVersions:      number;
}

export interface EngineHealth {
  name:          string;
  role:          string;
  status:        'Healthy' | 'Warning' | 'Critical';
  latencyP50Ms:  number;
  latencyP95Ms:  number;
  queueDepth:    number;
  uptimeSeconds: number;
  lastCheck:     string;
}

export type ThroughputWindow = '5m' | '15m' | '1h' | '6h' | '24h';

export interface EngineMetrics {
  ingestPerSec:              number;
  recallPerSec:              number;
  embedPerSec:               number;
  profileExtractionsPerSec?: number;
}

export interface ThroughputData {
  window:  ThroughputWindow;
  engines: Record<string, EngineMetrics>;
}

export interface HeatmapPoint {
  x:       number;  // 0-23 (hour)
  y:       number;  // 0-6  (day of week)
  density: number;
}

export interface HeatmapData {
  points:     HeatmapPoint[];
  xLabel:     string;
  yLabel:     string;
  maxDensity: number;
}
```

### 3. `ui/src/types/session.ts`

```typescript
export interface Session {
  id:            string;
  user_id:       string;
  title:         string;
  agent_id?:     string;
  status:        'active' | 'completed' | 'failed';
  message_count: number;
  created_at:    string;
  updated_at:    string;
}

export interface Message {
  id:              string;
  role:            'user' | 'assistant' | 'system' | 'tool';
  content:         string;
  timestamp:       string;
  memory_sources?: string[];
}

export interface Conversation {
  session_id: string;
  messages:   Message[];
}

export interface PaginatedResponse<T> {
  data:      T[];
  total:     number;
  page:      number;
  page_size: number;
  has_more:  boolean;
}

export interface SessionFilters {
  status?:    'active' | 'completed' | 'failed';
  user_id?:   string;
  agent_id?:  string;
  search?:    string;
  sort?:      string;
  page?:      number;
  page_size?: number;
}

export interface SessionTimeline {
  event_type: string;
  engine:     string;
  memory_id:  string;
  timestamp:  string;
  latency_ms: number;
  details:    Record<string, unknown>;
}

export interface SessionDiff {
  session_id: string;
  added:   Array<{ engine: string; memory_id: string; content: string }>;
  updated: Array<{ engine: string; memory_id: string; field: string; before: unknown; after: unknown }>;
  deleted: Array<{ engine: string; memory_id: string }>;
}

export interface WorkingMemory {
  session_id: string;
  summary:    string;
  entities:   string[];
}

export interface UserSummary {
  user_id:        string;
  context_string: string;
  token_count:    number;
}
```

### 4. `ui/src/types/memory.ts`

```typescript
export type SearchMode         = 'semantic' | 'bm25' | 'hybrid' | 'graph';
export type RerankingStrategy  = 'cross_encoder' | 'rrf' | 'none';

export interface MemoryFilters {
  memory_type?: string;
  date_from?:   string;
  date_to?:     string;
  policy_tags?: string[];
}

export interface MemorySearchQuery {
  query:     string;
  mode:      SearchMode;
  engines:   string[];
  filters:   MemoryFilters;
  limit:     number;
  offset:    number;
  reranking: RerankingStrategy;
}

export interface MemoryItem {
  id:               string;
  engine:           string;
  memoryType:       string;
  title:            string;
  summary:          string;
  content:          string;
  score:            number;
  entities:         string[];
  sourceSessions:   string[];
  temporalValidity: { from: string | null; to: string | null };
  policyTags:       string[];
  versionChain:     string | null;
  metadata:         Record<string, unknown>;
}

export interface MemoryFacets {
  byEngine: Record<string, number>;
  byType:   Record<string, number>;
}

export interface MemorySearchResult {
  results:   MemoryItem[];
  total:     number;
  facets:    MemoryFacets;
  latencyMs: number;
}

export interface MemoryVersion {
  id:             string;
  memory_id:      string;
  content:        string;
  version_number: number;
  is_latest:      boolean;
  diff:           string;
  created_at:     string;
}

export const ALL_ENGINES = [
  'graphiti',
  'cognee',
  'memobase',
  'zep',
  'supermemory',
  'openviking',
] as const;

export type EngineId = typeof ALL_ENGINES[number];
```

### 5. `ui/src/types/adaptive.ts`

```typescript
export interface AdaptiveMemory {
  id:         string;
  content:    string;
  source:     string;
  is_latest:  boolean;
  version:    number;
  created_at: string;
  updated_at: string;
}

export interface ExternalConnector {
  id:             string;
  type:           'google_drive' | 'gmail' | 'notion' | 'onedrive' | 'github';
  status:         'Connected' | 'Disconnected' | 'Error';
  document_count: number;
  last_sync?:     string;
}

export interface AdaptiveAnalytics {
  creation_rate:        number;
  deletion_rate:        number;
  contradiction_count:  number;
  connector_sync_count: number;
  storage_usage_bytes:  number;
}

export interface ForgetRules {
  ttl_days:             number;
  inactivity_days:      number;
  low_score_threshold:  number;
  auto_prune:           boolean;
}
```

### 6. `ui/src/types/profile.ts`

```typescript
export interface ProfileEntry {
  topic:      string;
  sub_topic:  string;
  content:    string;
  updated_at: string;
}

export interface UserProfile {
  user_id:    string;
  profiles:   ProfileEntry[];
  created_at: string;
  updated_at: string;
}

export interface BufferZone {
  user_id:     string;
  pending:     string[];
  token_count: number;
  threshold:   number;
  flush_pct:   number;
}

export interface ContextAssembly {
  user_id:                string;
  context_string:         string;
  token_count:            number;
  profile_section_tokens: number;
  event_section_tokens:   number;
  latency_ms:             number;
}

export interface UserEvent {
  id:          string;
  type:        string;
  content:     string;
  timestamp:   string;
  session_id?: string;
}

export interface ProfileConfig {
  flush_threshold:    number;
  buffer_token_limit: number;
  ttl_days:           number;
}
```

### 7. `ui/src/types/governance.ts`

```typescript
export interface Tenant {
  id:         string;
  name:       string;
  slug:       string;
  plan:       'free' | 'pro' | 'enterprise';
  status:     'active' | 'suspended';
  created_at: string;
}

export interface Policy {
  id:         string;
  tenant_id:  string;
  name:       string;
  rego_code:  string;
  scope:      string;
  enabled:    boolean;
  created_at: string;
}

export interface AuditLogEntry {
  id:          string;
  actor_id:    string;
  action:      string;
  entity_type: string;
  entity_id:   string;
  result:      'success' | 'failure';
  created_at:  string;
}

export interface AuditFilters {
  action?:      string;
  actor_id?:    string;
  entity_type?: string;
  from?:        string;
  to?:          string;
}

export interface GDPRPreviewResponse {
  user_id:             string;
  estimated_items:     number;
  breakdown_by_engine: Record<string, number>;
  warnings:            string[];
}
```

### 8. `ui/src/types/observability.ts`

```typescript
export interface MetricPoint {
  timestamp: string;
  value:     number;
  label:     string;
}

export interface MetricsResponse {
  latency:    MetricPoint[];
  error_rate: MetricPoint[];
  throughput: MetricPoint[];
}

export interface TraceSpan {
  id:        string;
  trace_id:  string;
  span_id:   string;
  operation: string;
  service:   string;
  duration:  number;
  status:    'ok' | 'slow' | 'error';
  timestamp: string;
}

export interface ErrorEntry {
  id:             string;
  message:        string;
  service:        string;
  count:          number;
  lastOccurrence: string;
  stack?:         string;
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
```

### 9. `ui/src/types/pipeline.ts`

```typescript
export interface QueueMetrics {
  depth:       number;
  throughput:  number;
  retry_count: number;
}

export interface PipelineJob {
  id:          string;
  engine:      string;
  type:        'ingest' | 'index' | 'sync' | 'cognify';
  status:      'Running' | 'Idle' | 'Failed' | 'Completed';
  progress:    number;
  items_total: number;
  items_done:  number;
  created_at:  string;
  updated_at:  string;
}

export interface PipelineWorker {
  id:     string;
  engine: string;
  status: 'idle' | 'busy' | 'offline';
}

export interface PipelineTemplate {
  id:     string;
  name:   string;
  engine: string;
  config: Record<string, unknown>;
}

export interface PipelineStatus {
  engine:    string;
  status:    'idle' | 'running' | 'paused' | 'error';
  job_count: number;
}
```

### 10. `ui/src/types/infrastructure.ts`

```typescript
export interface ServiceInfo {
  name:    string;
  version: string;
  status:  'Healthy' | 'Warning' | 'Critical';
  uptime:  number;
}

export interface DatabaseHealth {
  name:       string;
  type:       'PostgreSQL' | 'Neo4j' | 'Redis' | 'NATS' | 'MinIO' | 'Qdrant';
  status:     'Healthy' | 'Warning' | 'Critical';
  latency_ms: number;
}

export interface ResourceMetrics {
  service:         string;
  cpu_usage_pct:   number;
  memory_usage_mb: number;
  disk_usage_pct:  number;
}

export interface InfraTopology {
  mode:       'monolith' | 'gateway';
  node_count: number;
  services:   ServiceInfo[];
}

export interface DeploymentInfo {
  service:    string;
  version:    string;
  git_commit: string;
  started_at: string;
}
```

### 11. `ui/src/types/org.ts`

```typescript
export interface OrgSettings {
  name:                  string;
  slug:                  string;
  timezone:              string;
  max_agents:            number;
  max_memories_per_user: number;
}

export interface OrgMember {
  id:          string;
  name:        string;
  email:       string;
  role:        'owner' | 'admin' | 'editor' | 'viewer';
  avatar_url?: string;
  joined_at:   string;
}

export interface OrgRole {
  id:          string;
  name:        string;
  permissions: string[];
}

export interface APIKey {
  id:            string;
  name:          string;
  prefix:        string;
  permissions:   string[];
  expires_at?:   string;
  created_at:    string;
  last_used_at?: string;
}

export interface CreateKeyResponse {
  key:     APIKey;
  raw_key: string;  // Shown ONCE
}

export interface RateLimitConfig {
  scope:     'global' | 'per_key' | 'per_endpoint';
  rps:       number;
  rpm:       number;
  burst:     number;
  tier_name: 'free' | 'pro' | 'enterprise';
}

export interface Webhook {
  id:           string;
  url:          string;
  events:       string[];
  status:       'active' | 'disabled';
  success_rate: number;
  created_at:   string;
}
```

---

## Files tạo ra

```
ui/src/types/
├── auth.ts           ← NEW
├── dashboard.ts      ← NEW
├── session.ts        ← NEW
├── memory.ts         ← NEW
├── adaptive.ts       ← NEW
├── profile.ts        ← NEW
├── governance.ts     ← NEW
├── observability.ts  ← NEW
├── pipeline.ts       ← NEW
├── infrastructure.ts ← NEW
└── org.ts            ← NEW
```

---

## Acceptance Criteria

- [x] `npx tsc --noEmit` không lỗi với toàn bộ type files
- [x] Không có `any` type (dùng `unknown` hoặc type cụ thể)
- [x] `PaginatedResponse<T>` là generic type dùng chung
- [x] `ALL_ENGINES` là readonly const tuple
- [x] Tất cả union types dùng string literals thay vì string

---

## Sau khi hoàn thành

```bash
cd ui && npx tsc --noEmit
# → 0 errors → chuyển sang TASK-API-003
```
