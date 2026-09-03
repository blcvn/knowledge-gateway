# UI Solution: UI-SOL-INTEL-001 — User Profile Assembly UI

**Solution ID:** UI-SOL-INTEL-001  
**CR References:** [CR-INTEL-001](../../../../docs/crs/v4/intelligence/CR-INTEL-001-User-Profile-Assembly.md)  
**Backend Solution:** [SOL-INTEL-001](../../../../backend/specs/crs/v4/intelligence/solutions/SOL-INTEL-001-User-Profile-Assembly.md)  
**Feature:** User Profile Assembly — 4 Categories, Confidence Bars, Context Preview  
**Priority:** 🔴 Critical  
**Frontend Component:** `ui/src/pages/profiles/`

---

## 1. Mục Đích

Xây dựng User Profile UI:
- Hiển thị 4 profile categories (preference, fact, goal, habit)
- Confidence/score bar per attribute
- Context string preview (`< 100ms` assembly)
- Manual flush button
- Buffer zone status

---

## 2. Backend API Contract

```http
GET /v1/console/profiles            → UserProfile[]
GET /v1/console/profiles/{user_id}  → UserProfile
GET /v1/console/profiles/{user_id}/context → ContextAssembly
GET /v1/console/profiles/{user_id}/buffers → BufferZone
GET /v1/console/profiles/{user_id}/events  → UserEvent[]
GET /v1/console/profiles/config     → ProfileConfig
PUT /v1/console/profiles/config     → ProfileConfig

# From CR-INTEL-001 (direct memobase API):
POST /v1/memobase/users/{user_id}/flush → { processed_blobs, updated_profiles, duration_ms }
```

### TypeScript Types (Backend Profile Structure)

```typescript
// Extended profile from CR-INTEL-001
interface ProfileEntry {
  topic:       string;      // "preference" | "fact" | "goal" | "habit"
  sub_topic:   string;      // "coding_style" | "name" | "current_project" | "work_hours"
  content:     string;      // "Clean Architecture" | "Bình" | "VNP Memory" | "9am-11pm"
  confidence?: number;      // 0.0-1.0 (profile_score)
}
```

---

## 3. Components Architecture

### 3.1 User Profile Detail Page

```
UserProfilePage
├── ProfileHeader
│   ├── UserId              ← "u_123"
│   ├── LastUpdated         ← "Updated 2 hours ago"
│   └── FlushButton         ← "Force Flush" (trigger YOLO extraction)
├── ProfileCategories (tabbed or grid)
│   ├── PreferenceTab
│   │   └── ProfileEntries  ← coding_style: "Clean Architecture" [0.95]
│   ├── FactTab
│   │   └── ProfileEntries  ← name: "Bình" [1.0], role: "Backend Eng" [0.92]
│   ├── GoalTab
│   │   └── ProfileEntries  ← current_project: "VNP Memory" [0.88]
│   └── HabitTab
│       └── ProfileEntries  ← work_hours: "9am-11pm" [0.75]
├── ContextAssemblyPanel    ← right column
│   ├── ContextString       ← "Tên: Bình | Role: Backend Eng..."
│   ├── TokenCount          ← "24 tokens"
│   ├── LatencyBadge        ← "Generated in 45ms"
│   └── CopyButton
└── BufferZoneCard
    ├── TokenCount          ← "12 / 20 blobs"
    ├── FillBar             ← progress bar
    └── AutoFlushTimer      ← "Auto-flush in: idle timeout"
```

### 3.2 Profile Entry Component

```typescript
// Confidence bar + value display
function ProfileEntryRow({ entry }: { entry: ProfileEntry }) {
  const confidencePct = Math.round((entry.confidence ?? 1) * 100);
  
  const confidenceColor =
    confidencePct >= 90 ? 'bg-green-500' :
    confidencePct >= 70 ? 'bg-blue-500'  :
    confidencePct >= 50 ? 'bg-amber-500' :
                          'bg-red-400';
  
  return (
    <div className="flex items-center gap-3 py-2 border-b">
      <span className="text-sm text-gray-500 w-32">{entry.sub_topic}</span>
      <span className="flex-1 font-medium">{entry.content}</span>
      <div className="flex items-center gap-1 w-24">
        <div className={`h-1.5 rounded ${confidenceColor}`}
             style={{ width: `${confidencePct}%` }} />
        <span className="text-xs text-gray-400">{confidencePct}%</span>
      </div>
    </div>
  );
}
```

### 3.3 Flush Button with Progress

```typescript
function FlushButton({ userId }: { userId: string }) {
  const flush = useMutation({
    mutationFn: () => profileApi.flush(userId),
    onSuccess: (result) => {
      toast.success(
        `Flush complete: ${result.updated_profiles} profiles updated in ${result.duration_ms}ms`
      );
      queryClient.invalidateQueries({ queryKey: ['profiles', userId] });
    },
  });
  
  return (
    <Button
      onClick={() => flush.mutate()}
      loading={flush.isPending}
      variant="outline"
    >
      {flush.isPending ? 'Flushing...' : 'Force Flush'}
    </Button>
  );
}
```

---

## 4. Context Assembly Preview

```
Context String (mono font):
"Tên: Bình | Role: Backend Engineer | Thích: Go, Clean Architecture | Project: VNP Memory"

Token metrics:
Profile section: 18 tokens
Event section:   6 tokens  
Total:          24 tokens
Generated in:   45ms
```

---

## 5. Acceptance Criteria (Frontend)

- [ ] 4 category tabs: preference, fact, goal, habit
- [ ] Confidence bars: 0-100% with green/blue/amber/red color coding
- [ ] Context string displayed with token count
- [ ] Context latency displayed (expected `< 100ms`)
- [ ] Flush button triggers extraction + shows results
- [ ] Buffer zone: fill progress bar (current blobs / threshold)
- [ ] Auto-flush timer: "Auto-flush when idle for X minutes"
- [ ] Profile list: search by user_id with autocomplete
