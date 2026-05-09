---
id: FEAT-004
title: gRPC Client Registry + Circuit Breaker
service: vnp-gateway
version: 1.0.0
status: Done
priority: P0
created: 2026-05-09
updated: 2026-05-09
linked_sol: SOL-001
---

## Mục Tiêu

Implement gRPC client connection pool with health tracking and per-service circuit breaker using sony/gobreaker.

## Scope

### In Scope
- ServiceRegistry: manage gRPC connections to all 35 services
- Connection pooling with configurable max connections
- Per-service circuit breaker (sony/gobreaker): closed → open → half-open → closed
- Health check polling (background goroutine)
- Forward(ctx, target, request) with automatic retry on transient failures

### Out of Scope
- Service mesh (Istio/Linkerd)
- Client-side load balancing across multiple instances

## Acceptance Criteria
- [ ] AC-1: Given service address config, When gateway starts, Then all gRPC connections established
- [ ] AC-2: Given service down, When 5 consecutive failures, Then circuit opens → 503
- [ ] AC-3: Given circuit open, When 60s elapsed, Then half-open → allow 3 probe requests
- [ ] AC-4: Given half-open and probe succeeds, Then circuit closes → normal routing
- [ ] AC-5: Given Forward call, When tenant context exists, Then gRPC metadata includes x-tenant-id

## Test Requirements
- **Unit tests**: Circuit breaker state transitions, health check logic
- **Integration tests**: Mock gRPC server, circuit open/close scenarios
- **Minimum coverage**: 80%
