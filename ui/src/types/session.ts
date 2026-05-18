export interface Session {
  id: string;
  user_id: string;
  title: string;
  agent_id?: string;
  status?: 'active' | 'completed' | 'failed';
  message_count?: number;
  created_at: string;
  updated_at: string;
}

export interface Message {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: string;
  memory_sources?: string[];
}

export interface Conversation {
  session_id: string;
  messages: Message[];
}

export interface WorkingMemory {
  session_id: string;
  summary: string;
  entities: string[];
}
