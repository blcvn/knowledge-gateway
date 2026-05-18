import type { UserProfile, BufferZone, UserEvent, ContextAssembly } from '../types/profile';

export const profileMock = {
  users: [{ user_id: 'user_1', created_at: new Date().toISOString(), updated_at: new Date().toISOString(), profiles: [] }] as UserProfile[],
  userDetail: {
    user_id: 'user_1',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    profiles: [
      { topic: 'Preferences', sub_topic: 'Theme', content: 'Dark mode' }
    ]
  } as UserProfile,
  buffers: [{ user_id: 'user_1', buffer_type: 'core', token_count: 500, token_threshold: 1000, idle_timeout: '5m', last_flush: new Date().toISOString(), flush_count: 10 }] as BufferZone[],
  events: [{ id: 'evt_1', user_id: 'user_1', gist: 'User logged in', tags: ['auth'], created_at: new Date().toISOString() }] as UserEvent[],
  context: { user_id: 'user_1', context_string: 'User prefers dark mode.', token_count: 5, profile_section_tokens: 3, event_section_tokens: 2, latency_ms: 20 } as ContextAssembly,
};
