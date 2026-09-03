# UI Solution: UI-SOL-CORE-002 — Cross-Engine Recall UI

**Solution ID:** UI-SOL-CORE-002  
**CR References:** [CR-CORE-002](../../../../docs/crs/v3/core-memory/CR-CORE-002-Cross-Engine-Recall.md)  
**Backend Solution:** [SOL-CORE-002](../../../../backend/specs/crs/v3/core-memory/solutions/SOL-CORE-002-Cross-Engine-Recall.md)  
**Feature:** Cross-Engine Recall — RRF Fusion Search, Multi-engine Results  
**Priority:** 🔴 Critical  
**Frontend Component:** `ui/src/pages/memory-explorer/`

---

## 1. Mục Đích

Xây dựng UI cho Cross-Engine Recall:
- Search form với `types` filter (multi-select) và `time_range`
- Hiển thị RRF fused results với per-engine attribution
- Engines queried badge list
- Latency và hit count display

---

## 2. Backend API Contract

```http
POST /v1/memory/recall
{
  "user_id":    string,
  "query":      string,
  "types":      string[],           // optional engine type filter
  "time_range": { "from": string, "to": string } | null,
  "limit":      number
}
→ {
    "results": [
      { "id": string, "content": string, "type": string, "engine": string, "score": number }
    ],
    "total_hits":       number,
    "engines_queried":  string[]
  }
```

---

## 3. Components Architecture

### 3.1 Cross-Engine Recall Panel

```
CrossEngineRecallPage
├── RecallSearchBar         ← main query input
├── RecallFilters
│   ├── TypeMultiSelect     ← episodic, semantic, conversational, profile, procedural, adaptive
│   ├── TimeRangePicker     ← from + to date pickers
│   └── LimitSlider         ← 5 | 10 | 20 | 50 results
├── RecallResultHeader
│   ├── HitCount            ← "47 total hits"
│   ├── EnginesQueried      ← badge list: [graphiti] [zep] [cognee]
│   └── LatencyBadge        ← "< 500ms"
└── RecallResultsList
    └── RecallResultCard
        ├── EngineTag        ← colored engine badge
        ├── TypeTag          ← memory type
        ├── FusedScoreBar    ← RRF score 0.0-1.0 (larger = more relevant)
        ├── ContentPreview   ← full content (collapsible if long)
        └── MetaRow          ← memory ID, timestamp
```

### 3.2 Fused Score Visualization

```typescript
// Score bar: 0.0 → 1.0
// RRF formula: score = Σ 1/(60 + rank_i) per engine
// Display as percentage width bar + numeric value

function ScoreBar({ score }: { score: number }) {
  const pct = Math.round(score * 100);
  const color = score > 0.8 ? 'bg-green-500' :
                score > 0.5 ? 'bg-yellow-500' : 'bg-gray-400';
  return (
    <div className="flex items-center gap-2">
      <div className={`h-2 rounded ${color}`} style={{ width: `${pct}%` }} />
      <span className="text-xs text-gray-500">{score.toFixed(3)}</span>
    </div>
  );
}
```

### 3.3 Engines Queried Display

```typescript
// After recall: show which engines were actually queried
// Example: "Queried: [graphiti] [zep] (2 of 6 engines)"
// Engines NOT queried shown as grayed out

function EnginesQueriedBar({ engines }: { engines: string[] }) {
  const all = ['cognee', 'graphiti', 'zep', 'memobase', 'openviking', 'supermemory'];
  return (
    <div className="flex gap-1">
      {all.map(e => (
        <span key={e} className={engines.includes(e) 
          ? ENGINE_COLORS[e]          // active: colored
          : 'bg-gray-100 text-gray-400'  // inactive: grayed
        }>
          {e}
        </span>
      ))}
    </div>
  );
}
```

---

## 4. React Query Hook

```typescript
// ui/src/api/hooks/useMemoryRecall.ts

export function useMemoryRecall() {
  return useMutation({
    mutationFn: (req: RecallRequest) => memoryApi.recall(req),
  });
}

// Usage in component:
const recall = useMemoryRecall();
const handleSearch = (query: string, types: string[], timeRange: TimeRange) => {
  recall.mutate({ user_id, query, types, time_range: timeRange, limit: 10 });
};
```

---

## 5. Acceptance Criteria (Frontend)

- [ ] Query input + types multi-select + time range pickers
- [ ] Submit → `POST /v1/memory/recall` với đúng format
- [ ] Results hiển thị với engine badge + RRF score bar
- [ ] `engines_queried` hiển thị đúng engines đã query
- [ ] `total_hits` và latency visible trong header
- [ ] `time_range` null khi không chọn dates
- [ ] Results sorted by score descending (UI mirrors backend sort)
- [ ] Empty state khi `total_hits = 0`
