# UI Solution: UI-SOL-CORE-005 — Temporal Reasoning UI

**Solution ID:** UI-SOL-CORE-005  
**CR References:** [CR-CORE-005](../../../../docs/crs/v3/core-memory/CR-CORE-005-Temporal-Reasoning.md)  
**Backend Solution:** [SOL-CORE-005](../../../../backend/specs/crs/v3/core-memory/solutions/SOL-CORE-005-Temporal-Reasoning.md)  
**Feature:** Temporal Reasoning — Timeline View, is_latest Badge, Temporal Filter  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/memory-explorer/`, `ui/src/components/memory/TemporalFilter.tsx`

---

## 1. Mục Đích

Xây dựng UI cho Temporal Reasoning:
- `is_latest` badge trên memory cards
- Temporal range filter trong search
- Timeline view: chronological history of a subject's facts
- Contradiction detection visualization (old fact → superseded by new)

---

## 2. Backend API Contract

```http
# Search với temporal filters
POST /v1/memory/recall
{
  "types": ["episodic"],
  "time_range": { "from": "2026-09-01", "to": "2026-09-03" }
}
→ results with is_latest field per item

# Timeline API
GET /v1/memory/timeline?user_id=u_123&subject=project_deadline
→ [
    { id, content, valid_at, invalid_at, is_latest, subject },
    ...  // ordered by valid_at
  ]

# Memory detail includes temporal fields
GET /v1/console/memory/{id}
→ MemoryItem with temporalValidity: { from, to }
```

### TypeScript Types Extension

```typescript
// Extend MemoryItem for temporal fields
interface EpisodicMemoryItem extends MemoryItem {
  is_latest:    boolean;
  valid_at:     string;            // when this fact became valid
  invalid_at?:  string | null;     // when superseded (null = still valid)
  subject?:     string;            // entity this fact describes
}

interface TimelineEntry {
  id:         string;
  content:    string;
  valid_at:   string;
  invalid_at: string | null;
  is_latest:  boolean;
  subject:    string;
}
```

---

## 3. Components Architecture

### 3.1 Temporal Filter Component

```
TemporalFilter
├── IsLatestToggle          ← "Show only latest facts" (default ON)
├── DateRangePicker
│   ├── FromDate
│   └── ToDate
└── SubjectSearch           ← "Filter by subject entity"

// When isLatest=true: query includes is_latest filter
// When date range set: query includes time_range
```

### 3.2 is_latest Badge

```typescript
// On MemoryCard component
function TemporalBadges({ item }: { item: EpisodicMemoryItem }) {
  return (
    <div className="flex gap-1">
      {item.is_latest ? (
        <span className="badge-green">✓ LATEST</span>
      ) : (
        <span className="badge-gray line-through">SUPERSEDED</span>
      )}
      {item.invalid_at && (
        <span className="text-xs text-gray-400">
          Valid until: {formatDate(item.invalid_at)}
        </span>
      )}
    </div>
  );
}
```

### 3.3 Subject Timeline View

```
SubjectTimelinePage
├── SubjectInput            ← "project_deadline"
├── UserIdSelect
└── Timeline (vertical, newest at top)
    └── TimelineEntry
        ├── ValidAtDate     ← "Sep 3, 2026 — 12:00 PM"
        ├── IsLatestBadge   ← LATEST or SUPERSEDED
        ├── ContentCard     ← memory content
        └── SupersededBy    ← "→ superseded by [newer entry] on Sep 3"

Visual:
──── Sep 3 ──── "Project deadline changed to Oct 1" [LATEST]
      ↑ supersedes
──── Sep 1 ──── "Project deadline is Sep 30" [SUPERSEDED]
      ↑ supersedes
──── Aug 15 ─── "Project deadline is Sep 15" [SUPERSEDED]
```

### 3.4 Contradiction Indicator

```typescript
// When is_latest=false AND invalid_at is set:
// Show: [SUPERSEDED] badge + "superseded by newer entry"
// When clicking: expand to show the superseding entry

function ContradictionIndicator({ oldFact, newFact }: Props) {
  return (
    <div className="border-l-2 border-orange-300 pl-3">
      <div className="text-sm text-gray-500">
        ⚠️ Superseded by newer fact on {formatDate(oldFact.invalid_at)}:
      </div>
      <blockquote className="text-sm italic">{newFact.content}</blockquote>
    </div>
  );
}
```

---

## 4. React Query Hooks

```typescript
// ui/src/api/hooks/useTemporalMemory.ts

export function useMemoryTimeline(userId: string, subject: string) {
  return useQuery({
    queryKey: ['memory', 'timeline', userId, subject],
    queryFn:  () => memoryApi.getTimeline({ user_id: userId, subject }),
    enabled:  !!userId && !!subject,
  });
}
```

---

## 5. Acceptance Criteria (Frontend)

- [ ] `is_latest=true` → green `LATEST` badge
- [ ] `is_latest=false` → gray `SUPERSEDED` badge (with strikethrough)
- [ ] `temporalValidity.from/to` displayed as date range
- [ ] Date range filter passes `time_range` to recall API
- [ ] Subject timeline shows facts in chronological order
- [ ] Superseded facts show pointer to newer fact
- [ ] Default search: `is_latest=true` filter active
- [ ] Toggle "Show all versions" disables is_latest filter
