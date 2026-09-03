---
id: TDD-sm-analytics
title: Technical Design — sm-analytics
service: sm-analytics
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Supermemory
---

# Technical Design — sm-analytics

> **Group**: Supermemory | **gRPC Port**: 9078 | **Health Port**: 9123

## 1. Service Overview

Usage tracking, token economics, reporting dashboards. Tracks API requests, token usage/savings, storage growth, and chat analytics per organization with time-based aggregation.

## 2. Domain Layer

- **ApiRequest**: id, type (add|search|fast_search|request|update|delete|chat|search_v4), org_id, user_id, key_id, status_code, duration_ms, input(JSONB), output(JSONB), original_tokens, final_tokens, tokens_saved, cost_saved_usd, model, provider, conversation_id, context_modified, metadata, origin (api|mcp|web), created_at
- **RequestType**: enum — add, search, fast_search, request, update, delete, chat, search_v4
- **AnalyticsPeriod**: enum — 24h, 7d, 30d, all
- **UsageMetrics**: total_requests, requests_by_type, avg_duration, total_tokens, tokens_saved, cost_saved
- **MemoryMetrics**: memories_created, memories_deleted, storage_growth, top_containers
- **ChatMetrics**: total_chats, models_used, providers, context_modifications

## 3. gRPC API

```protobuf
service SmAnalyticsService {
  rpc GetUsageAnalytics(AnalyticsRequest) returns (UsageResponse);
  rpc GetMemoryAnalytics(AnalyticsRequest) returns (MemoryAnalyticsResponse);
  rpc GetChatAnalytics(AnalyticsRequest) returns (ChatAnalyticsResponse);
}
```

AnalyticsRequest: `{period?: "24h"|"7d"|"30d"|"all", from?: datetime, to?: datetime, page: int, limit: int}`

## 4. NATS Events

| Direction | Subject | Purpose |
|-----------|---------|---------|
| Subscribe | `sm.auth.api_key.used` | Log API request with type, duration, tokens |

## 5. Data Model

### Tables
- `api_requests`: id(PK), type, org_id, user_id, key_id, status_code, duration_ms, input(JSONB), output(JSONB), original_tokens, final_tokens, tokens_saved, cost_saved_usd, model, provider, conversation_id, context_modified(BOOL), metadata(JSONB), origin, created_at
- `daily_aggregates`: org_id, date, request_type, count, total_duration_ms, total_tokens, total_saved_tokens, total_cost_saved — materialized view

### Key Indexes
- `idx_req_org_created` (org_id, created_at DESC) — time-range queries
- `idx_req_org_type` (org_id, type, created_at) — per-type aggregation
- `idx_req_org_key` (org_id, key_id) — per-key analytics

## 6. Token Economics Algorithms

### Token Calculation
Calculates the savings when context is pruned or rewritten via MCP or frontend:
- `tokens_saved = original_tokens - final_tokens` (can be negative if query expansion is larger)
- `cost_saved_usd = (tokens_saved / 1000) * model_cost_per_1k`

## 7. Observability

- **Metrics**: analytics_query_total, analytics_query_latency, events_processed_total
- **Health**: gRPC + HTTP /healthz on port 9123

---

> **Next Steps**: FEAT-001 (Usage Analytics API), FEAT-002 (Token Economics Dashboard), FEAT-003 (Daily Aggregation Materialized View)

## Task Specs Registry

_To be populated during implementation._

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| TASK-ANA-001 | Implement Domain Models | Pending | P0 |
| TASK-ANA-002 | Implement Usecases | Pending | P0 |
| TASK-ANA-003 | Implement Adapters and Repositories | Pending | P0 |
| TASK-ANA-004 | Infrastructure and Telemetry setup | Pending | P1 |
