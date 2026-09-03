# Solution: SOL-SM-009 — Analytics & Token Economics

**CR ID:** CR-SM-009  
**Solution ID:** SOL-SM-009  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Tạo mới `services/analytics-service/` với event-driven tracking (NATS subscriptions), pre-computed hourly aggregations, và Token Economics formula. Tận dụng PostgreSQL TimescaleDB-style aggregation (hoặc native JSONB + cron) cho dashboard queries < 100ms.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| Observability infra | `services/obs-service/` | Có: OpenTelemetry + Prometheus |
| Prometheus metrics | `gateway/infra/middleware/` | Đang track latency/throughput |
| NATS events | Nhiều services | Events sẵn sàng để subscribe |
| `analytics/` domain | `services/vnp-platform/internal/domain/analytics/` | Minimal |

### Gap phân tích

- Chưa có request-level analytics storage (chỉ có Prometheus metrics)
- Chưa track token usage và cost savings
- Không có API-key-level breakdown
- Thiếu hourly pre-computed aggregations
- Chưa có dashboard analytics API

---

## 3. Thiết kế Giải pháp

### 3.1. Cấu trúc Service Mới

```
services/analytics-service/
├── internal/
│   ├── domain/
│   │   ├── request.go        # APIRequest entity
│   │   ├── aggregation.go    # HourlyAggregation entity
│   │   └── repository.go     # RequestRepository, AggregationRepository
│   ├── usecase/
│   │   ├── track_request.go        # Log API request
│   │   ├── get_usage_analytics.go  # Usage breakdown
│   │   ├── get_memory_stats.go     # Memory growth stats
│   │   └── get_token_economics.go  # Token savings + cost USD
│   ├── adapter/
│   │   ├── grpc/                   # AnalyticsService gRPC
│   │   ├── subscriber/
│   │   │   └── event_tracker.go    # NATS subscribers
│   │   └── cron/
│   │       └── aggregator.go       # Hourly aggregation job
│   └── infra/
│       └── postgres/
│           ├── request_repo.go
│           └── aggregation_repo.go
```

### 3.2. Domain Model

```go
// services/analytics-service/internal/domain/request.go

package domain

import "time"

type RequestType string

const (
    ReqTypeAdd        RequestType = "add"
    ReqTypeSearch     RequestType = "search"
    ReqTypeFastSearch RequestType = "fast_search"
    ReqTypeUpdate     RequestType = "update"
    ReqTypeDelete     RequestType = "delete"
    ReqTypeChat       RequestType = "chat"
    ReqTypeSearchV4   RequestType = "search_v4"
    ReqTypeProfile    RequestType = "profile"
    ReqTypeMCP        RequestType = "mcp"
)

type Origin string

const (
    OriginAPI     Origin = "api"
    OriginMCP     Origin = "mcp"
    OriginConsole Origin = "console"
)

type APIRequest struct {
    ID             string
    Type           RequestType
    OrgID          string
    UserID         string
    KeyID          string        // API key used
    StatusCode     int
    Duration       int64         // milliseconds
    OriginalTokens int           // Token count of raw input
    FinalTokens    int           // Token count after memory-based optimization
    TokensSaved    int           // OriginalTokens - FinalTokens
    CostSavedUSD   float64       // Dollar value of savings
    Model          *string       // "gpt-4o" | "claude-3-haiku" | etc.
    Provider       *string       // "openai" | "anthropic" | "google"
    Origin         Origin        // api | mcp | console
    CreatedAt      time.Time
}

type HourlyAggregation struct {
    HourBucket  time.Time   // Truncated to hour
    OrgID       string
    RequestType RequestType
    KeyID       *string     // Null = aggregate across all keys
    Count       int
    TotalDurationMs int64
    AvgDurationMs   float64
    TotalTokensSaved int
    TotalCostSavedUSD float64
    ErrorCount  int
}

type MemoryStats struct {
    OrgID            string
    TotalMemories    int
    MemoriesGrowth7d int    // Growth in last 7 days
    SearchQueries    int
    TotalConnections int
    TotalDocuments   int
    UpdatedAt        time.Time
}
```

