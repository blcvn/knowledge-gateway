---
id: TASK-049
title: Memory Explorer Upgrade — 6 Memory Type Tabs + API Search
service: ui
version: 1.0.0
status: Done
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
linked_feat: FEAT-002
depends_on: TASK-047
---

## Mục Tiêu
Upgrade Memory Explorer từ 4 memory types (mock) sang 6 types (API) theo UX Spec Section 6.2.

## Changes Required
1. Add 2 new tabs: "Profile (Memobase)" và "Adaptive (Supermemory)"
2. Replace mock search results with `useMemorySearch()` hook
3. Add advanced filters: memory version (Supermemory), profile schema, confidence score
4. Side Inspector: add version history + profile associations panels
5. Result Card: add version chain display + engine badge

## Acceptance Criteria
- [x] AC-1: 7 result tabs (All + 6 memory types)
- [x] AC-2: Search via API with type/engine/tenant filters
- [x] AC-3: Profile tab shows Memobase-sourced memories
- [x] AC-4: Adaptive tab shows Supermemory memories with version info
- [x] AC-5: Side Inspector shows version history for adaptive memories
