/**
 * Session Types — matches backend /v1/console/sessions/* responses
 */

export interface Session {
  id:            string;
  user_id:       string;
  title:         string;
  agent_id?:     string;
  status:        'active' | 'completed' | 'failed';
  message_count: number;
  created_at:    string;
  updated_at:    string;
}

export interface Message {
  id:               string;
  role:             'user' | 'assistant' | 'system' | 'tool';
  content:          string;
  timestamp:        string;
  memory_sources?:  string[];
}

export interface Conversation {
  session_id: string;
  messages:   Message[];
}

export interface WorkingMemory {
  session_id: string;
  summary:    string;
  entities:   string[];
}

export interface UserSummary {
  user_id:        string;
  context_string: string;
  token_count:    number;
}

export interface SessionFilters {
  status?:    'active' | 'completed' | 'failed';
  user_id?:   string;
  agent_id?:  string;
  search?:    string;
  sort?:      string;
  page?:      number;
  page_size?: number;
}

export interface SessionTimeline {
  event_type: string;
  engine:     string;
  memory_id:  string;
  timestamp:  string;
  latency_ms: number;
  details:    Record<string, unknown>;
}

export interface SessionDiff {
  session_id: string;
  added:      Array<{ engine: string; memory_id: string; content: string }>;
  updated:    Array<{ engine: string; memory_id: string; field: string; before: unknown; after: unknown }>;
  deleted:    Array<{ engine: string; memory_id: string }>;
}
