import type { MemorySearchResult, MemoryItem } from '../types/memory';

const mockItem: MemoryItem = {
  id: 'mem_123',
  engine: 'memobase',
  memoryType: 'profile',
  title: 'Mock Profile Memory',
  summary: 'This is a mock memory item.',
  content: 'Full content of the mock memory.',
  score: 0.95,
  entities: ['Mock', 'Data'],
  sourceSessions: ['sess_1'],
  temporalValidity: { from: '2026-01-01', to: null },
  policyTags: ['public'],
  versionChain: null,
  metadata: {},
};

export const memoryMock = {
  searchResult: {
    results: [mockItem],
    total: 1,
    facets: {
      byEngine: { memobase: 1 },
      byType: { profile: 1 },
    },
    latencyMs: 45,
  } as MemorySearchResult,
  
  detail: mockItem,
};
