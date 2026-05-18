---
id: FEAT-012
title: User Profiles Module (Memobase Integration)
service: ui
version: 1.0.0
status: Draft
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_prd: ux_spec.md (Section 6.4)
linked_sol: SOL-002
depends_on: TASK-047
---

## Mục Tiêu
Xây dựng module quản lý User Profiles — giao diện cho Memobase engine. Biến raw conversations thành structured, actionable user knowledge. Route: `/app/profiles`.

## Bối Cảnh Nghiệp Vụ
Memobase tự động extract user profiles từ conversations. Admin cần browse, search, configure profile schemas, monitor buffer zones, và preview context assembly.

## Scope
### In Scope
- 5 sub-sections: Profile Explorer, Profile Detail, Profile Config, Buffer Zone Monitor, Event Timeline
- Context Assembly Preview
- Dữ liệu lấy từ Memobase APIs (TASK-045 profile.service.ts)
- Sidebar navigation entry "User Profiles" với icon User (teal color)

### Out of Scope
- Profile analytics dashboard (Phase 2)
- Cross-user pattern discovery (Phase 2)

## Thiết Kế Kỹ Thuật

### Route Structure
```
/app/profiles              → Profile Explorer (list all users)
/app/profiles/:userId      → Profile Detail View
/app/profiles/config       → Profile Config Editor
/app/profiles/buffers      → Buffer Zone Monitor
/app/profiles/events       → Event Timeline
```

### Component Structure
```
src/app/components/UserProfiles.tsx        ← Main container with tabs
src/app/components/profiles/
├── ProfileExplorer.tsx                    ← Browse + search users by tenant
├── ProfileDetail.tsx                      ← Tree view of user profile
├── ProfileConfigEditor.tsx                ← Schema editor + strict mode toggle
├── BufferZoneMonitor.tsx                  ← Real-time buffer status
├── EventTimeline.tsx                      ← Chronological events per user
└── ContextAssemblyPreview.tsx             ← Preview prompt-ready context
```

### API Integration (via useProfiles hooks)
| UI Section | Hook | API Endpoint |
|---|---|---|
| Profile Explorer | `useProfileList()` | `GET /api/v1/users` |
| Profile Detail | `useProfileDetail(userId)` | `GET /api/v1/users/profile/{uid}` |
| Profile Config | `useProfileConfig()` | `GET /api/v1/project/profile_config` |
| Buffer Monitor | `useBufferStatus(userId)` | `GET /api/v1/users/buffer/capacity/{uid}/{type}` |
| Event Timeline | `useUserEvents(userId)` | `GET /api/v1/users/event/{uid}` |
| Context Preview | `useContextAssembly(userId)` | `GET /api/v1/users/context/{uid}` |
| Project Usage | `useProjectUsage()` | `GET /api/v1/project/usage` |

### Profile Detail View Layout
```
User: user_123
├── Preferences
│   ├── coding_style: "functional, minimal comments"
│   ├── language: "TypeScript"
│   └── theme: "dark mode"
├── Projects
│   ├── vnp-memory: "blockchain infrastructure"
│   └── openledger: "AI platform"
└── Goals
    ├── short_term: "ship v1.0"
    └── long_term: "enterprise adoption"
```
Render as collapsible tree with topic → sub_topic → content structure.

### Buffer Zone Monitor
- Active buffers per user (card grid)
- Token accumulation progress bar (current / threshold)
- Flush history table
- LLM call count badge (fixed 3 per flush)

### UX Patterns
- Memory badge: Teal color (#14b8a6) with User icon for Profile type
- Token budget visualization: horizontal stacked bar (profile vs events)
- Latency metric display: target < 100ms with green/yellow/red indicator

## Acceptance Criteria
- [ ] AC-1: Route `/app/profiles` loads Profile Explorer with user list
- [ ] AC-2: Clicking user navigates to Profile Detail with tree view
- [ ] AC-3: Profile Config Editor allows schema CRUD + strict mode toggle
- [ ] AC-4: Buffer Zone Monitor shows token progress and flush history
- [ ] AC-5: Event Timeline displays chronological events with search
- [ ] AC-6: Context Assembly Preview renders prompt-ready string with token count
- [ ] AC-7: Sidebar contains "User Profiles" entry with teal User icon
- [ ] AC-8: All data fetched via React Query hooks (or mock fallback)

## Definition of Done
- [ ] Component renders without errors
- [ ] TypeScript strict, no `any`
- [ ] Responsive layout (Desktop + Tablet)
- [ ] Loading/Error/Empty states implemented
