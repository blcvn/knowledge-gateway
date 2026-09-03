# Frontend ↔ Backend API Specifications

> **Version**: 2.0.0 — Derived from `ui/src/services/*` + `ui/src/types/*` (2026-06-18)  
> **Source of truth for frontend**: `ui/src/services/*.ts` (real HTTP calls, no mocks)  
> **Cross-referenced with**: `specs/backend-api-specs.md` (gateway router)  
> **Coverage**: 100% — Every API call the frontend makes is documented here with typed request/response schemas.

---

## Overview

| Domain | Base Path | Auth Required | Backend Service |
|--------|-----------|---------------|-----------------|
| Auth | `/v1/auth` | Partial (login is public) | `vnp-auth` (❌ Missing — see §14) |
| Dashboard | `/v1/console/dashboard` | ✅ `admin` | `vnp-dashboard` |
| Memory Explorer | `/v1/console/memory` | ✅ `admin` | `vnp-search-hub` |
| Graph Studio | `/v1/console/graph` | ✅ `admin` | `graphiti-store` / `cognee-search` |
| User Profiles | `/v1/console/profiles` | ✅ `admin` | `memobase-context` |
| Adaptive Memory | `/v1/console/adaptive` | ✅ `admin` | `sm-memory` / `sm-connector` |
| Sessions | `/v1/console/sessions` | ✅ `admin` | `zep-core` |
| Governance | `/v1/console/governance` | ✅ `super_admin` | `vnp-admin` / `vnp-event` |
| Pipelines | `/v1/console/pipelines` | ✅ `admin` | `vnp-pipelines` |
| Infrastructure | `/v1/console/infra` | ✅ `admin` | `vnp-infra` |
| Observability | `/v1/console/observability` | ✅ `admin` | `vnp-observability` |
| Org Settings | `/v1/console/org` | ✅ `admin` | ❌ Missing — see §14 |
| SDK / API Keys | `/v1/console/sdk` | ✅ `admin` | ❌ Missing — see §14 |
| WebSocket | `/v1/console/ws` | ✅ JWT in query | `vnp-gateway` |

---

## Common Types

```typescript
// Shared base types — from ui/src/types/api.ts

type EngineType = 'cognee' | 'graphiti' | 'zep' | 'openviking' | 'memobase' | 'supermemory' | 'kgs';
type MemoryType = 'episodic' | 'semantic' | 'conversational' | 'procedural' | 'profile' | 'adaptive';
type HealthStatus = 'Healthy' | 'Warning' | 'Critical';
type PipelineStatus = 'Running' | 'Completed' | 'Failed' | 'Queued';
type SearchMode = 'semantic' | 'bm25' | 'hybrid' | 'graph';
type RerankingStrategy = 'cross_encoder' | 'rrf' | 'none';

interface PaginatedResponse<T> {
  data:      T[];
  total:     number;
  page:      number;
  page_size: number;   // snake_case from backend
  pageSize:  number;   // camelCase alias (frontend compat)
  has_more:  boolean;
  hasMore:   boolean;  // camelCase alias
}

interface ApiErrorResponse {
  message: string;
  code:    string;
  status:  number;
  details?: Record<string, unknown>;
}
```

---

## HTTP Client — Global Behaviour

> Source: `ui/src/lib/api-client.ts`

| Behaviour | Detail |
|-----------|--------|
| **Base URL** | `VITE_API_BASE_URL` env var (default: `http://localhost:8080`) |
| **Auth header** | `Authorization: Bearer <access_token>` (from `localStorage`) |
| **Tenant header** | `X-Tenant-ID: <tenant_id>` (from `localStorage`) |
| **Token refresh** | Auto-retry on `401` using `POST /v1/auth/refresh` |
| **Error format** | `ApiErrorResponse` (`message`, `code`, `status`) |

**LocalStorage keys used by the client:**

| Key | Value |
|-----|-------|
| `access_token` | JWT Bearer token |
| `refresh_token` | Refresh token |
| `tenant_id` | Tenant identifier |

---

## 1. Auth API

**Base Path:** `/v1/auth`  
**Backend Status:** ❌ Not yet implemented in gateway (see §14 Known Gaps)  
**Source:** `ui/src/services/auth.ts`, `ui/src/services/auth-api.service.ts`

### Types

