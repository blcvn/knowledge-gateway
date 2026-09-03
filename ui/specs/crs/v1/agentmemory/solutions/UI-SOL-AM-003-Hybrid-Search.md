# UI Solution: UI-SOL-AM-003 — Hybrid Search Engine

**Solution ID:** UI-SOL-AM-003  
**CR References:** [CR-AM-003](../../../../docs/crs/v1/agentmemory/CR-AM-003-Hybrid-Search-Engine.md)  
**Backend Solution:** [SOL-003-Hybrid-Search-Engine.md](../../../../backend/specs/crs/v1/agentmemory/solutions/SOL-003-Hybrid-Search-Engine.md)  
**Feature:** Hybrid Search — BM25 + Vector + Graph + RRF Fusion  
**Priority:** 🔴 Critical  
**Frontend Component:** `ui/src/pages/memory-explorer/`

---

## 1. Mục Đích

Xây dựng UI Hybrid Search cho phép:
- Search memory với 4 modes: `semantic`, `bm25`, `hybrid`, `graph`
- Chọn engines để search (multi-select từ 7 engines)
- Xem RRF fusion score và per-engine score breakdown
- Filter theo type, date, policy tags
- Reranking strategy selection

---

## 2. Backend API Alignment

### API Endpoint Chính

```typescript
// POST /v1/console/memory/search
interface MemorySearchQuery {
  query:     string;
  mode:      'semantic' | 'bm25' | 'hybrid' | 'graph';
  engines:   EngineType[];        // filter engines
  filters:   {
    memory_type?: string;
    date_from?:   string;
    date_to?:     string;
    policy_tags?: string[];
  };
  limit:     number;
  offset:    number;
  reranking: 'cross_encoder' | 'rrf' | 'none';
}

interface MemorySearchResult {
  results:   MemoryItem[];
  total:     number;
  facets: {
    byEngine: Record<string, number>;
    byType:   Record<string, number>;
  };
  latencyMs: number;
}
```

---

## 3. Components Architecture

### 3.1 Unified Search Bar

```
MemorySearchPage
├── SearchInput                 ← full-width query input
├── SearchControls (row)
│   ├── ModeSelector            ← Tabs: Semantic | BM25 | Hybrid | Graph
│   ├── EngineMultiSelect       ← checkboxes: cognee, graphiti, zep, ...
│   └── RerankingDropdown       ← RRF | Cross-encoder | None
├── AdvancedFilters (collapsible)
│   ├── MemoryTypeFilter        ← episodic, semantic, conversational, ...
│   ├── DateRangePicker         ← date_from + date_to
│   └── PolicyTagsInput         ← tag chips input
├── SearchResultsSection
│   ├── SearchMetaBar           ← "{total} results in {latencyMs}ms"
│   ├── FacetsPanel (left)      ← byEngine + byType breakdown
│   └── ResultsList
│       ├── MemoryResultCard
│       │   ├── EngineBadge     ← colored by engine type
│       │   ├── MemoryTypeBadge ← episodic/semantic/...
│       │   ├── ScoreBar        ← RRF score 0.0–1.0
│       │   ├── ContentPreview  ← truncated summary
│       │   ├── EntitiesChips   ← entity list
│       │   └── ActionButtons   ← View / View Neighbors
│       └── Pagination
```

### 3.2 Search Mode Visual Differences

| Mode | UI Indicator | Extra Info Shown |
|------|-------------|-----------------|
| `semantic` | Purple border | Embedding similarity score |
| `bm25` | Blue border | Term frequency highlights |
| `hybrid` | Gradient border | Combined RRF score |
| `graph` | Orange border | Graph hops, path |

### 3.3 Engine Selector

```typescript
// Multi-select với engine health indicator
const engineOptions = [
  { id: 'cognee',      label: 'Cognee (Semantic)',       status: 'healthy' },
  { id: 'graphiti',    label: 'Graphiti (Episodic)',      status: 'healthy' },
  { id: 'zep',         label: 'Zep (Conversational)',     status: 'healthy' },
  { id: 'memobase',    label: 'Memobase (Profile)',       status: 'warning' },
  { id: 'openviking',  label: 'OpenViking (Procedural)',  status: 'healthy' },
  { id: 'supermemory', label: 'Supermemory (Adaptive)',   status: 'healthy' },
  { id: 'kgs',         label: 'KGS Platform',             status: 'healthy' },
];
// Engines với status != 'healthy' hiển thị warning icon
```

---

## 4. React Query Hook

```typescript
// ui/src/api/hooks/useMemorySearch.ts
export function useMemorySearch() {
  return useMutation({
    mutationFn: (query: MemorySearchQuery) => memoryApi.search(query),
  });
}

// Debounce query: 300ms sau khi user dừng gõ
// URL sync: query params → URL để share-able search URLs
// Cache: last 5 searches cached in sessionStorage
```

---

## 5. Score & Facet Visualization

### RRF Score Bar
```
Score: 0.892 [████████████░░] (graphiti: 0.91, cognee: 0.88, zep: 0.82)
```

### Facets Panel
```
By Engine          By Type
────────────       ──────────────
graphiti  │42      episodic    │62
cognee    │28      semantic    │35
zep       │17      conversational│18
memobase  │ 6      profile     │ 6
```

---

## 6. Acceptance Criteria (Frontend)

- [ ] Search trong `< 1s` với debounce 300ms
- [ ] Mode selector thay đổi search behavior ngay lập tức
- [ ] Engine multi-select lọc đúng engines (không query engines bị deselect)
- [ ] Facets panel cập nhật sau mỗi search
- [ ] Score bar hiển thị RRF score 0.0–1.0
- [ ] URL sync: search params persist qua browser back/forward
- [ ] Empty state khi không có kết quả với suggested queries
- [ ] Latency hiển thị `{n}ms` ở meta bar
