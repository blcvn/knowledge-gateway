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
export type EngineType = 'cognee' | 'graphiti' | 'zep' | 'openviking' | 'memobase' | 'supermemory' | 'kgs';

// Memory type
export type MemoryType = 'episodic' | 'semantic' | 'conversational' | 'procedural' | 'profile' | 'adaptive';

// Status
export type HealthStatus = 'Healthy' | 'Warning' | 'Critical';
export type PipelineStatus = 'Running' | 'Completed' | 'Failed' | 'Queued';