```typescript
// ui/src/types/auth.ts

interface LoginRequest {
  email:    string;
  password: string;
}

interface AuthUser {
  id:          string;
  name:        string;
  email:       string;
  role:        string;        // e.g., "admin", "super_admin"
  tenant_id:   string;
  avatar_url?: string;
}

interface LoginResponse {
  access_token:  string;      // JWT RS256 — stored in localStorage
  refresh_token: string;      // stored in localStorage
  expires_in:    number;      // seconds
  token_type:    'Bearer';
  user:          AuthUser;    // tenant_id extracted and stored separately
}

interface RefreshResponse {
  access_token: string;
  expires_in:   number;
}
```

### Endpoints

| Method | Path | Request Body | Response | Description |
|--------|------|-------------|----------|-------------|
| `POST` | `/v1/auth/login` | `LoginRequest` | `LoginResponse` | Authenticate user. Tokens stored in `localStorage`. |
| `POST` | `/v1/auth/logout` | `{ refresh_token: string }` | `void` | Invalidate session server-side. Always clears `localStorage`. |
| `GET` | `/v1/auth/me` | — | `AuthUser` | Returns current user from JWT context. |
| `POST` | `/v1/auth/refresh` | `{ refresh_token: string }` | `RefreshResponse` | Exchange refresh token for new access token. Called automatically on 401. |

> **Note:** `loginWithGoogle()` and `register()` are stubbed — they throw immediately. No SSO backend exists.

---

## 2. Dashboard API

**Base Path:** `/v1/console/dashboard`  
**Backend Service:** `vnp-dashboard`  
**Source:** `ui/src/services/dashboard.service.ts`, `ui/src/types/dashboard.ts`

### Types

```typescript
// ui/src/types/dashboard.ts

interface KPIData {
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

interface EngineHealth {
  name:           EngineType;
  role:           string;
  status:         HealthStatus;   // 'Healthy' | 'Warning' | 'Critical'
  latencyP50Ms:   number;
  latencyP95Ms:   number;
  queueDepth:     number;
  uptimeSeconds:  number;
  lastCheck:      string;         // ISO 8601
}

interface MemoryFlowMetrics {
  ingestPerSec:              number;
  recallPerSec:              number;
  embedPerSec:               number;
  profileExtractionsPerSec?: number;
  queueBacklog?:             number;
}

interface ThroughputData {
  window:  string;                                // e.g., "1h", "5m", "24h"
  engines: Record<EngineType, MemoryFlowMetrics>;
}

interface HeatmapData {
  points: Array<{ x: number; y: number; density: number }>;
}
```

### Endpoints

| Method | Path | Query Params | Response | Description |
|--------|------|-------------|----------|-------------|
| `GET` | `/v1/console/dashboard/metrics` | — | `KPIData` | KPI cards: agents, latency, savings, errors, sessions, profiles |
| `GET` | `/v1/console/dashboard/health` | — | `EngineHealth[]` | Aggregated engine health for all 7 engines |
| `GET` | `/v1/console/dashboard/throughput` | `window` = `5m` \| `1h` \| `24h` (default `1h`) | `ThroughputData` | Per-engine memory flow metrics |
| `GET` | `/v1/console/dashboard/heatmap` | — | `HeatmapData` | Memory density heatmap data points |

---

## 3. Memory Explorer API

**Base Path:** `/v1/console/memory`  
**Backend Service:** `vnp-search-hub` (search/recall), `sm-memory` (versions)  
**Source:** `ui/src/services/memory.service.ts`, `ui/src/types/memory.ts`

### Types

```typescript
// ui/src/types/memory.ts

interface MemoryFilters {
  memory_type?: string;
  date_from?:   string;  // ISO 8601
  date_to?:     string;
  policy_tags?: string[];
}

interface MemorySearchQuery {
  query:     string;
  mode:      SearchMode;      // 'semantic' | 'bm25' | 'hybrid' | 'graph'
  engines:   EngineType[];
  filters:   MemoryFilters;
  limit:     number;
  offset:    number;
  reranking: RerankingStrategy;
}

interface MemoryItem {
  id:               string;    // Format: "engine:local_id" — must be URL-encoded
  engine:           EngineType;
  memoryType:       MemoryType;
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

interface MemorySearchResult {
  results:   MemoryItem[];
  total:     number;
  facets: {
    byEngine: Record<string, number>;
    byType:   Record<string, number>;
  };
  latencyMs: number;
}

interface MemoryVersion {
  id:             string;
  memory_id:      string;
  content:        string;
  version_number: number;
  is_latest:      boolean;
  diff:           string;    // unified diff string
  created_at:     string;    // ISO 8601
}
```

### Endpoints

