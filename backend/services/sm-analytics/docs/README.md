---
id: DOC-S01
service: sm-analytics
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Supermemory Team
---

# sm-analytics

> **Group**: Supermemory | **gRPC Port**: 9078 | **Health Port**: 9123 | **Origin**: Supermemory

## Purpose

Usage analytics and reporting service. Tracks **API request patterns**, **token economics** (original vs final tokens, savings), **cost tracking**, **memory usage metrics**, and **chat analytics** per organization with time-based aggregation.

### Business Capability

- **Usage Analytics**: API requests per type (add/search/delete/chat), status codes, duration
- **Token Economics**: Track original tokens, final tokens, tokens saved, cost savings in USD
- **Memory Analytics**: Memory creation/deletion rates, storage growth, top containers
- **Chat Analytics**: Chat sessions, model usage, provider distribution, context modification rates
- **Time-Based Queries**: Filter by period (24h/7d/30d/all) or custom date range
- **Pagination**: Large result set support with page/limit

## API Surface

```protobuf
service SmAnalyticsService {
  rpc GetUsageAnalytics(AnalyticsRequest) returns (UsageResponse);
  rpc GetMemoryAnalytics(AnalyticsRequest) returns (MemoryAnalyticsResponse);
  rpc GetChatAnalytics(AnalyticsRequest) returns (ChatAnalyticsResponse);
}
```

### REST (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v3/analytics/usage` | API usage metrics |
| GET | `/v3/analytics/memory` | Memory lifecycle metrics |
| GET | `/v3/analytics/chat` | Chat interaction metrics |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL | SQL | Analytics data persistence |
| sm-auth | NATS sub | `sm.auth.api_key.used` → request tracking |

## Owner

- **Team**: VNP Memory — Supermemory