### 3.3. Token Economics Formula

```go
// services/analytics-service/internal/usecase/get_token_economics.go

// Model pricing (per 1000 tokens, USD)
var modelPricing = map[string]float64{
    "gpt-4o":              0.005,
    "gpt-4o-mini":         0.000150,
    "claude-3-5-sonnet":   0.003,
    "claude-3-haiku":      0.000250,
    "gemini-1.5-pro":      0.00125,
    "default":             0.002,  // Fallback
}

func CalculateCostSaved(tokensSaved int, model string) float64 {
    price, ok := modelPricing[model]
    if !ok {
        price = modelPricing["default"]
    }
    return float64(tokensSaved) / 1000.0 * price
}

// GetTokenEconomics aggregates savings theo period
type TokenEconomicsResponse struct {
    Period        string                 // "7d" | "30d" | "90d" | "lifetime"
    TotalSaved    int64                  // Total tokens saved
    AmountSavedUSD float64              // Total USD saved
    TokensByDay   []DayTokenStats       // Breakdown theo ngày
    LatencyTrends []LatencyTrendPoint   // Avg latency theo ngày
}

type DayTokenStats struct {
    Date       time.Time
    Saved      int
    CostSaved  float64
}

func (uc *GetTokenEconomicsUseCase) Execute(ctx context.Context, req TokenEconomicsRequest) (*TokenEconomicsResponse, error) {
    since := periodToTime(req.Period) // 7d → now-7days

    // Query pre-computed hourly aggregations
    aggs, _ := uc.aggRepo.ListByPeriod(ctx, req.OrgID, since)

    // Aggregate by day
    dayMap := make(map[string]*DayTokenStats)
    var totalSaved int64
    var totalCostUSD float64

    for _, agg := range aggs {
        day := agg.HourBucket.Format("2006-01-02")
        if dayMap[day] == nil {
            dayMap[day] = &DayTokenStats{Date: agg.HourBucket.Truncate(24*time.Hour)}
        }
        dayMap[day].Saved += agg.TotalTokensSaved
        dayMap[day].CostSaved += agg.TotalCostSavedUSD
        totalSaved += int64(agg.TotalTokensSaved)
        totalCostUSD += agg.TotalCostSavedUSD
    }

    return &TokenEconomicsResponse{
        Period:         req.Period,
        TotalSaved:     totalSaved,
        AmountSavedUSD: totalCostUSD,
        TokensByDay:    sortDayStats(dayMap),
        LatencyTrends:  uc.buildLatencyTrends(aggs),
    }, nil
}
```

### 3.4. Event-Driven Tracking

```go
// services/analytics-service/internal/adapter/subscriber/event_tracker.go

type EventTracker struct {
    nats    NATSClient
    reqRepo RequestRepository
    aggRepo AggregationRepository
}

func (t *EventTracker) Start(ctx context.Context) {
    // Track API key usage
    t.nats.Subscribe(ctx, "auth.api_key.used", func(msg APIKeyUsedEvent) {
        t.reqRepo.Create(ctx, &APIRequest{
            Type:    ReqTypeAPI,
            OrgID:   msg.OrgID,
            UserID:  msg.UserID,
            KeyID:   msg.KeyID,
            Origin:  OriginAPI,
            CreatedAt: time.Now(),
        })
    })

    // Track document processing (với token info)
    t.nats.Subscribe(ctx, "document.processed", func(msg DocumentProcessedEvent) {
        tokensSaved := msg.OriginalTokens - msg.FinalTokens
        costSaved := CalculateCostSaved(tokensSaved, msg.Model)

        t.reqRepo.Create(ctx, &APIRequest{
            Type:           ReqTypeAdd,
            OrgID:          msg.OrgID,
            Duration:       msg.ProcessingMs,
            OriginalTokens: msg.OriginalTokens,
            FinalTokens:    msg.FinalTokens,
            TokensSaved:    tokensSaved,
            CostSavedUSD:   costSaved,
            Model:          &msg.Model,
            Provider:       &msg.Provider,
            CreatedAt:      time.Now(),
        })
    })

    // Track memory creation → update memory count
    t.nats.Subscribe(ctx, "memory.created", func(msg MemoryCreatedEvent) {
        t.updateMemoryStats(ctx, msg.OrgID, +1)
    })

    // Track search requests
    t.nats.Subscribe(ctx, "search.executed", func(msg SearchExecutedEvent) {
        t.reqRepo.Create(ctx, &APIRequest{
            Type:     toRequestType(msg.Mode),
            OrgID:    msg.OrgID,
            Duration: msg.DurationMs,
            Origin:   OriginAPI,
        })
    })
}
```