| Method | Path | Body / Query | Response | Description |
|--------|------|-------------|----------|-------------|
| `POST` | `/v1/console/memory/search` | `MemorySearchQuery` | `MemorySearchResult` | Unified cross-engine memory search with reranking |
| `GET` | `/v1/console/memory/{id}` | — | `MemoryItem` | Memory detail with provenance. **ID must be `encodeURIComponent()`-encoded** (e.g., `graphiti:ep_abc` → `graphiti%3Aep_abc`) |
| `GET` | `/v1/console/memory/{id}/neighbors` | `strategy` = `semantic`\|`graph`\|`temporal` (default `semantic`), `limit` = number (default `10`) | `MemorySearchResult` | Graph or semantic neighbors |
| `GET` | `/v1/console/memory/{id}/versions` | — | `MemoryVersion[]` | Version history — **only available for Supermemory** (ids starting with `sm:`) |

---

## 4. Graph Studio API

**Base Path:** `/v1/console/graph`  
**Backend Services:** `graphiti-store` (subgraph/timeline/query/entity), `cognee-search` (ontology)  
**Source:** `ui/src/services/graph.service.ts`, `ui/src/types/graph.ts`

### Types

```typescript
// ui/src/types/graph.ts

interface GraphNode {
  id:          string;
  label:       string;
  type:        string;
  properties?: Record<string, unknown>;
}

interface GraphEdge {
  id:          string;
  source:      string;
  target:      string;
  type:        string;
  properties?: Record<string, unknown>;
}

interface SubgraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

interface OntologySchema {
  classes:       string[];
  relationships: string[];
  properties:    Record<string, string[]>;
}
```

### Endpoints

| Method | Path | Body | Response | Description |
|--------|------|------|----------|-------------|
| `POST` | `/v1/console/graph/subgraph` | `Record<string, string>` (query params) | `SubgraphData` | Query subgraph by entity name/id |
| `GET` | `/v1/console/graph/entity/{id}` | — | `Record<string, unknown>` | Entity detail with neighbors |
| `POST` | `/v1/console/graph/timeline` | `Record<string, string>` (query params) | `unknown[]` | Temporal subgraph for a time range |
| `GET` | `/v1/console/graph/ontology` | — | `OntologySchema` | Fetch current ontology schema |
| `PUT` | `/v1/console/graph/ontology` | `OntologySchema` | `OntologySchema` | Update ontology schema (saved to `cognee-search`) |
| `POST` | `/v1/console/graph/query` | `{ query: string }` (Cypher/NL) | `SubgraphData` | Execute Cypher or natural language query |

---

## 5. User Profiles API

**Base Path:** `/v1/console/profiles`  
**Backend Services:** `memobase-context` (profiles/context/config), `memobase-ingestion` (buffers), `vnp-event` (events)  
**Source:** `ui/src/services/profile.service.ts`, `ui/src/types/profile.ts`

### Types

```typescript
// ui/src/types/profile.ts

interface ProfileEntry {
  topic:       string;
  sub_topic:   string;
  content:     string;
  confidence?: number;
}

interface UserProfile {
  user_id:    string;
  profiles:   ProfileEntry[];
  created_at: string;
  updated_at: string;
}

interface ProfileSchemaEntry {
  topic:        string;
  sub_topic:    string;
  description?: string;
}

interface ProfileConfig {
  profiles:    ProfileSchemaEntry[];
  strict_mode: boolean;
}

interface BufferZone {
  user_id:         string;
  buffer_type:     string;
  token_count:     number;
  token_threshold: number;
  idle_timeout:    string;
  last_flush:      string;
  flush_count:     number;
}

interface UserEvent {
  id:         string;
  user_id:    string;
  gist:       string;
  tags:       string[];
  created_at: string;
  embedding?: number[];
}

interface ContextAssembly {
  user_id:                string;
  context_string:         string;
  token_count:            number;
  profile_section_tokens: number;
  event_section_tokens:   number;
  latency_ms:             number;
}
```

### Endpoints

| Method | Path | Body | Response | Description |
|--------|------|------|----------|-------------|
| `GET` | `/v1/console/profiles` | — | `UserProfile[]` | List all user profiles |
| `GET` | `/v1/console/profiles/{user_id}` | — | `UserProfile` | Profile detail (topics + entries) |
| `GET` | `/v1/console/profiles/{user_id}/buffers` | — | `BufferZone` | Working memory buffer state (token count, flush timing) |
| `GET` | `/v1/console/profiles/{user_id}/context` | — | `ContextAssembly` | Context assembly preview (profile + events → assembled string) |
| `GET` | `/v1/console/profiles/{user_id}/events` | — | `UserEvent[]` | User event timeline |
| `GET` | `/v1/console/profiles/config` | — | `ProfileConfig` | Profile schema config (topics list + strict mode) |
| `PUT` | `/v1/console/profiles/config` | `Partial<ProfileConfig>` | `ProfileConfig` | Update profile schema (partial update supported) |

