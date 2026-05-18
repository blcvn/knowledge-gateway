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
