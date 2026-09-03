---
id: DOC-S02
service: sm-analytics
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-analytics — API Reference

## gRPC Service Definition

```protobuf
service SmAnalyticsService {
  rpc GetUsageReport(UsageReportRequest) returns (UsageReport);
  rpc GetTokenUsage(TokenUsageRequest) returns (TokenUsage);
  rpc GetStorageMetrics(StorageMetricsRequest) returns (StorageMetrics);
  rpc GetActiveUsers(ActiveUsersRequest) returns (ActiveUsersResponse);
}
```

## RPCs: GetUsageReport, GetTokenUsage, GetStorageMetrics, GetActiveUsers

## NATS Events

Subscribed: All `sm.*` events → aggregate usage metrics.
