import type { AdaptiveMemory, MemoryVersion, ExternalConnector } from '../types/adaptive';

export const adaptiveMock = {
  memories: [{
    id: 'mem_a1',
    content: 'Adaptive memory mock',
    memory_type: 'dynamic',
    is_latest: true,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString()
  }] as AdaptiveMemory[],
  
  versions: [{
    id: 'v_1',
    memory_id: 'mem_a1',
    content: 'Adaptive memory mock',
    version_number: 1,
    is_latest: true,
    created_at: new Date().toISOString()
  }] as MemoryVersion[],
  
  connectors: [{
    id: 'conn_1',
    type: 'google_drive',
    status: 'Connected',
    last_sync: new Date().toISOString(),
    document_count: 42,
    sync_frequency: '1h'
  }] as ExternalConnector[],
};
