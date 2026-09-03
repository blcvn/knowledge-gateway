# UI Solution: UI-SOL-PLAT-004 — OpenTelemetry Tracing UI

**Solution ID:** UI-SOL-PLAT-004  
**CR References:** [CR-PLAT-004](../../../../docs/crs/v3/platform/CR-PLAT-004-OpenTelemetry-Tracing.md)  
**Feature:** OpenTelemetry — Distributed Trace Viewer, Span Details  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/observability/`

---

## 1. Mục Đích

Xây dựng Distributed Trace Viewer:
- List traces với filters (service, status, operation, time range)
- Trace detail: span timeline (Gantt-style)
- Span detail panel với full attributes
- Error span highlighting

---

## 2. Backend API Contract

```http
GET /v1/console/observability/traces
Query: service, status, operation, from, to
→ TraceSpan[]

GET /v1/console/observability/traces/{id}
→ TraceSpan (full detail)
```

### TypeScript Types

```typescript
interface TraceSpan {
  id?:          string;
  trace_id:     string;
  span_id:      string;
  name?:        string;
  operation?:   string;
  service:      string;
  duration_ms?: number;
  duration?:    number;
  status?:      'ok' | 'slow' | 'error' | string;
  timestamp?:   string;
}
```

---

## 3. Components

### 3.1 Traces List

```
ObservabilityTracesPage
├── TraceFilters
│   ├── ServiceSelect       ← cognee-ingestion | graphiti | ...
│   ├── StatusSelect        ← ok | slow | error
│   ├── OperationInput      ← filter by operation name
│   └── DateRange
├── TracesTable
│   └── TraceRow
│       ├── TraceId         ← short hash (first 8 chars)
│       ├── Service         ← service badge
│       ├── Operation
│       ├── Duration        ← colored: green (<100ms) / amber (100-500ms) / red (>500ms)
│       ├── Status          ← ok/slow/error badge
│       ├── Timestamp       ← relative
│       └── DetailLink
└── SpanDetailDrawer (right slide-in)
    ├── SpanHeader          ← trace_id + span_id
    ├── GanttTimeline       ← (simplified: single span bar)
    └── AttributesTable     ← all span attributes as key-value
```

### 3.2 Duration Color Coding

```typescript
function DurationBadge({ ms }: { ms: number }) {
  const color = ms < 100   ? 'text-green-600' :
                ms < 500   ? 'text-amber-600'  :
                             'text-red-600';
  return <span className={color}>{ms}ms</span>;
}
```

### 3.3 Trace Status Badges

```typescript
const STATUS_STYLES = {
  ok:    'bg-green-100 text-green-700',
  slow:  'bg-amber-100 text-amber-700',
  error: 'bg-red-100 text-red-700',
};
```

---

## 4. React Query Hook

```typescript
export function useTraces(filters: TraceFilters) {
  return useQuery({
    queryKey: ['observability', 'traces', filters],
    queryFn:  () => observabilityApi.getTraces(filters),
    refetchInterval: 15_000,
  });
}
```

---

## 5. Acceptance Criteria (Frontend)

- [ ] Traces list with service/status/operation/date filters
- [ ] Duration color-coded: green/amber/red by threshold
- [ ] Error traces highlighted (red row background)
- [ ] Trace detail drawer with all span attributes
- [ ] Date range filter with preset: Last 1h / 6h / 24h / 7d
- [ ] Empty state: "No traces found for selected filters"
