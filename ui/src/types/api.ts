// Pagination
export interface PaginatedResponse<T> {
  data:      T[];
  total:     number;
  page:      number;
  page_size: number;
  pageSize:  number;   // camelCase alias
  has_more:  boolean;
  hasMore:   boolean;  // camelCase alias
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
export type HealthStatus    = 'Healthy' | 'Warning' | 'Critical';
export type PipelineStatus  = 'Running' | 'Completed' | 'Failed' | 'Queued';

// Search modes
export type SearchMode        = 'semantic' | 'bm25' | 'hybrid' | 'graph';
export type RerankingStrategy = 'cross_encoder' | 'rrf' | 'none';

// Engine constants
export const ALL_ENGINES = [
  'graphiti',
  'cognee',
  'memobase',
  'zep',
  'supermemory',
  'openviking',
] as const;

export type EngineId = typeof ALL_ENGINES[number];
