---
id: TASK-050
title: Sidebar & Navigation Upgrade — Add 2 New Modules
service: ui
version: 1.0.0
status: Done
priority: P1
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
depends_on: FEAT-012, FEAT-013
---

## Mục Tiêu
Cập nhật Sidebar.tsx và App.tsx để bao gồm 2 modules mới (User Profiles, Adaptive Memory) theo UX Spec Information Architecture (Section 4).

## Changes Required

### Sidebar.tsx
Add 2 new menu items between "Graph Studio" và "Sessions":
1. **User Profiles** — Icon: `UserCircle` (lucide-react), Color: Teal, Key: `profiles`
2. **Adaptive Memory** — Icon: `Sparkles` (lucide-react), Color: Amber, Key: `adaptive`

Final sidebar order (13 items):
1. Overview
2. Memory Explorer
3. Graph Studio
4. **User Profiles** ← NEW
5. **Adaptive Memory** ← NEW
6. Context Debugger
7. Sessions
8. Governance
9. Pipelines
10. Infrastructure
11. Observability
12. API & SDK
13. Settings

### App.tsx
Add 2 new cases to `renderContent()`:
```typescript
case 'profiles':
  return <UserProfiles />;
case 'adaptive':
  return <AdaptiveMemory />;
```

## Acceptance Criteria
- [x] AC-1: Sidebar shows 13 items (was 11)
- [x] AC-2: "User Profiles" has teal UserCircle icon
- [x] AC-3: "Adaptive Memory" has amber Sparkles icon
- [x] AC-4: Clicking each navigates to correct module
- [x] AC-5: Active state highlight works for new items