---

## 6. Adaptive Memory API

**Base Path:** `/v1/console/adaptive`  
**Backend Services:** `sm-memory` (memories/versions/analytics/forget-rules), `sm-connector` (connectors)  
**Source:** `ui/src/services/adaptive.service.ts`, `ui/src/types/adaptive.ts`

### Types

```typescript
// ui/src/types/adaptive.ts

interface AdaptiveMemory {
  id:               string;
  content:          string;
  memory_type:      'static' | 'dynamic';
  is_latest:        boolean;
  parent_id?:       string;
  root_id?:         string;
  relation_type?:   'updates' | 'extends' | 'derives';
  created_at:       string;
  updated_at:       string;
  forget_after?:    string;          // e.g., "30d"
  metadata?:        Record<string, unknown>;
}

interface MemoryVersion {            // (adaptive version — same shape as memory version)
  id:             string;
  memory_id:      string;
  content:        string;
  version_number: number;
  is_latest:      boolean;
  diff?:          string;
  created_at:     string;
}

interface ForgetRule {
  id:                       string;
  memory_type:              'static' | 'dynamic';
  forget_after:             string;  // e.g., "30d", "90d"
  noise_filter:             boolean;
  contradiction_resolution: 'keep_latest' | 'keep_both' | 'manual';
}

interface ExternalConnector {
  id:              string;
  type:            'google_drive' | 'gmail' | 'notion' | 'onedrive' | 'github';
  status:          'Connected' | 'Disconnected' | 'Error';
  last_sync:       string | null;
  document_count:  number;
  sync_frequency:  string;
  error_message?:  string;
}

interface AdaptiveAnalytics {
  creation_rate:         number;
  deletion_rate:         number;
  contradiction_count:   number;
  connector_sync_count:  number;
  storage_usage_bytes:   number;
}
```

### Endpoints

| Method | Path | Body | Response | Description |
|--------|------|------|----------|-------------|
| `GET` | `/v1/console/adaptive/memories` | — | `AdaptiveMemory[]` | List adaptive memories |
| `GET` | `/v1/console/adaptive/memories/{id}/versions` | — | `MemoryVersion[]` | Version chain for one adaptive memory. ID is `encodeURIComponent`-encoded |
| `GET` | `/v1/console/adaptive/connectors` | — | `ExternalConnector[]` | List external connectors (Google Drive, Gmail, Notion, etc.) |
| `POST` | `/v1/console/adaptive/connectors` | `Partial<ExternalConnector>` | `ExternalConnector` | Create a new connector |
| `POST` | `/v1/console/adaptive/connectors/{id}/sync` | `{}` | `{ job_id: string }` | Trigger sync job. Returns `job_id` for status polling |
| `GET` | `/v1/console/adaptive/analytics` | — | `AdaptiveAnalytics` | Creation rate, contradiction count, storage usage |
| `GET` | `/v1/console/adaptive/forget-rules` | — | `ForgetRule[]` | Fetch auto-forget rules |
| `PUT` | `/v1/console/adaptive/forget-rules` | `ForgetRule[]` | `ForgetRule[]` | Replace all forget rules in bulk |

---

## 7. Sessions API

**Base Path:** `/v1/console/sessions`  
**Backend Services:** `zep-core` (sessions/messages), `vnp-event` (diff), `ov-session` (working-memory), `memobase-context` (user-summary)  
**Source:** `ui/src/services/session.service.ts`, `ui/src/types/session.ts`

### Types

```typescript
// ui/src/types/session.ts

interface Session {
  id:            string;
  user_id:       string;
  title:         string;
  agent_id?:     string;
  status:        'active' | 'completed' | 'failed';
  message_count: number;
  created_at:    string;
  updated_at:    string;
}

interface Message {
  id:              string;
  role:            'user' | 'assistant' | 'system' | 'tool';
  content:         string;
  timestamp:       string;
  memory_sources?: string[];
}

interface Conversation {
  session_id: string;
  messages:   Message[];
}

interface WorkingMemory {
  session_id: string;
  summary:    string;
  entities:   string[];
}

interface UserSummary {
  user_id:        string;
  context_string: string;
  token_count:    number;
}

interface SessionTimeline {
  event_type: string;
  engine:     string;
  memory_id:  string;
  timestamp:  string;
  latency_ms: number;
  details:    Record<string, unknown>;
}

interface SessionDiff {
  session_id: string;
  added:   Array<{ engine: string; memory_id: string; content: string }>;
  updated: Array<{ engine: string; memory_id: string; field: string; before: unknown; after: unknown }>;
  deleted: Array<{ engine: string; memory_id: string }>;
}

interface SessionFilters {
  status?:    'active' | 'completed' | 'failed';
  user_id?:   string;
  agent_id?:  string;
  search?:    string;
  sort?:      string;
  page?:      number;
  page_size?: number;
}
```

