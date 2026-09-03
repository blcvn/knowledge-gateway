# TASK-UI-008 — Tạo `MemoryEmptyState` component

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-008 |
| **Layer** | Frontend — TypeScript / React |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-007 §4](../solutions/SOL-007-Gap-Fixes.md) |
| **Priority** | 🔴 P0 |
| **Depends On** | TASK-UI-007 |
| **Estimated** | 0.5h |

---

## Context

Khi `POST /v1/console/memory/search` trả về `results: []`, UI cần hiển thị empty state thay vì blank screen. Ngoài ra cần handle cả loading skeleton và error state.

---

## Target Files

| Action | File Path |
|---|---|
| MODIFY | `ui/src/components/MemoryExplorer/MemorySearchResults.tsx` (hoặc file hiển thị kết quả search) |

---

## Implementation

```tsx
// Tìm component render search results và bổ sung 3 state:

function MemorySearchResults({ query }: { query: Partial<MemorySearchQuery> }) {
  const { data, isLoading, isError, refetch } = useMemorySearch(query);

  // 1. Loading
  if (isLoading) {
    return (
      <div className="memory-skeleton">
        {[1, 2, 3].map(i => (
          <div key={i} className="memory-skeleton-card animate-pulse" />
        ))}
      </div>
    );
  }

  // 2. Error
  if (isError) {
    return (
      <div className="memory-empty-state">
        <span className="icon">⚠️</span>
        <h3>Search failed</h3>
        <p>Could not reach the memory search service.</p>
        <button onClick={() => refetch()}>Retry</button>
      </div>
    );
  }

  // 3. Empty
  if (!data || data.results.length === 0) {
    return (
      <div className="memory-empty-state">
        <span className="icon">🔍</span>
        <h3>No memories found</h3>
        <p>
          No results for "<strong>{query.query}</strong>".
          Try different keywords or expand engine filters.
        </p>
      </div>
    );
  }

  // 4. Results
  return (
    <div>
      <div className="memory-facets">
        {/* Render facets: byEngine, byType */}
      </div>
      <div className="memory-list">
        {data.results.map(item => (
          <MemoryCard key={item.id} item={item} />
        ))}
      </div>
      <div className="memory-footer">
        <span>Latency: {data.latencyMs}ms</span>
        <span>Total: {data.total}</span>
      </div>
    </div>
  );
}
```

---

## Verification

- Render component với `query.query = ""` → empty state hiển thị
- Render khi `isLoading = true` → skeleton hiển thị
- Render khi `isError = true` → error state + Retry button
