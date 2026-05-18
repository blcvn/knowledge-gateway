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
