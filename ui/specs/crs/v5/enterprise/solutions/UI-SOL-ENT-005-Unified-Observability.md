# UI Solution: UI-SOL-ENT-005 — Unified Observability Dashboard

**Solution ID:** UI-SOL-ENT-005  
**CR References:** [CR-ENT-005](../../../../docs/crs/v5/enterprise/CR-ENT-005-Unified-Observability.md)  
**Backend Solution:** [SOL-ENT-005](../../../../backend/specs/crs/v5/enterprise/solutions/SOL-ENT-005-Unified-Observability.md)  
**Feature:** Unified Observability — LLM Cost Dashboard, Prometheus Metrics, Alerts  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/observability/`

---

## 1. Mục Đích

Xây dựng Unified Observability Dashboard:
- LLM cost analytics: by provider, model, task
- Memory operation metrics: latency p50/p95/p99 per engine
- Error explorer với service filter
- Cost alert visualization
- Prometheus metrics integration

---

## 2. Backend API Contract

```http
GET /v1/console/observability/metrics         → MetricsResponse (latency[], error_rate[], throughput[])
GET /v1/console/observability/traces          → TraceSpan[]
GET /v1/console/observability/errors          → ErrorEntry[]
GET /v1/console/observability/costs           → CostEntry[]

# New endpoints from CR-ENT-005:
GET /v1/console/analytics/llm-cost?from=...&to=... → {
    total_usd: number,
    by_provider: Record<string, number>,
    by_task:     Record<string, number>
  }
GET /v1/console/analytics/memory?tenant_id=... → {
    total_memories: number,
    by_engine:      Record<string, number>,
    storage_bytes:  number
  }
```

### TypeScript Types

```typescript
interface CostEntry {
  model:         string;      // "gpt-4o" | "claude-3-5-sonnet"
  engine:        string;      // which engine called the LLM
  tokens_input:  number;
  tokens_output: number;
  cost_usd:      number;
  date:          string;
}

interface LLMCostSummary {
  total_usd:    number;
  by_provider:  { openai: number; anthropic: number; google?: number };
  by_task:      { extraction: number; consolidation: number; classification?: number };
}
```

---

## 3. Components Architecture

### 3.1 Observability Dashboard Tabs

```
ObservabilityPage
├── MetricsTabs
│   ├── 📊 Metrics            ← latency/throughput/error_rate charts
│   ├── 💰 LLM Costs         ← cost breakdown + budget alerts
│   ├── 🔍 Traces            ← distributed trace viewer
│   └── ❌ Errors            ← error explorer
└── [Content per tab]
```

### 3.2 LLM Cost Dashboard

```
LLMCostDashboard
├── DateRangePicker         ← from/to for cost query
├── CostSummaryCards        ← 3 cards
│   ├── TotalUSD            ← "$12.45 today"
│   ├── TopProvider         ← "OpenAI: $8.20 (66%)"
│   └── TopTask             ← "Consolidation: $9.35 (75%)"
├── CostByProviderChart     ← donut: openai vs anthropic vs google
├── CostByTaskChart         ← bar: extraction | consolidation | classification
├── CostTimelineChart       ← area chart: cost/hour over selected range
│   └── TimeWindow: today | 7d | 30d
├── CostTable (detailed)    ← model, engine, tokens, cost per entry
└── BudgetAlertCard
    ├── DailyBudget         ← "$100/day limit"
    ├── CurrentUsage        ← "$12.45 / $100" (12.45%)
    ├── UsageBar
    └── AlertThreshold      ← "Alert at 80%"
```

### 3.3 Memory Operation Metrics

```
MetricsDashboard
├── EngineLatencyGrid       ← one card per engine (7 engines)
│   └── EngineLatencyCard
│       ├── EngineName
│       ├── P50: 23ms
│       ├── P95: 89ms (amber if > 500ms)
│       └── P99: 312ms (red if > 1000ms)
├── ThroughputChart         ← line chart: ingest/recall per minute
├── ErrorRateChart          ← bar chart: error % per engine
└── AlertList               ← current active alerts
    └── AlertEntry
        ├── Severity        ← warning | critical
        ├── Service
        └── Message         ← "p95 latency > 500ms on graphiti"
```

### 3.4 Cost Alert Visualization

```typescript
// Budget usage indicator
function BudgetUsageAlert({ current, limit }: { current: number; limit: number }) {
  const pct = Math.round((current / limit) * 100);
  
  const status =
    pct >= 100 ? 'exceeded' :
    pct >= 80  ? 'warning'  :
                 'ok';
  
  const STATUS_STYLES = {
    ok:       'bg-green-50 border-green-200',
    warning:  'bg-amber-50 border-amber-200',
    exceeded: 'bg-red-50 border-red-200',
  };
  
  return (
    <div className={`rounded-lg border p-4 ${STATUS_STYLES[status]}`}>
      {status === 'exceeded' && <AlertBanner>Budget exceeded!</AlertBanner>}
      {status === 'warning'  && <AlertBanner>80% budget used — monitor closely</AlertBanner>}
      <ProgressBar value={pct} max={100} color={status} />
      <p>${current.toFixed(2)} / ${limit}/day ({pct}%)</p>
    </div>
  );
}
```

### 3.5 Error Explorer

```
ErrorExplorer
├── ServiceFilter           ← all services dropdown
├── ErrorsTable
│   └── ErrorRow
│       ├── ErrorMessage    ← truncated to 80 chars
│       ├── Service         ← badge
│       ├── Count           ← occurrence count
│       ├── LastOccurrence  ← "2 minutes ago"
│       └── StackButton     ← expand stack trace
└── ErrorDetailDrawer
    ├── FullMessage
    ├── Service
    ├── Count + Timeline
    └── StackTrace          ← mono font, scrollable
```

---

## 4. React Query Hooks

```typescript
export function useLLMCosts(dateRange: DateRange) {
  return useQuery({
    queryKey: ['analytics', 'llm-cost', dateRange],
    queryFn:  () => analyticsApi.getLLMCost(dateRange),
    refetchInterval: 60_000,   // refresh every minute
  });
}

export function useObservabilityMetrics() {
  return useQuery({
    queryKey: ['observability', 'metrics'],
    queryFn:  () => observabilityApi.getMetrics(),
    refetchInterval: 30_000,
  });
}
```

---

## 5. Acceptance Criteria (Frontend)

- [ ] LLM cost by provider (OpenAI vs Anthropic) in donut chart
- [ ] LLM cost by task (extraction vs consolidation) in bar chart
- [ ] Daily budget usage bar with warning/exceeded states
- [ ] Engine latency grid: P50/P95/P99 per engine (7 cards)
- [ ] P95 > 500ms → amber highlight; P99 > 1000ms → red
- [ ] Error explorer filterable by service
- [ ] Error stack trace expandable in drawer
- [ ] Cost timeline chart: today/7d/30d windows
- [ ] Alert list: shows active p95 > 500ms and cost budget alerts