### Endpoints

| Method | Path | Query Params / Body | Response | Description |
|--------|------|---------------------|----------|-------------|
| `GET` | `/v1/console/sessions` | `status`, `user_id`, `agent_id`, `search`, `sort`, `page`, `page_size` | `PaginatedResponse<Session>` | Paginated list of sessions with filtering |
| `GET` | `/v1/console/sessions/live` | — | `Session[]` | Currently active live sessions |
| `GET` | `/v1/console/sessions/{id}` | — | `Conversation` | Full conversation with all messages |
| `GET` | `/v1/console/sessions/{id}/timeline` | — | `SessionTimeline[]` | Ordered timeline of memory events during session |
| `GET` | `/v1/console/sessions/{id}/diff` | — | `SessionDiff` | Memory diff (added/updated/deleted) per engine |
| `GET` | `/v1/console/sessions/{id}/working-memory` | — | `WorkingMemory` | Current working memory state (summary + entities) |
| `GET` | `/v1/console/sessions/{id}/user-summary` | — | `UserSummary` | Generated user memory summary |

---

## 8. Governance API

**Base Path:** `/v1/console/governance`  
**Backend Services:** `vnp-admin` (tenants/policies/audit), `vnp-event` (GDPR)  
**Auth:** `super_admin` role required  
**Source:** `ui/src/services/governance.service.ts`, `ui/src/types/governance.ts`

### Types

```typescript
// ui/src/types/governance.ts

interface Tenant {
  id:         string;
  name:       string;
  slug?:      string;
  plan?:      'free' | 'pro' | 'enterprise';
  created_at: string;
  status:     'Active' | 'Suspended' | 'active' | 'suspended';
}

interface Policy {
  id:           string;
  name:         string;
  description?: string;
  rego_code:    string;    // OPA Rego policy code
  scope:        string;
  enabled:      boolean;
  tenant_id?:   string;
  created_at?:  string;
}

interface AuditLogEntry {
  id:          string;
  tenant_id:   string;
  actor_id:    string;
  action:      string;
  entity_type: string;
  entity_id?:  string;
  result:      string;
  created_at:  string;
}

interface AuditFilters {
  action?:      string;
  actor_id?:    string;
  entity_type?: string;
  from?:        string;  // ISO 8601
  to?:          string;
}

interface GDPRPreviewResponse {
  user_id:             string;
  estimated_items:     number;
  breakdown_by_engine: Record<string, number>;
  warnings:            string[];
}

interface GDPRForgetResponse {
  success:       boolean;
  deleted_count: number;
}
```

### Endpoints

| Method | Path | Body / Query | Response | Description |
|--------|------|-------------|----------|-------------|
| `GET` | `/v1/console/governance/tenants` | — | `Tenant[]` | List all tenants |
| `POST` | `/v1/console/governance/tenants` | `Partial<Tenant>` | `Tenant` | Create a new tenant |
| `PUT` | `/v1/console/governance/tenants/{id}` | `Partial<Tenant>` | `Tenant` | Update tenant details |
| `GET` | `/v1/console/governance/policies` | — | `Policy[]` | List all OPA policies |
| `POST` | `/v1/console/governance/policies` | `Partial<Policy>` | `Policy` | Create new policy |
| `PUT` | `/v1/console/governance/policies/{id}` | `Partial<Policy>` | `Policy` | Update existing policy |
| `GET` | `/v1/console/governance/audit` | `action`, `actor_id`, `entity_type`, `from`, `to` | `AuditLogEntry[]` | Search audit logs with filters |
| `POST` | `/v1/console/governance/gdpr/forget/preview` | `{ user_id: string }` | `GDPRPreviewResponse` | **Step 1**: Dry-run GDPR forget — shows what would be deleted |
| `POST` | `/v1/console/governance/gdpr/forget` | `{ user_id: string }` | `GDPRForgetResponse` | **Step 2**: Execute cascading deletion (requires UI confirmation) |

