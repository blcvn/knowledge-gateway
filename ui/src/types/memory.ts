import { EngineType, MemoryType } from './api';

export interface MemorySearchQuery {
  query: string;
  mode: string;
  engines: EngineType[];
  filters: Record<string, unknown>;
  limit: number;
  offset: number;
  reranking: string;
}

export interface MemoryItem {
  id: string;
  engine: EngineType;
  memoryType: MemoryType;
  title: string;
  summary: string;
  content: string;
  score: number;
  entities: string[];
  sourceSessions: string[];
  temporalValidity: { from: string; to: string | null };
  policyTags: string[];
  versionChain: string | null;
  metadata: Record<string, unknown>;
}

export interface MemorySearchResult {
  results: MemoryItem[];
  total: number;
  facets: {
    byEngine: Record<string, number>;
    byType: Record<string, number>;
  };
  latencyMs: number;
}
