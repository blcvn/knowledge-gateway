# TASK-SM-011 — services/analytics-service: Token Usage & Memory Analytics

**Task ID:** TASK-SM-011  
**Wave:** 5 (Ecosystem)  
**Solution:** [SOL-SM-009](../solutions/SOL-SM-009-Analytics-Token-Economics.md)  
**Depends on:** TASK-SM-006 (memory.created/forgotten events), TASK-SM-005 (document.processed)  
**Ước tính:** 4h  
**Priority:** Medium

---

## Mục tiêu

Tạo `services/analytics-service/` với:
1. Token usage tracking per OrgID (document ingestion + LLM calls)
2. Memory stats (created, forgotten counts)
3. NATS event consumers (document.processed, memory.created, memory.forgotten)
4. Daily/weekly/monthly aggregation
5. REST API: usage stats + token economics

---

## Công việc cụ thể

### 1. Tạo Domain Model

**`services/analytics-service/internal/domain/analytics.go`**

```go
type UsageEvent struct {
    ID          string
    OrgID       string
    Type        EventType    // "token_used" | "memory_created" | "memory_forgotten" | "document_ingested" | "search_query"
    TokenCount  int          // for token_used events
    Metadata    map[string]any
    OccurredAt  time.Time
}

type OrgStats struct {
    OrgID           string
    Period          string       // "daily" | "weekly" | "monthly"
    PeriodStart     time.Time
    PeriodEnd       time.Time
    TokensUsed      int64        // total tokens consumed
    MemoriesCreated int
    MemoriesForgotten int
    DocumentsIngested int
    SearchQueries   int
    StorageBytesUsed int64
}

type TokenBudget struct {
    OrgID       string
    Plan        string  // free | pro | enterprise
    MonthlyLimit int64  // tokens per month
    UsedThisMonth int64
    ResetsAt    time.Time  // first day of next month
}
```

### 2. Tạo PostgreSQL Schema

**`services/analytics-service/migrations/001_create_analytics.sql`**

```sql
CREATE TABLE usage_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL,
    type        TEXT NOT NULL,
    token_count INT DEFAULT 0,
    metadata    JSONB DEFAULT '{}',
    occurred_at TIMESTAMPTZ DEFAULT now()
);

-- Partition by month for performance
CREATE INDEX idx_usage_events_org_time ON usage_events(org_id, occurred_at DESC);
CREATE INDEX idx_usage_events_type ON usage_events(org_id, type, occurred_at DESC);

CREATE TABLE org_stats (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL,
    period           TEXT NOT NULL,        -- daily|weekly|monthly
    period_start     TIMESTAMPTZ NOT NULL,
    period_end       TIMESTAMPTZ NOT NULL,
    tokens_used      BIGINT DEFAULT 0,
    memories_created INT DEFAULT 0,
    memories_forgotten INT DEFAULT 0,
    documents_ingested INT DEFAULT 0,
    search_queries   INT DEFAULT 0,
    storage_bytes    BIGINT DEFAULT 0,
    UNIQUE (org_id, period, period_start)
);

CREATE TABLE token_budgets (
    org_id         UUID PRIMARY KEY,
    plan           TEXT NOT NULL DEFAULT 'free',
    monthly_limit  BIGINT NOT NULL DEFAULT 100000,
    used_this_month BIGINT DEFAULT 0,
    resets_at      TIMESTAMPTZ NOT NULL
);
```

### 3. Implement NATS Event Consumers

**`services/analytics-service/internal/adapter/subscriber/`**

```go
// document_events.go: "document.processed" → log usage_event type=document_ingested
// "document.failed" → log type=document_failed

// memory_events.go: "memory.created" → type=memory_created
// "memory.forgotten" → type=memory_forgotten

// search_events.go: "sm.search.completed" → type=search_query + token_count

// All consumers: fire-and-forget, non-blocking
// Use batch INSERT (every 100 events or every 5 seconds, whichever first)
```

### 4. Implement Stats Aggregation (Cron)

**`services/analytics-service/internal/adapter/cron/aggregate.go`**

```go
// Aggregation cron:
// Daily: runs at 00:05 UTC → aggregate yesterday's events → upsert org_stats
// Weekly: runs on Monday 00:10 UTC
// Monthly: runs on 1st at 00:15 UTC

// SQL aggregation:
// INSERT INTO org_stats (org_id, period, period_start, period_end, tokens_used, ...)
// SELECT org_id, 'daily', $period_start, $period_end,
//        SUM(token_count), COUNT(*) FILTER (WHERE type='memory_created'), ...
// FROM usage_events
// WHERE occurred_at >= $period_start AND occurred_at < $period_end
// GROUP BY org_id
// ON CONFLICT (org_id, period, period_start) DO UPDATE SET ...
```

### 5. Implement Token Budget Management

**`services/analytics-service/internal/usecase/check_budget.go`**

```go
// CheckTokenBudget: deduct tokens + check limit
// Returns: ErrBudgetExceeded if used >= limit
// Used by document-service before starting embedding (expensive LLM call)
func (uc *CheckTokenBudgetUseCase) Deduct(ctx, orgID string, tokens int) error

// GetBudget: current usage stats for org
func (uc *CheckTokenBudgetUseCase) GetBudget(ctx, orgID string) (*TokenBudget, error)
```

### 6. REST Endpoints

```
GET /api/v1/analytics/usage          → OrgStats (period: daily|weekly|monthly)
GET /api/v1/analytics/tokens         → TokenBudget for current org
GET /api/v1/analytics/memories       → Memory creation/forget trend
GET /api/v1/analytics/documents      → Document ingestion stats
```

### 7. Tests

- `TestUsageEvent_BatchInsert`: 100 events → single INSERT
- `TestAggregation_Daily`: mock events → correct daily totals
- `TestTokenBudget_Deduct`: deduct 1000 tokens → used_this_month increases
- `TestTokenBudget_ExceedLimit`: deduct beyond limit → ErrBudgetExceeded
- `TestTokenBudget_ResetsMonthly`: mock new month → used_this_month = 0

---

## Acceptance Criteria

- [ ] `go build ./services/analytics-service/...` không lỗi
- [ ] NATS document.processed → usage_event inserted
- [ ] NATS memory.created → memory count incremented
- [ ] GET /analytics/usage → daily stats với correct period
- [ ] TokenBudget.Deduct beyond limit → ErrBudgetExceeded
- [ ] Daily aggregation cron runs without error
- [ ] `go test ./services/analytics-service/...` pass

---

## Files tạo ra

```
services/analytics-service/
├── internal/
│   ├── domain/
│   │   └── analytics.go
│   ├── usecase/
│   │   ├── check_budget.go
│   │   ├── check_budget_test.go
│   │   ├── get_stats.go
│   │   └── log_event.go
│   ├── adapter/
│   │   ├── subscriber/
│   │   │   ├── document_events.go
│   │   │   ├── memory_events.go
│   │   │   └── search_events.go
│   │   ├── cron/
│   │   │   └── aggregate.go
│   │   └── grpc/
│   │       └── analytics_server.go
│   └── infra/
│       └── postgres/
│           ├── usage_event_repo.go
│           ├── stats_repo.go
│           └── budget_repo.go
└── migrations/
    └── 001_create_analytics.sql

gateway/adapter/handler/
└── analytics_handler.go
```

## Sau khi hoàn thành

Chạy: `go build ./services/analytics-service/... && go test ./services/analytics-service/...`