---

## 9. Pipelines API

**Base Path:** `/v1/console/pipelines`  
**Backend Service:** `vnp-pipelines`  
**Source:** `ui/src/services/pipeline.service.ts`, `ui/src/types/pipeline.ts`

### Types

```typescript
// ui/src/types/pipeline.ts

interface PipelineJob {
  id:         string;
  engine:     EngineType;
  status:     PipelineStatus;  // 'Running' | 'Completed' | 'Failed' | 'Queued'
  progress:   number;          // 0–100
  created_at: string;
  updated_at: string;
}

interface PipelineStage {
  name:        string;
  status:      PipelineStatus;
  duration_ms: number;
}

interface QueueMetrics {
  depth:       number;
  throughput:  number;
  retry_count: number;
}

interface WorkerStatus {
  engine:  EngineType;
  running: number;
  idle:    number;
}
```

### Endpoints

| Method | Path | Response | Description |
|--------|------|----------|-------------|
| `GET` | `/v1/console/pipelines/status` | `unknown` | All engines pipeline overview |
| `GET` | `/v1/console/pipelines/queues` | `QueueMetrics` | Queue metrics (depth, throughput, retries) |
| `GET` | `/v1/console/pipelines/workers` | `WorkerStatus[]` | Active workers per engine |
| `GET` | `/v1/console/pipelines/templates` | `unknown[]` | Available pipeline templates |
| `GET` | `/v1/console/pipelines/{engine}` | `unknown` | Engine-specific pipeline overview |
| `GET` | `/v1/console/pipelines/{engine}/jobs` | `PipelineJob[]` | Jobs for a specific engine |
| `GET` | `/v1/console/pipelines/{engine}/jobs/{id}` | `PipelineJob` | Detailed job info with stages |

> **Note:** `{engine}` is one of the `EngineType` values.

---

## 10. Infrastructure API

**Base Path:** `/v1/console/infra`  
**Backend Service:** `vnp-infra`  
**Source:** `ui/src/services/infrastructure.service.ts`, `ui/src/types/infrastructure.ts`

### Types

```typescript
// ui/src/types/infrastructure.ts

interface ServiceInfo {
  name:     string;
  version:  string;
  status:   HealthStatus;
  uptime:   number;       // seconds
  port?:    number;
  address?: string;
}

interface DatabaseHealth {
  name:       string;
  type:       'PostgreSQL' | 'Redis' | 'Neo4j' | 'Qdrant' | 'NATS';
  status:     HealthStatus;
  latency_ms: number;
  host?:      string;
  version?:   string;
}

interface ResourceMetrics {
  service:         string;
  cpu_usage_pct:   number;
  memory_usage_mb: number;
  disk_usage_pct:  number;
  pod?:            string;
}

interface InfraTopology {
  mode:        'monolith' | 'microservices';
  node_count:  number;
  services:    string[];
  deployed_at: string;
}

interface DeploymentInfo {
  service:     string;
  version:     string;
  deployed_at: string;
  status:      'running' | 'stopped' | 'error';
  replicas:    number;
}
```

### Endpoints

| Method | Path | Response | Description |
|--------|------|----------|-------------|
| `GET` | `/v1/console/infra/topology` | `InfraTopology` | Service topology (mode, node count, services list) |
| `GET` | `/v1/console/infra/services` | `ServiceInfo[]` | All services health status |
| `GET` | `/v1/console/infra/services/{name}` | `ServiceInfo` | Single service detail. `{name}` is `encodeURIComponent`-encoded |
| `GET` | `/v1/console/infra/databases` | `DatabaseHealth[]` | DB health: PG, Neo4j, Redis, Qdrant, NATS |
| `GET` | `/v1/console/infra/resources` | `ResourceMetrics[]` | CPU/memory/disk usage per service/pod |
| `GET` | `/v1/console/infra/deployments` | `DeploymentInfo[]` | Deployment history with version and replica count |

---

## 11. Observability API

**Base Path:** `/v1/console/observability`  
**Backend Service:** `vnp-observability`  
**Source:** `ui/src/services/observability.service.ts`, `ui/src/types/observability.ts`

### Types

