# UI Solution: UI-SOL-AM-006 — Consolidation Pipeline Monitor

**Solution ID:** UI-SOL-AM-006  
**CR References:** [CR-AM-006](../../../../docs/crs/v1/agentmemory/CR-AM-006-Consolidation-Pipeline.md)  
**Backend Solution:** [SOL-006-Consolidation-Pipeline.md](../../../../backend/specs/crs/v1/agentmemory/solutions/SOL-006-Consolidation-Pipeline.md)  
**Feature:** Memory Consolidation Pipeline — 4-Tier Sleep Model  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/pipelines/`

---

## 1. Mục Đích

Xây dựng UI cho Consolidation Pipeline Monitor:
- Xem trạng thái 4-tier consolidation pipeline (L0 hoạt động ngắn, L3 chạy hàng tuần)
- Monitor queue depth và throughput per tier
- Xem active consolidation jobs và progress
- Track worker status per engine

---

## 2. Backend API Alignment

### API Endpoints

| Method | Path | Mô tả |
|--------|------|--------|
| `GET` | `/v1/console/pipelines/status` | Tổng quan tất cả engine pipelines |
| `GET` | `/v1/console/pipelines/queues` | Queue metrics (depth, throughput, retries) |
| `GET` | `/v1/console/pipelines/workers` | Workers per engine |
| `GET` | `/v1/console/pipelines/{engine}/jobs` | Jobs cho engine cụ thể |
| `GET` | `/v1/console/pipelines/{engine}/jobs/{id}` | Job detail + stages |

### TypeScript Types

```typescript
// ui/src/types/pipeline.ts (extend existing)

type ConsolidationTier = 'L0' | 'L1' | 'L2' | 'L3';

interface ConsolidationTierStatus {
  tier:           ConsolidationTier;
  description:    string;     // "Dedup (every 5min)" | "Merge (hourly)"...
  interval:       string;     // "5m" | "1h" | "24h" | "7d"
  jobs_pending:   number;
  jobs_completed: number;
  last_run:       string;
  next_run:       string;
  status:         'running' | 'sleeping' | 'idle';
}
```

---

## 3. Components Architecture

### 3.1 Pipeline Overview

```
PipelinesPage
├── TierStatusGrid              ← 4 tier cards
│   ├── TierCard (L0)           ← "Dedup — runs every 5min"
│   ├── TierCard (L1)           ← "Merge — runs hourly"
│   ├── TierCard (L2)           ← "Consolidate — runs daily"
│   └── TierCard (L3)           ← "Archive — runs weekly"
├── QueueDepthChart             ← line chart per engine over time
├── WorkerStatusTable           ← engine → running/idle workers
└── ActiveJobsList              ← currently running jobs with progress bars
    └── JobRow
        ├── EngineBadge
        ├── TierBadge           ← L0/L1/L2/L3
        ├── ProgressBar         ← 0-100%
        ├── StagesTimeline      ← horizontal mini-stages
        └── Duration            ← "running for 2m 30s"
```

### 3.2 Tier Card Design

```
┌─────────────────────────────┐
│  L0 — DEDUP                 │
│  Every 5 minutes            │
│  ● RUNNING                  │
│                             │
│  Pending:   47 jobs         │
│  Completed: 1,234 today     │
│  Next run:  in 2m 30s       │
└─────────────────────────────┘
```

### 3.3 4-Tier Sleep Model Visual

```
Timeline: ──[L0:5min]──[L0:5min]──[L1:1h]──[L0:5min]──[L0:5min]──[L2:24h]──▶
                                     ▲
                                  Current time
```

---

## 4. React Query Hooks

```typescript
// ui/src/api/hooks/usePipelines.ts

export function usePipelineStatus() {
  return useQuery({
    queryKey: ['pipelines', 'status'],
    queryFn:  () => pipelineApi.getStatus(),
    refetchInterval: 10_000,    // refresh every 10s
  });
}

export function usePipelineQueues() {
  return useQuery({
    queryKey: ['pipelines', 'queues'],
    queryFn:  () => pipelineApi.getQueues(),
    refetchInterval: 5_000,
  });
}

export function useEngineJobs(engine: EngineType) {
  return useQuery({
    queryKey: ['pipelines', engine, 'jobs'],
    queryFn:  () => pipelineApi.getEngineJobs(engine),
    refetchInterval: 3_000,     // jobs thay đổi nhanh
  });
}
```

---

## 5. Acceptance Criteria (Frontend)

- [ ] 4 tier cards hiển thị đúng status, next_run countdown
- [ ] Queue depth chart cập nhật mỗi 5s
- [ ] Active jobs list với progress bars cập nhật realtime
- [ ] Worker table hiển thị running/idle count per engine
- [ ] Job detail page: stages timeline với duration per stage
- [ ] Color coding: L0=green, L1=blue, L2=purple, L3=orange