### 3.5. Hourly Aggregation Cron Job

```go
// services/analytics-service/internal/adapter/cron/aggregator.go

type HourlyAggregator struct {
    reqRepo RequestRepository
    aggRepo AggregationRepository
}

// Chạy mỗi giờ (phút 5 để tránh conflict với cron jobs khác)
// Cron: "5 * * * *"
func (a *HourlyAggregator) Run(ctx context.Context) {
    // Aggregate last completed hour
    lastHour := time.Now().Truncate(time.Hour).Add(-time.Hour)

    // Query raw requests cho giờ đó
    requests, _ := a.reqRepo.ListByHour(ctx, lastHour)

    // Group by (OrgID, RequestType, KeyID)
    groupMap := make(map[string]*HourlyAggregation)

    for _, req := range requests {
        key := fmt.Sprintf("%s:%s:%s", req.OrgID, req.Type, req.KeyID)
        if groupMap[key] == nil {
            groupMap[key] = &HourlyAggregation{
                HourBucket:  lastHour,
                OrgID:       req.OrgID,
                RequestType: req.Type,
                KeyID:       &req.KeyID,
            }
        }
        agg := groupMap[key]
        agg.Count++
        agg.TotalDurationMs += req.Duration
        agg.TotalTokensSaved += req.TokensSaved
        agg.TotalCostSavedUSD += req.CostSavedUSD
        if req.StatusCode >= 400 { agg.ErrorCount++ }
    }

    // Compute averages
    for _, agg := range groupMap {
        if agg.Count > 0 {
            agg.AvgDurationMs = float64(agg.TotalDurationMs) / float64(agg.Count)
        }
        a.aggRepo.Upsert(ctx, agg)
    }

    slog.Info("hourly aggregation completed", "hour", lastHour, "groups", len(groupMap))
}
```

---

## 4. Database Schema

```sql
-- api_requests table (partitioned by month for performance)
CREATE TABLE api_requests (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type              TEXT NOT NULL,
    org_id            UUID NOT NULL,
    user_id           UUID,
    key_id            TEXT,
    status_code       INT,
    duration          BIGINT DEFAULT 0,  -- milliseconds
    original_tokens   INT DEFAULT 0,
    final_tokens      INT DEFAULT 0,
    tokens_saved      INT GENERATED ALWAYS AS (original_tokens - final_tokens) STORED,
    cost_saved_usd    FLOAT DEFAULT 0,
    model             TEXT,
    provider          TEXT,
    origin            TEXT DEFAULT 'api',
    created_at        TIMESTAMPTZ DEFAULT now()
) PARTITION BY RANGE (created_at);

-- Partitions tạo hàng tháng
CREATE TABLE api_requests_2026_06 PARTITION OF api_requests
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

-- Hourly aggregations (pre-computed)
CREATE TABLE hourly_aggregations (
    hour_bucket         TIMESTAMPTZ NOT NULL,
    org_id              UUID NOT NULL,
    request_type        TEXT NOT NULL,
    key_id              TEXT,
    count               INT DEFAULT 0,
    total_duration_ms   BIGINT DEFAULT 0,
    avg_duration_ms     FLOAT DEFAULT 0,
    total_tokens_saved  INT DEFAULT 0,
    total_cost_saved_usd FLOAT DEFAULT 0,
    error_count         INT DEFAULT 0,
    PRIMARY KEY (hour_bucket, org_id, request_type, COALESCE(key_id, ''))
);

-- Memory stats (materialized, updated by events)
CREATE TABLE memory_stats (
    org_id             UUID PRIMARY KEY,
    total_memories     INT DEFAULT 0,
    total_documents    INT DEFAULT 0,
    total_connections  INT DEFAULT 0,
    search_queries     INT DEFAULT 0,
    updated_at         TIMESTAMPTZ DEFAULT now()
);

-- Indexes
CREATE INDEX idx_api_req_org_created ON api_requests(org_id, created_at DESC);
CREATE INDEX idx_api_req_key ON api_requests(key_id, created_at DESC);
CREATE INDEX idx_hourly_org_hour ON hourly_aggregations(org_id, hour_bucket DESC);
```

