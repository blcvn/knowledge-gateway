---
id: TASK-044
title: TypeScript Types & API Configuration
service: ui
version: 1.0.0
status: Done
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
---

## Mục Tiêu
Tạo bộ TypeScript interfaces hoàn chỉnh cho tất cả API responses từ 6 memory engines + Gateway, cùng với API configuration module cho phép chuyển đổi giữa các environment.

## Scope

### In Scope
- Tạo thư mục `src/types/` với 11 type files
- Tạo `src/config/api.config.ts` cho API base URLs và feature flags
- TypeScript strict mode, không cho phép `any`

### Out of Scope
- Viết service layer (TASK-045)
- Viết hooks (TASK-047)

## Thiết Kế Kỹ Thuật

### 1. API Configuration (`src/config/api.config.ts`)

```typescript
export const API_CONFIG = {
  baseUrl: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
  useMockData: import.meta.env.VITE_USE_MOCK_DATA === 'true',
  engines: {
    cognee:      { baseUrl: '/api/v1/cognee',    port: 8080 },
    graphiti:    { baseUrl: '/api/v1/graphiti',   port: 8080 },
    zep:         { baseUrl: '/api/v1/zep',        port: 8080 },
    openviking:  { baseUrl: '/api/v1/openviking', port: 8080 },
    memobase:    { baseUrl: '/api/v1/memobase',   port: 8080 },
    supermemory: { baseUrl: '/api/v1/supermemory', port: 8080 },
  },
  gateway: {
    admin: '/v1/admin',
    memory: '/v1/memory',
    graph: '/v1/graph',
  },
} as const;
```

### 2. Common API Types (`src/types/api.ts`)

```typescript
// Pagination
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  hasMore: boolean;
}

// API Error
export interface ApiErrorResponse {
  message: string;
  code: string;
  status: number;
  details?: Record<string, unknown>;
}

// Engine identifier
export type EngineType = 'cognee' | 'graphiti' | 'zep' | 'openviking' | 'memobase' | 'supermemory';

// Memory type
export type MemoryType = 'episodic' | 'semantic' | 'conversational' | 'procedural' | 'profile' | 'adaptive';

// Status
export type HealthStatus = 'Healthy' | 'Warning' | 'Critical';
export type PipelineStatus = 'Running' | 'Completed' | 'Failed' | 'Queued';
```

### 3. Type Files (Mapping to UX Spec API Requirements)

| File | Maps to UX Spec | Key Interfaces |
|---|---|---|
| `dashboard.ts` | Section 6.1 + API Section 11 | `KPIData`, `EngineHealth`, `MemoryFlowMetrics`, `HeatmapData` |
| `memory.ts` | Section 6.2 | `MemoryItem`, `MemorySearchQuery`, `MemorySearchResult` |
| `profile.ts` | Section 6.4 (Memobase APIs) | `UserProfile`, `ProfileConfig`, `BufferZone`, `UserEvent`, `ContextAssembly` |
| `adaptive.ts` | Section 6.5 (Supermemory APIs) | `AdaptiveMemory`, `MemoryVersion`, `ForgetRule`, `ExternalConnector` |
| `graph.ts` | Section 6.3 | `GraphNode`, `GraphEdge`, `SubgraphData`, `OntologySchema` |
| `session.ts` | Section 6.7 | `Session`, `Conversation`, `Message`, `WorkingMemory` |
| `governance.ts` | Section 6.8 | `Tenant`, `Policy`, `AuditLogEntry`, `RetentionRule`, `GDPRRequest` |
| `pipeline.ts` | Section 6.9 | `PipelineJob`, `PipelineStage`, `QueueMetrics`, `WorkerStatus` |
| `infrastructure.ts` | Section 6.10 | `ServiceInfo`, `DatabaseHealth`, `ResourceMetrics` |
| `observability.ts` | Section 6.11 | `MetricPoint`, `TraceSpan`, `ErrorEntry`, `CostAnalytics` |

### 4. Profile Types Detail (`src/types/profile.ts`)

Based on UX Spec Section 6.4 + Memobase API:

```typescript
export interface UserProfile {
  user_id: string;
  profiles: ProfileEntry[];
  created_at: string;
  updated_at: string;
}

export interface ProfileEntry {
  topic: string;
  sub_topic: string;
  content: string;
  confidence?: number;
}

export interface ProfileConfig {
  profiles: ProfileSchemaEntry[];
  strict_mode: boolean;
}

export interface ProfileSchemaEntry {
  topic: string;
  sub_topic: string;
  description?: string;
}

export interface BufferZone {
  user_id: string;
  buffer_type: string;
  token_count: number;
  token_threshold: number;
  idle_timeout: string;
  last_flush: string;
  flush_count: number;
}

export interface UserEvent {
  id: string;
  user_id: string;
  gist: string;
  tags: string[];
  created_at: string;
  embedding?: number[];
}

export interface ContextAssembly {
  user_id: string;
  context_string: string;
  token_count: number;
  profile_section_tokens: number;
  event_section_tokens: number;
  latency_ms: number;
}

export interface ProjectBilling {
  total_llm_calls: number;
  total_tokens: number;
  cost_estimate: number;
}

export interface ProjectUsage {
  total_users: number;
  total_profiles: number;
  total_events: number;
  active_buffers: number;
}
```

### 5. Adaptive Memory Types Detail (`src/types/adaptive.ts`)

Based on UX Spec Section 6.5 + Supermemory API:

```typescript
export interface AdaptiveMemory {
  id: string;
  content: string;
  memory_type: 'static' | 'dynamic';
  is_latest: boolean;
  parent_id?: string;
  root_id?: string;
  relation_type?: 'updates' | 'extends' | 'derives';
  created_at: string;
  updated_at: string;
  forget_after?: string;
  metadata?: Record<string, unknown>;
}

export interface MemoryVersion {
  id: string;
  memory_id: string;
  content: string;
  version_number: number;
  is_latest: boolean;
  diff?: string;
  created_at: string;
}

export interface ForgetRule {
  id: string;
  memory_type: 'static' | 'dynamic';
  forget_after: string; // duration, e.g., "30d", "90d"
  noise_filter: boolean;
  contradiction_resolution: 'keep_latest' | 'keep_both' | 'manual';
}

export interface ExternalConnector {
  id: string;
  type: 'google_drive' | 'gmail' | 'notion' | 'onedrive' | 'github';
  status: 'Connected' | 'Disconnected' | 'Error';
  last_sync: string | null;
  document_count: number;
  sync_frequency: string;
  error_message?: string;
}

export interface ConnectorSyncHistory {
  id: string;
  connector_id: string;
  status: 'success' | 'failed';
  documents_synced: number;
  duration_ms: number;
  error?: string;
  synced_at: string;
}

export interface AdaptiveAnalytics {
  creation_rate: number;
  deletion_rate: number;
  contradiction_count: number;
  connector_sync_count: number;
  storage_usage_bytes: number;
}
```

## Acceptance Criteria
- [x] AC-1: `src/types/` chứa 11 type files, tất cả compile thành công với `tsc --noEmit`
- [x] AC-2: `src/config/api.config.ts` export `API_CONFIG` với tất cả engine URLs
- [x] AC-3: Không có `any` type trong bất kỳ file nào
- [x] AC-4: Types cover 100% API endpoints được liệt kê trong UX Spec Section 11
- [x] AC-5: Profile types mapping chính xác với Memobase API contract
- [x] AC-6: Adaptive types mapping chính xác với Supermemory API contract

## Definition of Done
- [x] Tất cả files tạo đúng thư mục
- [x] `tsc --noEmit` pass
- [x] ESLint pass
