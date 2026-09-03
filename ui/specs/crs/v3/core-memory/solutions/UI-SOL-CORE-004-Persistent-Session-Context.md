# UI Solution: UI-SOL-CORE-004 — Persistent Session Context UI

**Solution ID:** UI-SOL-CORE-004  
**CR References:** [CR-CORE-004](../../../../docs/crs/v3/core-memory/CR-CORE-004-Persistent-Session-Context.md)  
**Backend Solution:** [SOL-CORE-004](../../../../backend/specs/crs/v3/core-memory/solutions/SOL-CORE-004-Persistent-Session-Context.md)  
**Feature:** Persistent Session Context — Cross-session History Browser  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/sessions/`, `ui/src/pages/profiles/`

---

## 1. Mục Đích

Xây dựng UI cho Persistent Session Context:
- Cross-session history browser: xem lịch sử tất cả sessions của user
- Context assembly preview: xem context được assembled từ Zep + Memobase
- Working memory state per session
- User summary (cross-session Memobase summary)

---

## 2. Backend API Contract

```http
# Session list (all sessions for user)
GET /v1/console/sessions?user_id=u_123&status=completed

# Session detail (messages)
GET /v1/console/sessions/{id}
→ Conversation { session_id, messages[] }

# Working memory for session
GET /v1/console/sessions/{id}/working-memory
→ { session_id, summary, entities: string[] }

# User long-term summary (cross-session Memobase)
GET /v1/console/sessions/{id}/user-summary
→ { user_id, context_string, token_count }

# Context assembly preview (Memobase)
GET /v1/console/profiles/{user_id}/context
→ { user_id, context_string, token_count, profile_section_tokens, event_section_tokens, latency_ms }
```

---

## 3. Components Architecture

### 3.1 Cross-Session History Browser

```
UserSessionHistoryPage
├── UserSelector            ← select user_id
├── SessionTimeline         ← vertical chronological list
│   └── SessionEntry
│       ├── DateHeader      ← "September 3, 2026"
│       ├── SessionCard
│       │   ├── TimeRange   ← "10:00 AM — 10:47 AM"
│       │   ├── MessageCount← "47 messages"
│       │   ├── StatusBadge ← completed/active/failed
│       │   ├── AgentBadge  ← agent_id
│       │   └── ViewButton  ← open session detail
│       └── ContextSummary  ← collapsible: key entities/topics from session
└── UserContextPanel (right sidebar)
    ├── WorkingMemoryCard   ← current session summary + entities
    └── LongTermContextCard ← cross-session Memobase context string
```

### 3.2 Context Assembly Preview

```
ContextAssemblyPanel
├── ContextHeader           ← "Assembled Context for: u_123"
├── ContextString           ← the assembled context text (mono font)
├── TokenMetrics
│   ├── TotalTokens         ← "247 tokens"
│   ├── ProfileSection      ← "Profile: 124 tokens"
│   └── EventSection        ← "Events: 123 tokens"
├── LatencyBadge            ← "Generated in 45ms"
└── CopyButton              ← copy context_string to clipboard
```

### 3.3 Working Memory Card

```typescript
interface WorkingMemoryDisplay {
  summary:  string;         // compressed session summary
  entities: string[];       // detected entities chips
}

// Display:
// Summary: "User asked about JWT auth implementation and Clean Architecture patterns..."
// Entities: [JWT] [auth] [Clean Architecture] [Go] [middleware]
```

---

## 4. React Query Hooks

```typescript
// ui/src/api/hooks/useSessionContext.ts

export function useUserSessions(userId: string) {
  return useQuery({
    queryKey: ['sessions', { userId }],
    queryFn:  () => sessionApi.getSessions({ user_id: userId }),
    enabled:  !!userId,
  });
}

export function useWorkingMemory(sessionId: string) {
  return useQuery({
    queryKey: ['sessions', sessionId, 'working-memory'],
    queryFn:  () => sessionApi.getWorkingMemory(sessionId),
    enabled:  !!sessionId,
  });
}

export function useUserContextAssembly(userId: string) {
  return useQuery({
    queryKey: ['profiles', userId, 'context'],
    queryFn:  () => profileApi.getContext(userId),
    enabled:  !!userId,
    staleTime: 30_000,    // context assembly < 200ms, cache 30s
  });
}
```

---

## 5. Context Persistence Visualization

```
Session Timeline (showing persistent context across sessions)

[Sep 1 — Session 1]  "Introduced myself as Bình, Go developer"
          │ Context flows to →
[Sep 2 — Session 2]  "AI knew: 'Bình, Go developer, VNP Memory project'"
          │ Context flows to →
[Sep 3 — Session 3]  "AI knew: 'Bình, prefers Clean Architecture, VNP project deadline Oct 1'"
                       ↑ Enriched from profile extraction
```

---

## 6. Acceptance Criteria (Frontend)

- [ ] Session timeline sorted chronologically (newest first)
- [ ] Session entities shown as chips (collapsible)
- [ ] Working memory summary displayed for each selected session
- [ ] Cross-session context string displayed in ProfileContext panel
- [ ] Token count breakdown: profile section vs event section
- [ ] Context latency displayed (`< 200ms` expected)
- [ ] Context copy-to-clipboard button
- [ ] User selector with search/autocomplete
