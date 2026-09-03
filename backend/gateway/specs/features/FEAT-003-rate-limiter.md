---
id: FEAT-003
title: Redis Sliding Window Rate Limiter
service: vnp-gateway
version: 1.0.0
status: Done
priority: P1
created: 2026-05-09
updated: 2026-05-09
linked_sol: SOL-001
---

## Mục Tiêu

Implement per-tenant, per-endpoint rate limiting using Redis sorted sets (sliding window algorithm).

## Scope

### In Scope
- Redis sliding window rate limiter
- 3 tiers: Free (60 RPM), Pro (600 RPM), Enterprise (6000 RPM)
- Response headers: X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset, Retry-After
- Fail-open when Redis is unavailable

### Out of Scope
- IP-based rate limiting (tenant-based only)
- Per-user rate limiting (per-tenant only)

## Acceptance Criteria
- [ ] AC-1: Given free-tier tenant, When 61st request in 60s, Then return 429 with Retry-After
- [ ] AC-2: Given pro-tier tenant, When 600 requests in 60s, Then all succeed
- [ ] AC-3: Given Redis down, When request arrives, Then pass-through (fail-open)
- [ ] AC-4: Given rate-limited request, Then response includes X-RateLimit-* headers

## Test Requirements
- **Unit tests**: Sliding window algorithm, tier resolution
- **Integration tests**: Redis-backed rate limit verification
- **Minimum coverage**: 80%
