---
id: FEAT-013
title: Adaptive Memory Module (Supermemory Integration)
service: ui
version: 1.0.0
status: Draft
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_prd: ux_spec.md (Section 6.5)
linked_sol: SOL-002
depends_on: TASK-047
---

## Mục Tiêu
Xây dựng module quản lý Adaptive Memory — giao diện cho Supermemory engine. Memory tự tiến hóa, tự quên, tự resolve contradictions, tích hợp external data connectors. Route: `/app/adaptive`.

## Bối Cảnh Nghiệp Vụ
Supermemory quản lý living knowledge graph với auto-forget, version chains, external connectors (Google Drive, Gmail, Notion, GitHub). Admin cần visualize, configure, monitor.

## Scope
### In Scope
- 5 sub-sections: Memory Version Explorer, Auto-Forget Rules, External Connectors, Memory Graph, Analytics
- Dữ liệu lấy từ Supermemory APIs (TASK-045 adaptive.service.ts)
- Sidebar navigation entry "Adaptive Memory" với icon Sparkle (amber color)

### Out of Scope
- Connector marketplace (Phase 2)
- Memory cost optimizer (Phase 2)

## Thiết Kế Kỹ Thuật

### Route Structure
```
/app/adaptive              → Memory Version Explorer
/app/adaptive/versions     → Version chain detail
/app/adaptive/connectors   → External Connectors management
/app/adaptive/forget-rules → Auto-Forget rule configuration
```

### Component Structure
```
src/app/components/AdaptiveMemory.tsx         ← Main container with tabs
src/app/components/adaptive/
├── MemoryVersionExplorer.tsx                 ← Version chain tree
├── AutoForgetRules.tsx                       ← Rule configuration
├── ExternalConnectors.tsx                    ← Connector status + management
├── AdaptiveMemoryGraph.tsx                   ← React Flow graph visualization
└── AdaptiveAnalytics.tsx                     ← Creation/deletion rate charts
```

### API Integration (via useAdaptiveMemory hooks)
| UI Section | Hook | API Endpoint |
|---|---|---|
| Memory List | `useAdaptiveMemories()` | `GET /api/v1/memories` |
| Version Chain | `useMemoryVersions(id)` | `GET /api/v1/memories/{id}/versions` |
| Connectors | `useConnectors()` | `GET /api/v1/connectors` |
| Analytics | `useAdaptiveAnalytics()` | `GET /api/v1/analytics` |
| Search | `useAdaptiveSearch(q)` | `GET /api/v1/search` |

### Memory Version Explorer
- Version chain visualization: parent → root (tree/timeline)
- `isLatest` flag with green badge
- Relation types: updates, extends, derives (edge labels)
- Diff view between versions (side-by-side)
- Node coloring: Static (blue), Dynamic (amber)

### External Connectors Table
| Connector | Status | Last Sync | Docs | Actions |
|---|---|---|---|---|
| Google Drive | Connected (green) | 2h ago | 1,234 | Settings, Sync Now, Disconnect |
| Gmail | Connected (green) | 4h ago | 892 | Settings, Sync Now |
| Notion | Disconnected (gray) | — | — | Connect |
| OneDrive | Connected (green) | 4h ago | 456 | Settings |
| GitHub | Connected (green) | 30m ago | 2,891 | Settings |

### Auto-Forget Rules Configuration
- Configure `forgetAfter` duration per memory type (input: "30d", "90d")
- Static vs Dynamic memory toggle
- Noise filtering rules (checkbox list)
- Contradiction resolution: dropdown (keep_latest / keep_both / manual)
- Contradiction resolution history table

### UX Patterns
- Memory badge: Amber color (#f59e0b) with Sparkle icon for Adaptive type
- Time-decay opacity for aging memories in graph view
- Edge coloring: updates (blue), extends (green), derives (purple)

## Acceptance Criteria
- [ ] AC-1: Route `/app/adaptive` loads Memory Version Explorer
- [ ] AC-2: Version chain renders tree with parent→root links
- [ ] AC-3: Diff view between versions works correctly
- [ ] AC-4: External Connectors table shows status, sync info, actions
- [ ] AC-5: Auto-Forget Rules allow CRUD with duration input
- [ ] AC-6: Analytics charts render creation/deletion rates
- [ ] AC-7: Sidebar contains "Adaptive Memory" with amber Sparkle icon
- [ ] AC-8: All data fetched via React Query hooks (or mock fallback)

## Definition of Done
- [ ] Component renders without errors
- [ ] TypeScript strict, no `any`
- [ ] Responsive layout
- [ ] Loading/Error/Empty states
