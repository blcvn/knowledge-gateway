---
id: FEAT-PIP-007
title: graphiti-store gRPC Client Adapter
service: graphiti-pipeline
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement gRPC client adapter kết nối graphiti-pipeline → graphiti-store:9024. Dùng cho SaveBulk (persist nodes/edges/episode) và RollbackBulk (saga compensation). Bao gồm circuit breaker + deadline propagation.

## Scope

### In Scope
- `internal/adapter/client/store_client.go` — implements StoreClient port
- SaveBulk: atomic persistence of nodes + edges + episode
- RollbackBulk: saga compensation on SaveBulk failure
- Circuit breaker (gobreaker) with configurable thresholds
- gRPC deadline propagation from parent context
- OTel span injection into outgoing gRPC calls

### Out of Scope
- graphiti-store service implementation
- Proto file generation

## Thiết Kế Kỹ Thuật

### StoreClient Implementation

```go
type StoreGRPCClient struct {
    client pb.GraphitiStoreServiceClient
    cb     *gobreaker.CircuitBreaker
}

func (c *StoreGRPCClient) SaveBulk(ctx context.Context, req SaveBulkRequest) error {
    return c.cb.Execute(func() (interface{}, error) {
        _, err := c.client.SaveBulk(ctx, toProto(req))
        return nil, err
    })
}

func (c *StoreGRPCClient) RollbackBulk(ctx context.Context, episodeID string) error {
    _, err := c.client.RollbackBulk(ctx, &pb.RollbackBulkRequest{EpisodeId: episodeID})
    return err
}
```

### Circuit Breaker Config

| Setting | Default | Description |
|---------|---------|-------------|
| MaxRequests (half-open) | 3 | Requests allowed in half-open state |
| Interval | 60s | Cyclic period for failure count reset |
| Timeout | 30s | Time to wait before moving from open to half-open |
| ReadyToTrip | 5 consecutive failures | When to open the circuit |

## Acceptance Criteria

- [ ] AC-1: SaveBulk sends nodes + edges + episode to graphiti-store atomically
- [ ] AC-2: Circuit breaker opens after 5 consecutive gRPC failures
- [ ] AC-3: Circuit breaker auto-recovers after timeout (half-open → closed on success)
- [ ] AC-4: gRPC deadline from parent context propagated to store call
- [ ] AC-5: OTel trace spans injected into outgoing gRPC metadata
- [ ] AC-6: RollbackBulk called during saga compensation cleans up partial writes

## Test Requirements
- **Unit tests**: StoreClient with mock gRPC server, circuit breaker state transitions
- **Minimum coverage**: 80%