```typescript
// ui/src/types/observability.ts

interface MetricPoint {
  timestamp: string;
  value:     number;
  label?:    string;  // "p95" | "error_rate" | "throughput"
}

interface MetricsResponse {
  latency:    MetricPoint[];
  error_rate: MetricPoint[];
  throughput: MetricPoint[];
}

interface TraceSpan {
  id?:          string;
  trace_id:     string;
  span_id:      string;
  name?:        string;
  operation?:   string;
  service:      string;
  duration_ms?: number;
  duration?:    number;   // alias
  status?:      'ok' | 'slow' | 'error' | string;
  timestamp?:   string;
}

interface ErrorEntry {
  id:              string;
  message:         string;
  service:         string;
  count?:          number;
  timestamp?:      string;
  lastOccurrence?: string;
  stack?:          string;
}

interface CostEntry {
  model:         string;    // LLM model name
  engine:        string;
  tokens_input:  number;
  tokens_output: number;
  cost_usd:      number;
  date:          string;
}

interface TraceFilters {
  service?:   string;
  status?:    'ok' | 'slow' | 'error';
  operation?: string;
  from?:      string;  // ISO 8601
  to?:        string;
}
```

### Endpoints

| Method | Path | Query Params | Response | Description |
|--------|------|-------------|----------|-------------|
| `GET` | `/v1/console/observability/metrics` | — | `MetricsResponse` | Time-series metrics: latency[], error_rate[], throughput[] |
| `GET` | `/v1/console/observability/traces` | `service`, `status`, `operation`, `from`, `to` | `TraceSpan[]` | Distributed trace spans with filters |
| `GET` | `/v1/console/observability/traces/{id}` | — | `TraceSpan` | Single trace span detail |
| `GET` | `/v1/console/observability/errors` | `service` | `ErrorEntry[]` | Error explorer, optionally filtered by service |
| `GET` | `/v1/console/observability/costs` | — | `CostEntry[]` | LLM cost analytics (model, engine, tokens, USD) |

---

## 12. Org Settings & SDK API

**Base Paths:** `/v1/console/org`, `/v1/console/sdk`  
**Backend Status:** ❌ Not yet implemented in gateway (see §14 Known Gaps)  
**Source:** `ui/src/services/org.service.ts`, `ui/src/types/org.ts`

### Types

```typescript
// ui/src/types/org.ts

interface OrgSettings {
  name:                  string;
  slug:                  string;
  domain?:               string;
  timezone:              string;
  max_agents:            number;
  max_memories_per_user: number;
  plan?:                 'free' | 'pro' | 'enterprise';
}

interface OrgMember {
  id:        string;
  name:      string;
  email:     string;
  role:      string;
  status:    'active' | 'inactive';
  joined_at: string;
}

interface OrgRole {
  id:          string;
  name:        string;
  permissions: string[];
}

interface APIKey {
  id:          string;
  name:        string;
  prefix:      string;    // visible prefix e.g. "vnp_prod_sk_3f9a..."
  scopes:      string[];
  created_at:  string;
  last_used?:  string;
  expires_at?: string;
  status:      'active' | 'revoked' | 'expired';
}

// IMPORTANT: raw_key is shown ONLY ONCE — UI must display immediately
interface CreateKeyResponse {
  key:     APIKey;
  raw_key: string;
}

interface CreateKeyPayload {
  name:             string;
  permissions:      string[];
  expires_in_days?: number;
}

interface RateLimitConfig {
  scope:  string;
  rps:    number;
  rpm:    number;
  burst:  number;
  tier:   'enterprise' | 'standard' | 'restricted';
}

interface Webhook {
  id:           string;
  url:          string;
  events:       string[];
  status:       'active' | 'paused' | 'failed';
  secret?:      string;
  success_rate: number;
  created_at:   string;
}

interface CreateWebhookPayload {
  url:     string;
  events:  string[];
  secret?: string;
}
```

### Endpoints — Org (`/v1/console/org`)

| Method | Path | Body | Response | Description |
|--------|------|------|----------|-------------|
| `GET` | `/v1/console/org/settings` | — | `OrgSettings` | Get organization settings |
| `PUT` | `/v1/console/org/settings` | `Partial<OrgSettings>` | `OrgSettings` | Update organization settings |
| `GET` | `/v1/console/org/members` | — | `OrgMember[]` | List org members |
| `GET` | `/v1/console/org/roles` | — | `OrgRole[]` | List available roles and permissions |

### Endpoints — SDK (`/v1/console/sdk`)

