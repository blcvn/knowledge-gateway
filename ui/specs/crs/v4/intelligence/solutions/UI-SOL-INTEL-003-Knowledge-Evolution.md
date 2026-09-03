# UI Solution: UI-SOL-INTEL-003 — Knowledge Evolution UI

**Solution ID:** UI-SOL-INTEL-003  
**CR References:** [CR-INTEL-003](../../../../docs/crs/v4/intelligence/CR-INTEL-003-Knowledge-Evolution.md)  
**Backend Solution:** [SOL-INTEL-003](../../../../backend/specs/crs/v4/intelligence/solutions/SOL-INTEL-003-Knowledge-Evolution.md)  
**Feature:** Knowledge Evolution — Contradiction Resolution, Evolution History  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/memory-explorer/`

---

## 1. Mục Đích

Xây dựng Knowledge Evolution UI:
- Contradiction detection list: memories về cùng subject có xung đột
- Resolution history: cách giải quyết xung đột (keep_latest, keep_both, manual)
- Evolution graph: timeline thay đổi của 1 topic/entity
- Merge conflict UI: manual resolution interface

---

## 2. Backend API Contract

```http
# Get contradictions for user
GET /v1/console/memory/contradictions?user_id=u_123
→ [
    {
      "subject": "project_deadline",
      "conflicts": [
        { "id": "m1", "content": "deadline Sep 30", "valid_at": "2026-08-01", "is_latest": false },
        { "id": "m2", "content": "deadline Oct 1", "valid_at": "2026-09-03", "is_latest": true }
      ],
      "resolution": "keep_latest"
    }
  ]

# Get evolution timeline for a concept
GET /v1/console/memory/evolution?user_id=u_123&subject=project_deadline
→ TimelineEntry[] (ordered by valid_at)

# Manual resolution
POST /v1/console/memory/resolve-conflict
{ "memory_ids": string[], "action": "keep_latest" | "keep_both" | "manual", "manual_content"?: string }
```

---

## 3. Components

### 3.1 Contradiction Detection Panel

```
ContradictionsPage
├── UserSelector
├── ContradictionList
│   └── ContradictionCard
│       ├── SubjectHeader   ← "Subject: project_deadline"
│       ├── ConflictPair    ← two conflicting memories side-by-side
│       │   ├── OldFact     ← "deadline Sep 30" (Aug 1) [SUPERSEDED]
│       │   └── NewFact     ← "deadline Oct 1" (Sep 3) [LATEST]
│       ├── ResolutionBadge ← "Auto: keep_latest"
│       └── ManualResolveBtn← "Override resolution"
└── ResolutionSummary       ← "12 contradictions, 10 auto-resolved, 2 pending"
```

### 3.2 Evolution Timeline (per topic)

```
EvolutionTimelinePage
├── SubjectInput            ← "project_deadline" or "preferred_language"
├── UserSelector
└── EvolutionChart          ← horizontal timeline
    ├── TimelineAxis        ← dates
    └── FactNodes           ← each fact as a node on timeline
        ├── FactBubble      ← content snippet + date
        ├── IsLatestMark    ← ★ for current/latest
        └── SupersededLine  ← arrow from old → new
```

### 3.3 Manual Conflict Resolution

```typescript
// When user clicks "Override resolution"
function ManualResolveModal({ conflict }: { conflict: Contradiction }) {
  const [action, setAction] = useState<'keep_latest' | 'keep_both' | 'manual'>('keep_latest');
  const [manualContent, setManualContent] = useState('');
  
  return (
    <Modal>
      <h2>Resolve Conflict: {conflict.subject}</h2>
      
      <ConflictDisplay conflict={conflict} />
      
      <RadioGroup value={action} onChange={setAction}>
        <Radio value="keep_latest">Keep Latest (recommended)</Radio>
        <Radio value="keep_both">Keep Both (no contradiction)</Radio>
        <Radio value="manual">Manual Resolution</Radio>
      </RadioGroup>
      
      {action === 'manual' && (
        <Textarea
          value={manualContent}
          onChange={e => setManualContent(e.target.value)}
          placeholder="Enter the resolved content..."
        />
      )}
      
      <Button onClick={() => resolve({ action, manualContent })}>
        Apply Resolution
      </Button>
    </Modal>
  );
}
```

---

## 4. Acceptance Criteria (Frontend)

- [ ] Contradiction list shows conflicting memory pairs
- [ ] Subject label visible on each contradiction card
- [ ] Old fact labeled `SUPERSEDED`, new fact labeled `LATEST`
- [ ] Auto-resolution badge: "Auto: keep_latest"
- [ ] Manual resolve: 3 options (keep_latest, keep_both, manual)
- [ ] Evolution timeline: facts ordered by valid_at
- [ ] Current/latest fact marked with ★ star icon
- [ ] Resolution summary: count of auto vs pending
