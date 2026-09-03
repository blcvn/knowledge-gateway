/**
 * Memory Types — matches backend /v1/console/memory/* responses
 */

import { EngineType, MemoryType, SearchMode, RerankingStrategy } from './api';

export type { SearchMode, RerankingStrategy };

export interface MemoryFilters {
  memory_type?: string;
  date_from?:   string;
  date_to?:     string;
  policy_tags?: string[];
}

export interface MemorySearchQuery {
  query:     string;
  mode:      SearchMode;
  engines:   EngineType[];
  filters:   MemoryFilters;
  limit:     number;
  offset:    number;
  reranking: RerankingStrategy;
}

export interface MemoryItem {
  id:               string;
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

export interface MemorySearchResult {
  results:   MemoryItem[];
  total:     number;
  facets: {
    byEngine: Record<string, number>;
    byType:   Record<string, number>;
  };
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