---

## 5. API Endpoints (Gateway)

```go
// gateway/adapter/handler/analytics_handler.go

func (h *AnalyticsHandler) Register(mux *http.ServeMux) {
    // Requires analytics:read permission
    mux.Handle("GET /api/v1/analytics/usage",
        RequirePermission(PermAnalyticsRead)(http.HandlerFunc(h.GetUsage)))
    mux.Handle("GET /api/v1/analytics/memory",
        RequirePermission(PermAnalyticsRead)(http.HandlerFunc(h.GetMemoryStats)))
    mux.Handle("GET /api/v1/analytics/chat",
        RequirePermission(PermAnalyticsRead)(http.HandlerFunc(h.GetTokenEconomics)))
}
```

**GET /api/v1/analytics/usage response:**
```json
{
  "byType": [
    { "type": "search", "count": 1250, "avgDuration": 245, "lastUsed": "2026-06-17T00:00:00Z" },
    { "type": "add", "count": 340, "avgDuration": 1200, "lastUsed": "2026-06-16T22:00:00Z" }
  ],
  "byKey": [
    { "keyId": "key_abc", "count": 890, "lastUsed": "2026-06-17T00:00:00Z" }
  ],
  "hourly": [
    { "hour": "2026-06-17T00:00:00Z", "count": 45, "avgDuration": 230 }
  ]
}
```

**GET /api/v1/analytics/chat?period=30d response:**
```json
{
  "period": "30d",
  "totalSaved": 2500000,
  "amountSavedUSD": 12.50,
  "tokensByDay": [
    { "date": "2026-06-16", "saved": 85000, "costSaved": 0.425 }
  ],
  "latencyTrends": [
    { "date": "2026-06-16", "avgDuration": 238 }
  ]
}
```

---

## 6. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | Domain model + DB schema (partitioned) | 1 ngày |
| **P2** | Event-driven tracking (NATS subscribers) | 2 ngày |
| **P3** | Hourly aggregation cron job | 1 ngày |
| **P4** | GetUsageAnalytics API | 1 ngày |
| **P5** | GetMemoryStats API | 0.5 ngày |
| **P6** | Token Economics (formula + API) | 1.5 ngày |
| **P7** | Gateway integration + REST handlers | 1 ngày |
| **P8** | Tests + Acceptance Criteria | 1 ngày |

**Tổng:** ~9 ngày (Wave 5)

---

## 7. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| Mỗi API request được log đủ fields | NATS subscribers → APIRequest entity |
| Usage breakdown theo operation và API key | HourlyAggregation GROUP BY type + key_id |
| /analytics/chat → USD tiết kiệm 30 ngày | GetTokenEconomics với period=30d |
| Memory growth chart | MemoryStats + daily aggregation |
| Aggregation < 100ms | Pre-computed hourly aggs (no real-time query) |
