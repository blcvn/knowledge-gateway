# 09 — Analytics Service

> **gRPC**: 9008 | **Health**: 9088

---

## 1. Purpose

Usage tracking, token economics reporting, cost savings calculation. Thu thập tất cả API request logs, aggregate theo key/org/hourly, và cung cấp dashboard analytics.

---

## 2. Clean Architecture

```
services/analytics-service/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # APIRequest, UsageAggregation, TokenMetrics
│   │   ├── value_object.go     # RequestType, TimePeriod, AggregationDimension
│   │   └── errors.go
│   ├── usecase/
│   │   ├── track_request.go    # Log individual API request
│   │   ├── get_usage.go        # Aggregated usage analytics
│   │   ├── get_memory_stats.go # Memory growth metrics
│   │   ├── get_token_economics.go # Token savings, cost analytics
│   │   ├── aggregate.go        # Periodic aggregation (cron)
│   │   ├── port/
│   │   │   ├── input.go        # TrackRequestUC, GetUsageUC
│   │   │   └── output.go       # RequestRepo, AggregationRepo
│   │   └── dto/
│   │       └── analytics.go    # UsageOutput, TokenEconomicsOutput
│   ├── adapter/
│   │   ├── grpc/handler.go     # AnalyticsServiceServer implementation
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── api_request.go      # Request log insert + queries
│   │   │       └── aggregation.go      # Pre-computed aggregations
│   │   ├── event/
│   │   │   └── subscriber.go          # NATS: auth.api_key.used, *.completed
│   │   └── scheduler/
│   │       └── aggregation_cron.go    # Hourly/daily aggregation
│   └── infra/
│       ├── config/config.go
│       └── wire/wire.go
├── migrations/
│   ├── 001_create_api_requests.up.sql
│   └── 002_create_aggregations.up.sql
└── Dockerfile
```

---

## 3. Request Tracking Model

```go
type APIRequest struct {
    ID              string
    Type            RequestType     // add|search|fast_search|update|delete|chat|search_v4
    OrgID           string
    UserID          string
    KeyID           string          // Which API key was used
    StatusCode      int
    Duration        int64           // ms
    OriginalTokens  int             // Before optimization
    FinalTokens     int             // After optimization
    TokensSaved     int             // Computed: original - final
    CostSavedUSD    float64         // Dollar value of savings
    Model           *string         // LLM model used
    Provider        *string         // LLM provider
    Origin          string          // api | mcp | console
    CreatedAt       time.Time
}
```

---

## 4. Aggregation Queries

| Endpoint | Dimensions | Metrics |
|----------|-----------|---------|
| `GetUsageAnalytics` | By RequestType, By APIKey, Hourly | count, avgDuration, lastUsed |
| `GetMemoryStats` | By OrgID | totalMemories, memoriesGrowth, searchQueries, totalConnections |
| `GetTokenEconomics` | By period (7d/30d/90d/lifetime) | tokensByDay, latencyTrends, amountSaved |

---

## 5. gRPC Interface

```protobuf
service AnalyticsService {
  rpc TrackRequest(TrackRequestEvent) returns (google.protobuf.Empty);
  rpc GetUsageAnalytics(UsageAnalyticsRequest) returns (UsageAnalyticsResponse);
  rpc GetMemoryStats(MemoryStatsRequest) returns (MemoryStatsResponse);
  rpc GetTokenEconomics(TokenEconomicsRequest) returns (TokenEconomicsResponse);
}
```
