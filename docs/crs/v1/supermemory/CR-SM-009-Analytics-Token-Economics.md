# Change Request: CR-SM-009 — Analytics & Token Economics

**CR ID:** CR-SM-009  
**Component:** `services/analytics-service` [NEW SERVICE]  
**Priority:** Medium  
**Status:** In Progress
**Reference:** Supermemory PRD §7.3, SRS §2.9, specs/services/09-analytics-service.md

---

## 1. Mô tả

Xây dựng **Analytics Service** — theo dõi sử dụng và tính toán tiết kiệm Token Economics:

1. **Request Tracking**: Log từng API request với type, duration, status code, token usage.
2. **Token Economics**: Tính toán token savings (original vs optimized) và cost savings (USD).
3. **Usage Analytics**: Aggregated metrics theo API key, time period, operation type.
4. **Memory Growth Stats**: Tổng số memories, growth rate, search queries count.
5. **Hourly Aggregation**: Pre-computed aggregations để dashboard query nhanh.

---

## 2. Vấn đề hiện tại

- VNP Memory chưa có hệ thống analytics chi tiết.
- Không track được token usage → không biết hệ thống tiết kiệm bao nhiêu chi phí LLM.
- Thiếu visibility vào việc API key nào đang dùng nhiều tài nguyên nhất.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/analytics-service/` (Port gRPC: 9008)

### 3.2. Request Tracking Model

```go
type APIRequest struct {
    ID             string
    Type           RequestType   // add|search|fast_search|update|delete|chat|search_v4
    OrgID          string
    UserID         string
    KeyID          string        // API key used
    StatusCode     int
    Duration       int64         // milliseconds
    OriginalTokens int           // Token count before optimization
    FinalTokens    int           // Token count after optimization
    TokensSaved    int           // OriginalTokens - FinalTokens
    CostSavedUSD   float64       // Dollar value of token savings
    Model          *string       // LLM model used (gpt-4o, claude-3, etc.)
    Provider       *string       // openai | anthropic | google
    Origin         string        // api | mcp | console
    CreatedAt      time.Time
}
```

### 3.3. Aggregation Queries

| Endpoint | Dimensions | Metrics |
|----------|-----------|---------|
| `GetUsageAnalytics` | By RequestType, By APIKey, Hourly | count, avgDuration, lastUsed |
| `GetMemoryStats` | By OrgID | totalMemories, memoriesGrowth, searchQueries, totalConnections |
| `GetTokenEconomics` | By period (7d/30d/90d/lifetime) | tokensByDay, latencyTrends, amountSavedUSD |

### 3.4. Event-Driven Tracking

- Subscribe `auth.api_key.used` → log API key usage.
- Subscribe `document.processed` → log processing metrics.
- Subscribe `memory.created` → update memory count stats.
- Cron job chạy mỗi giờ để pre-compute aggregations.

### 3.5. API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/api/v1/analytics/usage` | Usage analytics (by type, by key, hourly) |
| `GET` | `/api/v1/analytics/memory` | Memory growth statistics |
| `GET` | `/api/v1/analytics/chat` | Token economics (savings, latency trends) |

### 3.6. Token Economics Formula

```
Token Savings = OriginalTokens - FinalTokens
Cost Saved (USD) = TokensSaved × (ModelPrice per token)
// Ví dụ: gpt-4o $0.005/1K tokens → 1000 tokens saved = $0.005
```

---

## 4. Acceptance Criteria

- [ ] Mỗi API request được log với đủ fields (type, duration, tokens, cost).
- [ ] Dashboard `GET /analytics/usage` trả về breakdown theo operation type và API key.
- [ ] `GET /analytics/chat` trả về tổng số USD tiết kiệm được trong 30 ngày.
- [ ] Memory growth chart hiển thị xu hướng tăng/giảm theo ngày.
- [ ] Aggregation query chạy trong < 100ms (pre-computed).
