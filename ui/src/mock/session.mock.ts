import type { Session, Conversation } from '../types/session';

export const sessionMock = {
  sessions: [
    {
      id: 'sess_1',
      user_id: 'user_123',
      title: 'Memory Architecture Discussion',
      agent_id: 'agent-001',
      status: 'active',
      message_count: 12,
      created_at: '2026-05-13T09:00:00Z',
      updated_at: '2026-05-13T10:45:00Z',
    },
    {
      id: 'sess_2',
      user_id: 'user_456',
      title: 'Graphiti Query Optimization',
      agent_id: 'agent-002',
      status: 'completed',
      message_count: 8,
      created_at: '2026-05-12T14:00:00Z',
      updated_at: '2026-05-12T15:30:00Z',
    },
    {
      id: 'sess_3',
      user_id: 'user_789',
      title: 'User Profile Schema Design',
      agent_id: 'agent-001',
      status: 'active',
      message_count: 5,
      created_at: '2026-05-13T08:00:00Z',
      updated_at: '2026-05-13T08:45:00Z',
    },
    {
      id: 'sess_4',
      user_id: 'user_123',
      title: 'Supermemory Auto-Forget Rules',
      agent_id: 'agent-003',
      status: 'completed',
      message_count: 15,
      created_at: '2026-05-11T10:00:00Z',
      updated_at: '2026-05-11T12:00:00Z',
    },
  ] satisfies Session[],

  conversation: {
    session_id: 'sess_1',
    messages: [
      { id: 'msg_1', role: 'user', content: 'How does Graphiti handle temporal episodic memory?', timestamp: '2026-05-13T09:00:00Z' },
      { id: 'msg_2', role: 'assistant', content: 'Graphiti stores episodic memories as a temporal knowledge graph. Each episode captures who did what, where, and when, preserving causal chains and temporal ordering.', timestamp: '2026-05-13T09:00:05Z', memory_sources: ['graphiti:ep_abc', 'cognee:sem_001'] },
      { id: 'msg_3', role: 'user', content: 'What are the main differences from Memobase?', timestamp: '2026-05-13T09:15:00Z' },
      { id: 'msg_4', role: 'assistant', content: 'Memobase focuses on long-term user profiling: it extracts structured profiles from conversations (goals, preferences, expertise) and maintains a persistent user model with topic hierarchies.', timestamp: '2026-05-13T09:15:06Z', memory_sources: ['memobase:prof_123', 'graphiti:ep_xyz'] },
      { id: 'msg_5', role: 'user', content: 'How can I combine both in my AI agent?', timestamp: '2026-05-13T09:30:00Z' },
      { id: 'msg_6', role: 'assistant', content: 'The recommended pattern is: use Memobase for the persistent user context layer and Graphiti for session-specific episodic context. The /v1/context/assemble endpoint orchestrates both automatically.', timestamp: '2026-05-13T09:30:08Z', memory_sources: ['memobase:prof_123', 'graphiti:ep_abc', 'graphiti:ep_xyz'] },
    ],
  } satisfies Conversation,
};
