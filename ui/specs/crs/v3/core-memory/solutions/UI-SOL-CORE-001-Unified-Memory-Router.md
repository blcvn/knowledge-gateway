# UI Solution: UI-SOL-CORE-001 — Unified Memory Router UI

**Solution ID:** UI-SOL-CORE-001  
**CR References:** [CR-CORE-001](../../../../docs/crs/v3/core-memory/CR-CORE-001-Unified-Memory-Router.md)  
**Backend Solution:** [SOL-CORE-001](../../../../backend/specs/crs/v3/core-memory/solutions/SOL-CORE-001-Unified-Memory-Router.md)  
**Feature:** Unified Memory Store — Type selector, Engine routing badge  
**Priority:** 🔴 Critical  
**Frontend Component:** `ui/src/pages/memory-explorer/` + `ui/src/components/memory/MemoryStoreForm.tsx`

---

## 1. Mục Đích

Xây dựng UI cho Unified Memory Router:
- Form nhập memory với type selector (`auto` hoặc explicit)
- Hiển thị resolved type và engine sau khi store
- Engine badge trên từng memory card
- Status tracking: `processing` → `completed`

---

## 2. Backend API Contract

```http
POST /v1/memory/store
{
  "user_id":  string,
  "content":  string,
  "type":     "auto" | "episodic" | "semantic" | "conversational" | "profile" | "procedural" | "adaptive",
  "metadata": { "source": string, "session_id"?: string }
}
→ 202 Accepted {
    "id":     "mem_abc123",
    "type":   "profile",       // resolved type
    "engine": "memobase",      // selected engine
    "status": "processing"
  }
```

---

## 3. Components Architecture

### 3.1 Memory Store Form

```
MemoryStoreForm (modal or side panel)
├── UserIdInput             ← select user from dropdown
├── ContentTextarea         ← memory content, multi-line
├── TypeSelector            ← segment control
│   ├── [Auto 🤖]           ← LLM classifies
│   ├── [Episodic 📅]
│   ├── [Semantic 🧠]
│   ├── [Conversational 💬]
│   ├── [Profile 👤]
│   ├── [Procedural ⚙️]
│   └── [Adaptive 🔄]
├── MetadataSection (collapsible)
│   ├── SourceInput         ← "chat" | "api" | "import"
│   └── SessionIdInput
└── StoreButton             ← "Store Memory"

After Submit:
└── StoreResult
    ├── StatusBadge         ← "PROCESSING" (animated)
    ├── TypeBadge           ← resolved type (if auto)
    ├── EngineBadge         ← selected engine (colored)
    └── MemoryId            ← "ID: mem_abc123"
```

### 3.2 Engine → Color Mapping

```typescript
const ENGINE_COLORS: Record<string, string> = {
  cognee:      'bg-purple-100 text-purple-800',
  graphiti:    'bg-blue-100 text-blue-800',
  zep:         'bg-green-100 text-green-800',
  memobase:    'bg-orange-100 text-orange-800',
  openviking:  'bg-cyan-100 text-cyan-800',
  supermemory: 'bg-pink-100 text-pink-800',
  kgs:         'bg-indigo-100 text-indigo-800',
};
```

### 3.3 Type Selector Auto Mode

```typescript
// "auto" mode: show info tooltip
// "When 'Auto' is selected, the system uses AI to classify your memory
//  and route it to the most appropriate engine."

// After store with auto: animate from "Auto 🤖" → "Profile 👤" (memobase)
// Show: "Auto-classified as: Profile → Memobase"
```

---

## 4. React Query Hook

```typescript
// ui/src/api/hooks/useMemoryStore.ts

export function useMemoryStore() {
  const qc = useQueryClient();
  
  return useMutation({
    mutationFn: (req: StoreMemoryRequest) =>
      memoryApi.store(req),
    onSuccess: (result) => {
      // Invalidate search results
      qc.invalidateQueries({ queryKey: ['memory', 'search'] });
      // Show success toast
      toast.success(`Memory stored → ${result.engine} (${result.type})`);
    },
    onError: (err: ApiError) => {
      toast.error(getErrorMessage(err));
    },
  });
}
```

---

## 5. Routing Rules Display

```
Memory Type → Engine Routing Table (info panel)
─────────────────────────────────────────────
episodic      →  Graphiti   (Episodic Memory)
semantic      →  Cognee     (Semantic Memory)
conversational→  Zep        (Conversational)
profile       →  Memobase   (User Profile)
procedural    →  OpenViking (Filesystem)
adaptive      →  Supermemory(Adaptive Memory)
auto          →  AI selects best engine
```

---

## 6. Acceptance Criteria (Frontend)

- [ ] Form có type selector với 7 options (auto + 6 explicit)
- [ ] Submit → `202 Accepted` → hiển thị engine badge
- [ ] Auto mode: hiển thị resolved type sau submit
- [ ] Engine badge color-coded theo ENGINE_COLORS map
- [ ] Status badge `PROCESSING` với loading animation
- [ ] Error handling: invalid content → validation message
- [ ] Success toast: "Memory stored → memobase (profile)"
- [ ] Form clears sau successful submit (with delay 2s)
