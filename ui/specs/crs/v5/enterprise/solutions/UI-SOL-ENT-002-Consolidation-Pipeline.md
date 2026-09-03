# UI Solution: UI-SOL-ENT-002 — Consolidation Pipeline UI (Enterprise)

**Solution ID:** UI-SOL-ENT-002  
**CR References:** [CR-ENT-002](../../../../docs/crs/v5/enterprise/CR-ENT-002-Consolidation-Pipeline.md)  
**Backend Solution:** [SOL-ENT-002](../../../../backend/specs/crs/v5/enterprise/solutions/SOL-ENT-002-Consolidation-Pipeline.md)  
**Feature:** Consolidation Pipeline — 4-Tier Sleep Model Visualization  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/pipelines/`

---

## 1. Mục Đích

Enterprise-level Consolidation Pipeline UI:
- 4-tier sleep model visualization (L0: 5min, L1: 1h, L2: 24h, L3: 7d)
- Queue depth monitoring per tier
- Worker status per engine
- Pipeline health and SLA monitoring
- Historical throughput charts

---

## 2. Backend API Contract

```http
GET /v1/console/pipelines/status     → pipeline overview
GET /v1/console/pipelines/queues     → QueueMetrics
GET /v1/console/pipelines/workers    → WorkerStatus[]
GET /v1/console/pipelines/templates  → pipeline templates
GET /v1/console/pipelines/{engine}/jobs → PipelineJob[]
```

---

## 3. Components

### 3.1 4-Tier Sleep Model Visualization

```
ConsolidationPipelinePage
├── TierVisualization       ← horizontal timeline showing tiers
│   ├── L0 Block (5min)     ← "Dedup & Noise Filter"
│   ├── L1 Block (1h)       ← "Smart Merge"
│   ├── L2 Block (24h)      ← "Cross-Engine Consolidate"
│   └── L3 Block (7d)       ← "Archive & Prune"
├── TierStatusCards (4 cards, responsive grid)
│   ├── TierCard L0
│   │   ├── Title: "L0 — Dedup (every 5min)"
│   │   ├── Status badge (running/sleeping/idle)
│   │   ├── Pending jobs: 47
│   │   ├── Completed today: 1,234
│   │   └── Next run countdown
│   ├── TierCard L1...
│   ├── TierCard L2...
│   └── TierCard L3...
├── QueueDepthChart         ← stacked bar chart per tier
│   └── TimeWindow: 1h|6h|24h
└── WorkerTable             ← engine × running/idle workers
```

### 3.2 SLA Monitoring Panel

```
SLAMonitoringPanel
├── SLAMetrics
│   ├── L0 Completion Rate  ← "99.2% on-time" (green)
│   ├── L1 Completion Rate  ← "97.8% on-time" (amber warning)
│   ├── L2 Completion Rate  ← "100% on-time" (green)
│   └── L3 Completion Rate  ← "99.5% on-time" (green)
└── AlertList               ← SLA violations
    └── AlertEntry          ← "L1 delayed by 3m on graphiti engine"
```

### 3.3 Historical Throughput Chart

```typescript
// Recharts AreaChart showing jobs/hour over time per tier
function ThroughputChart({ data }: { data: TierThroughputData[] }) {
  return (
    <AreaChart data={data}>
      <Area type="monotone" dataKey="L0" stroke="#22c55e" fill="#dcfce7" />
      <Area type="monotone" dataKey="L1" stroke="#3b82f6" fill="#dbeafe" />
      <Area type="monotone" dataKey="L2" stroke="#8b5cf6" fill="#ede9fe" />
      <Area type="monotone" dataKey="L3" stroke="#f97316" fill="#ffedd5" />
    </AreaChart>
  );
}
```

---

## 4. Acceptance Criteria (Frontend)

- [ ] 4 tier cards showing status (running/sleeping/idle)
- [ ] Next run countdown per tier (L0: < 5min, L1: < 1h, etc.)
- [ ] Queue depth chart updated every 5s
- [ ] Worker table shows running/idle per engine
- [ ] SLA compliance rates per tier
- [ ] Historical throughput chart: 1h/6h/24h time windows