| Method | Path | Body | Response | Description |
|--------|------|------|----------|-------------|
| `GET` | `/v1/console/sdk/keys` | — | `APIKey[]` | List API keys. **`raw_key` NOT included** — only masked prefix |
| `POST` | `/v1/console/sdk/keys` | `CreateKeyPayload` | `CreateKeyResponse` | Create API key. **`raw_key` returned ONCE** — UI must show immediately |
| `DELETE` | `/v1/console/sdk/keys/{id}` | — | `void` | Revoke API key permanently |
| `GET` | `/v1/console/sdk/rate-limits` | — | `RateLimitConfig[]` | Get rate limit configs per scope/tier |
| `GET` | `/v1/console/sdk/webhooks` | — | `Webhook[]` | List configured webhooks |
| `POST` | `/v1/console/sdk/webhooks` | `CreateWebhookPayload` | `Webhook` | Create new webhook |
| `DELETE` | `/v1/console/sdk/webhooks/{id}` | — | `void` | Delete webhook |

---

## 13. WebSocket — Realtime Channel

**Path:** `/v1/console/ws?token=<jwt>`  
**Protocol:** WebSocket (authenticated via query param `token`)

### Channels

| Channel | Payload | Description |
|---------|---------|-------------|
| `engine.health` | `EngineHealth` | Real-time engine health updates |
| `memory.flow` | `MemoryFlowMetrics` | Ingest/recall throughput metrics |
| `pipeline.progress` | `{ job_id: string; progress: number; status: PipelineStatus }` | Pipeline job progress |
| `alerts` | `{ level: 'warning' \| 'critical'; message: string; service: string }` | System alerts |

> **Frontend usage:** Dashboard page subscribes to `engine.health` and `memory.flow` for live graphs.

---

## 14. Known Gaps — Missing Backend Endpoints

> These endpoints are **called by the frontend** but **not yet implemented** in the gateway router.  
> See [CR-001](../crs/v2/api-update/CR-001-frontend-api-alignment.md) for the Change Request.

| Status | Method | Path | Section |
|--------|--------|------|---------|
| ❌ Missing | `POST` | `/v1/auth/login` | §1 Auth |
| ❌ Missing | `POST` | `/v1/auth/logout` | §1 Auth |
| ❌ Missing | `GET` | `/v1/auth/me` | §1 Auth |
| ❌ Missing | `POST` | `/v1/auth/refresh` | §1 Auth |
| ❌ Missing | `GET` | `/v1/console/org/settings` | §12 Org |
| ❌ Missing | `PUT` | `/v1/console/org/settings` | §12 Org |
| ❌ Missing | `GET` | `/v1/console/org/members` | §12 Org |
| ❌ Missing | `GET` | `/v1/console/org/roles` | §12 Org |
| ❌ Missing | `GET` | `/v1/console/sdk/keys` | §12 SDK |
| ❌ Missing | `POST` | `/v1/console/sdk/keys` | §12 SDK |
| ❌ Missing | `DELETE` | `/v1/console/sdk/keys/{id}` | §12 SDK |
| ❌ Missing | `GET` | `/v1/console/sdk/rate-limits` | §12 SDK |
| ❌ Missing | `GET` | `/v1/console/sdk/webhooks` | §12 SDK |
| ❌ Missing | `POST` | `/v1/console/sdk/webhooks` | §12 SDK |
| ❌ Missing | `DELETE` | `/v1/console/sdk/webhooks/{id}` | §12 SDK |
| ⚠️ Undocumented | `GET` | `/v1/console/sessions` | §7 Sessions — query params `status`, `user_id`, `agent_id`, `search`, `sort`, `page`, `page_size` not documented in backend |

---

## 15. Error Response Format

All API errors follow the standard format (from `ApiErrorResponse`):

```json
{
  "message": "Human-readable error message",
  "code":    "INVALID_ARGUMENT",
  "status":  400,
  "details": { "field": "query", "reason": "must not be empty" }
}
```

| HTTP Status | Description | Frontend Behaviour |
|------------|-------------|-------------------|
| `400` | Bad request / validation failed | Show validation message to user |
| `401` | Missing/invalid auth | Auto-refresh token → if fails, redirect to `/login` |
| `403` | Insufficient permissions | Show "Access Denied" error |
| `404` | Resource not found | Show 404 state in component |
| `429` | Rate limit exceeded | Show retry-after toast |
| `500` | Server error | Show generic error state |
| `503` | Service unavailable | Show "Service Temporarily Unavailable" |
| `504` | Timeout | Show timeout error with retry button |

---

## 16. Rate Limiting Headers

When rate limited, the backend returns:

```
X-RateLimit-Limit:     600
X-RateLimit-Remaining: 0
X-RateLimit-Reset:     1683820800
Retry-After:           30
```

| Tier | Req/min | Burst |
|------|---------|-------|
| Free | 60 | 10 |
| Pro | 600 | 50 |
| Enterprise | 6000 | 200 |
